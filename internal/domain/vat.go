package domain

import "errors"

// 부가가치세 (Korean Value Added Tax).
//
// SOURCE / 근거
//   - 부가가치세법 제30조 (세율): 부가가치세의 세율은 100분의 10으로 한다.
//   - 부가가치세법 제21조~제24조 (영세율): 재화의 수출 등에 대하여는 0%를 적용한다.
//   - 부가가치세법 제26조 (재화 또는 용역의 공급에 대한 면세): 면세 대상은
//     부가가치세를 과세하지 아니하며 세금계산서가 아닌 계산서를 발급한다.
//
// Every amount here is whole won (int64). The database columns for tax invoice
// amounts are BIGINT and the NTS interface carries whole won, so there is no
// sub-won value to represent at any point.

// VAT errors
var (
	ErrInvalidVATCategory   = errors.New("invalid vat category")
	ErrNegativeSupplyAmount = errors.New("supply amount must not be negative")
	ErrVATAmountMismatch    = errors.New("tax amount does not match supply amount at the applicable rate")
)

// VATCategory classifies a supply for VAT purposes.
type VATCategory string

const (
	// VATCategoryTaxable 과세: the standard 10% rate applies.
	VATCategoryTaxable VATCategory = "taxable"
	// VATCategoryZeroRated 영세율: the supply is taxable but at 0%
	// (exports and similar). A tax invoice is still issued, with tax 0.
	VATCategoryZeroRated VATCategory = "zero_rated"
	// VATCategoryExempt 면세: the supply is outside VAT altogether.
	// No VAT arises and no input tax may be deducted.
	VATCategoryExempt VATCategory = "exempt"
)

// StandardVATRatePercent is the statutory rate, in percent.
// 부가가치세법 제30조.
const StandardVATRatePercent int64 = 10

// IsValid reports whether the category is one of the three defined values.
func (c VATCategory) IsValid() bool {
	switch c {
	case VATCategoryTaxable, VATCategoryZeroRated, VATCategoryExempt:
		return true
	}
	return false
}

// RatePercent returns the VAT rate applicable to the category, in percent.
// Zero-rated and exempt supplies both carry 0; they differ in input tax
// treatment, not in the output rate.
func (c VATCategory) RatePercent() int64 {
	if c == VATCategoryTaxable {
		return StandardVATRatePercent
	}
	return 0
}

// CalculateVAT returns the VAT on a supply amount.
//
// The result is truncated below 1 won: 공급가액 1,234,567 -> 세액 123,456
// (123,456.7 exact). See tax_rounding.go for the basis and the open question
// about the truncation unit.
func CalculateVAT(supplyAmount int64, category VATCategory) (int64, error) {
	if !category.IsValid() {
		return 0, ErrInvalidVATCategory
	}
	if supplyAmount < 0 {
		return 0, ErrNegativeSupplyAmount
	}
	rate := category.RatePercent()
	if rate == 0 {
		return 0, nil
	}
	return RoundDownTo(supplyAmount*rate/100, WonUnit), nil
}

// SplitTaxInclusive splits a VAT-inclusive amount (공급대가) into the supply
// amount and the tax.
//
// 공급가액 = 공급대가 x 100/110, truncated below 1 won; 세액 = 공급대가 -
// 공급가액. Deriving the supply amount first and taking the tax as the
// remainder is what keeps supply + tax exactly equal to the amount actually
// charged; computing both independently can leave a one-won gap.
//
// CONFIRM / 확인 필요: 공급대가 역산 시 공급가액을 먼저 절사하는 방식과
// 세액(= 공급대가/11)을 먼저 절사하는 방식이 실무에서 모두 쓰인다. 두 결과는
// 1원 차이가 날 수 있으므로 국세청 서식 기준을 확인할 것.
func SplitTaxInclusive(taxInclusiveAmount int64, category VATCategory) (supplyAmount, taxAmount int64, err error) {
	if !category.IsValid() {
		return 0, 0, ErrInvalidVATCategory
	}
	if taxInclusiveAmount < 0 {
		return 0, 0, ErrNegativeSupplyAmount
	}
	rate := category.RatePercent()
	if rate == 0 {
		return taxInclusiveAmount, 0, nil
	}
	supplyAmount = RoundDownTo(taxInclusiveAmount*100/(100+rate), WonUnit)
	return supplyAmount, taxInclusiveAmount - supplyAmount, nil
}

// ValidateVATAmount checks a tax amount supplied by a client against the
// amount the statutory rate produces.
//
// A tolerance of one won is allowed because the counterparty's system may
// round rather than truncate the fraction; anything larger is a real
// discrepancy that would be filed with the NTS as understated or overstated
// output tax.
func ValidateVATAmount(supplyAmount, taxAmount int64, category VATCategory) error {
	expected, err := CalculateVAT(supplyAmount, category)
	if err != nil {
		return err
	}
	diff := taxAmount - expected
	if diff < 0 {
		diff = -diff
	}
	if diff > 1 {
		return ErrVATAmountMismatch
	}
	return nil
}
