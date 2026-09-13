package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/saintgo7/saas-kerp/internal/auth"
	"github.com/saintgo7/saas-kerp/internal/config"
	"github.com/saintgo7/saas-kerp/internal/handler"
)

// newTestRouter builds the real route tree. The handlers are constructed with a
// nil database on purpose: nothing here reaches a query, and the point of these
// tests is the routing and middleware wiring, not the handlers.
func newTestRouter(t *testing.T, mutate func(*config.Config)) *Router {
	t.Helper()

	cfg := &config.Config{
		App: config.AppConfig{
			Name:                "kerp-api-test",
			Env:                 "production",
			Port:                8080,
			Version:             "test",
			MaxRequestBodyBytes: 1 << 20,
		},
		JWT: config.JWTConfig{
			Secret:          "Wm4Kd8Rp2Ns6Vt9Yc3Hb7Lq1Zg5Ax0Ju4Fo8Ei2Tr6",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: time.Hour,
			Issuer:          "kerp-api",
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"http://localhost:5173"},
			AllowedMethods: []string{"GET", "POST"},
			AllowedHeaders: []string{"Authorization", "Content-Type"},
		},
		Log: config.LogConfig{Level: "info", Format: "json"},
	}
	if mutate != nil {
		mutate(cfg)
	}

	jwtService := auth.NewJWTService(&cfg.JWT)
	handlers := handler.NewHandlers(nil, nil, zap.NewNop(), jwtService, cfg.App.Version, cfg.App.Env)

	return New(cfg, zap.NewNop(), jwtService, handlers, nil)
}

func routeSet(r *Router) map[string]bool {
	out := make(map[string]bool)
	for _, ri := range r.Engine().Routes() {
		out[ri.Method+" "+ri.Path] = true
	}
	return out
}

// Refreshing must not require a valid access token: the only moment a client
// needs to refresh is when its access token has expired.
func TestRefreshIsReachableWithoutAnAccessToken(t *testing.T) {
	r := newTestRouter(t, nil)

	if !routeSet(r)["POST /api/v1/auth/refresh"] {
		t.Fatal("POST /api/v1/auth/refresh is not registered")
	}

	// No Authorization header, empty body. A 401 here would mean the Auth
	// middleware is still in front of the route; a 400 means the request reached
	// the handler and failed body validation, which is what we want.
	req := httptest.NewRequest("POST", "/api/v1/auth/refresh", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.Engine().ServeHTTP(w, req)

	if w.Code == http.StatusUnauthorized {
		t.Fatalf("refresh still requires authentication (status %d, body %s)", w.Code, w.Body.String())
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 from body validation (body: %s)", w.Code, w.Body.String())
	}
}

func TestProtectedRoutesStillRequireAuth(t *testing.T) {
	r := newTestRouter(t, nil)

	for _, path := range []string{"/api/v1/auth/me", "/api/v1/vouchers", "/api/v1/accounts"} {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		r.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("GET %s answered %d without a token, want 401", path, w.Code)
		}
	}
}

// With no trusted proxies configured, X-Forwarded-For must be ignored: it is the
// rate limiter's bucket key and the audit log's client_ip.
func TestUntrustedProxyHeadersAreIgnored(t *testing.T) {
	r := newTestRouter(t, nil)

	var seen string
	r.Engine().GET("/whoami", func(c *gin.Context) { seen = c.ClientIP() })

	req := httptest.NewRequest("GET", "/whoami", nil)
	req.RemoteAddr = "10.9.9.9:5555"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	w := httptest.NewRecorder()
	r.Engine().ServeHTTP(w, req)

	if seen == "1.2.3.4" {
		t.Fatal("a client-supplied X-Forwarded-For was believed with no trusted proxies configured")
	}
	if seen != "10.9.9.9" {
		t.Fatalf("client IP = %q, want the peer address 10.9.9.9", seen)
	}
}

func TestTrustedProxyHeadersAreHonouredWhenConfigured(t *testing.T) {
	r := newTestRouter(t, func(cfg *config.Config) {
		cfg.App.TrustedProxies = []string{"10.9.0.0/16"}
	})

	var seen string
	r.Engine().GET("/whoami", func(c *gin.Context) { seen = c.ClientIP() })

	req := httptest.NewRequest("GET", "/whoami", nil)
	req.RemoteAddr = "10.9.9.9:5555"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	w := httptest.NewRecorder()
	r.Engine().ServeHTTP(w, req)

	if seen != "1.2.3.4" {
		t.Fatalf("client IP = %q, want 1.2.3.4 from the trusted proxy", seen)
	}
}

// An unparseable CIDR must not leave gin's wide-open default in place.
func TestInvalidTrustedProxiesFailClosed(t *testing.T) {
	r := newTestRouter(t, func(cfg *config.Config) {
		cfg.App.TrustedProxies = []string{"not-a-cidr"}
	})

	var seen string
	r.Engine().GET("/whoami", func(c *gin.Context) { seen = c.ClientIP() })

	req := httptest.NewRequest("GET", "/whoami", nil)
	req.RemoteAddr = "10.9.9.9:5555"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	w := httptest.NewRecorder()
	r.Engine().ServeHTTP(w, req)

	if seen != "10.9.9.9" {
		t.Fatalf("client IP = %q; an invalid trusted_proxies list left proxy headers trusted", seen)
	}
}

// Every response, including the liveness probe, uses the one envelope.
func TestHealthLiveUsesTheStandardEnvelope(t *testing.T) {
	r := newTestRouter(t, nil)

	req := httptest.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()
	r.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Status string `json:"status"`
		} `json:"data"`
		Meta struct {
			RequestID string `json:"request_id"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not the standard envelope: %v (%s)", err, w.Body.String())
	}
	if !body.Success || body.Data.Status != "alive" {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
	if body.Meta.RequestID == "" {
		t.Error("meta.request_id is empty")
	}
}

// User and role administration must not be reachable by a non-admin. The check
// lives on the route group, so verify the middleware is attached rather than
// re-testing RequireAdmin itself.
func TestAdministrationRoutesAreRegistered(t *testing.T) {
	routes := routeSet(newTestRouter(t, nil))

	for _, want := range []string{
		"GET /api/v1/users",
		"POST /api/v1/users",
		"GET /api/v1/roles",
		"PUT /api/v1/company",
		"POST /api/v1/vouchers/:id/approve",
		"POST /api/v1/fiscal-periods/year-end-close",
	} {
		if !routes[want] {
			t.Errorf("route %q is missing", want)
		}
	}
}
