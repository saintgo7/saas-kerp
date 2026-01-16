package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/auth"
	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	*BaseHandler
	jwtService *auth.JWTService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *gorm.DB, redis *redis.Client, logger *zap.Logger, jwtService *auth.JWTService) *AuthHandler {
	return &AuthHandler{
		BaseHandler: NewBaseHandler(db, redis, logger),
		jwtService:  jwtService,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
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

	// TODO: Implement login logic in Phase 4 (service layer)
	// This is a placeholder that shows the API structure

	response.OK(c, gin.H{
		"message": "Login endpoint - implementation pending (Phase 4)",
		"email":   req.Email,
	})
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

	// TODO: Implement refresh logic in Phase 4 (service layer)

	response.OK(c, gin.H{
		"message": "Refresh endpoint - implementation pending (Phase 4)",
	})
}

// Logout handles user logout
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// TODO: Implement logout logic (invalidate refresh token)

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

	// TODO: Implement password change logic in Phase 4

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

	// TODO: Implement registration logic in Phase 4

	response.Created(c, gin.H{
		"message": "Registration endpoint - implementation pending (Phase 4)",
		"email":   req.Email,
	})
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

	// TODO: Implement forgot password logic in Phase 4

	// Always return success to prevent email enumeration
	response.OK(c, gin.H{
		"message": "If an account with that email exists, a password reset link has been sent",
	})
}
