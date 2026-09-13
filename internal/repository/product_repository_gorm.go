package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// productRepositoryGorm implements ProductRepository using GORM.
type productRepositoryGorm struct {
	db *gorm.DB
}

// NewProductRepositoryGorm creates a ProductRepository backed by GORM.
func NewProductRepositoryGorm(db *gorm.DB) ProductRepository {
	return &productRepositoryGorm{db: db}
}

// WithTransaction runs fn against a repository bound to one transaction.
func (r *productRepositoryGorm) WithTransaction(ctx context.Context, fn func(repo ProductRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&productRepositoryGorm{db: tx})
	})
}

// ---------------------------------------------------------------------------
// Products
// ---------------------------------------------------------------------------

// Create inserts a product.
func (r *productRepositoryGorm) Create(ctx context.Context, product *domain.Product) error {
	// Omit the association: Category is a read-side preload target, and letting
	// GORM upsert through it would silently create or modify a category row
	// whenever a caller passed a partly-filled Category struct.
	return r.db.WithContext(ctx).Omit("Category").Create(product).Error
}

// GetByID retrieves one product with its category.
func (r *productRepositoryGorm) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Product, error) {
	var product domain.Product
	err := r.db.WithContext(ctx).
		Preload("Category").
		Where("id = ? AND company_id = ?", id, companyID).
		First(&product).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}
	return &product, nil
}

// GetByCode retrieves one product by its tenant-unique code.
func (r *productRepositoryGorm) GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Product, error) {
	var product domain.Product
	err := r.db.WithContext(ctx).
		Preload("Category").
		Where("company_id = ? AND code = ?", companyID, code).
		First(&product).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}
	return &product, nil
}

// List retrieves a page of products.
func (r *productRepositoryGorm) List(ctx context.Context, filter *ProductFilter) ([]domain.Product, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Product{}).
		Where("company_id = ?", filter.CompanyID)

	if filter.CategoryID != nil {
		query = query.Where("category_id = ?", *filter.CategoryID)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.SearchTerm != "" {
		pattern := "%" + filter.SearchTerm + "%"
		query = query.Where(
			"code ILIKE ? OR name ILIKE ? OR specification ILIKE ?",
			pattern, pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var products []domain.Product
	// Preload rather than a join: the product list renders category.name for
	// every row, and without it the frontend shows an empty category column.
	if err := query.Preload("Category").Order("code ASC").Find(&products).Error; err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

// Update saves a product. Select is explicit so a zero-valued field still
// writes: Save on a struct with Omit would skip Description when cleared.
func (r *productRepositoryGorm) Update(ctx context.Context, product *domain.Product) error {
	return r.db.WithContext(ctx).Omit("Category").
		Where("company_id = ?", product.CompanyID).
		Save(product).Error
}

// Delete soft deletes a product.
func (r *productRepositoryGorm) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		Delete(&domain.Product{}).Error
}

// ExistsByCode reports whether another live product already holds the code.
func (r *productRepositoryGorm) ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	return r.existsBy(ctx, &domain.Product{}, companyID, "code", code, excludeID)
}

// ExistsByBarcode reports whether another live product already holds the barcode.
func (r *productRepositoryGorm) ExistsByBarcode(ctx context.Context, companyID uuid.UUID, barcode string, excludeID *uuid.UUID) (bool, error) {
	if barcode == "" {
		return false, nil
	}
	return r.existsBy(ctx, &domain.Product{}, companyID, "barcode", barcode, excludeID)
}

// existsBy is the shared shape of the uniqueness pre-checks.
func (r *productRepositoryGorm) existsBy(ctx context.Context, model interface{}, companyID uuid.UUID, column string, value interface{}, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).Model(model).
		Where("company_id = ? AND "+column+" = ?", companyID, value)
	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateActiveStatus flips is_active for a set of products.
func (r *productRepositoryGorm) UpdateActiveStatus(ctx context.Context, companyID uuid.UUID, ids []uuid.UUID, isActive bool) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&domain.Product{}).
		Where("company_id = ? AND id IN ?", companyID, ids).
		Update("is_active", isActive).Error
}

// CountProducts counts products, optionally only the active ones.
func (r *productRepositoryGorm) CountProducts(ctx context.Context, companyID uuid.UUID, activeOnly bool) (int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Product{}).Where("company_id = ?", companyID)
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

// HasStock reports whether any warehouse holds a non-zero balance of the product.
func (r *productRepositoryGorm) HasStock(ctx context.Context, companyID, productID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Stock{}).
		Where("company_id = ? AND product_id = ? AND quantity <> 0", companyID, productID).
		Count(&count).Error
	return count > 0, err
}

// HasMovements reports whether the product appears in the stock ledger.
func (r *productRepositoryGorm) HasMovements(ctx context.Context, companyID, productID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.StockMovement{}).
		Where("company_id = ? AND product_id = ?", companyID, productID).
		Count(&count).Error
	return count > 0, err
}

// HasOrderLines reports whether the product appears on a purchase or sales order.
func (r *productRepositoryGorm) HasOrderLines(ctx context.Context, companyID, productID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.PurchaseOrderItem{}).
		Where("company_id = ? AND product_id = ?", companyID, productID).
		Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	err := r.db.WithContext(ctx).Model(&domain.SalesOrderItem{}).
		Where("company_id = ? AND product_id = ?", companyID, productID).
		Count(&count).Error
	return count > 0, err
}

// ---------------------------------------------------------------------------
// Categories
// ---------------------------------------------------------------------------

// CreateCategory inserts a product category.
func (r *productRepositoryGorm) CreateCategory(ctx context.Context, category *domain.ProductCategory) error {
	return r.db.WithContext(ctx).Create(category).Error
}

// GetCategoryByID retrieves one category.
func (r *productRepositoryGorm) GetCategoryByID(ctx context.Context, companyID, id uuid.UUID) (*domain.ProductCategory, error) {
	var category domain.ProductCategory
	err := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, err
	}
	return &category, nil
}

// ListCategories retrieves a page of categories.
func (r *productRepositoryGorm) ListCategories(ctx context.Context, filter *CategoryFilter) ([]domain.ProductCategory, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.ProductCategory{}).
		Where("company_id = ?", filter.CompanyID)

	if filter.ParentID != nil {
		query = query.Where("parent_id = ?", *filter.ParentID)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.SearchTerm != "" {
		pattern := "%" + filter.SearchTerm + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var categories []domain.ProductCategory
	if err := query.Order("level ASC, code ASC").Find(&categories).Error; err != nil {
		return nil, 0, err
	}
	return categories, total, nil
}

// UpdateCategory saves a category.
func (r *productRepositoryGorm) UpdateCategory(ctx context.Context, category *domain.ProductCategory) error {
	return r.db.WithContext(ctx).
		Where("company_id = ?", category.CompanyID).
		Save(category).Error
}

// DeleteCategory soft deletes a category.
func (r *productRepositoryGorm) DeleteCategory(ctx context.Context, companyID, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		Delete(&domain.ProductCategory{}).Error
}

// ExistsCategoryByCode reports whether another live category holds the code.
func (r *productRepositoryGorm) ExistsCategoryByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	return r.existsBy(ctx, &domain.ProductCategory{}, companyID, "code", code, excludeID)
}

// CategoryHasChildren reports whether the category still has child categories.
func (r *productRepositoryGorm) CategoryHasChildren(ctx context.Context, companyID, id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.ProductCategory{}).
		Where("company_id = ? AND parent_id = ?", companyID, id).
		Count(&count).Error
	return count > 0, err
}

// CategoryHasProducts reports whether any product still points at the category.
func (r *productRepositoryGorm) CategoryHasProducts(ctx context.Context, companyID, id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Product{}).
		Where("company_id = ? AND category_id = ?", companyID, id).
		Count(&count).Error
	return count > 0, err
}

// ---------------------------------------------------------------------------
// Warehouses
// ---------------------------------------------------------------------------

// CreateWarehouse inserts a warehouse.
func (r *productRepositoryGorm) CreateWarehouse(ctx context.Context, warehouse *domain.Warehouse) error {
	return r.db.WithContext(ctx).Create(warehouse).Error
}

// GetWarehouseByID retrieves one warehouse.
func (r *productRepositoryGorm) GetWarehouseByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Warehouse, error) {
	var warehouse domain.Warehouse
	err := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		First(&warehouse).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrWarehouseNotFound
		}
		return nil, err
	}
	return &warehouse, nil
}

// ListWarehouses retrieves a page of warehouses.
func (r *productRepositoryGorm) ListWarehouses(ctx context.Context, filter *WarehouseFilter) ([]domain.Warehouse, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Warehouse{}).
		Where("company_id = ?", filter.CompanyID)

	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.SearchTerm != "" {
		pattern := "%" + filter.SearchTerm + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var warehouses []domain.Warehouse
	if err := query.Order("is_default DESC, code ASC").Find(&warehouses).Error; err != nil {
		return nil, 0, err
	}
	return warehouses, total, nil
}

// UpdateWarehouse saves a warehouse.
func (r *productRepositoryGorm) UpdateWarehouse(ctx context.Context, warehouse *domain.Warehouse) error {
	return r.db.WithContext(ctx).
		Where("company_id = ?", warehouse.CompanyID).
		Save(warehouse).Error
}

// DeleteWarehouse soft deletes a warehouse.
func (r *productRepositoryGorm) DeleteWarehouse(ctx context.Context, companyID, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		Delete(&domain.Warehouse{}).Error
}

// ExistsWarehouseByCode reports whether another live warehouse holds the code.
func (r *productRepositoryGorm) ExistsWarehouseByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	return r.existsBy(ctx, &domain.Warehouse{}, companyID, "code", code, excludeID)
}

// ClearDefaultWarehouse takes the default flag off every warehouse but excludeID.
//
// uq_warehouses_one_default permits exactly one flagged row per tenant, so
// promoting a new default has to clear the old one first, in the same
// transaction. Callers reach this through WithTransaction.
func (r *productRepositoryGorm) ClearDefaultWarehouse(ctx context.Context, companyID uuid.UUID, excludeID *uuid.UUID) error {
	query := r.db.WithContext(ctx).Model(&domain.Warehouse{}).
		Where("company_id = ? AND is_default = ?", companyID, true)
	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}
	return query.Update("is_default", false).Error
}

// WarehouseHasStock reports whether the warehouse holds any non-zero balance.
func (r *productRepositoryGorm) WarehouseHasStock(ctx context.Context, companyID, warehouseID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Stock{}).
		Where("company_id = ? AND warehouse_id = ? AND quantity <> 0", companyID, warehouseID).
		Count(&count).Error
	return count > 0, err
}

// WarehouseHasOrders reports whether any order still points at the warehouse.
func (r *productRepositoryGorm) WarehouseHasOrders(ctx context.Context, companyID, warehouseID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.PurchaseOrder{}).
		Where("company_id = ? AND warehouse_id = ?", companyID, warehouseID).
		Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	err := r.db.WithContext(ctx).Model(&domain.SalesOrder{}).
		Where("company_id = ? AND warehouse_id = ?", companyID, warehouseID).
		Count(&count).Error
	return count > 0, err
}

// CountWarehouses counts warehouses, optionally only the active ones.
func (r *productRepositoryGorm) CountWarehouses(ctx context.Context, companyID uuid.UUID, activeOnly bool) (int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Warehouse{}).Where("company_id = ?", companyID)
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}
