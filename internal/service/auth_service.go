package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/saintgo7/saas-kerp/internal/auth"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	jwtService       *auth.JWTService
	logger           *zap.Logger

	// unitOfWork lets Register create the company and its first user in one
	// transaction. See NewAuthService.
	unitOfWork repository.UnitOfWork

	// tenantSession installs app.current_tenant once authentication has
	// established which company the caller belongs to. See WithTenantSession.
	tenantSession repository.TenantSession
}

// maxAuthCandidates bounds how many same-address accounts a single login
// attempt will bcrypt-verify.
//
// users are unique per (company_id, email), so one address can legitimately
// exist in several companies and every candidate has to be checked - see
// authenticate below. Each check is a full bcrypt comparison (tens of
// milliseconds by design), so an unbounded loop would turn "an address that
// exists in N tenants" into N x 70ms of CPU per unauthenticated request.
// Candidates arrive ordered by created_at, so the cap keeps the oldest - the
// accounts that existed before anyone started piling on.
const maxAuthCandidates = 10

// decoyHash is a bcrypt hash of a value nobody knows, compared against when no
// account matches the address at all.
//
// Without it the "unknown e-mail" path returns after a single cheap query while
// the "known e-mail, wrong password" path pays for a bcrypt comparison, and the
// difference - tens of milliseconds, trivially measurable over the network -
// tells an unauthenticated caller which addresses have accounts.
//
// Computed lazily so the cost lands on the first failed login rather than on
// every process that imports this package.
var decoyHash = sync.OnceValue(func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(uuid.NewString()), bcrypt.DefaultCost)
	if err != nil {
		return nil
	}
	return hash
})

// burnPasswordComparison performs a bcrypt comparison that cannot succeed, to
// keep the no-such-user path as expensive as the wrong-password path.
func burnPasswordComparison(password string) {
	if hash := decoyHash(); hash != nil {
		_ = bcrypt.CompareHashAndPassword(hash, []byte(password))
	}
}

// NewAuthService creates a new auth service.
//
// unitOfWork is variadic only to keep this signature source compatible with
// the existing call site; Register cannot work without it. Self-service
// registration has to insert a companies row before the users row, because
// users.company_id references companies(id) - and there is no way to do that
// atomically through userRepo alone. Wire it:
//
//	service.NewAuthService(userRepo, refreshTokenRepo, jwtService, logger,
//	    repository.NewUnitOfWork(db))
func NewAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtService *auth.JWTService,
	logger *zap.Logger,
	unitOfWork ...repository.UnitOfWork,
) *AuthService {
	svc := &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtService:       jwtService,
		logger:           logger,
	}
	if len(unitOfWork) > 0 {
		svc.unitOfWork = unitOfWork[0]
	}
	return svc
}

// WithTenantSession wires the component that sets app.current_tenant once
// authentication knows which company the caller belongs to, and returns the
// service so it can be chained onto NewAuthService.
//
// The authentication routes are unauthenticated, so middleware.TenantSession -
// which reads the company id out of the JWT - cannot run for them. Without this
// the post-authentication writes (refresh-token insert, last-login update,
// registration) execute with no tenant set, which under RLS means they are
// rejected: the failure is loud, not silent, but login stops working.
//
// It is a setter rather than another constructor parameter so the existing
// NewAuthService signature keeps compiling for callers that build the service
// without a database (tests with fake repositories).
func (s *AuthService) WithTenantSession(tenantSession repository.TenantSession) *AuthService {
	s.tenantSession = tenantSession
	return s
}

// ErrRegistrationUnavailable is returned when self-service registration is
// invoked on a service that was constructed without a UnitOfWork.
var ErrRegistrationUnavailable = errors.New("self-service registration is not configured")

// ErrAmbiguousCredentials is returned when one e-mail address and password
// verify against accounts in more than one company.
//
// The address is unique per company, not globally, so this is reachable
// whenever two tenants hold the same address AND the same password. Picking one
// - which the previous global `WHERE email = ?` ... `First()` did implicitly -
// issues a session for a company the caller may have nothing to do with, so the
// login is refused instead.
//
// The HTTP layer must not distinguish it from ErrInvalidCredentials: telling
// the caller "your password also opens an account somewhere else" is an oracle
// over other tenants' credentials.
var ErrAmbiguousCredentials = errors.New("credentials match accounts in more than one company")

// authenticate resolves an e-mail address and password to exactly one user,
// across every tenant that holds the address.
//
// The address is unique per (company_id, email), so the candidate set can hold
// more than one row. Every candidate is verified rather than the first match
// winning: taking the first would reintroduce the arbitrary-tenant selection
// that the old global lookup did implicitly. When two tenants share both the
// address and the password the login is refused (ErrAmbiguousCredentials)
// rather than guessed at.
//
// Returns ErrInvalidCredentials for both "no such address" and "wrong
// password", and burns an equivalent bcrypt comparison on the former so the two
// cannot be told apart by response time.
func (s *AuthService) authenticate(ctx context.Context, email, password string) (*domain.User, error) {
	candidates, err := s.userRepo.FindAuthCandidatesByEmail(ctx, email)
	if err != nil {
		s.logger.Error("authentication failed: database error", zap.Error(err))
		return nil, err
	}

	if len(candidates) == 0 {
		burnPasswordComparison(password)
		return nil, domain.ErrInvalidCredentials
	}

	if len(candidates) > maxAuthCandidates {
		s.logger.Warn("authentication candidate set truncated",
			zap.Int("candidates", len(candidates)),
			zap.Int("cap", maxAuthCandidates),
		)
		candidates = candidates[:maxAuthCandidates]
	}

	// Every candidate is checked even after one verifies: stopping early makes
	// the response time depend on the position of the matching account, which
	// leaks the ordering of accounts sharing the address.
	var matched *domain.User
	ambiguous := false
	for i := range candidates {
		if candidates[i].CheckPassword(password) {
			if matched != nil {
				ambiguous = true
				continue
			}
			matched = &candidates[i]
		}
	}

	if ambiguous {
		s.logger.Warn("login refused: credentials match more than one company",
			zap.String("email", email),
		)
		return nil, ErrAmbiguousCredentials
	}
	if matched == nil {
		return nil, domain.ErrInvalidCredentials
	}
	return matched, nil
}

// withTenant runs fn with app.current_tenant set to companyID.
//
// When no TenantSession is configured the work still runs, just without the
// session variable - that is the current deployment, whose DSN is a superuser
// and therefore bypasses RLS anyway. Once the DSN moves to the least-privilege
// role the session becomes load-bearing, which is why the wiring is explicit
// rather than silently optional forever.
func (s *AuthService) withTenant(ctx context.Context, companyID uuid.UUID, fn func(ctx context.Context) error) error {
	if s.tenantSession == nil {
		return fn(ctx)
	}
	tenantCtx, release, err := s.tenantSession.Begin(ctx, companyID)
	if err != nil {
		return err
	}
	defer release()
	return fn(tenantCtx)
}

// LoginInput represents login request data
type LoginInput struct {
	Email    string
	Password string
}

// LoginOutput represents login response data
type LoginOutput struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int64        `json:"expires_in"`
	User         UserResponse `json:"user"`
}

// UserResponse represents user data in responses
type UserResponse struct {
	ID        uuid.UUID         `json:"id"`
	CompanyID uuid.UUID         `json:"company_id"`
	Email     string            `json:"email"`
	Name      string            `json:"name"`
	Role      domain.UserRole   `json:"role"`
	Status    domain.UserStatus `json:"status"`
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	// Resolve the address across every tenant holding it, then verify.
	user, err := s.authenticate(ctx, input.Email, input.Password)
	if err != nil {
		return nil, err
	}

	// Check user status
	if user.Status == domain.UserStatusInactive {
		s.logger.Debug("login failed: user inactive", zap.String("email", input.Email))
		return nil, domain.ErrUserInactive
	}
	if user.Status == domain.UserStatusLocked {
		s.logger.Debug("login failed: user locked", zap.String("email", input.Email))
		return nil, domain.ErrUserLocked
	}

	// Generate token pair
	tokenPair, err := s.jwtService.GenerateTokenPair(
		user.ID,
		user.CompanyID,
		user.Email,
		user.Name,
		user.GetRoles(),
	)
	if err != nil {
		s.logger.Error("login failed: token generation error", zap.Error(err))
		return nil, err
	}

	// Everything past this point is ordinary tenant-scoped work, so it runs
	// with app.current_tenant set to the company the credential resolved to.
	if err := s.withTenant(ctx, user.CompanyID, func(ctx context.Context) error {
		refreshToken := &domain.RefreshToken{
			UserID:    user.ID,
			Token:     tokenPair.RefreshToken,
			ExpiresAt: time.Now().Add(s.jwtService.GetRefreshTokenTTL()),
		}
		if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
			return err
		}

		// A failed last-login stamp must not fail the login itself.
		if err := s.userRepo.UpdateLastLogin(ctx, user.CompanyID, user.ID); err != nil {
			s.logger.Warn("failed to update last login", zap.Error(err))
		}
		return nil
	}); err != nil {
		s.logger.Error("login failed: refresh token storage error", zap.Error(err))
		return nil, err
	}

	s.logger.Info("user logged in", zap.String("user_id", user.ID.String()), zap.String("email", user.Email))

	return &LoginOutput{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
		User: UserResponse{
			ID:        user.ID,
			CompanyID: user.CompanyID,
			Email:     user.Email,
			Name:      user.Name,
			Role:      user.Role,
			Status:    user.Status,
		},
	}, nil
}

// RefreshInput represents refresh token request data
type RefreshInput struct {
	RefreshToken string
}

// RefreshOutput represents refresh token response data
type RefreshOutput struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Refresh generates new tokens using a refresh token
func (s *AuthService) Refresh(ctx context.Context, input RefreshInput) (*RefreshOutput, error) {
	// The token value is the only thing known at this point - not the tenant -
	// so this lookup goes through the pre-tenant path, which also resolves the
	// owning company_id.
	rt, err := s.refreshTokenRepo.FindByTokenForAuth(ctx, input.RefreshToken)
	if err != nil {
		if err == domain.ErrRefreshTokenNotFound {
			return nil, domain.ErrRefreshTokenNotFound
		}
		return nil, err
	}

	// Revoked tokens come back rather than being hidden by the query, so that
	// presenting one is distinguishable from presenting an unknown token.
	// Re-use of a revoked token is treated as compromise: every outstanding
	// token for that user is revoked rather than only refusing this request.
	if rt.Revoked {
		s.logger.Warn("refresh refused: revoked token replayed",
			zap.String("user_id", rt.UserID.String()),
		)
		if err := s.withTenant(ctx, rt.CompanyID, func(ctx context.Context) error {
			return s.refreshTokenRepo.RevokeByUserID(ctx, rt.UserID)
		}); err != nil {
			s.logger.Error("failed to revoke token family after replay", zap.Error(err))
		}
		return nil, domain.ErrRefreshTokenNotFound
	}

	if rt.IsExpired() {
		return nil, domain.ErrRefreshTokenExpired
	}

	var tokenPair *auth.TokenPair

	// From here the tenant is known, so the rest runs tenant-scoped.
	if err := s.withTenant(ctx, rt.CompanyID, func(ctx context.Context) error {
		// Re-read the account on every refresh. Without this a deactivated or
		// locked user keeps minting fresh access tokens for the whole refresh
		// TTL, because nothing else revisits their status after login.
		found, err := s.userRepo.FindByID(ctx, rt.CompanyID, rt.UserID)
		if err != nil {
			return err
		}
		switch found.Status {
		case domain.UserStatusInactive:
			return domain.ErrUserInactive
		case domain.UserStatusLocked:
			return domain.ErrUserLocked
		}
		pair, err := s.jwtService.GenerateTokenPair(
			found.ID,
			found.CompanyID,
			found.Email,
			found.Name,
			found.GetRoles(),
		)
		if err != nil {
			return err
		}
		tokenPair = pair

		// Rotate: the presented token is revoked and its replacement stored in
		// the same tenant-scoped unit of work. A failure here must not leave the
		// old token usable alongside a freshly issued one, so the error is
		// returned rather than logged and swallowed.
		if err := s.refreshTokenRepo.RevokeByToken(ctx, input.RefreshToken); err != nil {
			return err
		}
		return s.refreshTokenRepo.Create(ctx, &domain.RefreshToken{
			UserID:    found.ID,
			Token:     pair.RefreshToken,
			ExpiresAt: time.Now().Add(s.jwtService.GetRefreshTokenTTL()),
		})
	}); err != nil {
		return nil, err
	}

	return &RefreshOutput{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// Logout revokes all refresh tokens for a user
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	return s.refreshTokenRepo.RevokeByUserID(ctx, userID)
}

// LogoutByToken revokes a specific refresh token
func (s *AuthService) LogoutByToken(ctx context.Context, refreshToken string) error {
	return s.refreshTokenRepo.RevokeByToken(ctx, refreshToken)
}

// RegisterInput represents registration request data
type RegisterInput struct {
	CompanyID      uuid.UUID
	CompanyName    string
	BusinessNumber string
	Email          string
	Password       string
	Name           string
	Phone          string
}

// RegisterOutput represents registration response data
type RegisterOutput struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int64        `json:"expires_in"`
	User         UserResponse `json:"user"`
}

// Register creates a new user account with a new company
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	// Registering into an existing tenant reuses its id; otherwise a new
	// company is created below.
	companyID := input.CompanyID
	newCompany := companyID == uuid.Nil
	if newCompany {
		companyID = uuid.New()
	}

	// Uniqueness is per (company_id, email), not global - the same address may
	// legitimately hold accounts in several companies. The old check rejected
	// the address if ANY tenant used it, which made a shared address (a
	// bookkeeper working for several clients) unregisterable. A brand-new
	// company has no rows yet, so the check only applies when joining one.
	if !newCompany {
		exists, err := s.userRepo.ExistsByEmail(ctx, companyID, input.Email, nil)
		if err != nil {
			s.logger.Error("registration failed: database error", zap.Error(err))
			return nil, err
		}
		if exists {
			s.logger.Debug("registration failed: email exists in company",
				zap.String("email", input.Email),
				zap.String("company_id", companyID.String()),
			)
			return nil, domain.ErrUserEmailExists
		}
	}

	// Create user with admin role (first user of company)
	user, err := domain.NewUser(companyID, input.Email, input.Password, input.Name, domain.UserRoleAdmin)
	if err != nil {
		s.logger.Error("registration failed: user creation error", zap.Error(err))
		return nil, err
	}

	// The company row has to exist before the user row: users.company_id
	// references companies(id). Previously a fresh uuid was invented and no
	// company was ever inserted, so every self-service registration died on a
	// foreign key violation that the handler reported as a generic 500 -
	// registration had never worked.
	//
	// CompanyName and BusinessNumber were declared on RegisterInput and never
	// read; they are what the new company is built from.
	if s.unitOfWork == nil {
		s.logger.Error("registration failed: unit of work not configured")
		return nil, ErrRegistrationUnavailable
	}

	err = s.unitOfWork.Do(ctx, func(repos repository.Repositories) error {
		if newCompany {
			company := &domain.Company{
				BaseModel:      domain.BaseModel{ID: companyID},
				Code:           companyCodeFromID(companyID),
				Name:           companyNameOrDefault(input.CompanyName, input.Name),
				BusinessNumber: input.BusinessNumber,
				Status:         domain.CompanyStatusTrial,
				Settings:       domain.DefaultCompanySettings(),
			}
			if err := repos.Company.Create(ctx, company); err != nil {
				return err
			}
		}
		return repos.User.Create(ctx, user)
	})
	if err != nil {
		s.logger.Error("registration failed: database error", zap.Error(err))
		return nil, err
	}

	// Generate token pair
	tokenPair, err := s.jwtService.GenerateTokenPair(
		user.ID,
		user.CompanyID,
		user.Email,
		user.Name,
		user.GetRoles(),
	)
	if err != nil {
		s.logger.Error("registration failed: token generation error", zap.Error(err))
		return nil, err
	}

	// Store refresh token
	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(s.jwtService.GetRefreshTokenTTL()),
	}
	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		s.logger.Error("registration failed: refresh token storage error", zap.Error(err))
		return nil, err
	}

	s.logger.Info("user registered",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
		zap.String("company_id", companyID.String()),
	)

	return &RegisterOutput{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
		User: UserResponse{
			ID:        user.ID,
			CompanyID: user.CompanyID,
			Email:     user.Email,
			Name:      user.Name,
			Role:      user.Role,
			Status:    user.Status,
		},
	}, nil
}

// companyCodeFromID derives a unique company code from the tenant id.
// companies.code is unique, so it cannot be left empty for the second and
// later self-service registrations.
func companyCodeFromID(companyID uuid.UUID) string {
	return "C-" + strings.ToUpper(strings.ReplaceAll(companyID.String(), "-", ""))[:12]
}

// companyNameOrDefault falls back to the registering user's name when no
// company name was supplied; companies.name is NOT NULL.
func companyNameOrDefault(companyName, userName string) string {
	if strings.TrimSpace(companyName) != "" {
		return companyName
	}
	return userName
}

// ChangePasswordInput represents password change request data
type ChangePasswordInput struct {
	UserID          uuid.UUID
	CompanyID       uuid.UUID
	CurrentPassword string
	NewPassword     string
}

// ChangePassword changes the user's password
func (s *AuthService) ChangePassword(ctx context.Context, input ChangePasswordInput) error {
	// Find user
	user, err := s.userRepo.FindByID(ctx, input.CompanyID, input.UserID)
	if err != nil {
		s.logger.Debug("change password failed: user not found", zap.String("user_id", input.UserID.String()))
		return domain.ErrUserNotFound
	}

	// Verify current password
	if !user.CheckPassword(input.CurrentPassword) {
		s.logger.Debug("change password failed: invalid current password", zap.String("user_id", input.UserID.String()))
		return domain.ErrInvalidCredentials
	}

	// Set new password
	if err := user.SetPassword(input.NewPassword); err != nil {
		s.logger.Error("change password failed: password hashing error", zap.Error(err))
		return err
	}

	// Update user
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("change password failed: database error", zap.Error(err))
		return err
	}

	// Revoke all refresh tokens to force re-login
	if err := s.refreshTokenRepo.RevokeByUserID(ctx, input.UserID); err != nil {
		s.logger.Warn("failed to revoke refresh tokens after password change", zap.Error(err))
	}

	s.logger.Info("password changed", zap.String("user_id", input.UserID.String()))

	return nil
}

// ForgotPasswordInput represents forgot password request data
type ForgotPasswordInput struct {
	Email string
}

// ForgotPasswordOutput represents forgot password response data
type ForgotPasswordOutput struct {
	ResetToken string `json:"reset_token,omitempty"` // Only returned in development mode
	Message    string `json:"message"`
}

// ForgotPassword generates a password reset token
func (s *AuthService) ForgotPassword(ctx context.Context, input ForgotPasswordInput) (*ForgotPasswordOutput, error) {
	// The response is identical whether or not the address exists, so the
	// candidate lookup only decides what happens internally.
	const genericMessage = "If an account with that email exists, a password reset link has been sent"

	candidates, err := s.userRepo.FindAuthCandidatesByEmail(ctx, input.Email)
	if err != nil {
		s.logger.Error("forgot password failed: database error", zap.Error(err))
		return nil, err
	}
	if len(candidates) == 0 {
		s.logger.Debug("forgot password: email not found", zap.String("email", input.Email))
		return &ForgotPasswordOutput{Message: genericMessage}, nil
	}
	if len(candidates) > 1 {
		// One address across several companies: there is no way to tell which
		// account the request is for without asking, and issuing a token for a
		// guessed tenant would let whoever controls the address reset a
		// stranger's password. Refused internally, indistinguishably.
		s.logger.Warn("forgot password: address spans multiple companies",
			zap.String("email", input.Email),
			zap.Int("candidates", len(candidates)),
		)
		return &ForgotPasswordOutput{Message: genericMessage}, nil
	}
	user := &candidates[0]

	// Generate reset token
	resetToken := uuid.New().String()

	s.logger.Info("password reset token generated",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
	)

	// Return token for development purposes
	// In production, this would send an email instead
	return &ForgotPasswordOutput{
		ResetToken: resetToken, // TODO: Remove in production, send via email instead
		Message:    "If an account with that email exists, a password reset link has been sent",
	}, nil
}
