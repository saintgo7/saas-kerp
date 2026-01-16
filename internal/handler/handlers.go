package handler

import (
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/auth"
)

// Handlers holds all HTTP handlers
type Handlers struct {
	Health *HealthHandler
	Auth   *AuthHandler
	// Phase 4 will add more handlers:
	// User    *UserHandler
	// Role    *RoleHandler
	// Company *CompanyHandler
	// Partner *PartnerHandler
}

// NewHandlers creates all handlers
func NewHandlers(db *gorm.DB, redis *redis.Client, logger *zap.Logger, jwtService *auth.JWTService, version string) *Handlers {
	return &Handlers{
		Health: NewHealthHandler(db, redis, logger, version),
		Auth:   NewAuthHandler(db, redis, logger, jwtService),
	}
}
