// Package handler provides HTTP handlers for tax invoice operations.
package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
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
		tax.POST("", h.Create)
		tax.GET("", h.List)
		tax.GET("/:id", h.GetByID)
		tax.PUT("/:id", h.Update)
		tax.DELETE("/:id", h.Delete)
		tax.POST("/:id/issue", h.Issue)
		tax.POST("/:id/transmit", h.TransmitToNTS)
		tax.POST("/:id/cancel", h.Cancel)
		tax.GET("/summary", h.GetSummary)
		tax.POST("/sync", h.SyncFromHometax)
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
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(invoice))
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
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	totalPages := int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize))
	c.JSON(http.StatusOK, dto.SuccessWithMeta(invoices, &dto.MetaInfo{
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}))
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

	c.JSON(http.StatusOK, dto.SuccessResponse(invoice))
}

// Update handles PUT /tax-invoices/:id
func (h *TaxInvoiceHandler) Update(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, dto.ErrorResponse("SRV_001", "Not implemented"))
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
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("BIZ_004", err.Error()))
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
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("BIZ_004", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(invoice))
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
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("SRV_003", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(invoice))
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
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("BIZ_004", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(invoice))
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
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
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
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("SRV_003", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(map[string]int{"synced_count": count}))
}

// parseInt parses a string to int
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
