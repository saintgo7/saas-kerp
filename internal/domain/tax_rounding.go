package domain

// Rounding helpers shared by the Korean tax and payroll calculations.
//
// All statutory calculations in this package work in whole won (int64), never
// float64. A tax figure that is off by one won is a filing error, and float64
// cannot represent the intermediate values exactly.
//
// SOURCE / 근거
//   - 국고금관리법 제47조 (국고금의 단수계산): 국고금의 수입·지출에 끝수가
//     있을 때 그 끝수는 계산하지 아니한다. 실무에서 원천징수세액은 10원 미만을
//     절사하고, 부가가치세 세액은 1원 미만을 절사한다.
//
// CONFIRM / 확인 필요
//
//	두 절사 단위(부가세 1원 / 원천징수 10원)는 실무 관행에 따른 것이다.
//	적용 대상 신고 서식별로 국세청 고시를 확인한 뒤 truncationUnit 상수를
//	확정할 것. 반올림이 요구되는 서식이 있다면 RoundDownTo 대신 별도 함수를
//	추가하고 호출부에서 선택하도록 한다.
const (
	// WonUnit truncates below 1 won.
	WonUnit int64 = 1

	// TenWonUnit truncates below 10 won, the unit used for withholding tax.
	TenWonUnit int64 = 10
)

// RoundDownTo truncates an amount down to a multiple of unit.
//
// Truncation is toward zero, so a negative amount (a refund or a correction)
// loses its fraction in the same direction as a positive one and the two never
// disagree by a unit when they are netted.
func RoundDownTo(amount, unit int64) int64 {
	if unit <= 1 {
		return amount
	}
	remainder := amount % unit
	return amount - remainder
}
