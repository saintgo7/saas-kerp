package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/database"
	"github.com/saintgo7/saas-kerp/internal/errors"
)

// Tenant middleware ensures the request has a valid company context
func Tenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		companyID := appctx.GetCompanyID(c)

		if companyID == uuid.Nil {
			abortWithError(c, http.StatusBadRequest, errors.CodeTenantMismatch, "Company context required")
			return
		}

		c.Next()
	}
}

// TenantSession pins one PostgreSQL connection to the request, sets the
// app.current_tenant session variable on it and routes every statement of the
// request to that connection, so the Row Level Security policies defined in
// db/migrations/000010_rls_policies.up.sql actually apply.
//
// It must run after Auth and Tenant, because it needs the company ID that Auth
// puts in the context.
//
// Failure mode: if the connection cannot be pinned or the session variable
// cannot be set, the request is rejected with 503 rather than continuing without
// tenant isolation. Continuing would mean serving the request with no RLS
// predicate, which is precisely the condition this middleware exists to prevent.
//
// Pass a nil *gorm.DB (or disable database.tenant_guc) to make this a no-op.
func TenantSession(db *gorm.DB, logger *zap.Logger) gin.HandlerFunc {
	if db == nil {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		companyID := appctx.GetCompanyID(c)
		if companyID == uuid.Nil {
			// Tenant() already rejects this case; be defensive rather than pinning
			// a connection with an empty tenant.
			c.Next()
			return
		}

		ctx, release, err := database.AcquireTenantConn(c.Request.Context(), db, companyID)
		if err != nil {
			if logger != nil {
				logger.Error("failed to establish tenant database session",
					zap.String("request_id", appctx.GetRequestID(c)),
					zap.String("company_id", companyID.String()),
					zap.Error(err),
				)
			}
			abortWithError(c, http.StatusServiceUnavailable, errors.CodeUnavailable, "Service temporarily unavailable")
			return
		}
		defer release()

		// Repositories reach the pinned connection through this context: they all
		// call db.WithContext(ctx), and the GORM connection pool installed by
		// database.NewPostgresDB honours the pin.
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// TenantFromHeader extracts company ID from X-Company-ID header.
//
// SECURITY: this middleware lets the client name its own tenant. It is not
// registered on any route (see internal/router/) and no code path grants the
// "super_admin" role that gates it, so it is inert today. It must not be wired
// up as-is: the header company must be checked against the user's actual
// membership in the database before it is trusted. Prefer deleting it.
func TenantFromHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		headerCompanyID := c.GetHeader("X-Company-ID")
		if headerCompanyID == "" {
			c.Next()
			return
		}

		companyID, err := uuid.Parse(headerCompanyID)
		if err != nil {
			abortWithError(c, http.StatusBadRequest, errors.CodeValidation, "Invalid company ID format")
			return
		}

		// Override the company ID from JWT with the header value.
		// This is only allowed for super admins.
		if appctx.HasRole(c, "super_admin") {
			appctx.SetCompanyID(c, companyID)
		}

		c.Next()
	}
}

// ValidateCompanyAccess ensures the user has access to the requested company
// identified by a :company_id path parameter.
func ValidateCompanyAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get company ID from URL parameter
		paramCompanyID := c.Param("company_id")
		if paramCompanyID == "" {
			c.Next()
			return
		}

		requestedCompanyID, err := uuid.Parse(paramCompanyID)
		if err != nil {
			abortWithError(c, http.StatusBadRequest, errors.CodeValidation, "Invalid company ID format")
			return
		}

		userCompanyID := appctx.GetCompanyID(c)

		// Super admins can access any company
		if appctx.HasRole(c, "super_admin") {
			c.Next()
			return
		}

		// Regular users can only access their own company
		if requestedCompanyID != userCompanyID {
			abortWithError(c, http.StatusForbidden, errors.CodeTenantMismatch, "Access denied to this company")
			return
		}

		c.Next()
	}
}
