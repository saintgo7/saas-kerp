package handler

import (
	stderrors "errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/errors"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// StockHandler serves stock balances, the movement ledger and the derived
// alerts.
type StockHandler struct {
	service service.StockService
}

// NewStockHandler creates a StockHandler.
func NewStockHandler(svc service.StockService) *StockHandler {
	return &StockHandler{service: svc}
}

// respondStockError answers a mapped domain error, or a logged 500.
func respondStockError(c *gin.Context, err error) {
	switch {
	case stderrors.Is(err, domain.ErrStockNotFound):
		response.ErrorLogged(c, http.StatusNotFound, errors.CodeNotFound, "Stock record not found", err)
	case stderrors.Is(err, domain.ErrStockNegativeResult):
		// 422, not 400: the request was well formed, the warehouse simply does
		// not hold enough. Same class as the accounting errors in codes.go.
		response.ErrorLogged(c, http.StatusUnprocessableEntity, errors.CodeInsufficientBalance,
			"Insufficient stock for this movement", err)
	case stderrors.Is(err, domain.ErrStockInsufficient):
		response.ErrorLogged(c, http.StatusUnprocessableEntity, errors.CodeInsufficientBalance,
			"Insufficient stock for this movement", err)
	case stderrors.Is(err, domain.ErrStockQuantityNotPositive):
		response.ErrorLogged(c, http.StatusBadRequest, errors.CodeOutOfRange,
			"Movement quantity must be greater than zero", err)
	case stderrors.Is(err, domain.ErrStockInvalidMovement):
		response.ErrorLogged(c, http.StatusBadRequest, errors.CodeInvalidInput, "Invalid movement type", err)
	case stderrors.Is(err, domain.ErrStockSameWarehouse):
		response.ErrorLogged(c, http.StatusBadRequest, errors.CodeInvalidInput,
			"Source and destination warehouses must differ", err)
	case stderrors.Is(err, domain.ErrOrderNothingToPost):
		response.ErrorLogged(c, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction,
			"The requested quantity matches the current balance; nothing to adjust", err)
	default:
		// Product and warehouse lookups reach here too.
		if status, code, message, ok := productErrorStatus(err); ok {
			response.ErrorLogged(c, status, code, message, err)
			return
		}
		response.InternalErrorLogged(c, "Internal server error", err)
	}
}

// parseOptionalDate reads a YYYY-MM-DD query parameter that may be absent.
func parseOptionalDate(c *gin.Context, key string) (*time.Time, bool) {
	raw := c.Query(key)
	if raw == "" {
		return nil, true
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

// ListStocks handles GET /stocks.
func (h *StockHandler) ListStocks(c *gin.Context) {
	filter := &service.StockFilter{
		CompanyID:  appctx.GetCompanyID(c),
		SearchTerm: c.Query("search"),
		// Resolved in SQL, so "low stock" means low stock across the tenant and
		// not merely on the page being viewed.
		BelowMinStock: c.Query("below_min_stock") == "true",
	}

	warehouseID, ok := parseOptionalUUID(c, "warehouse_id")
	if !ok {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid warehouse ID")
		return
	}
	filter.WarehouseID = warehouseID

	productID, ok := parseOptionalUUID(c, "product_id")
	if !ok {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid product ID")
		return
	}
	filter.ProductID = productID
	filter.Page, filter.PageSize = parsePageParams(c, 20)

	stocks, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		respondStockError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		dto.FromStocks(stocks),
		listMeta(total, filter.Page, filter.PageSize),
	))
}

// ListMovements handles GET /stock-movements.
func (h *StockHandler) ListMovements(c *gin.Context) {
	filter := &service.MovementFilter{
		CompanyID:    appctx.GetCompanyID(c),
		MovementType: c.Query("movement_type"),
	}

	warehouseID, ok := parseOptionalUUID(c, "warehouse_id")
	if !ok {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid warehouse ID")
		return
	}
	filter.WarehouseID = warehouseID

	productID, ok := parseOptionalUUID(c, "product_id")
	if !ok {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid product ID")
		return
	}
	filter.ProductID = productID

	dateFrom, ok := parseOptionalDate(c, "date_from")
	if !ok {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid date_from; expected YYYY-MM-DD")
		return
	}
	filter.DateFrom = dateFrom

	dateTo, ok := parseOptionalDate(c, "date_to")
	if !ok {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid date_to; expected YYYY-MM-DD")
		return
	}
	if dateTo != nil {
		// A date filter means the whole day. Without this, date_to=2026-09-08
		// silently excludes every movement made that day.
		endOfDay := dateTo.Add(24*time.Hour - time.Nanosecond)
		filter.DateTo = &endOfDay
	}

	filter.Page, filter.PageSize = parsePageParams(c, 20)

	movements, total, err := h.service.ListMovements(c.Request.Context(), filter)
	if err != nil {
		respondStockError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		dto.FromStockMovements(movements),
		listMeta(total, filter.Page, filter.PageSize),
	))
}

// AdjustStock handles POST /stock-adjustments.
func (h *StockHandler) AdjustStock(c *gin.Context) {
	var req dto.AdjustStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", err.Error())
		return
	}

	in := &service.AdjustStockInput{
		CompanyID:      appctx.GetCompanyID(c),
		ProductID:      uuid.MustParse(req.ProductID),
		WarehouseID:    uuid.MustParse(req.WarehouseID),
		UserID:         appctx.GetUserID(c),
		TargetQuantity: req.TargetQuantity,
		Reason:         req.Reason,
	}

	movement, err := h.service.Adjust(c.Request.Context(), in)
	if err != nil {
		respondStockError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromStockMovement(movement)))
}

// TransferStock handles POST /stock-transfers.
func (h *StockHandler) TransferStock(c *gin.Context) {
	var req dto.TransferStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", err.Error())
		return
	}

	in := &service.TransferStockInput{
		CompanyID:       appctx.GetCompanyID(c),
		ProductID:       uuid.MustParse(req.ProductID),
		FromWarehouseID: uuid.MustParse(req.FromWarehouseID),
		ToWarehouseID:   uuid.MustParse(req.ToWarehouseID),
		Quantity:        req.Quantity,
		UserID:          appctx.GetUserID(c),
		Note:            req.Note,
	}

	if err := h.service.Transfer(c.Request.Context(), in); err != nil {
		respondStockError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListAlerts handles GET /stock-alerts.
//
// Alerts are derived, not stored, so there is no read state to update and no
// POST /stock-alerts/:id/read to go with this. See domain.StockAlert.
func (h *StockHandler) ListAlerts(c *gin.Context) {
	alertType := c.Query("alert_type")
	switch alertType {
	case "", string(domain.AlertLowStock), string(domain.AlertOutOfStock), string(domain.AlertOverstock):
	default:
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidInput, "Invalid alert type")
		return
	}

	alerts, err := h.service.ListAlerts(c.Request.Context(), appctx.GetCompanyID(c), alertType)
	if err != nil {
		respondStockError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromStockAlerts(alerts)))
}

// GetStats handles GET /stocks/stats.
func (h *StockHandler) GetStats(c *gin.Context) {
	summary, err := h.service.GetSummary(c.Request.Context(), appctx.GetCompanyID(c))
	if err != nil {
		respondStockError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.StockStatsResponse{
		StockRecords:    summary.StockRecords,
		ProductCount:    summary.ProductCount,
		WarehouseCount:  summary.WarehouseCount,
		TotalStockValue: summary.TotalStockValue,
		LowStockCount:   summary.LowStockCount,
		OutOfStockCount: summary.OutOfStockCount,
	}))
}
