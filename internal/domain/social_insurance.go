package domain

import (
	"errors"
	"fmt"
	"time"
)

// 4대보험 (Korean social insurance) contribution rates and calculations.
//
// RATES ARE NEVER HARD-CODED AT THE CALL SITE. They live in the year-keyed
// tables below, each entry carrying the authority that set it and the period
// it applies to, because every one of them is revised on its own schedule:
// 건강보험료율 is set annually by 건강보험정책심의위원회, the 국민연금
// 기준소득월액 ceiling and floor change every July, and 산재보험료율 is
// published per industry.
//
// A year with no verified entry returns ErrSocialInsuranceRatesUnavailable
// rather than falling back to the closest year. Silently applying last year's
// rate produces payroll that looks right and is wrong, which is worse than a
// visible failure - see the CONFIRM notes at the bottom of this file.

// Social insurance errors
var (
	ErrSocialInsuranceRatesUnavailable = errors.New("확인된 4대보험 요율이 없는 연도입니다")
	ErrPensionBaseUnavailable          = errors.New("확인된 국민연금 기준소득월액 상·하한이 없는 기간입니다")
	ErrNegativeMonthlyWage             = errors.New("monthly wage must not be negative")
)

// SocialInsuranceRates holds the contribution rates that apply to one year.
//
// Every rate is stored in HUNDRED-THOUSANDTHS (1/100000), which is the
// smallest unit any of these rates is published in: 9% = 9000,
// 7.09% = 7090, 3.545% = 3545, 0.9% = 900, 12.95% = 12950. Basis points would
// not represent 3.545% exactly, and a float64 rate would reintroduce the
// rounding problem this package exists to avoid.
type SocialInsuranceRates struct {
	Year int

	// NationalPensionTotal 국민연금 총 요율 (근로자 + 사업주).
	NationalPensionTotal int64
	// NationalPensionEmployee 국민연금 근로자 부담 요율.
	NationalPensionEmployee int64

	// HealthInsuranceTotal 건강보험 총 요율 (보수월액 대비).
	HealthInsuranceTotal int64
	// HealthInsuranceEmployee 건강보험 근로자 부담 요율.
	HealthInsuranceEmployee int64

	// LongTermCareOfHealth 장기요양보험료율: 건강보험료 대비 비율.
	// 노인장기요양보험법 제9조 - 장기요양보험료는 건강보험료액에
	// 장기요양보험료율을 곱하여 산정한다. 보수월액이 아니라 건강보험료가
	// 과세표준이라는 점이 다른 항목과 다르다.
	LongTermCareOfHealth int64

	// EmploymentInsuranceEmployee 고용보험 실업급여 근로자 부담 요율.
	EmploymentInsuranceEmployee int64
	// EmploymentInsuranceEmployer 고용보험 실업급여 사업주 부담 요율.
	// 사업주는 이에 더해 고용안정·직업능력개발사업 보험료를 전액 부담한다
	// (아래 EmploymentStability* 참조).
	EmploymentInsuranceEmployer int64

	// 고용안정·직업능력개발사업 보험료율. 사업주 전액 부담이며 상시근로자 수
	// 구간별로 다르다 (고용보험법 시행령 제12조).
	EmploymentStabilityUnder150    int64 // 150인 미만
	EmploymentStability150Priority int64 // 150인 이상 우선지원대상기업
	EmploymentStability150To999    int64 // 150인 이상 1,000인 미만
	EmploymentStability1000Plus    int64 // 1,000인 이상

	// CommuteAccidentRate 출퇴근재해 요율. 산재보험료율은 업종별이지만 이
	// 항목만은 전 업종 공통이다 (고용노동부 「사업종류별 산재보험료율」 고시).
	CommuteAccidentRate int64

	// Source records the authority and the announcement this row came from.
	Source string
}

// WorkplaceSize selects the 고용안정·직업능력개발사업 rate band.
type WorkplaceSize string

const (
	WorkplaceUnder150    WorkplaceSize = "under_150"
	Workplace150Priority WorkplaceSize = "priority_150_plus"
	Workplace150To999    WorkplaceSize = "from_150_to_999"
	Workplace1000Plus    WorkplaceSize = "over_1000"
)

// EmploymentStabilityRate returns the 고용안정·직업능력개발사업 rate for a
// workplace size band. An unrecognised band falls back to the smallest
// employer band, which carries the lowest rate, so callers must pass the real
// band rather than relying on the default.
func (r SocialInsuranceRates) EmploymentStabilityRate(size WorkplaceSize) int64 {
	switch size {
	case Workplace150Priority:
		return r.EmploymentStability150Priority
	case Workplace150To999:
		return r.EmploymentStability150To999
	case Workplace1000Plus:
		return r.EmploymentStability1000Plus
	default:
		return r.EmploymentStabilityUnder150
	}
}

// rateScale is the denominator for every rate in SocialInsuranceRates.
// A rate of 9000 means 9000/100000 = 9%.
const rateScale int64 = 100_000

// socialInsuranceRateTable is keyed by calendar year.
//
// Only years whose rates have been verified against the published source
// appear here. See the CONFIRM block at the bottom of the file for the years
// that are deliberately absent.
var socialInsuranceRateTable = map[int]SocialInsuranceRates{
	2024: {
		Year:                        2024,
		NationalPensionTotal:        9000,  // 9.0%
		NationalPensionEmployee:     4500,  // 4.5%
		HealthInsuranceTotal:        7090,  // 7.09%
		HealthInsuranceEmployee:     3545,  // 3.545%
		LongTermCareOfHealth:        12950, // 건강보험료의 12.95%
		EmploymentInsuranceEmployee: 900,   // 0.9%
		EmploymentInsuranceEmployer: 900,   // 0.9%
		// 고용안정·직업능력개발사업 (고용보험법 시행령 제12조)
		EmploymentStabilityUnder150:    250, // 0.25%
		EmploymentStability150Priority: 450, // 0.45%
		EmploymentStability150To999:    650, // 0.65%
		EmploymentStability1000Plus:    850, // 0.85%
		CommuteAccidentRate:            60,  // 0.06%
		Source: "국민연금법 제88조(9% 고정) / 국민건강보험법 시행령 제44조 " +
			"(2024년 보험료율 7.09%) / 노인장기요양보험법 시행령 제4조 " +
			"(2024년 건강보험료의 12.95%) / 고용보험법 시행령 제12조 " +
			"(2022.7.1. 개정, 실업급여 0.9%)",
	},
	2025: {
		Year:                           2025,
		NationalPensionTotal:           9000,
		NationalPensionEmployee:        4500,
		HealthInsuranceTotal:           7090, // 2024년과 동일 (동결)
		HealthInsuranceEmployee:        3545,
		LongTermCareOfHealth:           12950, // 동결
		EmploymentInsuranceEmployee:    900,
		EmploymentInsuranceEmployer:    900,
		EmploymentStabilityUnder150:    250,
		EmploymentStability150Priority: 450,
		EmploymentStability150To999:    650,
		EmploymentStability1000Plus:    850,
		CommuteAccidentRate:            60,
		Source: "국민연금법 제88조 / 2025년 건강보험료율 7.09% 동결 / " +
			"2025년 장기요양보험료율 건강보험료의 12.95% 동결 / " +
			"고용보험법 시행령 제12조",
	},
}

// pensionBasePeriod is one 국민연금 기준소득월액 ceiling/floor window.
// The window runs from 1 July to 30 June of the following year, so it is
// keyed by date rather than by calendar year.
type pensionBasePeriod struct {
	From   time.Time
	To     time.Time
	Max    int64 // 기준소득월액 상한액
	Min    int64 // 기준소득월액 하한액
	Source string
}

// pensionBasePeriods 국민연금 기준소득월액 상·하한 (국민연금법 제3조 제1항
// 제5호, 시행령 제5조). 보건복지부가 매년 3월 고시하고 그 해 7월부터 다음 해
// 6월까지 적용한다.
var pensionBasePeriods = []pensionBasePeriod{
	{
		From:   time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC),
		To:     time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
		Max:    5_900_000,
		Min:    370_000,
		Source: "보건복지부 고시, 2023.7.1.~2024.6.30. 적용",
	},
	{
		From:   time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		To:     time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
		Max:    6_170_000,
		Min:    390_000,
		Source: "보건복지부 고시, 2024.7.1.~2025.6.30. 적용",
	},
	{
		From:   time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		To:     time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		Max:    6_370_000,
		Min:    400_000,
		Source: "보건복지부 고시, 2025.7.1.~2026.6.30. 적용",
	},
}

// SocialInsuranceRatesFor returns the verified rates for a calendar year.
func SocialInsuranceRatesFor(year int) (SocialInsuranceRates, error) {
	rates, ok := socialInsuranceRateTable[year]
	if !ok {
		return SocialInsuranceRates{}, fmt.Errorf("%w: %d", ErrSocialInsuranceRatesUnavailable, year)
	}
	return rates, nil
}

// LatestConfirmedSocialInsuranceYear returns the most recent year present in
// the table. Callers that knowingly want to fall back to the last verified
// rates must do so explicitly through this function, so that the fallback is
// visible in the calling code and can be logged.
func LatestConfirmedSocialInsuranceYear() int {
	latest := 0
	for year := range socialInsuranceRateTable {
		if year > latest {
			latest = year
		}
	}
	return latest
}

// PensionBaseLimits returns the 기준소득월액 ceiling and floor applicable on a
// given date.
func PensionBaseLimits(on time.Time) (minBase, maxBase int64, err error) {
	day := time.Date(on.Year(), on.Month(), on.Day(), 0, 0, 0, 0, time.UTC)
	for _, p := range pensionBasePeriods {
		if !day.Before(p.From) && !day.After(p.To) {
			return p.Min, p.Max, nil
		}
	}
	return 0, 0, fmt.Errorf("%w: %s", ErrPensionBaseUnavailable, day.Format("2006-01-02"))
}

// SocialInsuranceContribution is the employee/employer split of one insurance.
type SocialInsuranceContribution struct {
	Employee int64 `json:"employee"`
	Employer int64 `json:"employer"`
}

// Total returns the combined contribution.
func (c SocialInsuranceContribution) Total() int64 {
	return c.Employee + c.Employer
}

// SocialInsuranceResult is the full 4대보험 calculation for one month.
type SocialInsuranceResult struct {
	// MonthlyWage 보수월액 as supplied.
	MonthlyWage int64 `json:"monthly_wage"`
	// PensionBase 국민연금 기준소득월액, after the ceiling and floor are applied.
	PensionBase int64 `json:"pension_base"`

	NationalPension     SocialInsuranceContribution `json:"national_pension"`
	HealthInsurance     SocialInsuranceContribution `json:"health_insurance"`
	LongTermCare        SocialInsuranceContribution `json:"long_term_care"`
	EmploymentInsurance SocialInsuranceContribution `json:"employment_insurance"`

	// IndustrialAccident 산재보험료. Borne entirely by the employer
	// (고용보험 및 산업재해보상보험의 보험료징수 등에 관한 법률 제13조).
	// Zero unless an industry rate was supplied - see CalculateIndustrialAccident.
	IndustrialAccident SocialInsuranceContribution `json:"industrial_accident"`

	// RatesSource echoes the source of the rate row used.
	RatesSource string `json:"rates_source"`
}

// EmployeeTotal returns everything withheld from the employee's pay.
func (r SocialInsuranceResult) EmployeeTotal() int64 {
	return r.NationalPension.Employee +
		r.HealthInsurance.Employee +
		r.LongTermCare.Employee +
		r.EmploymentInsurance.Employee
}

// EmployerTotal returns everything the employer contributes.
func (r SocialInsuranceResult) EmployerTotal() int64 {
	return r.NationalPension.Employer +
		r.HealthInsurance.Employer +
		r.LongTermCare.Employer +
		r.EmploymentInsurance.Employer +
		r.IndustrialAccident.Employer
}

// CalculateSocialInsurance computes the four insurances for one month.
//
// on is the date the contribution relates to; it selects both the annual rate
// row and the 국민연금 기준소득월액 window, which do not change on the same
// schedule.
//
// Rounding: each contribution is truncated below 10 won, which is how the
// 공단 notices are denominated. The employee share is computed from the base
// and the employer share is taken as (total - employee) so that the two halves
// always add up to the amount actually billed.
//
// CONFIRM / 확인 필요: 공단 고지금액의 단수처리(원 단위 절사 vs 10원 미만
// 절사)는 보험별로 다를 수 있다. 실제 고지서와 대사한 뒤 확정할 것.
func CalculateSocialInsurance(monthlyWage int64, on time.Time) (SocialInsuranceResult, error) {
	if monthlyWage < 0 {
		return SocialInsuranceResult{}, ErrNegativeMonthlyWage
	}

	rates, err := SocialInsuranceRatesFor(on.Year())
	if err != nil {
		return SocialInsuranceResult{}, err
	}

	minBase, maxBase, err := PensionBaseLimits(on)
	if err != nil {
		return SocialInsuranceResult{}, err
	}

	// 국민연금: 기준소득월액에 상·하한을 적용한 뒤 요율을 곱한다.
	pensionBase := monthlyWage
	if pensionBase > maxBase {
		pensionBase = maxBase
	}
	if pensionBase < minBase {
		pensionBase = minBase
	}

	pension := splitContribution(
		applyRate(pensionBase, rates.NationalPensionTotal),
		applyRate(pensionBase, rates.NationalPensionEmployee),
	)

	// 건강보험: 보수월액 기준. 상·하한은 별도로 존재하나 이 표에는 적재하지
	// 않았다 (아래 CONFIRM 참조).
	healthTotal := applyRate(monthlyWage, rates.HealthInsuranceTotal)
	health := splitContribution(
		healthTotal,
		applyRate(monthlyWage, rates.HealthInsuranceEmployee),
	)

	// 장기요양: 보수월액이 아니라 건강보험료를 과세표준으로 한다
	// (노인장기요양보험법 제9조). 절사된 건강보험료 총액을 기준으로 계산해야
	// 고지서와 어긋나지 않는다. 노사가 건강보험과 같은 1/2씩 부담한다.
	longTermTotal := applyRate(health.Total(), rates.LongTermCareOfHealth)
	longTermEmployee := RoundDownTo(longTermTotal/2, TenWonUnit)
	longTerm := SocialInsuranceContribution{
		Employee: longTermEmployee,
		Employer: longTermTotal - longTermEmployee,
	}

	employment := SocialInsuranceContribution{
		Employee: applyRate(monthlyWage, rates.EmploymentInsuranceEmployee),
		Employer: applyRate(monthlyWage, rates.EmploymentInsuranceEmployer),
	}

	return SocialInsuranceResult{
		MonthlyWage:         monthlyWage,
		PensionBase:         pensionBase,
		NationalPension:     pension,
		HealthInsurance:     health,
		LongTermCare:        longTerm,
		EmploymentInsurance: employment,
		RatesSource:         rates.Source,
	}, nil
}

// CalculateIndustrialAccident computes 산재보험료 from an industry rate.
//
// 산재보험료율은 업종별로 고용노동부장관이 매년 고시하며 (고용보험 및
// 산업재해보상보험의 보험료징수 등에 관한 법률 제14조), 전 업종에 공통으로
// 적용되는 단일 요율은 존재하지 않는다. 따라서 요율 표를 이 파일에 담지 않고
// 호출자가 해당 사업장의 고시 요율을 넘기도록 한다.
//
// industryRate is in hundred-thousandths, the same scale as the other rates
// (1.53% -> 1530). The premium is borne entirely by the employer.
func CalculateIndustrialAccident(monthlyWage, industryRate int64) (SocialInsuranceContribution, error) {
	if monthlyWage < 0 {
		return SocialInsuranceContribution{}, ErrNegativeMonthlyWage
	}
	if industryRate < 0 {
		return SocialInsuranceContribution{}, errors.New("industry rate must not be negative")
	}
	return SocialInsuranceContribution{
		Employee: 0,
		Employer: applyRate(monthlyWage, industryRate),
	}, nil
}

// applyRate multiplies a base by a rate in hundred-thousandths and truncates
// below 10 won.
func applyRate(base, rate int64) int64 {
	return RoundDownTo(base*rate/rateScale, TenWonUnit)
}

// splitContribution builds a contribution from a total and the employee half,
// deriving the employer half as the remainder so the two always sum to total.
func splitContribution(total, employee int64) SocialInsuranceContribution {
	if employee > total {
		employee = total
	}
	return SocialInsuranceContribution{
		Employee: employee,
		Employer: total - employee,
	}
}

// ============================================================================
// CONFIRM / 확인 필요 - 이 파일에 의도적으로 넣지 않은 값들
// ============================================================================
//
//  1. 2026년 요율 일체.
//     건강보험료율·장기요양보험료율은 매년 건강보험정책심의위원회가 결정하고
//     연도마다 달라진다. 2026년 값을 확인하지 못했으므로 표에 넣지 않았다.
//     SocialInsuranceRatesFor(2026)은 ErrSocialInsuranceRatesUnavailable을
//     반환한다. 확정 고시를 확인해 socialInsuranceRateTable에 한 줄 추가할 것.
//
//  2. 2026.7.1.~2027.6.30. 국민연금 기준소득월액 상·하한.
//     매년 3월 보건복지부 고시로 정해진다. 고시 확인 후
//     pensionBasePeriods에 추가할 것. 그 전까지 해당 기간 계산은
//     ErrPensionBaseUnavailable로 실패한다.
//
//  3. 건강보험 보수월액 상·하한.
//     국민건강보험법 시행령에 월별 보험료액의 상한·하한이 있으나 값을
//     확인하지 못해 적용하지 않았다. 고액 급여자의 건강보험료가 상한을
//     넘겨 계산될 수 있다.
//
//  4. 고용안정·직업능력개발사업 보험료율은 표에 담았다
//     (EmploymentStability*, 고용보험법 시행령 제12조). 다만
//     EmploymentInsuranceEmployer에는 **실업급여분만** 들어 있다.
//     CalculateSocialInsurance는 사업장 규모를 알지 못하므로 이 항목을
//     자동으로 더하지 않는다. 사업주 부담 총액을 구하려면
//     EmploymentStabilityRate(size)를 따로 더해야 한다.
//
//  5. 산재보험료율.
//     업종별 고시 요율이므로 표로 담지 않고 CalculateIndustrialAccident의
//     인자로 받는다.
//
//  6. 단수처리.
//     이 파일은 각 보험료를 10원 미만 절사한다. 공단 고지서의 실제
//     단수처리 규칙과 대사해 확정할 것.
