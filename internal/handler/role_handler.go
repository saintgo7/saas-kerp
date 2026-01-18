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

// RoleHandler handles HTTP requests for roles
type RoleHandler struct {
	service service.RoleService
}

// NewRoleHandler creates a new RoleHandler
func NewRoleHandler(svc service.RoleService) *RoleHandler {
	return &RoleHandler{service: svc}
}

// RegisterRoutes registers role routes
func (h *RoleHandler) RegisterRoutes(r *gin.RouterGroup) {
	roles := r.Group("/roles")
	{
		roles.GET("", h.List)
		roles.POST("", h.Create)
		roles.GET("/:id", h.GetByID)
		roles.PUT("/:id", h.Update)
		roles.DELETE("/:id", h.Delete)
		roles.PUT("/:id/permissions", h.SetPermissions)
		roles.GET("/:id/can-delete", h.CanDelete)
	}
}

// Create handles POST /roles
func (h *RoleHandler) Create(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	companyID := appctx.GetCompanyID(c)

	role, err := req.ToRole(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_002", err.Error()))
		return
	}

	if err := h.service.Create(c.Request.Context(), role); err != nil {
		switch err {
		case domain.ErrRoleCodeExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse("BIZ_001", "Role code already exists"))
		case domain.ErrRoleCodeEmpty:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_003", "Role code is required"))
		case domain.ErrRoleNameEmpty:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Role name is required"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		}
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromRole(role)))
}

// List handles GET /roles
func (h *RoleHandler) List(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	filter := repository.RoleFilter{
		CompanyID:  companyID,
		SearchTerm: c.Query("search"),
		Page:       1,
		PageSize:   50,
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
	if isActive := c.Query("is_active"); isActive != "" {
		active := isActive == "true"
		filter.IsActive = &active
	}

	roles, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		dto.FromRoles(roles),
		&dto.MetaInfo{
			Total:      total,
			Page:       filter.Page,
			PageSize:   filter.PageSize,
			TotalPages: int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize)),
		},
	))
}

// GetByID handles GET /roles/:id
func (h *RoleHandler) GetByID(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_005", "Invalid role ID"))
		return
	}

	role, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrRoleNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Role not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromRole(role)))
}

// Update handles PUT /roles/:id
func (h *RoleHandler) Update(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_005", "Invalid role ID"))
		return
	}

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	// Get existing role
	role, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrRoleNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Role not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	// Apply updates
	req.ApplyTo(role)

	if err := h.service.Update(c.Request.Context(), role); err != nil {
		switch err {
		case domain.ErrRoleCodeExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse("BIZ_001", "Role code already exists"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromRole(role)))
}

// Delete handles DELETE /roles/:id
func (h *RoleHandler) Delete(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_005", "Invalid role ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), companyID, id); err != nil {
		switch err {
		case domain.ErrRoleNotFound:
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Role not found"))
		case domain.ErrRoleInUse:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("BIZ_002", "Role is in use and cannot be deleted"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// SetPermissions handles PUT /roles/:id/permissions
func (h *RoleHandler) SetPermissions(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_005", "Invalid role ID"))
		return
	}

	var req dto.SetPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	permissions := req.ToPermissions()

	if err := h.service.SetPermissions(c.Request.Context(), companyID, id, permissions); err != nil {
		if err == domain.ErrRoleNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Role not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	// Get updated role
	role, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromRole(role)))
}

// CanDelete handles GET /roles/:id/can-delete
func (h *RoleHandler) CanDelete(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_005", "Invalid role ID"))
		return
	}

	canDelete, reason, err := h.service.CanDelete(c.Request.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrRoleNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Role not found"))
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
