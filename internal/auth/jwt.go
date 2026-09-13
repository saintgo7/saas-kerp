package auth

import (
	"crypto/rand"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/config"
	"github.com/saintgo7/saas-kerp/internal/errors"
)

// JWTService handles JWT token operations
type JWTService struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	issuer          string
}

// NewJWTService creates a new JWT service
func NewJWTService(cfg *config.JWTConfig) *JWTService {
	return &JWTService{
		secret:          []byte(cfg.Secret),
		accessTokenTTL:  cfg.AccessTokenTTL,
		refreshTokenTTL: cfg.RefreshTokenTTL,
		issuer:          cfg.Issuer,
	}
}

// GenerateAccessToken generates a new access token
func (s *JWTService) GenerateAccessToken(userID, companyID uuid.UUID, email, name string, roles []string) (string, error) {
	now := time.Now()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		UserID:    userID,
		CompanyID: companyID,
		Email:     email,
		Name:      name,
		Roles:     roles,
		TokenType: TokenTypeAccess,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", errors.Wrap(errors.CodeInternal, "failed to sign token", err)
	}

	return tokenString, nil
}

// GenerateRefreshToken generates a cryptographically secure refresh token
func (s *JWTService) GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", errors.Wrap(errors.CodeInternal, "failed to generate refresh token", err)
	}
	return hex.EncodeToString(bytes), nil
}

// ValidateToken validates and parses a JWT access token.
//
// It enforces, in addition to the signature and expiry checks:
//   - the signing algorithm is HS256 (no alg confusion / alg:none),
//   - an "exp" claim is present,
//   - the issuer matches the configured issuer,
//   - the token_type claim is "access", so a token minted for another purpose
//     (refresh, service-to-service, webhook, ...) can never authenticate a request.
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	return s.ValidateTokenOfType(tokenString, TokenTypeAccess)
}

// ValidateTokenOfType validates a JWT and additionally requires the given token type.
func (s *JWTService) ValidateTokenOfType(tokenString string, expected TokenType) (*Claims, error) {
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	}
	if s.issuer != "" {
		opts = append(opts, jwt.WithIssuer(s.issuer))
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	}, opts...)

	if err != nil {
		// golang-jwt/v5 returns joined/wrapped errors, so pointer comparison never
		// matches. errors.Is is required for the expiry branch to be reachable.
		if stderrors.Is(err, jwt.ErrTokenExpired) {
			return nil, errors.ErrTokenExpired
		}
		return nil, errors.Wrap(errors.CodeTokenInvalid, "invalid token", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.ErrTokenInvalid
	}

	// Reject tokens minted for a different purpose (e.g. a refresh token presented
	// to the Auth middleware).
	if claims.TokenType != expected {
		return nil, errors.ErrTokenInvalid
	}

	// A token without a subject cannot identify anybody.
	if claims.UserID == uuid.Nil {
		return nil, errors.ErrTokenInvalid
	}

	return claims, nil
}

// GenerateTokenPair generates both access and refresh tokens
func (s *JWTService) GenerateTokenPair(userID, companyID uuid.UUID, email, name string, roles []string) (*TokenPair, error) {
	accessToken, err := s.GenerateAccessToken(userID, companyID, email, name, roles)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
	}, nil
}

// GetAccessTokenTTL returns the access token TTL
func (s *JWTService) GetAccessTokenTTL() time.Duration {
	return s.accessTokenTTL
}

// GetRefreshTokenTTL returns the refresh token TTL
func (s *JWTService) GetRefreshTokenTTL() time.Duration {
	return s.refreshTokenTTL
}

// RefreshAccessToken generates a new access token from existing claims
func (s *JWTService) RefreshAccessToken(claims *Claims) (string, error) {
	return s.GenerateAccessToken(
		claims.UserID,
		claims.CompanyID,
		claims.Email,
		claims.Name,
		claims.Roles,
	)
}
