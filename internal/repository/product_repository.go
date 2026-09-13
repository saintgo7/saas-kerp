package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// ProductFilter defines filter criteria for listing products.
type ProductFilter struct {
	CompanyID  uuid.UUID
	CategoryID *uuid.UUID
	IsActive   *bool
	SearchTerm string // matched against code, name and specification
	Page       int
	PageSize   int
}

// CategoryFilter defines filter criteria for listing product categories.
type CategoryFilter struct {
	CompanyID  uuid.UUID
	ParentID   *uuid.UUID
	IsActive   *bool
	SearchTerm string
	Page       int
	PageSize   int
}

// WarehouseFilter defines filter criteria for listing warehouses.
type WarehouseFilter struct {
	CompanyID  uuid.UUID
	IsActive   *bool
	SearchTerm string
	Page       int
	PageSize   int
}

// ProductRepository is data access for the inventory master data: products,
// their categories and the warehouses stock lives in.
//
// Every method takes companyID and every query filters on it. That is belt and
// braces with the RLS policies from 000022: the policies are the guarantee, the
// explicit predicate is what keeps the code correct when a query runs on a
// connection whose app.current_tenant was not set (a background job, a test).
type ProductRepository interface {
	// Products
	Create(ctx context.Context, product *domain.Product) error
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Product, error)
	GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Product, error)
	List(ctx context.Context, filter *ProductFilter) ([]domain.Product, int64, error)
	Update(ctx context.Context, product *domain.Product) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error
	ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)
	ExistsByBarcode(ctx context.Context, companyID uuid.UUID, barcode string, excludeID *uuid.UUID) (bool, error)
	UpdateActiveStatus(ctx context.Context, companyID uuid.UUID, ids []uuid.UUID, isActive bool) error
	CountProducts(ctx context.Context, companyID uuid.UUID, activeOnly bool) (int64, error)

	// Referential checks used before a delete
	HasStock(ctx context.Context, companyID, productID uuid.UUID) (bool, error)
	HasMovements(ctx context.Context, companyID, productID uuid.UUID) (bool, error)
	HasOrderLines(ctx context.Context, companyID, productID uuid.UUID) (bool, error)

	// Categories
	CreateCategory(ctx context.Context, category *domain.ProductCategory) error
	GetCategoryByID(ctx context.Context, companyID, id uuid.UUID) (*domain.ProductCategory, error)
	ListCategories(ctx context.Context, filter *CategoryFilter) ([]domain.ProductCategory, int64, error)
	UpdateCategory(ctx context.Context, category *domain.ProductCategory) error
	DeleteCategory(ctx context.Context, companyID, id uuid.UUID) error
	ExistsCategoryByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)
	CategoryHasChildren(ctx context.Context, companyID, id uuid.UUID) (bool, error)
	CategoryHasProducts(ctx context.Context, companyID, id uuid.UUID) (bool, error)

	// Warehouses
	CreateWarehouse(ctx context.Context, warehouse *domain.Warehouse) error
	GetWarehouseByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Warehouse, error)
	ListWarehouses(ctx context.Context, filter *WarehouseFilter) ([]domain.Warehouse, int64, error)
	UpdateWarehouse(ctx context.Context, warehouse *domain.Warehouse) error
	DeleteWarehouse(ctx context.Context, companyID, id uuid.UUID) error
	ExistsWarehouseByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)
	ClearDefaultWarehouse(ctx context.Context, companyID uuid.UUID, excludeID *uuid.UUID) error
	WarehouseHasStock(ctx context.Context, companyID, warehouseID uuid.UUID) (bool, error)
	WarehouseHasOrders(ctx context.Context, companyID, warehouseID uuid.UUID) (bool, error)
	CountWarehouses(ctx context.Context, companyID uuid.UUID, activeOnly bool) (int64, error)

	// WithTransaction runs fn against a repository bound to one transaction.
	// Promoting a warehouse to default needs it: the old default has to be
	// cleared and the new one set together, or the partial unique index
	// uq_warehouses_one_default rejects the second statement.
	WithTransaction(ctx context.Context, fn func(repo ProductRepository) error) error
}
