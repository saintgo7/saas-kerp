package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/config"
	apperrors "github.com/saintgo7/saas-kerp/internal/errors"
)

func newTestService() *JWTService {
	return NewJWTService(&config.JWTConfig{
		Secret:          "kf8Qm2vZp7Ls4Xn9Td1Wb6Yr3Hc5Ju0Ae8Gi2Ko4Nq7Sv",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
		Issuer:          "kerp-api",
	})
}

// sign builds a token directly so tests can produce claims the service would
// never issue itself.
func (s *JWTService) sign(t *testing.T, claims *Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return signed
}

func baseClaims(svc *JWTService) *Claims {
	now := time.Now()
	return &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    svc.issuer,
			Subject:   uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		UserID:    uuid.New(),
		CompanyID: uuid.New(),
		Email:     "user@example.com",
		Name:      "Test User",
		Roles:     []string{"admin"},
		TokenType: TokenTypeAccess,
	}
}

func TestValidateToken_AcceptsIssuedAccessToken(t *testing.T) {
	svc := newTestService()

	userID, companyID := uuid.New(), uuid.New()
	token, err := svc.GenerateAccessToken(userID, companyID, "user@example.com", "Test User", []string{"admin"})
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken rejected a freshly issued token: %v", err)
	}
	if claims.UserID != userID || claims.CompanyID != companyID {
		t.Error("claims do not round-trip")
	}
}

// A token minted for another purpose must not authenticate a request, even
// though it is signed with the same key.
func TestValidateToken_RejectsNonAccessTokenType(t *testing.T) {
	svc := newTestService()

	claims := baseClaims(svc)
	claims.TokenType = TokenTypeRefresh

	if _, err := svc.ValidateToken(svc.sign(t, claims)); err == nil {
		t.Fatal("ValidateToken accepted a refresh-typed token")
	}
}

func TestValidateToken_RejectsMissingTokenType(t *testing.T) {
	svc := newTestService()

	claims := baseClaims(svc)
	claims.TokenType = ""

	if _, err := svc.ValidateToken(svc.sign(t, claims)); err == nil {
		t.Fatal("ValidateToken accepted a token with no token_type claim")
	}
}

func TestValidateToken_RejectsForeignIssuer(t *testing.T) {
	svc := newTestService()

	claims := baseClaims(svc)
	claims.Issuer = "some-other-service"

	if _, err := svc.ValidateToken(svc.sign(t, claims)); err == nil {
		t.Fatal("ValidateToken accepted a token from a different issuer")
	}
}

func TestValidateToken_RejectsMissingSubject(t *testing.T) {
	svc := newTestService()

	claims := baseClaims(svc)
	claims.UserID = uuid.Nil

	if _, err := svc.ValidateToken(svc.sign(t, claims)); err == nil {
		t.Fatal("ValidateToken accepted a token that identifies nobody")
	}
}

// The expiry branch used to compare errors with ==, which golang-jwt/v5 never
// satisfies because it wraps: expired tokens were reported as AUTH_006
// "invalid token" and clients could not tell "refresh me" from "log in again".
func TestValidateToken_ReportsExpiryDistinctly(t *testing.T) {
	svc := newTestService()

	claims := baseClaims(svc)
	past := time.Now().Add(-2 * time.Hour)
	claims.IssuedAt = jwt.NewNumericDate(past)
	claims.NotBefore = jwt.NewNumericDate(past)
	claims.ExpiresAt = jwt.NewNumericDate(past.Add(time.Minute))

	_, err := svc.ValidateToken(svc.sign(t, claims))
	if err == nil {
		t.Fatal("ValidateToken accepted an expired token")
	}
	if !apperrors.Is(err, apperrors.ErrTokenExpired) {
		t.Fatalf("expired token reported as %v, want %s", err, apperrors.CodeTokenExpired)
	}
}

func TestValidateToken_RejectsWrongSignature(t *testing.T) {
	svc := newTestService()
	other := NewJWTService(&config.JWTConfig{
		Secret:          "Zx4Rm8Qn1Bp6Vt3Yw9Kd2Fg7Hj5Lc0Ns8Ae4Ui1Or6Pz",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: time.Hour,
		Issuer:          "kerp-api",
	})

	token, err := other.GenerateAccessToken(uuid.New(), uuid.New(), "e@example.com", "n", nil)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	if _, err := svc.ValidateToken(token); err == nil {
		t.Fatal("ValidateToken accepted a token signed with a different key")
	}
}

func TestValidateToken_RejectsUnsignedToken(t *testing.T) {
	svc := newTestService()

	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, baseClaims(svc))
	token, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to build an alg:none token: %v", err)
	}

	if _, err := svc.ValidateToken(token); err == nil {
		t.Fatal("ValidateToken accepted an alg:none token")
	}
}

func TestGenerateRefreshToken_IsRandom(t *testing.T) {
	svc := newTestService()

	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		token, err := svc.GenerateRefreshToken()
		if err != nil {
			t.Fatalf("GenerateRefreshToken: %v", err)
		}
		if len(token) != 64 {
			t.Fatalf("refresh token should be 64 hex characters, got %d", len(token))
		}
		if _, dup := seen[token]; dup {
			t.Fatal("GenerateRefreshToken repeated a value")
		}
		seen[token] = struct{}{}
	}
}
