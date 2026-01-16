package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents JWT token claims
type Claims struct {
	jwt.RegisteredClaims

	// User identification
	UserID    uuid.UUID `json:"user_id"`
	CompanyID uuid.UUID `json:"company_id"`

	// User info
	Email string `json:"email"`
	Name  string `json:"name"`

	// Authorization
	Roles []string `json:"roles"`

	// Token type
	TokenType TokenType `json:"token_type,omitempty"`
}

// TokenType represents the type of JWT token
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// HasRole checks if the claims contain a specific role
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the claims contain any of the specified roles
func (c *Claims) HasAnyRole(roles ...string) bool {
	roleSet := make(map[string]bool)
	for _, r := range c.Roles {
		roleSet[r] = true
	}
	for _, r := range roles {
		if roleSet[r] {
			return true
		}
	}
	return false
}

// IsAdmin checks if the user has admin role
func (c *Claims) IsAdmin() bool {
	return c.HasRole("admin")
}

// TokenPair represents access and refresh token pair
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}
