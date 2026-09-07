//go:build inventory_integration

package repository_test

// End-to-end checks of the inventory repositories against a real PostgreSQL
// that has db/migrations applied. They exist because the defects this module is
// most exposed to cannot be reproduced against a mock:
//
//   - a GORM field name that does not map to the column 000022 created
//   - writing stocks.available_quantity, which is GENERATED ALWAYS
//   - two concurrent issues both succeeding and driving the balance negative
//   - a replayed approve or receive applying twice
//
// Run against a throwaway database that has db/migrations applied:
//
//	INVENTORY_TEST_DSN='postgres://user@127.0.0.1:5432/db?sslmode=disable' \
//	    go test -tags=inventory_integration ./internal/repository/ -run Inventory -v
//
// The build tag keeps them out of `go test ./...`, which has no database. The
// tag is `inventory_integration` rather than the `integration` the other suites
// in this package use, because those import testcontainers-go, which is not in
// go.mod - building with `-tags=integration` therefore fails before any test
// runs. Owning a separate tag keeps this suite runnable today without adding a
// dependency to a file this module does not own.

import (
	"context"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

type inventoryFixture struct {
	db          *gorm.DB
	companyID   uuid.UUID
	userID      uuid.UUID
	supplierID  uuid.UUID
	warehouseID uuid.UUID
	productID   uuid.UUID

	productRepo repository.ProductRepository
	stockRepo   repository.StockRepository
	orderRepo   repository.OrderRepository
}

func setupInventory(t *testing.T) *inventoryFixture {
	t.Helper()

	dsn := os.Getenv("INVENTORY_TEST_DSN")
	if dsn == "" {
		t.Skip("INVENTORY_TEST_DSN is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	f := &inventoryFixture{
		db:          db,
		companyID:   uuid.New(),
		userID:      uuid.New(),
		productRepo: repository.NewProductRepositoryGorm(db),
		stockRepo:   repository.NewStockRepositoryGorm(db),
		orderRepo:   repository.NewOrderRepositoryGorm(db),
	}

	// Tenant root and the two rows every inventory row references.
	exec := func(query string, args ...interface{}) {
		t.Helper()
		if err := db.Exec(query, args...).Error; err != nil {
			t.Fatalf("seed (%s): %v", query, err)
		}
	}
	exec(`INSERT INTO companies (id, business_number, name, representative) VALUES (?, ?, ?, ?)`,
		f.companyID, "999-99-"+uuid.New().String()[:5], "INV-TEST", "대표")
	exec(`INSERT INTO users (id, company_id, email, password_hash, name, role) VALUES (?, ?, ?, ?, ?, ?)`,
		f.userID, f.companyID, "inv-"+uuid.New().String()+"@example.test", "x", "재고담당", "admin")

	f.supplierID = uuid.New()
	exec(`INSERT INTO partners (id, company_id, partner_type, partner_name) VALUES (?, ?, ?, ?)`,
		f.supplierID, f.companyID, "vendor", "테스트공급사")

	t.Cleanup(func() {
		// companies cascades to everything created below it.
		db.Exec(`DELETE FROM companies WHERE id = ?`, f.companyID)
	})

	ctx := context.Background()

	warehouse := &domain.Warehouse{
		TenantModel: domain.TenantModel{CompanyID: f.companyID},
		Code:        "WH1",
		Name:        "본사창고",
		IsDefault:   true,
		IsActive:    true,
	}
	if err := f.productRepo.CreateWarehouse(ctx, warehouse); err != nil {
		t.Fatalf("create warehouse: %v", err)
	}
	f.warehouseID = warehouse.ID

	minStock := 10.0
	product := &domain.Product{
		TenantModel: domain.TenantModel{CompanyID: f.companyID},
		Code:        "P001",
		Name:        "테스트품목",
		Unit:        "EA",
		UnitPrice:   1500.50,
		CostPrice:   1000.25,
		MinStock:    &minStock,
		IsActive:    true,
		Barcode:     "8801234567890",
		ImageURL:    "https://example.test/p001.png",
	}
	if err := f.productRepo.Create(ctx, product); err != nil {
		t.Fatalf("create product: %v", err)
	}
	f.productID = product.ID

	return f
}

// Every GORM field must map to a column 000022 actually created, and the
// NUMERIC precisions must survive a round trip.
func TestInventoryProductRoundTrip(t *testing.T) {
	f := setupInventory(t)
	ctx := context.Background()

	product, err := f.productRepo.GetByID(ctx, f.companyID, f.productID)
	if err != nil {
		t.Fatalf("get product: %v", err)
	}

	if product.Code != "P001" || product.Name != "테스트품목" {
		t.Errorf("identity did not round trip: %+v", product)
	}
	// NUMERIC(18,2) keeps both decimals.
	if product.UnitPrice != 1500.50 || product.CostPrice != 1000.25 {
		t.Errorf("prices = %v / %v, want 1500.50 / 1000.25", product.UnitPrice, product.CostPrice)
	}
	if product.MinStock == nil || *product.MinStock != 10 {
		t.Errorf("min stock did not round trip: %v", product.MinStock)
	}
	if product.MaxStock != nil {
		t.Errorf("max stock should still be NULL, got %v", *product.MaxStock)
	}
	if product.ImageURL != "https://example.test/p001.png" {
		t.Errorf("image_url did not round trip: %q", product.ImageURL)
	}
}

// The partial unique indexes are the real guarantee behind the service's
// pre-checks, so they are exercised directly.
func TestInventoryProductCodeUniquePerTenant(t *testing.T) {
	f := setupInventory(t)
	ctx := context.Background()

	duplicate := &domain.Product{
		TenantModel: domain.TenantModel{CompanyID: f.companyID},
		Code:        "P001",
		Name:        "중복코드",
		Unit:        "EA",
		IsActive:    true,
	}
	if err := f.productRepo.Create(ctx, duplicate); err == nil {
		t.Error("a duplicate product code was accepted; uq_products_code is not doing its job")
	}
}

// stocks.available_quantity is GENERATED ALWAYS AS (quantity -
// reserved_quantity) STORED. Without the `->` tag on the field, GORM includes
// it in every INSERT and PostgreSQL rejects the statement outright.
func TestInventoryGeneratedAvailableQuantity(t *testing.T) {
	f := setupInventory(t)
	ctx := context.Background()

	movement := &domain.StockMovement{
		CompanyID:     f.companyID,
		ProductID:     f.productID,
		WarehouseID:   f.warehouseID,
		MovementType:  domain.MovementPurchaseIn,
		Quantity:      100,
		ReferenceType: domain.StockRefStockAdjustment,
		CreatedBy:     f.userID,
	}
	if err := f.stockRepo.ApplyMovement(ctx, movement); err != nil {
		t.Fatalf("apply movement: %v", err)
	}

	stock, err := f.stockRepo.Get(ctx, f.companyID, f.productID, f.warehouseID)
	if err != nil {
		t.Fatalf("get stock: %v", err)
	}
	if stock.Quantity != 100 {
		t.Errorf("quantity = %v, want 100", stock.Quantity)
	}
	if stock.AvailableQuantity != 100 {
		t.Errorf("available = %v, want 100 (the generated column was not read)", stock.AvailableQuantity)
	}

	// A second write must not try to set the generated column either.
	if err := f.stockRepo.ApplyMovement(ctx, &domain.StockMovement{
		CompanyID:    f.companyID,
		ProductID:    f.productID,
		WarehouseID:  f.warehouseID,
		MovementType: domain.MovementSalesOut,
		Quantity:     30,
		CreatedBy:    f.userID,
	}); err != nil {
		t.Fatalf("second movement: %v", err)
	}

	stock, _ = f.stockRepo.Get(ctx, f.companyID, f.productID, f.warehouseID)
	if stock.Quantity != 70 || stock.AvailableQuantity != 70 {
		t.Errorf("after issue: quantity=%v available=%v, want 70/70", stock.Quantity, stock.AvailableQuantity)
	}
}

// The ledger row must record the balance the database actually reached, not a
// number computed in Go.
func TestInventoryMovementRecordsRealBalances(t *testing.T) {
	f := setupInventory(t)
	ctx := context.Background()

	first := &domain.StockMovement{
		CompanyID: f.companyID, ProductID: f.productID, WarehouseID: f.warehouseID,
		MovementType: domain.MovementPurchaseIn, Quantity: 40, CreatedBy: f.userID,
		// Deliberately wrong: ApplyMovement must overwrite both.
		PreviousQuantity: 999, CurrentQuantity: 999,
	}
	if err := f.stockRepo.ApplyMovement(ctx, first); err != nil {
		t.Fatalf("first: %v", err)
	}
	if first.PreviousQuantity != 0 || first.CurrentQuantity != 40 {
		t.Errorf("first movement recorded %v -> %v, want 0 -> 40",
			first.PreviousQuantity, first.CurrentQuantity)
	}

	second := &domain.StockMovement{
		CompanyID: f.companyID, ProductID: f.productID, WarehouseID: f.warehouseID,
		MovementType: domain.MovementSalesOut, Quantity: 15, CreatedBy: f.userID,
	}
	if err := f.stockRepo.ApplyMovement(ctx, second); err != nil {
		t.Fatalf("second: %v", err)
	}
	if second.PreviousQuantity != 40 || second.CurrentQuantity != 25 {
		t.Errorf("second movement recorded %v -> %v, want 40 -> 25",
			second.PreviousQuantity, second.CurrentQuantity)
	}
}

// An over-issue must be refused as a domain error, not surfaced as a raw
// constraint violation.
func TestInventoryOverIssueIsRefused(t *testing.T) {
	f := setupInventory(t)
	ctx := context.Background()

	if err := f.stockRepo.ApplyMovement(ctx, &domain.StockMovement{
		CompanyID: f.companyID, ProductID: f.productID, WarehouseID: f.warehouseID,
		MovementType: domain.MovementPurchaseIn, Quantity: 10, CreatedBy: f.userID,
	}); err != nil {
		t.Fatalf("receipt: %v", err)
	}

	err := f.stockRepo.ApplyMovement(ctx, &domain.StockMovement{
		CompanyID: f.companyID, ProductID: f.productID, WarehouseID: f.warehouseID,
		MovementType: domain.MovementSalesOut, Quantity: 11, CreatedBy: f.userID,
	})
	if err != domain.ErrStockNegativeResult {
		t.Fatalf("over-issue = %v, want ErrStockNegativeResult", err)
	}

	// And the failed attempt must have left no ledger row behind.
	movements, total, err := f.stockRepo.ListMovements(ctx, &repository.MovementFilter{
		CompanyID: f.companyID, Page: 1, PageSize: 50,
	})
	if err != nil {
		t.Fatalf("list movements: %v", err)
	}
	if total != 1 || len(movements) != 1 {
		t.Fatalf("ledger holds %d rows, want exactly the successful receipt", total)
	}
	if movements[0].CreatedByName != "재고담당" {
		t.Errorf("movement created_by_name = %q, want 재고담당", movements[0].CreatedByName)
	}
}

// The property the whole design rests on: concurrent issues must not both
// succeed. This is the read-modify-write defect that GenerateVoucherNo was
// fixed for, and here it would leave the ledger and the balance permanently
// inconsistent.
func TestInventoryConcurrentIssuesDoNotOversell(t *testing.T) {
	f := setupInventory(t)
	ctx := context.Background()

	if err := f.stockRepo.ApplyMovement(ctx, &domain.StockMovement{
		CompanyID: f.companyID, ProductID: f.productID, WarehouseID: f.warehouseID,
		MovementType: domain.MovementPurchaseIn, Quantity: 100, CreatedBy: f.userID,
	}); err != nil {
		t.Fatalf("receipt: %v", err)
	}

	// Ten goroutines each issue 20 from a balance of 100. At most five can
	// succeed; the rest must fail on the CHECK.
	const workers = 10
	const each = 20.0

	var wg sync.WaitGroup
	results := make([]error, workers)
	start := make(chan struct{})

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			results[idx] = f.stockRepo.ApplyMovement(ctx, &domain.StockMovement{
				CompanyID: f.companyID, ProductID: f.productID, WarehouseID: f.warehouseID,
				MovementType: domain.MovementSalesOut, Quantity: each, CreatedBy: f.userID,
			})
		}(i)
	}
	close(start)
	wg.Wait()

	succeeded := 0
	for i, err := range results {
		switch err {
		case nil:
			succeeded++
		case domain.ErrStockNegativeResult:
			// expected for the losers
		default:
			t.Errorf("worker %d failed unexpectedly: %v", i, err)
		}
	}

	if succeeded != 5 {
		t.Errorf("%d issues of %v succeeded against a balance of 100, want exactly 5", succeeded, each)
	}

	stock, err := f.stockRepo.Get(ctx, f.companyID, f.productID, f.warehouseID)
	if err != nil {
		t.Fatalf("get stock: %v", err)
	}
	if stock.Quantity < 0 {
		t.Fatalf("the balance went negative: %v", stock.Quantity)
	}
	if stock.Quantity != 0 {
		t.Errorf("final balance = %v, want 0", stock.Quantity)
	}

	// The ledger and the cached balance must agree: one receipt plus the
	// issues that actually committed.
	_, total, err := f.stockRepo.ListMovements(ctx, &repository.MovementFilter{
		CompanyID: f.companyID, Page: 1, PageSize: 100,
	})
	if err != nil {
		t.Fatalf("list movements: %v", err)
	}
	if int(total) != 1+succeeded {
		t.Errorf("ledger holds %d rows, want %d", total, 1+succeeded)
	}
}

// Order numbers must be unique under concurrency; a read-modify-write would
// hand two callers the same one.
func TestInventoryOrderNumberingIsAtomic(t *testing.T) {
	f := setupInventory(t)
	ctx := context.Background()

	const workers = 20
	var wg sync.WaitGroup
	numbers := make([]string, workers)
	start := make(chan struct{})

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			number, err := f.orderRepo.NextOrderNumber(ctx, f.companyID, "purchase", 2026)
			if err != nil {
				t.Errorf("worker %d: %v", idx, err)
				return
			}
			numbers[idx] = number
		}(i)
	}
	close(start)
	wg.Wait()

	seen := make(map[string]bool, workers)
	for i, number := range numbers {
		if number == "" {
			t.Fatalf("worker %d produced no number", i)
		}
		if seen[number] {
			t.Fatalf("order number %q was handed out twice", number)
		}
		seen[number] = true
	}
	if len(seen) != workers {
		t.Errorf("got %d distinct numbers from %d callers", len(seen), workers)
	}
}

// The full order lifecycle, including the two properties a double-clicked
// button depends on: a repeated transition is refused, and a repeated receipt
// cannot over-receive.
func TestInventoryPurchaseOrderLifecycle(t *testing.T) {
	f := setupInventory(t)
	ctx := context.Background()

	order := &domain.PurchaseOrder{
		TenantModel: domain.TenantModel{CompanyID: f.companyID},
		OrderNumber: "PO-2026-900001",
		OrderDate:   time.Now().Truncate(24 * time.Hour),
		SupplierID:  f.supplierID,
		WarehouseID: f.warehouseID,
		TotalAmount: 15005,
		TaxAmount:   1501,
		GrandTotal:  16506,
		Status:      domain.POStatusDraft,
		CreatedBy:   f.userID,
		Items: []domain.PurchaseOrderItem{{
			TenantModel: domain.TenantModel{CompanyID: f.companyID},
			LineNo:      1,
			ProductID:   f.productID,
			Quantity:    10,
			UnitPrice:   1500.50,
			Amount:      15005,
			TaxAmount:   1501,
		}},
	}
	if err := f.orderRepo.CreatePurchaseOrder(ctx, order); err != nil {
		t.Fatalf("create order: %v", err)
	}

	loaded, err := f.orderRepo.GetPurchaseOrder(ctx, f.companyID, order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if len(loaded.Items) != 1 {
		t.Fatalf("loaded %d lines, want 1", len(loaded.Items))
	}
	if loaded.GrandTotal != loaded.TotalAmount+loaded.TaxAmount {
		t.Error("the grand-total identity did not survive the round trip")
	}
	// The joins the list screen needs.
	if loaded.Supplier == nil || loaded.Warehouse == nil {
		t.Error("supplier and warehouse must both be preloaded")
	}
	// The creator's display name is resolved by an explicit unqualified lookup,
	// not by a GORM association, so that only id and name are read - which
	// does not exist on a database that keeps its tables in public. This
	// assertion is what caught that.
	if loaded.CreatedByName != "재고담당" {
		t.Errorf("created_by_name = %q, want 재고담당", loaded.CreatedByName)
	}

	// draft -> pending -> approved
	if ok, err := f.orderRepo.TransitionPurchaseOrder(ctx, f.companyID, order.ID,
		[]domain.PurchaseOrderStatus{domain.POStatusDraft}, domain.POStatusPending, nil); err != nil || !ok {
		t.Fatalf("submit: ok=%v err=%v", ok, err)
	}
	if ok, err := f.orderRepo.TransitionPurchaseOrder(ctx, f.companyID, order.ID,
		[]domain.PurchaseOrderStatus{domain.POStatusPending}, domain.POStatusApproved, &f.userID); err != nil || !ok {
		t.Fatalf("approve: ok=%v err=%v", ok, err)
	}

	// The double-click. The second approve matches no row.
	ok, err := f.orderRepo.TransitionPurchaseOrder(ctx, f.companyID, order.ID,
		[]domain.PurchaseOrderStatus{domain.POStatusPending}, domain.POStatusApproved, &f.userID)
	if err != nil {
		t.Fatalf("second approve errored: %v", err)
	}
	if ok {
		t.Error("approving an already-approved order succeeded; the compare-and-swap is not working")
	}

	// approved -> ordered, then receive part of the line.
	if ok, err := f.orderRepo.TransitionPurchaseOrder(ctx, f.companyID, order.ID,
		[]domain.PurchaseOrderStatus{domain.POStatusApproved}, domain.POStatusOrdered, nil); err != nil || !ok {
		t.Fatalf("place: ok=%v err=%v", ok, err)
	}

	itemID := loaded.Items[0].ID
	if ok, err := f.orderRepo.AddReceivedQuantity(ctx, f.companyID, itemID, 4); err != nil || !ok {
		t.Fatalf("first receipt: ok=%v err=%v", ok, err)
	}
	if ok, err := f.orderRepo.AddReceivedQuantity(ctx, f.companyID, itemID, 6); err != nil || !ok {
		t.Fatalf("second receipt: ok=%v err=%v", ok, err)
	}

	// The line is now fully received; one more unit must be refused rather
	// than pushing received_quantity past quantity.
	ok, err = f.orderRepo.AddReceivedQuantity(ctx, f.companyID, itemID, 1)
	if err != nil {
		t.Fatalf("over-receipt errored: %v", err)
	}
	if ok {
		t.Error("an over-receipt was accepted")
	}

	final, err := f.orderRepo.GetPurchaseOrder(ctx, f.companyID, order.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if final.Items[0].ReceivedQuantity != 10 {
		t.Errorf("received quantity = %v, want 10", final.Items[0].ReceivedQuantity)
	}
	if final.ApprovedBy == nil || final.ApprovedAt == nil {
		t.Error("approval must record both who and when; chk_purchase_orders_approval requires it")
	}
}

// The summary must be computed by the database over the whole tenant, not from
// a page of rows.
func TestInventorySummaries(t *testing.T) {
	f := setupInventory(t)
	ctx := context.Background()

	if err := f.stockRepo.ApplyMovement(ctx, &domain.StockMovement{
		CompanyID: f.companyID, ProductID: f.productID, WarehouseID: f.warehouseID,
		MovementType: domain.MovementPurchaseIn, Quantity: 8, CreatedBy: f.userID,
	}); err != nil {
		t.Fatalf("receipt: %v", err)
	}

	summary, err := f.stockRepo.Summary(ctx, f.companyID)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.StockRecords != 1 || summary.ProductCount != 1 {
		t.Errorf("records=%d products=%d, want 1/1", summary.StockRecords, summary.ProductCount)
	}
	if summary.WarehouseCount != 1 {
		t.Errorf("warehouses = %d, want 1", summary.WarehouseCount)
	}
	// 8 units at a cost of 1000.25.
	if summary.TotalStockValue != 8*1000.25 {
		t.Errorf("stock value = %v, want %v", summary.TotalStockValue, 8*1000.25)
	}
	// 8 is at or below the reorder point of 10, and above zero.
	if summary.LowStockCount != 1 {
		t.Errorf("low stock count = %d, want 1", summary.LowStockCount)
	}
	if summary.OutOfStockCount != 0 {
		t.Errorf("out of stock count = %d, want 0", summary.OutOfStockCount)
	}

	alerts, err := f.stockRepo.ListAlertCandidates(ctx, f.companyID)
	if err != nil {
		t.Fatalf("alert candidates: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("got %d alert candidates, want 1", len(alerts))
	}
	derived := domain.DeriveStockAlert(&alerts[0])
	if derived == nil || derived.AlertType != domain.AlertLowStock {
		t.Errorf("derived alert = %+v, want low_stock", derived)
	}
}

// The low-stock filter must be resolved in SQL, so that it means "low across
// the tenant" and not "low among the rows on this page".
func TestInventoryBelowMinStockFilter(t *testing.T) {
	f := setupInventory(t)
	ctx := context.Background()

	// A second product with no reorder point at all. NULL min_stock is "no
	// threshold" and must never match the filter, even at a zero balance.
	unthresholded := &domain.Product{
		TenantModel: domain.TenantModel{CompanyID: f.companyID},
		Code:        "P002",
		Name:        "임계없음",
		Unit:        "EA",
		IsActive:    true,
	}
	if err := f.productRepo.Create(ctx, unthresholded); err != nil {
		t.Fatalf("create second product: %v", err)
	}

	for _, m := range []*domain.StockMovement{
		{CompanyID: f.companyID, ProductID: f.productID, WarehouseID: f.warehouseID,
			MovementType: domain.MovementPurchaseIn, Quantity: 5, CreatedBy: f.userID},
		{CompanyID: f.companyID, ProductID: unthresholded.ID, WarehouseID: f.warehouseID,
			MovementType: domain.MovementPurchaseIn, Quantity: 1, CreatedBy: f.userID},
	} {
		if err := f.stockRepo.ApplyMovement(ctx, m); err != nil {
			t.Fatalf("receipt: %v", err)
		}
	}

	stocks, total, err := f.stockRepo.List(ctx, &repository.StockFilter{
		CompanyID: f.companyID, BelowMinStock: true, Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(stocks) != 1 {
		t.Fatalf("filter returned %d rows, want only the item with a reorder point", total)
	}
	if stocks[0].ProductID != f.productID {
		t.Error("the filter returned the wrong product")
	}
	if stocks[0].Product == nil || stocks[0].Warehouse == nil {
		t.Error("the list must preload product and warehouse; the UI renders both")
	}
}

// The production path is the kerp_app role (NOSUPERUSER, NOBYPASSRLS) with
// app.current_tenant set on the connection. This test runs the repositories
// through that role to prove the 000022 policies and grants let the application
// do its work - and that they still stop it at the tenant boundary.
//
// Requires INVENTORY_TEST_APP_DSN, a DSN for kerp_app. Skipped when unset.
func TestInventoryRLSUnderTheApplicationRole(t *testing.T) {
	appDSN := os.Getenv("INVENTORY_TEST_APP_DSN")
	if appDSN == "" {
		t.Skip("INVENTORY_TEST_APP_DSN is not set")
	}

	// Seed as the migration role, which bypasses RLS.
	f := setupInventory(t)
	ctx := context.Background()

	if err := f.stockRepo.ApplyMovement(ctx, &domain.StockMovement{
		CompanyID: f.companyID, ProductID: f.productID, WarehouseID: f.warehouseID,
		MovementType: domain.MovementPurchaseIn, Quantity: 25, CreatedBy: f.userID,
	}); err != nil {
		t.Fatalf("seed stock: %v", err)
	}

	open := func(tenant string) repository.ProductRepository {
		t.Helper()
		dsn := appDSN
		if tenant != "" {
			// The GUC is pinned for the whole connection, which is what
			// middleware.TenantSession does per request.
			dsn += "&options=" + url.QueryEscape("-c app.current_tenant="+tenant)
		}
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			t.Fatalf("connect as the application role: %v", err)
		}
		return repository.NewProductRepositoryGorm(db)
	}

	// 1. No tenant set: the policies must fail closed, not open.
	noTenant := open("")
	_, total, err := noTenant.List(ctx, &repository.ProductFilter{
		CompanyID: f.companyID, Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list without a tenant: %v", err)
	}
	if total != 0 {
		t.Errorf("with app.current_tenant unset the application role saw %d products, want 0", total)
	}

	// 2. The right tenant: the application role can read its own data, which is
	//    what the GRANT block in 000022 is for.
	own := open(f.companyID.String())
	products, total, err := own.List(ctx, &repository.ProductFilter{
		CompanyID: f.companyID, Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list within the tenant: %v", err)
	}
	if total != 1 || len(products) != 1 {
		t.Fatalf("the tenant saw %d of its own products, want 1", total)
	}

	// 3. Another tenant's context must not reach these rows.
	other := open(uuid.New().String())
	_, total, err = other.List(ctx, &repository.ProductFilter{
		CompanyID: f.companyID, Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list as another tenant: %v", err)
	}
	if total != 0 {
		t.Errorf("another tenant saw %d products, want 0", total)
	}

	// 4. Writing into another tenant must be refused by the policy, not merely
	//    filtered out of reads.
	err = own.Create(ctx, &domain.Product{
		TenantModel: domain.TenantModel{CompanyID: uuid.New()},
		Code:        "CROSS",
		Name:        "타테넌트",
		Unit:        "EA",
		IsActive:    true,
	})
	if err == nil {
		t.Error("the application role inserted a row for another tenant; the WITH CHECK clause is not doing its job")
	}
}
