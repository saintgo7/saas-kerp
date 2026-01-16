package context

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GetRequestID returns the request ID from context
func GetRequestID(c *gin.Context) string {
	if v, exists := c.Get(KeyRequestID); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SetRequestID sets the request ID in context
func SetRequestID(c *gin.Context, id string) {
	c.Set(KeyRequestID, id)
}

// GetUserID returns the user ID from context
func GetUserID(c *gin.Context) uuid.UUID {
	if v, exists := c.Get(KeyUserID); exists {
		if id, ok := v.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}

// SetUserID sets the user ID in context
func SetUserID(c *gin.Context, id uuid.UUID) {
	c.Set(KeyUserID, id)
}

// GetCompanyID returns the company ID from context
func GetCompanyID(c *gin.Context) uuid.UUID {
	if v, exists := c.Get(KeyCompanyID); exists {
		if id, ok := v.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}

// SetCompanyID sets the company ID in context
func SetCompanyID(c *gin.Context, id uuid.UUID) {
	c.Set(KeyCompanyID, id)
}

// GetEmail returns the user email from context
func GetEmail(c *gin.Context) string {
	if v, exists := c.Get(KeyEmail); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SetEmail sets the user email in context
func SetEmail(c *gin.Context, email string) {
	c.Set(KeyEmail, email)
}

// GetUserName returns the user name from context
func GetUserName(c *gin.Context) string {
	if v, exists := c.Get(KeyUserName); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SetUserName sets the user name in context
func SetUserName(c *gin.Context, name string) {
	c.Set(KeyUserName, name)
}

// GetRoles returns the user roles from context
func GetRoles(c *gin.Context) []string {
	if v, exists := c.Get(KeyRoles); exists {
		if roles, ok := v.([]string); ok {
			return roles
		}
	}
	return nil
}

// SetRoles sets the user roles in context
func SetRoles(c *gin.Context, roles []string) {
	c.Set(KeyRoles, roles)
}

// HasRole checks if the user has a specific role
func HasRole(c *gin.Context, role string) bool {
	roles := GetRoles(c)
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the user has any of the specified roles
func HasAnyRole(c *gin.Context, roles ...string) bool {
	userRoles := GetRoles(c)
	roleSet := make(map[string]bool)
	for _, r := range userRoles {
		roleSet[r] = true
	}
	for _, r := range roles {
		if roleSet[r] {
			return true
		}
	}
	return false
}

// GetStartTime returns the request start time from context
func GetStartTime(c *gin.Context) time.Time {
	if v, exists := c.Get(KeyStartTime); exists {
		if t, ok := v.(time.Time); ok {
			return t
		}
	}
	return time.Time{}
}

// SetStartTime sets the request start time in context
func SetStartTime(c *gin.Context, t time.Time) {
	c.Set(KeyStartTime, t)
}

// GetLogger returns the logger from context
func GetLogger(c *gin.Context) *zap.Logger {
	if v, exists := c.Get(KeyLogger); exists {
		if l, ok := v.(*zap.Logger); ok {
			return l
		}
	}
	return nil
}

// SetLogger sets the logger in context
func SetLogger(c *gin.Context, logger *zap.Logger) {
	c.Set(KeyLogger, logger)
}

// GetClientIP returns the client IP from context
func GetClientIP(c *gin.Context) string {
	if v, exists := c.Get(KeyClientIP); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return c.ClientIP()
}

// SetClientIP sets the client IP in context
func SetClientIP(c *gin.Context, ip string) {
	c.Set(KeyClientIP, ip)
}

// IsAuthenticated returns true if the user is authenticated
func IsAuthenticated(c *gin.Context) bool {
	return GetUserID(c) != uuid.Nil
}

// IsTenantScoped returns true if the request has a company context
func IsTenantScoped(c *gin.Context) bool {
	return GetCompanyID(c) != uuid.Nil
}
