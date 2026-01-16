package router

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/saintgo7/saas-kerp/internal/auth"
	"github.com/saintgo7/saas-kerp/internal/config"
	"github.com/saintgo7/saas-kerp/internal/handler"
	"github.com/saintgo7/saas-kerp/internal/middleware"
)

// Router wraps gin.Engine with additional configuration
type Router struct {
	engine     *gin.Engine
	config     *config.Config
	logger     *zap.Logger
	jwtService *auth.JWTService
	handlers   *handler.Handlers
}

// New creates a new router with all middleware and routes configured
func New(cfg *config.Config, logger *zap.Logger, jwtService *auth.JWTService, handlers *handler.Handlers) *Router {
	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	}

	engine := gin.New()

	r := &Router{
		engine:     engine,
		config:     cfg,
		logger:     logger,
		jwtService: jwtService,
		handlers:   handlers,
	}

	r.setupMiddleware()
	r.setupRoutes()

	return r
}

// setupMiddleware configures the middleware chain
func (r *Router) setupMiddleware() {
	// Request ID must be first
	r.engine.Use(middleware.RequestID())

	// Logger (skip health check endpoints)
	r.engine.Use(middleware.Logger(r.logger))

	// Recovery from panics
	r.engine.Use(middleware.Recovery(r.logger))

	// CORS
	r.engine.Use(middleware.CORS(&r.config.CORS))

	// Rate limiting (if enabled)
	if r.config.RateLimit.Enabled {
		r.engine.Use(middleware.RateLimit(&r.config.RateLimit))
	}
}

// setupRoutes configures all routes
func (r *Router) setupRoutes() {
	// Health check endpoints (no auth required)
	r.engine.GET("/health", r.handlers.Health.Check)
	r.engine.GET("/health/ready", r.handlers.Health.Ready)
	r.engine.GET("/health/live", r.handlers.Health.Live)

	// API routes
	api := r.engine.Group("/api")

	// Register v1 routes
	RegisterV1Routes(api, r.jwtService, r.handlers)
}

// Engine returns the underlying gin.Engine
func (r *Router) Engine() *gin.Engine {
	return r.engine
}
