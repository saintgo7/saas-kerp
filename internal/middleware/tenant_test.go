package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// =============================================================================
// Tenant Middleware Tests
// =============================================================================

func TestTenant_WithValidCompanyID(t *testing.T) {
	router := gin.New()
	companyID := uuid.New()

	// Set company_id before Tenant middleware
	router.Use(func(c *gin.Context) {
		appctx.SetCompanyID(c, companyID)
		c.Next()
	})
	router.Use(Tenant())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenant_WithoutCompanyID(t *testing.T) {
	router := gin.New()
	router.Use(Tenant())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenant_WithNilCompanyID(t *testing.T) {
	router := gin.New()

	// Set company_id as uuid.Nil (zero UUID)
	router.Use(func(c *gin.Context) {
		appctx.SetCompanyID(c, uuid.Nil)
		c.Next()
	})
	router.Use(Tenant())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =============================================================================
// TenantFromHeader Middleware Tests
// =============================================================================

func TestTenantFromHeader_NoHeader(t *testing.T) {
	router := gin.New()
	originalCompanyID := uuid.New()

	router.Use(func(c *gin.Context) {
		appctx.SetCompanyID(c, originalCompanyID)
		c.Next()
	})
	router.Use(TenantFromHeader())
	router.GET("/test", func(c *gin.Context) {
		resultID := appctx.GetCompanyID(c)
		c.JSON(http.StatusOK, gin.H{"company_id": resultID.String()})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantFromHeader_ValidHeaderAsSuperAdmin(t *testing.T) {
	router := gin.New()
	originalCompanyID := uuid.New()
	headerCompanyID := uuid.New()

	router.Use(func(c *gin.Context) {
		appctx.SetCompanyID(c, originalCompanyID)
		appctx.SetRoles(c, []string{"super_admin"})
		c.Next()
	})
	router.Use(TenantFromHeader())
	router.GET("/test", func(c *gin.Context) {
		resultID := appctx.GetCompanyID(c)
		c.JSON(http.StatusOK, gin.H{"company_id": resultID.String()})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Company-ID", headerCompanyID.String())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), headerCompanyID.String())
}

func TestTenantFromHeader_ValidHeaderAsNormalUser(t *testing.T) {
	router := gin.New()
	originalCompanyID := uuid.New()
	headerCompanyID := uuid.New()

	router.Use(func(c *gin.Context) {
		appctx.SetCompanyID(c, originalCompanyID)
		appctx.SetRoles(c, []string{"user"}) // Not super_admin
		c.Next()
	})
	router.Use(TenantFromHeader())
	router.GET("/test", func(c *gin.Context) {
		resultID := appctx.GetCompanyID(c)
		c.JSON(http.StatusOK, gin.H{"company_id": resultID.String()})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Company-ID", headerCompanyID.String())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Should retain original company ID, not the header value
	assert.Contains(t, w.Body.String(), originalCompanyID.String())
}

func TestTenantFromHeader_InvalidUUID(t *testing.T) {
	router := gin.New()

	router.Use(TenantFromHeader())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Company-ID", "invalid-uuid")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =============================================================================
// ValidateCompanyAccess Middleware Tests
// =============================================================================

func TestValidateCompanyAccess_NoParamCompanyID(t *testing.T) {
	router := gin.New()
	userCompanyID := uuid.New()

	router.Use(func(c *gin.Context) {
		appctx.SetCompanyID(c, userCompanyID)
		c.Next()
	})
	router.Use(ValidateCompanyAccess())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestValidateCompanyAccess_MatchingCompanyID(t *testing.T) {
	router := gin.New()
	companyID := uuid.New()

	router.Use(func(c *gin.Context) {
		appctx.SetCompanyID(c, companyID)
		c.Next()
	})
	router.Use(ValidateCompanyAccess())
	router.GET("/companies/:company_id/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/companies/"+companyID.String()+"/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestValidateCompanyAccess_DifferentCompanyID_RegularUser(t *testing.T) {
	router := gin.New()
	userCompanyID := uuid.New()
	requestedCompanyID := uuid.New()

	router.Use(func(c *gin.Context) {
		appctx.SetCompanyID(c, userCompanyID)
		appctx.SetRoles(c, []string{"user"})
		c.Next()
	})
	router.Use(ValidateCompanyAccess())
	router.GET("/companies/:company_id/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/companies/"+requestedCompanyID.String()+"/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestValidateCompanyAccess_DifferentCompanyID_SuperAdmin(t *testing.T) {
	router := gin.New()
	userCompanyID := uuid.New()
	requestedCompanyID := uuid.New()

	router.Use(func(c *gin.Context) {
		appctx.SetCompanyID(c, userCompanyID)
		appctx.SetRoles(c, []string{"super_admin"})
		c.Next()
	})
	router.Use(ValidateCompanyAccess())
	router.GET("/companies/:company_id/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/companies/"+requestedCompanyID.String()+"/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestValidateCompanyAccess_InvalidUUID(t *testing.T) {
	router := gin.New()
	userCompanyID := uuid.New()

	router.Use(func(c *gin.Context) {
		appctx.SetCompanyID(c, userCompanyID)
		c.Next()
	})
	router.Use(ValidateCompanyAccess())
	router.GET("/companies/:company_id/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/companies/invalid-uuid/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =============================================================================
// Multi-Tenancy Isolation Tests (Critical)
// =============================================================================

func TestMultiTenancy_DataIsolation(t *testing.T) {
	// This test verifies the fundamental multi-tenancy isolation
	// Each tenant (company) should only see their own data

	companyA := uuid.New()
	companyB := uuid.New()

	tests := []struct {
		name              string
		userCompanyID     uuid.UUID
		requestedPath     string
		expectedStatus    int
		description       string
	}{
		{
			name:          "Company A accessing Company A resource",
			userCompanyID: companyA,
			requestedPath: "/companies/" + companyA.String() + "/data",
			expectedStatus: http.StatusOK,
			description:   "User should access own company data",
		},
		{
			name:          "Company A accessing Company B resource",
			userCompanyID: companyA,
			requestedPath: "/companies/" + companyB.String() + "/data",
			expectedStatus: http.StatusForbidden,
			description:   "User should NOT access other company data",
		},
		{
			name:          "Company B accessing Company B resource",
			userCompanyID: companyB,
			requestedPath: "/companies/" + companyB.String() + "/data",
			expectedStatus: http.StatusOK,
			description:   "User should access own company data",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()

			router.Use(func(c *gin.Context) {
				appctx.SetCompanyID(c, tc.userCompanyID)
				appctx.SetRoles(c, []string{"user"})
				c.Next()
			})
			router.Use(ValidateCompanyAccess())
			router.GET("/companies/:company_id/data", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			req := httptest.NewRequest("GET", tc.requestedPath, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code, tc.description)
		})
	}
}
