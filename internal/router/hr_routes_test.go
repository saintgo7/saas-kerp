package router

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/handler"
)

// newHRTestEngine builds an engine carrying only the HR routes, behind a stub
// that puts a company and a set of roles in the context the way
// middleware.Auth and middleware.Tenant do on the real path.
//
// The handlers are built with a nil database on purpose: nothing these tests
// reach issues a query. A request that got past the permission middleware
// would fail loudly instead of quietly passing.
func newHRTestEngine(t *testing.T, roles ...string) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()

	api := engine.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		appctx.SetUserID(c, uuid.New())
		appctx.SetCompanyID(c, uuid.New())
		appctx.SetRoles(c, roles)
		c.Next()
	})

	RegisterHRRoutes(api, handler.NewHRHandlers(nil, zap.NewNop()))
	return engine
}

// TestHRRoutesAreRegistered pins the endpoint list. Registering them is also
// the only way to find out that gin accepts the tree: a static segment next to
// a parameter at the same level (/employees/stats beside /employees/:id) is a
// panic in some routers.
func TestHRRoutesAreRegistered(t *testing.T) {
	engine := newHRTestEngine(t, "admin")

	got := map[string]bool{}
	for _, route := range engine.Routes() {
		got[route.Method+" "+route.Path] = true
	}

	want := []string{
		"GET /api/v1/departments",
		"GET /api/v1/departments/tree",
		"GET /api/v1/departments/code/:code",
		"GET /api/v1/departments/:id",
		"GET /api/v1/departments/:id/children",
		"GET /api/v1/departments/:id/can-delete",
		"POST /api/v1/departments",
		"PUT /api/v1/departments/:id",
		"POST /api/v1/departments/:id/move",
		"DELETE /api/v1/departments/:id",

		"GET /api/v1/positions",
		"GET /api/v1/positions/code/:code",
		"GET /api/v1/positions/:id",
		"GET /api/v1/positions/:id/can-delete",
		"POST /api/v1/positions",
		"PUT /api/v1/positions/:id",
		"DELETE /api/v1/positions/:id",

		"GET /api/v1/employees",
		"GET /api/v1/employees/stats",
		"GET /api/v1/employees/no/:employee_no",
		"GET /api/v1/employees/:id",
		"GET /api/v1/employees/:id/can-delete",
		"POST /api/v1/employees",
		"PUT /api/v1/employees/:id",
		"POST /api/v1/employees/:id/status",
		"DELETE /api/v1/employees/:id",

		"GET /api/v1/leave-types",
		"GET /api/v1/leave-types/:id",
		"POST /api/v1/leave-types",
		"PUT /api/v1/leave-types/:id",
		"DELETE /api/v1/leave-types/:id",

		"GET /api/v1/leaves",
		"GET /api/v1/leaves/:id",
		"POST /api/v1/leaves",
		"PUT /api/v1/leaves/:id",
		"DELETE /api/v1/leaves/:id",
		"POST /api/v1/leaves/:id/approve",
		"POST /api/v1/leaves/:id/reject",
		"POST /api/v1/leaves/:id/cancel",

		"GET /api/v1/leave-balances",
		"PUT /api/v1/leave-balances",
	}

	for _, route := range want {
		if !got[route] {
			t.Errorf("%s is not registered", route)
		}
	}

	// Nothing unexpected either: a stray route is a permission surface nobody
	// reviewed.
	expected := map[string]bool{}
	for _, route := range want {
		expected[route] = true
	}
	var extra []string
	for route := range got {
		if !expected[route] {
			extra = append(extra, route)
		}
	}
	sort.Strings(extra)
	if len(extra) > 0 {
		t.Errorf("unexpected HR routes: %s", strings.Join(extra, ", "))
	}
}

// TestRegisterHRRoutesToleratesUnwiredHandlers so that a partially wired
// handler.Handlers cannot take the whole API down at start-up.
func TestRegisterHRRoutesToleratesUnwiredHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/api/v1")

	RegisterHRRoutes(api, nil)
	RegisterHRRoutes(api, &handler.HRHandlers{})

	if len(engine.Routes()) != 0 {
		t.Errorf("unwired handlers registered %d routes", len(engine.Routes()))
	}
}

// TestViewersCannotWriteHRRecords is the read-only guarantee. A viewer that
// reaches any of these handlers would be creating, editing or deleting staff
// records, so the assertion is that the request is refused before it gets
// there.
func TestViewersCannotWriteHRRecords(t *testing.T) {
	engine := newHRTestEngine(t, "viewer")

	writes := []struct{ method, path string }{
		{"POST", "/api/v1/departments"},
		{"PUT", "/api/v1/departments/" + uuid.New().String()},
		{"POST", "/api/v1/departments/" + uuid.New().String() + "/move"},
		{"DELETE", "/api/v1/departments/" + uuid.New().String()},

		{"POST", "/api/v1/positions"},
		{"PUT", "/api/v1/positions/" + uuid.New().String()},
		{"DELETE", "/api/v1/positions/" + uuid.New().String()},

		{"POST", "/api/v1/employees"},
		{"PUT", "/api/v1/employees/" + uuid.New().String()},
		{"POST", "/api/v1/employees/" + uuid.New().String() + "/status"},
		{"DELETE", "/api/v1/employees/" + uuid.New().String()},

		{"POST", "/api/v1/leave-types"},
		{"PUT", "/api/v1/leave-types/" + uuid.New().String()},
		{"DELETE", "/api/v1/leave-types/" + uuid.New().String()},

		{"POST", "/api/v1/leaves"},
		{"PUT", "/api/v1/leaves/" + uuid.New().String()},
		{"DELETE", "/api/v1/leaves/" + uuid.New().String()},
		{"POST", "/api/v1/leaves/" + uuid.New().String() + "/approve"},
		{"POST", "/api/v1/leaves/" + uuid.New().String() + "/reject"},
		{"POST", "/api/v1/leaves/" + uuid.New().String() + "/cancel"},

		{"PUT", "/api/v1/leave-balances"},
	}

	for _, write := range writes {
		t.Run(write.method+" "+write.path, func(t *testing.T) {
			req := httptest.NewRequest(write.method, write.path, strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Errorf("a viewer got %d, want 403", w.Code)
			}
		})
	}
}

// TestApprovalsAndAdminOnlyRoutesRejectAPlainUser.
//
// A "user" may file and edit leave but not approve it, may edit an employee
// but not delete one, and may not grant the leave entitlement everyone else
// spends.
func TestApprovalsAndAdminOnlyRoutesRejectAPlainUser(t *testing.T) {
	engine := newHRTestEngine(t, "user")

	restricted := []struct{ method, path string }{
		{"DELETE", "/api/v1/departments/" + uuid.New().String()},
		{"POST", "/api/v1/leaves/" + uuid.New().String() + "/approve"},
		{"POST", "/api/v1/leaves/" + uuid.New().String() + "/reject"},
		{"DELETE", "/api/v1/employees/" + uuid.New().String()},
		{"DELETE", "/api/v1/leave-types/" + uuid.New().String()},
		{"PUT", "/api/v1/leave-balances"},
	}

	for _, route := range restricted {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Errorf("a plain user got %d, want 403", w.Code)
			}
		})
	}
}
