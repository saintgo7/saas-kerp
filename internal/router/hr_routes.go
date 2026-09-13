package router

import (
	"github.com/gin-gonic/gin"

	"github.com/saintgo7/saas-kerp/internal/handler"
	"github.com/saintgo7/saas-kerp/internal/middleware"
)

// RegisterHRRoutes registers the human-resources routes.
//
// It must be called on the TENANT group (authentication + tenant context +
// the PostgreSQL session variable the RLS policies read): every handler below
// reads the company id out of the request context and every repository query
// is scoped by it.
//
// The permission rules, in one place so they can be read as a whole:
//
//   - reading requires authentication only, so a "viewer" account can see the
//     organisation chart, the leave calendar and the balances;
//   - creating and editing requires RequireWriter (admin or user);
//   - approving or rejecting leave requires RequireApprover (admin), because
//     an approval moves a balance and is what payroll later reads;
//   - deleting an employee, deleting a department, deleting a leave type and
//     granting a leave entitlement require RequireAdmin: the first three
//     destroy records that other tables reference, and the last creates the
//     days everyone else spends.
//
// Nothing here exposes a resident registration number. The column is
// ciphertext, the domain field is json:"-", and dto.EmployeeResponse reports
// only whether one is on file.
func RegisterHRRoutes(tenant *gin.RouterGroup, hr *handler.HRHandlers) {
	if hr == nil {
		return
	}

	registerDepartmentRoutes(tenant, hr.Department)
	registerPositionRoutes(tenant, hr.Position)
	registerEmployeeRoutes(tenant, hr.Employee)
	registerLeaveRoutes(tenant, hr.Leave)
}

// registerDepartmentRoutes registers /departments.
//
// Deleting a department is RequireAdmin, the same bar as deleting an employee:
// it removes a node other tables point at (employees.department_id,
// voucher_entries.department_id), and the service refuses outright while
// anything still references it.
func registerDepartmentRoutes(tenant *gin.RouterGroup, h *handler.DepartmentHandler) {
	if h == nil {
		return
	}

	departments := tenant.Group("/departments")
	{
		departments.GET("", h.List)

		// The nested organisation chart. It is a separate endpoint rather than
		// a flag on the list, because it is a different shape - no pagination,
		// children inline - and because a static segment beside ":id" must be
		// declared for gin to route /departments/tree to it instead of parsing
		// "tree" as an identifier.
		departments.GET("/tree", h.Tree)

		departments.GET("/code/:code", h.GetByCode)
		departments.GET("/:id", h.GetByID)
		departments.GET("/:id/children", h.GetChildren)
		departments.GET("/:id/can-delete", h.CanDelete)

		departments.POST("", middleware.RequireWriter(), h.Create)
		departments.PUT("/:id", middleware.RequireWriter(), h.Update)

		// Re-parenting on its own, for a chart edit that moves a subtree
		// without resending the department's other fields. The circular
		// reference rules are the service's and apply to both paths.
		departments.POST("/:id/move", middleware.RequireWriter(), h.Move)

		departments.DELETE("/:id", middleware.RequireAdmin(), h.Delete)
	}
}

// registerPositionRoutes registers /positions.
func registerPositionRoutes(tenant *gin.RouterGroup, h *handler.PositionHandler) {
	if h == nil {
		return
	}

	positions := tenant.Group("/positions")
	{
		positions.GET("", h.List)
		positions.GET("/code/:code", h.GetByCode)
		positions.GET("/:id", h.GetByID)
		positions.GET("/:id/can-delete", h.CanDelete)

		positions.POST("", middleware.RequireWriter(), h.Create)
		positions.PUT("/:id", middleware.RequireWriter(), h.Update)
		positions.DELETE("/:id", middleware.RequireWriter(), h.Delete)
	}
}

// registerEmployeeRoutes registers /employees.
func registerEmployeeRoutes(tenant *gin.RouterGroup, h *handler.EmployeeHandler) {
	if h == nil {
		return
	}

	employees := tenant.Group("/employees")
	{
		employees.GET("", h.List)
		employees.GET("/stats", h.GetStats)
		employees.GET("/no/:employee_no", h.GetByEmployeeNo)
		employees.GET("/:id", h.GetByID)
		employees.GET("/:id/can-delete", h.CanDelete)

		employees.POST("", middleware.RequireWriter(), h.Create)
		employees.PUT("/:id", middleware.RequireWriter(), h.Update)

		// Resignation and reinstatement. This is the endpoint to use when
		// somebody leaves: it changes the status and records the date, and it
		// touches no payroll, salary or insurance row. Those are retained for
		// three years and the schema enforces it with ON DELETE RESTRICT
		// (db/migrations/000019_retention_and_amount_checks).
		employees.POST("/:id/status", middleware.RequireWriter(), h.ChangeStatus)

		// Deleting a person is admin-only and still refuses whenever anything
		// must be retained; the answer then points at the status endpoint.
		employees.DELETE("/:id", middleware.RequireAdmin(), h.Delete)
	}
}

// registerLeaveRoutes registers /leave-types, /leaves and /leave-balances.
func registerLeaveRoutes(tenant *gin.RouterGroup, h *handler.LeaveHandler) {
	if h == nil {
		return
	}

	leaveTypes := tenant.Group("/leave-types")
	{
		leaveTypes.GET("", h.ListTypes)
		leaveTypes.GET("/:id", h.GetType)

		leaveTypes.POST("", middleware.RequireWriter(), h.CreateType)
		leaveTypes.PUT("/:id", middleware.RequireWriter(), h.UpdateType)
		leaveTypes.DELETE("/:id", middleware.RequireAdmin(), h.DeleteType)
	}

	leaves := tenant.Group("/leaves")
	{
		leaves.GET("", h.List)
		leaves.GET("/:id", h.GetByID)

		leaves.POST("", middleware.RequireWriter(), h.Create)
		leaves.PUT("/:id", middleware.RequireWriter(), h.Update)
		leaves.DELETE("/:id", middleware.RequireWriter(), h.Delete)

		// An approval consumes a balance and is irreversible except by
		// cancelling, so it needs the approver role - the same rule vouchers
		// and tax invoices already follow.
		leaves.POST("/:id/approve", middleware.RequireApprover(), h.Approve)
		leaves.POST("/:id/reject", middleware.RequireApprover(), h.Reject)
		leaves.POST("/:id/cancel", middleware.RequireWriter(), h.Cancel)
	}

	balances := tenant.Group("/leave-balances")
	{
		balances.GET("", h.ListBalances)
		balances.PUT("", middleware.RequireAdmin(), h.GrantBalance)
	}
}
