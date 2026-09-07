package dto

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// ErrInvalidPayrollDate is returned when a date field is not YYYY-MM-DD.
var ErrInvalidPayrollDate = errors.New("payroll dates must be formatted as YYYY-MM-DD")

// payrollDateLayout is the only date format the payroll API accepts.
const payrollDateLayout = "2006-01-02"

// parsePayrollDate parses an optional YYYY-MM-DD field.
func parsePayrollDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	t, err := time.Parse(payrollDateLayout, value)
	if err != nil {
		return nil, ErrInvalidPayrollDate
	}
	return &t, nil
}

// ParseOptionalPayrollDate parses an optional YYYY-MM-DD field. An empty value
// yields (nil, nil); a malformed one yields ErrInvalidPayrollDate.
func ParseOptionalPayrollDate(value string) (*time.Time, error) {
	return parsePayrollDate(value)
}

// ParseRequiredPayrollDate parses a mandatory YYYY-MM-DD field.
func ParseRequiredPayrollDate(value string) (time.Time, error) {
	t, err := parsePayrollDate(value)
	if err != nil {
		return time.Time{}, err
	}
	if t == nil {
		return time.Time{}, ErrInvalidPayrollDate
	}
	return *t, nil
}

// payrollDateString renders a date column for the wire.
func payrollDateString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(payrollDateLayout)
}

// payrollDatePtrString renders an optional date column for the wire.
func payrollDatePtrString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return payrollDateString(*t)
}

// ---------------------------------------------------------------------------
// Requests
// ---------------------------------------------------------------------------

// PayrollNamedAmountRequest is one named earning or deduction line.
type PayrollNamedAmountRequest struct {
	Code   string `json:"code" binding:"required,max=50"`
	Name   string `json:"name" binding:"required,max=100"`
	Amount int64  `json:"amount" binding:"min=0"`

	// NonTaxable declares this line 비과세. The server does NOT infer it from
	// the name and does NOT enforce the statutory monthly ceilings - see the
	// CONFIRM block in internal/domain/payroll.go.
	NonTaxable bool `json:"non_taxable"`
}

// toNamedAmounts converts request lines to the domain type.
func toNamedAmounts(reqs []PayrollNamedAmountRequest) []domain.PayrollNamedAmount {
	if len(reqs) == 0 {
		return nil
	}
	out := make([]domain.PayrollNamedAmount, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, domain.PayrollNamedAmount{
			Code:       r.Code,
			Name:       r.Name,
			Amount:     r.Amount,
			NonTaxable: r.NonTaxable,
		})
	}
	return out
}

// PayrollEarningsRequest is the gross pay side of a payroll request.
type PayrollEarningsRequest struct {
	BaseSalary  int64 `json:"base_salary" binding:"min=0"`
	OvertimePay int64 `json:"overtime_pay" binding:"min=0"`
	NightPay    int64 `json:"night_pay" binding:"min=0"`
	HolidayPay  int64 `json:"holiday_pay" binding:"min=0"`
	Bonus       int64 `json:"bonus" binding:"min=0"`

	Allowances    []PayrollNamedAmountRequest `json:"allowances" binding:"omitempty,dive"`
	OtherEarnings []PayrollNamedAmountRequest `json:"other_earnings" binding:"omitempty,dive"`
}

// ToDomain converts the request into the calculation input.
func (r *PayrollEarningsRequest) ToDomain() domain.PayrollEarnings {
	return domain.PayrollEarnings{
		BaseSalary:    r.BaseSalary,
		OvertimePay:   r.OvertimePay,
		NightPay:      r.NightPay,
		HolidayPay:    r.HolidayPay,
		Bonus:         r.Bonus,
		Allowances:    toNamedAmounts(r.Allowances),
		OtherEarnings: toNamedAmounts(r.OtherEarnings),
	}
}

// CreatePayrollPeriodRequest opens a monthly payroll run.
type CreatePayrollPeriodRequest struct {
	PayYear     int    `json:"pay_year" binding:"required,min=2000,max=2100"`
	PayMonth    int    `json:"pay_month" binding:"required,min=1,max=12"`
	PeriodName  string `json:"period_name" binding:"max=50"`
	PeriodStart string `json:"period_start" binding:"required"`
	PeriodEnd   string `json:"period_end" binding:"required"`
	PaymentDate string `json:"payment_date"`
}

// CreatePayrollRequest opens a draft payroll record.
type CreatePayrollRequest struct {
	EmployeeID      string `json:"employee_id" binding:"required,uuid"`
	PayrollPeriodID string `json:"payroll_period_id" binding:"omitempty,uuid"`
	PayYear         int    `json:"pay_year" binding:"required,min=2000,max=2100"`
	PayMonth        int    `json:"pay_month" binding:"required,min=1,max=12"`
	PaymentDate     string `json:"payment_date"`

	Earnings PayrollEarningsRequest `json:"earnings"`

	WorkDays      int     `json:"work_days" binding:"min=0"`
	OvertimeHours float64 `json:"overtime_hours" binding:"min=0"`
	NightHours    float64 `json:"night_hours" binding:"min=0"`
	HolidayHours  float64 `json:"holiday_hours" binding:"min=0"`

	Notes    string `json:"notes" binding:"max=500"`
	BankCode string `json:"bank_code" binding:"max=10"`
}

// UpdatePayrollRequest replaces the earnings of a payroll record.
type UpdatePayrollRequest struct {
	Earnings PayrollEarningsRequest `json:"earnings"`

	WorkDays      int     `json:"work_days" binding:"min=0"`
	OvertimeHours float64 `json:"overtime_hours" binding:"min=0"`
	NightHours    float64 `json:"night_hours" binding:"min=0"`
	HolidayHours  float64 `json:"holiday_hours" binding:"min=0"`

	Notes    string `json:"notes" binding:"max=500"`
	BankCode string `json:"bank_code" binding:"max=10"`
}

// CalculatePayrollRequest carries the figures the server cannot derive.
//
// income_tax is required in practice: the 근로소득 간이세액표 is not loaded, so
// the server refuses to invent a monthly withholding amount. The only case in
// which it may be omitted is a recalculation of a payroll that already carries
// an operator-entered figure.
type CalculatePayrollRequest struct {
	IncomeTax      *int64 `json:"income_tax" binding:"omitempty,min=0"`
	LocalIncomeTax *int64 `json:"local_income_tax" binding:"omitempty,min=0"`

	OtherDeductions []PayrollNamedAmountRequest `json:"other_deductions" binding:"omitempty,dive"`

	// WorkplaceSize selects the 고용안정·직업능력개발사업 band
	// (고용보험법 시행령 제12조). Defaults to under_150.
	WorkplaceSize string `json:"workplace_size" binding:"omitempty,oneof=under_150 priority_150_plus from_150_to_999 over_1000"`

	// IndustrialAccidentRate is the 산재보험료율 in hundred-thousandths
	// (1.53% -> 1530). Omitting it means 산재보험료 is NOT computed; the
	// response reports that as industrial_accident_applied=false rather than
	// as a premium of zero.
	IndustrialAccidentRate *int64 `json:"industrial_accident_rate" binding:"omitempty,min=0"`

	// ContributionDate selects the rate year and the 국민연금 기준소득월액
	// window. Defaults to the payment date, then to the first of the pay month.
	ContributionDate string `json:"contribution_date"`
}

// Parsed returns the fields of the request that need conversion: the
// contribution date and the workplace-size band with its default applied.
//
// The handler assembles service.CalculatePayrollInput from this. internal/dto
// deliberately does not import internal/service - the two would form a cycle
// through internal/repository - so the crossing is done field by field at the
// handler, which imports both.
func (r *CalculatePayrollRequest) Parsed() (*time.Time, domain.WorkplaceSize, error) {
	contributionDate, err := parsePayrollDate(r.ContributionDate)
	if err != nil {
		return nil, "", err
	}
	size := domain.WorkplaceUnder150
	if r.WorkplaceSize != "" {
		size = domain.WorkplaceSize(r.WorkplaceSize)
	}
	return contributionDate, size, nil
}

// OtherDeductionLines converts the request's 기타공제 lines to the domain type.
func (r *CalculatePayrollRequest) OtherDeductionLines() []domain.PayrollNamedAmount {
	return toNamedAmounts(r.OtherDeductions)
}

// CalculatePeriodEmployeeRequest overrides the calculation for one employee.
type CalculatePeriodEmployeeRequest struct {
	EmployeeID string `json:"employee_id" binding:"required,uuid"`
	CalculatePayrollRequest
}

// CalculatePeriodRequest drives a whole-period recalculation.
type CalculatePeriodRequest struct {
	WorkplaceSize          string `json:"workplace_size" binding:"omitempty,oneof=under_150 priority_150_plus from_150_to_999 over_1000"`
	IndustrialAccidentRate *int64 `json:"industrial_accident_rate" binding:"omitempty,min=0"`
	ContributionDate       string `json:"contribution_date"`

	Employees []CalculatePeriodEmployeeRequest `json:"employees" binding:"omitempty,dive"`
}

// MarkPeriodPaidRequest records the payout of an approved run.
type MarkPeriodPaidRequest struct {
	PaymentDate string `json:"payment_date"`
}

// PayrollPreviewRequest computes a payslip without storing anything.
type PayrollPreviewRequest struct {
	PayYear  int `json:"pay_year" binding:"required,min=2000,max=2100"`
	PayMonth int `json:"pay_month" binding:"required,min=1,max=12"`

	ContributionDate string `json:"contribution_date"`

	Earnings PayrollEarningsRequest `json:"earnings"`

	IncomeTax       *int64                      `json:"income_tax" binding:"omitempty,min=0"`
	LocalIncomeTax  *int64                      `json:"local_income_tax" binding:"omitempty,min=0"`
	OtherDeductions []PayrollNamedAmountRequest `json:"other_deductions" binding:"omitempty,dive"`

	WorkplaceSize          string `json:"workplace_size" binding:"omitempty,oneof=under_150 priority_150_plus from_150_to_999 over_1000"`
	IndustrialAccidentRate *int64 `json:"industrial_accident_rate" binding:"omitempty,min=0"`

	// Qualification flags. All four default to true, which is a regular
	// full-time hire.
	NPSQualified  *bool `json:"nps_qualified"`
	NHISQualified *bool `json:"nhis_qualified"`
	EIQualified   *bool `json:"ei_qualified"`
	WCIQualified  *bool `json:"wci_qualified"`
}

// ToDomain converts a preview request into the calculation input.
func (r *PayrollPreviewRequest) ToDomain() (*domain.PayrollCalculationInput, error) {
	contributionDate, err := parsePayrollDate(r.ContributionDate)
	if err != nil {
		return nil, err
	}
	on := time.Date(r.PayYear, time.Month(r.PayMonth), 1, 0, 0, 0, 0, time.UTC)
	if contributionDate != nil {
		on = *contributionDate
	}

	size := domain.WorkplaceUnder150
	if r.WorkplaceSize != "" {
		size = domain.WorkplaceSize(r.WorkplaceSize)
	}

	qualification := domain.AllQualified()
	if r.NPSQualified != nil {
		qualification.NationalPension = *r.NPSQualified
	}
	if r.NHISQualified != nil {
		qualification.HealthInsurance = *r.NHISQualified
	}
	if r.EIQualified != nil {
		qualification.EmploymentInsurance = *r.EIQualified
	}
	if r.WCIQualified != nil {
		qualification.IndustrialAccident = *r.WCIQualified
	}

	return &domain.PayrollCalculationInput{
		PayYear:          r.PayYear,
		PayMonth:         r.PayMonth,
		ContributionDate: on,
		Earnings:         r.Earnings.ToDomain(),
		Deductions: domain.PayrollDeductionInput{
			IncomeTax:      r.IncomeTax,
			LocalIncomeTax: r.LocalIncomeTax,
			Others:         toNamedAmounts(r.OtherDeductions),
		},
		Qualification:          qualification,
		WorkplaceSize:          size,
		IndustrialAccidentRate: r.IndustrialAccidentRate,
	}, nil
}

// ---------------------------------------------------------------------------
// Responses
// ---------------------------------------------------------------------------

// PayrollPeriodResponse is one monthly payroll run.
type PayrollPeriodResponse struct {
	ID         uuid.UUID `json:"id"`
	PayYear    int       `json:"pay_year"`
	PayMonth   int       `json:"pay_month"`
	PeriodName string    `json:"period_name,omitempty"`

	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
	PaymentDate string `json:"payment_date,omitempty"`

	Status string `json:"status"`

	CalculatedAt *time.Time `json:"calculated_at,omitempty"`
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`

	TotalEmployees  int   `json:"total_employees"`
	TotalEarnings   int64 `json:"total_earnings"`
	TotalDeductions int64 `json:"total_deductions"`
	TotalNetPay     int64 `json:"total_net_pay"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FromPayrollPeriod converts a payroll period to its response shape.
func FromPayrollPeriod(p *domain.PayrollPeriod) PayrollPeriodResponse {
	return PayrollPeriodResponse{
		ID:              p.ID,
		PayYear:         p.PayYear,
		PayMonth:        p.PayMonth,
		PeriodName:      p.PeriodName,
		PeriodStart:     payrollDateString(p.PeriodStart),
		PeriodEnd:       payrollDateString(p.PeriodEnd),
		PaymentDate:     payrollDatePtrString(p.PaymentDate),
		Status:          string(p.Status),
		CalculatedAt:    p.CalculatedAt,
		ApprovedAt:      p.ApprovedAt,
		PaidAt:          p.PaidAt,
		TotalEmployees:  p.TotalEmployees,
		TotalEarnings:   p.TotalEarnings,
		TotalDeductions: p.TotalDeductions,
		TotalNetPay:     p.TotalNetPay,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

// FromPayrollPeriods converts a slice of payroll periods.
func FromPayrollPeriods(periods []*domain.PayrollPeriod) []PayrollPeriodResponse {
	out := make([]PayrollPeriodResponse, 0, len(periods))
	for _, p := range periods {
		out = append(out, FromPayrollPeriod(p))
	}
	return out
}

// PayrollItemResponse is one earning or deduction line.
type PayrollItemResponse struct {
	ItemType  string `json:"item_type"`
	ItemCode  string `json:"item_code"`
	ItemName  string `json:"item_name"`
	Amount    int64  `json:"amount"`
	IsTaxable bool   `json:"is_taxable"`
	IsFixed   bool   `json:"is_fixed"`
	SortOrder int    `json:"sort_order"`
}

// PayrollResponse is one employee's payroll record in full.
//
// It carries no bank account: domain.Payroll does not map
// payrolls.account_number_enc at all, so there is nothing here to leak.
type PayrollResponse struct {
	ID              uuid.UUID  `json:"id"`
	EmployeeID      uuid.UUID  `json:"employee_id"`
	PayrollPeriodID *uuid.UUID `json:"payroll_period_id,omitempty"`

	PayYear  int    `json:"pay_year"`
	PayMonth int    `json:"pay_month"`
	Status   string `json:"status"`

	BaseSalary    int64            `json:"base_salary"`
	OvertimePay   int64            `json:"overtime_pay"`
	NightPay      int64            `json:"night_pay"`
	HolidayPay    int64            `json:"holiday_pay"`
	Bonus         int64            `json:"bonus"`
	Allowances    map[string]int64 `json:"allowances,omitempty"`
	OtherEarnings map[string]int64 `json:"other_earnings,omitempty"`
	TotalEarnings int64            `json:"total_earnings"`

	IncomeTax      int64 `json:"income_tax"`
	LocalIncomeTax int64 `json:"local_income_tax"`

	NPSEmployee     int64 `json:"nps_employee"`
	NHISEmployee    int64 `json:"nhis_employee"`
	NHISLTCEmployee int64 `json:"nhis_ltc_employee"`
	EIEmployee      int64 `json:"ei_employee"`

	OtherDeductions map[string]int64 `json:"other_deductions,omitempty"`
	TotalDeductions int64            `json:"total_deductions"`

	NetPay int64 `json:"net_pay"`

	NPSEmployer       int64 `json:"nps_employer"`
	NHISEmployer      int64 `json:"nhis_employer"`
	NHISLTCEmployer   int64 `json:"nhis_ltc_employer"`
	EIEmployer        int64 `json:"ei_employer"`
	WCIEmployer       int64 `json:"wci_employer"`
	TotalEmployerCost int64 `json:"total_employer_cost"`

	PaymentDate string     `json:"payment_date,omitempty"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	BankCode    string     `json:"bank_code,omitempty"`

	WorkDays      int     `json:"work_days"`
	OvertimeHours float64 `json:"overtime_hours"`
	NightHours    float64 `json:"night_hours"`
	HolidayHours  float64 `json:"holiday_hours"`

	Notes string `json:"notes,omitempty"`

	Items []PayrollItemResponse `json:"items,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FromPayroll converts a payroll record to its response shape.
func FromPayroll(p *domain.Payroll) PayrollResponse {
	resp := PayrollResponse{
		ID:                p.ID,
		EmployeeID:        p.EmployeeID,
		PayrollPeriodID:   p.PayrollPeriodID,
		PayYear:           p.PayYear,
		PayMonth:          p.PayMonth,
		Status:            string(p.Status),
		BaseSalary:        p.BaseSalary,
		OvertimePay:       p.OvertimePay,
		NightPay:          p.NightPay,
		HolidayPay:        p.HolidayPay,
		Bonus:             p.Bonus,
		Allowances:        p.Allowances,
		OtherEarnings:     p.OtherEarnings,
		TotalEarnings:     p.TotalEarnings,
		IncomeTax:         p.IncomeTax,
		LocalIncomeTax:    p.LocalIncomeTax,
		NPSEmployee:       p.NPSEmployee,
		NHISEmployee:      p.NHISEmployee,
		NHISLTCEmployee:   p.NHISLTCEmployee,
		EIEmployee:        p.EIEmployee,
		OtherDeductions:   p.OtherDeductions,
		TotalDeductions:   p.TotalDeductions,
		NetPay:            p.NetPay,
		NPSEmployer:       p.NPSEmployer,
		NHISEmployer:      p.NHISEmployer,
		NHISLTCEmployer:   p.NHISLTCEmployer,
		EIEmployer:        p.EIEmployer,
		WCIEmployer:       p.WCIEmployer,
		TotalEmployerCost: p.TotalEmployerCost,
		PaymentDate:       payrollDatePtrString(p.PaymentDate),
		PaidAt:            p.PaidAt,
		BankCode:          p.BankCode,
		WorkDays:          p.WorkDays,
		OvertimeHours:     p.OvertimeHours,
		NightHours:        p.NightHours,
		HolidayHours:      p.HolidayHours,
		Notes:             p.Notes,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
	for _, it := range p.Items {
		resp.Items = append(resp.Items, PayrollItemResponse{
			ItemType:  string(it.ItemType),
			ItemCode:  it.ItemCode,
			ItemName:  it.ItemName,
			Amount:    it.Amount,
			IsTaxable: it.IsTaxable,
			IsFixed:   it.IsFixed,
			SortOrder: it.SortOrder,
		})
	}
	return resp
}

// PayrollListItemResponse is one row of the payroll register.
type PayrollListItemResponse struct {
	ID              uuid.UUID  `json:"id"`
	EmployeeID      uuid.UUID  `json:"employee_id"`
	EmployeeNo      string     `json:"employee_no"`
	EmployeeName    string     `json:"employee_name"`
	DepartmentName  string     `json:"department_name"`
	PayrollPeriodID *uuid.UUID `json:"payroll_period_id,omitempty"`
	PayYear         int        `json:"pay_year"`
	PayMonth        int        `json:"pay_month"`
	Status          string     `json:"status"`
	BaseSalary      int64      `json:"base_salary"`
	TotalEarnings   int64      `json:"total_earnings"`
	TotalDeductions int64      `json:"total_deductions"`
	NetPay          int64      `json:"net_pay"`
	PaymentDate     string     `json:"payment_date,omitempty"`
}

// FromPayrollListRows converts repository rows to the list response.
func FromPayrollListRows(rows []*repository.PayrollListRow) []PayrollListItemResponse {
	out := make([]PayrollListItemResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, PayrollListItemResponse{
			ID:              r.ID,
			EmployeeID:      r.EmployeeID,
			EmployeeNo:      r.EmployeeNo,
			EmployeeName:    r.EmployeeName,
			DepartmentName:  r.DepartmentName,
			PayrollPeriodID: r.PayrollPeriodID,
			PayYear:         r.PayYear,
			PayMonth:        r.PayMonth,
			Status:          r.Status,
			BaseSalary:      r.BaseSalary,
			TotalEarnings:   r.TotalEarnings,
			TotalDeductions: r.TotalDeductions,
			NetPay:          r.NetPay,
			PaymentDate:     payrollDatePtrString(r.PaymentDate),
		})
	}
	return out
}

// SocialInsuranceSplitResponse is one insurance's employee/employer split.
type SocialInsuranceSplitResponse struct {
	Employee int64 `json:"employee"`
	Employer int64 `json:"employer"`
	Total    int64 `json:"total"`
}

// PayrollCalculationResponse is a computed payslip that has not been stored.
type PayrollCalculationResponse struct {
	PayYear  int `json:"pay_year"`
	PayMonth int `json:"pay_month"`

	TotalEarnings    int64 `json:"total_earnings"`
	NonTaxableAmount int64 `json:"non_taxable_amount"`
	TaxableWage      int64 `json:"taxable_wage"`
	PensionBase      int64 `json:"pension_base"`

	NationalPension     SocialInsuranceSplitResponse `json:"national_pension"`
	HealthInsurance     SocialInsuranceSplitResponse `json:"health_insurance"`
	LongTermCare        SocialInsuranceSplitResponse `json:"long_term_care"`
	EmploymentInsurance SocialInsuranceSplitResponse `json:"employment_insurance"`
	IndustrialAccident  SocialInsuranceSplitResponse `json:"industrial_accident"`

	IncomeTax           int64 `json:"income_tax"`
	LocalIncomeTax      int64 `json:"local_income_tax"`
	OtherDeductionTotal int64 `json:"other_deduction_total"`
	TotalDeductions     int64 `json:"total_deductions"`
	NetPay              int64 `json:"net_pay"`

	EmployerCostTotal int64 `json:"employer_cost_total"`

	Items []PayrollItemResponse `json:"items"`

	// IndustrialAccidentApplied is false when no 산재보험료율 was supplied. The
	// UI must render 미산정 rather than 0원 in that case: the two mean very
	// different things to whoever reconciles against the 근로복지공단 notice.
	IndustrialAccidentApplied bool `json:"industrial_accident_applied"`

	// EmploymentStabilityRate is the 고용안정·직업능력개발사업 rate that was
	// added to the employer's 고용보험료, in hundred-thousandths.
	EmploymentStabilityRate int64 `json:"employment_stability_rate"`

	// RatesSource names the authority behind the rate row that was applied, so
	// an operator can check the figures against the published notice.
	RatesSource string `json:"rates_source"`
}

// splitResponse converts a domain contribution split.
func splitResponse(c domain.SocialInsuranceContribution) SocialInsuranceSplitResponse {
	return SocialInsuranceSplitResponse{
		Employee: c.Employee,
		Employer: c.Employer,
		Total:    c.Total(),
	}
}

// FromPayrollCalculation converts a calculation result to its response shape.
func FromPayrollCalculation(r *domain.PayrollCalculationResult) PayrollCalculationResponse {
	resp := PayrollCalculationResponse{
		PayYear:                   r.PayYear,
		PayMonth:                  r.PayMonth,
		TotalEarnings:             r.TotalEarnings,
		NonTaxableAmount:          r.NonTaxableAmount,
		TaxableWage:               r.TaxableWage,
		PensionBase:               r.SocialInsurance.PensionBase,
		NationalPension:           splitResponse(r.SocialInsurance.NationalPension),
		HealthInsurance:           splitResponse(r.SocialInsurance.HealthInsurance),
		LongTermCare:              splitResponse(r.SocialInsurance.LongTermCare),
		EmploymentInsurance:       splitResponse(r.SocialInsurance.EmploymentInsurance),
		IndustrialAccident:        splitResponse(r.SocialInsurance.IndustrialAccident),
		IncomeTax:                 r.IncomeTax,
		LocalIncomeTax:            r.LocalIncomeTax,
		OtherDeductionTotal:       r.OtherDeductionTotal,
		TotalDeductions:           r.TotalDeductions,
		NetPay:                    r.NetPay,
		EmployerCostTotal:         r.EmployerCost.Total,
		IndustrialAccidentApplied: r.IndustrialAccidentApplied,
		EmploymentStabilityRate:   r.EmploymentStabilityRate,
		RatesSource:               r.RatesSource,
	}
	for _, it := range r.Items {
		resp.Items = append(resp.Items, PayrollItemResponse{
			ItemType:  string(it.ItemType),
			ItemCode:  it.ItemCode,
			ItemName:  it.ItemName,
			Amount:    it.Amount,
			IsTaxable: it.IsTaxable,
			IsFixed:   it.IsFixed,
			SortOrder: it.SortOrder,
		})
	}
	return resp
}
