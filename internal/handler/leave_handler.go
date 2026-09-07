package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// LeaveHandler handles HTTP requests for leave types, leave requests and
// leave balances.
type LeaveHandler struct {
	service service.LeaveService
}

// NewLeaveHandler creates a new LeaveHandler.
func NewLeaveHandler(svc service.LeaveService) *LeaveHandler {
	return &LeaveHandler{service: svc}
}

// ---------------------------------------------------------------------------
// Leave types
// ---------------------------------------------------------------------------

// ListTypes handles GET /leave-types
func (h *LeaveHandler) ListTypes(c *gin.Context) {
	filter := &service.LeaveTypeFilter{
		CompanyID:  appctx.GetCompanyID(c),
		SearchTerm: c.Query("search"),
	}
	filter.Page, filter.PageSize = parsePageParams(c, 20)

	if isActive := c.Query("is_active"); isActive != "" {
		active := isActive == "true"
		filter.IsActive = &active
	}

	types, total, err := h.service.ListTypes(c.Request.Context(), filter)
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.Paginated(c, dto.FromLeaveTypes(types), filter.Page, filter.PageSize, total)
}

// GetType handles GET /leave-types/:id
func (h *LeaveHandler) GetType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "leave type")
		return
	}

	leaveType, err := h.service.GetType(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, dto.FromLeaveType(leaveType))
}

// CreateType handles POST /leave-types
func (h *LeaveHandler) CreateType(c *gin.Context) {
	var req dto.LeaveTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		hrBindError(c, err)
		return
	}

	leaveType := req.NewLeaveType(appctx.GetCompanyID(c))
	if err := h.service.CreateType(c.Request.Context(), leaveType); err != nil {
		respondHRError(c, err)
		return
	}

	response.Created(c, dto.FromLeaveType(leaveType))
}

// UpdateType handles PUT /leave-types/:id
func (h *LeaveHandler) UpdateType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "leave type")
		return
	}

	var req dto.LeaveTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		hrBindError(c, err)
		return
	}

	companyID := appctx.GetCompanyID(c)
	leaveType, err := h.service.GetType(c.Request.Context(), companyID, id)
	if err != nil {
		respondHRError(c, err)
		return
	}

	req.ApplyTo(leaveType)
	if err := h.service.UpdateType(c.Request.Context(), leaveType); err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, dto.FromLeaveType(leaveType))
}

// DeleteType handles DELETE /leave-types/:id
func (h *LeaveHandler) DeleteType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "leave type")
		return
	}

	if err := h.service.DeleteType(c.Request.Context(), appctx.GetCompanyID(c), id); err != nil {
		respondHRError(c, err)
		return
	}

	response.NoContent(c)
}

// ---------------------------------------------------------------------------
// Leave requests
// ---------------------------------------------------------------------------

// List handles GET /leaves
func (h *LeaveHandler) List(c *gin.Context) {
	filter := &service.LeaveFilter{
		CompanyID: appctx.GetCompanyID(c),
	}
	filter.Page, filter.PageSize = parsePageParams(c, 20)

	if raw := c.Query("status"); raw != "" {
		filter.Status = domain.LeaveStatus(raw)
	}

	var err error
	if filter.EmployeeID, err = queryUUID(c, "employee_id"); err != nil {
		hrInvalidID(c, "employee")
		return
	}
	if filter.LeaveTypeID, err = queryUUID(c, "leave_type_id"); err != nil {
		hrInvalidID(c, "leave type")
		return
	}
	if filter.DateFrom, err = queryDate(c, "date_from"); err != nil {
		respondHRError(c, err)
		return
	}
	if filter.DateTo, err = queryDate(c, "date_to"); err != nil {
		respondHRError(c, err)
		return
	}

	leaves, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.Paginated(c, dto.FromLeaves(leaves), filter.Page, filter.PageSize, total)
}

// GetByID handles GET /leaves/:id
func (h *LeaveHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "leave request")
		return
	}

	leave, err := h.service.GetByID(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, dto.FromLeave(leave))
}

// Create handles POST /leaves
func (h *LeaveHandler) Create(c *gin.Context) {
	var req dto.CreateLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		hrBindError(c, err)
		return
	}

	leave, err := req.ToLeave(appctx.GetCompanyID(c))
	if err != nil {
		respondHRError(c, err)
		return
	}
	if actorID := appctx.GetUserID(c); actorID != uuid.Nil {
		leave.CreatedBy = &actorID
	}

	if err := h.service.Create(c.Request.Context(), leave); err != nil {
		respondHRError(c, err)
		return
	}

	response.Created(c, dto.FromLeave(leave))
}

// Update handles PUT /leaves/:id
func (h *LeaveHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "leave request")
		return
	}

	var req dto.UpdateLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		hrBindError(c, err)
		return
	}

	companyID := appctx.GetCompanyID(c)
	leave, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		respondHRError(c, err)
		return
	}

	if err := req.ApplyTo(leave); err != nil {
		respondHRError(c, err)
		return
	}

	if err := h.service.Update(c.Request.Context(), leave); err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, dto.FromLeave(leave))
}

// Approve handles POST /leaves/:id/approve
func (h *LeaveHandler) Approve(c *gin.Context) {
	h.decide(c, func(companyID, id, actorID uuid.UUID) error {
		return h.service.Approve(c.Request.Context(), companyID, id, actorID)
	})
}

// Reject handles POST /leaves/:id/reject
func (h *LeaveHandler) Reject(c *gin.Context) {
	var req dto.RejectLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		hrBindError(c, err)
		return
	}

	h.decide(c, func(companyID, id, actorID uuid.UUID) error {
		return h.service.Reject(c.Request.Context(), companyID, id, actorID, req.Reason)
	})
}

// Cancel handles POST /leaves/:id/cancel
func (h *LeaveHandler) Cancel(c *gin.Context) {
	h.decide(c, func(companyID, id, _ uuid.UUID) error {
		return h.service.Cancel(c.Request.Context(), companyID, id)
	})
}

// Delete handles DELETE /leaves/:id
func (h *LeaveHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "leave request")
		return
	}

	if err := h.service.Delete(c.Request.Context(), appctx.GetCompanyID(c), id); err != nil {
		respondHRError(c, err)
		return
	}

	response.NoContent(c)
}

// decide runs one state transition and answers with the resulting request.
func (h *LeaveHandler) decide(c *gin.Context, action func(companyID, id, actorID uuid.UUID) error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "leave request")
		return
	}

	companyID := appctx.GetCompanyID(c)
	if err := action(companyID, id, appctx.GetUserID(c)); err != nil {
		respondHRError(c, err)
		return
	}

	leave, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, dto.FromLeave(leave))
}

// ---------------------------------------------------------------------------
// Balances
// ---------------------------------------------------------------------------

// ListBalances handles GET /leave-balances
func (h *LeaveHandler) ListBalances(c *gin.Context) {
	filter := &service.LeaveBalanceFilter{
		CompanyID: appctx.GetCompanyID(c),
	}
	filter.Page, filter.PageSize = parsePageParams(c, 20)

	var err error
	if filter.EmployeeID, err = queryUUID(c, "employee_id"); err != nil {
		hrInvalidID(c, "employee")
		return
	}
	if filter.LeaveTypeID, err = queryUUID(c, "leave_type_id"); err != nil {
		hrInvalidID(c, "leave type")
		return
	}
	if raw := c.Query("fiscal_year"); raw != "" {
		year, convErr := strconv.Atoi(raw)
		if convErr != nil {
			respondHRError(c, domain.ErrLeaveBalanceFiscalYear)
			return
		}
		filter.FiscalYear = &year
	}

	balances, total, err := h.service.ListBalances(c.Request.Context(), filter)
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.Paginated(c, dto.FromLeaveBalances(balances), filter.Page, filter.PageSize, total)
}

// GrantBalance handles PUT /leave-balances
//
// It sets an entitlement. used_days is never accepted from the client: it is
// maintained by approvals and cancellations.
func (h *LeaveHandler) GrantBalance(c *gin.Context) {
	var req dto.GrantLeaveBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		hrBindError(c, err)
		return
	}

	balance, err := req.ToBalance(appctx.GetCompanyID(c))
	if err != nil {
		respondHRError(c, err)
		return
	}

	if err := h.service.GrantBalance(c.Request.Context(), balance); err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, dto.FromLeaveBalance(balance))
}
