package service

// These tests are in `package service` rather than `package service_test`
// because the two things worth pinning down here - how a line is priced and how
// an adjustment turns a target quantity into a movement - are unexported. They
// are also the two places where a quiet arithmetic mistake would be invisible
// until an accountant found it.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// ---------------------------------------------------------------------------
// Line pricing
// ---------------------------------------------------------------------------

func TestPriceLines(t *testing.T) {
	lines, err := priceLines([]OrderLineInput{
		{Quantity: 10, UnitPrice: 1000},  // 10,000
		{Quantity: 3, UnitPrice: 3333.5}, // 10,000.5 -> 10,001 (half rounds up)
	})
	if err != nil {
		t.Fatalf("priceLines: %v", err)
	}

	if lines.amounts[0] != 10000 {
		t.Errorf("line 0 amount = %d, want 10000", lines.amounts[0])
	}
	if lines.amounts[1] != 10001 {
		t.Errorf("line 1 amount = %d, want 10001", lines.amounts[1])
	}
	if lines.totalAmount != 20001 {
		t.Errorf("total = %d, want 20001", lines.totalAmount)
	}

	// The order's VAT is 10% of the total, rounded once: 2000.1 -> 2000.
	if lines.taxAmount != 2000 {
		t.Errorf("order tax = %d, want 2000", lines.taxAmount)
	}

	// The identity chk_purchase_orders_amounts enforces in SQL.
	if lines.grandTotal != lines.totalAmount+lines.taxAmount {
		t.Errorf("grand total %d != total %d + tax %d",
			lines.grandTotal, lines.totalAmount, lines.taxAmount)
	}
}

func TestPriceLinesRejectsBadInput(t *testing.T) {
	if _, err := priceLines(nil); err != domain.ErrOrderNoItems {
		t.Errorf("empty order: got %v, want ErrOrderNoItems", err)
	}
	if _, err := priceLines([]OrderLineInput{{Quantity: 0, UnitPrice: 100}}); err != domain.ErrOrderNothingToPost {
		t.Errorf("zero quantity: got %v, want ErrOrderNothingToPost", err)
	}
	if _, err := priceLines([]OrderLineInput{{Quantity: -1, UnitPrice: 100}}); err != domain.ErrOrderNothingToPost {
		t.Errorf("negative quantity: got %v, want ErrOrderNothingToPost", err)
	}
	if _, err := priceLines([]OrderLineInput{{Quantity: 1, UnitPrice: -100}}); err != domain.ErrOrderNegativeAmount {
		t.Errorf("negative price: got %v, want ErrOrderNegativeAmount", err)
	}
}

// A zero unit price is legitimate - a free sample, a replacement under warranty
// - and must not be confused with a missing one.
func TestPriceLinesAcceptsAZeroUnitPrice(t *testing.T) {
	lines, err := priceLines([]OrderLineInput{{Quantity: 5, UnitPrice: 0}})
	if err != nil {
		t.Fatalf("priceLines: %v", err)
	}
	if lines.totalAmount != 0 || lines.taxAmount != 0 || lines.grandTotal != 0 {
		t.Errorf("free line priced as total=%d tax=%d grand=%d, want all zero",
			lines.totalAmount, lines.taxAmount, lines.grandTotal)
	}
}

// ---------------------------------------------------------------------------
// Stock adjustment
// ---------------------------------------------------------------------------

// The fakes embed the repository interfaces so that only the methods a test
// actually exercises need a body. An unstubbed call panics, which is the right
// outcome: it means the code under test reached for something the test did not
// intend to allow.

type fakeProductRepo struct {
	repository.ProductRepository
	product   *domain.Product
	warehouse *domain.Warehouse
	getErr    error
}

func (f *fakeProductRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*domain.Product, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.product, nil
}

func (f *fakeProductRepo) GetWarehouseByID(context.Context, uuid.UUID, uuid.UUID) (*domain.Warehouse, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.warehouse, nil
}

type fakeStockRepo struct {
	repository.StockRepository
	current  *domain.Stock
	getErr   error
	appliedT []*domain.StockMovement
}

func (f *fakeStockRepo) Get(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*domain.Stock, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.current, nil
}

func (f *fakeStockRepo) ApplyMovement(_ context.Context, m *domain.StockMovement) error {
	f.appliedT = append(f.appliedT, m)
	return nil
}

// WithTransaction runs fn against the same fake, which is what a real one-
// connection transaction amounts to for these tests.
func (f *fakeStockRepo) WithTransaction(ctx context.Context, fn func(repo repository.StockRepository) error) error {
	return fn(f)
}

func newAdjustFixture(currentQuantity float64) (*stockService, *fakeStockRepo) {
	stockRepo := &fakeStockRepo{
		current: &domain.Stock{Quantity: currentQuantity},
	}
	productRepo := &fakeProductRepo{
		product:   &domain.Product{IsActive: true},
		warehouse: &domain.Warehouse{IsActive: true},
	}
	return &stockService{stockRepo: stockRepo, productRepo: productRepo}, stockRepo
}

func TestAdjustComputesTheDeltaAndDirection(t *testing.T) {
	cases := []struct {
		name         string
		current      float64
		target       float64
		wantType     domain.StockMovementType
		wantQuantity float64
	}{
		{"raising the balance books an inbound adjustment", 100, 120, domain.MovementAdjustmentIn, 20},
		{"lowering the balance books an outbound adjustment", 100, 80, domain.MovementAdjustmentOut, 20},
		{"emptying the balance is allowed", 100, 0, domain.MovementAdjustmentOut, 100},
		{"fractional quantities survive", 10.5, 12.25, domain.MovementAdjustmentIn, 1.75},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, stockRepo := newAdjustFixture(tc.current)

			movement, err := svc.Adjust(context.Background(), &AdjustStockInput{
				CompanyID:      uuid.New(),
				ProductID:      uuid.New(),
				WarehouseID:    uuid.New(),
				UserID:         uuid.New(),
				TargetQuantity: tc.target,
				Reason:         "stocktake",
			})
			if err != nil {
				t.Fatalf("Adjust: %v", err)
			}

			if movement.MovementType != tc.wantType {
				t.Errorf("movement type = %q, want %q", movement.MovementType, tc.wantType)
			}
			// The quantity written to the ledger is always positive; the
			// direction is the type's job.
			if movement.Quantity <= 0 {
				t.Errorf("movement quantity = %v, want a positive number", movement.Quantity)
			}
			if diff := movement.Quantity - tc.wantQuantity; diff > 1e-9 || diff < -1e-9 {
				t.Errorf("movement quantity = %v, want %v", movement.Quantity, tc.wantQuantity)
			}
			if movement.ReferenceType != domain.StockRefStockAdjustment {
				t.Errorf("reference type = %q, want %q", movement.ReferenceType, domain.StockRefStockAdjustment)
			}
			if len(stockRepo.appliedT) != 1 {
				t.Errorf("posted %d movements, want exactly 1", len(stockRepo.appliedT))
			}
		})
	}
}

// Adjusting to the quantity already on hand must post nothing. Without the
// epsilon guard this becomes a movement of ~1e-16, which
// chk_stock_movements_quantity_positive rejects as a 500 rather than as the
// no-op it is.
func TestAdjustToTheCurrentQuantityPostsNothing(t *testing.T) {
	svc, stockRepo := newAdjustFixture(100)

	_, err := svc.Adjust(context.Background(), &AdjustStockInput{
		CompanyID:      uuid.New(),
		ProductID:      uuid.New(),
		WarehouseID:    uuid.New(),
		UserID:         uuid.New(),
		TargetQuantity: 100,
	})
	if err != domain.ErrOrderNothingToPost {
		t.Errorf("Adjust to the same quantity = %v, want ErrOrderNothingToPost", err)
	}
	if len(stockRepo.appliedT) != 0 {
		t.Errorf("posted %d movements, want none", len(stockRepo.appliedT))
	}
}

// The same guard, exercised at the precision the NUMERIC(18,3) columns actually
// store: a difference smaller than half the last stored digit is not a change.
func TestAdjustIgnoresSubPrecisionDifferences(t *testing.T) {
	svc, stockRepo := newAdjustFixture(10.0001)

	_, err := svc.Adjust(context.Background(), &AdjustStockInput{
		CompanyID:      uuid.New(),
		ProductID:      uuid.New(),
		WarehouseID:    uuid.New(),
		UserID:         uuid.New(),
		TargetQuantity: 10,
	})
	if err != domain.ErrOrderNothingToPost {
		t.Errorf("sub-precision adjustment = %v, want ErrOrderNothingToPost", err)
	}
	if len(stockRepo.appliedT) != 0 {
		t.Errorf("posted %d movements, want none", len(stockRepo.appliedT))
	}
}

func TestAdjustRejectsANegativeTarget(t *testing.T) {
	svc, stockRepo := newAdjustFixture(100)

	_, err := svc.Adjust(context.Background(), &AdjustStockInput{
		CompanyID:      uuid.New(),
		ProductID:      uuid.New(),
		WarehouseID:    uuid.New(),
		UserID:         uuid.New(),
		TargetQuantity: -1,
	})
	if err != domain.ErrStockNegativeResult {
		t.Errorf("negative target = %v, want ErrStockNegativeResult", err)
	}
	if len(stockRepo.appliedT) != 0 {
		t.Error("a rejected adjustment must post nothing")
	}
}

// A product that has never been stocked in the warehouse has no balance row.
// The adjustment must treat that as a balance of zero and create the row,
// rather than failing with "stock record not found".
func TestAdjustCreatesTheFirstBalance(t *testing.T) {
	stockRepo := &fakeStockRepo{getErr: domain.ErrStockNotFound}
	productRepo := &fakeProductRepo{
		product:   &domain.Product{IsActive: true},
		warehouse: &domain.Warehouse{IsActive: true},
	}
	svc := &stockService{stockRepo: stockRepo, productRepo: productRepo}

	movement, err := svc.Adjust(context.Background(), &AdjustStockInput{
		CompanyID:      uuid.New(),
		ProductID:      uuid.New(),
		WarehouseID:    uuid.New(),
		UserID:         uuid.New(),
		TargetQuantity: 42,
	})
	if err != nil {
		t.Fatalf("Adjust: %v", err)
	}
	if movement.MovementType != domain.MovementAdjustmentIn {
		t.Errorf("movement type = %q, want adjustment_in", movement.MovementType)
	}
	if movement.Quantity != 42 {
		t.Errorf("movement quantity = %v, want 42", movement.Quantity)
	}
}

func TestTransferRejectsTheSameWarehouse(t *testing.T) {
	svc, stockRepo := newAdjustFixture(100)
	warehouseID := uuid.New()

	err := svc.Transfer(context.Background(), &TransferStockInput{
		CompanyID:       uuid.New(),
		ProductID:       uuid.New(),
		FromWarehouseID: warehouseID,
		ToWarehouseID:   warehouseID,
		Quantity:        5,
		UserID:          uuid.New(),
	})
	if err != domain.ErrStockSameWarehouse {
		t.Errorf("same-warehouse transfer = %v, want ErrStockSameWarehouse", err)
	}
	if len(stockRepo.appliedT) != 0 {
		t.Error("a rejected transfer must post nothing")
	}
}

func TestTransferRejectsANonPositiveQuantity(t *testing.T) {
	svc, _ := newAdjustFixture(100)

	err := svc.Transfer(context.Background(), &TransferStockInput{
		CompanyID:       uuid.New(),
		ProductID:       uuid.New(),
		FromWarehouseID: uuid.New(),
		ToWarehouseID:   uuid.New(),
		Quantity:        0,
		UserID:          uuid.New(),
	})
	if err != domain.ErrStockQuantityNotPositive {
		t.Errorf("zero-quantity transfer = %v, want ErrStockQuantityNotPositive", err)
	}
}

// A transfer is two legs, and both must be posted - an issue with no matching
// receipt destroys stock.
func TestTransferPostsBothLegs(t *testing.T) {
	svc, stockRepo := newAdjustFixture(100)
	from, to := uuid.New(), uuid.New()

	err := svc.Transfer(context.Background(), &TransferStockInput{
		CompanyID:       uuid.New(),
		ProductID:       uuid.New(),
		FromWarehouseID: from,
		ToWarehouseID:   to,
		Quantity:        7,
		UserID:          uuid.New(),
	})
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if len(stockRepo.appliedT) != 2 {
		t.Fatalf("posted %d movements, want 2", len(stockRepo.appliedT))
	}

	out, in := stockRepo.appliedT[0], stockRepo.appliedT[1]
	if out.MovementType != domain.MovementTransferOut || out.WarehouseID != from {
		t.Errorf("first leg = %q from %v, want transfer_out from the source", out.MovementType, out.WarehouseID)
	}
	if in.MovementType != domain.MovementTransferIn || in.WarehouseID != to {
		t.Errorf("second leg = %q to %v, want transfer_in to the destination", in.MovementType, in.WarehouseID)
	}
	if out.Quantity != 7 || in.Quantity != 7 {
		t.Errorf("legs moved %v and %v, want 7 each", out.Quantity, in.Quantity)
	}
	// The inbound leg points back at the outbound one, so the pair is
	// recognisable as one transfer in the ledger.
	if in.ReferenceID == nil {
		t.Error("the inbound leg must reference the outbound one")
	}
}
