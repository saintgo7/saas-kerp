package router

import (
	"github.com/gin-gonic/gin"

	"github.com/saintgo7/saas-kerp/internal/auth"
	"github.com/saintgo7/saas-kerp/internal/config"
	"github.com/saintgo7/saas-kerp/internal/handler"
	"github.com/saintgo7/saas-kerp/internal/middleware"
)

// Credential endpoints get their own, much tighter token bucket than the global
// limiter: 1 request/second with a burst of 10 per client address.
const (
	credentialRatePerSecond = 1
	credentialBurst         = 10
)

// RegisterV1Routes registers all API v1 routes.
//
// tenantSession may be nil, in which case tenant routes run without the
// PostgreSQL session variable that the RLS policies read.
func RegisterV1Routes(api *gin.RouterGroup, jwtService *auth.JWTService, h *handler.Handlers, tenantSession gin.HandlerFunc, rateLimit *config.RateLimitConfig) {
	v1 := api.Group("/v1")

	// Public routes (no authentication required)
	registerPublicRoutes(v1, h, rateLimit)

	// Protected routes (authentication required)
	protected := v1.Group("")
	protected.Use(middleware.Auth(jwtService))
	registerProtectedRoutes(protected, h)

	// Tenant-scoped routes (authentication + company context required)
	tenant := v1.Group("")
	tenant.Use(middleware.Auth(jwtService))
	tenant.Use(middleware.Tenant())
	if tenantSession != nil {
		// Must come after Auth and Tenant: it needs the company ID they put in
		// the context.
		tenant.Use(tenantSession)
	}
	registerTenantRoutes(tenant, h)
}

// registerPublicRoutes registers routes that don't require authentication.
func registerPublicRoutes(v1 *gin.RouterGroup, h *handler.Handlers, rateLimit *config.RateLimitConfig) {
	authGroup := v1.Group("/auth")
	if rateLimit != nil {
		authGroup.Use(middleware.RateLimitCredentials(rateLimit, credentialRatePerSecond, credentialBurst))
	}
	{
		authGroup.POST("/login", h.Auth.Login)
		authGroup.POST("/register", h.Auth.Register)
		authGroup.POST("/forgot-password", h.Auth.ForgotPassword)

		// Token refresh is public on purpose. The refresh token in the request
		// body is itself the credential, and requiring a valid access token here
		// makes refreshing impossible in the only situation it is needed: after
		// the access token has expired. The handler reads nothing from the
		// authentication context.
		authGroup.POST("/refresh", h.Auth.Refresh)
	}
}

// registerProtectedRoutes registers routes that require authentication but not tenant context
func registerProtectedRoutes(protected *gin.RouterGroup, h *handler.Handlers) {
	authGroup := protected.Group("/auth")
	{
		authGroup.POST("/logout", h.Auth.Logout)
		authGroup.GET("/me", h.Auth.Me)
		authGroup.PUT("/password", h.Auth.ChangePassword)
	}
}

// registerTenantRoutes registers routes that require both authentication and tenant context
func registerTenantRoutes(tenant *gin.RouterGroup, h *handler.Handlers) {
	// Accounting routes
	h.Account.RegisterRoutes(tenant)
	h.Partner.RegisterRoutes(tenant)
	h.Voucher.RegisterRoutes(tenant)
	h.Ledger.RegisterRoutes(tenant)

	// User & role management routes. RequireAdmin is applied inside each
	// handler's own RegisterRoutes (user_handler.go / role_handler.go), so the
	// group is registered directly rather than wrapped a second time here.
	h.User.RegisterRoutes(tenant)
	h.Role.RegisterRoutes(tenant)

	// Company settings routes
	h.Company.RegisterRoutes(tenant)

	// Project management routes
	h.Project.RegisterRoutes(tenant)

	// Tax invoice routes. RegisterRoutes puts RequireWriter on
	// create/update/delete and RequireApprover on issue, transmit, cancel and
	// sync.
	h.TaxInvoice.RegisterRoutes(tenant)

	// HR: employees, positions, departments, leave.
	RegisterHRRoutes(tenant, h.HR)

	// Payroll and 4대보험. Reads are RequireWriter here, unlike the rest of the
	// API, because they expose salary figures.
	RegisterPayrollRoutes(tenant, h.Payroll)

	// Inventory: products, stock and purchase/sales orders.
	RegisterInventoryRoutes(tenant, NewInventoryHandlers(h.Product, h.Stock, h.Order))
}
