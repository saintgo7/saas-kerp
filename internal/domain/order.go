package domain

import (
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
)

// Order errors.
var (
	ErrOrderNotFound       = errors.New("order not found")
	ErrOrderNumberExists   = errors.New("order number already exists")
	ErrOrderNoItems        = errors.New("order must have at least one line")
	ErrOrderInvalidStatus  = errors.New("invalid order status")
	ErrOrderCannotEdit     = errors.New("order cannot be edited in its current status")
	ErrOrderCannotDelete   = errors.New("order cannot be deleted in its current status")
	ErrOrderCannotSubmit   = errors.New("order cannot be submitted in its current status")
	ErrOrderCannotApprove  = errors.New("order cannot be approved in its current status")
	ErrOrderCannotReject   = errors.New("order cannot be rejected in its current status")
	ErrOrderCannotPlace    = errors.New("order cannot be placed in its current status")
	ErrOrderCannotConfirm  = errors.New("order cannot be confirmed in its current status")
	ErrOrderCannotCancel   = errors.New("order cannot be cancelled in its current status")
	ErrOrderCannotReceive  = errors.New("order cannot receive stock in its current status")
	ErrOrderCannotShip     = errors.New("order cannot ship stock in its current status")
	ErrOrderLineNotFound   = errors.New("order line not found")
	ErrOrderOverReceipt    = errors.New("received quantity exceeds the ordered quantity")
	ErrOrderOverShipment   = errors.New("shipped quantity exceeds the ordered quantity")
	ErrOrderNothingToPost  = errors.New("no quantity to post")
	ErrOrderDateRange      = errors.New("expected date must not precede the order date")
	ErrOrderNegativeAmount = errors.New("order amounts must not be negative")
)

// VATRate is the Korean value-added tax rate applied to order amounts.
const VATRate = 0.1

// ---------------------------------------------------------------------------
// Purchase order status machine
// ---------------------------------------------------------------------------

// PurchaseOrderStatus is the lifecycle state of a purchase order. The values
// are PurchaseOrderStatus in web/src/types/inventory.ts and PURCHASE_ORDER_STATUS
// in web/src/constants; chk_purchase_orders_status repeats the set in SQL.
type PurchaseOrderStatus string

const (
	POStatusDraft     PurchaseOrderStatus = "draft"
	POStatusPending   PurchaseOrderStatus = "pending"
	POStatusApproved  PurchaseOrderStatus = "approved"
	POStatusOrdered   PurchaseOrderStatus = "ordered"
	POStatusPartial   PurchaseOrderStatus = "partial"
	POStatusCompleted PurchaseOrderStatus = "completed"
	POStatusCancelled PurchaseOrderStatus = "cancelled"
)

// The permitted transitions, written out once:
//
//	draft     -> pending (submit) | cancelled
//	pending   -> approved (approve) | draft (reject) | cancelled
//	approved  -> ordered (place) | cancelled
//	ordered   -> partial | completed (receive) | cancelled
//	partial   -> partial | completed (receive)
//	completed -> (terminal)
//	cancelled -> (terminal)
//
// partial has no cancel edge on purpose: stock has already been received
// against the order, and cancelling would leave those movements pointing at an
// order that claims nothing was ever ordered. Reverse the receipt with a return
// movement first.

// IsValid reports whether s is one of the seven known statuses.
func (s PurchaseOrderStatus) IsValid() bool {
	switch s {
	case POStatusDraft, POStatusPending, POStatusApproved, POStatusOrdered,
		POStatusPartial, POStatusCompleted, POStatusCancelled:
		return true
	}
	return false
}

// CanEdit reports whether the header and lines may still be changed.
func (s PurchaseOrderStatus) CanEdit() bool { return s == POStatusDraft }

// CanDelete reports whether the order may be removed outright.
func (s PurchaseOrderStatus) CanDelete() bool {
	return s == POStatusDraft || s == POStatusPending
}

// CanSubmit reports whether the order may be sent for approval.
func (s PurchaseOrderStatus) CanSubmit() bool { return s == POStatusDraft }

// CanApprove reports whether the order may be approved.
func (s PurchaseOrderStatus) CanApprove() bool { return s == POStatusPending }

// CanReject reports whether the order may be sent back to draft.
func (s PurchaseOrderStatus) CanReject() bool { return s == POStatusPending }

// CanPlace reports whether the approved order may be placed with the supplier.
func (s PurchaseOrderStatus) CanPlace() bool { return s == POStatusApproved }

// CanReceive reports whether goods may be booked in against the order.
func (s PurchaseOrderStatus) CanReceive() bool {
	return s == POStatusOrdered || s == POStatusPartial
}

// CanCancel reports whether the order may be cancelled.
func (s PurchaseOrderStatus) CanCancel() bool {
	switch s {
	case POStatusDraft, POStatusPending, POStatusApproved, POStatusOrdered:
		return true
	}
	return false
}

// IsTerminal reports whether no further transition is possible.
func (s PurchaseOrderStatus) IsTerminal() bool {
	return s == POStatusCompleted || s == POStatusCancelled
}

// ---------------------------------------------------------------------------
// Sales order status machine
// ---------------------------------------------------------------------------

// SalesOrderStatus is the lifecycle state of a sales order.
//
// It carries both `approved` and `confirmed`, which the purchase side does not:
// approval is the internal sign-off, confirmation is the customer's. The UI
// only offers shipping from confirmed or partial, so approved is a real state
// with a real successor rather than a synonym.
type SalesOrderStatus string

const (
	SOStatusDraft     SalesOrderStatus = "draft"
	SOStatusPending   SalesOrderStatus = "pending"
	SOStatusApproved  SalesOrderStatus = "approved"
	SOStatusConfirmed SalesOrderStatus = "confirmed"
	SOStatusPartial   SalesOrderStatus = "partial"
	SOStatusCompleted SalesOrderStatus = "completed"
	SOStatusCancelled SalesOrderStatus = "cancelled"
)

// Permitted transitions:
//
//	draft     -> pending (submit) | cancelled
//	pending   -> approved (approve) | draft (reject) | cancelled
//	approved  -> confirmed (confirm) | cancelled
//	confirmed -> partial | completed (ship) | cancelled
//	partial   -> partial | completed (ship)
//	completed -> (terminal)
//	cancelled -> (terminal)

// IsValid reports whether s is one of the seven known statuses.
func (s SalesOrderStatus) IsValid() bool {
	switch s {
	case SOStatusDraft, SOStatusPending, SOStatusApproved, SOStatusConfirmed,
		SOStatusPartial, SOStatusCompleted, SOStatusCancelled:
		return true
	}
	return false
}

// CanEdit reports whether the header and lines may still be changed.
func (s SalesOrderStatus) CanEdit() bool { return s == SOStatusDraft }

// CanDelete reports whether the order may be removed outright.
func (s SalesOrderStatus) CanDelete() bool {
	return s == SOStatusDraft || s == SOStatusPending
}

// CanSubmit reports whether the order may be sent for approval.
func (s SalesOrderStatus) CanSubmit() bool { return s == SOStatusDraft }

// CanApprove reports whether the order may be approved.
func (s SalesOrderStatus) CanApprove() bool { return s == SOStatusPending }

// CanReject reports whether the order may be sent back to draft.
func (s SalesOrderStatus) CanReject() bool { return s == SOStatusPending }

// CanConfirm reports whether the approved order may be confirmed.
func (s SalesOrderStatus) CanConfirm() bool { return s == SOStatusApproved }

// CanShip reports whether goods may be booked out against the order.
func (s SalesOrderStatus) CanShip() bool {
	return s == SOStatusConfirmed || s == SOStatusPartial
}

// CanCancel reports whether the order may be cancelled.
func (s SalesOrderStatus) CanCancel() bool {
	switch s {
	case SOStatusDraft, SOStatusPending, SOStatusApproved, SOStatusConfirmed:
		return true
	}
	return false
}

// IsTerminal reports whether no further transition is possible.
func (s SalesOrderStatus) IsTerminal() bool {
	return s == SOStatusCompleted || s == SOStatusCancelled
}

// ---------------------------------------------------------------------------
// Purchase order
// ---------------------------------------------------------------------------

// PurchaseOrder is an order placed with a supplier (발주서).
type PurchaseOrder struct {
	TenantModel

	OrderNumber  string     `gorm:"type:varchar(30);not null" json:"order_number"`
	OrderDate    time.Time  `gorm:"type:date;not null" json:"order_date"`
	ExpectedDate *time.Time `gorm:"type:date" json:"expected_date,omitempty"`

	SupplierID  uuid.UUID  `gorm:"type:uuid;not null" json:"supplier_id"`
	Supplier    *Partner   `gorm:"foreignKey:SupplierID" json:"supplier,omitempty"`
	WarehouseID uuid.UUID  `gorm:"type:uuid;not null" json:"warehouse_id"`
	Warehouse   *Warehouse `gorm:"foreignKey:WarehouseID" json:"warehouse,omitempty"`

	Items []PurchaseOrderItem `gorm:"foreignKey:PurchaseOrderID" json:"items,omitempty"`

	// Whole won. chk_purchase_orders_amounts makes grand_total = total + tax an
	// exact identity, which only holds because these are integers.
	TotalAmount int64 `gorm:"type:numeric(18,0);not null;default:0" json:"total_amount"`
	TaxAmount   int64 `gorm:"type:numeric(18,0);not null;default:0" json:"tax_amount"`
	GrandTotal  int64 `gorm:"type:numeric(18,0);not null;default:0" json:"grand_total"`

	Status PurchaseOrderStatus `gorm:"type:varchar(20);not null;default:'draft'" json:"status"`
	Note   string              `gorm:"type:text" json:"note,omitempty"`

	CreatedBy  uuid.UUID  `gorm:"type:uuid;not null" json:"created_by"`
	ApprovedBy *uuid.UUID `gorm:"type:uuid" json:"approved_by,omitempty"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`

	// Display names, filled in by the repository rather than by a GORM
	// association. See domain.StockMovement.CreatedByName for why an
	// association to User cannot be used here.
	CreatedByName  string `gorm:"-" json:"created_by_name,omitempty"`
	ApprovedByName string `gorm:"-" json:"approved_by_name,omitempty"`
}

// TableName specifies the table name for GORM.
func (PurchaseOrder) TableName() string {
	return "purchase_orders"
}

// PurchaseOrderItem is one line of a purchase order.
type PurchaseOrderItem struct {
	TenantModel

	PurchaseOrderID uuid.UUID `gorm:"type:uuid;not null" json:"purchase_order_id"`
	LineNo          int       `gorm:"not null" json:"line_no"`

	ProductID uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	Product   *Product  `gorm:"foreignKey:ProductID" json:"product,omitempty"`

	Quantity  float64 `gorm:"type:numeric(18,3);not null" json:"quantity"`
	UnitPrice float64 `gorm:"type:numeric(18,2);not null" json:"unit_price"`
	Amount    int64   `gorm:"type:numeric(18,0);not null" json:"amount"`
	TaxAmount int64   `gorm:"type:numeric(18,0);not null;default:0" json:"tax_amount"`

	ReceivedQuantity float64 `gorm:"type:numeric(18,3);not null;default:0" json:"received_quantity"`

	Note string `gorm:"type:text" json:"note,omitempty"`
}

// TableName specifies the table name for GORM.
func (PurchaseOrderItem) TableName() string {
	return "purchase_order_items"
}

// OutstandingQuantity is how much of the line is still to be received.
func (i *PurchaseOrderItem) OutstandingQuantity() float64 {
	return i.Quantity - i.ReceivedQuantity
}

// ---------------------------------------------------------------------------
// Sales order
// ---------------------------------------------------------------------------

// SalesOrder is an order taken from a customer (수주).
type SalesOrder struct {
	TenantModel

	OrderNumber  string     `gorm:"type:varchar(30);not null" json:"order_number"`
	OrderDate    time.Time  `gorm:"type:date;not null" json:"order_date"`
	ExpectedDate *time.Time `gorm:"type:date" json:"expected_date,omitempty"`

	CustomerID  uuid.UUID  `gorm:"type:uuid;not null" json:"customer_id"`
	Customer    *Partner   `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	WarehouseID uuid.UUID  `gorm:"type:uuid;not null" json:"warehouse_id"`
	Warehouse   *Warehouse `gorm:"foreignKey:WarehouseID" json:"warehouse,omitempty"`

	Items []SalesOrderItem `gorm:"foreignKey:SalesOrderID" json:"items,omitempty"`

	TotalAmount int64 `gorm:"type:numeric(18,0);not null;default:0" json:"total_amount"`
	TaxAmount   int64 `gorm:"type:numeric(18,0);not null;default:0" json:"tax_amount"`
	GrandTotal  int64 `gorm:"type:numeric(18,0);not null;default:0" json:"grand_total"`

	Status SalesOrderStatus `gorm:"type:varchar(20);not null;default:'draft'" json:"status"`
	Note   string           `gorm:"type:text" json:"note,omitempty"`

	CreatedBy  uuid.UUID  `gorm:"type:uuid;not null" json:"created_by"`
	ApprovedBy *uuid.UUID `gorm:"type:uuid" json:"approved_by,omitempty"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`

	// Display names, filled in by the repository rather than by a GORM
	// association. See domain.StockMovement.CreatedByName for why an
	// association to User cannot be used here.
	CreatedByName  string `gorm:"-" json:"created_by_name,omitempty"`
	ApprovedByName string `gorm:"-" json:"approved_by_name,omitempty"`
}

// TableName specifies the table name for GORM.
func (SalesOrder) TableName() string {
	return "sales_orders"
}

// SalesOrderItem is one line of a sales order.
type SalesOrderItem struct {
	TenantModel

	SalesOrderID uuid.UUID `gorm:"type:uuid;not null" json:"sales_order_id"`
	LineNo       int       `gorm:"not null" json:"line_no"`

	ProductID uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	Product   *Product  `gorm:"foreignKey:ProductID" json:"product,omitempty"`

	Quantity  float64 `gorm:"type:numeric(18,3);not null" json:"quantity"`
	UnitPrice float64 `gorm:"type:numeric(18,2);not null" json:"unit_price"`
	Amount    int64   `gorm:"type:numeric(18,0);not null" json:"amount"`
	TaxAmount int64   `gorm:"type:numeric(18,0);not null;default:0" json:"tax_amount"`

	ShippedQuantity float64 `gorm:"type:numeric(18,3);not null;default:0" json:"shipped_quantity"`

	Note string `gorm:"type:text" json:"note,omitempty"`
}

// TableName specifies the table name for GORM.
func (SalesOrderItem) TableName() string {
	return "sales_order_items"
}

// OutstandingQuantity is how much of the line is still to be shipped.
func (i *SalesOrderItem) OutstandingQuantity() float64 {
	return i.Quantity - i.ShippedQuantity
}

// ---------------------------------------------------------------------------
// Amount arithmetic
// ---------------------------------------------------------------------------

// LineAmount is the booked amount of one order line: quantity x unit price,
// rounded once to whole won.
//
// Rounding happens here and nowhere else. A line amount that stayed fractional
// would make the order total fractional too, and chk_*_orders_amounts would
// then be comparing values that cannot be equal.
func LineAmount(quantity, unitPrice float64) int64 {
	return int64(math.Round(quantity * unitPrice))
}

// LineTaxAmount is the VAT on one line, for display. The order's own tax is
// NOT the sum of these; see OrderTaxAmount.
func LineTaxAmount(amount int64) int64 {
	return int64(math.Round(float64(amount) * VATRate))
}

// OrderTaxAmount is the VAT on the order total.
//
// It rounds the total once rather than summing the per-line roundings, which
// is what web/src/pages/inventory/{Purchase,Sales}OrderPage.tsx do when they
// preview an order (Math.round(totalAmount * 0.1)). Summing per-line tax
// instead would differ from the preview by a won or two on orders with many
// lines, and the first person to notice would be an accountant reconciling the
// order against the tax invoice.
func OrderTaxAmount(totalAmount int64) int64 {
	return int64(math.Round(float64(totalAmount) * VATRate))
}
