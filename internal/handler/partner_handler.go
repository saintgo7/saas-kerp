package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// PartnerHandler handles HTTP requests for partners
type PartnerHandler struct {
	service service.PartnerService
}

// NewPartnerHandler creates a new PartnerHandler
func NewPartnerHandler(svc service.PartnerService) *PartnerHandler {
	return &PartnerHandler{service: svc}
}

// RegisterRoutes registers partner routes
func (h *PartnerHandler) RegisterRoutes(r *gin.RouterGroup) {
	partners := r.Group("/partners")
	{
		partners.POST("", h.Create)
		partners.GET("", h.List)
		partners.GET("/stats", h.GetStats)
		partners.GET("/:id", h.GetByID)
		partners.PUT("/:id", h.Update)
		partners.DELETE("/:id", h.Delete)
		partners.GET("/code/:code", h.GetByCode)
		partners.GET("/bizno/:bizno", h.GetByBusinessNumber)
		partners.GET("/:id/can-delete", h.CanDelete)
		partners.POST("/activate", h.Activate)
		partners.POST("/deactivate", h.Deactivate)
	}
}

// Create handles POST /partners
func (h *PartnerHandler) Create(c *gin.Context) {
	var req dto.CreatePartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	companyID := appctx.GetCompanyID(c)

	partner := &domain.Partner{
		TenantModel: domain.TenantModel{
			CompanyID: companyID,
		},
		Code:            req.Code,
		Name:            req.Name,
		NameEn:          req.NameEn,
		BusinessNumber:  req.BusinessNumber,
		PartnerType:     req.PartnerType,
		Representative:  req.Representative,
		Phone:           req.Phone,
		Fax:             req.Fax,
		Email:           req.Email,
		Website:         req.Website,
		ZipCode:         req.ZipCode,
		Address:         req.Address,
		AddressDetail:   req.AddressDetail,
		PaymentTermDays: req.PaymentTermDays,
		CreditLimit:     req.CreditLimit,
		IsActive:        true,
	}

	if req.IsActive != nil {
		partner.IsActive = *req.IsActive
	}

	if req.ARAccountID != "" {
		id, _ := uuid.Parse(req.ARAccountID)
		partner.ARAccountID = &id
	}
	if req.APAccountID != "" {
		id, _ := uuid.Parse(req.APAccountID)
		partner.APAccountID = &id
	}

	if err := h.service.Create(c.Request.Context(), partner); err != nil {
		switch err {
		case service.ErrPartnerCodeExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse("BIZ_001", "Partner code already exists"))
		case service.ErrPartnerBizNoExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse("BIZ_002", "Business number already exists"))
		case service.ErrPartnerInvalidType:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_003", "Invalid partner type"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		}
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromPartner(partner)))
}

// List handles GET /partners
func (h *PartnerHandler) List(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	filter := &service.PartnerFilter{
		CompanyID:   companyID,
		PartnerType: c.Query("type"),
		SearchTerm:  c.Query("search"),
		Page:        1,
		PageSize:    20,
	}

	if page := c.Query("page"); page != "" {
		if p, err := parseInt(page); err == nil {
			filter.Page = p
		}
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := parseInt(pageSize); err == nil {
			filter.PageSize = ps
		}
	}
	if isActive := c.Query("is_active"); isActive != "" {
		active := isActive == "true"
		filter.IsActive = &active
	}

	partners, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		dto.FromPartners(partners),
		&dto.MetaInfo{
			Total:      total,
			Page:       filter.Page,
			PageSize:   filter.PageSize,
			TotalPages: int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize)),
		},
	))
}

// GetByID handles GET /partners/:id
func (h *PartnerHandler) GetByID(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid partner ID"))
		return
	}

	partner, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Partner not found"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromPartner(partner)))
}

// GetByCode handles GET /partners/code/:code
func (h *PartnerHandler) GetByCode(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	code := c.Param("code")

	partner, err := h.service.GetByCode(c.Request.Context(), companyID, code)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Partner not found"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromPartner(partner)))
}

// GetByBusinessNumber handles GET /partners/bizno/:bizno
func (h *PartnerHandler) GetByBusinessNumber(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	bizNo := c.Param("bizno")

	partner, err := h.service.GetByBusinessNumber(c.Request.Context(), companyID, bizNo)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Partner not found"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromPartner(partner)))
}

// Update handles PUT /partners/:id
func (h *PartnerHandler) Update(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid partner ID"))
		return
	}

	var req dto.UpdatePartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	// Get existing partner
	partner, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Partner not found"))
		return
	}

	// Update fields
	partner.Code = req.Code
	partner.Name = req.Name
	partner.NameEn = req.NameEn
	partner.BusinessNumber = req.BusinessNumber
	partner.PartnerType = req.PartnerType
	partner.Representative = req.Representative
	partner.Phone = req.Phone
	partner.Fax = req.Fax
	partner.Email = req.Email
	partner.Website = req.Website
	partner.ZipCode = req.ZipCode
	partner.Address = req.Address
	partner.AddressDetail = req.AddressDetail
	partner.PaymentTermDays = req.PaymentTermDays
	partner.CreditLimit = req.CreditLimit

	if req.IsActive != nil {
		partner.IsActive = *req.IsActive
	}

	if req.ARAccountID != "" {
		accountID, _ := uuid.Parse(req.ARAccountID)
		partner.ARAccountID = &accountID
	} else {
		partner.ARAccountID = nil
	}
	if req.APAccountID != "" {
		accountID, _ := uuid.Parse(req.APAccountID)
		partner.APAccountID = &accountID
	} else {
		partner.APAccountID = nil
	}

	if err := h.service.Update(c.Request.Context(), partner); err != nil {
		switch err {
		case service.ErrPartnerCodeExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse("BIZ_001", "Partner code already exists"))
		case service.ErrPartnerBizNoExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse("BIZ_002", "Business number already exists"))
		case service.ErrPartnerInvalidType:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_003", "Invalid partner type"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromPartner(partner)))
}

// Delete handles DELETE /partners/:id
func (h *PartnerHandler) Delete(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid partner ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), companyID, id); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("BIZ_004", err.Error()))
		return
	}

	c.Status(http.StatusNoContent)
}

// CanDelete handles GET /partners/:id/can-delete
func (h *PartnerHandler) CanDelete(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid partner ID"))
		return
	}

	canDelete, reason, err := h.service.CanDelete(c.Request.Context(), companyID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Partner not found"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{
		"can_delete": canDelete,
		"reason":     reason,
	}))
}

// Activate handles POST /partners/activate
func (h *PartnerHandler) Activate(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	var req dto.BulkStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	ids := make([]uuid.UUID, len(req.IDs))
	for i, idStr := range req.IDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid partner ID: "+idStr))
			return
		}
		ids[i] = id
	}

	if err := h.service.Activate(c.Request.Context(), companyID, ids); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"activated": len(ids)}))
}

// Deactivate handles POST /partners/deactivate
func (h *PartnerHandler) Deactivate(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	var req dto.BulkStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	ids := make([]uuid.UUID, len(req.IDs))
	for i, idStr := range req.IDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid partner ID: "+idStr))
			return
		}
		ids[i] = id
	}

	if err := h.service.Deactivate(c.Request.Context(), companyID, ids); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"deactivated": len(ids)}))
}

// GetStats handles GET /partners/stats
func (h *PartnerHandler) GetStats(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	stats, err := h.service.GetStats(c.Request.Context(), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.PartnerStatsResponse{
		TotalCount:    stats.TotalCount,
		CustomerCount: stats.CustomerCount,
		VendorCount:   stats.VendorCount,
		ActiveCount:   stats.ActiveCount,
		InactiveCount: stats.InactiveCount,
	}))
}
