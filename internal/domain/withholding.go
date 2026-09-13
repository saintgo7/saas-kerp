package domain

import "errors"

// 원천징수 (Korean withholding tax).
//
// Every amount is whole won (int64).
//
// SOURCE / 근거
//   - 소득세법 제129조 제1항 제3호 (원천징수세율): 사업소득에 대해서는
//     100분의 3.
//   - 소득세법 제129조 제1항 제6호: 기타소득에 대해서는 기타소득금액의
//     100분의 20.
//   - 지방세법 제103조의13 (특별징수): 원천징수하는 소득세에 대한
//     개인지방소득세는 소득세액의 100분의 10.
//   - 소득세법 시행령 제87조 (기타소득의 필요경비계산): 강연료·원고료 등
//     열거된 기타소득은 수입금액의 100분의 60을 필요경비로 한다
//     (2019.1.1. 이후 지급분. 그 전에는 100분의 70).
//   - 소득세법 제84조 (기타소득의 과세최저한): 기타소득금액이 건별로 5만원
//     이하이면 소득세를 과세하지 아니한다.
//   - 국고금관리법 제47조: 세액의 10원 미만 끝수는 계산하지 아니한다.

// Withholding errors
var (
	ErrNegativePaymentAmount = errors.New("payment amount must not be negative")

	// ErrSimplifiedTaxTableUnavailable is returned by the earned income
	// withholding entry point. See CalculateEarnedIncomeWithholding.
	ErrSimplifiedTaxTableUnavailable = errors.New("근로소득 간이세액표가 적재되지 않았습니다")
)

// Statutory withholding rates, expressed in basis points (1/10000) so that
// 3% is 300 and no float64 appears anywhere in the calculation.
const (
	// BusinessIncomeWithholdingBps 사업소득 원천징수세율 3%.
	// 소득세법 제129조 제1항 제3호.
	BusinessIncomeWithholdingBps int64 = 300

	// OtherIncomeWithholdingBps 기타소득 원천징수세율 20%.
	// 소득세법 제129조 제1항 제6호.
	OtherIncomeWithholdingBps int64 = 2000

	// LocalIncomeTaxBps 개인지방소득세: 소득세액의 10%.
	// 지방세법 제103조의13.
	LocalIncomeTaxBps int64 = 1000

	// OtherIncomeStandardExpenseRateBps 기타소득 필요경비율 60%
	// (소득세법 시행령 제87조, 2019.1.1. 이후 지급분).
	OtherIncomeStandardExpenseRateBps int64 = 6000

	// OtherIncomeTaxFreeThreshold 기타소득 과세최저한: 건별 기타소득금액
	// 5만원 이하 (소득세법 제84조).
	OtherIncomeTaxFreeThreshold int64 = 50000
)

// WithholdingResult is the outcome of a withholding calculation.
type WithholdingResult struct {
	// PaymentAmount 지급금액 (gross).
	PaymentAmount int64 `json:"payment_amount"`
	// TaxableAmount 과세대상 소득금액. For business income this equals the
	// payment; for other income it is the payment less deemed expenses.
	TaxableAmount int64 `json:"taxable_amount"`
	// IncomeTax 소득세 (national).
	IncomeTax int64 `json:"income_tax"`
	// LocalIncomeTax 지방소득세.
	LocalIncomeTax int64 `json:"local_income_tax"`
	// TotalWithheld 원천징수 합계.
	TotalWithheld int64 `json:"total_withheld"`
	// NetPayment 실지급액 = PaymentAmount - TotalWithheld.
	NetPayment int64 `json:"net_payment"`
}

// CalculateBusinessIncomeWithholding computes withholding on 사업소득
// (freelance / professional service fees), the familiar 3.3%.
//
// 소득세 = 지급액 x 3%, 10원 미만 절사.
// 지방소득세 = 소득세 x 10%, 10원 미만 절사.
//
// The local tax is derived from the already-truncated income tax, not from the
// gross payment: 지방세법 제103조의13 defines it as a percentage of the income
// tax that was withheld.
func CalculateBusinessIncomeWithholding(paymentAmount int64) (WithholdingResult, error) {
	if paymentAmount < 0 {
		return WithholdingResult{}, ErrNegativePaymentAmount
	}

	incomeTax := RoundDownTo(paymentAmount*BusinessIncomeWithholdingBps/10000, TenWonUnit)
	localTax := RoundDownTo(incomeTax*LocalIncomeTaxBps/10000, TenWonUnit)
	total := incomeTax + localTax

	return WithholdingResult{
		PaymentAmount:  paymentAmount,
		TaxableAmount:  paymentAmount,
		IncomeTax:      incomeTax,
		LocalIncomeTax: localTax,
		TotalWithheld:  total,
		NetPayment:     paymentAmount - total,
	}, nil
}

// CalculateOtherIncomeWithholding computes withholding on 기타소득 where the
// statutory deemed expense rate applies (강연료, 원고료 and the other items
// listed in 소득세법 시행령 제87조).
//
// 기타소득금액 = 지급액 - 필요경비(지급액 x 60%).
// 기타소득금액이 건별 5만원 이하이면 과세최저한에 해당하여 과세하지 않는다
// (소득세법 제84조).
// 소득세 = 기타소득금액 x 20%, 10원 미만 절사. 지방소득세 = 소득세 x 10%.
//
// expenseRateBps lets the caller override the deemed expense rate for the
// categories that use a different one (or 0 for income with no deemed
// expenses, such as 상금·사례금). Pass OtherIncomeStandardExpenseRateBps for
// the common case.
func CalculateOtherIncomeWithholding(paymentAmount, expenseRateBps int64) (WithholdingResult, error) {
	if paymentAmount < 0 {
		return WithholdingResult{}, ErrNegativePaymentAmount
	}
	if expenseRateBps < 0 || expenseRateBps > 10000 {
		return WithholdingResult{}, errors.New("expense rate must be between 0 and 10000 bps")
	}

	deemedExpense := paymentAmount * expenseRateBps / 10000
	taxable := paymentAmount - deemedExpense

	result := WithholdingResult{
		PaymentAmount: paymentAmount,
		TaxableAmount: taxable,
		NetPayment:    paymentAmount,
	}

	// 과세최저한: 건별 기타소득금액 5만원 이하는 과세하지 않는다.
	if taxable <= OtherIncomeTaxFreeThreshold {
		return result, nil
	}

	incomeTax := RoundDownTo(taxable*OtherIncomeWithholdingBps/10000, TenWonUnit)
	localTax := RoundDownTo(incomeTax*LocalIncomeTaxBps/10000, TenWonUnit)

	result.IncomeTax = incomeTax
	result.LocalIncomeTax = localTax
	result.TotalWithheld = incomeTax + localTax
	result.NetPayment = paymentAmount - result.TotalWithheld
	return result, nil
}

// ============================================================================
// 근로소득 (earned income)
// ============================================================================

// EarnedIncomeDeduction 근로소득공제 (소득세법 제47조).
//
// 총급여액           공제액
// 500만원 이하        총급여액 x 70%
// 500만~1,500만원     350만원 + 500만원 초과분 x 40%
// 1,500만~4,500만원   750만원 + 1,500만원 초과분 x 15%
// 4,500만~1억원       1,200만원 + 4,500만원 초과분 x 5%
// 1억원 초과          1,475만원 + 1억원 초과분 x 2%
//
// 공제한도 2,000만원 (소득세법 제47조 제1항 단서, 2020년 귀속분부터).
func EarnedIncomeDeduction(grossSalary int64) int64 {
	if grossSalary <= 0 {
		return 0
	}

	const (
		b1 = 5_000_000
		b2 = 15_000_000
		b3 = 45_000_000
		b4 = 100_000_000

		deductionCap = 20_000_000
	)

	var deduction int64
	switch {
	case grossSalary <= b1:
		deduction = grossSalary * 70 / 100
	case grossSalary <= b2:
		deduction = 3_500_000 + (grossSalary-b1)*40/100
	case grossSalary <= b3:
		deduction = 7_500_000 + (grossSalary-b2)*15/100
	case grossSalary <= b4:
		deduction = 12_000_000 + (grossSalary-b3)*5/100
	default:
		deduction = 14_750_000 + (grossSalary-b4)*2/100
	}

	if deduction > deductionCap {
		deduction = deductionCap
	}
	return deduction
}

// basicRateBracket is one row of the 기본세율 table.
type basicRateBracket struct {
	upperBound  int64 // exclusive upper bound of the bracket; 0 means unbounded
	baseTax     int64 // 누진공제 방식이 아닌 "이 구간 시작까지의 누적세액"
	lowerBound  int64
	ratePercent int64
}

// basicRateBrackets 종합소득 과세표준 기본세율 (소득세법 제55조 제1항).
// 2023년 귀속분부터 적용되는 구간이다.
//
//	1,400만원 이하                6%
//	1,400만 ~ 5,000만원           84만원 + 1,400만원 초과분 15%
//	5,000만 ~ 8,800만원          624만원 + 5,000만원 초과분 24%
//	8,800만 ~ 1억5,000만원     1,536만원 + 8,800만원 초과분 35%
//	1억5,000만 ~ 3억원         3,706만원 + 1억5,000만원 초과분 38%
//	3억 ~ 5억원                9,406만원 + 3억원 초과분 40%
//	5억 ~ 10억원              1억7,406만원 + 5억원 초과분 42%
//	10억원 초과               3억8,406만원 + 10억원 초과분 45%
var basicRateBrackets = []basicRateBracket{
	{lowerBound: 0, upperBound: 14_000_000, baseTax: 0, ratePercent: 6},
	{lowerBound: 14_000_000, upperBound: 50_000_000, baseTax: 840_000, ratePercent: 15},
	{lowerBound: 50_000_000, upperBound: 88_000_000, baseTax: 6_240_000, ratePercent: 24},
	{lowerBound: 88_000_000, upperBound: 150_000_000, baseTax: 15_360_000, ratePercent: 35},
	{lowerBound: 150_000_000, upperBound: 300_000_000, baseTax: 37_060_000, ratePercent: 38},
	{lowerBound: 300_000_000, upperBound: 500_000_000, baseTax: 94_060_000, ratePercent: 40},
	{lowerBound: 500_000_000, upperBound: 1_000_000_000, baseTax: 174_060_000, ratePercent: 42},
	{lowerBound: 1_000_000_000, upperBound: 0, baseTax: 384_060_000, ratePercent: 45},
}

// IncomeTaxByBasicRate applies the 기본세율 (소득세법 제55조) to a tax base
// (과세표준) and returns the computed tax (산출세액) in whole won.
func IncomeTaxByBasicRate(taxBase int64) int64 {
	if taxBase <= 0 {
		return 0
	}
	for _, b := range basicRateBrackets {
		if b.upperBound == 0 || taxBase <= b.upperBound {
			return b.baseTax + (taxBase-b.lowerBound)*b.ratePercent/100
		}
	}
	return 0
}

// CalculateEarnedIncomeWithholding is the entry point for monthly earned
// income withholding (근로소득 간이세액).
//
// NOT IMPLEMENTED, deliberately.
//
// The monthly amount is not a rate applied to salary: it comes from the
// 근로소득 간이세액표 (소득세법 시행령 별표2), whose derivation folds in the
// 근로소득공제, 인적공제, 연금보험료공제, a bracketed 특별소득공제 estimate and
// the 근로소득세액공제. Two of those parameter sets are not reproduced here,
// so any number this function invented would be wrong, and a wrong withholding
// amount is worse than none - it is filed with the NTS every month and shows
// up as an under- or over-payment at 연말정산.
//
// TODO / 확인 필요: 국세청이 고시하는 간이세액표 원본(또는 그 산출 산식)을
// 적재한 뒤 구현할 것. EarnedIncomeDeduction and IncomeTaxByBasicRate above are
// the two pieces that are certain and are already available for 연말정산 style
// calculations.
func CalculateEarnedIncomeWithholding(monthlySalary int64, dependents int) (WithholdingResult, error) {
	return WithholdingResult{}, ErrSimplifiedTaxTableUnavailable
}
