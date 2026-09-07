package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// StockFilter defines filter criteria for listing stock balances.
type StockFilter struct {
	CompanyID   uuid.UUID
	WarehouseID *uuid.UUID
	ProductID   *uuid.UUID
	// BelowMinStock restricts the result to balances at or below the product's
	// reorder point. It is resolved in SQL, not by filtering a fetched page:
	// doing it in Go would make "low stock" mean "low stock on this page".
	BelowMinStock bool
	SearchTerm    string // matched against the product's code and name
	Page          int
	PageSize      int
}

// MovementFilter defines filter criteria for listing stock movements.
type MovementFilter struct {
	CompanyID    uuid.UUID
	WarehouseID  *uuid.UUID
	ProductID    *uuid.UUID
	MovementType string
	DateFrom     *time.Time
	DateTo       *time.Time
	Page         int
	PageSize     int
}

// StockSummary is the aggregate behind the stock dashboard cards.
type StockSummary struct {
	// StockRecords is the number of (product, warehouse) balance rows, not the
	// number of distinct products. The two differ as soon as one product is
	// stocked in two warehouses, so both are reported.
	StockRecords    int64   `json:"stock_records"`
	ProductCount    int64   `json:"product_count"`
	WarehouseCount  int64   `json:"warehouse_count"`
	TotalStockValue float64 `json:"total_stock_value"`
	LowStockCount   int64   `json:"low_stock_count"`
	OutOfStockCount int64   `json:"out_of_stock_count"`
}

// StockRepository is data access for balances and the movement ledger.
type StockRepository interface {
	// List returns a page of balances with their product and warehouse loaded.
	List(ctx context.Context, filter *StockFilter) ([]domain.Stock, int64, error)

	// Get returns one balance, or domain.ErrStockNotFound.
	Get(ctx context.Context, companyID, productID, warehouseID uuid.UUID) (*domain.Stock, error)

	// GetByID returns one balance by its own id.
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Stock, error)

	// ApplyMovement posts one movement and moves the cached balance by exactly
	// that much.
	//
	// It is the ONLY way stock changes in this codebase, and it does both halves
	// in a single transaction:
	//
	//  1. ensure a balance row exists for (company, product, warehouse)
	//  2. UPDATE stocks SET quantity = quantity + <delta> ... RETURNING quantity
	//  3. INSERT the movement with the before/after values step 2 returned
	//
	// Step 2 is a single statement, not a read followed by a write. Two
	// concurrent issues therefore serialise on the row lock, and the loser is
	// rejected by chk_stocks_quantity_non_negative rather than overwriting the
	// winner's balance. That read-modify-write is exactly the defect that was
	// fixed in voucher numbering, and it is worse here: the ledger and the cache
	// would disagree permanently.
	//
	// movement.PreviousQuantity and movement.CurrentQuantity are filled in from
	// the database; whatever the caller put there is ignored.
	ApplyMovement(ctx context.Context, movement *domain.StockMovement) error

	// ListMovements returns a page of the ledger, newest first.
	ListMovements(ctx context.Context, filter *MovementFilter) ([]domain.StockMovement, int64, error)

	// ListAlertCandidates returns every balance that breaches a threshold, with
	// its product loaded so the alert type can be derived. It is not paginated:
	// the caller needs the whole set to count it.
	ListAlertCandidates(ctx context.Context, companyID uuid.UUID) ([]domain.Stock, error)

	// Summary computes the dashboard aggregate over ALL of the tenant's stock.
	Summary(ctx context.Context, companyID uuid.UUID) (*StockSummary, error)

	// WithTransaction runs fn against a repository bound to one transaction.
	WithTransaction(ctx context.Context, fn func(repo StockRepository) error) error
}
