package domain

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Inventory master-data errors.
var (
	ErrProductNotFound       = errors.New("product not found")
	ErrProductCodeExists     = errors.New("product code already exists")
	ErrProductBarcodeExists  = errors.New("product barcode already exists")
	ErrProductInactive       = errors.New("product is not active")
	ErrProductInUse          = errors.New("product has stock or order history and cannot be deleted")
	ErrProductInvalidUnit    = errors.New("invalid product unit")
	ErrProductStockRange     = errors.New("max stock must not be lower than min stock")
	ErrProductNegativeAmount = errors.New("product prices must not be negative")

	ErrCategoryNotFound      = errors.New("product category not found")
	ErrCategoryCodeExists    = errors.New("product category code already exists")
	ErrCategoryHasChildren   = errors.New("product category has child categories")
	ErrCategoryInUse         = errors.New("product category is referenced by products")
	ErrCategoryParentCycle   = errors.New("product category cannot be its own ancestor")
	ErrCategoryParentMissing = errors.New("parent product category not found")

	ErrWarehouseNotFound   = errors.New("warehouse not found")
	ErrWarehouseCodeExists = errors.New("warehouse code already exists")
	ErrWarehouseInUse      = errors.New("warehouse has stock or order history and cannot be deleted")
)

// ProductUnits is the closed set of units of measure the UI offers
// (web/src/constants/index.ts PRODUCT_UNITS). Keeping it closed here means a
// typo cannot create a second, silently different unit for the same article.
var ProductUnits = []string{
	"EA", "BOX", "SET", "KG", "G", "L", "ML", "M", "CM", "PACK", "ROLL", "SHEET",
}

// IsValidProductUnit reports whether unit is one of ProductUnits.
func IsValidProductUnit(unit string) bool {
	for _, u := range ProductUnits {
		if u == unit {
			return true
		}
	}
	return false
}

// ProductCategory is a node in the product classification tree (품목 분류).
type ProductCategory struct {
	TenantModel

	// Soft delete, for the same reason Partner has one: a category that has
	// ever classified a product must not vanish from history.
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Code string `gorm:"type:varchar(20);not null" json:"code"`
	Name string `gorm:"type:varchar(100);not null" json:"name"`

	ParentID *uuid.UUID `gorm:"type:uuid" json:"parent_id,omitempty"`
	Level    int        `gorm:"not null;default:1" json:"level"`

	IsActive bool `gorm:"not null;default:true" json:"is_active"`
}

// TableName specifies the table name for GORM.
func (ProductCategory) TableName() string {
	return "product_categories"
}

// Warehouse is a stock location (창고).
type Warehouse struct {
	TenantModel

	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Code string `gorm:"type:varchar(20);not null" json:"code"`
	Name string `gorm:"type:varchar(100);not null" json:"name"`

	Address string `gorm:"type:varchar(200)" json:"address,omitempty"`
	Manager string `gorm:"type:varchar(50)" json:"manager,omitempty"`
	Phone   string `gorm:"type:varchar(20)" json:"phone,omitempty"`

	// At most one warehouse per tenant may carry this flag; the database
	// enforces it with the partial unique index uq_warehouses_one_default, so
	// setting a new default must clear the old one in the same transaction.
	IsDefault bool `gorm:"not null;default:false" json:"is_default"`
	IsActive  bool `gorm:"not null;default:true" json:"is_active"`
}

// TableName specifies the table name for GORM.
func (Warehouse) TableName() string {
	return "warehouses"
}

// Product is an inventory item (품목).
//
// Money and quantity types follow the tax-invoice precedent in this package:
// prices are NUMERIC(18,2) and may carry sub-won precision, quantities are
// NUMERIC(18,3), and booked amounts (on orders) are whole-won int64.
type Product struct {
	TenantModel

	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Code string `gorm:"type:varchar(50);not null" json:"code"`
	Name string `gorm:"type:varchar(200);not null" json:"name"`

	CategoryID *uuid.UUID       `gorm:"type:uuid" json:"category_id,omitempty"`
	Category   *ProductCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`

	Specification string `gorm:"type:varchar(200)" json:"specification,omitempty"`
	Unit          string `gorm:"type:varchar(20);not null;default:'EA'" json:"unit"`

	UnitPrice float64 `gorm:"type:numeric(18,2);not null;default:0" json:"unit_price"`
	CostPrice float64 `gorm:"type:numeric(18,2);not null;default:0" json:"cost_price"`

	// Pointers, not plain float64: "no reorder point set" is a different fact
	// from "reorder point is zero", and only the first must suppress the
	// low-stock alert for the item.
	MinStock *float64 `gorm:"type:numeric(18,3)" json:"min_stock,omitempty"`
	MaxStock *float64 `gorm:"type:numeric(18,3)" json:"max_stock,omitempty"`

	IsActive    bool   `gorm:"not null;default:true" json:"is_active"`
	Description string `gorm:"type:text" json:"description,omitempty"`
	Barcode     string `gorm:"type:varchar(50)" json:"barcode,omitempty"`
	ImageURL    string `gorm:"type:varchar(500)" json:"image_url,omitempty"`
}

// TableName specifies the table name for GORM.
func (Product) TableName() string {
	return "products"
}

// Validate checks the invariants that do not need a database round trip.
// Uniqueness of code and barcode is checked by the service, which needs the
// repository, and again by the partial unique indexes in 000022.
func (p *Product) Validate() error {
	if strings.TrimSpace(p.Code) == "" {
		return errors.New("product code is required")
	}
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("product name is required")
	}
	if !IsValidProductUnit(p.Unit) {
		return ErrProductInvalidUnit
	}
	if p.UnitPrice < 0 || p.CostPrice < 0 {
		return ErrProductNegativeAmount
	}
	if p.MinStock != nil && *p.MinStock < 0 {
		return ErrProductNegativeAmount
	}
	if p.MaxStock != nil && *p.MaxStock < 0 {
		return ErrProductNegativeAmount
	}
	if p.MinStock != nil && p.MaxStock != nil && *p.MaxStock < *p.MinStock {
		return ErrProductStockRange
	}
	return nil
}
