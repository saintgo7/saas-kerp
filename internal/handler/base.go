package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/database"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
)

// BaseHandler provides common functionality for all handlers
type BaseHandler struct {
	DB     *gorm.DB
	Redis  *redis.Client
	Logger *zap.Logger
}

// NewBaseHandler creates a new base handler
func NewBaseHandler(db *gorm.DB, redis *redis.Client, logger *zap.Logger) *BaseHandler {
	return &BaseHandler{
		DB:     db,
		Redis:  redis,
		Logger: logger,
	}
}

// GetUserID returns the authenticated user ID from context
func (h *BaseHandler) GetUserID(c *gin.Context) uuid.UUID {
	return appctx.GetUserID(c)
}

// GetCompanyID returns the company ID from context
func (h *BaseHandler) GetCompanyID(c *gin.Context) uuid.UUID {
	return appctx.GetCompanyID(c)
}

// GetEmail returns the user email from context
func (h *BaseHandler) GetEmail(c *gin.Context) string {
	return appctx.GetEmail(c)
}

// ScopedDB returns a database instance scoped to the current company
func (h *BaseHandler) ScopedDB(c *gin.Context) *gorm.DB {
	return database.ScopedDB(h.DB, h.GetCompanyID(c))
}

// BindJSON binds JSON body to the given struct and handles errors
func (h *BaseHandler) BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		response.BadRequest(c, err.Error())
		return false
	}
	return true
}

// ParseUUID parses a UUID from URL parameter
func (h *BaseHandler) ParseUUID(c *gin.Context, param string) (uuid.UUID, bool) {
	idStr := c.Param(param)
	if idStr == "" {
		response.BadRequest(c, "Missing "+param+" parameter")
		return uuid.Nil, false
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid UUID format for "+param)
		return uuid.Nil, false
	}

	return id, true
}

// ParseQueryUUID parses a UUID from query parameter
func (h *BaseHandler) ParseQueryUUID(c *gin.Context, param string) (uuid.UUID, bool) {
	idStr := c.Query(param)
	if idStr == "" {
		return uuid.Nil, true // Empty is OK for optional params
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid UUID format for "+param)
		return uuid.Nil, false
	}

	return id, true
}

// GetLogger returns the request-scoped logger
func (h *BaseHandler) GetLogger(c *gin.Context) *zap.Logger {
	logger := appctx.GetLogger(c)
	if logger != nil {
		return logger
	}
	return h.Logger
}

// HasRole checks if the current user has the specified role
func (h *BaseHandler) HasRole(c *gin.Context, role string) bool {
	return appctx.HasRole(c, role)
}

// IsAdmin checks if the current user is an admin
func (h *BaseHandler) IsAdmin(c *gin.Context) bool {
	return appctx.HasRole(c, "admin")
}
