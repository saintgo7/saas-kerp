// Package handler provides HTTP handlers for tax invoice operations.
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/errors"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
	"github.com/saintgo7/saas-kerp/internal/middleware"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// TaxInvoiceHandler handles HTTP requests for tax invoices.
type TaxInvoiceHandler struct {
	service *service.TaxInvoiceService
}

// NewTaxInvoiceHandler creates a new tax invoice handler.
func NewTaxInvoiceHandler(svc *service.TaxInvoiceService) *TaxInvoiceHandler {
	return &TaxInvoiceHandler{service: svc}
}

// RegisterRoutes registers tax invoice routes.
func (h *TaxInvoiceHandler) RegisterRoutes(r *gin.RouterGroup) {
	tax := r.Group("/tax-invoices")
	{
		tax.GET("", h.List)
		tax.GET("/summary", h.GetSummary)
		tax.GET("/:id", h.GetByID)

		tax.POST("", middleware.RequireWriter(), h.Create)
		tax.PUT("/:id", middleware.RequireWriter(), h.Update)
		tax.DELETE("/:id", middleware.RequireWriter(), h.Delete)

		// Issuing, transmitting to the National Tax Service and cancelling are
		// filings: they leave this system and cannot be taken back.
		tax.POST("/:id/issue", middleware.RequireApprover(), h.Issue)
		tax.POST("/:id/transmit", middleware.RequireApprover(), h.TransmitToNTS)
		tax.POST("/:id/cancel", middleware.RequireApprover(), h.Cancel)
		tax.POST("/sync", middleware.RequireApprover(), h.SyncFromHometax)
	}
}

// CreateTaxInvoiceRequest represents the request body for creating a tax invoice.
type CreateTaxInvoiceRequest struct {
	InvoiceNumber          string                        `json:"invoice_number" binding:"required"`
	InvoiceType            string                        `json:"invoice_type" binding:"required,oneof=sales purchase"`
	IssueDate              string                        `json:"issue_date" binding:"required"`
	SupplierBusinessNumber string                        `json:"supplier_business_number" binding:"required,len=10"`
	SupplierName           string                        `json:"supplier_name" binding:"required"`
	SupplierCEOName        string                        `json:"supplier_ceo_name"`
	SupplierAddress        string                        `json:"supplier_address"`
	BuyerBusinessNumber    string                        `json:"buyer_business_number" binding:"required,len=10"`
	BuyerName              string                        `json:"buyer_name" binding:"required"`
	BuyerCEOName           string                        `json:"buyer_ceo_name"`
	BuyerAddress           string                        `json:"buyer_address"`
	SupplyAmount           int64                         `json:"supply_amount" binding:"required"`
	TaxAmount              int64                         `json:"tax_amount" binding:"required"`
	Items                  []CreateTaxInvoiceItemRequest `json:"items"`
	Remarks                string                        `json:"remarks"`
}

// CreateTaxInvoiceItemRequest represents a line item in the create request.
type CreateTaxInvoiceItemRequest struct {
	SupplyDate    string  `json:"supply_date"`
	Description   string  `json:"description" binding:"required"`
	Specification string  `json:"specification"`
	Quantity      float64 `json:"quantity"`
	UnitPrice     float64 `json:"unit_price"`
	Amount        int64   `json:"amount"`
	TaxAmount     int64   `json:"tax_amount"`
	Remarks       string  `json:"remarks"`
}

// Create handles POST /tax-invoices
func (h *TaxInvoiceHandler) Create(c *gin.Context) {
	var req CreateTaxInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	companyID := appctx.GetCompanyID(c)
	userID := appctx.GetUserID(c)

	issueDate, err := time.Parse("2006-01-02", req.IssueDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid issue_date format (expected YYYY-MM-DD)"))
		return
	}

	input := &service.CreateInput{
		InvoiceNumber:          req.InvoiceNumber,
		InvoiceType:            domain.TaxInvoiceType(req.InvoiceType),
		IssueDate:              issueDate,
		SupplierBusinessNumber: req.SupplierBusinessNumber,
		SupplierName:           req.SupplierName,
		SupplierCEOName:        req.SupplierCEOName,
		SupplierAddress:        req.SupplierAddress,
		BuyerBusinessNumber:    req.BuyerBusinessNumber,
		BuyerName:              req.BuyerName,
		BuyerCEOName:           req.BuyerCEOName,
		BuyerAddress:           req.BuyerAddress,
		SupplyAmount:           req.SupplyAmount,
		TaxAmount:              req.TaxAmount,
		Remarks:                req.Remarks,
	}

	for _, item := range req.Items {
		itemInput := service.CreateItemInput{
			Description:   item.Description,
			Specification: item.Specification,
			Quantity:      item.Quantity,
			UnitPrice:     item.UnitPrice,
			Amount:        item.Amount,
			TaxAmount:     item.TaxAmount,
			Remarks:       item.Remarks,
		}
		if item.SupplyDate != "" {
			if sd, err := time.Parse("2006-01-02", item.SupplyDate); err == nil {
				itemInput.SupplyDate = &sd
			}
		}
		input.Items = append(input.Items, itemInput)
	}

	invoice, err := h.service.Create(c.Request.Context(), companyID, input, &userID)
	if err != nil {
		response.InternalErrorLogged(c, "Internal server error", err)
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromTaxInvoice(invoice)))
}

// List handles GET /tax-invoices
func (h *TaxInvoiceHandler) List(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	filter := &service.TaxInvoiceFilter{
		CompanyID: companyID,
		Page:      1,
		PageSize:  20,
	}

	// Parse query parameters
	filter.Page, filter.PageSize = parsePageParams(c, 20)
	if startDate := c.Query("start_date"); startDate != "" {
		if sd, err := time.Parse("2006-01-02", startDate); err == nil {
			filter.StartDate = &sd
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if ed, err := time.Parse("2006-01-02", endDate); err == nil {
			filter.EndDate = &ed
		}
	}
	if invoiceType := c.Query("invoice_type"); invoiceType != "" {
		it := domain.TaxInvoiceType(invoiceType)
		filter.InvoiceType = &it
	}
	if status := c.Query("status"); status != "" {
		st := domain.TaxInvoiceStatus(status)
		filter.Status = &st
	}

	invoices, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		response.InternalErrorLogged(c, "Internal server error", err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(dto.FromTaxInvoices(invoices), listMeta(total, filter.Page, filter.PageSize)))
}

// GetByID handles GET /tax-invoices/:id
func (h *TaxInvoiceHandler) GetByID(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid invoice ID"))
		return
	}

	invoice, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Invoice not found"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromTaxInvoice(invoice)))
}

// Update handles PUT /tax-invoices/:id.
//
// Editing an issued tax invoice is not a supported operation: a filed invoice is
// corrected by cancelling it and issuing a new one. The route is kept so clients
// get an explicit 501 rather than a silent 404 from the router.
func (h *TaxInvoiceHandler) Update(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse(errors.CodeUnavailable,
		"Editing a tax invoice is not supported; cancel it and issue a new one"))
}

// Delete handles DELETE /tax-invoices/:id
func (h *TaxInvoiceHandler) Delete(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid invoice ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), companyID, id); err != nil {
		response.ErrorLogged(c, http.StatusConflict, "BIZ_004", "Tax invoice operation is not allowed in the current state", err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Issue handles POST /tax-invoices/:id/issue
func (h *TaxInvoiceHandler) Issue(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	userID := appctx.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid invoice ID"))
		return
	}

	invoice, err := h.service.Issue(c.Request.Context(), companyID, id, &userID)
	if err != nil {
		response.ErrorLogged(c, http.StatusConflict, "BIZ_004", "Tax invoice operation is not allowed in the current state", err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromTaxInvoice(invoice)))
}

// TransmitRequest represents the request for transmitting to NTS.
type TransmitRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// TransmitToNTS handles POST /tax-invoices/:id/transmit
func (h *TaxInvoiceHandler) TransmitToNTS(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	userID := appctx.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid invoice ID"))
		return
	}

	var req TransmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	invoice, err := h.service.TransmitToNTS(c.Request.Context(), companyID, id, req.SessionID, &userID)
	if err != nil {
		response.ErrorLogged(c, http.StatusBadGateway, "SRV_003", "External service request failed", err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromTaxInvoice(invoice)))
}

// CancelRequest represents the request for cancelling an invoice.
type CancelRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// Cancel handles POST /tax-invoices/:id/cancel
func (h *TaxInvoiceHandler) Cancel(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	userID := appctx.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid invoice ID"))
		return
	}

	var req CancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	invoice, err := h.service.Cancel(c.Request.Context(), companyID, id, req.Reason, &userID)
	if err != nil {
		response.ErrorLogged(c, http.StatusConflict, "BIZ_004", "Tax invoice operation is not allowed in the current state", err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromTaxInvoice(invoice)))
}

// GetSummary handles GET /tax-invoices/summary
func (h *TaxInvoiceHandler) GetSummary(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	startDate, err := time.Parse("2006-01-02", c.Query("start_date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "start_date is required (YYYY-MM-DD)"))
		return
	}

	endDate, err := time.Parse("2006-01-02", c.Query("end_date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "end_date is required (YYYY-MM-DD)"))
		return
	}

	summary, err := h.service.GetSummary(c.Request.Context(), companyID, startDate, endDate)
	if err != nil {
		response.InternalErrorLogged(c, "Internal server error", err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(summary))
}

// SyncRequest represents the request for syncing from Hometax.
type SyncRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}

// SyncFromHometax handles POST /tax-invoices/sync
func (h *TaxInvoiceHandler) SyncFromHometax(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	userID := appctx.GetUserID(c)

	var req SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	count, err := h.service.SyncFromHometax(c.Request.Context(), companyID, req.SessionID, req.StartDate, req.EndDate, &userID)
	if err != nil {
		response.ErrorLogged(c, http.StatusBadGateway, "SRV_003", "External service request failed", err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(map[string]int{"synced_count": count}))
}
