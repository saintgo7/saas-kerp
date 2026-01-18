package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

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
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtService *auth.JWTService,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtService:       jwtService,
		logger:           logger,
	}
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
	// Find user by email
	user, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		if err == domain.ErrUserNotFound {
			s.logger.Debug("login failed: user not found", zap.String("email", input.Email))
			return nil, domain.ErrInvalidCredentials
		}
		s.logger.Error("login failed: database error", zap.Error(err))
		return nil, err
	}

	// Check password
	if !user.CheckPassword(input.Password) {
		s.logger.Debug("login failed: invalid password", zap.String("email", input.Email))
		return nil, domain.ErrInvalidCredentials
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

	// Store refresh token
	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(s.jwtService.GetRefreshTokenTTL()),
	}
	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		s.logger.Error("login failed: refresh token storage error", zap.Error(err))
		return nil, err
	}

	// Update last login
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn("failed to update last login", zap.Error(err))
		// Don't fail login for this error
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
	// Find refresh token
	rt, err := s.refreshTokenRepo.FindByToken(ctx, input.RefreshToken)
	if err != nil {
		if err == domain.ErrRefreshTokenNotFound {
			return nil, domain.ErrRefreshTokenNotFound
		}
		return nil, err
	}

	// Check if expired
	if rt.IsExpired() {
		return nil, domain.ErrRefreshTokenExpired
	}

	// Find user
	user, err := s.userRepo.FindByEmail(ctx, "")
	if err != nil {
		// We need to find user by ID, let's use a direct query
		return nil, err
	}

	// For now, we'll revoke the old refresh token
	if err := s.refreshTokenRepo.RevokeByToken(ctx, input.RefreshToken); err != nil {
		s.logger.Warn("failed to revoke old refresh token", zap.Error(err))
	}

	// Generate new token pair
	tokenPair, err := s.jwtService.GenerateTokenPair(
		rt.UserID,
		user.CompanyID,
		user.Email,
		user.Name,
		user.GetRoles(),
	)
	if err != nil {
		return nil, err
	}

	// Store new refresh token
	newRefreshToken := &domain.RefreshToken{
		UserID:    rt.UserID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(s.jwtService.GetRefreshTokenTTL()),
	}
	if err := s.refreshTokenRepo.Create(ctx, newRefreshToken); err != nil {
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
	// Check if email already exists
	_, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err == nil {
		s.logger.Debug("registration failed: email exists", zap.String("email", input.Email))
		return nil, domain.ErrUserEmailExists
	}
	if err != domain.ErrUserNotFound {
		s.logger.Error("registration failed: database error", zap.Error(err))
		return nil, err
	}

	// Generate company ID if not provided
	companyID := input.CompanyID
	if companyID == uuid.Nil {
		companyID = uuid.New()
	}

	// Create user with admin role (first user of company)
	user, err := domain.NewUser(companyID, input.Email, input.Password, input.Name, domain.UserRoleAdmin)
	if err != nil {
		s.logger.Error("registration failed: user creation error", zap.Error(err))
		return nil, err
	}

	// Save user to database
	if err := s.userRepo.Create(ctx, user); err != nil {
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
	// Find user by email (don't reveal if email exists)
	user, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		if err == domain.ErrUserNotFound {
			// Don't reveal if email exists
			s.logger.Debug("forgot password: email not found", zap.String("email", input.Email))
			return &ForgotPasswordOutput{
				Message: "If an account with that email exists, a password reset link has been sent",
			}, nil
		}
		s.logger.Error("forgot password failed: database error", zap.Error(err))
		return nil, err
	}

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
