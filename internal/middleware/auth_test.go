package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saintgo7/saas-kerp/internal/auth"
	"github.com/saintgo7/saas-kerp/internal/config"
	appctx "github.com/saintgo7/saas-kerp/internal/context"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// testJWTConfig creates a test JWT configuration
func testJWTConfig() *config.JWTConfig {
	return &config.JWTConfig{
		Secret:          "test-secret-key-for-testing-purpose",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		Issuer:          "test-issuer",
	}
}

// =============================================================================
// Auth Middleware Tests
// =============================================================================

func TestAuth_ValidToken(t *testing.T) {
	cfg := testJWTConfig()
	jwtService := auth.NewJWTService(cfg)

	userID := uuid.New()
	companyID := uuid.New()
	email := "test@example.com"
	name := "Test User"
	roles := []string{"user", "admin"}

	token, err := jwtService.GenerateAccessToken(userID, companyID, email, name, roles)
	require.NoError(t, err)

	router := gin.New()
	router.Use(Auth(jwtService))
	router.GET("/test", func(c *gin.Context) {
		resultUserID := appctx.GetUserID(c)
		resultCompanyID := appctx.GetCompanyID(c)
		resultEmail := appctx.GetEmail(c)
		resultName := appctx.GetUserName(c)
		resultRoles := appctx.GetRoles(c)

		assert.Equal(t, userID, resultUserID)
		assert.Equal(t, companyID, resultCompanyID)
		assert.Equal(t, email, resultEmail)
		assert.Equal(t, name, resultName)
		assert.Equal(t, roles, resultRoles)

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuth_MissingAuthorizationHeader(t *testing.T) {
	cfg := testJWTConfig()
	jwtService := auth.NewJWTService(cfg)

	router := gin.New()
	router.Use(Auth(jwtService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_InvalidAuthorizationFormat(t *testing.T) {
	cfg := testJWTConfig()
	jwtService := auth.NewJWTService(cfg)

	router := gin.New()
	router.Use(Auth(jwtService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	tests := []struct {
		name   string
		header string
	}{
		{"No Bearer prefix", "some-token"},
		{"Wrong prefix", "Basic some-token"},
		{"Empty token", "Bearer "},
		{"Only Bearer", "Bearer"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tc.header)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	cfg := testJWTConfig()
	jwtService := auth.NewJWTService(cfg)

	router := gin.New()
	router.Use(Auth(jwtService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-string")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_ExpiredToken(t *testing.T) {
	// Create JWT service with very short TTL
	cfg := &config.JWTConfig{
		Secret:          "test-secret-key-for-testing-purpose",
		AccessTokenTTL:  -time.Hour, // Already expired
		RefreshTokenTTL: 24 * time.Hour,
		Issuer:          "test-issuer",
	}
	jwtService := auth.NewJWTService(cfg)

	userID := uuid.New()
	companyID := uuid.New()
	token, err := jwtService.GenerateAccessToken(userID, companyID, "test@test.com", "Test", []string{"user"})
	require.NoError(t, err)

	router := gin.New()
	router.Use(Auth(jwtService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_WrongSecret(t *testing.T) {
	// Create token with one secret
	cfg1 := &config.JWTConfig{
		Secret:          "secret-one",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		Issuer:          "test-issuer",
	}
	jwtService1 := auth.NewJWTService(cfg1)

	token, err := jwtService1.GenerateAccessToken(uuid.New(), uuid.New(), "test@test.com", "Test", []string{"user"})
	require.NoError(t, err)

	// Create router with different secret
	cfg2 := &config.JWTConfig{
		Secret:          "secret-two",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		Issuer:          "test-issuer",
	}
	jwtService2 := auth.NewJWTService(cfg2)

	router := gin.New()
	router.Use(Auth(jwtService2))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// =============================================================================
// OptionalAuth Middleware Tests
// =============================================================================

func TestOptionalAuth_NoToken(t *testing.T) {
	cfg := testJWTConfig()
	jwtService := auth.NewJWTService(cfg)

	router := gin.New()
	router.Use(OptionalAuth(jwtService))
	router.GET("/test", func(c *gin.Context) {
		userID := appctx.GetUserID(c)
		assert.Equal(t, uuid.Nil, userID) // Should be nil UUID
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOptionalAuth_ValidToken(t *testing.T) {
	cfg := testJWTConfig()
	jwtService := auth.NewJWTService(cfg)

	userID := uuid.New()
	companyID := uuid.New()
	token, err := jwtService.GenerateAccessToken(userID, companyID, "test@test.com", "Test", []string{"user"})
	require.NoError(t, err)

	router := gin.New()
	router.Use(OptionalAuth(jwtService))
	router.GET("/test", func(c *gin.Context) {
		resultUserID := appctx.GetUserID(c)
		assert.Equal(t, userID, resultUserID)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOptionalAuth_InvalidToken(t *testing.T) {
	cfg := testJWTConfig()
	jwtService := auth.NewJWTService(cfg)

	router := gin.New()
	router.Use(OptionalAuth(jwtService))
	router.GET("/test", func(c *gin.Context) {
		userID := appctx.GetUserID(c)
		assert.Equal(t, uuid.Nil, userID) // Should continue without auth
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code) // Should still succeed
}

// =============================================================================
// RequireRoles Middleware Tests
// =============================================================================

func TestRequireRoles_HasRequiredRole(t *testing.T) {
	router := gin.New()

	router.Use(func(c *gin.Context) {
		appctx.SetRoles(c, []string{"user", "admin"})
		c.Next()
	})
	router.Use(RequireRoles("admin"))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRoles_MissingRequiredRole(t *testing.T) {
	router := gin.New()

	router.Use(func(c *gin.Context) {
		appctx.SetRoles(c, []string{"user"}) // No admin role
		c.Next()
	})
	router.Use(RequireRoles("admin"))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireRoles_MultipleAllowedRoles(t *testing.T) {
	tests := []struct {
		name           string
		userRoles      []string
		requiredRoles  []string
		expectedStatus int
	}{
		{
			name:           "Has first required role",
			userRoles:      []string{"admin"},
			requiredRoles:  []string{"admin", "super_admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Has second required role",
			userRoles:      []string{"super_admin"},
			requiredRoles:  []string{"admin", "super_admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Has neither required role",
			userRoles:      []string{"user"},
			requiredRoles:  []string{"admin", "super_admin"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "No roles",
			userRoles:      []string{},
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()

			router.Use(func(c *gin.Context) {
				appctx.SetRoles(c, tc.userRoles)
				c.Next()
			})
			router.Use(RequireRoles(tc.requiredRoles...))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// =============================================================================
// RequireAdmin Middleware Tests
// =============================================================================

func TestRequireAdmin_IsAdmin(t *testing.T) {
	router := gin.New()

	router.Use(func(c *gin.Context) {
		appctx.SetRoles(c, []string{"user", "admin"})
		c.Next()
	})
	router.Use(RequireAdmin())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAdmin_NotAdmin(t *testing.T) {
	router := gin.New()

	router.Use(func(c *gin.Context) {
		appctx.SetRoles(c, []string{"user"})
		c.Next()
	})
	router.Use(RequireAdmin())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// =============================================================================
// Integration Tests - Full Auth Flow
// =============================================================================

func TestAuth_FullFlow_Authenticated(t *testing.T) {
	cfg := testJWTConfig()
	jwtService := auth.NewJWTService(cfg)

	userID := uuid.New()
	companyID := uuid.New()
	email := "admin@company.com"
	name := "Admin User"
	roles := []string{"admin"}

	token, err := jwtService.GenerateAccessToken(userID, companyID, email, name, roles)
	require.NoError(t, err)

	router := gin.New()
	router.Use(Auth(jwtService))
	router.Use(Tenant())
	router.Use(RequireAdmin())
	router.GET("/admin/test", func(c *gin.Context) {
		// Verify all context values are set correctly
		assert.Equal(t, userID, appctx.GetUserID(c))
		assert.Equal(t, companyID, appctx.GetCompanyID(c))
		assert.Equal(t, email, appctx.GetEmail(c))
		assert.Equal(t, name, appctx.GetUserName(c))
		assert.True(t, appctx.HasRole(c, "admin"))

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuth_FullFlow_NotAdmin(t *testing.T) {
	cfg := testJWTConfig()
	jwtService := auth.NewJWTService(cfg)

	userID := uuid.New()
	companyID := uuid.New()
	roles := []string{"user"} // Not admin

	token, err := jwtService.GenerateAccessToken(userID, companyID, "user@company.com", "Normal User", roles)
	require.NoError(t, err)

	router := gin.New()
	router.Use(Auth(jwtService))
	router.Use(Tenant())
	router.Use(RequireAdmin())
	router.GET("/admin/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
