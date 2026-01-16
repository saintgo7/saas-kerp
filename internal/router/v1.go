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
	// Users - Phase 4 will implement
	users := tenant.Group("/users")
	{
		users.GET("", placeholderHandler("List users"))
		users.POST("", placeholderHandler("Create user"))
		users.GET("/:id", placeholderHandler("Get user"))
		users.PUT("/:id", placeholderHandler("Update user"))
		users.DELETE("/:id", placeholderHandler("Delete user"))
	}

	// Roles - Phase 4 will implement
	roles := tenant.Group("/roles")
	{
		roles.GET("", placeholderHandler("List roles"))
		roles.POST("", placeholderHandler("Create role"))
		roles.GET("/:id", placeholderHandler("Get role"))
		roles.PUT("/:id", placeholderHandler("Update role"))
		roles.DELETE("/:id", placeholderHandler("Delete role"))
	}

	// Company - Phase 4 will implement
	company := tenant.Group("/company")
	{
		company.GET("", placeholderHandler("Get company"))
		company.PUT("", placeholderHandler("Update company"))
	}

	// Partners - Phase 4 will implement
	partners := tenant.Group("/partners")
	{
		partners.GET("", placeholderHandler("List partners"))
		partners.POST("", placeholderHandler("Create partner"))
		partners.GET("/:id", placeholderHandler("Get partner"))
		partners.PUT("/:id", placeholderHandler("Update partner"))
		partners.DELETE("/:id", placeholderHandler("Delete partner"))
	}

	// Projects - Phase 4 will implement
	projects := tenant.Group("/projects")
	{
		projects.GET("", placeholderHandler("List projects"))
		projects.POST("", placeholderHandler("Create project"))
		projects.GET("/:id", placeholderHandler("Get project"))
		projects.PUT("/:id", placeholderHandler("Update project"))
		projects.DELETE("/:id", placeholderHandler("Delete project"))
	}
}

// placeholderHandler returns a handler that indicates the endpoint is not yet implemented
func placeholderHandler(description string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(501, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "NOT_IMPLEMENTED",
				"message": description + " - implementation pending (Phase 4)",
			},
		})
	}
}
