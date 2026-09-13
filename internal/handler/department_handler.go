package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// DepartmentHandler handles HTTP requests for the organisation chart.
type DepartmentHandler struct {
	service service.DepartmentService
}

// NewDepartmentHandler creates a new DepartmentHandler.
func NewDepartmentHandler(svc service.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{service: svc}
}

// repositoryDepartmentNotFound is the text the department repository uses for
// a missing row.
//
// internal/repository/department_repository_gorm.go answers a missing
// department with fmt.Errorf("department not found") instead of the
// domain.ErrDepartmentNotFound sentinel. errors.Is cannot recognise an
// anonymous error, so that value falls through the mapping table in
// hr_handlers.go and becomes a 500 - telling a client that asked for a
// department that does not exist that the server is broken, and logging a
// stack of noise for every mistyped identifier.
//
// Normalising it here keeps the repair inside the layer this change owns. The
// real fix is for the repository to return the sentinel; until it does, the
// comparison is against the exact string, so a repository that starts wrapping
// or rewording the error simply stops matching and the sentinels in the table
// take over.
const repositoryDepartmentNotFound = "department not found"

// respondDepartmentError answers a department error.
//
// Everything client-visible is decided by the table in hr_handlers.go; this
// only repairs the one error that arrives without an identity. No error text
// reaches the client from here.
func respondDepartmentError(c *gin.Context, err error) {
	if err != nil && err.Error() == repositoryDepartmentNotFound {
		err = domain.ErrDepartmentNotFound
	}
	respondHRError(c, err)
}

// List handles GET /departments
//
// This is the flat, paginated view. Use Tree for the nested organisation
// chart.
func (h *DepartmentHandler) List(c *gin.Context) {
	filter := &service.DepartmentFilter{
		CompanyID:  appctx.GetCompanyID(c),
		SearchTerm: c.Query("search"),
	}
	filter.Page, filter.PageSize = parsePageParams(c, 20)

	if raw := c.Query("is_active"); raw != "" {
		active := raw == "true"
		filter.IsActive = &active
	}

	parentID, err := queryUUID(c, "parent_id")
	if err != nil {
		hrInvalidID(c, "parent department")
		return
	}
	filter.ParentID = parentID

	departments, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		respondDepartmentError(c, err)
		return
	}

	response.Paginated(c, dto.FromDepartments(departments), filter.Page, filter.PageSize, total)
}

// Tree handles GET /departments/tree
//
// It returns the whole chart in one response, nested, because that is what an
// organisation chart is drawn from and a company has tens of departments, not
// thousands. Only active departments are included: that is what the repository
// query selects.
func (h *DepartmentHandler) Tree(c *gin.Context) {
	departments, err := h.service.GetTree(c.Request.Context(), appctx.GetCompanyID(c))
	if err != nil {
		respondDepartmentError(c, err)
		return
	}

	response.OK(c, dto.BuildDepartmentTree(departments))
}

// GetByID handles GET /departments/:id
func (h *DepartmentHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "department")
		return
	}

	department, err := h.service.GetByID(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondDepartmentError(c, err)
		return
	}

	response.OK(c, dto.FromDepartment(department))
}

// GetByCode handles GET /departments/code/:code
func (h *DepartmentHandler) GetByCode(c *gin.Context) {
	department, err := h.service.GetByCode(c.Request.Context(), appctx.GetCompanyID(c), c.Param("code"))
	if err != nil {
		respondDepartmentError(c, err)
		return
	}

	response.OK(c, dto.FromDepartment(department))
}

// GetChildren handles GET /departments/:id/children
func (h *DepartmentHandler) GetChildren(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "department")
		return
	}

	departments, err := h.service.GetChildren(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondDepartmentError(c, err)
		return
	}

	response.OK(c, dto.FromDepartments(departments))
}

// Create handles POST /departments
func (h *DepartmentHandler) Create(c *gin.Context) {
	var req dto.DepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		hrBindError(c, err)
		return
	}

	// The company comes from the authenticated session. dto.DepartmentRequest
	// has no company field and ApplyTo does not touch CompanyID, so there is
	// no path by which a request body could file a department under another
	// tenant.
	department := &domain.Department{
		TenantModel: domain.TenantModel{CompanyID: appctx.GetCompanyID(c)},
		IsActive:    true,
	}

	if err := req.ApplyTo(department); err != nil {
		hrInvalidID(c, "department reference")
		return
	}

	if err := h.service.Create(c.Request.Context(), department); err != nil {
		respondDepartmentError(c, err)
		return
	}

	response.Created(c, dto.FromDepartment(department))
}

// Update handles PUT /departments/:id
func (h *DepartmentHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "department")
		return
	}

	var req dto.DepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		hrBindError(c, err)
		return
	}

	companyID := appctx.GetCompanyID(c)

	// Load the stored row for THIS company first, then apply the request to
	// it. An object assembled from the request alone would let a client move a
	// department between tenants by echoing a different company id.
	department, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		respondDepartmentError(c, err)
		return
	}

	if err := req.ApplyTo(department); err != nil {
		hrInvalidID(c, "department reference")
		return
	}

	// The service owns the hierarchy rules from here: a department may not
	// become its own parent, nor a child of one of its own descendants, and
	// the level is recomputed from the parent.
	if err := h.service.Update(c.Request.Context(), department); err != nil {
		respondDepartmentError(c, err)
		return
	}

	response.OK(c, dto.FromDepartment(department))
}

// Move handles POST /departments/:id/move
//
// Re-parenting without resending the whole record, which is what an
// organisation chart edit actually is. The same circular-reference rules as
// Update apply.
func (h *DepartmentHandler) Move(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "department")
		return
	}

	var req dto.MoveDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		hrBindError(c, err)
		return
	}

	parentID, err := req.ParentUUID()
	if err != nil {
		hrInvalidID(c, "parent department")
		return
	}

	companyID := appctx.GetCompanyID(c)
	if err := h.service.Move(c.Request.Context(), companyID, id, parentID); err != nil {
		respondDepartmentError(c, err)
		return
	}

	department, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		respondDepartmentError(c, err)
		return
	}

	response.OK(c, dto.FromDepartment(department))
}

// Delete handles DELETE /departments/:id
//
// The service refuses while the department has children or is referenced by
// voucher entries; both come back as 409. Ask CanDelete first if the client
// wants to disable the button rather than explain the refusal.
func (h *DepartmentHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "department")
		return
	}

	if err := h.service.Delete(c.Request.Context(), appctx.GetCompanyID(c), id); err != nil {
		respondDepartmentError(c, err)
		return
	}

	response.NoContent(c)
}

// CanDelete handles GET /departments/:id/can-delete
func (h *DepartmentHandler) CanDelete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "department")
		return
	}

	canDelete, reason, err := h.service.CanDelete(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondDepartmentError(c, err)
		return
	}

	response.OK(c, gin.H{"can_delete": canDelete, "reason": reason})
}
