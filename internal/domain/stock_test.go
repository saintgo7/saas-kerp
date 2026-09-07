package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// Direction is carried by the movement type alone; quantity is always positive.
// If a type were ever misclassified, a receipt would silently issue stock, so
// the whole table is asserted rather than a sample.
func TestMovementDirection(t *testing.T) {
	inbound := []domain.StockMovementType{
		domain.MovementPurchaseIn, domain.MovementAdjustmentIn,
		domain.MovementTransferIn, domain.MovementReturnIn,
	}
	outbound := []domain.StockMovementType{
		domain.MovementSalesOut, domain.MovementAdjustmentOut,
		domain.MovementTransferOut, domain.MovementReturnOut,
	}

	for _, mt := range inbound {
		if !mt.IsInbound() {
			t.Errorf("%q should be inbound", mt)
		}
		if got := mt.SignedDelta(5); got != 5 {
			t.Errorf("SignedDelta(%q, 5) = %v, want 5", mt, got)
		}
	}
	for _, mt := range outbound {
		if mt.IsInbound() {
			t.Errorf("%q should be outbound", mt)
		}
		if got := mt.SignedDelta(5); got != -5 {
			t.Errorf("SignedDelta(%q, 5) = %v, want -5", mt, got)
		}
	}

	if len(inbound)+len(outbound) != 8 {
		t.Fatal("the movement type table has changed; update this test")
	}
}

func TestMovementTypeValidity(t *testing.T) {
	if domain.StockMovementType("purchase_out").IsValid() {
		t.Error("an unknown movement type was accepted")
	}
	if !domain.MovementPurchaseIn.IsValid() {
		t.Error("purchase_in should be valid")
	}
}

// newStock builds a balance with a product attached, which is what the alert
// derivation needs.
func newStock(quantity float64, minStock, maxStock *float64) *domain.Stock {
	return &domain.Stock{
		TenantModel: domain.TenantModel{
			BaseModel: domain.BaseModel{ID: uuid.New()},
			CompanyID: uuid.New(),
		},
		ProductID:     uuid.New(),
		WarehouseID:   uuid.New(),
		Quantity:      quantity,
		LastUpdatedAt: time.Now(),
		Product: &domain.Product{
			MinStock:  minStock,
			MaxStock:  maxStock,
			CostPrice: 700,
		},
	}
}

func f(v float64) *float64 { return &v }

func TestDeriveStockAlert(t *testing.T) {
	cases := []struct {
		name     string
		stock    *domain.Stock
		wantType domain.StockAlertType
		wantNone bool
	}{
		{
			// An empty balance is out_of_stock, not low_stock, even though zero
			// is also below every non-zero reorder point. They are different
			// operational situations and the UI colours them differently.
			name:     "empty balance is out of stock, not low stock",
			stock:    newStock(0, f(10), nil),
			wantType: domain.AlertOutOfStock,
		},
		{
			name:     "at the reorder point is low stock",
			stock:    newStock(10, f(10), nil),
			wantType: domain.AlertLowStock,
		},
		{
			name:     "below the reorder point is low stock",
			stock:    newStock(3, f(10), nil),
			wantType: domain.AlertLowStock,
		},
		{
			name:     "above the maximum is overstock",
			stock:    newStock(500, f(10), f(100)),
			wantType: domain.AlertOverstock,
		},
		{
			name:     "exactly at the maximum is not overstock",
			stock:    newStock(100, f(10), f(100)),
			wantNone: true,
		},
		{
			name:     "within thresholds raises nothing",
			stock:    newStock(50, f(10), f(100)),
			wantNone: true,
		},
		{
			// A product with no reorder point set must never raise low_stock.
			// NULL min_stock is "no threshold", not "threshold of zero"; the
			// same distinction the repository query relies on.
			name:     "no reorder point means no low-stock alert",
			stock:    newStock(1, nil, nil),
			wantNone: true,
		},
		{
			// ...but an empty balance is still reported, threshold or not.
			name:     "no reorder point still reports an empty balance",
			stock:    newStock(0, nil, nil),
			wantType: domain.AlertOutOfStock,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			alert := domain.DeriveStockAlert(tc.stock)
			if tc.wantNone {
				if alert != nil {
					t.Fatalf("expected no alert, got %q", alert.AlertType)
				}
				return
			}
			if alert == nil {
				t.Fatalf("expected a %q alert, got none", tc.wantType)
			}
			if alert.AlertType != tc.wantType {
				t.Errorf("alert type = %q, want %q", alert.AlertType, tc.wantType)
			}
			if alert.ID != tc.stock.ID {
				t.Error("a derived alert must carry the stock row's id, so the UI has a stable list key")
			}
			if alert.IsRead {
				t.Error("a derived alert can never be read; there is nowhere to record that")
			}
		})
	}
}

func TestDeriveStockAlertWithoutProduct(t *testing.T) {
	// The repository always preloads the product, but a caller that forgets to
	// must get nil rather than a nil-pointer dereference in a list handler.
	stock := &domain.Stock{Quantity: 0}
	if alert := domain.DeriveStockAlert(stock); alert != nil {
		t.Error("a balance with no product loaded must not derive an alert")
	}
	if domain.DeriveStockAlert(nil) != nil {
		t.Error("nil must derive no alert")
	}
}

func TestIsBelowMinStock(t *testing.T) {
	if newStock(1, nil, nil).IsBelowMinStock() {
		t.Error("a product with no reorder point is never below it")
	}
	if !newStock(10, f(10), nil).IsBelowMinStock() {
		t.Error("at the reorder point counts as below")
	}
	if newStock(11, f(10), nil).IsBelowMinStock() {
		t.Error("above the reorder point is not below it")
	}

	// The same nil-safety the list serialiser depends on.
	noProduct := &domain.Stock{Quantity: 0}
	if noProduct.IsBelowMinStock() {
		t.Error("a balance with no product loaded must not report low stock")
	}
	if got := noProduct.StockValue(); got != 0 {
		t.Errorf("StockValue with no product = %v, want 0", got)
	}
}

func TestStockValue(t *testing.T) {
	stock := newStock(12, nil, nil)
	if got := stock.StockValue(); got != 12*700 {
		t.Errorf("StockValue = %v, want %v", got, 12*700)
	}
}
