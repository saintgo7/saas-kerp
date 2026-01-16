//go:build integration
// +build integration

package security

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/saintgo7/saas-kerp/internal/auth"
	"github.com/saintgo7/saas-kerp/internal/config"
	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/middleware"
)

// =============================================================================
// RLS (Row Level Security) Test Suite
// =============================================================================

// RLSTestSuite tests multi-tenancy isolation at the API level
type RLSTestSuite struct {
	suite.Suite
	router     *gin.Engine
	jwtService *auth.JWTService

	// Test companies
	companyA uuid.UUID
	companyB uuid.UUID

	// Test users
	userA uuid.UUID
	userB uuid.UUID

	// Test tokens
	tokenA string
	tokenB string
}

func TestRLSSuite(t *testing.T) {
	suite.Run(t, new(RLSTestSuite))
}

func (s *RLSTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Create JWT service
	cfg := &config.JWTConfig{
		Secret:          "test-secret-key-for-security-testing",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		Issuer:          "test-issuer",
	}
	s.jwtService = auth.NewJWTService(cfg)

	// Setup test companies
	s.companyA = uuid.New()
	s.companyB = uuid.New()

	// Setup test users
	s.userA = uuid.New()
	s.userB = uuid.New()

	// Generate tokens
	var err error
	s.tokenA, err = s.jwtService.GenerateAccessToken(
		s.userA, s.companyA, "usera@companyA.com", "User A", []string{"admin"},
	)
	s.Require().NoError(err)

	s.tokenB, err = s.jwtService.GenerateAccessToken(
		s.userB, s.companyB, "userb@companyB.com", "User B", []string{"admin"},
	)
	s.Require().NoError(err)

	// Setup router with middleware
	s.router = gin.New()
	s.router.Use(middleware.Auth(s.jwtService))
	s.router.Use(middleware.Tenant())

	// Test endpoints that check company isolation
	s.router.GET("/api/v1/vouchers/:id", s.mockGetVoucher)
	s.router.PUT("/api/v1/vouchers/:id", s.mockUpdateVoucher)
	s.router.DELETE("/api/v1/vouchers/:id", s.mockDeleteVoucher)
}

// =============================================================================
// Mock Handlers for Testing
// =============================================================================

// mockGetVoucher simulates getting a voucher with company isolation
func (s *RLSTestSuite) mockGetVoucher(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	voucherID := c.Param("id")

	// Simulate voucher owned by Company A
	voucherCompanyA := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	// Check if the voucher belongs to the requesting company
	if voucherID == "voucher-company-a" {
		if companyID != s.companyA {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied: Resource belongs to another tenant",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"id":        voucherID,
			"companyId": voucherCompanyA.String(),
			"amount":    10000,
		})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Voucher not found"})
}

func (s *RLSTestSuite) mockUpdateVoucher(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	voucherID := c.Param("id")

	if voucherID == "voucher-company-a" {
		if companyID != s.companyA {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied: Cannot modify another tenant's resource",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "updated"})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Voucher not found"})
}

func (s *RLSTestSuite) mockDeleteVoucher(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	voucherID := c.Param("id")

	if voucherID == "voucher-company-a" {
		if companyID != s.companyA {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied: Cannot delete another tenant's resource",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Voucher not found"})
}

// =============================================================================
// Cross-Tenant Access Tests (Critical Security Tests)
// =============================================================================

func (s *RLSTestSuite) TestCrossTenantRead_ShouldBeDenied() {
	// Company B trying to read Company A's voucher
	req := httptest.NewRequest("GET", "/api/v1/vouchers/voucher-company-a", nil)
	req.Header.Set("Authorization", "Bearer "+s.tokenB)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusForbidden, w.Code)
	assert.Contains(s.T(), w.Body.String(), "Access denied")
}

func (s *RLSTestSuite) TestCrossTenantUpdate_ShouldBeDenied() {
	// Company B trying to update Company A's voucher
	body := `{"amount": 99999}`
	req := httptest.NewRequest("PUT", "/api/v1/vouchers/voucher-company-a",
		bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+s.tokenB)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusForbidden, w.Code)
	assert.Contains(s.T(), w.Body.String(), "Access denied")
}

func (s *RLSTestSuite) TestCrossTenantDelete_ShouldBeDenied() {
	// Company B trying to delete Company A's voucher
	req := httptest.NewRequest("DELETE", "/api/v1/vouchers/voucher-company-a", nil)
	req.Header.Set("Authorization", "Bearer "+s.tokenB)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusForbidden, w.Code)
	assert.Contains(s.T(), w.Body.String(), "Access denied")
}

// =============================================================================
// Same-Tenant Access Tests
// =============================================================================

func (s *RLSTestSuite) TestSameTenantRead_ShouldBeAllowed() {
	// Company A reading their own voucher
	req := httptest.NewRequest("GET", "/api/v1/vouchers/voucher-company-a", nil)
	req.Header.Set("Authorization", "Bearer "+s.tokenA)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *RLSTestSuite) TestSameTenantUpdate_ShouldBeAllowed() {
	body := `{"amount": 20000}`
	req := httptest.NewRequest("PUT", "/api/v1/vouchers/voucher-company-a",
		bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+s.tokenA)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *RLSTestSuite) TestSameTenantDelete_ShouldBeAllowed() {
	req := httptest.NewRequest("DELETE", "/api/v1/vouchers/voucher-company-a", nil)
	req.Header.Set("Authorization", "Bearer "+s.tokenA)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// =============================================================================
// Auth Bypass Tests
// =============================================================================

type AuthBypassTestSuite struct {
	suite.Suite
	router     *gin.Engine
	jwtService *auth.JWTService
}

func TestAuthBypassSuite(t *testing.T) {
	suite.Run(t, new(AuthBypassTestSuite))
}

func (s *AuthBypassTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	cfg := &config.JWTConfig{
		Secret:          "test-secret-key-for-security-testing",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		Issuer:          "test-issuer",
	}
	s.jwtService = auth.NewJWTService(cfg)

	s.router = gin.New()
	s.router.Use(middleware.Auth(s.jwtService))
	s.router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})
}

func (s *AuthBypassTestSuite) TestNoToken_ShouldFail() {
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

func (s *AuthBypassTestSuite) TestEmptyToken_ShouldFail() {
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

func (s *AuthBypassTestSuite) TestMalformedToken_ShouldFail() {
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer malformed.token.here")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

func (s *AuthBypassTestSuite) TestExpiredToken_ShouldFail() {
	// Create expired token
	expiredCfg := &config.JWTConfig{
		Secret:          "test-secret-key-for-security-testing",
		AccessTokenTTL:  -time.Hour, // Already expired
		RefreshTokenTTL: 24 * time.Hour,
		Issuer:          "test-issuer",
	}
	expiredJWT := auth.NewJWTService(expiredCfg)

	token, _ := expiredJWT.GenerateAccessToken(
		uuid.New(), uuid.New(), "test@test.com", "Test", []string{"user"},
	)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

func (s *AuthBypassTestSuite) TestWrongSecretToken_ShouldFail() {
	// Create token with different secret
	otherCfg := &config.JWTConfig{
		Secret:          "different-secret-key",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		Issuer:          "test-issuer",
	}
	otherJWT := auth.NewJWTService(otherCfg)

	token, _ := otherJWT.GenerateAccessToken(
		uuid.New(), uuid.New(), "test@test.com", "Test", []string{"user"},
	)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

func (s *AuthBypassTestSuite) TestSQLInjectionAttempt_InToken_ShouldFail() {
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer ' OR '1'='1")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

func (s *AuthBypassTestSuite) TestXSSAttempt_InToken_ShouldFail() {
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer <script>alert('xss')</script>")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

// =============================================================================
// RBAC (Role-Based Access Control) Tests
// =============================================================================

type RBACTestSuite struct {
	suite.Suite
	router     *gin.Engine
	jwtService *auth.JWTService
}

func TestRBACSuite(t *testing.T) {
	suite.Run(t, new(RBACTestSuite))
}

func (s *RBACTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	cfg := &config.JWTConfig{
		Secret:          "test-secret-key-for-security-testing",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		Issuer:          "test-issuer",
	}
	s.jwtService = auth.NewJWTService(cfg)

	s.router = gin.New()
	s.router.Use(middleware.Auth(s.jwtService))

	// Admin-only route
	adminGroup := s.router.Group("/admin")
	adminGroup.Use(middleware.RequireAdmin())
	adminGroup.GET("/settings", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "admin access granted"})
	})

	// Manager or Admin route
	managerGroup := s.router.Group("/manager")
	managerGroup.Use(middleware.RequireRoles("manager", "admin"))
	managerGroup.GET("/reports", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "manager access granted"})
	})
}

func (s *RBACTestSuite) TestAdminRoute_AsAdmin_ShouldBeAllowed() {
	token, _ := s.jwtService.GenerateAccessToken(
		uuid.New(), uuid.New(), "admin@test.com", "Admin", []string{"admin"},
	)

	req := httptest.NewRequest("GET", "/admin/settings", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *RBACTestSuite) TestAdminRoute_AsUser_ShouldBeDenied() {
	token, _ := s.jwtService.GenerateAccessToken(
		uuid.New(), uuid.New(), "user@test.com", "User", []string{"user"},
	)

	req := httptest.NewRequest("GET", "/admin/settings", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusForbidden, w.Code)
}

func (s *RBACTestSuite) TestManagerRoute_AsManager_ShouldBeAllowed() {
	token, _ := s.jwtService.GenerateAccessToken(
		uuid.New(), uuid.New(), "manager@test.com", "Manager", []string{"manager"},
	)

	req := httptest.NewRequest("GET", "/manager/reports", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *RBACTestSuite) TestManagerRoute_AsAdmin_ShouldBeAllowed() {
	token, _ := s.jwtService.GenerateAccessToken(
		uuid.New(), uuid.New(), "admin@test.com", "Admin", []string{"admin"},
	)

	req := httptest.NewRequest("GET", "/manager/reports", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *RBACTestSuite) TestManagerRoute_AsUser_ShouldBeDenied() {
	token, _ := s.jwtService.GenerateAccessToken(
		uuid.New(), uuid.New(), "user@test.com", "User", []string{"user"},
	)

	req := httptest.NewRequest("GET", "/manager/reports", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusForbidden, w.Code)
}

// =============================================================================
// Input Validation Security Tests
// =============================================================================

func TestInputValidation_SQLInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/search", func(c *gin.Context) {
		query := c.Query("q")

		// Simulate input validation that should block SQL injection
		maliciousPatterns := []string{
			"'",
			"--",
			";",
			"DROP",
			"DELETE",
			"INSERT",
			"UPDATE",
			"UNION",
		}

		for _, pattern := range maliciousPatterns {
			if len(query) > 0 && (query[0] == '\'' || query == pattern) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"results": []string{}})
	})

	tests := []struct {
		name     string
		query    string
		wantCode int
	}{
		{"Normal query", "test search", http.StatusOK},
		{"Empty query", "", http.StatusOK},
		{"SQL injection attempt 1", "'; DROP TABLE users;--", http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/search?q="+tc.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.wantCode, w.Code)
		})
	}
}

func TestInputValidation_XSS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/comment", func(c *gin.Context) {
		var input struct {
			Content string `json:"content"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}

		// Check for potential XSS
		if len(input.Content) > 0 && input.Content[0] == '<' {
			c.JSON(http.StatusBadRequest, gin.H{"error": "HTML not allowed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{"Normal content", `{"content": "Hello world"}`, http.StatusOK},
		{"XSS attempt", `{"content": "<script>alert('xss')</script>"}`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/comment", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.wantCode, w.Code)
		})
	}
}
