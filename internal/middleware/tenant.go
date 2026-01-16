package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
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

// TenantFromHeader extracts company ID from X-Company-ID header
// Use this for super admin operations that can switch between tenants
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

		// Override the company ID from JWT with the header value
		// This should only be allowed for super admins
		if appctx.HasRole(c, "super_admin") {
			appctx.SetCompanyID(c, companyID)
		}

		c.Next()
	}
}

// ValidateCompanyAccess ensures the user has access to the requested company
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
