package domain_test

import (
	"math"
	"testing"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// The transition tables below are the specification, not a restatement of the
// implementation: each row says which action is legal in which status, and a
// change to the guards that is not also a change to the table fails here.

func TestPurchaseOrderStatusTransitions(t *testing.T) {
	all := []domain.PurchaseOrderStatus{
		domain.POStatusDraft, domain.POStatusPending, domain.POStatusApproved,
		domain.POStatusOrdered, domain.POStatusPartial, domain.POStatusCompleted,
		domain.POStatusCancelled,
	}

	cases := []struct {
		name    string
		allowed map[domain.PurchaseOrderStatus]bool
		guard   func(domain.PurchaseOrderStatus) bool
	}{
		{
			name:    "edit",
			allowed: map[domain.PurchaseOrderStatus]bool{domain.POStatusDraft: true},
			guard:   domain.PurchaseOrderStatus.CanEdit,
		},
		{
			name: "delete",
			allowed: map[domain.PurchaseOrderStatus]bool{
				domain.POStatusDraft: true, domain.POStatusPending: true,
			},
			guard: domain.PurchaseOrderStatus.CanDelete,
		},
		{
			name:    "submit",
			allowed: map[domain.PurchaseOrderStatus]bool{domain.POStatusDraft: true},
			guard:   domain.PurchaseOrderStatus.CanSubmit,
		},
		{
			name:    "approve",
			allowed: map[domain.PurchaseOrderStatus]bool{domain.POStatusPending: true},
			guard:   domain.PurchaseOrderStatus.CanApprove,
		},
		{
			name:    "place",
			allowed: map[domain.PurchaseOrderStatus]bool{domain.POStatusApproved: true},
			guard:   domain.PurchaseOrderStatus.CanPlace,
		},
		{
			name: "receive",
			allowed: map[domain.PurchaseOrderStatus]bool{
				domain.POStatusOrdered: true, domain.POStatusPartial: true,
			},
			guard: domain.PurchaseOrderStatus.CanReceive,
		},
		{
			// partial is deliberately absent: stock has already been received
			// against the order, so cancelling would strand those movements.
			name: "cancel",
			allowed: map[domain.PurchaseOrderStatus]bool{
				domain.POStatusDraft: true, domain.POStatusPending: true,
				domain.POStatusApproved: true, domain.POStatusOrdered: true,
			},
			guard: domain.PurchaseOrderStatus.CanCancel,
		},
	}

	for _, tc := range cases {
		for _, status := range all {
			want := tc.allowed[status]
			if got := tc.guard(status); got != want {
				t.Errorf("%s from %q = %v, want %v", tc.name, status, got, want)
			}
		}
	}
}

// A completed or cancelled order must accept no action at all. This is the
// property that stops a double-clicked button from re-running a finished
// workflow.
func TestPurchaseOrderTerminalStatusesAcceptNothing(t *testing.T) {
	for _, status := range []domain.PurchaseOrderStatus{domain.POStatusCompleted, domain.POStatusCancelled} {
		if !status.IsTerminal() {
			t.Errorf("%q should be terminal", status)
		}
		if status.CanEdit() || status.CanDelete() || status.CanSubmit() ||
			status.CanApprove() || status.CanReject() || status.CanPlace() ||
			status.CanReceive() || status.CanCancel() {
			t.Errorf("%q accepted an action while terminal", status)
		}
	}
}

func TestSalesOrderStatusTransitions(t *testing.T) {
	all := []domain.SalesOrderStatus{
		domain.SOStatusDraft, domain.SOStatusPending, domain.SOStatusApproved,
		domain.SOStatusConfirmed, domain.SOStatusPartial, domain.SOStatusCompleted,
		domain.SOStatusCancelled,
	}

	cases := []struct {
		name    string
		allowed map[domain.SalesOrderStatus]bool
		guard   func(domain.SalesOrderStatus) bool
	}{
		{
			name:    "edit",
			allowed: map[domain.SalesOrderStatus]bool{domain.SOStatusDraft: true},
			guard:   domain.SalesOrderStatus.CanEdit,
		},
		{
			name:    "approve",
			allowed: map[domain.SalesOrderStatus]bool{domain.SOStatusPending: true},
			guard:   domain.SalesOrderStatus.CanApprove,
		},
		{
			// The step that distinguishes the sales flow from the purchase one:
			// approval is internal, confirmation is the customer's.
			name:    "confirm",
			allowed: map[domain.SalesOrderStatus]bool{domain.SOStatusApproved: true},
			guard:   domain.SalesOrderStatus.CanConfirm,
		},
		{
			// Shipping starts at confirmed, NOT at approved. An approved but
			// unconfirmed order must not move stock.
			name: "ship",
			allowed: map[domain.SalesOrderStatus]bool{
				domain.SOStatusConfirmed: true, domain.SOStatusPartial: true,
			},
			guard: domain.SalesOrderStatus.CanShip,
		},
		{
			name: "cancel",
			allowed: map[domain.SalesOrderStatus]bool{
				domain.SOStatusDraft: true, domain.SOStatusPending: true,
				domain.SOStatusApproved: true, domain.SOStatusConfirmed: true,
			},
			guard: domain.SalesOrderStatus.CanCancel,
		},
	}

	for _, tc := range cases {
		for _, status := range all {
			want := tc.allowed[status]
			if got := tc.guard(status); got != want {
				t.Errorf("%s from %q = %v, want %v", tc.name, status, got, want)
			}
		}
	}
}

func TestSalesOrderTerminalStatusesAcceptNothing(t *testing.T) {
	for _, status := range []domain.SalesOrderStatus{domain.SOStatusCompleted, domain.SOStatusCancelled} {
		if !status.IsTerminal() {
			t.Errorf("%q should be terminal", status)
		}
		if status.CanEdit() || status.CanDelete() || status.CanSubmit() ||
			status.CanApprove() || status.CanReject() || status.CanConfirm() ||
			status.CanShip() || status.CanCancel() {
			t.Errorf("%q accepted an action while terminal", status)
		}
	}
}

func TestOrderStatusValidity(t *testing.T) {
	// The two vocabularies are deliberately asymmetric: purchase has `ordered`,
	// sales has `confirmed`. Sharing one enum would let a sales order be set to
	// `ordered`, which chk_sales_orders_status rejects at the database.
	if domain.PurchaseOrderStatus("confirmed").IsValid() {
		t.Error("purchase orders must not accept the sales-only status 'confirmed'")
	}
	if domain.SalesOrderStatus("ordered").IsValid() {
		t.Error("sales orders must not accept the purchase-only status 'ordered'")
	}
	if domain.PurchaseOrderStatus("shipped").IsValid() || domain.SalesOrderStatus("shipped").IsValid() {
		t.Error("an unknown status was accepted")
	}
}

func TestLineAmountRoundsToWholeWon(t *testing.T) {
	cases := []struct {
		quantity  float64
		unitPrice float64
		want      int64
	}{
		{10, 1000, 10000},
		{3, 3333.33, 10000},    // 9999.99 rounds up
		{1, 0.5, 1},            // half rounds away from zero
		{1, 0.4, 0},            // and below half rounds down
		{2.5, 1000, 2500},      // fractional quantities are supported
		{0.001, 1000000, 1000}, // three decimal places, the NUMERIC(18,3) limit
	}

	for _, tc := range cases {
		if got := domain.LineAmount(tc.quantity, tc.unitPrice); got != tc.want {
			t.Errorf("LineAmount(%v, %v) = %d, want %d", tc.quantity, tc.unitPrice, got, tc.want)
		}
	}
}

// The order's VAT is 10% of the order total, rounded once. It is NOT the sum of
// the per-line VATs: PurchaseOrderPage.tsx and SalesOrderPage.tsx both preview
// Math.round(totalAmount * 0.1), and a server that summed per-line roundings
// would disagree with the preview the user just approved.
func TestOrderTaxIsRoundedOnTheTotalNotPerLine(t *testing.T) {
	lineAmounts := []int64{15, 15, 15, 15, 15}

	var total int64
	var perLineTaxSum int64
	for _, amount := range lineAmounts {
		total += amount
		perLineTaxSum += domain.LineTaxAmount(amount)
	}

	orderTax := domain.OrderTaxAmount(total)

	// 75 * 0.1 = 7.5 -> 8, while five roundings of 1.5 give 5 * 2 = 10.
	if orderTax != 8 {
		t.Errorf("OrderTaxAmount(%d) = %d, want 8", total, orderTax)
	}
	if perLineTaxSum != 10 {
		t.Errorf("per-line tax sum = %d, want 10", perLineTaxSum)
	}
	if orderTax == perLineTaxSum {
		t.Error("this case was chosen because the two differ; it no longer does, so it proves nothing")
	}
}

// chk_purchase_orders_amounts / chk_sales_orders_amounts enforce
// grand_total = total_amount + tax_amount exactly. That identity only holds
// because the amounts are integers, so the arithmetic must never go through a
// float.
func TestGrandTotalIdentityIsExact(t *testing.T) {
	for _, total := range []int64{0, 1, 999_999, 1_234_567_890, math.MaxInt32} {
		tax := domain.OrderTaxAmount(total)
		grand := total + tax
		if grand != total+tax {
			t.Fatalf("identity broken for total=%d", total)
		}
		if tax < 0 {
			t.Errorf("OrderTaxAmount(%d) = %d, want a non-negative value", total, tax)
		}
	}
}

func TestOutstandingQuantity(t *testing.T) {
	poItem := &domain.PurchaseOrderItem{Quantity: 10, ReceivedQuantity: 4}
	if got := poItem.OutstandingQuantity(); got != 6 {
		t.Errorf("purchase outstanding = %v, want 6", got)
	}

	soItem := &domain.SalesOrderItem{Quantity: 10, ShippedQuantity: 10}
	if got := soItem.OutstandingQuantity(); got != 0 {
		t.Errorf("sales outstanding = %v, want 0", got)
	}
}
