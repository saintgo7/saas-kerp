package router

import (
	"github.com/gin-gonic/gin"

	"github.com/saintgo7/saas-kerp/internal/auth"
	"github.com/saintgo7/saas-kerp/internal/handler"
	"github.com/saintgo7/saas-kerp/internal/middleware"
)

// RegisterV1Routes registers all API v1 routes
func RegisterV1Routes(api *gin.RouterGroup, jwtService *auth.JWTService, h *handler.Handlers) {
	v1 := api.Group("/v1")

	// Public routes (no authentication required)
	registerPublicRoutes(v1, h)

	// Protected routes (authentication required)
	protected := v1.Group("")
	protected.Use(middleware.Auth(jwtService))
	registerProtectedRoutes(protected, h)

	// Tenant-scoped routes (authentication + company context required)
	tenant := v1.Group("")
	tenant.Use(middleware.Auth(jwtService))
	tenant.Use(middleware.Tenant())
	registerTenantRoutes(tenant, h)
}

// registerPublicRoutes registers routes that don't require authentication
func registerPublicRoutes(v1 *gin.RouterGroup, h *handler.Handlers) {
	auth := v1.Group("/auth")
	{
		auth.POST("/login", h.Auth.Login)
		auth.POST("/register", h.Auth.Register)
		auth.POST("/forgot-password", h.Auth.ForgotPassword)
	}
}

// registerProtectedRoutes registers routes that require authentication but not tenant context
func registerProtectedRoutes(protected *gin.RouterGroup, h *handler.Handlers) {
	auth := protected.Group("/auth")
	{
		auth.POST("/refresh", h.Auth.Refresh)
		auth.POST("/logout", h.Auth.Logout)
		auth.GET("/me", h.Auth.Me)
		auth.PUT("/password", h.Auth.ChangePassword)
	}
}

// registerTenantRoutes registers routes that require both authentication and tenant context
func registerTenantRoutes(tenant *gin.RouterGroup, h *handler.Handlers) {
	// Accounting routes
	h.Account.RegisterRoutes(tenant)
	h.Partner.RegisterRoutes(tenant)
	h.Voucher.RegisterRoutes(tenant)
	h.Ledger.RegisterRoutes(tenant)

	// User management routes
	h.User.RegisterRoutes(tenant)

	// Role management routes
	h.Role.RegisterRoutes(tenant)

	// Company settings routes
	h.Company.RegisterRoutes(tenant)

	// Project management routes
	h.Project.RegisterRoutes(tenant)
}

