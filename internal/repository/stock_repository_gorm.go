package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// stockRepositoryGorm implements StockRepository using GORM.
type stockRepositoryGorm struct {
	db *gorm.DB
}

// NewStockRepositoryGorm creates a StockRepository backed by GORM.
func NewStockRepositoryGorm(db *gorm.DB) StockRepository {
	return &stockRepositoryGorm{db: db}
}

// WithTransaction runs fn against a repository bound to one transaction.
func (r *stockRepositoryGorm) WithTransaction(ctx context.Context, fn func(repo StockRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&stockRepositoryGorm{db: tx})
	})
}

// List returns a page of balances.
func (r *stockRepositoryGorm) List(ctx context.Context, filter *StockFilter) ([]domain.Stock, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Stock{}).
		Where("stocks.company_id = ?", filter.CompanyID)

	// The product join is needed for the search term and for the low-stock
	// predicate. It is added once, here, so the count and the page agree.
	needsProductJoin := filter.SearchTerm != "" || filter.BelowMinStock
	if needsProductJoin {
		query = query.Joins("JOIN products ON products.id = stocks.product_id AND products.deleted_at IS NULL")
	}

	if filter.WarehouseID != nil {
		query = query.Where("stocks.warehouse_id = ?", *filter.WarehouseID)
	}
	if filter.ProductID != nil {
		query = query.Where("stocks.product_id = ?", *filter.ProductID)
	}
	if filter.SearchTerm != "" {
		pattern := "%" + filter.SearchTerm + "%"
		query = query.Where("products.code ILIKE ? OR products.name ILIKE ?", pattern, pattern)
	}
	if filter.BelowMinStock {
		// A product with no reorder point is never "low": NULL min_stock must
		// not be coerced to zero, or every item with an empty balance would be
		// reported the moment it hit zero.
		query = query.Where("products.min_stock IS NOT NULL AND stocks.quantity <= products.min_stock")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var stocks []domain.Stock
	err := query.
		Preload("Product").
		Preload("Product.Category").
		Preload("Warehouse").
		Order("stocks.warehouse_id, stocks.product_id").
		Find(&stocks).Error
	if err != nil {
		return nil, 0, err
	}
	return stocks, total, nil
}

// Get returns the balance of one product in one warehouse.
func (r *stockRepositoryGorm) Get(ctx context.Context, companyID, productID, warehouseID uuid.UUID) (*domain.Stock, error) {
	var stock domain.Stock
	err := r.db.WithContext(ctx).
		Preload("Product").
		Preload("Warehouse").
		Where("company_id = ? AND product_id = ? AND warehouse_id = ?", companyID, productID, warehouseID).
		First(&stock).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrStockNotFound
		}
		return nil, err
	}
	return &stock, nil
}

// GetByID returns one balance by its own id.
func (r *stockRepositoryGorm) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Stock, error) {
	var stock domain.Stock
	err := r.db.WithContext(ctx).
		Preload("Product").
		Preload("Warehouse").
		Where("id = ? AND company_id = ?", id, companyID).
		First(&stock).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrStockNotFound
		}
		return nil, err
	}
	return &stock, nil
}

// ApplyMovement posts a movement and moves the cached balance atomically.
// See the interface comment for why every step is shaped the way it is.
func (r *stockRepositoryGorm) ApplyMovement(ctx context.Context, movement *domain.StockMovement) error {
	if movement == nil {
		return errors.New("stock movement is nil")
	}
	if !movement.MovementType.IsValid() {
		return domain.ErrStockInvalidMovement
	}
	if movement.Quantity <= 0 {
		return domain.ErrStockQuantityNotPositive
	}

	apply := func(tx *gorm.DB) error {
		// 1. A balance row must exist before it can be moved. DO NOTHING rather
		//    than DO UPDATE: two concurrent first-receipts both reach here, and
		//    only one may create the row - the other must fall through to the
		//    UPDATE below and add to what the winner created.
		ensure := tx.Exec(`
			INSERT INTO stocks (company_id, product_id, warehouse_id, quantity, reserved_quantity)
			VALUES (?, ?, ?, 0, 0)
			ON CONFLICT (company_id, product_id, warehouse_id) DO NOTHING
		`, movement.CompanyID, movement.ProductID, movement.WarehouseID)
		if ensure.Error != nil {
			return ensure.Error
		}

		// 2. The atomic step. `quantity + ?` is evaluated by PostgreSQL against
		//    the row it just locked, so no value read into Go can go stale
		//    between the read and the write.
		delta := movement.MovementType.SignedDelta(movement.Quantity)

		var updated struct {
			Quantity float64
		}
		res := tx.Raw(`
			UPDATE stocks
			   SET quantity = quantity + ?,
			       last_updated_at = NOW(),
			       updated_at = NOW()
			 WHERE company_id = ? AND product_id = ? AND warehouse_id = ?
			RETURNING quantity
		`, delta, movement.CompanyID, movement.ProductID, movement.WarehouseID).Scan(&updated)

		if res.Error != nil {
			// chk_stocks_quantity_non_negative is how an over-issue is refused,
			// including the concurrent case where the balance was still
			// sufficient when the request was validated. Reported as a domain
			// error so the handler answers 422 rather than a raw 500.
			if isCheckViolation(res.Error, "chk_stocks_quantity_non_negative") ||
				isCheckViolation(res.Error, "chk_stocks_reserved_within_quantity") {
				return domain.ErrStockNegativeResult
			}
			return res.Error
		}
		if res.RowsAffected == 0 {
			// The row existed a statement ago. Zero rows here means the tenant
			// predicate did not match - an RLS policy refusal, not a race.
			return domain.ErrStockNotFound
		}

		// 3. The ledger row records the balance the database actually reached,
		//    never a number computed in Go.
		movement.CurrentQuantity = updated.Quantity
		movement.PreviousQuantity = updated.Quantity - delta

		return tx.Omit("Product", "Warehouse").Create(movement).Error
	}

	// Reuse the caller's transaction when there is one. Receiving a purchase
	// order posts several movements and updates the order in one unit of work;
	// opening a nested transaction per movement would let a later failure leave
	// earlier receipts committed.
	if isInTransaction(r.db) {
		return apply(r.db.WithContext(ctx))
	}
	return r.db.WithContext(ctx).Transaction(apply)
}

// ListMovements returns a page of the ledger, newest first.
func (r *stockRepositoryGorm) ListMovements(ctx context.Context, filter *MovementFilter) ([]domain.StockMovement, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.StockMovement{}).
		Where("company_id = ?", filter.CompanyID)

	if filter.WarehouseID != nil {
		query = query.Where("warehouse_id = ?", *filter.WarehouseID)
	}
	if filter.ProductID != nil {
		query = query.Where("product_id = ?", *filter.ProductID)
	}
	if filter.MovementType != "" {
		query = query.Where("movement_type = ?", filter.MovementType)
	}
	if filter.DateFrom != nil {
		query = query.Where("created_at >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("created_at <= ?", *filter.DateTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var movements []domain.StockMovement
	err := query.
		Preload("Product").
		Preload("Warehouse").
		Order("created_at DESC, id DESC").
		Find(&movements).Error
	if err != nil {
		return nil, 0, err
	}

	ids := make([]uuid.UUID, 0, len(movements))
	for i := range movements {
		ids = append(ids, movements[i].CreatedBy)
	}
	names, err := lookupUserNames(ctx, r.db, filter.CompanyID, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range movements {
		movements[i].CreatedByName = names[movements[i].CreatedBy]
	}

	return movements, total, nil
}

// lookupUserNames resolves user ids to display names for one tenant.
//
// It selects id and name only. A GORM association would read the whole user
// row - password hash included - to render a display name.
//
// The company_id predicate is not decoration: without it this would resolve a
// name for a user id belonging to another tenant, which is a cross-tenant read
// however small.
func lookupUserNames(ctx context.Context, db *gorm.DB, companyID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	names := make(map[uuid.UUID]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}

	// De-duplicate: a list of 100 movements is usually a handful of users.
	unique := make([]uuid.UUID, 0, len(ids))
	seen := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return names, nil
	}

	var rows []struct {
		ID   uuid.UUID
		Name string
	}
	err := db.WithContext(ctx).
		Table("users").
		Select("id, name").
		Where("company_id = ? AND id IN ? AND deleted_at IS NULL", companyID, unique).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		names[row.ID] = row.Name
	}
	return names, nil
}

// ListAlertCandidates returns every balance that breaches a threshold.
func (r *stockRepositoryGorm) ListAlertCandidates(ctx context.Context, companyID uuid.UUID) ([]domain.Stock, error) {
	var stocks []domain.Stock
	err := r.db.WithContext(ctx).Model(&domain.Stock{}).
		Joins("JOIN products ON products.id = stocks.product_id AND products.deleted_at IS NULL").
		Where("stocks.company_id = ?", companyID).
		Where(`
			stocks.quantity <= 0
			OR (products.min_stock IS NOT NULL AND stocks.quantity <= products.min_stock)
			OR (products.max_stock IS NOT NULL AND stocks.quantity > products.max_stock)
		`).
		Preload("Product").
		Preload("Warehouse").
		Order("stocks.quantity ASC").
		Find(&stocks).Error
	if err != nil {
		return nil, err
	}
	return stocks, nil
}

// Summary computes the dashboard aggregate over all of the tenant's stock.
//
// Every figure is computed by the database over the whole tenant. Computing
// them in Go from a page of results would make each card report "…of the 20
// rows currently on screen", which is the failure the mock frontend has today.
func (r *stockRepositoryGorm) Summary(ctx context.Context, companyID uuid.UUID) (*StockSummary, error) {
	var row struct {
		StockRecords    int64
		ProductCount    int64
		TotalStockValue float64
		LowStockCount   int64
		OutOfStockCount int64
	}

	err := r.db.WithContext(ctx).Raw(`
		SELECT
		    COUNT(*)                                        AS stock_records,
		    COUNT(DISTINCT s.product_id)                    AS product_count,
		    COALESCE(SUM(s.quantity * p.cost_price), 0)     AS total_stock_value,
		    COUNT(*) FILTER (
		        WHERE s.quantity > 0
		          AND p.min_stock IS NOT NULL
		          AND s.quantity <= p.min_stock)            AS low_stock_count,
		    COUNT(*) FILTER (WHERE s.quantity <= 0)         AS out_of_stock_count
		FROM stocks s
		JOIN products p ON p.id = s.product_id AND p.deleted_at IS NULL
		WHERE s.company_id = ?
	`, companyID).Scan(&row).Error
	if err != nil {
		return nil, err
	}

	var warehouseCount int64
	if err := r.db.WithContext(ctx).Model(&domain.Warehouse{}).
		Where("company_id = ? AND is_active = ?", companyID, true).
		Count(&warehouseCount).Error; err != nil {
		return nil, err
	}

	return &StockSummary{
		StockRecords:    row.StockRecords,
		ProductCount:    row.ProductCount,
		WarehouseCount:  warehouseCount,
		TotalStockValue: row.TotalStockValue,
		LowStockCount:   row.LowStockCount,
		OutOfStockCount: row.OutOfStockCount,
	}, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// isInTransaction reports whether db is already inside a transaction, so that
// a repository method can join the caller's unit of work instead of opening a
// second one.
func isInTransaction(db *gorm.DB) bool {
	if db == nil || db.Statement == nil {
		return false
	}
	committer, ok := db.Statement.ConnPool.(gorm.TxCommitter)
	return ok && committer != nil
}

// isCheckViolation reports whether err is a PostgreSQL check-constraint
// violation for the named constraint.
//
// Matched on the constraint name rather than on the SQLSTATE alone: this
// repository must distinguish "you tried to issue more than you have" (a
// business answer, 422) from any other integrity failure (a bug, 500). The
// message is only read here, never forwarded to a client.
func isCheckViolation(err error, constraint string) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, constraint)
}
