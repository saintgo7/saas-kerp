package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

func TestCalculateVAT(t *testing.T) {
	tests := []struct {
		name     string
		supply   int64
		category domain.VATCategory
		expected int64
	}{
		// 부가가치세법 제30조: 세율 10%.
		{"round figure", 1_000_000, domain.VATCategoryTaxable, 100_000},
		{"ten million", 10_000_000, domain.VATCategoryTaxable, 1_000_000},
		// 1,234,567 x 10% = 123,456.7 -> 원 미만 절사.
		{"fraction truncated", 1_234_567, domain.VATCategoryTaxable, 123_456},
		{"one won", 1, domain.VATCategoryTaxable, 0},
		{"nine won", 9, domain.VATCategoryTaxable, 0},
		{"ten won", 10, domain.VATCategoryTaxable, 1},
		{"zero", 0, domain.VATCategoryTaxable, 0},
		// 영세율: 세금계산서는 발급하되 세액은 0.
		{"zero rated", 10_000_000, domain.VATCategoryZeroRated, 0},
		// 면세: 부가가치세를 과세하지 않는다.
		{"exempt", 10_000_000, domain.VATCategoryExempt, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.CalculateVAT(tt.supply, tt.category)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCalculateVAT_Rejects(t *testing.T) {
	_, err := domain.CalculateVAT(1000, domain.VATCategory("bogus"))
	assert.ErrorIs(t, err, domain.ErrInvalidVATCategory)

	_, err = domain.CalculateVAT(-1, domain.VATCategoryTaxable)
	assert.ErrorIs(t, err, domain.ErrNegativeSupplyAmount)
}

func TestSplitTaxInclusive(t *testing.T) {
	t.Run("exact split", func(t *testing.T) {
		supply, tax, err := domain.SplitTaxInclusive(11_000, domain.VATCategoryTaxable)
		require.NoError(t, err)
		assert.Equal(t, int64(10_000), supply)
		assert.Equal(t, int64(1_000), tax)
	})

	t.Run("supply truncated, tax absorbs the remainder", func(t *testing.T) {
		// 10,000 / 1.1 = 9,090.909... -> 공급가액 9,090, 세액 910.
		supply, tax, err := domain.SplitTaxInclusive(10_000, domain.VATCategoryTaxable)
		require.NoError(t, err)
		assert.Equal(t, int64(9_090), supply)
		assert.Equal(t, int64(910), tax)
		// The two halves must always reconstruct the amount charged.
		assert.Equal(t, int64(10_000), supply+tax)
	})

	t.Run("zero rated keeps the whole amount as supply", func(t *testing.T) {
		supply, tax, err := domain.SplitTaxInclusive(10_000, domain.VATCategoryZeroRated)
		require.NoError(t, err)
		assert.Equal(t, int64(10_000), supply)
		assert.Equal(t, int64(0), tax)
	})
}

func TestValidateVATAmount(t *testing.T) {
	t.Run("accepts the exact statutory amount", func(t *testing.T) {
		assert.NoError(t, domain.ValidateVATAmount(1_000_000, 100_000, domain.VATCategoryTaxable))
	})

	t.Run("accepts a one won rounding difference", func(t *testing.T) {
		// Truncation gives 123,456; a counterparty that rounds gives 123,457.
		assert.NoError(t, domain.ValidateVATAmount(1_234_567, 123_457, domain.VATCategoryTaxable))
	})

	t.Run("rejects a two won difference", func(t *testing.T) {
		assert.ErrorIs(t,
			domain.ValidateVATAmount(1_234_567, 123_458, domain.VATCategoryTaxable),
			domain.ErrVATAmountMismatch)
	})

	t.Run("rejects zero tax on a taxable supply", func(t *testing.T) {
		// This is the understated-output-tax case: 10,000,000 supplied with
		// tax_amount 0 would be filed with the NTS as 0 VAT.
		assert.ErrorIs(t,
			domain.ValidateVATAmount(10_000_000, 0, domain.VATCategoryTaxable),
			domain.ErrVATAmountMismatch)
	})

	t.Run("rejects tax on an exempt supply", func(t *testing.T) {
		assert.ErrorIs(t,
			domain.ValidateVATAmount(10_000_000, 1_000_000, domain.VATCategoryExempt),
			domain.ErrVATAmountMismatch)
	})
}
