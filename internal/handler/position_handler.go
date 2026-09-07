package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// PositionHandler handles HTTP requests for job positions.
type PositionHandler struct {
	service service.PositionService
}

// NewPositionHandler creates a new PositionHandler.
func NewPositionHandler(svc service.PositionService) *PositionHandler {
	return &PositionHandler{service: svc}
}

// List handles GET /positions
func (h *PositionHandler) List(c *gin.Context) {
	filter := &service.PositionFilter{
		CompanyID:  appctx.GetCompanyID(c),
		SearchTerm: c.Query("search"),
	}
	filter.Page, filter.PageSize = parsePageParams(c, 20)

	if isActive := c.Query("is_active"); isActive != "" {
		active := isActive == "true"
		filter.IsActive = &active
	}

	positions, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.Paginated(c, dto.FromPositions(positions), filter.Page, filter.PageSize, total)
}

// GetByID handles GET /positions/:id
func (h *PositionHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "position")
		return
	}

	position, err := h.service.GetByID(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, dto.FromPosition(position))
}

// GetByCode handles GET /positions/code/:code
func (h *PositionHandler) GetByCode(c *gin.Context) {
	position, err := h.service.GetByCode(c.Request.Context(), appctx.GetCompanyID(c), c.Param("code"))
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, dto.FromPosition(position))
}

// Create handles POST /positions
func (h *PositionHandler) Create(c *gin.Context) {
	var req dto.CreatePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		hrBindError(c, err)
		return
	}

	position := req.ToPosition(appctx.GetCompanyID(c))
	if err := h.service.Create(c.Request.Context(), position); err != nil {
		respondHRError(c, err)
		return
	}

	response.Created(c, dto.FromPosition(position))
}

// Update handles PUT /positions/:id
func (h *PositionHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "position")
		return
	}

	var req dto.UpdatePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		hrBindError(c, err)
		return
	}

	companyID := appctx.GetCompanyID(c)

	// Load the stored row first: the update is applied to what the database
	// holds for THIS company, never to an object assembled from the request.
	position, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		respondHRError(c, err)
		return
	}

	req.ApplyTo(position)
	if err := h.service.Update(c.Request.Context(), position); err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, dto.FromPosition(position))
}

// Delete handles DELETE /positions/:id
func (h *PositionHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "position")
		return
	}

	if err := h.service.Delete(c.Request.Context(), appctx.GetCompanyID(c), id); err != nil {
		respondHRError(c, err)
		return
	}

	response.NoContent(c)
}

// CanDelete handles GET /positions/:id/can-delete
func (h *PositionHandler) CanDelete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		hrInvalidID(c, "position")
		return
	}

	canDelete, reason, err := h.service.CanDelete(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondHRError(c, err)
		return
	}

	response.OK(c, gin.H{"can_delete": canDelete, "reason": reason})
}
