package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

func TestSocialInsuranceRatesFor(t *testing.T) {
	t.Run("returns a verified year", func(t *testing.T) {
		rates, err := domain.SocialInsuranceRatesFor(2025)
		require.NoError(t, err)

		// 국민연금법 제88조: 9%, 노사 각 4.5%.
		assert.Equal(t, int64(9000), rates.NationalPensionTotal)
		assert.Equal(t, int64(4500), rates.NationalPensionEmployee)
		// The employee share must be exactly half of the total, otherwise the
		// employer share derived as (total - employee) is silently wrong.
		assert.Equal(t, rates.NationalPensionTotal, rates.NationalPensionEmployee*2)
		assert.Equal(t, rates.HealthInsuranceTotal, rates.HealthInsuranceEmployee*2)

		assert.NotEmpty(t, rates.Source, "every rate row must name its source")
	})

	t.Run("refuses a year that has not been verified", func(t *testing.T) {
		// Deliberate: falling back to the previous year's rate produces payroll
		// that looks right and is wrong. See the CONFIRM block in
		// social_insurance.go.
		_, err := domain.SocialInsuranceRatesFor(2026)
		assert.ErrorIs(t, err, domain.ErrSocialInsuranceRatesUnavailable)
	})

	t.Run("every row in the table names its source", func(t *testing.T) {
		latest := domain.LatestConfirmedSocialInsuranceYear()
		require.GreaterOrEqual(t, latest, 2024)
		rates, err := domain.SocialInsuranceRatesFor(latest)
		require.NoError(t, err)
		assert.NotEmpty(t, rates.Source)
	})
}

func TestPensionBaseLimits(t *testing.T) {
	t.Run("2025년 7월 이후 적용분", func(t *testing.T) {
		minBase, maxBase, err := domain.PensionBaseLimits(time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		assert.Equal(t, int64(400_000), minBase)
		assert.Equal(t, int64(6_370_000), maxBase)
	})

	t.Run("2023년 7월 ~ 2024년 6월 적용분", func(t *testing.T) {
		minBase, maxBase, err := domain.PensionBaseLimits(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		assert.Equal(t, int64(370_000), minBase)
		assert.Equal(t, int64(5_900_000), maxBase)
	})

	t.Run("2024년 7월 ~ 2025년 6월 적용분", func(t *testing.T) {
		minBase, maxBase, err := domain.PensionBaseLimits(time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		assert.Equal(t, int64(390_000), minBase)
		assert.Equal(t, int64(6_170_000), maxBase)
	})

	t.Run("고시되지 않은 기간은 실패한다", func(t *testing.T) {
		_, _, err := domain.PensionBaseLimits(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
		assert.ErrorIs(t, err, domain.ErrPensionBaseUnavailable)
	})
}

func TestCalculateSocialInsurance(t *testing.T) {
	on := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)

	t.Run("보수월액 300만원, 2025년 요율", func(t *testing.T) {
		got, err := domain.CalculateSocialInsurance(3_000_000, on)
		require.NoError(t, err)

		// 국민연금 9% = 270,000 (노사 각 135,000)
		assert.Equal(t, int64(3_000_000), got.PensionBase)
		assert.Equal(t, int64(135_000), got.NationalPension.Employee)
		assert.Equal(t, int64(135_000), got.NationalPension.Employer)
		assert.Equal(t, int64(270_000), got.NationalPension.Total())

		// 건강보험 7.09% = 212,700 (노사 각 106,350)
		assert.Equal(t, int64(106_350), got.HealthInsurance.Employee)
		assert.Equal(t, int64(212_700), got.HealthInsurance.Total())

		// 장기요양 = 건강보험료 212,700 x 12.95% = 27,544.65 -> 27,540
		assert.Equal(t, int64(27_540), got.LongTermCare.Total())
		assert.Equal(t, int64(13_770), got.LongTermCare.Employee)
		assert.Equal(t, int64(13_770), got.LongTermCare.Employer)

		// 고용보험 실업급여 0.9% = 27,000 each
		assert.Equal(t, int64(27_000), got.EmploymentInsurance.Employee)
		assert.Equal(t, int64(27_000), got.EmploymentInsurance.Employer)

		// 근로자 공제 합계
		assert.Equal(t, int64(282_120), got.EmployeeTotal())
	})

	t.Run("국민연금 기준소득월액 상한이 적용된다", func(t *testing.T) {
		got, err := domain.CalculateSocialInsurance(10_000_000, on)
		require.NoError(t, err)

		// 상한 6,370,000이 적용되므로 연금은 급여가 아니라 상한액 기준이다.
		assert.Equal(t, int64(6_370_000), got.PensionBase)
		assert.Equal(t, int64(286_650), got.NationalPension.Employee)

		// 건강보험은 상한 적용 없이 보수월액 기준이다.
		assert.Equal(t, int64(354_500), got.HealthInsurance.Employee)
	})

	t.Run("국민연금 기준소득월액 하한이 적용된다", func(t *testing.T) {
		got, err := domain.CalculateSocialInsurance(300_000, on)
		require.NoError(t, err)

		assert.Equal(t, int64(400_000), got.PensionBase)
		assert.Equal(t, int64(18_000), got.NationalPension.Employee)
	})

	t.Run("노사 부담분의 합은 항상 총액과 같다", func(t *testing.T) {
		for _, wage := range []int64{0, 1_234_567, 3_000_000, 8_888_888} {
			got, err := domain.CalculateSocialInsurance(wage, on)
			require.NoError(t, err)
			assert.Equal(t,
				got.HealthInsurance.Employee+got.HealthInsurance.Employer,
				got.HealthInsurance.Total())
			assert.Equal(t,
				got.LongTermCare.Employee+got.LongTermCare.Employer,
				got.LongTermCare.Total())
		}
	})

	t.Run("확인되지 않은 연도는 계산하지 않는다", func(t *testing.T) {
		_, err := domain.CalculateSocialInsurance(3_000_000, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
		assert.ErrorIs(t, err, domain.ErrSocialInsuranceRatesUnavailable)
	})

	t.Run("음수 보수월액은 거부한다", func(t *testing.T) {
		_, err := domain.CalculateSocialInsurance(-1, on)
		assert.ErrorIs(t, err, domain.ErrNegativeMonthlyWage)
	})
}

func TestCalculateIndustrialAccident(t *testing.T) {
	t.Run("사업주가 전액 부담한다", func(t *testing.T) {
		// 요율은 업종별 고시값이므로 호출자가 넘긴다. 1.53% -> 1530.
		got, err := domain.CalculateIndustrialAccident(3_000_000, 1530)
		require.NoError(t, err)

		assert.Equal(t, int64(0), got.Employee)
		assert.Equal(t, int64(45_900), got.Employer)
	})

	t.Run("음수 요율은 거부한다", func(t *testing.T) {
		_, err := domain.CalculateIndustrialAccident(3_000_000, -1)
		assert.Error(t, err)
	})
}

func TestEmploymentStabilityRate(t *testing.T) {
	rates, err := domain.SocialInsuranceRatesFor(2025)
	require.NoError(t, err)

	// 고용보험법 시행령 제12조. 사업주 전액 부담이며 규모가 클수록 높다.
	assert.Equal(t, int64(250), rates.EmploymentStabilityRate(domain.WorkplaceUnder150))
	assert.Equal(t, int64(450), rates.EmploymentStabilityRate(domain.Workplace150Priority))
	assert.Equal(t, int64(650), rates.EmploymentStabilityRate(domain.Workplace150To999))
	assert.Equal(t, int64(850), rates.EmploymentStabilityRate(domain.Workplace1000Plus))

	// CalculateSocialInsurance 는 사업장 규모를 모르므로 이 항목을 더하지
	// 않는다. 사업주 부담분에 실업급여분만 들어 있음을 못박아 둔다.
	got, err := domain.CalculateSocialInsurance(3_000_000, time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, int64(27_000), got.EmploymentInsurance.Employer,
		"실업급여 0.9% 만. 고용안정분은 호출자가 따로 더해야 한다")
}
