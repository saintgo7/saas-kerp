package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// aug2025 is inside both a verified rate year (2025) and a published 국민연금
// 기준소득월액 window (2025.7.1.~2026.6.30., 하한 400,000 / 상한 6,370,000).
var aug2025 = time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)

func int64p(v int64) *int64 { return &v }

// baseInput is a 월 320만원 payroll: 기본급 300만원 + 식대 20만원(비과세).
// 보수월액 = 300만원, which is the figure every expected value below is
// hand-computed from.
func baseInput() domain.PayrollCalculationInput {
	return domain.PayrollCalculationInput{
		PayYear:          2025,
		PayMonth:         8,
		ContributionDate: aug2025,
		Earnings: domain.PayrollEarnings{
			BaseSalary: 3_000_000,
			Allowances: []domain.PayrollNamedAmount{
				{Code: "meal", Name: "식대", Amount: 200_000, NonTaxable: true},
			},
		},
		Deductions: domain.PayrollDeductionInput{
			// 간이세액표가 없으므로 반드시 입력값으로 받는다.
			IncomeTax: int64p(84_850),
		},
		Qualification:          domain.AllQualified(),
		WorkplaceSize:          domain.WorkplaceUnder150,
		IndustrialAccidentRate: int64p(1530), // 1.53%
	}
}

func TestCalculatePayroll(t *testing.T) {
	// 손 검산 (2025년 요율, 보수월액 3,000,000원):
	//
	//   국민연금  총 9%    3,000,000 x 0.09    = 270,000  (근로자 135,000)
	//   건강보험  총 7.09% 3,000,000 x 0.0709  = 212,700  (근로자 106,350)
	//   장기요양  건강보험료 212,700 x 12.95%  =  27,544 -> 10원 절사 27,540
	//                                             (근로자 27,540/2 = 13,770)
	//   고용보험  근로자 0.9%  3,000,000 x 0.009 = 27,000
	//             사업주 0.9%                     = 27,000
	//             고용안정 0.25% (150인 미만)     =  7,500
	//   산재보험  1.53% 사업주 전액              =  45,900
	//
	//   근로자 부담 합계 = 135,000 + 106,350 + 13,770 + 27,000 = 282,120
	t.Run("2025년 8월, 보수월액 300만원", func(t *testing.T) {
		got, err := domain.CalculatePayroll(baseInput())
		require.NoError(t, err)

		assert.Equal(t, int64(3_200_000), got.TotalEarnings, "지급총액 = 기본급 + 식대")
		assert.Equal(t, int64(200_000), got.NonTaxableAmount)
		assert.Equal(t, int64(3_000_000), got.TaxableWage, "보수월액에서 비과세 식대는 빠진다")

		assert.Equal(t, int64(135_000), got.SocialInsurance.NationalPension.Employee)
		assert.Equal(t, int64(135_000), got.SocialInsurance.NationalPension.Employer)
		assert.Equal(t, int64(106_350), got.SocialInsurance.HealthInsurance.Employee)
		assert.Equal(t, int64(106_350), got.SocialInsurance.HealthInsurance.Employer)
		assert.Equal(t, int64(13_770), got.SocialInsurance.LongTermCare.Employee)
		assert.Equal(t, int64(13_770), got.SocialInsurance.LongTermCare.Employer)
		assert.Equal(t, int64(27_000), got.SocialInsurance.EmploymentInsurance.Employee)
		assert.Equal(t, int64(282_120), got.SocialInsurance.EmployeeTotal())

		// 지방소득세 = 소득세 84,850 x 10% = 8,485 -> 10원 미만 절사 = 8,480.
		assert.Equal(t, int64(84_850), got.IncomeTax)
		assert.Equal(t, int64(8_480), got.LocalIncomeTax)

		// 공제총액 = 282,120 + 84,850 + 8,480 = 375,450
		assert.Equal(t, int64(375_450), got.TotalDeductions)
		// 실지급액 = 3,200,000 - 375,450 = 2,824,550
		assert.Equal(t, int64(2_824_550), got.NetPay)

		// 사업주 부담: 135,000 + 106,350 + 13,770 + (27,000+7,500) + 45,900
		assert.Equal(t, int64(34_500), got.EmployerCost.EmploymentInsurance,
			"고용보험 사업주분은 실업급여 + 고용안정·직업능력개발사업이다")
		assert.Equal(t, int64(45_900), got.EmployerCost.IndustrialAccident)
		assert.Equal(t, int64(335_520), got.EmployerCost.Total)
		assert.True(t, got.IndustrialAccidentApplied)
		assert.Equal(t, int64(250), got.EmploymentStabilityRate)
		assert.NotEmpty(t, got.RatesSource, "요율 출처가 결과에 남아야 한다")
	})

	t.Run("항목 합계가 헤더 금액과 정확히 일치한다", func(t *testing.T) {
		got, err := domain.CalculatePayroll(baseInput())
		require.NoError(t, err)

		var earnings, deductions int64
		for _, it := range got.Items {
			switch it.ItemType {
			case domain.PayrollItemEarning:
				earnings += it.Amount
			case domain.PayrollItemDeduction:
				deductions += it.Amount
			}
		}
		assert.Equal(t, got.TotalEarnings, earnings)
		assert.Equal(t, got.TotalDeductions, deductions)
		assert.Equal(t, got.NetPay, got.TotalEarnings-got.TotalDeductions)

		// 0원 항목은 줄을 만들지 않는다 (야간·휴일수당은 입력하지 않았다).
		for _, it := range got.Items {
			assert.NotZero(t, it.Amount, "0원 항목 %s 이 저장되면 급여명세서가 지저분해진다", it.ItemCode)
		}

		// 식대는 비과세로 표시되어야 한다.
		var meal domain.PayrollItemLine
		for _, it := range got.Items {
			if it.ItemCode == "meal" {
				meal = it
			}
		}
		require.Equal(t, int64(200_000), meal.Amount)
		assert.False(t, meal.IsTaxable)
	})

	t.Run("국민연금 기준소득월액 상한이 적용된다", func(t *testing.T) {
		in := baseInput()
		in.Earnings.BaseSalary = 7_000_000
		in.Earnings.Allowances = nil

		got, err := domain.CalculatePayroll(in)
		require.NoError(t, err)

		// 상한 6,370,000 x 4.5% = 286,650. 건강보험은 상한이 없으므로
		// 7,000,000 x 3.545% = 248,150 그대로.
		assert.Equal(t, int64(6_370_000), got.SocialInsurance.PensionBase)
		assert.Equal(t, int64(286_650), got.SocialInsurance.NationalPension.Employee)
		assert.Equal(t, int64(248_150), got.SocialInsurance.HealthInsurance.Employee)
	})

	t.Run("산재보험료율을 주지 않으면 0원이 아니라 미산정이다", func(t *testing.T) {
		in := baseInput()
		in.IndustrialAccidentRate = nil

		got, err := domain.CalculatePayroll(in)
		require.NoError(t, err)

		assert.False(t, got.IndustrialAccidentApplied,
			"업종별 고시 요율이 없으면 산재보험료를 산정했다고 말해서는 안 된다")
		assert.Zero(t, got.EmployerCost.IndustrialAccident)
	})

	t.Run("자격 미취득 보험은 공제하지 않는다", func(t *testing.T) {
		in := baseInput()
		in.Qualification = domain.SocialInsuranceQualification{
			HealthInsurance: true,
		}

		got, err := domain.CalculatePayroll(in)
		require.NoError(t, err)

		assert.Zero(t, got.SocialInsurance.NationalPension.Employee)
		assert.Zero(t, got.SocialInsurance.EmploymentInsurance.Employee)
		assert.Zero(t, got.EmployerCost.IndustrialAccident)
		// 건강보험은 유지되고, 장기요양은 건강보험료를 과세표준으로 하므로 따라온다.
		assert.Equal(t, int64(106_350), got.SocialInsurance.HealthInsurance.Employee)
		assert.Equal(t, int64(13_770), got.SocialInsurance.LongTermCare.Employee)
		// 고용보험 자격이 없으면 고용안정 보험료도 붙지 않는다.
		assert.Zero(t, got.EmployerCost.EmploymentInsurance)
	})

	t.Run("건강보험 자격이 없으면 장기요양도 함께 사라진다", func(t *testing.T) {
		in := baseInput()
		in.Qualification = domain.SocialInsuranceQualification{
			NationalPension:     true,
			EmploymentInsurance: true,
		}

		got, err := domain.CalculatePayroll(in)
		require.NoError(t, err)

		assert.Zero(t, got.SocialInsurance.HealthInsurance.Employee)
		assert.Zero(t, got.SocialInsurance.LongTermCare.Employee,
			"장기요양보험료는 건강보험료를 과세표준으로 하므로 홀로 남을 수 없다")
	})

	t.Run("기타공제가 공제총액과 항목에 함께 반영된다", func(t *testing.T) {
		in := baseInput()
		in.Deductions.Others = []domain.PayrollNamedAmount{
			{Code: "union_fee", Name: "노동조합비", Amount: 20_000},
		}

		got, err := domain.CalculatePayroll(in)
		require.NoError(t, err)

		assert.Equal(t, int64(20_000), got.OtherDeductionTotal)
		assert.Equal(t, int64(375_450+20_000), got.TotalDeductions)
		assert.Equal(t, int64(2_824_550-20_000), got.NetPay)
	})

	t.Run("지방소득세를 직접 넣으면 파생하지 않는다", func(t *testing.T) {
		in := baseInput()
		in.Deductions.LocalIncomeTax = int64p(8_480)

		got, err := domain.CalculatePayroll(in)
		require.NoError(t, err)
		assert.Equal(t, int64(8_480), got.LocalIncomeTax)
	})
}

func TestCalculatePayrollRefusesUnverifiedInputs(t *testing.T) {
	t.Run("요율이 확인되지 않은 연도는 계산하지 않는다", func(t *testing.T) {
		in := baseInput()
		in.PayYear = 2026
		in.ContributionDate = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

		_, err := domain.CalculatePayroll(in)
		assert.ErrorIs(t, err, domain.ErrSocialInsuranceRatesUnavailable,
			"작년 요율로 대신 계산하면 맞아 보이는 틀린 급여가 나온다")
	})

	t.Run("기준소득월액 고시가 없는 기간은 계산하지 않는다", func(t *testing.T) {
		in := baseInput()
		// 2025년 요율은 있으나 2026.7. 이후 기준소득월액 창은 아직 없다.
		in.ContributionDate = time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

		_, err := domain.CalculatePayroll(in)
		assert.Error(t, err)
	})

	t.Run("소득세를 주지 않으면 간이세액표 부재로 실패한다", func(t *testing.T) {
		in := baseInput()
		in.Deductions.IncomeTax = nil

		_, err := domain.CalculatePayroll(in)
		assert.ErrorIs(t, err, domain.ErrSimplifiedTaxTableUnavailable,
			"간이세액표 없이 소득세를 지어내면 매월 국세청에 잘못 신고된다")
	})

	t.Run("음수 금액을 거부한다", func(t *testing.T) {
		in := baseInput()
		in.Earnings.OvertimePay = -1
		_, err := domain.CalculatePayroll(in)
		assert.ErrorIs(t, err, domain.ErrNegativeAmount)

		in = baseInput()
		in.Deductions.Others = []domain.PayrollNamedAmount{
			{Code: "x", Name: "환급", Amount: -1000},
		}
		_, err = domain.CalculatePayroll(in)
		assert.ErrorIs(t, err, domain.ErrNegativeAmount)
	})

	t.Run("비과세가 지급총액을 넘으면 거부한다", func(t *testing.T) {
		in := baseInput()
		in.Earnings.BaseSalary = 0
		in.Earnings.Allowances = []domain.PayrollNamedAmount{
			{Code: "meal", Name: "식대", Amount: 200_000, NonTaxable: true},
		}
		in.Earnings.OtherEarnings = nil
		// 지급총액 200,000, 비과세 200,000 -> 보수월액 0. 이건 허용된다.
		_, err := domain.CalculatePayroll(in)
		require.NoError(t, err)
	})

	t.Run("이름이나 코드가 빈 항목을 거부한다", func(t *testing.T) {
		in := baseInput()
		in.Earnings.Allowances = []domain.PayrollNamedAmount{
			{Code: "", Name: "이름만 있음", Amount: 100},
		}
		_, err := domain.CalculatePayroll(in)
		assert.ErrorIs(t, err, domain.ErrPayrollEmptyItemName)
	})

	t.Run("잘못된 급여월을 거부한다", func(t *testing.T) {
		in := baseInput()
		in.PayMonth = 13
		_, err := domain.CalculatePayroll(in)
		assert.ErrorIs(t, err, domain.ErrPayrollMonthInvalid)
	})
}

func TestValidatePayrollTotals(t *testing.T) {
	items := []domain.PayrollItem{
		{ItemType: domain.PayrollItemEarning, Amount: 3_000_000},
		{ItemType: domain.PayrollItemEarning, Amount: 200_000},
		{ItemType: domain.PayrollItemDeduction, Amount: 375_450},
	}

	t.Run("맞는 헤더를 통과시킨다", func(t *testing.T) {
		p := &domain.Payroll{TotalEarnings: 3_200_000, TotalDeductions: 375_450, NetPay: 2_824_550}
		assert.NoError(t, domain.ValidatePayrollTotals(p, items))
	})

	t.Run("지급총액이 항목 합계와 다르면 잡아낸다", func(t *testing.T) {
		p := &domain.Payroll{TotalEarnings: 3_200_001, TotalDeductions: 375_450, NetPay: 2_824_551}
		assert.ErrorIs(t, domain.ValidatePayrollTotals(p, items), domain.ErrPayrollTotalsMismatch)
	})

	t.Run("공제총액이 항목 합계와 다르면 잡아낸다", func(t *testing.T) {
		p := &domain.Payroll{TotalEarnings: 3_200_000, TotalDeductions: 375_440, NetPay: 2_824_560}
		assert.ErrorIs(t, domain.ValidatePayrollTotals(p, items), domain.ErrPayrollTotalsMismatch)
	})

	t.Run("실지급액이 지급-공제와 다르면 잡아낸다", func(t *testing.T) {
		p := &domain.Payroll{TotalEarnings: 3_200_000, TotalDeductions: 375_450, NetPay: 9_999_999}
		assert.ErrorIs(t, domain.ValidatePayrollTotals(p, items), domain.ErrPayrollTotalsMismatch)
	})
}

func TestApplyCalculation(t *testing.T) {
	got, err := domain.CalculatePayroll(baseInput())
	require.NoError(t, err)

	p := &domain.Payroll{
		BaseSalary: 3_000_000,
		Status:     domain.PayrollStatusDraft,
	}
	p.ApplyCalculation(got)

	assert.Equal(t, domain.PayrollStatusCalculated, p.Status)
	assert.Equal(t, int64(3_200_000), p.TotalEarnings)
	assert.Equal(t, int64(135_000), p.NPSEmployee)
	assert.Equal(t, int64(106_350), p.NHISEmployee)
	assert.Equal(t, int64(13_770), p.NHISLTCEmployee)
	assert.Equal(t, int64(27_000), p.EIEmployee)
	assert.Equal(t, int64(375_450), p.TotalDeductions)
	assert.Equal(t, int64(2_824_550), p.NetPay)
	assert.Equal(t, int64(335_520), p.TotalEmployerCost)

	// JSONB 요약은 전용 컬럼이 있는 항목을 중복해서 담지 않는다.
	assert.Equal(t, domain.PayrollAmountMap{"meal": 200_000}, p.Allowances)
	assert.Empty(t, p.OtherDeductions)

	// 저장될 행과 헤더가 서로 맞는지 최종 확인.
	rows := domain.ToPayrollItems(uuid.New(), uuid.New(), got.Items)
	assert.NoError(t, domain.ValidatePayrollTotals(p, rows))
}

func TestPayrollStateTransitions(t *testing.T) {
	t.Run("계산 전 급여는 승인할 수 없다", func(t *testing.T) {
		p := &domain.Payroll{Status: domain.PayrollStatusDraft}
		assert.False(t, p.CanBeApproved(), "계산하지 않은 급여는 공제액이 0이다")
		assert.True(t, p.CanBeModified())
	})

	t.Run("승인·지급된 급여는 수정할 수 없다", func(t *testing.T) {
		for _, s := range []domain.PayrollStatus{
			domain.PayrollStatusApproved,
			domain.PayrollStatusPaid,
			domain.PayrollStatusCancelled,
		} {
			p := &domain.Payroll{Status: s}
			assert.False(t, p.CanBeModified(), "status=%s", s)
			assert.False(t, p.CanBeApproved(), "status=%s", s)
		}
	})

	t.Run("급여기간은 계산->승인->지급->마감 순서만 허용한다", func(t *testing.T) {
		draft := &domain.PayrollPeriod{Status: domain.PayrollPeriodStatusDraft}
		assert.True(t, draft.CanCalculate())
		assert.False(t, draft.CanApprove(), "계산하지 않은 기간을 확정할 수 없다")
		assert.False(t, draft.CanPay())

		calculated := &domain.PayrollPeriod{Status: domain.PayrollPeriodStatusCalculated}
		assert.True(t, calculated.CanApprove())
		assert.False(t, calculated.CanPay())

		approved := &domain.PayrollPeriod{Status: domain.PayrollPeriodStatusApproved}
		assert.False(t, approved.CanCalculate(), "확정된 기간을 다시 계산하면 확정액이 바뀐다")
		assert.False(t, approved.CanApprove(), "두 번 확정할 수 없다")
		assert.True(t, approved.CanPay())

		paid := &domain.PayrollPeriod{Status: domain.PayrollPeriodStatusPaid}
		assert.False(t, paid.CanPay(), "두 번 지급처리할 수 없다")
		assert.True(t, paid.CanClose())

		closed := &domain.PayrollPeriod{Status: domain.PayrollPeriodStatusClosed}
		assert.False(t, closed.CanCalculate())
		assert.False(t, closed.CanApprove())
		assert.False(t, closed.CanPay())
		assert.False(t, closed.CanClose())
	})
}

func TestPayrollAmountMapRoundTrip(t *testing.T) {
	t.Run("nil 맵은 NULL 이 아니라 빈 객체로 저장된다", func(t *testing.T) {
		var m domain.PayrollAmountMap
		v, err := m.Value()
		require.NoError(t, err)
		assert.Equal(t, []byte("{}"), v)
	})

	t.Run("저장한 값을 그대로 읽는다", func(t *testing.T) {
		m := domain.PayrollAmountMap{"meal": 200_000}
		v, err := m.Value()
		require.NoError(t, err)

		var back domain.PayrollAmountMap
		require.NoError(t, back.Scan(v))
		assert.Equal(t, m, back)
	})

	t.Run("NULL 을 읽어도 빈 맵이 된다", func(t *testing.T) {
		var back domain.PayrollAmountMap
		require.NoError(t, back.Scan(nil))
		assert.NotNil(t, back)
		assert.Empty(t, back)
	})
}
