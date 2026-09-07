package router

import (
	"github.com/gin-gonic/gin"

	"github.com/saintgo7/saas-kerp/internal/handler"
	"github.com/saintgo7/saas-kerp/internal/middleware"
)

// RegisterPayrollRoutes registers the payroll and 4대보험 routes.
//
// It must be called on the TENANT group (authentication + tenant context + the
// PostgreSQL session variable the RLS policies read): every handler reads the
// company id out of the request context and every repository query is scoped
// by it.
//
// The permission rules, in one place so they can be read as a whole:
//
//   - reading requires authentication only, matching every other module in
//     this codebase (vouchers, tax invoices, HR);
//   - READING is RequireWriter (admin or user), not merely authenticated.
//     Salary figures and 4대보험 records are personnel data; a viewer role
//     has no business enumerating what every employee is paid. It also keeps
//     read and write at the same level - whoever may edit payroll may read it.
//   - creating, editing and CALCULATING requires RequireWriter (admin or user),
//     because a calculation is a draft figure that recalculating reverses;
//   - confirming a payroll run, marking it paid, closing it, and submitting or
//     cancelling a 4대보험 신고 require RequireApprover (admin). Those are the
//     irreversible steps: an approved run is what the bank transfer file and
//     the 원천징수이행상황신고서 are built from, and a 신고 leaves for a 공단
//     and cannot be taken back;
//   - the credential status listing requires RequireAdmin. It exposes no
//     secret - the encrypted columns are not even mapped by the domain model -
//     but which agency portals a company holds logins for is itself
//     administrative information.
//
// NOTE for whoever reviews the sensitivity of this: payroll reads are open to
// the "viewer" role, because that is the convention every other module follows
// and the RBAC vocabulary here has only admin/user/viewer. If salary figures
// should be narrower than "any authenticated user of the tenant", that needs a
// dedicated role rather than a one-off exception here.
func RegisterPayrollRoutes(tenant *gin.RouterGroup, h *handler.PayrollHandlers) {
	if h == nil {
		return
	}

	registerPayrollPeriodRoutes(tenant, h.Payroll)
	registerPayrollRecordRoutes(tenant, h.Payroll)
	registerInsuranceRoutes(tenant, h.Insurance)
}

// registerPayrollPeriodRoutes registers /payroll-periods.
func registerPayrollPeriodRoutes(tenant *gin.RouterGroup, h *handler.PayrollHandler) {
	if h == nil {
		return
	}

	periods := tenant.Group("/payroll-periods")
	{
		periods.GET("", middleware.RequireWriter(), h.ListPeriods)
		periods.GET("/:id", middleware.RequireWriter(), h.GetPeriod)

		periods.POST("", middleware.RequireWriter(), h.CreatePeriod)

		// Calculating is reversible: it recomputes deductions from the stored
		// earnings and can be run again.
		periods.POST("/:id/calculate", middleware.RequireWriter(), h.CalculatePeriod)

		// Confirming, paying out and closing are not. Approving is what the
		// bank transfer and the 원천징수이행상황신고서 are built from, and the
		// service refuses to confirm a run whose headers disagree with their
		// line items.
		periods.POST("/:id/approve", middleware.RequireApprover(), h.ApprovePeriod)
		periods.POST("/:id/pay", middleware.RequireApprover(), h.PayPeriod)
		periods.POST("/:id/close", middleware.RequireApprover(), h.ClosePeriod)
	}
}

// registerPayrollRecordRoutes registers /payrolls.
func registerPayrollRecordRoutes(tenant *gin.RouterGroup, h *handler.PayrollHandler) {
	if h == nil {
		return
	}

	payrolls := tenant.Group("/payrolls")
	{
		payrolls.GET("", middleware.RequireWriter(), h.List)

		// The static segment is registered before the ":id" wildcard, the same
		// way the voucher routes do it, so /payrolls/preview is matched by its
		// own route instead of being parsed as a payroll id and answering
		// "invalid payroll ID".
		//
		// Preview computes a payslip and writes nothing at all - no payroll, no
		// item rows, no 4대보험 register entry.
		payrolls.POST("/preview", middleware.RequireWriter(), h.Preview)

		payrolls.GET("/:id", middleware.RequireWriter(), h.GetByID)

		payrolls.POST("", middleware.RequireWriter(), h.Create)
		payrolls.PUT("/:id", middleware.RequireWriter(), h.Update)
		payrolls.DELETE("/:id", middleware.RequireWriter(), h.Delete)
		payrolls.POST("/:id/calculate", middleware.RequireWriter(), h.Calculate)
	}
}

// registerInsuranceRoutes registers /insurance.
func registerInsuranceRoutes(tenant *gin.RouterGroup, h *handler.InsuranceHandler) {
	if h == nil {
		return
	}

	insurance := tenant.Group("/insurance")
	{
		// 사업장 (workplace registrations)
		insurance.GET("/workplaces", middleware.RequireWriter(), h.ListWorkplaces)
		insurance.GET("/workplaces/:id", middleware.RequireWriter(), h.GetWorkplace)
		insurance.POST("/workplaces", middleware.RequireWriter(), h.CreateWorkplace)
		insurance.PUT("/workplaces/:id", middleware.RequireWriter(), h.UpdateWorkplace)

		// 자격 (employee qualifications)
		insurance.GET("/employees", middleware.RequireWriter(), h.ListEmployeeInsurance)
		insurance.GET("/employees/:employee_id", middleware.RequireWriter(), h.GetEmployeeInsurance)
		insurance.PUT("/employees/:employee_id", middleware.RequireWriter(), h.UpsertEmployeeInsurance)

		// 보험료 (assessed contributions).
		//
		// There is no endpoint that writes a contribution. The register is
		// produced by the payroll calculation, inside the payroll transaction,
		// so the two can never report different figures for the same month.
		insurance.GET("/contributions", middleware.RequireWriter(), h.ListContributions)
		insurance.POST("/contributions/preview", middleware.RequireWriter(), h.PreviewContribution)
		insurance.GET("/summary", middleware.RequireWriter(), h.GetSummary)

		// 신고서 (reports to the agencies)
		insurance.GET("/reports", middleware.RequireWriter(), h.ListReports)
		insurance.GET("/reports/:id", middleware.RequireWriter(), h.GetReport)
		insurance.POST("/reports", middleware.RequireWriter(), h.CreateReport)
		insurance.DELETE("/reports/:id", middleware.RequireWriter(), h.DeleteReport)

		// Submitting leaves for a 공단 and cannot be taken back, so it needs an
		// approver - the same rule tax invoice transmission follows. It
		// currently answers 501: the EDI transport is not wired and the
		// service changes nothing rather than reporting a filing that never
		// happened.
		insurance.POST("/reports/:id/submit", middleware.RequireApprover(), h.SubmitReport)
		insurance.POST("/reports/:id/cancel", middleware.RequireApprover(), h.CancelReport)

		// 공단 접속 자격증명 - status only, never the credentials themselves.
		insurance.GET("/credentials", middleware.RequireAdmin(), h.ListCredentialStatus)

		// EDI 작업 이력
		insurance.GET("/edi-jobs", middleware.RequireWriter(), h.ListEDIJobs)
		insurance.GET("/edi-jobs/:id", middleware.RequireWriter(), h.GetEDIJob)
	}
}
