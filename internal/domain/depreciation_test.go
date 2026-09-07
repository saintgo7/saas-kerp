package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

func TestDecliningBalanceRateThousandths(t *testing.T) {
	// 법인세법 시행규칙 별표4 (정률법 상각률). The published rates are the
	// exact value 1 - 0.05^(1/n) rounded UP at the fourth decimal place:
	//
	//   n=4  exact 0.527129 -> 0.528
	//   n=5  exact 0.450719 -> 0.451
	//   n=6  exact 0.393038 -> 0.394
	//   n=8  exact 0.312344 -> 0.313
	//   n=10 exact 0.258866 -> 0.259
	//
	// Rounding to nearest would give 0.527/0.393/0.312 and disagree with the
	// table on three of the five, which is why the implementation ceils.
	tests := []struct {
		usefulLife int
		expected   int64
	}{
		{4, 528},
		{5, 451},
		{6, 394},
		{8, 313},
		{10, 259},
	}

	for _, tt := range tests {
		t.Run(t.Name(), func(t *testing.T) {
			got, err := domain.DecliningBalanceRateThousandths(tt.usefulLife)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got, "내용연수 %d년", tt.usefulLife)
		})
	}

	t.Run("float view agrees with the integer rate", func(t *testing.T) {
		rate, err := domain.DecliningBalanceRate(5)
		require.NoError(t, err)
		assert.InDelta(t, 0.451, rate, 1e-9)
	})

	t.Run("rejects a useful life below one year", func(t *testing.T) {
		_, err := domain.DecliningBalanceRateThousandths(0)
		assert.ErrorIs(t, err, domain.ErrInvalidUsefulLife)
	})
}

func TestStraightLineAnnualExpense(t *testing.T) {
	t.Run("잔존가액 0, 내용연수 5년", func(t *testing.T) {
		got, err := domain.StraightLineAnnualExpense(10_000_000, 0, 5)
		require.NoError(t, err)
		assert.Equal(t, int64(2_000_000), got)
	})

	t.Run("잔존가액이 있는 경우", func(t *testing.T) {
		got, err := domain.StraightLineAnnualExpense(10_000_000, 1_000_000, 5)
		require.NoError(t, err)
		assert.Equal(t, int64(1_800_000), got)
	})

	t.Run("나누어떨어지지 않으면 절사한다", func(t *testing.T) {
		got, err := domain.StraightLineAnnualExpense(10_000_000, 0, 3)
		require.NoError(t, err)
		assert.Equal(t, int64(3_333_333), got)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		_, err := domain.StraightLineAnnualExpense(-1, 0, 5)
		assert.ErrorIs(t, err, domain.ErrNegativeAcquisitionCost)

		_, err = domain.StraightLineAnnualExpense(10_000_000, 0, 0)
		assert.ErrorIs(t, err, domain.ErrInvalidUsefulLife)

		_, err = domain.StraightLineAnnualExpense(10_000_000, 20_000_000, 5)
		assert.ErrorIs(t, err, domain.ErrInvalidSalvageValue)
	})
}

func TestMonthsInFirstYear(t *testing.T) {
	// 법인세법 시행령 제26조 제7항: 월할계산, 취득한 달을 포함한다.
	tests := []struct {
		month    int
		expected int
	}{
		{1, 12},
		{3, 10},
		{7, 6},
		{12, 1},
	}
	for _, tt := range tests {
		got, err := domain.MonthsInFirstYear(tt.month)
		require.NoError(t, err)
		assert.Equal(t, tt.expected, got, "%d월 취득", tt.month)
	}

	_, err := domain.MonthsInFirstYear(0)
	assert.ErrorIs(t, err, domain.ErrInvalidDepreciationMonths)
	_, err = domain.MonthsInFirstYear(13)
	assert.ErrorIs(t, err, domain.ErrInvalidDepreciationMonths)
}

func TestStraightLineSchedule(t *testing.T) {
	t.Run("취득가액 1,000만원 / 내용연수 5년 / 잔존가액 0", func(t *testing.T) {
		schedule, err := domain.StraightLineSchedule(10_000_000, 0, 5, 12)
		require.NoError(t, err)
		require.Len(t, schedule, 5)

		for i := 0; i < 4; i++ {
			assert.Equal(t, int64(2_000_000), schedule[i].Expense, "%d년차", i+1)
		}
		// 마지막 해에는 비망가액 1,000원을 남긴다.
		assert.Equal(t, int64(1_999_000), schedule[4].Expense)
		assert.Equal(t, domain.MemorandumValue, schedule[4].BookValue)
		assert.Equal(t, int64(9_999_000), schedule[4].AccumulatedDepreciation)

		assertScheduleConsistent(t, 10_000_000, schedule)
	})

	t.Run("잔존가액이 있으면 그 금액에서 멈춘다", func(t *testing.T) {
		schedule, err := domain.StraightLineSchedule(10_000_000, 1_000_000, 5, 12)
		require.NoError(t, err)
		require.Len(t, schedule, 5)

		assert.Equal(t, int64(1_800_000), schedule[4].Expense)
		assert.Equal(t, int64(1_000_000), schedule[4].BookValue)
		assert.Equal(t, int64(9_000_000), schedule[4].AccumulatedDepreciation)

		assertScheduleConsistent(t, 10_000_000, schedule)
	})

	t.Run("사업연도 중 취득 시 첫 해를 월할계산한다", func(t *testing.T) {
		// 3월 취득 -> 첫 해 10개월.
		months, err := domain.MonthsInFirstYear(3)
		require.NoError(t, err)

		schedule, err := domain.StraightLineSchedule(12_000_000, 0, 5, months)
		require.NoError(t, err)

		// 첫 해가 짧아진 만큼 상각이 한 해 더 이어진다.
		require.Len(t, schedule, 6)
		assert.Equal(t, 10, schedule[0].Months)
		assert.Equal(t, int64(2_000_000), schedule[0].Expense, "2,400,000 x 10/12")
		assert.Equal(t, int64(2_400_000), schedule[1].Expense)
		assert.Equal(t, domain.MemorandumValue, schedule[5].BookValue)

		assertScheduleConsistent(t, 12_000_000, schedule)
	})

	t.Run("취득가액이 비망가액 이하이면 상각하지 않는다", func(t *testing.T) {
		schedule, err := domain.StraightLineSchedule(500, 0, 5, 12)
		require.NoError(t, err)
		assert.Empty(t, schedule)
	})
}

func TestDecliningBalanceSchedule(t *testing.T) {
	t.Run("취득가액 1,000만원 / 내용연수 5년 (상각률 0.451)", func(t *testing.T) {
		schedule, err := domain.DecliningBalanceSchedule(10_000_000, 5, 12)
		require.NoError(t, err)
		require.Len(t, schedule, 6)

		// 매년 미상각잔액 x 0.451.
		expected := []struct {
			expense   int64
			bookValue int64
		}{
			{4_510_000, 5_490_000},
			{2_475_990, 3_014_010},
			{1_359_318, 1_654_692},
			{746_266, 908_426},
			{409_700, 498_726},
			// 미상각잔액 498,726이 취득가액의 5%(500,000) 이하가 되었으므로
			// 이 사업연도에 비망가액만 남기고 전액 상각한다.
			{497_726, domain.MemorandumValue},
		}
		for i, want := range expected {
			assert.Equal(t, want.expense, schedule[i].Expense, "%d년차 상각비", i+1)
			assert.Equal(t, want.bookValue, schedule[i].BookValue, "%d년차 장부가액", i+1)
		}

		assertScheduleConsistent(t, 10_000_000, schedule)
	})

	t.Run("상각비는 매년 감소한다", func(t *testing.T) {
		schedule, err := domain.DecliningBalanceSchedule(50_000_000, 8, 12)
		require.NoError(t, err)
		require.NotEmpty(t, schedule)

		// The final year is the write-off of the remaining balance, which may
		// exceed the previous year's charge, so it is excluded.
		for i := 1; i < len(schedule)-1; i++ {
			assert.Less(t, schedule[i].Expense, schedule[i-1].Expense,
				"%d년차 상각비가 전년보다 커졌다", i+1)
		}
		assert.Equal(t, domain.MemorandumValue, schedule[len(schedule)-1].BookValue)

		assertScheduleConsistent(t, 50_000_000, schedule)
	})

	t.Run("사업연도 중 취득 시 첫 해를 월할계산한다", func(t *testing.T) {
		full, err := domain.DecliningBalanceSchedule(10_000_000, 5, 12)
		require.NoError(t, err)
		partial, err := domain.DecliningBalanceSchedule(10_000_000, 5, 6)
		require.NoError(t, err)

		assert.Equal(t, full[0].Expense/2, partial[0].Expense, "6개월분은 12개월분의 절반")
		assertScheduleConsistent(t, 10_000_000, partial)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		_, err := domain.DecliningBalanceSchedule(-1, 5, 12)
		assert.ErrorIs(t, err, domain.ErrNegativeAcquisitionCost)

		_, err = domain.DecliningBalanceSchedule(10_000_000, 0, 12)
		assert.ErrorIs(t, err, domain.ErrInvalidUsefulLife)

		_, err = domain.DecliningBalanceSchedule(10_000_000, 5, 0)
		assert.ErrorIs(t, err, domain.ErrInvalidDepreciationMonths)
	})
}

// assertScheduleConsistent checks the invariants that must hold for any
// depreciation schedule regardless of method: the accumulated depreciation is
// the running sum of the charges, it never exceeds the acquisition cost, and
// cost - accumulated is the book value on every line.
func assertScheduleConsistent(t *testing.T, acquisitionCost int64, schedule []domain.DepreciationPeriod) {
	t.Helper()

	var running int64
	for i, period := range schedule {
		assert.Equal(t, i+1, period.Year, "연차는 1부터 연속이어야 한다")
		assert.Positive(t, period.Expense, "%d년차 상각비가 0 이하", i+1)

		running += period.Expense
		assert.Equal(t, running, period.AccumulatedDepreciation,
			"%d년차 감가상각누계액", i+1)
		assert.Equal(t, acquisitionCost-running, period.BookValue,
			"%d년차 장부가액 = 취득가액 - 누계액", i+1)
		assert.GreaterOrEqual(t, period.BookValue, int64(0),
			"장부가액이 음수가 되었다")
	}
}
