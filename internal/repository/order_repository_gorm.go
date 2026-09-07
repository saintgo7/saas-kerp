package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// orderRepositoryGorm implements OrderRepository using GORM.
type orderRepositoryGorm struct {
	db *gorm.DB
}

// NewOrderRepositoryGorm creates an OrderRepository backed by GORM.
func NewOrderRepositoryGorm(db *gorm.DB) OrderRepository {
	return &orderRepositoryGorm{db: db}
}

// WithTransaction runs fn against an order and a stock repository that share
// one transaction.
func (r *orderRepositoryGorm) WithTransaction(ctx context.Context, fn func(orderRepo OrderRepository, stockRepo StockRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&orderRepositoryGorm{db: tx}, &stockRepositoryGorm{db: tx})
	})
}

// statusStrings converts a typed status list into the form a SQL IN takes.
func purchaseStatusStrings(statuses []domain.PurchaseOrderStatus) []string {
	out := make([]string, len(statuses))
	for i, s := range statuses {
		out[i] = string(s)
	}
	return out
}

func salesStatusStrings(statuses []domain.SalesOrderStatus) []string {
	out := make([]string, len(statuses))
	for i, s := range statuses {
		out[i] = string(s)
	}
	return out
}

// ---------------------------------------------------------------------------
// Purchase orders
// ---------------------------------------------------------------------------

// CreatePurchaseOrder inserts the order and its lines together.
func (r *orderRepositoryGorm) CreatePurchaseOrder(ctx context.Context, order *domain.PurchaseOrder) error {
	// Associations that exist only to be preloaded on the way out must not be
	// written on the way in: a partly-filled Supplier or Warehouse struct would
	// otherwise be upserted over the real row.
	return r.db.WithContext(ctx).
		Omit("Supplier", "Warehouse", "Items.Product").
		Create(order).Error
}

// GetPurchaseOrder reads one order with everything the UI renders.
func (r *orderRepositoryGorm) GetPurchaseOrder(ctx context.Context, companyID, id uuid.UUID) (*domain.PurchaseOrder, error) {
	var order domain.PurchaseOrder
	err := r.db.WithContext(ctx).
		Preload("Supplier").
		Preload("Warehouse").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("line_no ASC") }).
		Preload("Items.Product").
		Where("id = ? AND company_id = ?", id, companyID).
		First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
	}
	if err := r.attachPurchaseUserNames(ctx, companyID, []*domain.PurchaseOrder{&order}); err != nil {
		return nil, err
	}
	return &order, nil
}

// attachPurchaseUserNames fills in the creator and approver display names.
//
// Done with an explicit lookup rather than a GORM association so that only id
// and name are read, scoped by company_id; see lookupUserNames.
func (r *orderRepositoryGorm) attachPurchaseUserNames(ctx context.Context, companyID uuid.UUID, orders []*domain.PurchaseOrder) error {
	ids := make([]uuid.UUID, 0, len(orders)*2)
	for _, order := range orders {
		ids = append(ids, order.CreatedBy)
		if order.ApprovedBy != nil {
			ids = append(ids, *order.ApprovedBy)
		}
	}
	names, err := lookupUserNames(ctx, r.db, companyID, ids)
	if err != nil {
		return err
	}
	for _, order := range orders {
		order.CreatedByName = names[order.CreatedBy]
		if order.ApprovedBy != nil {
			order.ApprovedByName = names[*order.ApprovedBy]
		}
	}
	return nil
}

// attachSalesUserNames is attachPurchaseUserNames for the sales aggregate.
func (r *orderRepositoryGorm) attachSalesUserNames(ctx context.Context, companyID uuid.UUID, orders []*domain.SalesOrder) error {
	ids := make([]uuid.UUID, 0, len(orders)*2)
	for _, order := range orders {
		ids = append(ids, order.CreatedBy)
		if order.ApprovedBy != nil {
			ids = append(ids, *order.ApprovedBy)
		}
	}
	names, err := lookupUserNames(ctx, r.db, companyID, ids)
	if err != nil {
		return err
	}
	for _, order := range orders {
		order.CreatedByName = names[order.CreatedBy]
		if order.ApprovedBy != nil {
			order.ApprovedByName = names[*order.ApprovedBy]
		}
	}
	return nil
}

// ListPurchaseOrders returns a page of orders.
func (r *orderRepositoryGorm) ListPurchaseOrders(ctx context.Context, filter *PurchaseOrderFilter) ([]domain.PurchaseOrder, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.PurchaseOrder{}).
		Where("purchase_orders.company_id = ?", filter.CompanyID)

	if filter.SearchTerm != "" {
		// The list screen searches the supplier's name as well as the order
		// number, so the partner has to be joined for the predicate. Joined
		// once, before the count, so both agree.
		query = query.Joins("JOIN partners ON partners.id = purchase_orders.supplier_id")
		pattern := "%" + filter.SearchTerm + "%"
		query = query.Where("purchase_orders.order_number ILIKE ? OR partners.partner_name ILIKE ?", pattern, pattern)
	}
	if filter.Status != "" {
		query = query.Where("purchase_orders.status = ?", filter.Status)
	}
	if filter.SupplierID != nil {
		query = query.Where("purchase_orders.supplier_id = ?", *filter.SupplierID)
	}
	if filter.WarehouseID != nil {
		query = query.Where("purchase_orders.warehouse_id = ?", *filter.WarehouseID)
	}
	if filter.DateFrom != nil {
		query = query.Where("purchase_orders.order_date >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("purchase_orders.order_date <= ?", *filter.DateTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var orders []domain.PurchaseOrder
	// Items are preloaded on the list too: the receive dialog is opened from a
	// list row and works off that row's lines, so leaving them out would force
	// a second round trip for a modal the user has already opened.
	err := query.
		Preload("Supplier").
		Preload("Warehouse").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("line_no ASC") }).
		Preload("Items.Product").
		Order("purchase_orders.order_date DESC, purchase_orders.order_number DESC").
		Find(&orders).Error
	if err != nil {
		return nil, 0, err
	}

	refs := make([]*domain.PurchaseOrder, len(orders))
	for i := range orders {
		refs[i] = &orders[i]
	}
	if err := r.attachPurchaseUserNames(ctx, filter.CompanyID, refs); err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// UpdatePurchaseOrder saves the order header.
func (r *orderRepositoryGorm) UpdatePurchaseOrder(ctx context.Context, order *domain.PurchaseOrder) error {
	return r.db.WithContext(ctx).
		Omit("Supplier", "Warehouse", "Items").
		Where("company_id = ?", order.CompanyID).
		Save(order).Error
}

// ReplacePurchaseOrderItems swaps the whole line set of a draft order.
func (r *orderRepositoryGorm) ReplacePurchaseOrderItems(ctx context.Context, companyID, orderID uuid.UUID, items []domain.PurchaseOrderItem) error {
	replace := func(tx *gorm.DB) error {
		if err := tx.Where("company_id = ? AND purchase_order_id = ?", companyID, orderID).
			Delete(&domain.PurchaseOrderItem{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		return tx.Omit("Product").Create(&items).Error
	}
	if isInTransaction(r.db) {
		return replace(r.db.WithContext(ctx))
	}
	return r.db.WithContext(ctx).Transaction(replace)
}

// DeletePurchaseOrder removes an order whose status is in `allowed`.
//
// The status predicate is part of the DELETE rather than a preceding SELECT:
// checking first and deleting after leaves a window in which the order is
// approved between the two statements and gets deleted anyway.
func (r *orderRepositoryGorm) DeletePurchaseOrder(ctx context.Context, companyID, id uuid.UUID, allowed []domain.PurchaseOrderStatus) (bool, error) {
	res := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ? AND status IN ?", id, companyID, purchaseStatusStrings(allowed)).
		Delete(&domain.PurchaseOrder{})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// TransitionPurchaseOrder is the compare-and-swap described on the interface.
func (r *orderRepositoryGorm) TransitionPurchaseOrder(ctx context.Context, companyID, id uuid.UUID, from []domain.PurchaseOrderStatus, to domain.PurchaseOrderStatus, approvedBy *uuid.UUID) (bool, error) {
	updates := map[string]interface{}{
		"status":     string(to),
		"updated_at": gorm.Expr("NOW()"),
	}
	if approvedBy != nil {
		// chk_purchase_orders_approval requires both columns on any status past
		// pending, so they are always written together.
		updates["approved_by"] = *approvedBy
		updates["approved_at"] = gorm.Expr("NOW()")
	}

	res := r.db.WithContext(ctx).Model(&domain.PurchaseOrder{}).
		Where("id = ? AND company_id = ? AND status IN ?", id, companyID, purchaseStatusStrings(from)).
		Updates(updates)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// LockPurchaseOrder reads the order under SELECT ... FOR UPDATE.
func (r *orderRepositoryGorm) LockPurchaseOrder(ctx context.Context, companyID, id uuid.UUID) (*domain.PurchaseOrder, error) {
	var order domain.PurchaseOrder
	// The lock is taken on the header only. Locking the lines too (FOR UPDATE
	// with a join) is unnecessary: every receipt goes through the header first,
	// so the header row is the serialisation point for the whole order.
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND company_id = ?", id, companyID).
		First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
	}

	if err := r.db.WithContext(ctx).
		Where("company_id = ? AND purchase_order_id = ?", companyID, id).
		Order("line_no ASC").
		Find(&order.Items).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// AddReceivedQuantity raises one line's received quantity.
func (r *orderRepositoryGorm) AddReceivedQuantity(ctx context.Context, companyID, itemID uuid.UUID, quantity float64) (bool, error) {
	if quantity <= 0 {
		return false, domain.ErrOrderNothingToPost
	}
	// `received_quantity + ? <= quantity` in the WHERE clause is what makes a
	// replayed receipt a no-op instead of an over-receipt: the second request
	// finds no row to update rather than pushing the total past the order.
	res := r.db.WithContext(ctx).Model(&domain.PurchaseOrderItem{}).
		Where("id = ? AND company_id = ? AND received_quantity + ? <= quantity", itemID, companyID, quantity).
		Updates(map[string]interface{}{
			"received_quantity": gorm.Expr("received_quantity + ?", quantity),
			"updated_at":        gorm.Expr("NOW()"),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// PurchaseOrderSummary computes the purchase dashboard aggregate.
func (r *orderRepositoryGorm) PurchaseOrderSummary(ctx context.Context, companyID uuid.UUID) (*PurchaseOrderSummary, error) {
	var summary PurchaseOrderSummary
	err := r.db.WithContext(ctx).Raw(`
		SELECT
		    COUNT(*)                                                     AS total_count,
		    COUNT(*) FILTER (WHERE status = 'draft')                     AS draft_count,
		    COUNT(*) FILTER (WHERE status IN ('pending', 'approved', 'ordered', 'partial'))
		                                                                 AS pending_count,
		    COUNT(*) FILTER (WHERE status = 'completed')                 AS completed_count,
		    COUNT(*) FILTER (WHERE status = 'cancelled')                 AS cancelled_count,
		    COALESCE(SUM(grand_total) FILTER (WHERE status <> 'cancelled'), 0) AS total_amount,
		    COALESCE(SUM(grand_total) FILTER (WHERE status = 'completed'), 0)  AS completed_amount
		FROM purchase_orders
		WHERE company_id = ?
	`, companyID).Scan(&summary).Error
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

// ---------------------------------------------------------------------------
// Sales orders
// ---------------------------------------------------------------------------

// CreateSalesOrder inserts the order and its lines together.
func (r *orderRepositoryGorm) CreateSalesOrder(ctx context.Context, order *domain.SalesOrder) error {
	return r.db.WithContext(ctx).
		Omit("Customer", "Warehouse", "Items.Product").
		Create(order).Error
}

// GetSalesOrder reads one order with everything the UI renders.
func (r *orderRepositoryGorm) GetSalesOrder(ctx context.Context, companyID, id uuid.UUID) (*domain.SalesOrder, error) {
	var order domain.SalesOrder
	err := r.db.WithContext(ctx).
		Preload("Customer").
		Preload("Warehouse").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("line_no ASC") }).
		Preload("Items.Product").
		Where("id = ? AND company_id = ?", id, companyID).
		First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
	}
	if err := r.attachSalesUserNames(ctx, companyID, []*domain.SalesOrder{&order}); err != nil {
		return nil, err
	}
	return &order, nil
}

// ListSalesOrders returns a page of orders.
func (r *orderRepositoryGorm) ListSalesOrders(ctx context.Context, filter *SalesOrderFilter) ([]domain.SalesOrder, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.SalesOrder{}).
		Where("sales_orders.company_id = ?", filter.CompanyID)

	if filter.SearchTerm != "" {
		query = query.Joins("JOIN partners ON partners.id = sales_orders.customer_id")
		pattern := "%" + filter.SearchTerm + "%"
		query = query.Where("sales_orders.order_number ILIKE ? OR partners.partner_name ILIKE ?", pattern, pattern)
	}
	if filter.Status != "" {
		query = query.Where("sales_orders.status = ?", filter.Status)
	}
	if filter.CustomerID != nil {
		query = query.Where("sales_orders.customer_id = ?", *filter.CustomerID)
	}
	if filter.WarehouseID != nil {
		query = query.Where("sales_orders.warehouse_id = ?", *filter.WarehouseID)
	}
	if filter.DateFrom != nil {
		query = query.Where("sales_orders.order_date >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("sales_orders.order_date <= ?", *filter.DateTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var orders []domain.SalesOrder
	err := query.
		Preload("Customer").
		Preload("Warehouse").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("line_no ASC") }).
		Preload("Items.Product").
		Order("sales_orders.order_date DESC, sales_orders.order_number DESC").
		Find(&orders).Error
	if err != nil {
		return nil, 0, err
	}

	refs := make([]*domain.SalesOrder, len(orders))
	for i := range orders {
		refs[i] = &orders[i]
	}
	if err := r.attachSalesUserNames(ctx, filter.CompanyID, refs); err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// UpdateSalesOrder saves the order header.
func (r *orderRepositoryGorm) UpdateSalesOrder(ctx context.Context, order *domain.SalesOrder) error {
	return r.db.WithContext(ctx).
		Omit("Customer", "Warehouse", "Items").
		Where("company_id = ?", order.CompanyID).
		Save(order).Error
}

// ReplaceSalesOrderItems swaps the whole line set of a draft order.
func (r *orderRepositoryGorm) ReplaceSalesOrderItems(ctx context.Context, companyID, orderID uuid.UUID, items []domain.SalesOrderItem) error {
	replace := func(tx *gorm.DB) error {
		if err := tx.Where("company_id = ? AND sales_order_id = ?", companyID, orderID).
			Delete(&domain.SalesOrderItem{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		return tx.Omit("Product").Create(&items).Error
	}
	if isInTransaction(r.db) {
		return replace(r.db.WithContext(ctx))
	}
	return r.db.WithContext(ctx).Transaction(replace)
}

// DeleteSalesOrder removes an order whose status is in `allowed`.
func (r *orderRepositoryGorm) DeleteSalesOrder(ctx context.Context, companyID, id uuid.UUID, allowed []domain.SalesOrderStatus) (bool, error) {
	res := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ? AND status IN ?", id, companyID, salesStatusStrings(allowed)).
		Delete(&domain.SalesOrder{})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// TransitionSalesOrder is the compare-and-swap described on the interface.
func (r *orderRepositoryGorm) TransitionSalesOrder(ctx context.Context, companyID, id uuid.UUID, from []domain.SalesOrderStatus, to domain.SalesOrderStatus, approvedBy *uuid.UUID) (bool, error) {
	updates := map[string]interface{}{
		"status":     string(to),
		"updated_at": gorm.Expr("NOW()"),
	}
	if approvedBy != nil {
		updates["approved_by"] = *approvedBy
		updates["approved_at"] = gorm.Expr("NOW()")
	}

	res := r.db.WithContext(ctx).Model(&domain.SalesOrder{}).
		Where("id = ? AND company_id = ? AND status IN ?", id, companyID, salesStatusStrings(from)).
		Updates(updates)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// LockSalesOrder reads the order under SELECT ... FOR UPDATE.
func (r *orderRepositoryGorm) LockSalesOrder(ctx context.Context, companyID, id uuid.UUID) (*domain.SalesOrder, error) {
	var order domain.SalesOrder
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND company_id = ?", id, companyID).
		First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
	}

	if err := r.db.WithContext(ctx).
		Where("company_id = ? AND sales_order_id = ?", companyID, id).
		Order("line_no ASC").
		Find(&order.Items).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// AddShippedQuantity raises one line's shipped quantity.
func (r *orderRepositoryGorm) AddShippedQuantity(ctx context.Context, companyID, itemID uuid.UUID, quantity float64) (bool, error) {
	if quantity <= 0 {
		return false, domain.ErrOrderNothingToPost
	}
	res := r.db.WithContext(ctx).Model(&domain.SalesOrderItem{}).
		Where("id = ? AND company_id = ? AND shipped_quantity + ? <= quantity", itemID, companyID, quantity).
		Updates(map[string]interface{}{
			"shipped_quantity": gorm.Expr("shipped_quantity + ?", quantity),
			"updated_at":       gorm.Expr("NOW()"),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// SalesOrderSummary computes the sales dashboard aggregate.
func (r *orderRepositoryGorm) SalesOrderSummary(ctx context.Context, companyID uuid.UUID) (*SalesOrderSummary, error) {
	var summary SalesOrderSummary
	err := r.db.WithContext(ctx).Raw(`
		SELECT
		    COUNT(*)                                                     AS total_count,
		    COUNT(*) FILTER (WHERE status = 'draft')                     AS draft_count,
		    COUNT(*) FILTER (WHERE status IN ('confirmed', 'partial'))   AS pending_ship_count,
		    COUNT(*) FILTER (WHERE status = 'completed')                 AS completed_count,
		    COUNT(*) FILTER (WHERE status = 'cancelled')                 AS cancelled_count,
		    COALESCE(SUM(grand_total) FILTER (WHERE status <> 'cancelled'), 0) AS total_amount,
		    COALESCE(SUM(grand_total) FILTER (WHERE status = 'completed'), 0)  AS completed_amount
		FROM sales_orders
		WHERE company_id = ?
	`, companyID).Scan(&summary).Error
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

// ---------------------------------------------------------------------------
// Numbering
// ---------------------------------------------------------------------------

// orderNumberPrefixes maps an order type to the prefix its numbers carry.
var orderNumberPrefixes = map[string]string{
	"purchase": "PO",
	"sales":    "SO",
}

// NextOrderNumber reserves the next order number atomically.
func (r *orderRepositoryGorm) NextOrderNumber(ctx context.Context, companyID uuid.UUID, orderType string, year int) (string, error) {
	prefix, ok := orderNumberPrefixes[orderType]
	if !ok {
		return "", fmt.Errorf("unknown order type %q", orderType)
	}

	// One statement reserves the number and returns the value it reserved. The
	// counter can only move forward, and two concurrent callers get two
	// different values.
	var lastNumber int
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO order_sequences (company_id, order_type, fiscal_year, prefix, last_number, updated_at)
		VALUES (?, ?, ?, ?, 1, NOW())
		ON CONFLICT (company_id, order_type, fiscal_year)
		DO UPDATE SET last_number = order_sequences.last_number + 1, updated_at = NOW()
		RETURNING last_number
	`, companyID, orderType, year, prefix).Scan(&lastNumber).Error
	if err != nil {
		return "", err
	}
	if lastNumber <= 0 {
		// A zero here means the RETURNING produced no row, which under RLS means
		// the tenant context was not set. Reported rather than papered over with
		// a number that would collide on the next INSERT.
		return "", fmt.Errorf("order number generation returned no sequence value for %s %d", orderType, year)
	}

	return fmt.Sprintf("%s-%d-%06d", prefix, year, lastNumber), nil
}
