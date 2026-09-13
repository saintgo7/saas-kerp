package dto

import (
	"time"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// The wire format is snake_case, as everywhere else in this API; the React app
// converts to camelCase in its api layer (web/src/api/*.ts). Field names are
// otherwise the snake_case form of the interfaces in web/src/types/inventory.ts.

// The inventory responses format every timestamp through these two helpers,
// using the layouts already declared in tax_invoice_dto.go, so that the same
// field cannot come back rendered two ways from two endpoints.

// formatTime renders an instant for the wire, or "" for the zero time.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(timestampLayout)
}

// formatDate renders a calendar date. Order dates are DATE columns, and sending
// them with a time-of-day makes a client in a behind-UTC timezone render the
// previous day.
func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(dateLayout)
}

// ---------------------------------------------------------------------------
// Product category
// ---------------------------------------------------------------------------

// ProductCategoryResponse is a product category in API responses.
type ProductCategoryResponse struct {
	ID        string `json:"id"`
	CompanyID string `json:"company_id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	ParentID  string `json:"parent_id,omitempty"`
	Level     int    `json:"level"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// FromProductCategory converts a domain category to its response form.
func FromProductCategory(c *domain.ProductCategory) ProductCategoryResponse {
	resp := ProductCategoryResponse{
		ID:        c.ID.String(),
		CompanyID: c.CompanyID.String(),
		Code:      c.Code,
		Name:      c.Name,
		Level:     c.Level,
		IsActive:  c.IsActive,
		CreatedAt: formatTime(c.CreatedAt),
		UpdatedAt: formatTime(c.UpdatedAt),
	}
	if c.ParentID != nil {
		resp.ParentID = c.ParentID.String()
	}
	return resp
}

// FromProductCategories converts a slice of categories.
func FromProductCategories(categories []domain.ProductCategory) []ProductCategoryResponse {
	out := make([]ProductCategoryResponse, len(categories))
	for i := range categories {
		out[i] = FromProductCategory(&categories[i])
	}
	return out
}

// CreateProductCategoryRequest is the body of a category create.
type CreateProductCategoryRequest struct {
	Code     string `json:"code" binding:"required,max=20"`
	Name     string `json:"name" binding:"required,max=100"`
	ParentID string `json:"parent_id,omitempty" binding:"omitempty,uuid"`
	IsActive *bool  `json:"is_active,omitempty"`
}

// UpdateProductCategoryRequest is the body of a category update.
type UpdateProductCategoryRequest struct {
	Code     string `json:"code" binding:"required,max=20"`
	Name     string `json:"name" binding:"required,max=100"`
	ParentID string `json:"parent_id,omitempty" binding:"omitempty,uuid"`
	IsActive *bool  `json:"is_active,omitempty"`
}

// ---------------------------------------------------------------------------
// Warehouse
// ---------------------------------------------------------------------------

// WarehouseResponse is a warehouse in API responses.
type WarehouseResponse struct {
	ID        string `json:"id"`
	CompanyID string `json:"company_id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Address   string `json:"address,omitempty"`
	Manager   string `json:"manager,omitempty"`
	Phone     string `json:"phone,omitempty"`
	IsDefault bool   `json:"is_default"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// FromWarehouse converts a domain warehouse to its response form.
func FromWarehouse(w *domain.Warehouse) WarehouseResponse {
	return WarehouseResponse{
		ID:        w.ID.String(),
		CompanyID: w.CompanyID.String(),
		Code:      w.Code,
		Name:      w.Name,
		Address:   w.Address,
		Manager:   w.Manager,
		Phone:     w.Phone,
		IsDefault: w.IsDefault,
		IsActive:  w.IsActive,
		CreatedAt: formatTime(w.CreatedAt),
		UpdatedAt: formatTime(w.UpdatedAt),
	}
}

// FromWarehouses converts a slice of warehouses.
func FromWarehouses(warehouses []domain.Warehouse) []WarehouseResponse {
	out := make([]WarehouseResponse, len(warehouses))
	for i := range warehouses {
		out[i] = FromWarehouse(&warehouses[i])
	}
	return out
}

// CreateWarehouseRequest is the body of a warehouse create.
type CreateWarehouseRequest struct {
	Code      string `json:"code" binding:"required,max=20"`
	Name      string `json:"name" binding:"required,max=100"`
	Address   string `json:"address,omitempty" binding:"max=200"`
	Manager   string `json:"manager,omitempty" binding:"max=50"`
	Phone     string `json:"phone,omitempty" binding:"max=20"`
	IsDefault *bool  `json:"is_default,omitempty"`
	IsActive  *bool  `json:"is_active,omitempty"`
}

// UpdateWarehouseRequest is the body of a warehouse update.
type UpdateWarehouseRequest struct {
	Code      string `json:"code" binding:"required,max=20"`
	Name      string `json:"name" binding:"required,max=100"`
	Address   string `json:"address,omitempty" binding:"max=200"`
	Manager   string `json:"manager,omitempty" binding:"max=50"`
	Phone     string `json:"phone,omitempty" binding:"max=20"`
	IsDefault *bool  `json:"is_default,omitempty"`
	IsActive  *bool  `json:"is_active,omitempty"`
}

// ---------------------------------------------------------------------------
// Product
// ---------------------------------------------------------------------------

// ProductResponse is a product in API responses.
type ProductResponse struct {
	ID            string                   `json:"id"`
	CompanyID     string                   `json:"company_id"`
	Code          string                   `json:"code"`
	Name          string                   `json:"name"`
	CategoryID    string                   `json:"category_id,omitempty"`
	Category      *ProductCategoryResponse `json:"category,omitempty"`
	Specification string                   `json:"specification,omitempty"`
	Unit          string                   `json:"unit"`
	UnitPrice     float64                  `json:"unit_price"`
	CostPrice     float64                  `json:"cost_price"`
	MinStock      *float64                 `json:"min_stock,omitempty"`
	MaxStock      *float64                 `json:"max_stock,omitempty"`
	IsActive      bool                     `json:"is_active"`
	Description   string                   `json:"description,omitempty"`
	Barcode       string                   `json:"barcode,omitempty"`
	ImageURL      string                   `json:"image_url,omitempty"`
	CreatedAt     string                   `json:"created_at"`
	UpdatedAt     string                   `json:"updated_at"`
}

// FromProduct converts a domain product to its response form.
//
// Category is embedded rather than left to the client to resolve: the product
// list renders a category name on every row, and with only category_id the
// column would be blank until the client made a second request per row.
func FromProduct(p *domain.Product) ProductResponse {
	resp := ProductResponse{
		ID:            p.ID.String(),
		CompanyID:     p.CompanyID.String(),
		Code:          p.Code,
		Name:          p.Name,
		Specification: p.Specification,
		Unit:          p.Unit,
		UnitPrice:     p.UnitPrice,
		CostPrice:     p.CostPrice,
		MinStock:      p.MinStock,
		MaxStock:      p.MaxStock,
		IsActive:      p.IsActive,
		Description:   p.Description,
		Barcode:       p.Barcode,
		ImageURL:      p.ImageURL,
		CreatedAt:     formatTime(p.CreatedAt),
		UpdatedAt:     formatTime(p.UpdatedAt),
	}
	if p.CategoryID != nil {
		resp.CategoryID = p.CategoryID.String()
	}
	if p.Category != nil {
		category := FromProductCategory(p.Category)
		resp.Category = &category
	}
	return resp
}

// FromProducts converts a slice of products.
func FromProducts(products []domain.Product) []ProductResponse {
	out := make([]ProductResponse, len(products))
	for i := range products {
		out[i] = FromProduct(&products[i])
	}
	return out
}

// CreateProductRequest is the body of a product create.
//
// The unit is constrained to the same twelve values the form offers
// (PRODUCT_UNITS); `oneof` here is what stops "ea" and "EA" becoming two units.
type CreateProductRequest struct {
	Code          string   `json:"code" binding:"required,max=50"`
	Name          string   `json:"name" binding:"required,max=200"`
	CategoryID    string   `json:"category_id,omitempty" binding:"omitempty,uuid"`
	Specification string   `json:"specification,omitempty" binding:"max=200"`
	Unit          string   `json:"unit" binding:"required,oneof=EA BOX SET KG G L ML M CM PACK ROLL SHEET"`
	UnitPrice     float64  `json:"unit_price" binding:"gte=0"`
	CostPrice     float64  `json:"cost_price" binding:"gte=0"`
	MinStock      *float64 `json:"min_stock,omitempty" binding:"omitempty,gte=0"`
	MaxStock      *float64 `json:"max_stock,omitempty" binding:"omitempty,gte=0"`
	IsActive      *bool    `json:"is_active,omitempty"`
	Description   string   `json:"description,omitempty" binding:"max=500"`
	Barcode       string   `json:"barcode,omitempty" binding:"max=50"`
	ImageURL      string   `json:"image_url,omitempty" binding:"max=500"`
}

// UpdateProductRequest is the body of a product update.
type UpdateProductRequest struct {
	Code          string   `json:"code" binding:"required,max=50"`
	Name          string   `json:"name" binding:"required,max=200"`
	CategoryID    string   `json:"category_id,omitempty" binding:"omitempty,uuid"`
	Specification string   `json:"specification,omitempty" binding:"max=200"`
	Unit          string   `json:"unit" binding:"required,oneof=EA BOX SET KG G L ML M CM PACK ROLL SHEET"`
	UnitPrice     float64  `json:"unit_price" binding:"gte=0"`
	CostPrice     float64  `json:"cost_price" binding:"gte=0"`
	MinStock      *float64 `json:"min_stock,omitempty" binding:"omitempty,gte=0"`
	MaxStock      *float64 `json:"max_stock,omitempty" binding:"omitempty,gte=0"`
	IsActive      *bool    `json:"is_active,omitempty"`
	Description   string   `json:"description,omitempty" binding:"max=500"`
	Barcode       string   `json:"barcode,omitempty" binding:"max=50"`
	ImageURL      string   `json:"image_url,omitempty" binding:"max=500"`
}

// ProductStatsResponse is the product master-data counter card.
type ProductStatsResponse struct {
	TotalCount    int64 `json:"total_count"`
	ActiveCount   int64 `json:"active_count"`
	InactiveCount int64 `json:"inactive_count"`
}
