package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/handler"
)

// newPayrollRouteEngine builds the payroll route tree on a bare engine.
//
// The handlers are constructed with a nil database on purpose: nothing here
// reaches a query. The tests below only exercise routing and the RBAC
// middleware, both of which run before any handler body.
func newPayrollRouteEngine(t *testing.T, roles []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	tenant := engine.Group("/api/v1")
	if roles != nil {
		tenant.Use(func(c *gin.Context) {
			appctx.SetRoles(c, roles)
			c.Next()
		})
	}
	RegisterPayrollRoutes(tenant, handler.NewPayrollHandlers(nil))
	return engine
}

// TestRegisterPayrollRoutesBuildsTree is the guard against a gin route
// conflict. /payrolls/preview is a static segment sitting next to the ":id"
// wildcard, which gin rejects with a panic at registration time if the tree
// cannot hold both - and a panic here takes the whole API down at startup, not
// just this module.
func TestRegisterPayrollRoutesBuildsTree(t *testing.T) {
	engine := newPayrollRouteEngine(t, nil)

	// The handler name gin records is the LAST handler in the chain, so this
	// also proves each path is wired to the function it claims - in particular
	// that POST /payrolls/preview reaches Preview and not the ":id" route.
	want := map[string]string{
		"GET /api/v1/payroll-periods":                "ListPeriods",
		"POST /api/v1/payroll-periods":               "CreatePeriod",
		"GET /api/v1/payroll-periods/:id":            "GetPeriod",
		"POST /api/v1/payroll-periods/:id/calculate": "CalculatePeriod",
		"POST /api/v1/payroll-periods/:id/approve":   "ApprovePeriod",
		"POST /api/v1/payroll-periods/:id/pay":       "PayPeriod",
		"POST /api/v1/payroll-periods/:id/close":     "ClosePeriod",

		"GET /api/v1/payrolls":                "List",
		"POST /api/v1/payrolls":               "Create",
		"POST /api/v1/payrolls/preview":       "Preview",
		"GET /api/v1/payrolls/:id":            "GetByID",
		"PUT /api/v1/payrolls/:id":            "Update",
		"DELETE /api/v1/payrolls/:id":         "Delete",
		"POST /api/v1/payrolls/:id/calculate": "Calculate",

		"GET /api/v1/insurance/workplaces":             "ListWorkplaces",
		"POST /api/v1/insurance/workplaces":            "CreateWorkplace",
		"GET /api/v1/insurance/workplaces/:id":         "GetWorkplace",
		"PUT /api/v1/insurance/workplaces/:id":         "UpdateWorkplace",
		"GET /api/v1/insurance/employees":              "ListEmployeeInsurance",
		"GET /api/v1/insurance/employees/:employee_id": "GetEmployeeInsurance",
		"PUT /api/v1/insurance/employees/:employee_id": "UpsertEmployeeInsurance",
		"GET /api/v1/insurance/contributions":          "ListContributions",
		"POST /api/v1/insurance/contributions/preview": "PreviewContribution",
		"GET /api/v1/insurance/summary":                "GetSummary",
		"GET /api/v1/insurance/reports":                "ListReports",
		"POST /api/v1/insurance/reports":               "CreateReport",
		"GET /api/v1/insurance/reports/:id":            "GetReport",
		"DELETE /api/v1/insurance/reports/:id":         "DeleteReport",
		"POST /api/v1/insurance/reports/:id/submit":    "SubmitReport",
		"POST /api/v1/insurance/reports/:id/cancel":    "CancelReport",
		"GET /api/v1/insurance/credentials":            "ListCredentialStatus",
		"GET /api/v1/insurance/edi-jobs":               "ListEDIJobs",
		"GET /api/v1/insurance/edi-jobs/:id":           "GetEDIJob",
	}

	got := make(map[string]string, len(want))
	for _, r := range engine.Routes() {
		got[r.Method+" "+r.Path] = lastHandlerName(r.Handler)
	}

	for key, handlerName := range want {
		actual, ok := got[key]
		if !ok {
			t.Errorf("route not registered: %s", key)
			continue
		}
		if actual != handlerName {
			t.Errorf("%s is served by %q, want %q", key, actual, handlerName)
		}
	}
}

// TestRegisterPayrollRoutesNilSafe checks the guards that let the router be
// built before the orchestrator has wired the handlers in.
func TestRegisterPayrollRoutesNilSafe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	tenant := engine.Group("/api/v1")

	RegisterPayrollRoutes(tenant, nil)
	if n := len(engine.Routes()); n != 0 {
		t.Errorf("a nil PayrollHandlers registered %d routes, want 0", n)
	}

	// A partially built struct must not panic either.
	RegisterPayrollRoutes(tenant, &handler.PayrollHandlers{})
	if n := len(engine.Routes()); n != 0 {
		t.Errorf("an empty PayrollHandlers registered %d routes, want 0", n)
	}
}

// TestPayrollRoutesRejectViewerWrites is the RBAC guard.
//
// It only asserts the DENIED direction: a rejected request never reaches a
// handler, so the nil database is never touched. Asserting the allowed
// direction would need a real database and would prove less - the risk here is
// a write endpoint that forgot its middleware, not a read one that has too
// much.
func TestPayrollRoutesRejectViewerWrites(t *testing.T) {
	engine := newPayrollRouteEngine(t, []string{"viewer"})

	writes := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/payroll-periods"},
		{http.MethodPost, "/api/v1/payroll-periods/" + sampleUUID + "/calculate"},
		{http.MethodPost, "/api/v1/payroll-periods/" + sampleUUID + "/approve"},
		{http.MethodPost, "/api/v1/payroll-periods/" + sampleUUID + "/pay"},
		{http.MethodPost, "/api/v1/payroll-periods/" + sampleUUID + "/close"},
		{http.MethodPost, "/api/v1/payrolls"},
		{http.MethodPost, "/api/v1/payrolls/preview"},
		{http.MethodPut, "/api/v1/payrolls/" + sampleUUID},
		{http.MethodDelete, "/api/v1/payrolls/" + sampleUUID},
		{http.MethodPost, "/api/v1/payrolls/" + sampleUUID + "/calculate"},
		{http.MethodPost, "/api/v1/insurance/workplaces"},
		{http.MethodPut, "/api/v1/insurance/workplaces/" + sampleUUID},
		{http.MethodPut, "/api/v1/insurance/employees/" + sampleUUID},
		{http.MethodPost, "/api/v1/insurance/contributions/preview"},
		{http.MethodPost, "/api/v1/insurance/reports"},
		{http.MethodDelete, "/api/v1/insurance/reports/" + sampleUUID},
		{http.MethodPost, "/api/v1/insurance/reports/" + sampleUUID + "/submit"},
		{http.MethodPost, "/api/v1/insurance/reports/" + sampleUUID + "/cancel"},
		// Credential status is admin-only.
		{http.MethodGet, "/api/v1/insurance/credentials"},
	}

	for _, w := range writes {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(w.method, w.path, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s answered %d for a viewer, want 403", w.method, w.path, rec.Code)
		}
	}
}

// TestPayrollApprovalsRejectPlainUser checks that the irreversible steps need
// an approver, not merely a writer.
//
// Confirming a payroll run is what the bank transfer file is built from, and a
// 4대보험 신고 leaves for a 공단; both are restricted the same way voucher
// posting and tax invoice transmission already are.
func TestPayrollApprovalsRejectPlainUser(t *testing.T) {
	engine := newPayrollRouteEngine(t, []string{"user"})

	approverOnly := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/payroll-periods/" + sampleUUID + "/approve"},
		{http.MethodPost, "/api/v1/payroll-periods/" + sampleUUID + "/pay"},
		{http.MethodPost, "/api/v1/payroll-periods/" + sampleUUID + "/close"},
		{http.MethodPost, "/api/v1/insurance/reports/" + sampleUUID + "/submit"},
		{http.MethodPost, "/api/v1/insurance/reports/" + sampleUUID + "/cancel"},
		{http.MethodGet, "/api/v1/insurance/credentials"},
	}

	for _, w := range approverOnly {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(w.method, w.path, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s answered %d for a plain user, want 403", w.method, w.path, rec.Code)
		}
	}
}

// sampleUUID is any well-formed UUID; these tests never get past the middleware
// so its value is irrelevant.
const sampleUUID = "0192f3a0-0000-7000-8000-000000000001"

// lastHandlerName reduces gin's fully qualified handler name to the method
// name, e.g. ".../handler.(*PayrollHandler).Preview-fm" -> "Preview".
func lastHandlerName(qualified string) string {
	name := qualified
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	return strings.TrimSuffix(name, "-fm")
}
