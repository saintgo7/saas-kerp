package router

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

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
	db         *gorm.DB
}

// New creates a new router with all middleware and routes configured.
//
// db is used by the tenant session middleware to set the PostgreSQL session
// variable the RLS policies read; pass nil (or disable database.tenant_guc) to
// leave that middleware out.
func New(cfg *config.Config, logger *zap.Logger, jwtService *auth.JWTService, handlers *handler.Handlers, db *gorm.DB) *Router {
	// Set Gin mode based on environment. Anything that is not development runs
	// in release mode: the previous form left "staging" on gin's debug default,
	// which prints routes and stack traces and enables development-only
	// behaviour in handlers.
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	r := &Router{
		engine:     engine,
		config:     cfg,
		logger:     logger,
		jwtService: jwtService,
		handlers:   handlers,
		db:         db,
	}

	r.setupTrustedProxies()
	r.setupMiddleware()
	r.setupRoutes()

	return r
}

// setupTrustedProxies decides whose X-Forwarded-For / X-Real-IP headers are
// believed.
//
// gin's default is to trust every peer (0.0.0.0/0 and ::/0), which means
// c.ClientIP() returns whatever the client put in X-Forwarded-For. That forges
// the rate limiter's bucket key and every client_ip in the audit log at the same
// time. An empty app.trusted_proxies therefore turns proxy trust off entirely
// and falls back to the real peer address; deployments behind nginx or Traefik
// must list the proxy network instead.
func (r *Router) setupTrustedProxies() {
	proxies := r.config.App.TrustedProxies

	if len(proxies) == 0 {
		r.engine.ForwardedByClientIP = false
		if err := r.engine.SetTrustedProxies(nil); err != nil {
			r.logger.Error("failed to clear trusted proxies", zap.Error(err))
		}
		r.logger.Info("proxy headers are not trusted; client IP is the peer address")
		return
	}

	r.engine.ForwardedByClientIP = true
	if err := r.engine.SetTrustedProxies(proxies); err != nil {
		// Fail closed: an unparseable CIDR must not leave the wide-open default
		// in place.
		r.logger.Error("invalid app.trusted_proxies; disabling proxy trust", zap.Error(err))
		r.engine.ForwardedByClientIP = false
		_ = r.engine.SetTrustedProxies(nil)
		return
	}
	r.logger.Info("trusting proxy headers from configured networks", zap.Strings("trusted_proxies", proxies))
}

// setupMiddleware configures the middleware chain
func (r *Router) setupMiddleware() {
	// Request ID must be first
	r.engine.Use(middleware.RequestID())

	// Logger (skip health check endpoints)
	r.engine.Use(middleware.Logger(r.logger))

	// Recovery from panics
	r.engine.Use(middleware.Recovery(r.logger))

	// Cap the request body before any handler reads it.
	r.engine.Use(middleware.BodyLimit(r.config.App.MaxRequestBodyBytes))

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
	RegisterV1Routes(api, r.jwtService, r.handlers, r.tenantSession(), &r.config.RateLimit)
}

// tenantSession returns the middleware that establishes the per-request tenant
// database session, or nil when it is disabled.
func (r *Router) tenantSession() gin.HandlerFunc {
	if r.db == nil || !r.config.Database.TenantGUC {
		return nil
	}
	return middleware.TenantSession(r.db, r.logger)
}

// Engine returns the underlying gin.Engine
func (r *Router) Engine() *gin.Engine {
	return r.engine
}
