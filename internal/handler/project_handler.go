package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// ProjectHandler handles HTTP requests for projects
type ProjectHandler struct {
	service service.ProjectService
}

// NewProjectHandler creates a new ProjectHandler
func NewProjectHandler(svc service.ProjectService) *ProjectHandler {
	return &ProjectHandler{service: svc}
}

// RegisterRoutes registers project routes
func (h *ProjectHandler) RegisterRoutes(r *gin.RouterGroup) {
	projects := r.Group("/projects")
	{
		projects.GET("", h.List)
		projects.POST("", h.Create)
		projects.GET("/stats", h.GetStats)
		projects.GET("/:id", h.GetByID)
		projects.PUT("/:id", h.Update)
		projects.DELETE("/:id", h.Delete)
		projects.GET("/:id/can-delete", h.CanDelete)
		projects.GET("/code/:code", h.GetByCode)
	}
}

// Create handles POST /projects
func (h *ProjectHandler) Create(c *gin.Context) {
	var req dto.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	companyID := appctx.GetCompanyID(c)

	project, err := req.ToProject(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_002", err.Error()))
		return
	}

	if err := h.service.Create(c.Request.Context(), project); err != nil {
		switch err {
		case domain.ErrProjectCodeExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse("BIZ_001", "Project code already exists"))
		case domain.ErrProjectCodeEmpty:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_003", "Project code is required"))
		case domain.ErrProjectNameEmpty:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Project name is required"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		}
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromProject(project)))
}

// List handles GET /projects
func (h *ProjectHandler) List(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	filter := repository.ProjectFilter{
		CompanyID:  companyID,
		SearchTerm: c.Query("search"),
		Page:       1,
		PageSize:   20,
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			filter.Page = p
		}
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 {
			filter.PageSize = ps
		}
	}
	if status := c.Query("status"); status != "" {
		s := domain.ProjectStatus(status)
		if s.IsValid() {
			filter.Status = &s
		}
	}
	if managerID := c.Query("manager_id"); managerID != "" {
		if id, err := uuid.Parse(managerID); err == nil {
			filter.ManagerID = &id
		}
	}
	if isActive := c.Query("is_active"); isActive != "" {
		active := isActive == "true"
		filter.IsActive = &active
	}

	projects, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		dto.FromProjects(projects),
		&dto.MetaInfo{
			Total:      total,
			Page:       filter.Page,
			PageSize:   filter.PageSize,
			TotalPages: int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize)),
		},
	))
}

// GetByID handles GET /projects/:id
func (h *ProjectHandler) GetByID(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_005", "Invalid project ID"))
		return
	}

	project, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrProjectNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Project not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromProject(project)))
}

// GetByCode handles GET /projects/code/:code
func (h *ProjectHandler) GetByCode(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	code := c.Param("code")

	project, err := h.service.GetByCode(c.Request.Context(), companyID, code)
	if err != nil {
		if err == domain.ErrProjectNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Project not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromProject(project)))
}

// Update handles PUT /projects/:id
func (h *ProjectHandler) Update(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_005", "Invalid project ID"))
		return
	}

	var req dto.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	// Get existing project
	project, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrProjectNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Project not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	// Apply updates
	req.ApplyTo(project)

	if err := h.service.Update(c.Request.Context(), project); err != nil {
		switch err {
		case domain.ErrProjectCodeExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse("BIZ_001", "Project code already exists"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromProject(project)))
}

// Delete handles DELETE /projects/:id
func (h *ProjectHandler) Delete(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_005", "Invalid project ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), companyID, id); err != nil {
		switch err {
		case domain.ErrProjectNotFound:
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Project not found"))
		case domain.ErrProjectInUse:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("BIZ_002", "Project is in use and cannot be deleted"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// CanDelete handles GET /projects/:id/can-delete
func (h *ProjectHandler) CanDelete(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_005", "Invalid project ID"))
		return
	}

	canDelete, reason, err := h.service.CanDelete(c.Request.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrProjectNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Project not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{
		"can_delete": canDelete,
		"reason":     reason,
	}))
}

// GetStats handles GET /projects/stats
func (h *ProjectHandler) GetStats(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	stats, err := h.service.GetStats(c.Request.Context(), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.ProjectStatsResponse{
		TotalCount:      stats.TotalCount,
		ActiveCount:     stats.ActiveCount,
		CompletedCount:  stats.CompletedCount,
		OnHoldCount:     stats.OnHoldCount,
		TotalBudget:     stats.TotalBudget,
		TotalActualCost: stats.TotalActualCost,
	}))
}
