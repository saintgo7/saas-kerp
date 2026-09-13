package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

func TestRoundAmount(t *testing.T) {
	tests := []struct {
		name     string
		in       float64
		expected float64
	}{
		{"already at scale", 100.00, 100.00},
		{"rounds down", 33.333, 33.33},
		{"rounds up", 33.336, 33.34},
		{"negative rounds away from zero", -33.336, -33.34},
		{"sub-cent becomes zero", 0.004, 0.00},
		{"zero", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, domain.RoundAmount(tt.in), 1e-9)
		})
	}
}

func TestAmountsEqual(t *testing.T) {
	assert.True(t, domain.AmountsEqual(100.00, 100.00))
	// Accumulated binary representation error across many DECIMAL(18,2) rows.
	assert.True(t, domain.AmountsEqual(100.00, 100.0000000001))
	// One stored unit apart is a real difference.
	assert.False(t, domain.AmountsEqual(100.00, 100.01))
	assert.False(t, domain.AmountsEqual(99.99, 100.00))
}

// ============================================================================
// The audit case: 33.333 x 3 against 100.00
// ============================================================================

func TestVoucher_CalculateTotals_QuantizesEachLine(t *testing.T) {
	// Go used to sum the raw values to 100.000 on both sides and call the
	// voucher balanced, while PostgreSQL stored 33.33 three times - 99.99 -
	// leaving the ledger 0.01 short on every such voucher.
	voucher := &domain.Voucher{
		VoucherType: domain.VoucherTypeGeneral,
		Entries: []domain.VoucherEntry{
			{DebitAmount: 33.333},
			{DebitAmount: 33.333},
			{DebitAmount: 33.334},
			{CreditAmount: 100.00},
		},
	}

	voucher.CalculateTotals()

	// Every line is now quantized to what the database will actually store.
	assert.InDelta(t, 33.33, voucher.Entries[0].DebitAmount, 1e-9)
	assert.InDelta(t, 33.33, voucher.Entries[1].DebitAmount, 1e-9)
	assert.InDelta(t, 33.33, voucher.Entries[2].DebitAmount, 1e-9)

	// And the header total is the sum of those stored values, not of the
	// unrounded inputs.
	assert.InDelta(t, 99.99, voucher.TotalDebit, 1e-9)
	assert.InDelta(t, 100.00, voucher.TotalCredit, 1e-9)

	assert.False(t, voucher.IsBalanced(), "0.01 불균형이 드러나야 한다")
	assert.ErrorIs(t, voucher.ValidateBalance(), domain.ErrVoucherUnbalanced)
}

func TestVoucher_ValidateBalance_UsesEpsilon(t *testing.T) {
	t.Run("tolerates accumulated float error", func(t *testing.T) {
		voucher := &domain.Voucher{
			TotalDebit:  1_000_000.00,
			TotalCredit: 1_000_000.0000000001,
		}
		assert.NoError(t, voucher.ValidateBalance())
		assert.True(t, voucher.IsBalanced())
	})

	t.Run("still rejects a one unit difference", func(t *testing.T) {
		voucher := &domain.Voucher{
			TotalDebit:  1_000_000.00,
			TotalCredit: 1_000_000.01,
		}
		assert.ErrorIs(t, voucher.ValidateBalance(), domain.ErrVoucherUnbalanced)
	})
}

func TestVoucherEntry_Validate_AtStoredScale(t *testing.T) {
	t.Run("rejects an amount that would be stored as zero", func(t *testing.T) {
		// 0.004 is non-zero in Go but stored as 0.00, which violates
		// chk_entry_amount and surfaced as a raw 500 rather than a validation
		// error.
		entry := &domain.VoucherEntry{DebitAmount: 0.004}
		assert.ErrorIs(t, entry.Validate(), domain.ErrEntryZeroAmount)
	})

	t.Run("accepts the smallest storable amount", func(t *testing.T) {
		entry := &domain.VoucherEntry{DebitAmount: 0.01}
		assert.NoError(t, entry.Validate())
	})

	t.Run("rejects negative amounts", func(t *testing.T) {
		entry := &domain.VoucherEntry{DebitAmount: -100}
		assert.ErrorIs(t, entry.Validate(), domain.ErrEntryZeroAmount)
	})

	t.Run("rejects both sides set", func(t *testing.T) {
		entry := &domain.VoucherEntry{DebitAmount: 100, CreditAmount: 100}
		assert.ErrorIs(t, entry.Validate(), domain.ErrEntryInvalidAmount)
	})

	t.Run("Normalize quantizes both sides", func(t *testing.T) {
		entry := &domain.VoucherEntry{DebitAmount: 33.336, CreditAmount: 0}
		entry.Normalize()
		assert.InDelta(t, 33.34, entry.DebitAmount, 1e-9)
	})
}

// ============================================================================
// Trial balance
// ============================================================================

func TestTrialBalance_Validate_UsesEpsilon(t *testing.T) {
	t.Run("a balanced ledger is not reported as unbalanced", func(t *testing.T) {
		// Summing hundreds of DECIMAL(18,2) rows as float64 accumulates
		// representation error; an exact == comparison reported a perfectly
		// balanced ledger as "is_balanced": false in the API response.
		tb := &domain.TrialBalance{
			TotalDebit:  123_456_789.01,
			TotalCredit: 123_456_789.0100000001,
		}
		assert.True(t, tb.Validate())
		assert.True(t, tb.IsBalanced)
	})

	t.Run("a real imbalance is still reported", func(t *testing.T) {
		tb := &domain.TrialBalance{
			TotalDebit:  123_456_789.01,
			TotalCredit: 123_456_789.02,
		}
		assert.False(t, tb.Validate())
	})
}

// ============================================================================
// Closing balance sign convention
// ============================================================================

func TestLedgerBalance_ClosingBalanceByNature(t *testing.T) {
	revenue := domain.LedgerBalance{
		AccountID:     uuid.New(),
		ClosingCredit: 1_000_000,
	}
	expense := domain.LedgerBalance{
		AccountID:    uuid.New(),
		ClosingDebit: 600_000,
	}

	// GetClosingBalance is stated on the debit side, so a credit-nature
	// account reports a negative number. Mixing the two conventions is what
	// made the year-end close report -1,600,000 for a 400,000 profit.
	assert.Equal(t, -1_000_000.0, revenue.GetClosingBalance())
	assert.Equal(t, 600_000.0, expense.GetClosingBalance())

	// By nature, both are stated as they appear on the financial statements.
	assert.Equal(t, 1_000_000.0, revenue.GetClosingBalanceByNature(domain.AccountNatureCredit))
	assert.Equal(t, 600_000.0, expense.GetClosingBalanceByNature(domain.AccountNatureDebit))

	netIncome := revenue.GetClosingBalanceByNature(domain.AccountNatureCredit) -
		expense.GetClosingBalanceByNature(domain.AccountNatureDebit)
	require.Equal(t, 400_000.0, netIncome)
}

// ============================================================================
// Cancel is a real state transition
// ============================================================================

func TestVoucher_Cancel_StateTransition(t *testing.T) {
	t.Run("cancels a pending voucher", func(t *testing.T) {
		v := &domain.Voucher{Status: domain.VoucherStatusPending}
		require.NoError(t, v.Cancel())
		assert.Equal(t, domain.VoucherStatusCancelled, v.Status)
	})

	t.Run("refuses to cancel an already cancelled voucher", func(t *testing.T) {
		// A second cancel used to succeed and rewrite updated_at, which made
		// the audit trail's cancellation time untrustworthy.
		v := &domain.Voucher{Status: domain.VoucherStatusCancelled}
		assert.ErrorIs(t, v.Cancel(), domain.ErrVoucherCannotCancel)
	})

	t.Run("refuses to cancel a rejected voucher", func(t *testing.T) {
		v := &domain.Voucher{Status: domain.VoucherStatusRejected}
		assert.ErrorIs(t, v.Cancel(), domain.ErrVoucherCannotCancel)
	})

	t.Run("refuses to cancel a posted voucher", func(t *testing.T) {
		v := &domain.Voucher{Status: domain.VoucherStatusPosted}
		assert.ErrorIs(t, v.Cancel(), domain.ErrVoucherCannotCancel)
	})
}
