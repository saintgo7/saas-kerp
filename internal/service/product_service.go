package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// Filters re-exported from the repository so handlers do not import it.
type (
	ProductFilter   = repository.ProductFilter
	CategoryFilter  = repository.CategoryFilter
	WarehouseFilter = repository.WarehouseFilter
)

// ProductStats holds the product master-data counters.
type ProductStats struct {
	TotalCount    int64 `json:"total_count"`
	ActiveCount   int64 `json:"active_count"`
	InactiveCount int64 `json:"inactive_count"`
}

// ProductService is the business logic for inventory master data.
type ProductService interface {
	// Products
	Create(ctx context.Context, product *domain.Product) error
	Update(ctx context.Context, product *domain.Product) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Product, error)
	GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Product, error)
	List(ctx context.Context, filter *ProductFilter) ([]domain.Product, int64, error)
	SetActive(ctx context.Context, companyID uuid.UUID, ids []uuid.UUID, isActive bool) error
	CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error)
	GetStats(ctx context.Context, companyID uuid.UUID) (*ProductStats, error)

	// Categories
	CreateCategory(ctx context.Context, category *domain.ProductCategory) error
	UpdateCategory(ctx context.Context, category *domain.ProductCategory) error
	DeleteCategory(ctx context.Context, companyID, id uuid.UUID) error
	GetCategory(ctx context.Context, companyID, id uuid.UUID) (*domain.ProductCategory, error)
	ListCategories(ctx context.Context, filter *CategoryFilter) ([]domain.ProductCategory, int64, error)

	// Warehouses
	CreateWarehouse(ctx context.Context, warehouse *domain.Warehouse) error
	UpdateWarehouse(ctx context.Context, warehouse *domain.Warehouse) error
	DeleteWarehouse(ctx context.Context, companyID, id uuid.UUID) error
	GetWarehouse(ctx context.Context, companyID, id uuid.UUID) (*domain.Warehouse, error)
	ListWarehouses(ctx context.Context, filter *WarehouseFilter) ([]domain.Warehouse, int64, error)
}

// productService implements ProductService.
type productService struct {
	repo repository.ProductRepository
}

// NewProductService creates a ProductService.
func NewProductService(repo repository.ProductRepository) ProductService {
	return &productService{repo: repo}
}

// ---------------------------------------------------------------------------
// Products
// ---------------------------------------------------------------------------

// Create validates and inserts a product.
func (s *productService) Create(ctx context.Context, product *domain.Product) error {
	product.Code = strings.TrimSpace(product.Code)
	product.Name = strings.TrimSpace(product.Name)
	product.Barcode = strings.TrimSpace(product.Barcode)

	if err := product.Validate(); err != nil {
		return err
	}
	if err := s.checkCategory(ctx, product.CompanyID, product.CategoryID); err != nil {
		return err
	}

	// Pre-checked here for a clean 409 with a message the user can act on. The
	// partial unique indexes in 000022 are the actual guarantee - two concurrent
	// creates both pass this check and the second INSERT is refused by the
	// database, which is the correct outcome.
	exists, err := s.repo.ExistsByCode(ctx, product.CompanyID, product.Code, nil)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrProductCodeExists
	}

	if product.Barcode != "" {
		exists, err = s.repo.ExistsByBarcode(ctx, product.CompanyID, product.Barcode, nil)
		if err != nil {
			return err
		}
		if exists {
			return domain.ErrProductBarcodeExists
		}
	}

	return s.repo.Create(ctx, product)
}

// Update validates and saves a product.
func (s *productService) Update(ctx context.Context, product *domain.Product) error {
	product.Code = strings.TrimSpace(product.Code)
	product.Name = strings.TrimSpace(product.Name)
	product.Barcode = strings.TrimSpace(product.Barcode)

	if err := product.Validate(); err != nil {
		return err
	}
	if err := s.checkCategory(ctx, product.CompanyID, product.CategoryID); err != nil {
		return err
	}

	if _, err := s.repo.GetByID(ctx, product.CompanyID, product.ID); err != nil {
		return err
	}

	exists, err := s.repo.ExistsByCode(ctx, product.CompanyID, product.Code, &product.ID)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrProductCodeExists
	}

	if product.Barcode != "" {
		exists, err = s.repo.ExistsByBarcode(ctx, product.CompanyID, product.Barcode, &product.ID)
		if err != nil {
			return err
		}
		if exists {
			return domain.ErrProductBarcodeExists
		}
	}

	return s.repo.Update(ctx, product)
}

// checkCategory rejects a category id that does not belong to the tenant.
//
// Without it a caller could attach one tenant's product to another tenant's
// category: the foreign key only checks that the row exists, and the RLS policy
// on product_categories does not apply to a value the application merely stores.
func (s *productService) checkCategory(ctx context.Context, companyID uuid.UUID, categoryID *uuid.UUID) error {
	if categoryID == nil {
		return nil
	}
	if _, err := s.repo.GetCategoryByID(ctx, companyID, *categoryID); err != nil {
		return domain.ErrCategoryNotFound
	}
	return nil
}

// Delete removes a product that carries no history.
func (s *productService) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	canDelete, _, err := s.CanDelete(ctx, companyID, id)
	if err != nil {
		return err
	}
	if !canDelete {
		return domain.ErrProductInUse
	}
	return s.repo.Delete(ctx, companyID, id)
}

// GetByID retrieves one product.
func (s *productService) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Product, error) {
	return s.repo.GetByID(ctx, companyID, id)
}

// GetByCode retrieves one product by code.
func (s *productService) GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Product, error) {
	return s.repo.GetByCode(ctx, companyID, code)
}

// List retrieves a page of products.
func (s *productService) List(ctx context.Context, filter *ProductFilter) ([]domain.Product, int64, error) {
	return s.repo.List(ctx, filter)
}

// SetActive flips is_active for a set of products.
func (s *productService) SetActive(ctx context.Context, companyID uuid.UUID, ids []uuid.UUID, isActive bool) error {
	return s.repo.UpdateActiveStatus(ctx, companyID, ids, isActive)
}

// CanDelete reports whether a product may be removed, and why not.
func (s *productService) CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error) {
	if _, err := s.repo.GetByID(ctx, companyID, id); err != nil {
		return false, "", err
	}

	hasStock, err := s.repo.HasStock(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}
	if hasStock {
		return false, "product still holds stock", nil
	}

	hasMovements, err := s.repo.HasMovements(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}
	if hasMovements {
		return false, "product appears in the stock ledger", nil
	}

	hasLines, err := s.repo.HasOrderLines(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}
	if hasLines {
		return false, "product appears on a purchase or sales order", nil
	}

	return true, "", nil
}

// GetStats returns the product counters.
func (s *productService) GetStats(ctx context.Context, companyID uuid.UUID) (*ProductStats, error) {
	total, err := s.repo.CountProducts(ctx, companyID, false)
	if err != nil {
		return nil, err
	}
	active, err := s.repo.CountProducts(ctx, companyID, true)
	if err != nil {
		return nil, err
	}
	return &ProductStats{
		TotalCount:    total,
		ActiveCount:   active,
		InactiveCount: total - active,
	}, nil
}

// ---------------------------------------------------------------------------
// Categories
// ---------------------------------------------------------------------------

// CreateCategory validates and inserts a category.
func (s *productService) CreateCategory(ctx context.Context, category *domain.ProductCategory) error {
	category.Code = strings.TrimSpace(category.Code)
	category.Name = strings.TrimSpace(category.Name)

	if category.Code == "" || category.Name == "" {
		return domain.ErrCategoryNotFound
	}

	if err := s.resolveCategoryLevel(ctx, category); err != nil {
		return err
	}

	exists, err := s.repo.ExistsCategoryByCode(ctx, category.CompanyID, category.Code, nil)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrCategoryCodeExists
	}

	return s.repo.CreateCategory(ctx, category)
}

// UpdateCategory validates and saves a category.
func (s *productService) UpdateCategory(ctx context.Context, category *domain.ProductCategory) error {
	category.Code = strings.TrimSpace(category.Code)
	category.Name = strings.TrimSpace(category.Name)

	if _, err := s.repo.GetCategoryByID(ctx, category.CompanyID, category.ID); err != nil {
		return err
	}

	// A category that is its own parent, or its own grandparent, makes the tree
	// a ring: every walk over it runs forever. The database CHECK only catches
	// the direct case, so the walk below catches the rest.
	if category.ParentID != nil {
		if *category.ParentID == category.ID {
			return domain.ErrCategoryParentCycle
		}
		if err := s.assertNoCycle(ctx, category.CompanyID, category.ID, *category.ParentID); err != nil {
			return err
		}
	}

	if err := s.resolveCategoryLevel(ctx, category); err != nil {
		return err
	}

	exists, err := s.repo.ExistsCategoryByCode(ctx, category.CompanyID, category.Code, &category.ID)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrCategoryCodeExists
	}

	return s.repo.UpdateCategory(ctx, category)
}

// maxCategoryDepth bounds the ancestor walk. A tree deeper than this is a bug,
// and the bound means a ring introduced by some other writer cannot hang a
// request even if assertNoCycle is somehow bypassed.
const maxCategoryDepth = 32

// assertNoCycle refuses a parent that already has id among its ancestors.
func (s *productService) assertNoCycle(ctx context.Context, companyID, id, parentID uuid.UUID) error {
	cursor := &parentID
	for depth := 0; cursor != nil && depth < maxCategoryDepth; depth++ {
		if *cursor == id {
			return domain.ErrCategoryParentCycle
		}
		parent, err := s.repo.GetCategoryByID(ctx, companyID, *cursor)
		if err != nil {
			return domain.ErrCategoryParentMissing
		}
		cursor = parent.ParentID
	}
	if cursor != nil {
		return domain.ErrCategoryParentCycle
	}
	return nil
}

// resolveCategoryLevel derives the depth from the parent rather than trusting
// the client, so the tree cannot claim a level its position does not support.
func (s *productService) resolveCategoryLevel(ctx context.Context, category *domain.ProductCategory) error {
	if category.ParentID == nil {
		category.Level = 1
		return nil
	}
	parent, err := s.repo.GetCategoryByID(ctx, category.CompanyID, *category.ParentID)
	if err != nil {
		return domain.ErrCategoryParentMissing
	}
	category.Level = parent.Level + 1
	return nil
}

// DeleteCategory removes a leaf category that classifies nothing.
func (s *productService) DeleteCategory(ctx context.Context, companyID, id uuid.UUID) error {
	if _, err := s.repo.GetCategoryByID(ctx, companyID, id); err != nil {
		return err
	}

	hasChildren, err := s.repo.CategoryHasChildren(ctx, companyID, id)
	if err != nil {
		return err
	}
	if hasChildren {
		return domain.ErrCategoryHasChildren
	}

	hasProducts, err := s.repo.CategoryHasProducts(ctx, companyID, id)
	if err != nil {
		return err
	}
	if hasProducts {
		return domain.ErrCategoryInUse
	}

	return s.repo.DeleteCategory(ctx, companyID, id)
}

// GetCategory retrieves one category.
func (s *productService) GetCategory(ctx context.Context, companyID, id uuid.UUID) (*domain.ProductCategory, error) {
	return s.repo.GetCategoryByID(ctx, companyID, id)
}

// ListCategories retrieves a page of categories.
func (s *productService) ListCategories(ctx context.Context, filter *CategoryFilter) ([]domain.ProductCategory, int64, error) {
	return s.repo.ListCategories(ctx, filter)
}

// ---------------------------------------------------------------------------
// Warehouses
// ---------------------------------------------------------------------------

// CreateWarehouse validates and inserts a warehouse.
func (s *productService) CreateWarehouse(ctx context.Context, warehouse *domain.Warehouse) error {
	warehouse.Code = strings.TrimSpace(warehouse.Code)
	warehouse.Name = strings.TrimSpace(warehouse.Name)

	if warehouse.Code == "" || warehouse.Name == "" {
		return domain.ErrWarehouseNotFound
	}

	exists, err := s.repo.ExistsWarehouseByCode(ctx, warehouse.CompanyID, warehouse.Code, nil)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrWarehouseCodeExists
	}

	if !warehouse.IsDefault {
		return s.repo.CreateWarehouse(ctx, warehouse)
	}

	// Promoting to default has to clear the incumbent in the same transaction,
	// or uq_warehouses_one_default rejects the INSERT.
	return s.repo.WithTransaction(ctx, func(repo repository.ProductRepository) error {
		if err := repo.ClearDefaultWarehouse(ctx, warehouse.CompanyID, nil); err != nil {
			return err
		}
		return repo.CreateWarehouse(ctx, warehouse)
	})
}

// UpdateWarehouse validates and saves a warehouse.
func (s *productService) UpdateWarehouse(ctx context.Context, warehouse *domain.Warehouse) error {
	warehouse.Code = strings.TrimSpace(warehouse.Code)
	warehouse.Name = strings.TrimSpace(warehouse.Name)

	if _, err := s.repo.GetWarehouseByID(ctx, warehouse.CompanyID, warehouse.ID); err != nil {
		return err
	}

	exists, err := s.repo.ExistsWarehouseByCode(ctx, warehouse.CompanyID, warehouse.Code, &warehouse.ID)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrWarehouseCodeExists
	}

	if !warehouse.IsDefault {
		return s.repo.UpdateWarehouse(ctx, warehouse)
	}

	return s.repo.WithTransaction(ctx, func(repo repository.ProductRepository) error {
		if err := repo.ClearDefaultWarehouse(ctx, warehouse.CompanyID, &warehouse.ID); err != nil {
			return err
		}
		return repo.UpdateWarehouse(ctx, warehouse)
	})
}

// DeleteWarehouse removes a warehouse that holds nothing and is referenced by
// no order.
func (s *productService) DeleteWarehouse(ctx context.Context, companyID, id uuid.UUID) error {
	if _, err := s.repo.GetWarehouseByID(ctx, companyID, id); err != nil {
		return err
	}

	hasStock, err := s.repo.WarehouseHasStock(ctx, companyID, id)
	if err != nil {
		return err
	}
	if hasStock {
		return domain.ErrWarehouseInUse
	}

	hasOrders, err := s.repo.WarehouseHasOrders(ctx, companyID, id)
	if err != nil {
		return err
	}
	if hasOrders {
		return domain.ErrWarehouseInUse
	}

	return s.repo.DeleteWarehouse(ctx, companyID, id)
}

// GetWarehouse retrieves one warehouse.
func (s *productService) GetWarehouse(ctx context.Context, companyID, id uuid.UUID) (*domain.Warehouse, error) {
	return s.repo.GetWarehouseByID(ctx, companyID, id)
}

// ListWarehouses retrieves a page of warehouses.
func (s *productService) ListWarehouses(ctx context.Context, filter *WarehouseFilter) ([]domain.Warehouse, int64, error) {
	return s.repo.ListWarehouses(ctx, filter)
}
