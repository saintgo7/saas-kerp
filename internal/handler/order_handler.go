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

// OrderHandler serves purchase and sales orders.
type OrderHandler struct {
	service service.OrderService
}

// NewOrderHandler creates an OrderHandler.
func NewOrderHandler(svc service.OrderService) *OrderHandler {
	return &OrderHandler{service: svc}
}

// respondOrderError answers a mapped domain error, or a logged 500.
//
// The refused-transition errors answer 409 rather than 400: the request was
// valid, the order had simply already moved on. A double-clicked Approve lands
// here, and 409 is what tells the client to re-read rather than to re-send.
func respondOrderError(c *gin.Context, err error) {
	switch {
	case stderrors.Is(err, domain.ErrOrderNotFound):
		response.ErrorLogged(c, http.StatusNotFound, errors.CodeNotFound, "Order not found", err)
	case stderrors.Is(err, domain.ErrOrderLineNotFound):
		response.ErrorLogged(c, http.StatusNotFound, errors.CodeNotFound, "Order line not found", err)

	case stderrors.Is(err, domain.ErrOrderCannotEdit):
		response.ErrorLogged(c, http.StatusConflict, errors.CodeConflict, "Only a draft order can be edited", err)
	case stderrors.Is(err, domain.ErrOrderCannotDelete):
		response.ErrorLogged(c, http.StatusConflict, errors.CodeConflict, "Only a draft or pending order can be deleted", err)
	case stderrors.Is(err, domain.ErrOrderCannotSubmit):
		response.ErrorLogged(c, http.StatusConflict, errors.CodeConflict, "Order cannot be submitted in its current status", err)
	case stderrors.Is(err, domain.ErrOrderCannotApprove):
		response.ErrorLogged(c, http.StatusConflict, errors.CodeConflict, "Order cannot be approved in its current status", err)
	case stderrors.Is(err, domain.ErrOrderCannotReject):
		response.ErrorLogged(c, http.StatusConflict, errors.CodeConflict, "Order cannot be rejected in its current status", err)
	case stderrors.Is(err, domain.ErrOrderCannotPlace):
		response.ErrorLogged(c, http.StatusConflict, errors.CodeConflict, "Order cannot be placed in its current status", err)
	case stderrors.Is(err, domain.ErrOrderCannotConfirm):
		response.ErrorLogged(c, http.StatusConflict, errors.CodeConflict, "Order cannot be confirmed in its current status", err)
	case stderrors.Is(err, domain.ErrOrderCannotCancel):
		response.ErrorLogged(c, http.StatusConflict, errors.CodeConflict, "Order cannot be cancelled in its current status", err)
	case stderrors.Is(err, domain.ErrOrderCannotReceive):
		response.ErrorLogged(c, http.StatusConflict, errors.CodeConflict, "Order cannot receive stock in its current status", err)
	case stderrors.Is(err, domain.ErrOrderCannotShip):
		response.ErrorLogged(c, http.StatusConflict, errors.CodeConflict, "Order cannot ship stock in its current status", err)

	case stderrors.Is(err, domain.ErrOrderOverReceipt):
		response.ErrorLogged(c, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction,
			"Received quantity exceeds the outstanding quantity", err)
	case stderrors.Is(err, domain.ErrOrderOverShipment):
		response.ErrorLogged(c, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction,
			"Shipped quantity exceeds the outstanding quantity", err)
	case stderrors.Is(err, domain.ErrOrderNothingToPost):
		response.ErrorLogged(c, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction,
			"Nothing to post", err)
	case stderrors.Is(err, domain.ErrOrderNoItems):
		response.ErrorLogged(c, http.StatusBadRequest, errors.CodeMissingField, "An order must have at least one line", err)
	case stderrors.Is(err, domain.ErrOrderDateRange):
		response.ErrorLogged(c, http.StatusBadRequest, errors.CodeOutOfRange,
			"Expected date must not precede the order date", err)
	case stderrors.Is(err, domain.ErrOrderNegativeAmount):
		response.ErrorLogged(c, http.StatusBadRequest, errors.CodeOutOfRange, "Order amounts must not be negative", err)
	case stderrors.Is(err, domain.ErrProductInactive):
		response.ErrorLogged(c, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction,
			"An inactive product cannot be ordered", err)

	default:
		// Stock shortages surface from the ship path; product and warehouse
		// lookups surface from validation.
		if stderrors.Is(err, domain.ErrStockNegativeResult) {
			response.ErrorLogged(c, http.StatusUnprocessableEntity, errors.CodeInsufficientBalance,
				"Insufficient stock to ship this order", err)
			return
		}
		if status, code, message, ok := productErrorStatus(err); ok {
			response.ErrorLogged(c, status, code, message, err)
			return
		}
		response.InternalErrorLogged(c, "Internal server error", err)
	}
}

// parseOrderDates converts the two date strings of an order request. Both have
// already passed the `datetime=2006-01-02` binding tag.
func parseOrderDates(orderDate, expectedDate string) (time.Time, *time.Time) {
	parsedOrder, _ := time.Parse("2006-01-02", orderDate)
	if expectedDate == "" {
		return parsedOrder, nil
	}
	parsedExpected, _ := time.Parse("2006-01-02", expectedDate)
	return parsedOrder, &parsedExpected
}

// toOrderLines converts the submitted lines. Every product id has passed the
// `uuid` binding tag.
func toOrderLines(items []dto.OrderItemRequest) []service.OrderLineInput {
	lines := make([]service.OrderLineInput, len(items))
	for i, item := range items {
		lines[i] = service.OrderLineInput{
			ProductID: uuid.MustParse(item.ProductID),
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Note:      item.Note,
		}
	}
	return lines
}

// toPostLines converts the lines of a receipt or a shipment.
func toPostLines(lines []dto.PostLineRequest) []service.PostLine {
	out := make([]service.PostLine, len(lines))
	for i, line := range lines {
		out[i] = service.PostLine{
			ItemID:   uuid.MustParse(line.ItemID),
			Quantity: line.Quantity,
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Purchase orders
// ---------------------------------------------------------------------------

// ListPurchaseOrders handles GET /purchase-orders.
func (h *OrderHandler) ListPurchaseOrders(c *gin.Context) {
	filter := &service.PurchaseOrderFilter{
		CompanyID:  appctx.GetCompanyID(c),
		Status:     c.Query("status"),
		SearchTerm: c.Query("search"),
	}

	supplierID, ok := parseOptionalUUID(c, "supplier_id")
	if !ok {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid supplier ID")
		return
	}
	filter.SupplierID = supplierID

	warehouseID, ok := parseOptionalUUID(c, "warehouse_id")
	if !ok {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid warehouse ID")
		return
	}
	filter.WarehouseID = warehouseID

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
	filter.DateTo = dateTo

	if filter.Status != "" && !domain.PurchaseOrderStatus(filter.Status).IsValid() {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidInput, "Invalid order status")
		return
	}

	filter.Page, filter.PageSize = parsePageParams(c, 20)

	orders, total, err := h.service.ListPurchaseOrders(c.Request.Context(), filter)
	if err != nil {
		respondOrderError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		dto.FromPurchaseOrders(orders),
		listMeta(total, filter.Page, filter.PageSize),
	))
}

// GetPurchaseOrder handles GET /purchase-orders/:id.
func (h *OrderHandler) GetPurchaseOrder(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid order ID")
		return
	}

	order, err := h.service.GetPurchaseOrder(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromPurchaseOrder(order)))
}

// CreatePurchaseOrder handles POST /purchase-orders.
func (h *OrderHandler) CreatePurchaseOrder(c *gin.Context) {
	var req dto.CreatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", err.Error())
		return
	}

	orderDate, expectedDate := parseOrderDates(req.OrderDate, req.ExpectedDate)
	in := &service.PurchaseOrderInput{
		CompanyID:    appctx.GetCompanyID(c),
		UserID:       appctx.GetUserID(c),
		OrderDate:    orderDate,
		ExpectedDate: expectedDate,
		SupplierID:   uuid.MustParse(req.SupplierID),
		WarehouseID:  uuid.MustParse(req.WarehouseID),
		Note:         req.Note,
		Items:        toOrderLines(req.Items),
	}

	order, err := h.service.CreatePurchaseOrder(c.Request.Context(), in)
	if err != nil {
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromPurchaseOrder(order)))
}

// UpdatePurchaseOrder handles PUT /purchase-orders/:id.
func (h *OrderHandler) UpdatePurchaseOrder(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid order ID")
		return
	}

	var req dto.UpdatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", err.Error())
		return
	}

	orderDate, expectedDate := parseOrderDates(req.OrderDate, req.ExpectedDate)
	in := &service.PurchaseOrderInput{
		CompanyID:    appctx.GetCompanyID(c),
		UserID:       appctx.GetUserID(c),
		OrderDate:    orderDate,
		ExpectedDate: expectedDate,
		SupplierID:   uuid.MustParse(req.SupplierID),
		WarehouseID:  uuid.MustParse(req.WarehouseID),
		Note:         req.Note,
		Items:        toOrderLines(req.Items),
	}

	order, err := h.service.UpdatePurchaseOrder(c.Request.Context(), id, in)
	if err != nil {
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromPurchaseOrder(order)))
}

// DeletePurchaseOrder handles DELETE /purchase-orders/:id.
func (h *OrderHandler) DeletePurchaseOrder(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid order ID")
		return
	}

	if err := h.service.DeletePurchaseOrder(c.Request.Context(), appctx.GetCompanyID(c), id); err != nil {
		respondOrderError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// purchaseTransition builds a handler for one purchase-order transition.
func (h *OrderHandler) purchaseTransition(action func(ctx *gin.Context, companyID, id uuid.UUID) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid order ID")
			return
		}
		companyID := appctx.GetCompanyID(c)
		if err := action(c, companyID, id); err != nil {
			respondOrderError(c, err)
			return
		}
		// The order is re-read so the client gets the status it landed in
		// rather than having to guess it from the action it invoked.
		order, err := h.service.GetPurchaseOrder(c.Request.Context(), companyID, id)
		if err != nil {
			respondOrderError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromPurchaseOrder(order)))
	}
}

// SubmitPurchaseOrder handles POST /purchase-orders/:id/submit.
func (h *OrderHandler) SubmitPurchaseOrder(c *gin.Context) {
	h.purchaseTransition(func(ctx *gin.Context, companyID, id uuid.UUID) error {
		return h.service.SubmitPurchaseOrder(ctx.Request.Context(), companyID, id)
	})(c)
}

// ApprovePurchaseOrder handles POST /purchase-orders/:id/approve.
func (h *OrderHandler) ApprovePurchaseOrder(c *gin.Context) {
	h.purchaseTransition(func(ctx *gin.Context, companyID, id uuid.UUID) error {
		return h.service.ApprovePurchaseOrder(ctx.Request.Context(), companyID, id, appctx.GetUserID(ctx))
	})(c)
}

// RejectPurchaseOrder handles POST /purchase-orders/:id/reject.
func (h *OrderHandler) RejectPurchaseOrder(c *gin.Context) {
	h.purchaseTransition(func(ctx *gin.Context, companyID, id uuid.UUID) error {
		return h.service.RejectPurchaseOrder(ctx.Request.Context(), companyID, id)
	})(c)
}

// PlacePurchaseOrder handles POST /purchase-orders/:id/place.
func (h *OrderHandler) PlacePurchaseOrder(c *gin.Context) {
	h.purchaseTransition(func(ctx *gin.Context, companyID, id uuid.UUID) error {
		return h.service.PlacePurchaseOrder(ctx.Request.Context(), companyID, id)
	})(c)
}

// CancelPurchaseOrder handles POST /purchase-orders/:id/cancel.
func (h *OrderHandler) CancelPurchaseOrder(c *gin.Context) {
	h.purchaseTransition(func(ctx *gin.Context, companyID, id uuid.UUID) error {
		return h.service.CancelPurchaseOrder(ctx.Request.Context(), companyID, id)
	})(c)
}

// ReceivePurchaseOrder handles POST /purchase-orders/:id/receive.
func (h *OrderHandler) ReceivePurchaseOrder(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid order ID")
		return
	}

	var req dto.PostStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", err.Error())
		return
	}

	order, err := h.service.ReceivePurchaseOrder(c.Request.Context(),
		appctx.GetCompanyID(c), id, appctx.GetUserID(c), toPostLines(req.Lines))
	if err != nil {
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromPurchaseOrder(order)))
}

// GetPurchaseOrderStats handles GET /purchase-orders/stats.
func (h *OrderHandler) GetPurchaseOrderStats(c *gin.Context) {
	summary, err := h.service.GetPurchaseOrderSummary(c.Request.Context(), appctx.GetCompanyID(c))
	if err != nil {
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.PurchaseOrderStatsResponse{
		TotalCount:      summary.TotalCount,
		DraftCount:      summary.DraftCount,
		PendingCount:    summary.PendingCount,
		CompletedCount:  summary.CompletedCount,
		CancelledCount:  summary.CancelledCount,
		TotalAmount:     summary.TotalAmount,
		CompletedAmount: summary.CompletedAmount,
	}))
}

// ---------------------------------------------------------------------------
// Sales orders
// ---------------------------------------------------------------------------

// ListSalesOrders handles GET /sales-orders.
func (h *OrderHandler) ListSalesOrders(c *gin.Context) {
	filter := &service.SalesOrderFilter{
		CompanyID:  appctx.GetCompanyID(c),
		Status:     c.Query("status"),
		SearchTerm: c.Query("search"),
	}

	customerID, ok := parseOptionalUUID(c, "customer_id")
	if !ok {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid customer ID")
		return
	}
	filter.CustomerID = customerID

	warehouseID, ok := parseOptionalUUID(c, "warehouse_id")
	if !ok {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid warehouse ID")
		return
	}
	filter.WarehouseID = warehouseID

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
	filter.DateTo = dateTo

	if filter.Status != "" && !domain.SalesOrderStatus(filter.Status).IsValid() {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidInput, "Invalid order status")
		return
	}

	filter.Page, filter.PageSize = parsePageParams(c, 20)

	orders, total, err := h.service.ListSalesOrders(c.Request.Context(), filter)
	if err != nil {
		respondOrderError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		dto.FromSalesOrders(orders),
		listMeta(total, filter.Page, filter.PageSize),
	))
}

// GetSalesOrder handles GET /sales-orders/:id.
func (h *OrderHandler) GetSalesOrder(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid order ID")
		return
	}

	order, err := h.service.GetSalesOrder(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromSalesOrder(order)))
}

// CreateSalesOrder handles POST /sales-orders.
func (h *OrderHandler) CreateSalesOrder(c *gin.Context) {
	var req dto.CreateSalesOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", err.Error())
		return
	}

	orderDate, expectedDate := parseOrderDates(req.OrderDate, req.ExpectedDate)
	in := &service.SalesOrderInput{
		CompanyID:    appctx.GetCompanyID(c),
		UserID:       appctx.GetUserID(c),
		OrderDate:    orderDate,
		ExpectedDate: expectedDate,
		CustomerID:   uuid.MustParse(req.CustomerID),
		WarehouseID:  uuid.MustParse(req.WarehouseID),
		Note:         req.Note,
		Items:        toOrderLines(req.Items),
	}

	order, err := h.service.CreateSalesOrder(c.Request.Context(), in)
	if err != nil {
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromSalesOrder(order)))
}

// UpdateSalesOrder handles PUT /sales-orders/:id.
func (h *OrderHandler) UpdateSalesOrder(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid order ID")
		return
	}

	var req dto.UpdateSalesOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", err.Error())
		return
	}

	orderDate, expectedDate := parseOrderDates(req.OrderDate, req.ExpectedDate)
	in := &service.SalesOrderInput{
		CompanyID:    appctx.GetCompanyID(c),
		UserID:       appctx.GetUserID(c),
		OrderDate:    orderDate,
		ExpectedDate: expectedDate,
		CustomerID:   uuid.MustParse(req.CustomerID),
		WarehouseID:  uuid.MustParse(req.WarehouseID),
		Note:         req.Note,
		Items:        toOrderLines(req.Items),
	}

	order, err := h.service.UpdateSalesOrder(c.Request.Context(), id, in)
	if err != nil {
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromSalesOrder(order)))
}

// DeleteSalesOrder handles DELETE /sales-orders/:id.
func (h *OrderHandler) DeleteSalesOrder(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid order ID")
		return
	}

	if err := h.service.DeleteSalesOrder(c.Request.Context(), appctx.GetCompanyID(c), id); err != nil {
		respondOrderError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// salesTransition builds a handler for one sales-order transition.
func (h *OrderHandler) salesTransition(action func(ctx *gin.Context, companyID, id uuid.UUID) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid order ID")
			return
		}
		companyID := appctx.GetCompanyID(c)
		if err := action(c, companyID, id); err != nil {
			respondOrderError(c, err)
			return
		}
		order, err := h.service.GetSalesOrder(c.Request.Context(), companyID, id)
		if err != nil {
			respondOrderError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromSalesOrder(order)))
	}
}

// SubmitSalesOrder handles POST /sales-orders/:id/submit.
func (h *OrderHandler) SubmitSalesOrder(c *gin.Context) {
	h.salesTransition(func(ctx *gin.Context, companyID, id uuid.UUID) error {
		return h.service.SubmitSalesOrder(ctx.Request.Context(), companyID, id)
	})(c)
}

// ApproveSalesOrder handles POST /sales-orders/:id/approve.
func (h *OrderHandler) ApproveSalesOrder(c *gin.Context) {
	h.salesTransition(func(ctx *gin.Context, companyID, id uuid.UUID) error {
		return h.service.ApproveSalesOrder(ctx.Request.Context(), companyID, id, appctx.GetUserID(ctx))
	})(c)
}

// RejectSalesOrder handles POST /sales-orders/:id/reject.
func (h *OrderHandler) RejectSalesOrder(c *gin.Context) {
	h.salesTransition(func(ctx *gin.Context, companyID, id uuid.UUID) error {
		return h.service.RejectSalesOrder(ctx.Request.Context(), companyID, id)
	})(c)
}

// ConfirmSalesOrder handles POST /sales-orders/:id/confirm.
func (h *OrderHandler) ConfirmSalesOrder(c *gin.Context) {
	h.salesTransition(func(ctx *gin.Context, companyID, id uuid.UUID) error {
		return h.service.ConfirmSalesOrder(ctx.Request.Context(), companyID, id)
	})(c)
}

// CancelSalesOrder handles POST /sales-orders/:id/cancel.
func (h *OrderHandler) CancelSalesOrder(c *gin.Context) {
	h.salesTransition(func(ctx *gin.Context, companyID, id uuid.UUID) error {
		return h.service.CancelSalesOrder(ctx.Request.Context(), companyID, id)
	})(c)
}

// ShipSalesOrder handles POST /sales-orders/:id/ship.
func (h *OrderHandler) ShipSalesOrder(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid order ID")
		return
	}

	var req dto.PostStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", err.Error())
		return
	}

	order, err := h.service.ShipSalesOrder(c.Request.Context(),
		appctx.GetCompanyID(c), id, appctx.GetUserID(c), toPostLines(req.Lines))
	if err != nil {
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromSalesOrder(order)))
}

// GetSalesOrderStats handles GET /sales-orders/stats.
func (h *OrderHandler) GetSalesOrderStats(c *gin.Context) {
	summary, err := h.service.GetSalesOrderSummary(c.Request.Context(), appctx.GetCompanyID(c))
	if err != nil {
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.SalesOrderStatsResponse{
		TotalCount:       summary.TotalCount,
		DraftCount:       summary.DraftCount,
		PendingShipCount: summary.PendingShipCount,
		CompletedCount:   summary.CompletedCount,
		CancelledCount:   summary.CancelledCount,
		TotalAmount:      summary.TotalAmount,
		CompletedAmount:  summary.CompletedAmount,
	}))
}
