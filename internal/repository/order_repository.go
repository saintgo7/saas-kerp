package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// PurchaseOrderFilter defines filter criteria for listing purchase orders.
type PurchaseOrderFilter struct {
	CompanyID   uuid.UUID
	Status      string
	SupplierID  *uuid.UUID
	WarehouseID *uuid.UUID
	DateFrom    *time.Time
	DateTo      *time.Time
	SearchTerm  string // matched against the order number and the supplier name
	Page        int
	PageSize    int
}

// SalesOrderFilter defines filter criteria for listing sales orders.
type SalesOrderFilter struct {
	CompanyID   uuid.UUID
	Status      string
	CustomerID  *uuid.UUID
	WarehouseID *uuid.UUID
	DateFrom    *time.Time
	DateTo      *time.Time
	SearchTerm  string // matched against the order number and the customer name
	Page        int
	PageSize    int
}

// PostLine is one line of a receipt or a shipment: how much of which order line
// is being booked into or out of stock.
type PostLine struct {
	ItemID   uuid.UUID
	Quantity float64
}

// PurchaseOrderSummary is the aggregate behind the purchase dashboard cards.
type PurchaseOrderSummary struct {
	TotalCount     int64 `json:"total_count"`
	DraftCount     int64 `json:"draft_count"`
	PendingCount   int64 `json:"pending_count"`
	CompletedCount int64 `json:"completed_count"`
	CancelledCount int64 `json:"cancelled_count"`
	// TotalAmount excludes cancelled orders. The mock frontend includes them on
	// the purchase card and excludes them on the sales card; a cancelled order
	// is not committed spend, so both are computed the same way here and the
	// divergence is recorded in the handover notes.
	TotalAmount     int64 `json:"total_amount"`
	CompletedAmount int64 `json:"completed_amount"`
}

// SalesOrderSummary is the aggregate behind the sales dashboard cards.
type SalesOrderSummary struct {
	TotalCount       int64 `json:"total_count"`
	DraftCount       int64 `json:"draft_count"`
	PendingShipCount int64 `json:"pending_ship_count"`
	CompletedCount   int64 `json:"completed_count"`
	CancelledCount   int64 `json:"cancelled_count"`
	TotalAmount      int64 `json:"total_amount"`
	CompletedAmount  int64 `json:"completed_amount"`
}

// OrderRepository is data access for purchase and sales orders.
//
// Every state transition here is a compare-and-swap: the UPDATE carries the
// status it expects to find, and a zero RowsAffected means somebody already
// made the transition. That is what stops a double-clicked Approve from
// approving twice, and it costs one predicate rather than a lock.
type OrderRepository interface {
	// --- purchase orders ---
	CreatePurchaseOrder(ctx context.Context, order *domain.PurchaseOrder) error
	GetPurchaseOrder(ctx context.Context, companyID, id uuid.UUID) (*domain.PurchaseOrder, error)
	ListPurchaseOrders(ctx context.Context, filter *PurchaseOrderFilter) ([]domain.PurchaseOrder, int64, error)
	UpdatePurchaseOrder(ctx context.Context, order *domain.PurchaseOrder) error
	ReplacePurchaseOrderItems(ctx context.Context, companyID, orderID uuid.UUID, items []domain.PurchaseOrderItem) error
	DeletePurchaseOrder(ctx context.Context, companyID, id uuid.UUID, allowed []domain.PurchaseOrderStatus) (bool, error)

	// TransitionPurchaseOrder moves the order from one of `from` to `to`.
	// It reports false when no row matched, which means the order was already
	// past that point.
	TransitionPurchaseOrder(ctx context.Context, companyID, id uuid.UUID, from []domain.PurchaseOrderStatus, to domain.PurchaseOrderStatus, approvedBy *uuid.UUID) (bool, error)

	// LockPurchaseOrder reads the order with its lines under SELECT ... FOR
	// UPDATE, so that a receipt sees a stable set of outstanding quantities.
	LockPurchaseOrder(ctx context.Context, companyID, id uuid.UUID) (*domain.PurchaseOrder, error)

	// AddReceivedQuantity raises one line's received quantity, refusing to take
	// it past the ordered quantity. Reports false when the increment would
	// over-receive, which is what a replayed request looks like.
	AddReceivedQuantity(ctx context.Context, companyID, itemID uuid.UUID, quantity float64) (bool, error)

	PurchaseOrderSummary(ctx context.Context, companyID uuid.UUID) (*PurchaseOrderSummary, error)

	// --- sales orders ---
	CreateSalesOrder(ctx context.Context, order *domain.SalesOrder) error
	GetSalesOrder(ctx context.Context, companyID, id uuid.UUID) (*domain.SalesOrder, error)
	ListSalesOrders(ctx context.Context, filter *SalesOrderFilter) ([]domain.SalesOrder, int64, error)
	UpdateSalesOrder(ctx context.Context, order *domain.SalesOrder) error
	ReplaceSalesOrderItems(ctx context.Context, companyID, orderID uuid.UUID, items []domain.SalesOrderItem) error
	DeleteSalesOrder(ctx context.Context, companyID, id uuid.UUID, allowed []domain.SalesOrderStatus) (bool, error)

	TransitionSalesOrder(ctx context.Context, companyID, id uuid.UUID, from []domain.SalesOrderStatus, to domain.SalesOrderStatus, approvedBy *uuid.UUID) (bool, error)
	LockSalesOrder(ctx context.Context, companyID, id uuid.UUID) (*domain.SalesOrder, error)
	AddShippedQuantity(ctx context.Context, companyID, itemID uuid.UUID, quantity float64) (bool, error)

	SalesOrderSummary(ctx context.Context, companyID uuid.UUID) (*SalesOrderSummary, error)

	// --- numbering ---
	// NextOrderNumber reserves the next number for the tenant and year in one
	// atomic statement. Same shape as GenerateVoucherNo: a read-modify-write
	// from Go would hand two concurrent orders the same number and the second
	// INSERT would die on uq_*_orders_number.
	NextOrderNumber(ctx context.Context, companyID uuid.UUID, orderType string, year int) (string, error)

	// WithTransaction runs fn against repositories bound to one transaction.
	// Receiving and shipping need it: the order lines, the order status and the
	// stock movements must commit together or not at all.
	WithTransaction(ctx context.Context, fn func(orderRepo OrderRepository, stockRepo StockRepository) error) error
}
