package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/saintgo7/saas-kerp/internal/auth"
	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/errors"
)

// Auth middleware validates JWT tokens and sets user context
func Auth(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			abortWithError(c, http.StatusUnauthorized, errors.CodeUnauthorized, "Missing authorization header")
			return
		}

		// Validate Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			abortWithError(c, http.StatusUnauthorized, errors.CodeUnauthorized, "Invalid authorization format")
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			abortWithError(c, http.StatusUnauthorized, errors.CodeUnauthorized, "Empty token")
			return
		}

		// Validate token
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			if errors.Is(err, errors.ErrTokenExpired) {
				abortWithError(c, http.StatusUnauthorized, errors.CodeTokenExpired, "Token has expired")
				return
			}
			abortWithError(c, http.StatusUnauthorized, errors.CodeTokenInvalid, "Invalid token")
			return
		}

		// Set user information in context
		appctx.SetUserID(c, claims.UserID)
		appctx.SetCompanyID(c, claims.CompanyID)
		appctx.SetEmail(c, claims.Email)
		appctx.SetUserName(c, claims.Name)
		appctx.SetRoles(c, claims.Roles)

		c.Next()
	}
}

// OptionalAuth middleware validates JWT tokens if present, but doesn't require them
func OptionalAuth(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Next()
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			c.Next()
			return
		}

		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			// Token invalid but optional, continue without auth
			c.Next()
			return
		}

		// Set user information in context
		appctx.SetUserID(c, claims.UserID)
		appctx.SetCompanyID(c, claims.CompanyID)
		appctx.SetEmail(c, claims.Email)
		appctx.SetUserName(c, claims.Name)
		appctx.SetRoles(c, claims.Roles)

		c.Next()
	}
}

// RequireRoles middleware checks if the user has any of the required roles
func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !appctx.HasAnyRole(c, roles...) {
			abortWithError(c, http.StatusForbidden, errors.CodeInsufficientRole, "Insufficient permissions")
			return
		}
		c.Next()
	}
}

// RequireAdmin middleware checks if the user has admin role
func RequireAdmin() gin.HandlerFunc {
	return RequireRoles("admin")
}

// abortWithError is a helper to abort with a standardized error response
func abortWithError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
		"meta": gin.H{
			"request_id": appctx.GetRequestID(c),
		},
	})
}
