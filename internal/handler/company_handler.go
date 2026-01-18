package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// CompanyHandler handles HTTP requests for company info
type CompanyHandler struct {
	service service.CompanyService
}

// NewCompanyHandler creates a new CompanyHandler
func NewCompanyHandler(svc service.CompanyService) *CompanyHandler {
	return &CompanyHandler{service: svc}
}

// RegisterRoutes registers company routes
func (h *CompanyHandler) RegisterRoutes(r *gin.RouterGroup) {
	company := r.Group("/company")
	{
		company.GET("", h.Get)
		company.PUT("", h.Update)
		company.GET("/settings", h.GetSettings)
		company.PUT("/settings", h.UpdateSettings)
	}
}

// Get handles GET /company
func (h *CompanyHandler) Get(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	company, err := h.service.GetByID(c.Request.Context(), companyID)
	if err != nil {
		if err == domain.ErrCompanyNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Company not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromCompany(company)))
}

// Update handles PUT /company
func (h *CompanyHandler) Update(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	var req dto.UpdateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	// Get existing company
	company, err := h.service.GetByID(c.Request.Context(), companyID)
	if err != nil {
		if err == domain.ErrCompanyNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Company not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	// Apply updates
	req.ApplyTo(company)

	if err := h.service.Update(c.Request.Context(), company); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromCompany(company)))
}

// GetSettings handles GET /company/settings
func (h *CompanyHandler) GetSettings(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	company, err := h.service.GetByID(c.Request.Context(), companyID)
	if err != nil {
		if err == domain.ErrCompanyNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Company not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.CompanySettingsResponse{
		FiscalYearStart:     company.Settings.FiscalYearStart,
		DefaultCurrency:     company.Settings.DefaultCurrency,
		DecimalPlaces:       company.Settings.DecimalPlaces,
		TaxRate:             company.Settings.TaxRate,
		VoucherAutoNumber:   company.Settings.VoucherAutoNumber,
		VoucherNumberFormat: company.Settings.VoucherNumberFormat,
		InvoicePrefix:       company.Settings.InvoicePrefix,
		Timezone:            company.Settings.Timezone,
		DateFormat:          company.Settings.DateFormat,
		Language:            company.Settings.Language,
	}))
}

// UpdateSettings handles PUT /company/settings
func (h *CompanyHandler) UpdateSettings(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	var req dto.UpdateCompanySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	// Get existing company
	company, err := h.service.GetByID(c.Request.Context(), companyID)
	if err != nil {
		if err == domain.ErrCompanyNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Company not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	// Apply settings updates
	req.ApplyTo(company)

	if err := h.service.UpdateSettings(c.Request.Context(), company); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.CompanySettingsResponse{
		FiscalYearStart:     company.Settings.FiscalYearStart,
		DefaultCurrency:     company.Settings.DefaultCurrency,
		DecimalPlaces:       company.Settings.DecimalPlaces,
		TaxRate:             company.Settings.TaxRate,
		VoucherAutoNumber:   company.Settings.VoucherAutoNumber,
		VoucherNumberFormat: company.Settings.VoucherNumberFormat,
		InvoicePrefix:       company.Settings.InvoicePrefix,
		Timezone:            company.Settings.Timezone,
		DateFormat:          company.Settings.DateFormat,
		Language:            company.Settings.Language,
	}))
}
