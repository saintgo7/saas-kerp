package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// EmployeeHandler handles HTTP requests for employees.
type EmployeeHandler struct {
	service service.EmployeeService
}

// NewEmployeeHandler creates a new EmployeeHandler.
func NewEmployeeHandler(svc service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{service: svc}
}

// List handles GET /employees
func (h *EmployeeHandler) List(c *gin.Context) {
	filter := &service.EmployeeFilter{
		CompanyID:  appctx.GetCompanyID(c),
		SearchTerm: c.Query("search"),
		SortBy:     c.Query("sort_by"),
		SortDir:    c.Query("sort_dir"),
	}
	filter.Page, filter.PageSize = parsePageParams(c, 20)

	if raw := c.Query("status"); raw != "" {
		filter.Status = domain.EmployeeStatus(raw)
	}
	if raw := c.Query("employment_type"); raw != "" {
		filter.EmploymentType = domain.EmploymentType(raw)
	}

	var err error
	if filter.DepartmentID, err = queryUUID(c, "department_id"); err != nil {
		hrInvalidID(c, "department")
		return
	}
	if filter.PositionID, err = queryUUID(c, "position_id"); err != nil {
		hrInvalidID(c, "position")
		return
	}
	if filter.ManagerID, err = queryUUID(c, "manager_id"); err != nil {
		hrInvalidID(c, "manager")
		return
	}
	if filter.HiredFrom, err = queryDate(c, "hired_from"); err != nil {
		respondHRError(c, err)
		return
	}
	if filter.HiredTo, err = queryDate(c, "hired_to"); err != nil {
		respondHRError(c, err)
		return
	}

	employees, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.Paginated(c, dto.FromEmployees(employees), filter.Page, filter.PageSize, total)
}

// GetStats handles GET /employees/stats
func (h *EmployeeHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context(), appctx.GetCompanyID(c))
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, dto.EmployeeStatsResponse{
		TotalCount:      stats.TotalCount,
		ActiveCount:     stats.ActiveCount,
		OnLeaveCount:    stats.OnLeaveCount,
		ResignedCount:   stats.ResignedCount,
		TerminatedCount: stats.TerminatedCount,
	})
}

// GetByID handles GET /employees/:id
func (h *EmployeeHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "employee")
		return
	}

	employee, err := h.service.GetByID(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, dto.FromEmployee(employee))
}

// GetByEmployeeNo handles GET /employees/no/:employee_no
func (h *EmployeeHandler) GetByEmployeeNo(c *gin.Context) {
	employee, err := h.service.GetByEmployeeNo(c.Request.Context(), appctx.GetCompanyID(c), c.Param("employee_no"))
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, dto.FromEmployee(employee))
}

// Create handles POST /employees
func (h *EmployeeHandler) Create(c *gin.Context) {
	var req dto.EmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		hrBindError(c, err)
		return
	}

	actorID := appctx.GetUserID(c)
	employee := &domain.Employee{
		TenantModel: domain.TenantModel{CompanyID: appctx.GetCompanyID(c)},
	}
	if actorID != uuid.Nil {
		employee.CreatedBy = &actorID
		employee.UpdatedBy = &actorID
	}

	if err := req.ApplyTo(employee); err != nil {
		respondHRError(c, err)
		return
	}

	// The plaintext resident number travels as its own argument and is
	// encrypted inside the service. It is never assigned to the domain object
	// here and never reaches a response.
	residentNumber := ""
	if req.ResidentNumber != nil {
		residentNumber = *req.ResidentNumber
	}

	if err := h.service.Create(c.Request.Context(), employee, residentNumber); err != nil {
		respondHRError(c, err)
		return
	}

	response.Created(c, dto.FromEmployee(employee))
}

// Update handles PUT /employees/:id
func (h *EmployeeHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "employee")
		return
	}

	var req dto.EmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		hrBindError(c, err)
		return
	}

	companyID := appctx.GetCompanyID(c)

	// Load the stored row for THIS company first, then apply the request to
	// it. An object built from the request alone would let a client move an
	// employee between tenants by echoing a different company_id.
	employee, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		respondHRError(c, err)
		return
	}

	if err := req.ApplyTo(employee); err != nil {
		respondHRError(c, err)
		return
	}
	if actorID := appctx.GetUserID(c); actorID != uuid.Nil {
		employee.UpdatedBy = &actorID
	}

	if err := h.service.Update(c.Request.Context(), employee, req.ResidentNumber); err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, dto.FromEmployee(employee))
}

// ChangeStatus handles POST /employees/:id/status
//
// This is the resignation path. It never deletes anything: payroll, salary and
// insurance rows stay exactly where they are, which is what the three-year
// retention requirement and the ON DELETE RESTRICT foreign keys demand.
func (h *EmployeeHandler) ChangeStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "employee")
		return
	}

	var req dto.EmployeeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		hrBindError(c, err)
		return
	}

	resignationDate, err := req.ResignationDateValue()
	if err != nil {
		respondHRError(c, err)
		return
	}

	companyID := appctx.GetCompanyID(c)
	change := service.EmployeeStatusChange{
		Status:          domain.EmployeeStatus(req.Status),
		ResignationDate: resignationDate,
		Reason:          req.Reason,
		ActorID:         appctx.GetUserID(c),
	}

	if err := h.service.ChangeStatus(c.Request.Context(), companyID, id, change); err != nil {
		respondHRError(c, err)
		return
	}

	employee, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, dto.FromEmployee(employee))
}

// Delete handles DELETE /employees/:id
func (h *EmployeeHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "employee")
		return
	}

	if err := h.service.Delete(c.Request.Context(), appctx.GetCompanyID(c), id); err != nil {
		respondHRError(c, err)
		return
	}

	response.NoContent(c)
}

// CanDelete handles GET /employees/:id/can-delete
func (h *EmployeeHandler) CanDelete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "employee")
		return
	}

	canDelete, reason, err := h.service.CanDelete(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, gin.H{"can_delete": canDelete, "reason": reason})
}

// queryUUID reads an optional UUID query parameter.
func queryUUID(c *gin.Context, key string) (*uuid.UUID, error) {
	raw := c.Query(key)
	if raw == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// queryDate reads an optional YYYY-MM-DD query parameter.
func queryDate(c *gin.Context, key string) (*time.Time, error) {
	raw := c.Query(key)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, dto.ErrInvalidHRDate
	}
	return &parsed, nil
}
