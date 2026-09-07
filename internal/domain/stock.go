package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Stock errors.
var (
	ErrStockNotFound            = errors.New("stock record not found")
	ErrStockInsufficient        = errors.New("insufficient stock")
	ErrStockInvalidMovement     = errors.New("invalid stock movement type")
	ErrStockQuantityNotPositive = errors.New("stock movement quantity must be greater than zero")
	ErrStockNegativeResult      = errors.New("stock movement would drive the balance negative")
	ErrStockSameWarehouse       = errors.New("transfer source and destination warehouses must differ")
)

// StockMovementType classifies one line of the stock ledger. The values are the
// StockMovementType union in web/src/types/inventory.ts, and the same set is
// enforced by chk_stock_movements_type in 000022.
type StockMovementType string

const (
	MovementPurchaseIn    StockMovementType = "purchase_in"
	MovementSalesOut      StockMovementType = "sales_out"
	MovementAdjustmentIn  StockMovementType = "adjustment_in"
	MovementAdjustmentOut StockMovementType = "adjustment_out"
	MovementTransferIn    StockMovementType = "transfer_in"
	MovementTransferOut   StockMovementType = "transfer_out"
	MovementReturnIn      StockMovementType = "return_in"
	MovementReturnOut     StockMovementType = "return_out"
)

// IsValid reports whether t is one of the eight known movement types.
func (t StockMovementType) IsValid() bool {
	switch t {
	case MovementPurchaseIn, MovementSalesOut,
		MovementAdjustmentIn, MovementAdjustmentOut,
		MovementTransferIn, MovementTransferOut,
		MovementReturnIn, MovementReturnOut:
		return true
	}
	return false
}

// IsInbound reports whether the movement raises the balance.
//
// Direction lives here and in the movement type alone. Quantity is always
// stored positive (chk_stock_movements_quantity_positive), so no caller can
// invert a receipt into an issue by passing a negative number.
func (t StockMovementType) IsInbound() bool {
	switch t {
	case MovementPurchaseIn, MovementAdjustmentIn, MovementTransferIn, MovementReturnIn:
		return true
	}
	return false
}

// SignedDelta turns a positive quantity into the change it makes to the balance.
func (t StockMovementType) SignedDelta(quantity float64) float64 {
	if t.IsInbound() {
		return quantity
	}
	return -quantity
}

// Stock is the cached on-hand balance of one product in one warehouse.
//
// It is a cache: stock_movements is the ledger of record. Nothing may write
// Quantity by assignment - see StockRepository.ApplyMovement, which changes it
// with a single atomic UPDATE inside the same transaction as the movement row.
type Stock struct {
	TenantModel

	ProductID   uuid.UUID  `gorm:"type:uuid;not null" json:"product_id"`
	Product     *Product   `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	WarehouseID uuid.UUID  `gorm:"type:uuid;not null" json:"warehouse_id"`
	Warehouse   *Warehouse `gorm:"foreignKey:WarehouseID" json:"warehouse,omitempty"`

	Quantity         float64 `gorm:"type:numeric(18,3);not null;default:0" json:"quantity"`
	ReservedQuantity float64 `gorm:"type:numeric(18,3);not null;default:0" json:"reserved_quantity"`

	// AvailableQuantity is GENERATED ALWAYS AS (quantity - reserved_quantity)
	// STORED in PostgreSQL. The "->" tag makes it read-only for GORM: without
	// it every INSERT and UPDATE would try to write a generated column and the
	// statement would be rejected outright.
	AvailableQuantity float64 `gorm:"type:numeric(18,3);->" json:"available_quantity"`

	LastUpdatedAt time.Time `gorm:"not null;default:now()" json:"last_updated_at"`
}

// TableName specifies the table name for GORM.
func (Stock) TableName() string {
	return "stocks"
}

// IsBelowMinStock reports whether the balance has fallen to or below the
// product's reorder point. A product with no reorder point is never below it.
func (s *Stock) IsBelowMinStock() bool {
	if s.Product == nil || s.Product.MinStock == nil {
		return false
	}
	return s.Quantity <= *s.Product.MinStock
}

// StockValue is the carrying value of the balance at cost.
func (s *Stock) StockValue() float64 {
	if s.Product == nil {
		return 0
	}
	return s.Quantity * s.Product.CostPrice
}

// StockMovement is one line of the append-only stock ledger.
//
// It deliberately does NOT embed TenantModel: the table has no updated_at
// column, because a movement is never modified once written. Embedding the
// shared model would make GORM emit updated_at on every INSERT and fail.
type StockMovement struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v7()" json:"id"`
	CompanyID uuid.UUID `gorm:"type:uuid;not null;index" json:"company_id"`

	ProductID   uuid.UUID  `gorm:"type:uuid;not null" json:"product_id"`
	Product     *Product   `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	WarehouseID uuid.UUID  `gorm:"type:uuid;not null" json:"warehouse_id"`
	Warehouse   *Warehouse `gorm:"foreignKey:WarehouseID" json:"warehouse,omitempty"`

	MovementType StockMovementType `gorm:"type:varchar(20);not null" json:"movement_type"`

	// Always positive. Direction comes from MovementType.
	Quantity         float64 `gorm:"type:numeric(18,3);not null" json:"quantity"`
	PreviousQuantity float64 `gorm:"type:numeric(18,3);not null" json:"previous_quantity"`
	CurrentQuantity  float64 `gorm:"type:numeric(18,3);not null" json:"current_quantity"`

	// What caused the movement: "purchase_order", "sales_order",
	// "stock_adjustment", "stock_transfer". Not a foreign key - the referent is
	// polymorphic - but always populated so a balance can be traced back.
	ReferenceType string     `gorm:"type:varchar(30)" json:"reference_type,omitempty"`
	ReferenceID   *uuid.UUID `gorm:"type:uuid" json:"reference_id,omitempty"`

	Note string `gorm:"type:text" json:"note,omitempty"`

	CreatedBy uuid.UUID `gorm:"type:uuid;not null" json:"created_by"`

	// CreatedByName is filled in by the repository, not by a GORM association.
	//
	// A `Creator *User` belongs-to would pull the whole user row - password
	// hash included - into a stock movement listing. The explicit lookup in
	// lookupUserNames selects id and name only, and carries the company_id
	// predicate that an association would not.
	CreatedByName string `gorm:"-" json:"created_by_name,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName specifies the table name for GORM.
func (StockMovement) TableName() string {
	return "stock_movements"
}

// Reference constants for StockMovement.ReferenceType.
const (
	StockRefPurchaseOrder   = "purchase_order"
	StockRefSalesOrder      = "sales_order"
	StockRefStockAdjustment = "stock_adjustment"
	StockRefStockTransfer   = "stock_transfer"
)

// StockAlertType classifies a stock alert.
type StockAlertType string

const (
	AlertLowStock   StockAlertType = "low_stock"
	AlertOutOfStock StockAlertType = "out_of_stock"
	AlertOverstock  StockAlertType = "overstock"
)

// StockAlert is a threshold breach on one balance.
//
// It has NO TABLE, on purpose. An alert is a pure function of the current
// balance and the product's min/max stock, so storing it would create a second
// copy of a fact the database already holds - one that goes stale the moment a
// movement is posted and that needs a background job nobody has written to stay
// current. StockService derives alerts on read instead; see StockAlert.ID for
// what that costs.
type StockAlert struct {
	// ID is the id of the stock row the alert was derived from, not a row id of
	// its own. It is stable for as long as the breach lasts, which is what the
	// UI needs for a list key.
	ID uuid.UUID `json:"id"`

	CompanyID   uuid.UUID  `json:"company_id"`
	ProductID   uuid.UUID  `json:"product_id"`
	Product     *Product   `json:"product,omitempty"`
	WarehouseID uuid.UUID  `json:"warehouse_id"`
	Warehouse   *Warehouse `json:"warehouse,omitempty"`

	AlertType       StockAlertType `json:"alert_type"`
	CurrentQuantity float64        `json:"current_quantity"`
	Threshold       float64        `json:"threshold"`

	// IsRead is always false: a derived alert has nowhere to record that
	// somebody dismissed it. Kept so the response matches the frontend's
	// StockAlert interface rather than silently dropping a field.
	IsRead bool `json:"is_read"`

	// CreatedAt is the moment the balance last changed, which is the closest
	// honest answer to "when did this alert arise".
	CreatedAt time.Time `json:"created_at"`
}

// DeriveStockAlert returns the alert raised by a balance, or nil when the
// balance is within its thresholds.
//
// Order matters: an empty balance is reported as out_of_stock rather than
// low_stock, because those are different operational situations even though a
// zero balance is also below every non-zero reorder point.
func DeriveStockAlert(s *Stock) *StockAlert {
	if s == nil || s.Product == nil {
		return nil
	}

	base := StockAlert{
		ID:              s.ID,
		CompanyID:       s.CompanyID,
		ProductID:       s.ProductID,
		Product:         s.Product,
		WarehouseID:     s.WarehouseID,
		Warehouse:       s.Warehouse,
		CurrentQuantity: s.Quantity,
		IsRead:          false,
		CreatedAt:       s.LastUpdatedAt,
	}

	switch {
	case s.Quantity <= 0:
		base.AlertType = AlertOutOfStock
		if s.Product.MinStock != nil {
			base.Threshold = *s.Product.MinStock
		}
		return &base

	case s.Product.MinStock != nil && s.Quantity <= *s.Product.MinStock:
		base.AlertType = AlertLowStock
		base.Threshold = *s.Product.MinStock
		return &base

	case s.Product.MaxStock != nil && s.Quantity > *s.Product.MaxStock:
		base.AlertType = AlertOverstock
		base.Threshold = *s.Product.MaxStock
		return &base
	}

	return nil
}
