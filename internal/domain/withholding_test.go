package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

func TestCalculateBusinessIncomeWithholding(t *testing.T) {
	t.Run("the familiar 3.3% on a round amount", func(t *testing.T) {
		got, err := domain.CalculateBusinessIncomeWithholding(1_000_000)
		require.NoError(t, err)

		assert.Equal(t, int64(30_000), got.IncomeTax, "소득세 3%")
		assert.Equal(t, int64(3_000), got.LocalIncomeTax, "지방소득세 = 소득세의 10%")
		assert.Equal(t, int64(33_000), got.TotalWithheld)
		assert.Equal(t, int64(967_000), got.NetPayment)
	})

	t.Run("truncates both taxes below 10 won", func(t *testing.T) {
		// 1,234,567 x 3% = 37,037.01 -> 37,030 (10원 미만 절사)
		// 37,030 x 10%   =  3,703    ->  3,700
		got, err := domain.CalculateBusinessIncomeWithholding(1_234_567)
		require.NoError(t, err)

		assert.Equal(t, int64(37_030), got.IncomeTax)
		assert.Equal(t, int64(3_700), got.LocalIncomeTax)
		assert.Equal(t, int64(40_730), got.TotalWithheld)
		assert.Equal(t, int64(1_193_837), got.NetPayment)
	})

	t.Run("three million", func(t *testing.T) {
		got, err := domain.CalculateBusinessIncomeWithholding(3_000_000)
		require.NoError(t, err)
		assert.Equal(t, int64(90_000), got.IncomeTax)
		assert.Equal(t, int64(9_000), got.LocalIncomeTax)
		assert.Equal(t, int64(99_000), got.TotalWithheld)
	})

	t.Run("rejects a negative payment", func(t *testing.T) {
		_, err := domain.CalculateBusinessIncomeWithholding(-1)
		assert.ErrorIs(t, err, domain.ErrNegativePaymentAmount)
	})
}

func TestCalculateOtherIncomeWithholding(t *testing.T) {
	t.Run("standard 60% deemed expense", func(t *testing.T) {
		// 지급액 1,000,000 - 필요경비 600,000 = 기타소득금액 400,000
		// 소득세 400,000 x 20% = 80,000, 지방소득세 8,000
		got, err := domain.CalculateOtherIncomeWithholding(1_000_000, domain.OtherIncomeStandardExpenseRateBps)
		require.NoError(t, err)

		assert.Equal(t, int64(400_000), got.TaxableAmount)
		assert.Equal(t, int64(80_000), got.IncomeTax)
		assert.Equal(t, int64(8_000), got.LocalIncomeTax)
		assert.Equal(t, int64(88_000), got.TotalWithheld)
		assert.Equal(t, int64(912_000), got.NetPayment)
	})

	t.Run("과세최저한: 기타소득금액 5만원 이하는 과세하지 않는다", func(t *testing.T) {
		// 소득세법 제84조. 지급액 125,000 - 필요경비 75,000 = 50,000.
		got, err := domain.CalculateOtherIncomeWithholding(125_000, domain.OtherIncomeStandardExpenseRateBps)
		require.NoError(t, err)

		assert.Equal(t, int64(50_000), got.TaxableAmount)
		assert.Equal(t, int64(0), got.IncomeTax)
		assert.Equal(t, int64(0), got.TotalWithheld)
		assert.Equal(t, int64(125_000), got.NetPayment, "전액 지급")
	})

	t.Run("just above 과세최저한", func(t *testing.T) {
		// 125,001 - 75,000 = 50,001 -> 과세 대상.
		got, err := domain.CalculateOtherIncomeWithholding(125_001, domain.OtherIncomeStandardExpenseRateBps)
		require.NoError(t, err)

		assert.Equal(t, int64(50_001), got.TaxableAmount)
		assert.Equal(t, int64(10_000), got.IncomeTax)
		assert.Equal(t, int64(1_000), got.LocalIncomeTax)
	})

	t.Run("no deemed expense", func(t *testing.T) {
		got, err := domain.CalculateOtherIncomeWithholding(1_000_000, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(1_000_000), got.TaxableAmount)
		assert.Equal(t, int64(200_000), got.IncomeTax)
		assert.Equal(t, int64(20_000), got.LocalIncomeTax)
	})

	t.Run("rejects an out of range expense rate", func(t *testing.T) {
		_, err := domain.CalculateOtherIncomeWithholding(1_000_000, 10_001)
		assert.Error(t, err)
	})
}

func TestEarnedIncomeDeduction(t *testing.T) {
	// 소득세법 제47조. Each case sits on a bracket boundary so the table can be
	// read straight off the statute.
	tests := []struct {
		name        string
		grossSalary int64
		expected    int64
	}{
		{"500만원 (70%)", 5_000_000, 3_500_000},
		{"1,000만원 (350만 + 초과분 40%)", 10_000_000, 5_500_000},
		{"1,500만원 경계", 15_000_000, 7_500_000},
		{"3,000만원 (750만 + 초과분 15%)", 30_000_000, 9_750_000},
		{"4,500만원 경계", 45_000_000, 12_000_000},
		{"5,000만원 (1,200만 + 초과분 5%)", 50_000_000, 12_250_000},
		{"1억원 경계", 100_000_000, 14_750_000},
		{"2억원 (1,475만 + 초과분 2%)", 200_000_000, 16_750_000},
		{"공제한도 2,000만원 적용", 1_000_000_000, 20_000_000},
		{"급여 없음", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, domain.EarnedIncomeDeduction(tt.grossSalary))
		})
	}
}

func TestIncomeTaxByBasicRate(t *testing.T) {
	// 소득세법 제55조 기본세율. The boundary cases double as a check that the
	// cumulative base tax of each bracket equals the tax computed at the end of
	// the previous one; an inconsistent table would show up here immediately.
	tests := []struct {
		name     string
		taxBase  int64
		expected int64
	}{
		{"과세표준 없음", 0, 0},
		{"1,000만원 (6%)", 10_000_000, 600_000},
		{"1,400만원 경계", 14_000_000, 840_000},
		{"3,000만원", 30_000_000, 3_240_000},
		{"5,000만원 경계", 50_000_000, 6_240_000},
		{"8,800만원 경계", 88_000_000, 15_360_000},
		{"1억5,000만원 경계", 150_000_000, 37_060_000},
		{"3억원 경계", 300_000_000, 94_060_000},
		{"5억원 경계", 500_000_000, 174_060_000},
		{"10억원 경계", 1_000_000_000, 384_060_000},
		{"20억원 (최고세율 45%)", 2_000_000_000, 834_060_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, domain.IncomeTaxByBasicRate(tt.taxBase))
		})
	}
}

func TestCalculateEarnedIncomeWithholding_NotImplemented(t *testing.T) {
	// The 간이세액표 is not loaded. This must fail loudly rather than return a
	// plausible-looking number: the amount is withheld and filed every month.
	_, err := domain.CalculateEarnedIncomeWithholding(3_000_000, 1)
	assert.ErrorIs(t, err, domain.ErrSimplifiedTaxTableUnavailable)
}

func TestRoundDownTo(t *testing.T) {
	assert.Equal(t, int64(37_030), domain.RoundDownTo(37_037, domain.TenWonUnit))
	assert.Equal(t, int64(37_030), domain.RoundDownTo(37_030, domain.TenWonUnit))
	assert.Equal(t, int64(0), domain.RoundDownTo(9, domain.TenWonUnit))
	assert.Equal(t, int64(123_456), domain.RoundDownTo(123_456, domain.WonUnit))
	// Truncation is toward zero on both sides of the axis.
	assert.Equal(t, int64(-37_030), domain.RoundDownTo(-37_037, domain.TenWonUnit))
}
