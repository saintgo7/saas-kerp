package domain

import (
	"errors"
	"math"
)

// 감가상각 (depreciation).
//
// SOURCE / 근거
//   - 법인세법 시행령 제26조 (감가상각방법): 감가상각은 정액법·정률법 등
//     신고한 방법에 따른다.
//   - 법인세법 시행령 제26조 제6항: 감가상각자산의 잔존가액은 0으로 한다.
//     다만 정률법을 적용하는 경우에는 취득가액의 100분의 5에 상당하는
//     금액을 잔존가액으로 하여 상각률을 계산하되, 미상각잔액이 취득가액의
//     100분의 5 이하가 되는 사업연도에 그 잔액을 모두 상각한다.
//   - 법인세법 시행령 제26조: 상각이 종료된 자산에 대해서는 취득가액의
//     일부(1,000원)를 비망가액으로 남긴다.
//   - 법인세법 시행규칙 별표4 (정률법 상각률): 내용연수별 상각률은
//     잔존가액을 취득가액의 5%로 하여 1 - (0.05)^(1/내용연수)로 산정한
//     값을 소수점 넷째 자리에서 반올림한 것이다. 예: 4년 0.528,
//     5년 0.451, 6년 0.394, 8년 0.313, 10년 0.259.
//
// All amounts are whole won (int64).

// Depreciation errors
var (
	ErrInvalidUsefulLife         = errors.New("useful life must be at least 1 year")
	ErrNegativeAcquisitionCost   = errors.New("acquisition cost must not be negative")
	ErrInvalidSalvageValue       = errors.New("salvage value must be between 0 and the acquisition cost")
	ErrInvalidDepreciationMonths = errors.New("months in service must be between 1 and 12")
)

// MemorandumValue 비망가액. Once an asset is fully depreciated, 1,000 won is
// left on the books so that the asset remains visible in the fixed asset
// register until it is disposed of.
const MemorandumValue int64 = 1000

// DecliningBalanceSalvageRatePercent is the 5% of acquisition cost used ONLY
// to derive the declining balance rate (법인세법 시행령 제26조 제6항 단서).
// It is not a salvage value that survives on the books.
const DecliningBalanceSalvageRatePercent = 5

// DepreciationMethod identifies a depreciation method.
type DepreciationMethod string

const (
	// DepreciationStraightLine 정액법.
	DepreciationStraightLine DepreciationMethod = "straight_line"
	// DepreciationDecliningBalance 정률법.
	DepreciationDecliningBalance DepreciationMethod = "declining_balance"
)

// DepreciationPeriod is one year of a depreciation schedule.
type DepreciationPeriod struct {
	// Year is the ordinal year of the schedule, starting at 1.
	Year int `json:"year"`
	// Months is the number of months depreciated in this year. It is 12
	// except in the first year of an asset acquired mid-year.
	Months int `json:"months"`
	// Expense 당기 감가상각비.
	Expense int64 `json:"expense"`
	// AccumulatedDepreciation 감가상각누계액.
	AccumulatedDepreciation int64 `json:"accumulated_depreciation"`
	// BookValue 기말 장부가액.
	BookValue int64 `json:"book_value"`
}

// StraightLineAnnualExpense returns the full-year 정액법 depreciation charge.
//
//	감가상각비 = (취득가액 - 잔존가액) / 내용연수
//
// 법인세법상 잔존가액은 0이므로 salvageValue에는 통상 0을 넘긴다. 회계목적의
// 잔존가액이 있는 경우를 위해 인자로 남겨 두었다.
func StraightLineAnnualExpense(acquisitionCost, salvageValue int64, usefulLifeYears int) (int64, error) {
	if acquisitionCost < 0 {
		return 0, ErrNegativeAcquisitionCost
	}
	if usefulLifeYears < 1 {
		return 0, ErrInvalidUsefulLife
	}
	if salvageValue < 0 || salvageValue > acquisitionCost {
		return 0, ErrInvalidSalvageValue
	}
	return (acquisitionCost - salvageValue) / int64(usefulLifeYears), nil
}

// DecliningBalanceRateThousandths returns the 정률법 상각률 for a useful life,
// expressed in thousandths (0.451 -> 451).
//
//	상각률 = 1 - (잔존가액비율)^(1/내용연수), 잔존가액비율 = 0.05
//
// The exact value is rounded UP at the fourth decimal place. That is what
// reproduces the table in 법인세법 시행규칙 별표4: the exact rates are
// 0.527129 (4년), 0.450719 (5년), 0.393038 (6년), 0.312344 (8년) and
// 0.258866 (10년), while the published rates are 0.528, 0.451, 0.394, 0.313
// and 0.259 - rounding to nearest would give 0.527, 0.451, 0.393, 0.312 and
// 0.259, which disagrees on three of the five.
//
// CONFIRM / 확인 필요: 별표4 원문과 전 내용연수 구간을 대조할 것. 위 다섯 개
// 값에서만 올림 규칙이 확인되었다.
//
// The rate is kept as an integer so the schedule below is computed entirely in
// integers. Multiplying a book value by a float64 rate makes the yearly
// expense depend on binary rounding (5,490,000 x 0.451 can land on
// 2,475,989.999...), which is not acceptable for a figure that goes on a tax
// return.
func DecliningBalanceRateThousandths(usefulLifeYears int) (int64, error) {
	if usefulLifeYears < 1 {
		return 0, ErrInvalidUsefulLife
	}
	ratio := float64(DecliningBalanceSalvageRatePercent) / 100.0
	exact := 1 - math.Pow(ratio, 1/float64(usefulLifeYears))
	return int64(math.Ceil(exact * 1000)), nil
}

// DecliningBalanceRate returns the same rate as a float64, for display.
func DecliningBalanceRate(usefulLifeYears int) (float64, error) {
	thousandths, err := DecliningBalanceRateThousandths(usefulLifeYears)
	if err != nil {
		return 0, err
	}
	return float64(thousandths) / 1000, nil
}

// MonthsInFirstYear returns the number of months to depreciate in the year an
// asset is acquired.
//
// 법인세법 시행령 제26조 제7항: 사업연도 중에 취득한 자산의 감가상각비는
// 월할계산하며, 1개월 미만의 일수는 1개월로 한다. 취득한 달을 포함하므로
// 3월 취득 자산의 첫 사업연도 상각월수는 10개월이다.
func MonthsInFirstYear(acquisitionMonth int) (int, error) {
	if acquisitionMonth < 1 || acquisitionMonth > 12 {
		return 0, ErrInvalidDepreciationMonths
	}
	return 13 - acquisitionMonth, nil
}

// StraightLineSchedule builds the full 정액법 schedule.
//
// monthsInFirstYear allows mid-year acquisition (see MonthsInFirstYear); pass
// 12 for an asset held for the whole first year. The schedule runs until the
// book value reaches the salvage value, or MemorandumValue when the salvage
// value is zero, and the final year absorbs whatever rounding is left over so
// the accumulated depreciation is exact.
func StraightLineSchedule(acquisitionCost, salvageValue int64, usefulLifeYears, monthsInFirstYear int) ([]DepreciationPeriod, error) {
	annual, err := StraightLineAnnualExpense(acquisitionCost, salvageValue, usefulLifeYears)
	if err != nil {
		return nil, err
	}
	if monthsInFirstYear < 1 || monthsInFirstYear > 12 {
		return nil, ErrInvalidDepreciationMonths
	}

	floor := salvageValue
	if floor == 0 {
		floor = MemorandumValue
	}
	if acquisitionCost <= floor {
		return nil, nil
	}

	var (
		schedule    []DepreciationPeriod
		accumulated int64
		bookValue   = acquisitionCost
	)

	// A mid-year acquisition spills the unused months into an extra year, so
	// the schedule can be one year longer than the useful life.
	maxYears := usefulLifeYears
	if monthsInFirstYear < 12 {
		maxYears++
	}

	for year := 1; year <= maxYears; year++ {
		months := 12
		if year == 1 {
			months = monthsInFirstYear
		}

		expense := annual * int64(months) / 12

		// Never depreciate below the floor.
		if bookValue-expense < floor {
			expense = bookValue - floor
		}
		if expense <= 0 {
			break
		}

		accumulated += expense
		bookValue -= expense
		schedule = append(schedule, DepreciationPeriod{
			Year:                    year,
			Months:                  months,
			Expense:                 expense,
			AccumulatedDepreciation: accumulated,
			BookValue:               bookValue,
		})

		if bookValue <= floor {
			break
		}
	}

	return schedule, nil
}

// DecliningBalanceSchedule builds the full 정률법 schedule.
//
//	당기 상각비 = 미상각잔액 x 상각률
//
// 법인세법 시행령 제26조 제6항 단서에 따라 미상각잔액이 취득가액의 5% 이하가
// 되는 사업연도에 잔액을 모두 상각하되, 비망가액 1,000원은 남긴다.
func DecliningBalanceSchedule(acquisitionCost int64, usefulLifeYears, monthsInFirstYear int) ([]DepreciationPeriod, error) {
	if acquisitionCost < 0 {
		return nil, ErrNegativeAcquisitionCost
	}
	rateThousandths, err := DecliningBalanceRateThousandths(usefulLifeYears)
	if err != nil {
		return nil, err
	}
	if monthsInFirstYear < 1 || monthsInFirstYear > 12 {
		return nil, ErrInvalidDepreciationMonths
	}
	if acquisitionCost <= MemorandumValue {
		return nil, nil
	}

	// 미상각잔액이 이 금액 이하가 되면 그 사업연도에 전액 상각한다.
	writeOffThreshold := acquisitionCost * DecliningBalanceSalvageRatePercent / 100

	var (
		schedule    []DepreciationPeriod
		accumulated int64
		bookValue   = acquisitionCost
	)

	maxYears := usefulLifeYears + 1
	if monthsInFirstYear < 12 {
		maxYears++
	}

	for year := 1; year <= maxYears; year++ {
		months := 12
		if year == 1 {
			months = monthsInFirstYear
		}

		var expense int64
		if bookValue <= writeOffThreshold {
			// Final write-off, less the memorandum value.
			expense = bookValue - MemorandumValue
		} else {
			expense = bookValue * rateThousandths * int64(months) / (1000 * 12)
			if bookValue-expense < MemorandumValue {
				expense = bookValue - MemorandumValue
			}
		}

		if expense <= 0 {
			break
		}

		accumulated += expense
		bookValue -= expense
		schedule = append(schedule, DepreciationPeriod{
			Year:                    year,
			Months:                  months,
			Expense:                 expense,
			AccumulatedDepreciation: accumulated,
			BookValue:               bookValue,
		})

		if bookValue <= MemorandumValue {
			break
		}
	}

	return schedule, nil
}
