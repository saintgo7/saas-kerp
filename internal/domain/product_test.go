package domain_test

import (
	"testing"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

func validProduct() *domain.Product {
	return &domain.Product{
		Code:      "P001",
		Name:      "볼트 M6",
		Unit:      "EA",
		UnitPrice: 1000,
		CostPrice: 700,
	}
}

func TestProductValidate(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*domain.Product)
		wantErr error
		wantAny bool // an error is required but its identity is not asserted
	}{
		{name: "a well-formed product passes", mutate: func(*domain.Product) {}},
		{
			name:    "the code is required",
			mutate:  func(p *domain.Product) { p.Code = "   " },
			wantAny: true,
		},
		{
			name:    "the name is required",
			mutate:  func(p *domain.Product) { p.Name = "" },
			wantAny: true,
		},
		{
			// The unit set is closed so that "ea" and "EA" cannot become two
			// units for the same article.
			name:    "an unknown unit is rejected",
			mutate:  func(p *domain.Product) { p.Unit = "ea" },
			wantErr: domain.ErrProductInvalidUnit,
		},
		{
			name:    "a negative unit price is rejected",
			mutate:  func(p *domain.Product) { p.UnitPrice = -1 },
			wantErr: domain.ErrProductNegativeAmount,
		},
		{
			name:    "a negative cost price is rejected",
			mutate:  func(p *domain.Product) { p.CostPrice = -0.01 },
			wantErr: domain.ErrProductNegativeAmount,
		},
		{
			name: "max stock below min stock is rejected",
			mutate: func(p *domain.Product) {
				p.MinStock = f(100)
				p.MaxStock = f(10)
			},
			wantErr: domain.ErrProductStockRange,
		},
		{
			name: "equal min and max stock is allowed",
			mutate: func(p *domain.Product) {
				p.MinStock = f(10)
				p.MaxStock = f(10)
			},
		},
		{
			// Only one threshold set is a normal configuration and must not
			// trip the range check.
			name:   "min stock alone is allowed",
			mutate: func(p *domain.Product) { p.MinStock = f(10) },
		},
		{
			name:   "max stock alone is allowed",
			mutate: func(p *domain.Product) { p.MaxStock = f(10) },
		},
		{
			name:    "a negative reorder point is rejected",
			mutate:  func(p *domain.Product) { p.MinStock = f(-1) },
			wantErr: domain.ErrProductNegativeAmount,
		},
		{
			// Zero is a legitimate reorder point and must be distinguishable
			// from "no reorder point"; both are accepted here and the two are
			// told apart by the pointer being nil.
			name:   "a zero reorder point is allowed",
			mutate: func(p *domain.Product) { p.MinStock = f(0) },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			product := validProduct()
			tc.mutate(product)

			err := product.Validate()
			switch {
			case tc.wantErr != nil:
				if err != tc.wantErr {
					t.Errorf("Validate() = %v, want %v", err, tc.wantErr)
				}
			case tc.wantAny:
				if err == nil {
					t.Error("Validate() = nil, want an error")
				}
			default:
				if err != nil {
					t.Errorf("Validate() = %v, want nil", err)
				}
			}
		})
	}
}

func TestProductUnits(t *testing.T) {
	// The set must match PRODUCT_UNITS in web/src/constants/index.ts, which is
	// what the form's dropdown offers. A unit the form can produce but the API
	// rejects is a form that cannot be submitted.
	frontendUnits := []string{
		"EA", "BOX", "SET", "KG", "G", "L", "ML", "M", "CM", "PACK", "ROLL", "SHEET",
	}
	if len(domain.ProductUnits) != len(frontendUnits) {
		t.Fatalf("the unit set has %d entries, the frontend offers %d",
			len(domain.ProductUnits), len(frontendUnits))
	}
	for _, unit := range frontendUnits {
		if !domain.IsValidProductUnit(unit) {
			t.Errorf("the frontend offers %q but the API rejects it", unit)
		}
	}
	if domain.IsValidProductUnit("") || domain.IsValidProductUnit("TON") {
		t.Error("an unlisted unit was accepted")
	}
}

func TestProductTableNames(t *testing.T) {
	// The table names are what the RLS policies in 000022 are attached to; a
	// rename here without a migration would silently query a table with no
	// policy on it.
	if got := (domain.Product{}).TableName(); got != "products" {
		t.Errorf("Product.TableName() = %q", got)
	}
	if got := (domain.ProductCategory{}).TableName(); got != "product_categories" {
		t.Errorf("ProductCategory.TableName() = %q", got)
	}
	if got := (domain.Warehouse{}).TableName(); got != "warehouses" {
		t.Errorf("Warehouse.TableName() = %q", got)
	}
	if got := (domain.Stock{}).TableName(); got != "stocks" {
		t.Errorf("Stock.TableName() = %q", got)
	}
	if got := (domain.StockMovement{}).TableName(); got != "stock_movements" {
		t.Errorf("StockMovement.TableName() = %q", got)
	}
	if got := (domain.PurchaseOrder{}).TableName(); got != "purchase_orders" {
		t.Errorf("PurchaseOrder.TableName() = %q", got)
	}
	if got := (domain.PurchaseOrderItem{}).TableName(); got != "purchase_order_items" {
		t.Errorf("PurchaseOrderItem.TableName() = %q", got)
	}
	if got := (domain.SalesOrder{}).TableName(); got != "sales_orders" {
		t.Errorf("SalesOrder.TableName() = %q", got)
	}
	if got := (domain.SalesOrderItem{}).TableName(); got != "sales_order_items" {
		t.Errorf("SalesOrderItem.TableName() = %q", got)
	}
}
