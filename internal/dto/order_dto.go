package dto

import (
	"github.com/saintgo7/saas-kerp/internal/domain"
)

// ---------------------------------------------------------------------------
// Requests
// ---------------------------------------------------------------------------

// OrderItemRequest is one submitted order line.
//
// There is no amount field, deliberately. The line's amount and tax are derived
// on the server from the quantity and the unit price; accepting them from the
// client would let a caller book an order total that has nothing to do with
// what it ordered.
type OrderItemRequest struct {
	ProductID string  `json:"product_id" binding:"required,uuid"`
	Quantity  float64 `json:"quantity" binding:"required,gt=0"`
	UnitPrice float64 `json:"unit_price" binding:"gte=0"`
	Note      string  `json:"note,omitempty" binding:"max=500"`
}

// CreatePurchaseOrderRequest is the body of a purchase order create.
//
// order_number is absent on purpose: it is reserved by the server
// (order_sequences) so two concurrent creates cannot claim the same one.
type CreatePurchaseOrderRequest struct {
	OrderDate    string             `json:"order_date" binding:"required,datetime=2006-01-02"`
	ExpectedDate string             `json:"expected_date,omitempty" binding:"omitempty,datetime=2006-01-02"`
	SupplierID   string             `json:"supplier_id" binding:"required,uuid"`
	WarehouseID  string             `json:"warehouse_id" binding:"required,uuid"`
	Note         string             `json:"note,omitempty" binding:"max=1000"`
	Items        []OrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

// UpdatePurchaseOrderRequest is the body of a purchase order update.
type UpdatePurchaseOrderRequest struct {
	OrderDate    string             `json:"order_date" binding:"required,datetime=2006-01-02"`
	ExpectedDate string             `json:"expected_date,omitempty" binding:"omitempty,datetime=2006-01-02"`
	SupplierID   string             `json:"supplier_id" binding:"required,uuid"`
	WarehouseID  string             `json:"warehouse_id" binding:"required,uuid"`
	Note         string             `json:"note,omitempty" binding:"max=1000"`
	Items        []OrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

// CreateSalesOrderRequest is the body of a sales order create.
type CreateSalesOrderRequest struct {
	OrderDate    string             `json:"order_date" binding:"required,datetime=2006-01-02"`
	ExpectedDate string             `json:"expected_date,omitempty" binding:"omitempty,datetime=2006-01-02"`
	CustomerID   string             `json:"customer_id" binding:"required,uuid"`
	WarehouseID  string             `json:"warehouse_id" binding:"required,uuid"`
	Note         string             `json:"note,omitempty" binding:"max=1000"`
	Items        []OrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

// UpdateSalesOrderRequest is the body of a sales order update.
type UpdateSalesOrderRequest struct {
	OrderDate    string             `json:"order_date" binding:"required,datetime=2006-01-02"`
	ExpectedDate string             `json:"expected_date,omitempty" binding:"omitempty,datetime=2006-01-02"`
	CustomerID   string             `json:"customer_id" binding:"required,uuid"`
	WarehouseID  string             `json:"warehouse_id" binding:"required,uuid"`
	Note         string             `json:"note,omitempty" binding:"max=1000"`
	Items        []OrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

// PostLineRequest names one order line and how much of it to post.
type PostLineRequest struct {
	ItemID   string  `json:"item_id" binding:"required,uuid"`
	Quantity float64 `json:"quantity" binding:"required,gt=0"`
}

// PostStockRequest is the body of a receipt or a shipment.
type PostStockRequest struct {
	Lines []PostLineRequest `json:"lines" binding:"required,min=1,dive"`
}

// ---------------------------------------------------------------------------
// Responses
// ---------------------------------------------------------------------------

// PurchaseOrderItemResponse is one purchase order line in API responses.
type PurchaseOrderItemResponse struct {
	ID               string           `json:"id"`
	PurchaseOrderID  string           `json:"purchase_order_id"`
	LineNo           int              `json:"line_no"`
	ProductID        string           `json:"product_id"`
	Product          *ProductResponse `json:"product,omitempty"`
	Quantity         float64          `json:"quantity"`
	UnitPrice        float64          `json:"unit_price"`
	Amount           int64            `json:"amount"`
	TaxAmount        int64            `json:"tax_amount"`
	ReceivedQuantity float64          `json:"received_quantity"`
	// OutstandingQuantity is quantity - received_quantity, which the receive
	// dialog computes on every row. Sent so the dialog and the server cannot
	// disagree about how much is still open.
	OutstandingQuantity float64 `json:"outstanding_quantity"`
	Note                string  `json:"note,omitempty"`
}

// FromPurchaseOrderItem converts one line.
func FromPurchaseOrderItem(i *domain.PurchaseOrderItem) PurchaseOrderItemResponse {
	resp := PurchaseOrderItemResponse{
		ID:                  i.ID.String(),
		PurchaseOrderID:     i.PurchaseOrderID.String(),
		LineNo:              i.LineNo,
		ProductID:           i.ProductID.String(),
		Quantity:            i.Quantity,
		UnitPrice:           i.UnitPrice,
		Amount:              i.Amount,
		TaxAmount:           i.TaxAmount,
		ReceivedQuantity:    i.ReceivedQuantity,
		OutstandingQuantity: i.OutstandingQuantity(),
		Note:                i.Note,
	}
	if i.Product != nil {
		product := FromProduct(i.Product)
		resp.Product = &product
	}
	return resp
}

// PurchaseOrderResponse is a purchase order in API responses.
type PurchaseOrderResponse struct {
	ID        string `json:"id"`
	CompanyID string `json:"company_id"`

	OrderNumber  string `json:"order_number"`
	OrderDate    string `json:"order_date"`
	ExpectedDate string `json:"expected_date,omitempty"`

	SupplierID  string             `json:"supplier_id"`
	Supplier    *PartnerResponse   `json:"supplier,omitempty"`
	WarehouseID string             `json:"warehouse_id"`
	Warehouse   *WarehouseResponse `json:"warehouse,omitempty"`

	Items []PurchaseOrderItemResponse `json:"items"`

	TotalAmount int64 `json:"total_amount"`
	TaxAmount   int64 `json:"tax_amount"`
	GrandTotal  int64 `json:"grand_total"`

	Status string `json:"status"`
	Note   string `json:"note,omitempty"`

	// See StockMovementResponse.CreatedBy for why the name travels alongside
	// the id.
	CreatedBy      string `json:"created_by"`
	CreatedByName  string `json:"created_by_name,omitempty"`
	ApprovedBy     string `json:"approved_by,omitempty"`
	ApprovedByName string `json:"approved_by_name,omitempty"`
	ApprovedAt     string `json:"approved_at,omitempty"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// FromPurchaseOrder converts a domain purchase order to its response form.
func FromPurchaseOrder(o *domain.PurchaseOrder) PurchaseOrderResponse {
	resp := PurchaseOrderResponse{
		ID:          o.ID.String(),
		CompanyID:   o.CompanyID.String(),
		OrderNumber: o.OrderNumber,
		OrderDate:   formatDate(o.OrderDate),
		SupplierID:  o.SupplierID.String(),
		WarehouseID: o.WarehouseID.String(),
		TotalAmount: o.TotalAmount,
		TaxAmount:   o.TaxAmount,
		GrandTotal:  o.GrandTotal,
		Status:      string(o.Status),
		Note:        o.Note,
		CreatedBy:   o.CreatedBy.String(),
		CreatedAt:   formatTime(o.CreatedAt),
		UpdatedAt:   formatTime(o.UpdatedAt),
	}

	if o.ExpectedDate != nil {
		resp.ExpectedDate = formatDate(*o.ExpectedDate)
	}
	if o.ApprovedBy != nil {
		resp.ApprovedBy = o.ApprovedBy.String()
	}
	if o.ApprovedAt != nil {
		resp.ApprovedAt = formatTime(*o.ApprovedAt)
	}
	resp.CreatedByName = o.CreatedByName
	resp.ApprovedByName = o.ApprovedByName
	if o.Supplier != nil {
		supplier := FromPartner(o.Supplier)
		resp.Supplier = &supplier
	}
	if o.Warehouse != nil {
		warehouse := FromWarehouse(o.Warehouse)
		resp.Warehouse = &warehouse
	}

	// Never nil: the list screen iterates items unconditionally, and a null
	// would be a TypeError rather than an empty table.
	resp.Items = make([]PurchaseOrderItemResponse, len(o.Items))
	for i := range o.Items {
		resp.Items[i] = FromPurchaseOrderItem(&o.Items[i])
	}
	return resp
}

// FromPurchaseOrders converts a slice of purchase orders.
func FromPurchaseOrders(orders []domain.PurchaseOrder) []PurchaseOrderResponse {
	out := make([]PurchaseOrderResponse, len(orders))
	for i := range orders {
		out[i] = FromPurchaseOrder(&orders[i])
	}
	return out
}

// SalesOrderItemResponse is one sales order line in API responses.
type SalesOrderItemResponse struct {
	ID           string           `json:"id"`
	SalesOrderID string           `json:"sales_order_id"`
	LineNo       int              `json:"line_no"`
	ProductID    string           `json:"product_id"`
	Product      *ProductResponse `json:"product,omitempty"`
	Quantity     float64          `json:"quantity"`
	UnitPrice    float64          `json:"unit_price"`
	Amount       int64            `json:"amount"`
	TaxAmount    int64            `json:"tax_amount"`

	ShippedQuantity     float64 `json:"shipped_quantity"`
	OutstandingQuantity float64 `json:"outstanding_quantity"`
	Note                string  `json:"note,omitempty"`
}

// FromSalesOrderItem converts one line.
func FromSalesOrderItem(i *domain.SalesOrderItem) SalesOrderItemResponse {
	resp := SalesOrderItemResponse{
		ID:                  i.ID.String(),
		SalesOrderID:        i.SalesOrderID.String(),
		LineNo:              i.LineNo,
		ProductID:           i.ProductID.String(),
		Quantity:            i.Quantity,
		UnitPrice:           i.UnitPrice,
		Amount:              i.Amount,
		TaxAmount:           i.TaxAmount,
		ShippedQuantity:     i.ShippedQuantity,
		OutstandingQuantity: i.OutstandingQuantity(),
		Note:                i.Note,
	}
	if i.Product != nil {
		product := FromProduct(i.Product)
		resp.Product = &product
	}
	return resp
}

// SalesOrderResponse is a sales order in API responses.
type SalesOrderResponse struct {
	ID        string `json:"id"`
	CompanyID string `json:"company_id"`

	OrderNumber  string `json:"order_number"`
	OrderDate    string `json:"order_date"`
	ExpectedDate string `json:"expected_date,omitempty"`

	CustomerID  string             `json:"customer_id"`
	Customer    *PartnerResponse   `json:"customer,omitempty"`
	WarehouseID string             `json:"warehouse_id"`
	Warehouse   *WarehouseResponse `json:"warehouse,omitempty"`

	Items []SalesOrderItemResponse `json:"items"`

	TotalAmount int64 `json:"total_amount"`
	TaxAmount   int64 `json:"tax_amount"`
	GrandTotal  int64 `json:"grand_total"`

	Status string `json:"status"`
	Note   string `json:"note,omitempty"`

	CreatedBy      string `json:"created_by"`
	CreatedByName  string `json:"created_by_name,omitempty"`
	ApprovedBy     string `json:"approved_by,omitempty"`
	ApprovedByName string `json:"approved_by_name,omitempty"`
	ApprovedAt     string `json:"approved_at,omitempty"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// FromSalesOrder converts a domain sales order to its response form.
func FromSalesOrder(o *domain.SalesOrder) SalesOrderResponse {
	resp := SalesOrderResponse{
		ID:          o.ID.String(),
		CompanyID:   o.CompanyID.String(),
		OrderNumber: o.OrderNumber,
		OrderDate:   formatDate(o.OrderDate),
		CustomerID:  o.CustomerID.String(),
		WarehouseID: o.WarehouseID.String(),
		TotalAmount: o.TotalAmount,
		TaxAmount:   o.TaxAmount,
		GrandTotal:  o.GrandTotal,
		Status:      string(o.Status),
		Note:        o.Note,
		CreatedBy:   o.CreatedBy.String(),
		CreatedAt:   formatTime(o.CreatedAt),
		UpdatedAt:   formatTime(o.UpdatedAt),
	}

	if o.ExpectedDate != nil {
		resp.ExpectedDate = formatDate(*o.ExpectedDate)
	}
	if o.ApprovedBy != nil {
		resp.ApprovedBy = o.ApprovedBy.String()
	}
	if o.ApprovedAt != nil {
		resp.ApprovedAt = formatTime(*o.ApprovedAt)
	}
	resp.CreatedByName = o.CreatedByName
	resp.ApprovedByName = o.ApprovedByName
	if o.Customer != nil {
		customer := FromPartner(o.Customer)
		resp.Customer = &customer
	}
	if o.Warehouse != nil {
		warehouse := FromWarehouse(o.Warehouse)
		resp.Warehouse = &warehouse
	}

	resp.Items = make([]SalesOrderItemResponse, len(o.Items))
	for i := range o.Items {
		resp.Items[i] = FromSalesOrderItem(&o.Items[i])
	}
	return resp
}

// FromSalesOrders converts a slice of sales orders.
func FromSalesOrders(orders []domain.SalesOrder) []SalesOrderResponse {
	out := make([]SalesOrderResponse, len(orders))
	for i := range orders {
		out[i] = FromSalesOrder(&orders[i])
	}
	return out
}

// PurchaseOrderStatsResponse is the purchase dashboard card.
type PurchaseOrderStatsResponse struct {
	TotalCount     int64 `json:"total_count"`
	DraftCount     int64 `json:"draft_count"`
	PendingCount   int64 `json:"pending_count"`
	CompletedCount int64 `json:"completed_count"`
	CancelledCount int64 `json:"cancelled_count"`
	// TotalAmount excludes cancelled orders, on both this card and the sales
	// one. The mock frontend includes them here and excludes them there.
	TotalAmount     int64 `json:"total_amount"`
	CompletedAmount int64 `json:"completed_amount"`
}

// SalesOrderStatsResponse is the sales dashboard card.
type SalesOrderStatsResponse struct {
	TotalCount       int64 `json:"total_count"`
	DraftCount       int64 `json:"draft_count"`
	PendingShipCount int64 `json:"pending_ship_count"`
	CompletedCount   int64 `json:"completed_count"`
	CancelledCount   int64 `json:"cancelled_count"`
	TotalAmount      int64 `json:"total_amount"`
	CompletedAmount  int64 `json:"completed_amount"`
}
