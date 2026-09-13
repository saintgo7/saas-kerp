package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// 급여 (Korean payroll).
//
// AMOUNTS ARE int64 WHOLE WON. The payroll and insurance columns are
// DECIMAL(18,0) (db/migrations/000007, and 000018 deliberately left them at
// scale 0: "Korean payroll settles in whole won"). float64 has no business
// here - a payroll figure that is off by one won is a filing error, and the
// statutory helpers this file builds on (social_insurance.go, withholding.go,
// tax_rounding.go) are all int64 for the same reason.
//
// NOTHING IS CALCULATED IN THIS FILE THAT ALREADY EXISTS ELSEWHERE.
// 4대보험 comes from CalculateSocialInsurance / CalculateIndustrialAccident,
// 지방소득세 from LocalIncomeTaxBps, truncation from RoundDownTo. A year whose
// rates have not been verified fails loudly rather than falling back - see the
// CONFIRM block at the bottom of social_insurance.go.

// Payroll errors.
var (
	ErrPayrollNotFound          = errors.New("payroll not found")
	ErrPayrollPeriodNotFound    = errors.New("payroll period not found")
	ErrPayrollPeriodExists      = errors.New("payroll period already exists for this month")
	ErrPayrollExists            = errors.New("payroll already exists for this employee and month")
	ErrPayrollNotEditable       = errors.New("payroll cannot be modified in its current status")
	ErrPayrollPeriodNotEditable = errors.New("payroll period cannot be modified in its current status")
	ErrPayrollNotCalculated     = errors.New("payroll must be calculated before it is approved")
	ErrPayrollTotalsMismatch    = errors.New("payroll header totals do not match the sum of its items")
	ErrPayrollMonthInvalid      = errors.New("pay month must be between 1 and 12")
	ErrPayrollPeriodDates       = errors.New("period_start must not be after period_end")
	ErrNegativeAmount           = errors.New("amount must not be negative")
	ErrNonTaxableExceedsGross   = errors.New("non-taxable amount exceeds total earnings")
	ErrPayrollEmptyItemName     = errors.New("every payroll item needs a code and a name")
)

// PayrollPeriodStatus is the lifecycle of one monthly payroll run.
//
// draft -> calculated -> approved -> paid -> closed. "processing" exists in the
// CHECK constraint for a long-running batch; nothing sets it yet.
type PayrollPeriodStatus string

const (
	PayrollPeriodStatusDraft      PayrollPeriodStatus = "draft"
	PayrollPeriodStatusProcessing PayrollPeriodStatus = "processing"
	PayrollPeriodStatusCalculated PayrollPeriodStatus = "calculated"
	PayrollPeriodStatusApproved   PayrollPeriodStatus = "approved"
	PayrollPeriodStatusPaid       PayrollPeriodStatus = "paid"
	PayrollPeriodStatusClosed     PayrollPeriodStatus = "closed"
)

// PayrollStatus is the lifecycle of one employee's payroll record.
type PayrollStatus string

const (
	PayrollStatusDraft      PayrollStatus = "draft"
	PayrollStatusCalculated PayrollStatus = "calculated"
	PayrollStatusApproved   PayrollStatus = "approved"
	PayrollStatusPaid       PayrollStatus = "paid"
	PayrollStatusCancelled  PayrollStatus = "cancelled"
)

// PayrollItemType distinguishes an earning line from a deduction line.
type PayrollItemType string

const (
	PayrollItemEarning   PayrollItemType = "earning"
	PayrollItemDeduction PayrollItemType = "deduction"
)

// Item codes written by CalculatePayroll. They are stable identifiers, not
// display labels: the UI reads item_name, reconciliation reads item_code.
const (
	PayrollItemCodeBaseSalary     = "base_salary"
	PayrollItemCodeOvertimePay    = "overtime_pay"
	PayrollItemCodeNightPay       = "night_pay"
	PayrollItemCodeHolidayPay     = "holiday_pay"
	PayrollItemCodeBonus          = "bonus"
	PayrollItemCodeNPS            = "nps_employee"
	PayrollItemCodeNHIS           = "nhis_employee"
	PayrollItemCodeNHISLTC        = "nhis_ltc_employee"
	PayrollItemCodeEI             = "ei_employee"
	PayrollItemCodeIncomeTax      = "income_tax"
	PayrollItemCodeLocalIncomeTax = "local_income_tax"
)

// ---------------------------------------------------------------------------
// Persistence models
// ---------------------------------------------------------------------------

// PayrollAmountMap is the Go mapping for the JSONB amount columns on payrolls
// (allowances, other_earnings, other_deductions). Values are whole won.
type PayrollAmountMap map[string]int64

// Value implements driver.Valuer. A nil map is stored as {} rather than NULL so
// the column always holds a JSON object.
func (m PayrollAmountMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

// Scan implements sql.Scanner.
func (m *PayrollAmountMap) Scan(value interface{}) error {
	if value == nil {
		*m = PayrollAmountMap{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("payroll amount map: unsupported source type %T", value)
	}
	if len(data) == 0 {
		*m = PayrollAmountMap{}
		return nil
	}
	return json.Unmarshal(data, m)
}

// GormDataType tells GORM the column type.
func (PayrollAmountMap) GormDataType() string { return "jsonb" }

// PayrollPeriod is one monthly payroll run for a company.
type PayrollPeriod struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()" json:"id"`
	CompanyID uuid.UUID `gorm:"type:uuid;not null;index" json:"company_id"`

	PayYear    int    `gorm:"not null" json:"pay_year"`
	PayMonth   int    `gorm:"not null" json:"pay_month"`
	PeriodName string `gorm:"type:varchar(50)" json:"period_name,omitempty"`

	PeriodStart time.Time  `gorm:"type:date;not null" json:"period_start"`
	PeriodEnd   time.Time  `gorm:"type:date;not null" json:"period_end"`
	PaymentDate *time.Time `gorm:"type:date" json:"payment_date,omitempty"`

	Status PayrollPeriodStatus `gorm:"type:varchar(20);not null;default:draft" json:"status"`

	CalculatedAt *time.Time `json:"calculated_at,omitempty"`
	CalculatedBy *uuid.UUID `gorm:"type:uuid" json:"calculated_by,omitempty"`
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	ApprovedBy   *uuid.UUID `gorm:"type:uuid" json:"approved_by,omitempty"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`

	TotalEmployees  int   `gorm:"default:0" json:"total_employees"`
	TotalEarnings   int64 `gorm:"type:decimal(18,0);default:0" json:"total_earnings"`
	TotalDeductions int64 `gorm:"type:decimal(18,0);default:0" json:"total_deductions"`
	TotalNetPay     int64 `gorm:"type:decimal(18,0);default:0" json:"total_net_pay"`

	VoucherID *uuid.UUID `gorm:"type:uuid" json:"voucher_id,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName specifies the table name for GORM.
func (PayrollPeriod) TableName() string { return "payroll_periods" }

// Payroll is one employee's payroll record for one month.
//
// payrolls.account_number_enc is deliberately NOT mapped. It is the employee's
// encrypted bank account; a field that does not exist on the struct cannot be
// selected by GORM, cannot be marshalled into a response and cannot be logged
// by accident. The bank transfer file generator is the only thing that needs
// it, and it does not exist yet.
type Payroll struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()" json:"id"`
	CompanyID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"company_id"`
	EmployeeID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"employee_id"`
	PayrollPeriodID *uuid.UUID `gorm:"type:uuid" json:"payroll_period_id,omitempty"`

	PayYear  int `gorm:"not null" json:"pay_year"`
	PayMonth int `gorm:"not null" json:"pay_month"`

	Status PayrollStatus `gorm:"type:varchar(20);not null;default:draft" json:"status"`

	// Earnings
	BaseSalary    int64            `gorm:"type:decimal(18,0);not null;default:0" json:"base_salary"`
	OvertimePay   int64            `gorm:"type:decimal(18,0);not null;default:0" json:"overtime_pay"`
	NightPay      int64            `gorm:"type:decimal(18,0);not null;default:0" json:"night_pay"`
	HolidayPay    int64            `gorm:"type:decimal(18,0);not null;default:0" json:"holiday_pay"`
	Bonus         int64            `gorm:"type:decimal(18,0);not null;default:0" json:"bonus"`
	Allowances    PayrollAmountMap `gorm:"type:jsonb" json:"allowances,omitempty"`
	OtherEarnings PayrollAmountMap `gorm:"type:jsonb" json:"other_earnings,omitempty"`
	TotalEarnings int64            `gorm:"type:decimal(18,0);not null;default:0" json:"total_earnings"`

	// Deductions - taxes
	IncomeTax      int64 `gorm:"type:decimal(18,0);not null;default:0" json:"income_tax"`
	LocalIncomeTax int64 `gorm:"type:decimal(18,0);not null;default:0" json:"local_income_tax"`

	// Deductions - 4대보험 employee portion
	NPSEmployee     int64 `gorm:"column:nps_employee;type:decimal(18,0);not null;default:0" json:"nps_employee"`
	NHISEmployee    int64 `gorm:"column:nhis_employee;type:decimal(18,0);not null;default:0" json:"nhis_employee"`
	NHISLTCEmployee int64 `gorm:"column:nhis_ltc_employee;type:decimal(18,0);not null;default:0" json:"nhis_ltc_employee"`
	EIEmployee      int64 `gorm:"column:ei_employee;type:decimal(18,0);not null;default:0" json:"ei_employee"`

	OtherDeductions PayrollAmountMap `gorm:"type:jsonb" json:"other_deductions,omitempty"`
	TotalDeductions int64            `gorm:"type:decimal(18,0);not null;default:0" json:"total_deductions"`

	NetPay int64 `gorm:"type:decimal(18,0);not null;default:0" json:"net_pay"`

	// Employer cost (reference only; not withheld from the employee)
	NPSEmployer       int64 `gorm:"column:nps_employer;type:decimal(18,0);not null;default:0" json:"nps_employer"`
	NHISEmployer      int64 `gorm:"column:nhis_employer;type:decimal(18,0);not null;default:0" json:"nhis_employer"`
	NHISLTCEmployer   int64 `gorm:"column:nhis_ltc_employer;type:decimal(18,0);not null;default:0" json:"nhis_ltc_employer"`
	EIEmployer        int64 `gorm:"column:ei_employer;type:decimal(18,0);not null;default:0" json:"ei_employer"`
	WCIEmployer       int64 `gorm:"column:wci_employer;type:decimal(18,0);not null;default:0" json:"wci_employer"`
	TotalEmployerCost int64 `gorm:"type:decimal(18,0);not null;default:0" json:"total_employer_cost"`

	// Payment
	PaymentDate *time.Time `gorm:"type:date" json:"payment_date,omitempty"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	BankCode    string     `gorm:"type:varchar(10)" json:"bank_code,omitempty"`

	// Work summary
	WorkDays      int     `gorm:"default:0" json:"work_days"`
	OvertimeHours float64 `gorm:"type:decimal(5,1);default:0" json:"overtime_hours"`
	NightHours    float64 `gorm:"type:decimal(5,1);default:0" json:"night_hours"`
	HolidayHours  float64 `gorm:"type:decimal(5,1);default:0" json:"holiday_hours"`

	Notes string `gorm:"type:varchar(500)" json:"notes,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Items is populated by the repository on a detail read. It is not a GORM
	// association: the join would pull items into every list query.
	Items []PayrollItem `gorm:"-" json:"items,omitempty"`
}

// TableName specifies the table name for GORM.
func (Payroll) TableName() string { return "payrolls" }

// PayrollItem is one earning or deduction line of a payroll record.
type PayrollItem struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()" json:"id"`
	PayrollID uuid.UUID `gorm:"type:uuid;not null;index" json:"payroll_id"`
	CompanyID uuid.UUID `gorm:"type:uuid;not null;index" json:"company_id"`

	ItemType PayrollItemType `gorm:"type:varchar(20);not null" json:"item_type"`
	ItemCode string          `gorm:"type:varchar(50);not null" json:"item_code"`
	ItemName string          `gorm:"type:varchar(100);not null" json:"item_name"`

	Amount int64 `gorm:"type:decimal(18,0);not null" json:"amount"`

	IsTaxable bool `gorm:"default:true" json:"is_taxable"`
	IsFixed   bool `gorm:"default:false" json:"is_fixed"`

	SortOrder int `gorm:"default:0" json:"sort_order"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName specifies the table name for GORM.
func (PayrollItem) TableName() string { return "payroll_items" }

// ---------------------------------------------------------------------------
// State transitions
// ---------------------------------------------------------------------------

// CanBeModified reports whether the payroll may still be edited or recalculated.
func (p *Payroll) CanBeModified() bool {
	return p.Status == PayrollStatusDraft || p.Status == PayrollStatusCalculated
}

// CanBeApproved reports whether the payroll is ready for approval. A draft that
// was never calculated has zero deductions, so approving it would confirm a
// payroll on which nothing was withheld.
func (p *Payroll) CanBeApproved() bool {
	return p.Status == PayrollStatusCalculated
}

// CanCalculate reports whether the period may still be (re)calculated.
func (pp *PayrollPeriod) CanCalculate() bool {
	switch pp.Status {
	case PayrollPeriodStatusDraft, PayrollPeriodStatusProcessing, PayrollPeriodStatusCalculated:
		return true
	default:
		return false
	}
}

// CanApprove reports whether the period may be confirmed.
func (pp *PayrollPeriod) CanApprove() bool {
	return pp.Status == PayrollPeriodStatusCalculated
}

// CanPay reports whether the period may be marked paid.
func (pp *PayrollPeriod) CanPay() bool {
	return pp.Status == PayrollPeriodStatusApproved
}

// CanClose reports whether the period may be closed.
func (pp *PayrollPeriod) CanClose() bool {
	return pp.Status == PayrollPeriodStatusPaid
}

// ---------------------------------------------------------------------------
// Calculation input / output
// ---------------------------------------------------------------------------

// PayrollNamedAmount is an earning or deduction that has no dedicated column.
type PayrollNamedAmount struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Amount int64  `json:"amount"`

	// NonTaxable excludes the amount from 과세대상 근로소득 and therefore also
	// from the 보수월액 that the four insurances are assessed on.
	//
	// The CALLER decides this, per line. This package does not infer it from
	// the item name and does not apply the statutory ceilings (식대 월 20만원,
	// 자가운전보조금 월 20만원 등, 소득세법 시행령 제12조·제17조의2): those are
	// per-item limits whose current values are not loaded here, and guessing
	// one produces a 원천징수 filing that looks right and is wrong. See CONFIRM.
	NonTaxable bool `json:"non_taxable"`
}

// PayrollEarnings is the gross pay side of a payroll calculation.
type PayrollEarnings struct {
	BaseSalary  int64 `json:"base_salary"`
	OvertimePay int64 `json:"overtime_pay"`
	NightPay    int64 `json:"night_pay"`
	HolidayPay  int64 `json:"holiday_pay"`
	Bonus       int64 `json:"bonus"`

	Allowances    []PayrollNamedAmount `json:"allowances,omitempty"`
	OtherEarnings []PayrollNamedAmount `json:"other_earnings,omitempty"`
}

// PayrollDeductionInput carries the deductions this package cannot derive.
type PayrollDeductionInput struct {
	// IncomeTax 근로소득 원천징수세액.
	//
	// REQUIRED in practice. CalculateEarnedIncomeWithholding is deliberately
	// unimplemented because the 근로소득 간이세액표 (소득세법 시행령 별표2) is
	// not loaded, so leaving this nil makes the calculation fail with
	// ErrSimplifiedTaxTableUnavailable instead of inventing a number.
	IncomeTax *int64 `json:"income_tax,omitempty"`

	// LocalIncomeTax 지방소득세. When nil it is derived as 10% of IncomeTax
	// (지방세법 제103조의13), truncated below 10 won.
	LocalIncomeTax *int64 `json:"local_income_tax,omitempty"`

	// Others 기타공제 (노조회비, 가불금 상계 등). Must be non-negative: the
	// chk_payrolls_amounts constraint requires total_deductions >= 0.
	Others []PayrollNamedAmount `json:"others,omitempty"`
}

// SocialInsuranceQualification mirrors employee_insurance.*_qualified. An
// employee who has not acquired (or who has lost) a qualification has no
// contribution withheld for it.
type SocialInsuranceQualification struct {
	NationalPension     bool `json:"national_pension"`
	HealthInsurance     bool `json:"health_insurance"`
	EmploymentInsurance bool `json:"employment_insurance"`
	IndustrialAccident  bool `json:"industrial_accident"`
}

// AllQualified is the default for a regular full-time employee.
func AllQualified() SocialInsuranceQualification {
	return SocialInsuranceQualification{
		NationalPension:     true,
		HealthInsurance:     true,
		EmploymentInsurance: true,
		IndustrialAccident:  true,
	}
}

// PayrollCalculationInput is everything CalculatePayroll needs.
type PayrollCalculationInput struct {
	PayYear  int
	PayMonth int

	// ContributionDate selects the annual rate row AND the 국민연금
	// 기준소득월액 window, which are revised on different schedules (the
	// pension ceiling changes every July). It is normally the payment date.
	ContributionDate time.Time

	Earnings   PayrollEarnings
	Deductions PayrollDeductionInput

	Qualification SocialInsuranceQualification

	// WorkplaceSize selects the 고용안정·직업능력개발사업 band, which the
	// employer pays in full on top of its 실업급여 share
	// (고용보험법 시행령 제12조).
	WorkplaceSize WorkplaceSize

	// IndustrialAccidentRate 산재보험료율 in hundred-thousandths (1.53% ->
	// 1530). There is no single nationwide rate - it is published per industry
	// - so a nil value means 산재보험료 is NOT computed and the result says so
	// through IndustrialAccidentApplied. It is never silently treated as zero.
	//
	// CONFIRM: the caller must pass the TOTAL rate including 출퇴근재해요율.
	// This function does not add SocialInsuranceRates.CommuteAccidentRate,
	// because the 고시 rate a user copies in may or may not already include it
	// and double counting is as wrong as omitting.
	IndustrialAccidentRate *int64
}

// PayrollItemLine is one line CalculatePayroll produces. The persisted
// payroll_items rows and the payroll header totals are BOTH derived from this
// single list, so the header can never disagree with the lines.
type PayrollItemLine struct {
	ItemType  PayrollItemType `json:"item_type"`
	ItemCode  string          `json:"item_code"`
	ItemName  string          `json:"item_name"`
	Amount    int64           `json:"amount"`
	IsTaxable bool            `json:"is_taxable"`
	IsFixed   bool            `json:"is_fixed"`
	SortOrder int             `json:"sort_order"`
}

// PayrollEmployerCost is the employer's side of one month's payroll.
type PayrollEmployerCost struct {
	NationalPension     int64 `json:"national_pension"`
	HealthInsurance     int64 `json:"health_insurance"`
	LongTermCare        int64 `json:"long_term_care"`
	EmploymentInsurance int64 `json:"employment_insurance"`
	IndustrialAccident  int64 `json:"industrial_accident"`
	Total               int64 `json:"total"`
}

// PayrollCalculationResult is the full outcome of one payroll calculation.
type PayrollCalculationResult struct {
	PayYear  int `json:"pay_year"`
	PayMonth int `json:"pay_month"`

	TotalEarnings    int64 `json:"total_earnings"`
	NonTaxableAmount int64 `json:"non_taxable_amount"`

	// TaxableWage 과세대상 보수월액: the base the four insurances are assessed
	// on and the base a 간이세액 lookup would use.
	TaxableWage int64 `json:"taxable_wage"`

	SocialInsurance SocialInsuranceResult `json:"social_insurance"`

	IncomeTax           int64 `json:"income_tax"`
	LocalIncomeTax      int64 `json:"local_income_tax"`
	OtherDeductionTotal int64 `json:"other_deduction_total"`
	TotalDeductions     int64 `json:"total_deductions"`

	// NetPay may be negative: a month whose deductions exceed gross pay is
	// legitimate (mid-month joiner with a 연말정산 clawback), which is why
	// chk_payrolls_amounts does not constrain it.
	NetPay int64 `json:"net_pay"`

	EmployerCost PayrollEmployerCost `json:"employer_cost"`

	Items []PayrollItemLine `json:"items"`

	// IndustrialAccidentApplied is false when no industry rate was supplied.
	// The UI must show 산재보험료 as "미산정", not as 0원.
	IndustrialAccidentApplied bool `json:"industrial_accident_applied"`

	// EmploymentStabilityRate is the 고용안정·직업능력개발사업 rate that was
	// added to the employer's 고용보험료, in hundred-thousandths.
	EmploymentStabilityRate int64 `json:"employment_stability_rate"`

	// RatesSource echoes the authority behind the rate row that was used.
	RatesSource string `json:"rates_source"`
}

// ---------------------------------------------------------------------------
// Calculation
// ---------------------------------------------------------------------------

// CalculatePayroll computes one employee's payroll for one month.
//
// It composes the existing statutory helpers rather than reimplementing them:
// CalculateSocialInsurance for 국민연금·건강보험·장기요양·고용보험,
// CalculateIndustrialAccident for 산재보험, LocalIncomeTaxBps + RoundDownTo for
// 지방소득세. A year with no verified rate row makes this fail with
// ErrSocialInsuranceRatesUnavailable, and a nil IncomeTax makes it fail with
// ErrSimplifiedTaxTableUnavailable. Both are deliberate: see the CONFIRM blocks
// in social_insurance.go and withholding.go.
func CalculatePayroll(in PayrollCalculationInput) (PayrollCalculationResult, error) {
	if in.PayMonth < 1 || in.PayMonth > 12 {
		return PayrollCalculationResult{}, ErrPayrollMonthInvalid
	}

	gross, nonTaxable, err := sumEarnings(in.Earnings)
	if err != nil {
		return PayrollCalculationResult{}, err
	}
	if nonTaxable > gross {
		return PayrollCalculationResult{}, ErrNonTaxableExceedsGross
	}
	taxableWage := gross - nonTaxable

	// 4대보험. Always evaluated, even for an employee exempt from all four, so
	// that a missing rate row for the year is reported instead of silently
	// producing zeros.
	si, err := CalculateSocialInsurance(taxableWage, in.ContributionDate)
	if err != nil {
		return PayrollCalculationResult{}, err
	}
	rates, err := SocialInsuranceRatesFor(in.ContributionDate.Year())
	if err != nil {
		return PayrollCalculationResult{}, err
	}

	if !in.Qualification.NationalPension {
		si.NationalPension = SocialInsuranceContribution{}
	}
	if !in.Qualification.HealthInsurance {
		// 장기요양보험료 is assessed on the 건강보험료 itself
		// (노인장기요양보험법 제9조), so it cannot outlive it.
		si.HealthInsurance = SocialInsuranceContribution{}
		si.LongTermCare = SocialInsuranceContribution{}
	}
	if !in.Qualification.EmploymentInsurance {
		si.EmploymentInsurance = SocialInsuranceContribution{}
	}

	// 고용안정·직업능력개발사업: employer only, and CalculateSocialInsurance
	// deliberately leaves it out because it does not know the workplace size.
	var stabilityRate, stabilityPremium int64
	if in.Qualification.EmploymentInsurance {
		stabilityRate = rates.EmploymentStabilityRate(in.WorkplaceSize)
		stabilityPremium = applyRate(taxableWage, stabilityRate)
	}

	// 산재보험: employer only, industry rate supplied by the caller.
	industrialApplied := false
	if in.Qualification.IndustrialAccident && in.IndustrialAccidentRate != nil {
		wci, err := CalculateIndustrialAccident(taxableWage, *in.IndustrialAccidentRate)
		if err != nil {
			return PayrollCalculationResult{}, err
		}
		si.IndustrialAccident = wci
		industrialApplied = true
	} else {
		si.IndustrialAccident = SocialInsuranceContribution{}
	}

	// 근로소득세. No table, no guess.
	var incomeTax int64
	if in.Deductions.IncomeTax == nil {
		return PayrollCalculationResult{}, ErrSimplifiedTaxTableUnavailable
	}
	incomeTax = *in.Deductions.IncomeTax
	if incomeTax < 0 {
		return PayrollCalculationResult{}, fmt.Errorf("income tax: %w", ErrNegativeAmount)
	}

	// 지방소득세 = 소득세의 10% (지방세법 제103조의13), 10원 미만 절사. It is
	// derived from the already-truncated income tax, exactly as
	// CalculateBusinessIncomeWithholding does it.
	var localIncomeTax int64
	if in.Deductions.LocalIncomeTax != nil {
		localIncomeTax = *in.Deductions.LocalIncomeTax
		if localIncomeTax < 0 {
			return PayrollCalculationResult{}, fmt.Errorf("local income tax: %w", ErrNegativeAmount)
		}
	} else {
		localIncomeTax = RoundDownTo(incomeTax*LocalIncomeTaxBps/10000, TenWonUnit)
	}

	var otherDeductionTotal int64
	for _, d := range in.Deductions.Others {
		if err := validateNamedAmount(d); err != nil {
			return PayrollCalculationResult{}, err
		}
		otherDeductionTotal += d.Amount
	}

	employeeSI := si.EmployeeTotal()
	totalDeductions := employeeSI + incomeTax + localIncomeTax + otherDeductionTotal

	employer := PayrollEmployerCost{
		NationalPension:     si.NationalPension.Employer,
		HealthInsurance:     si.HealthInsurance.Employer,
		LongTermCare:        si.LongTermCare.Employer,
		EmploymentInsurance: si.EmploymentInsurance.Employer + stabilityPremium,
		IndustrialAccident:  si.IndustrialAccident.Employer,
	}
	employer.Total = employer.NationalPension + employer.HealthInsurance +
		employer.LongTermCare + employer.EmploymentInsurance + employer.IndustrialAccident

	result := PayrollCalculationResult{
		PayYear:                   in.PayYear,
		PayMonth:                  in.PayMonth,
		TotalEarnings:             gross,
		NonTaxableAmount:          nonTaxable,
		TaxableWage:               taxableWage,
		SocialInsurance:           si,
		IncomeTax:                 incomeTax,
		LocalIncomeTax:            localIncomeTax,
		OtherDeductionTotal:       otherDeductionTotal,
		TotalDeductions:           totalDeductions,
		NetPay:                    gross - totalDeductions,
		EmployerCost:              employer,
		IndustrialAccidentApplied: industrialApplied,
		EmploymentStabilityRate:   stabilityRate,
		RatesSource:               si.RatesSource,
	}
	result.Items = buildPayrollItems(in, result)

	// Self-check: the lines and the header come from this function, so a
	// mismatch here is a bug in this function, not bad input. Catching it here
	// keeps it out of the database.
	if err := ValidatePayrollItemTotals(result.TotalEarnings, result.TotalDeductions, result.NetPay, result.Items); err != nil {
		return PayrollCalculationResult{}, err
	}

	return result, nil
}

// sumEarnings totals the gross pay and the declared non-taxable portion.
func sumEarnings(e PayrollEarnings) (gross, nonTaxable int64, err error) {
	fixed := []struct {
		name   string
		amount int64
	}{
		{"base_salary", e.BaseSalary},
		{"overtime_pay", e.OvertimePay},
		{"night_pay", e.NightPay},
		{"holiday_pay", e.HolidayPay},
		{"bonus", e.Bonus},
	}
	for _, f := range fixed {
		if f.amount < 0 {
			return 0, 0, fmt.Errorf("%s: %w", f.name, ErrNegativeAmount)
		}
		gross += f.amount
	}

	for _, group := range [][]PayrollNamedAmount{e.Allowances, e.OtherEarnings} {
		for _, a := range group {
			if err := validateNamedAmount(a); err != nil {
				return 0, 0, err
			}
			gross += a.Amount
			if a.NonTaxable {
				nonTaxable += a.Amount
			}
		}
	}

	return gross, nonTaxable, nil
}

// validateNamedAmount rejects a line that cannot be stored or reconciled.
func validateNamedAmount(a PayrollNamedAmount) error {
	if a.Code == "" || a.Name == "" {
		return ErrPayrollEmptyItemName
	}
	if a.Amount < 0 {
		return fmt.Errorf("%s: %w", a.Code, ErrNegativeAmount)
	}
	return nil
}

// buildPayrollItems turns the calculation into the line items that are stored
// in payroll_items. Every non-zero component becomes exactly one line.
func buildPayrollItems(in PayrollCalculationInput, r PayrollCalculationResult) []PayrollItemLine {
	items := make([]PayrollItemLine, 0, 16)
	order := 0

	addEarning := func(code, name string, amount int64, taxable, fixed bool) {
		if amount == 0 {
			return
		}
		order++
		items = append(items, PayrollItemLine{
			ItemType:  PayrollItemEarning,
			ItemCode:  code,
			ItemName:  name,
			Amount:    amount,
			IsTaxable: taxable,
			IsFixed:   fixed,
			SortOrder: order,
		})
	}
	addDeduction := func(code, name string, amount int64) {
		if amount == 0 {
			return
		}
		order++
		items = append(items, PayrollItemLine{
			ItemType:  PayrollItemDeduction,
			ItemCode:  code,
			ItemName:  name,
			Amount:    amount,
			IsTaxable: false,
			IsFixed:   true,
			SortOrder: order,
		})
	}

	addEarning(PayrollItemCodeBaseSalary, "기본급", in.Earnings.BaseSalary, true, true)
	addEarning(PayrollItemCodeOvertimePay, "연장근로수당", in.Earnings.OvertimePay, true, false)
	addEarning(PayrollItemCodeNightPay, "야간근로수당", in.Earnings.NightPay, true, false)
	addEarning(PayrollItemCodeHolidayPay, "휴일근로수당", in.Earnings.HolidayPay, true, false)
	addEarning(PayrollItemCodeBonus, "상여금", in.Earnings.Bonus, true, false)
	for _, a := range in.Earnings.Allowances {
		addEarning(a.Code, a.Name, a.Amount, !a.NonTaxable, true)
	}
	for _, a := range in.Earnings.OtherEarnings {
		addEarning(a.Code, a.Name, a.Amount, !a.NonTaxable, false)
	}

	addDeduction(PayrollItemCodeNPS, "국민연금", r.SocialInsurance.NationalPension.Employee)
	addDeduction(PayrollItemCodeNHIS, "건강보험", r.SocialInsurance.HealthInsurance.Employee)
	addDeduction(PayrollItemCodeNHISLTC, "장기요양보험", r.SocialInsurance.LongTermCare.Employee)
	addDeduction(PayrollItemCodeEI, "고용보험", r.SocialInsurance.EmploymentInsurance.Employee)
	addDeduction(PayrollItemCodeIncomeTax, "소득세", r.IncomeTax)
	addDeduction(PayrollItemCodeLocalIncomeTax, "지방소득세", r.LocalIncomeTax)
	for _, d := range in.Deductions.Others {
		addDeduction(d.Code, d.Name, d.Amount)
	}

	return items
}

// ---------------------------------------------------------------------------
// Reconciliation
// ---------------------------------------------------------------------------

// ValidatePayrollItemTotals checks that a header agrees with its lines.
//
// This is the check the tax invoice code was missing: a header whose total does
// not equal the sum of its rows is a document that reconciles against nothing,
// and once it is committed there is no way to tell which of the two numbers was
// meant. Amounts are int64 whole won so the comparison is exact - no epsilon.
func ValidatePayrollItemTotals(totalEarnings, totalDeductions, netPay int64, items []PayrollItemLine) error {
	var earnings, deductions int64
	for _, it := range items {
		switch it.ItemType {
		case PayrollItemEarning:
			earnings += it.Amount
		case PayrollItemDeduction:
			deductions += it.Amount
		default:
			return fmt.Errorf("%w: unknown item type %q", ErrPayrollTotalsMismatch, it.ItemType)
		}
	}
	return comparePayrollTotals(totalEarnings, totalDeductions, netPay, earnings, deductions)
}

// ValidatePayrollTotals is ValidatePayrollItemTotals for a persisted payroll and
// the rows actually stored against it. The service runs it before approval, so
// nothing is confirmed whose register does not add up.
func ValidatePayrollTotals(p *Payroll, items []PayrollItem) error {
	var earnings, deductions int64
	for _, it := range items {
		switch it.ItemType {
		case PayrollItemEarning:
			earnings += it.Amount
		case PayrollItemDeduction:
			deductions += it.Amount
		default:
			return fmt.Errorf("%w: unknown item type %q", ErrPayrollTotalsMismatch, it.ItemType)
		}
	}
	return comparePayrollTotals(p.TotalEarnings, p.TotalDeductions, p.NetPay, earnings, deductions)
}

// comparePayrollTotals is the shared body of the two validators above.
func comparePayrollTotals(headerEarnings, headerDeductions, headerNet, lineEarnings, lineDeductions int64) error {
	if headerEarnings != lineEarnings {
		return fmt.Errorf("%w: 지급총액 %d, 지급항목 합계 %d",
			ErrPayrollTotalsMismatch, headerEarnings, lineEarnings)
	}
	if headerDeductions != lineDeductions {
		return fmt.Errorf("%w: 공제총액 %d, 공제항목 합계 %d",
			ErrPayrollTotalsMismatch, headerDeductions, lineDeductions)
	}
	if headerNet != headerEarnings-headerDeductions {
		return fmt.Errorf("%w: 실지급액 %d, 지급총액-공제총액 %d",
			ErrPayrollTotalsMismatch, headerNet, headerEarnings-headerDeductions)
	}
	return nil
}

// ApplyCalculation copies a calculation onto a payroll record. The caller
// persists the record and the items returned by the same calculation inside one
// transaction; splitting them is what leaves a header that disagrees with its
// rows.
func (p *Payroll) ApplyCalculation(r PayrollCalculationResult) {
	p.TotalEarnings = r.TotalEarnings
	p.IncomeTax = r.IncomeTax
	p.LocalIncomeTax = r.LocalIncomeTax
	p.NPSEmployee = r.SocialInsurance.NationalPension.Employee
	p.NHISEmployee = r.SocialInsurance.HealthInsurance.Employee
	p.NHISLTCEmployee = r.SocialInsurance.LongTermCare.Employee
	p.EIEmployee = r.SocialInsurance.EmploymentInsurance.Employee
	p.TotalDeductions = r.TotalDeductions
	p.NetPay = r.NetPay

	p.NPSEmployer = r.EmployerCost.NationalPension
	p.NHISEmployer = r.EmployerCost.HealthInsurance
	p.NHISLTCEmployer = r.EmployerCost.LongTermCare
	p.EIEmployer = r.EmployerCost.EmploymentInsurance
	p.WCIEmployer = r.EmployerCost.IndustrialAccident
	p.TotalEmployerCost = r.EmployerCost.Total

	// The JSONB summaries are rebuilt from the same item list the rows come
	// from, so the two representations cannot drift apart.
	p.Allowances = namedAmountMap(r.Items, PayrollItemEarning, payrollFixedEarningCodes)
	p.OtherDeductions = namedAmountMap(r.Items, PayrollItemDeduction, payrollStatutoryDeductionCodes)
	p.Status = PayrollStatusCalculated
}

// payrollFixedEarningCodes are the earnings that have their own column, so they
// are not repeated in the allowances JSONB summary.
var payrollFixedEarningCodes = map[string]bool{
	PayrollItemCodeBaseSalary:  true,
	PayrollItemCodeOvertimePay: true,
	PayrollItemCodeNightPay:    true,
	PayrollItemCodeHolidayPay:  true,
	PayrollItemCodeBonus:       true,
}

// payrollStatutoryDeductionCodes are the deductions that have their own column.
var payrollStatutoryDeductionCodes = map[string]bool{
	PayrollItemCodeNPS:            true,
	PayrollItemCodeNHIS:           true,
	PayrollItemCodeNHISLTC:        true,
	PayrollItemCodeEI:             true,
	PayrollItemCodeIncomeTax:      true,
	PayrollItemCodeLocalIncomeTax: true,
}

// namedAmountMap collects the item lines of one type whose code is not already
// a dedicated column.
func namedAmountMap(items []PayrollItemLine, itemType PayrollItemType, skip map[string]bool) PayrollAmountMap {
	out := PayrollAmountMap{}
	for _, it := range items {
		if it.ItemType != itemType || skip[it.ItemCode] {
			continue
		}
		out[it.ItemCode] += it.Amount
	}
	return out
}

// ToPayrollItems turns calculation lines into storable rows.
func ToPayrollItems(companyID, payrollID uuid.UUID, lines []PayrollItemLine) []PayrollItem {
	items := make([]PayrollItem, 0, len(lines))
	for _, l := range lines {
		items = append(items, PayrollItem{
			PayrollID: payrollID,
			CompanyID: companyID,
			ItemType:  l.ItemType,
			ItemCode:  l.ItemCode,
			ItemName:  l.ItemName,
			Amount:    l.Amount,
			IsTaxable: l.IsTaxable,
			IsFixed:   l.IsFixed,
			SortOrder: l.SortOrder,
		})
	}
	return items
}

// ============================================================================
// CONFIRM / 확인 필요
// ============================================================================
//
//  1. 근로소득 간이세액표 (소득세법 시행령 별표2).
//     Not loaded. CalculatePayroll therefore REQUIRES the caller to supply
//     income_tax and fails with ErrSimplifiedTaxTableUnavailable otherwise.
//     Loading the NTS table is what makes automatic monthly withholding
//     possible; until then every payroll carries an operator-entered figure.
//
//  2. 비과세 한도.
//     Which allowances are non-taxable, and up to what monthly ceiling
//     (식대 20만원, 자가운전보조금 20만원, 출산·보육수당 등), is declared by the
//     caller per line through PayrollNamedAmount.NonTaxable. This package does
//     NOT enforce the ceilings, so an allowance flagged non-taxable above its
//     statutory limit reduces both 과세표준 and 보수월액 too far. Load
//     소득세법 시행령 제12조·제17조의2 limits before relying on this.
//
//  3. 산재보험료율.
//     Industry-specific; supplied by the caller. 출퇴근재해요율 is NOT added
//     automatically - pass the total rate. SocialInsuranceRates.CommuteAccidentRate
//     (0.06%) is available if the caller needs to add it.
//
//  4. 4대보험 단수처리.
//     Inherited from CalculateSocialInsurance: 10원 미만 절사 on each
//     insurance. Reconcile against a real 공단 고지서 before relying on it.
//
//  5. 일할계산 (mid-month joiners and leavers).
//     Not implemented. A month in which an employee starts or resigns is
//     computed on the full amounts the caller passes in; prorating the base
//     salary and the insurance base is the caller's job today.
