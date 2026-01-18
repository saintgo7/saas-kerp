package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/auth"
	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	*BaseHandler
	jwtService  *auth.JWTService
	authService *service.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *gorm.DB, redis *redis.Client, logger *zap.Logger, jwtService *auth.JWTService) *AuthHandler {
	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)

	// Initialize auth service
	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtService, logger)

	return &AuthHandler{
		BaseHandler: NewBaseHandler(db, redis, logger),
		jwtService:  jwtService,
		authService: authService,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int64       `json:"expires_in"`
	User         interface{} `json:"user"`
}

// Login handles user login
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if !h.BindJSON(c, &req) {
		return
	}

	result, err := h.authService.Login(c.Request.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		switch err {
		case domain.ErrInvalidCredentials:
			response.Unauthorized(c, "Invalid email or password")
		case domain.ErrUserInactive:
			response.Forbidden(c, "User account is inactive")
		case domain.ErrUserLocked:
			response.Forbidden(c, "User account is locked")
		default:
			h.Logger.Error("login failed", zap.Error(err))
			response.InternalError(c, "Login failed")
		}
		return
	}

	response.OK(c, result)
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh handles token refresh
// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !h.BindJSON(c, &req) {
		return
	}

	result, err := h.authService.Refresh(c.Request.Context(), service.RefreshInput{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		switch err {
		case domain.ErrRefreshTokenNotFound:
			response.Unauthorized(c, "Invalid refresh token")
		case domain.ErrRefreshTokenExpired:
			response.Unauthorized(c, "Refresh token expired")
		default:
			h.Logger.Error("refresh failed", zap.Error(err))
			response.InternalError(c, "Token refresh failed")
		}
		return
	}

	response.OK(c, result)
}

// Logout handles user logout
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := appctx.GetUserID(c)

	if err := h.authService.Logout(c.Request.Context(), userID); err != nil {
		h.Logger.Warn("logout: failed to revoke tokens", zap.Error(err))
		// Don't fail the logout
	}

	response.OK(c, gin.H{
		"message": "Logged out successfully",
	})
}

// Me returns the current user information
// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID := appctx.GetUserID(c)
	companyID := appctx.GetCompanyID(c)
	email := appctx.GetEmail(c)
	name := appctx.GetUserName(c)
	roles := appctx.GetRoles(c)

	response.OK(c, gin.H{
		"user_id":    userID,
		"company_id": companyID,
		"email":      email,
		"name":       name,
		"roles":      roles,
	})
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword handles password change
// PUT /api/v1/auth/password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if !h.BindJSON(c, &req) {
		return
	}

	// Validate new password
	if errors := auth.ValidatePassword(req.NewPassword); len(errors) > 0 {
		details := make([]response.FieldError, len(errors))
		for i, err := range errors {
			details[i] = response.FieldError{Field: "new_password", Message: err}
		}
		response.ValidationError(c, details)
		return
	}

	// Get user ID and company ID from context
	userID := appctx.GetUserID(c)
	companyID := appctx.GetCompanyID(c)

	// Call service to change password
	err := h.authService.ChangePassword(c.Request.Context(), service.ChangePasswordInput{
		UserID:          userID,
		CompanyID:       companyID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		switch err {
		case domain.ErrUserNotFound:
			response.NotFound(c, "User not found")
		case domain.ErrInvalidCredentials:
			response.Unauthorized(c, "Current password is incorrect")
		case domain.ErrPasswordTooShort:
			response.BadRequest(c, "New password must be at least 8 characters")
		default:
			h.Logger.Error("password change failed", zap.Error(err))
			response.InternalError(c, "Password change failed")
		}
		return
	}

	response.OK(c, gin.H{
		"message": "Password changed successfully",
	})
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	CompanyName    string `json:"company_name" binding:"required"`
	BusinessNumber string `json:"business_number" binding:"required,len=10"`
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=8"`
	Name           string `json:"name" binding:"required"`
	Phone          string `json:"phone"`
}

// Register handles user registration
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if !h.BindJSON(c, &req) {
		return
	}

	// Validate password
	if errors := auth.ValidatePassword(req.Password); len(errors) > 0 {
		details := make([]response.FieldError, len(errors))
		for i, err := range errors {
			details[i] = response.FieldError{Field: "password", Message: err}
		}
		response.ValidationError(c, details)
		return
	}

	// Call service to register user
	result, err := h.authService.Register(c.Request.Context(), service.RegisterInput{
		CompanyName:    req.CompanyName,
		BusinessNumber: req.BusinessNumber,
		Email:          req.Email,
		Password:       req.Password,
		Name:           req.Name,
		Phone:          req.Phone,
	})
	if err != nil {
		switch err {
		case domain.ErrUserEmailExists:
			response.Conflict(c, "Email already registered")
		case domain.ErrEmailRequired:
			response.BadRequest(c, "Email is required")
		case domain.ErrPasswordRequired:
			response.BadRequest(c, "Password is required")
		case domain.ErrNameRequired:
			response.BadRequest(c, "Name is required")
		case domain.ErrPasswordTooShort:
			response.BadRequest(c, "Password must be at least 8 characters")
		default:
			h.Logger.Error("registration failed", zap.Error(err))
			response.InternalError(c, "Registration failed")
		}
		return
	}

	response.Created(c, result)
}

// ForgotPasswordRequest represents a forgot password request
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPassword handles forgot password
// POST /api/v1/auth/forgot-password
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !h.BindJSON(c, &req) {
		return
	}

	// Call service to generate reset token
	result, err := h.authService.ForgotPassword(c.Request.Context(), service.ForgotPasswordInput{
		Email: req.Email,
	})
	if err != nil {
		h.Logger.Error("forgot password failed", zap.Error(err))
		// Don't reveal internal errors, always return success message
		response.OK(c, gin.H{
			"message": "If an account with that email exists, a password reset link has been sent",
		})
		return
	}

	// Store reset token in Redis with 1 hour expiration
	if result.ResetToken != "" {
		redisKey := "password_reset:" + result.ResetToken
		if err := h.Redis.Set(c.Request.Context(), redisKey, req.Email, time.Hour).Err(); err != nil {
			h.Logger.Error("failed to store reset token in Redis", zap.Error(err))
			// Continue anyway, don't reveal error to user
		}
	}

	// TODO: Send actual email in production
	// For development, include the token in response
	responseData := gin.H{
		"message": result.Message,
	}
	if result.ResetToken != "" {
		// Development only: include reset token in response
		responseData["reset_token"] = result.ResetToken
		responseData["note"] = "Development mode: Token included in response. In production, this will be sent via email."
	}

	response.OK(c, responseData)
}
