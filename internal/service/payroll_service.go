package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// PayrollService is the application layer for 급여.
//
// TWO RULES DRIVE THE SHAPE OF THIS FILE.
//
//  1. Nothing is invented. The 4대보험 rates live in internal/domain and a year
//     that has not been verified fails; the 근로소득 간이세액표 is not loaded, so
//     the monthly 소득세 must be supplied by the operator. Both failures are
//     translated into a message that says what is missing, not swallowed.
//
//  2. A payroll header and its lines move together or not at all. Every write
//     that touches more than one table goes through repo.WithTransaction. The
//     voucher AddEntry path used to commit the line before the header and left
//     the two permanently disagreeing; that is not repeated here.
type PayrollService struct {
	repo      repository.PayrollRepository
	insurance repository.InsuranceRepository
}

// NewPayrollService creates a PayrollService.
func NewPayrollService(repo repository.PayrollRepository, insurance repository.InsuranceRepository) *PayrollService {
	return &PayrollService{repo: repo, insurance: insurance}
}

// Service-level errors. They are separate from the domain errors so the handler
// can map "you asked for something that is not allowed right now" to a status
// code without inspecting driver errors.
var (
	// ErrPayrollPeriodLocked is returned when a payroll belongs to a period
	// that has already been approved, paid or closed.
	ErrPayrollPeriodLocked = errors.New("급여기간이 확정되어 개별 급여를 변경할 수 없습니다")

	// ErrPayrollConcurrentChange is returned when a guarded status transition
	// matched no row, which means another request changed the record first.
	ErrPayrollConcurrentChange = errors.New("다른 요청이 먼저 상태를 변경했습니다")

	// ErrPayrollPeriodEmpty is returned when a period is confirmed with no
	// payroll records in it.
	ErrPayrollPeriodEmpty = errors.New("급여기간에 급여 자료가 없습니다")
)

// ---------------------------------------------------------------------------
// Inputs
// ---------------------------------------------------------------------------

// CreatePayrollPeriodInput describes a new monthly payroll run.
type CreatePayrollPeriodInput struct {
	PayYear     int
	PayMonth    int
	PeriodName  string
	PeriodStart time.Time
	PeriodEnd   time.Time
	PaymentDate *time.Time
}

// PayrollEarningsInput is the gross pay side of a create/update request.
type PayrollEarningsInput struct {
	Earnings      domain.PayrollEarnings
	WorkDays      int
	OvertimeHours float64
	NightHours    float64
	HolidayHours  float64
	Notes         string
	BankCode      string
}

// CreatePayrollInput describes a new payroll record.
type CreatePayrollInput struct {
	EmployeeID      uuid.UUID
	PayYear         int
	PayMonth        int
	PayrollPeriodID *uuid.UUID
	PaymentDate     *time.Time
	PayrollEarningsInput
}

// CalculatePayrollInput carries the figures the calculation cannot derive.
type CalculatePayrollInput struct {
	// IncomeTax 근로소득 원천징수세액. Required unless the payroll was already
	// calculated once, in which case the stored figure is reused.
	IncomeTax *int64
	// LocalIncomeTax overrides the derived 지방소득세.
	LocalIncomeTax *int64

	OtherDeductions []domain.PayrollNamedAmount

	WorkplaceSize          domain.WorkplaceSize
	IndustrialAccidentRate *int64

	// ContributionDate selects the rate year and the 국민연금 기준소득월액
	// window. See resolveContributionDate for the default.
	ContributionDate *time.Time
}

// CalculatePeriodInput drives a whole-period recalculation.
type CalculatePeriodInput struct {
	WorkplaceSize          domain.WorkplaceSize
	IndustrialAccidentRate *int64
	ContributionDate       *time.Time

	// Employees carries per-employee figures, keyed by employee id. An employee
	// with no entry here keeps the 소득세 already stored against the payroll,
	// and is skipped if there is none.
	Employees map[uuid.UUID]CalculatePayrollInput
}

// PayrollSkipped explains why one payroll was left out of a batch.
type PayrollSkipped struct {
	PayrollID  uuid.UUID `json:"payroll_id"`
	EmployeeID uuid.UUID `json:"employee_id"`
	Reason     string    `json:"reason"`
}

// CalculatePeriodResult reports what a batch calculation did.
type CalculatePeriodResult struct {
	Period    *domain.PayrollPeriod `json:"period"`
	Processed int                   `json:"processed"`
	Skipped   []PayrollSkipped      `json:"skipped"`
}

// ---------------------------------------------------------------------------
// Periods
// ---------------------------------------------------------------------------

// CreatePeriod opens a monthly payroll run.
func (s *PayrollService) CreatePeriod(ctx context.Context, companyID uuid.UUID, in *CreatePayrollPeriodInput) (*domain.PayrollPeriod, error) {
	if in.PayMonth < 1 || in.PayMonth > 12 {
		return nil, domain.ErrPayrollMonthInvalid
	}
	if in.PeriodStart.After(in.PeriodEnd) {
		return nil, domain.ErrPayrollPeriodDates
	}

	// payroll_periods is UNIQUE(company_id, pay_year, pay_month). Checking
	// first turns the constraint violation into a message an operator can act
	// on; the constraint itself remains the real guarantee.
	existing, err := s.repo.GetPeriodByMonth(ctx, companyID, in.PayYear, in.PayMonth)
	if err != nil && !errors.Is(err, domain.ErrPayrollPeriodNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrPayrollPeriodExists
	}

	period := &domain.PayrollPeriod{
		CompanyID:   companyID,
		PayYear:     in.PayYear,
		PayMonth:    in.PayMonth,
		PeriodName:  in.PeriodName,
		PeriodStart: in.PeriodStart,
		PeriodEnd:   in.PeriodEnd,
		PaymentDate: in.PaymentDate,
		Status:      domain.PayrollPeriodStatusDraft,
	}
	if err := s.repo.CreatePeriod(ctx, period); err != nil {
		return nil, err
	}
	return period, nil
}

// GetPeriod loads one payroll period.
func (s *PayrollService) GetPeriod(ctx context.Context, companyID, id uuid.UUID) (*domain.PayrollPeriod, error) {
	return s.repo.GetPeriodByID(ctx, companyID, id)
}

// ListPeriods returns a page of payroll periods.
func (s *PayrollService) ListPeriods(ctx context.Context, filter *repository.PayrollPeriodFilter) ([]*domain.PayrollPeriod, int64, error) {
	return s.repo.ListPeriods(ctx, filter)
}

// CalculatePeriod recalculates every payroll in a period.
//
// The whole batch is one transaction. A rate that is missing for the year fails
// the entire run rather than leaving half the company calculated at this year's
// rates and half at nothing.
func (s *PayrollService) CalculatePeriod(ctx context.Context, companyID, periodID uuid.UUID,
	in *CalculatePeriodInput, userID *uuid.UUID) (*CalculatePeriodResult, error) {

	period, err := s.repo.GetPeriodByID(ctx, companyID, periodID)
	if err != nil {
		return nil, err
	}
	if !period.CanCalculate() {
		return nil, fmt.Errorf("%w: status=%s", domain.ErrPayrollPeriodNotEditable, period.Status)
	}

	payrolls, err := s.repo.ListPayrollsByPeriod(ctx, companyID, periodID)
	if err != nil {
		return nil, err
	}
	if len(payrolls) == 0 {
		return nil, ErrPayrollPeriodEmpty
	}

	result := &CalculatePeriodResult{}
	fromStatus := period.Status

	err = s.repo.WithTransaction(ctx, func(tx repository.PayrollTx) error {
		for _, payroll := range payrolls {
			if !payroll.CanBeModified() {
				result.Skipped = append(result.Skipped, PayrollSkipped{
					PayrollID:  payroll.ID,
					EmployeeID: payroll.EmployeeID,
					Reason:     fmt.Sprintf("상태가 %s 이어서 재계산 대상이 아닙니다", payroll.Status),
				})
				continue
			}

			perEmployee := CalculatePayrollInput{
				WorkplaceSize:          in.WorkplaceSize,
				IndustrialAccidentRate: in.IndustrialAccidentRate,
				ContributionDate:       in.ContributionDate,
			}
			if override, ok := in.Employees[payroll.EmployeeID]; ok {
				perEmployee = override
				if perEmployee.WorkplaceSize == "" {
					perEmployee.WorkplaceSize = in.WorkplaceSize
				}
				if perEmployee.IndustrialAccidentRate == nil {
					perEmployee.IndustrialAccidentRate = in.IndustrialAccidentRate
				}
				if perEmployee.ContributionDate == nil {
					perEmployee.ContributionDate = in.ContributionDate
				}
			}

			// 소득세는 지어내지 않는다. 이전에 계산된 적이 있으면 그때
			// 입력된 값을 그대로 쓰고, 없으면 이 사람만 건너뛴다.
			if perEmployee.IncomeTax == nil {
				if payroll.Status != domain.PayrollStatusCalculated {
					result.Skipped = append(result.Skipped, PayrollSkipped{
						PayrollID:  payroll.ID,
						EmployeeID: payroll.EmployeeID,
						Reason:     "소득세(근로소득 간이세액)가 입력되지 않았습니다",
					})
					continue
				}
				stored := payroll.IncomeTax
				perEmployee.IncomeTax = &stored
			}

			if _, err := s.calculateOne(ctx, tx, companyID, payroll, &perEmployee); err != nil {
				return err
			}
			result.Processed++
		}

		if result.Processed == 0 {
			return ErrPayrollPeriodEmpty
		}

		// The period header is written from what the rows actually say, read
		// back inside this same transaction.
		totals, err := tx.Payroll.SummarizePeriod(ctx, companyID, periodID)
		if err != nil {
			return err
		}

		now := time.Now()
		rows, err := tx.Payroll.TransitionPeriod(ctx, companyID, periodID,
			fromStatus, domain.PayrollPeriodStatusCalculated, map[string]interface{}{
				"calculated_at":    now,
				"calculated_by":    userID,
				"total_employees":  totals.TotalEmployees,
				"total_earnings":   totals.TotalEarnings,
				"total_deductions": totals.TotalDeductions,
				"total_net_pay":    totals.TotalNetPay,
			})
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrPayrollConcurrentChange
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.GetPeriodByID(ctx, companyID, periodID)
	if err != nil {
		return nil, err
	}
	result.Period = updated
	return result, nil
}

// ApprovePeriod confirms a calculated payroll run.
//
// This is the irreversible step - it is what the bank transfer file and the
// 원천징수이행상황신고서 are built from - so it re-reconciles every payroll
// against its stored line items before it commits. A header that disagrees with
// its rows fails the whole approval and names the employee.
func (s *PayrollService) ApprovePeriod(ctx context.Context, companyID, periodID uuid.UUID, userID *uuid.UUID) (*domain.PayrollPeriod, error) {
	period, err := s.repo.GetPeriodByID(ctx, companyID, periodID)
	if err != nil {
		return nil, err
	}
	if !period.CanApprove() {
		return nil, fmt.Errorf("%w: status=%s", domain.ErrPayrollPeriodNotEditable, period.Status)
	}

	err = s.repo.WithTransaction(ctx, func(tx repository.PayrollTx) error {
		payrolls, err := tx.Payroll.ListPayrollsByPeriod(ctx, companyID, periodID)
		if err != nil {
			return err
		}
		if len(payrolls) == 0 {
			return ErrPayrollPeriodEmpty
		}

		// One GROUP BY for the whole period rather than one query per employee.
		itemTotals, err := tx.Payroll.SumItemsByPeriod(ctx, companyID, periodID)
		if err != nil {
			return err
		}
		byPayroll := make(map[uuid.UUID]repository.PayrollItemTotals, len(itemTotals))
		for _, t := range itemTotals {
			byPayroll[t.PayrollID] = t
		}

		approvable := 0
		for _, p := range payrolls {
			if p.Status == domain.PayrollStatusCancelled {
				continue
			}
			if !p.CanBeApproved() {
				return fmt.Errorf("%w: 사원 %s (상태 %s)", domain.ErrPayrollNotCalculated, p.EmployeeID, p.Status)
			}

			totals := byPayroll[p.ID]
			if totals.EarningTotal != p.TotalEarnings ||
				totals.DeductionTotal != p.TotalDeductions ||
				p.NetPay != p.TotalEarnings-p.TotalDeductions {
				return fmt.Errorf("%w: 사원 %s (지급 %d/%d, 공제 %d/%d, 실지급 %d)",
					domain.ErrPayrollTotalsMismatch, p.EmployeeID,
					p.TotalEarnings, totals.EarningTotal,
					p.TotalDeductions, totals.DeductionTotal,
					p.NetPay)
			}
			approvable++
		}
		if approvable == 0 {
			return ErrPayrollPeriodEmpty
		}

		if _, err := tx.Payroll.TransitionPayrollsByPeriod(ctx, companyID, periodID,
			domain.PayrollStatusCalculated, domain.PayrollStatusApproved, nil); err != nil {
			return err
		}

		totals, err := tx.Payroll.SummarizePeriod(ctx, companyID, periodID)
		if err != nil {
			return err
		}

		now := time.Now()
		rows, err := tx.Payroll.TransitionPeriod(ctx, companyID, periodID,
			domain.PayrollPeriodStatusCalculated, domain.PayrollPeriodStatusApproved,
			map[string]interface{}{
				"approved_at":      now,
				"approved_by":      userID,
				"total_employees":  totals.TotalEmployees,
				"total_earnings":   totals.TotalEarnings,
				"total_deductions": totals.TotalDeductions,
				"total_net_pay":    totals.TotalNetPay,
			})
		if err != nil {
			return err
		}
		// The status is part of the WHERE clause, so a second concurrent
		// approval matches nothing and lands here instead of confirming the
		// run twice.
		if rows == 0 {
			return ErrPayrollConcurrentChange
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.repo.GetPeriodByID(ctx, companyID, periodID)
}

// MarkPeriodPaid records that an approved payroll run has been paid out.
func (s *PayrollService) MarkPeriodPaid(ctx context.Context, companyID, periodID uuid.UUID, paidOn *time.Time) (*domain.PayrollPeriod, error) {
	period, err := s.repo.GetPeriodByID(ctx, companyID, periodID)
	if err != nil {
		return nil, err
	}
	if !period.CanPay() {
		return nil, fmt.Errorf("%w: status=%s", domain.ErrPayrollPeriodNotEditable, period.Status)
	}

	now := time.Now()
	paymentDate := now
	if paidOn != nil {
		paymentDate = *paidOn
	}

	err = s.repo.WithTransaction(ctx, func(tx repository.PayrollTx) error {
		if _, err := tx.Payroll.TransitionPayrollsByPeriod(ctx, companyID, periodID,
			domain.PayrollStatusApproved, domain.PayrollStatusPaid, map[string]interface{}{
				"paid_at":      now,
				"payment_date": paymentDate,
			}); err != nil {
			return err
		}

		rows, err := tx.Payroll.TransitionPeriod(ctx, companyID, periodID,
			domain.PayrollPeriodStatusApproved, domain.PayrollPeriodStatusPaid,
			map[string]interface{}{
				"paid_at":      now,
				"payment_date": paymentDate,
			})
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrPayrollConcurrentChange
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.repo.GetPeriodByID(ctx, companyID, periodID)
}

// ClosePeriod locks a paid payroll run against further change.
func (s *PayrollService) ClosePeriod(ctx context.Context, companyID, periodID uuid.UUID) (*domain.PayrollPeriod, error) {
	period, err := s.repo.GetPeriodByID(ctx, companyID, periodID)
	if err != nil {
		return nil, err
	}
	if !period.CanClose() {
		return nil, fmt.Errorf("%w: status=%s", domain.ErrPayrollPeriodNotEditable, period.Status)
	}

	rows, err := s.repo.TransitionPeriod(ctx, companyID, periodID,
		domain.PayrollPeriodStatusPaid, domain.PayrollPeriodStatusClosed, nil)
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrPayrollConcurrentChange
	}
	return s.repo.GetPeriodByID(ctx, companyID, periodID)
}

// ---------------------------------------------------------------------------
// Payroll records
// ---------------------------------------------------------------------------

// CreatePayroll opens a draft payroll record for one employee and month.
//
// The earning lines are written in the same transaction as the header, so even
// a draft reconciles: total_earnings equals the sum of its rows and net_pay
// equals gross while nothing has been deducted yet.
func (s *PayrollService) CreatePayroll(ctx context.Context, companyID uuid.UUID, in *CreatePayrollInput) (*domain.Payroll, error) {
	if in.PayMonth < 1 || in.PayMonth > 12 {
		return nil, domain.ErrPayrollMonthInvalid
	}
	if in.EmployeeID == uuid.Nil {
		return nil, fmt.Errorf("%w: employee_id", domain.ErrPayrollEmptyItemName)
	}

	existing, err := s.repo.GetPayrollByEmployeeMonth(ctx, companyID, in.EmployeeID, in.PayYear, in.PayMonth)
	if err != nil && !errors.Is(err, domain.ErrPayrollNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrPayrollExists
	}

	if in.PayrollPeriodID != nil {
		if err := s.assertPeriodOpen(ctx, companyID, *in.PayrollPeriodID); err != nil {
			return nil, err
		}
	}

	gross, lines, err := draftEarningLines(in.Earnings)
	if err != nil {
		return nil, err
	}

	payroll := &domain.Payroll{
		CompanyID:       companyID,
		EmployeeID:      in.EmployeeID,
		PayrollPeriodID: in.PayrollPeriodID,
		PayYear:         in.PayYear,
		PayMonth:        in.PayMonth,
		Status:          domain.PayrollStatusDraft,
		BaseSalary:      in.Earnings.BaseSalary,
		OvertimePay:     in.Earnings.OvertimePay,
		NightPay:        in.Earnings.NightPay,
		HolidayPay:      in.Earnings.HolidayPay,
		Bonus:           in.Earnings.Bonus,
		TotalEarnings:   gross,
		TotalDeductions: 0,
		NetPay:          gross,
		PaymentDate:     in.PaymentDate,
		WorkDays:        in.WorkDays,
		OvertimeHours:   in.OvertimeHours,
		NightHours:      in.NightHours,
		HolidayHours:    in.HolidayHours,
		Notes:           in.Notes,
		BankCode:        in.BankCode,
	}
	payroll.Allowances = namedLinesToMap(in.Earnings.Allowances)
	payroll.OtherEarnings = namedLinesToMap(in.Earnings.OtherEarnings)
	payroll.OtherDeductions = domain.PayrollAmountMap{}

	err = s.repo.WithTransaction(ctx, func(tx repository.PayrollTx) error {
		if err := tx.Payroll.CreatePayroll(ctx, payroll); err != nil {
			return err
		}
		return tx.Payroll.ReplaceItems(ctx, companyID, payroll.ID,
			domain.ToPayrollItems(companyID, payroll.ID, lines))
	})
	if err != nil {
		return nil, err
	}

	payroll.Items = domain.ToPayrollItems(companyID, payroll.ID, lines)
	return payroll, nil
}

// UpdatePayroll replaces the earnings of a draft or calculated payroll.
//
// Replacing the earnings resets the record to draft: the deductions that were
// calculated from the old gross no longer apply to the new one, and leaving
// them in place is how a payslip ends up withholding 4대보험 on a salary that
// was never paid.
func (s *PayrollService) UpdatePayroll(ctx context.Context, companyID, id uuid.UUID, in *PayrollEarningsInput) (*domain.Payroll, error) {
	payroll, err := s.repo.GetPayrollByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if !payroll.CanBeModified() {
		return nil, fmt.Errorf("%w: status=%s", domain.ErrPayrollNotEditable, payroll.Status)
	}
	if payroll.PayrollPeriodID != nil {
		if err := s.assertPeriodOpen(ctx, companyID, *payroll.PayrollPeriodID); err != nil {
			return nil, err
		}
	}

	gross, lines, err := draftEarningLines(in.Earnings)
	if err != nil {
		return nil, err
	}

	payroll.BaseSalary = in.Earnings.BaseSalary
	payroll.OvertimePay = in.Earnings.OvertimePay
	payroll.NightPay = in.Earnings.NightPay
	payroll.HolidayPay = in.Earnings.HolidayPay
	payroll.Bonus = in.Earnings.Bonus
	payroll.Allowances = namedLinesToMap(in.Earnings.Allowances)
	payroll.OtherEarnings = namedLinesToMap(in.Earnings.OtherEarnings)
	payroll.OtherDeductions = domain.PayrollAmountMap{}
	payroll.TotalEarnings = gross
	payroll.WorkDays = in.WorkDays
	payroll.OvertimeHours = in.OvertimeHours
	payroll.NightHours = in.NightHours
	payroll.HolidayHours = in.HolidayHours
	payroll.Notes = in.Notes
	payroll.BankCode = in.BankCode

	// Back to draft, with every deduction cleared.
	payroll.Status = domain.PayrollStatusDraft
	payroll.IncomeTax = 0
	payroll.LocalIncomeTax = 0
	payroll.NPSEmployee = 0
	payroll.NHISEmployee = 0
	payroll.NHISLTCEmployee = 0
	payroll.EIEmployee = 0
	payroll.NPSEmployer = 0
	payroll.NHISEmployer = 0
	payroll.NHISLTCEmployer = 0
	payroll.EIEmployer = 0
	payroll.WCIEmployer = 0
	payroll.TotalEmployerCost = 0
	payroll.TotalDeductions = 0
	payroll.NetPay = gross

	err = s.repo.WithTransaction(ctx, func(tx repository.PayrollTx) error {
		if err := tx.Payroll.UpdatePayroll(ctx, payroll); err != nil {
			return err
		}
		return tx.Payroll.ReplaceItems(ctx, companyID, payroll.ID,
			domain.ToPayrollItems(companyID, payroll.ID, lines))
	})
	if err != nil {
		return nil, err
	}

	payroll.Items = domain.ToPayrollItems(companyID, payroll.ID, lines)
	return payroll, nil
}

// GetPayroll loads one payroll record together with its line items.
func (s *PayrollService) GetPayroll(ctx context.Context, companyID, id uuid.UUID) (*domain.Payroll, error) {
	payroll, err := s.repo.GetPayrollByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.ListItems(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	payroll.Items = items
	return payroll, nil
}

// ListPayrolls returns a page of payroll rows.
func (s *PayrollService) ListPayrolls(ctx context.Context, filter *repository.PayrollFilter) ([]*repository.PayrollListRow, int64, error) {
	return s.repo.ListPayrolls(ctx, filter)
}

// DeletePayroll removes a payroll record that has not been confirmed.
func (s *PayrollService) DeletePayroll(ctx context.Context, companyID, id uuid.UUID) error {
	payroll, err := s.repo.GetPayrollByID(ctx, companyID, id)
	if err != nil {
		return err
	}
	if !payroll.CanBeModified() {
		return fmt.Errorf("%w: status=%s", domain.ErrPayrollNotEditable, payroll.Status)
	}
	if payroll.PayrollPeriodID != nil {
		if err := s.assertPeriodOpen(ctx, companyID, *payroll.PayrollPeriodID); err != nil {
			return err
		}
	}
	return s.repo.DeletePayroll(ctx, companyID, id)
}

// CalculatePayroll computes the deductions for one payroll record.
func (s *PayrollService) CalculatePayroll(ctx context.Context, companyID, id uuid.UUID,
	in *CalculatePayrollInput) (*domain.Payroll, error) {

	payroll, err := s.repo.GetPayrollByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if !payroll.CanBeModified() {
		return nil, fmt.Errorf("%w: status=%s", domain.ErrPayrollNotEditable, payroll.Status)
	}
	if payroll.PayrollPeriodID != nil {
		if err := s.assertPeriodOpen(ctx, companyID, *payroll.PayrollPeriodID); err != nil {
			return nil, err
		}
	}

	// Reuse the stored 소득세 when the caller does not supply one and the
	// record was already calculated once.
	if in.IncomeTax == nil && payroll.Status == domain.PayrollStatusCalculated {
		stored := payroll.IncomeTax
		in.IncomeTax = &stored
	}

	var calculated *domain.Payroll
	err = s.repo.WithTransaction(ctx, func(tx repository.PayrollTx) error {
		p, err := s.calculateOne(ctx, tx, companyID, payroll, in)
		if err != nil {
			return err
		}
		calculated = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return calculated, nil
}

// PreviewCalculation computes a payroll without touching the database.
//
// It is the endpoint the UI uses to show what a salary would net before the
// record is created, so it must never write - including the 4대보험 register.
func (s *PayrollService) PreviewCalculation(ctx context.Context, in *domain.PayrollCalculationInput) (*domain.PayrollCalculationResult, error) {
	result, err := domain.CalculatePayroll(*in)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ---------------------------------------------------------------------------
// Internals
// ---------------------------------------------------------------------------

// calculateOne computes and persists one payroll inside an open transaction.
//
// The header, the item rows and the 4대보험 monthly register are three tables
// and they are written here, together. Nothing in this function commits.
func (s *PayrollService) calculateOne(ctx context.Context, tx repository.PayrollTx,
	companyID uuid.UUID, payroll *domain.Payroll, in *CalculatePayrollInput) (*domain.Payroll, error) {

	// The earnings are reconstructed from the stored earning rows, which carry
	// the per-line 과세/비과세 flag that the JSONB summary columns do not.
	items, err := tx.Payroll.ListItems(ctx, companyID, payroll.ID)
	if err != nil {
		return nil, err
	}
	earnings := earningsFromItems(items)

	// A qualification record is optional; its absence means a regular employee
	// covered by all four insurances.
	qualification := domain.AllQualified()
	record, err := tx.Insurance.GetEmployeeInsurance(ctx, companyID, payroll.EmployeeID)
	if err != nil && !errors.Is(err, domain.ErrEmployeeInsuranceNotFound) {
		return nil, err
	}
	if record != nil {
		qualification = record.Qualification()
	}

	calcInput := domain.PayrollCalculationInput{
		PayYear:          payroll.PayYear,
		PayMonth:         payroll.PayMonth,
		ContributionDate: resolveContributionDate(in.ContributionDate, payroll),
		Earnings:         earnings,
		Deductions: domain.PayrollDeductionInput{
			IncomeTax:      in.IncomeTax,
			LocalIncomeTax: in.LocalIncomeTax,
			Others:         in.OtherDeductions,
		},
		Qualification:          qualification,
		WorkplaceSize:          in.WorkplaceSize,
		IndustrialAccidentRate: in.IndustrialAccidentRate,
	}

	result, err := domain.CalculatePayroll(calcInput)
	if err != nil {
		return nil, err
	}

	payroll.ApplyCalculation(result)
	rows := domain.ToPayrollItems(companyID, payroll.ID, result.Items)

	// Belt and braces: the domain already self-checked, but this is the last
	// point before the rows reach the database.
	if err := domain.ValidatePayrollTotals(payroll, rows); err != nil {
		return nil, err
	}

	if err := tx.Payroll.UpdatePayroll(ctx, payroll); err != nil {
		return nil, err
	}
	if err := tx.Payroll.ReplaceItems(ctx, companyID, payroll.ID, rows); err != nil {
		return nil, err
	}

	contribution := &domain.InsuranceMonthlyContribution{
		CompanyID:         companyID,
		EmployeeID:        payroll.EmployeeID,
		ContributionYear:  payroll.PayYear,
		ContributionMonth: payroll.PayMonth,
		PayrollID:         &payroll.ID,
	}
	contribution.ApplyPayrollContribution(result)
	if err := tx.Insurance.UpsertContribution(ctx, contribution); err != nil {
		return nil, err
	}

	payroll.Items = rows
	return payroll, nil
}

// assertPeriodOpen refuses a change to a payroll whose period is confirmed.
//
// Without this, editing one employee's record after the run was approved would
// silently make the period header disagree with the sum of its payrolls.
func (s *PayrollService) assertPeriodOpen(ctx context.Context, companyID, periodID uuid.UUID) error {
	period, err := s.repo.GetPeriodByID(ctx, companyID, periodID)
	if err != nil {
		return err
	}
	if !period.CanCalculate() {
		return fmt.Errorf("%w: status=%s", ErrPayrollPeriodLocked, period.Status)
	}
	return nil
}

// resolveContributionDate picks the date that selects the rate year and the
// 국민연금 기준소득월액 window.
//
// Preference order: the caller's explicit date, then the payroll's payment
// date, then the first day of the pay month. The pension ceiling changes on
// 1 July, so a July or August payroll paid in the following month can land in a
// different window depending on which of these applies - the caller should pass
// the real payment date when it matters.
func resolveContributionDate(explicit *time.Time, payroll *domain.Payroll) time.Time {
	if explicit != nil {
		return *explicit
	}
	if payroll.PaymentDate != nil {
		return *payroll.PaymentDate
	}
	return time.Date(payroll.PayYear, time.Month(payroll.PayMonth), 1, 0, 0, 0, 0, time.UTC)
}

// draftEarningLines totals the earnings of a draft and turns them into rows.
//
// It reuses the domain validation by running a calculation whose statutory part
// is switched off: the point is to reject a negative or unnamed line here,
// before it is stored, with the same rules the real calculation applies.
func draftEarningLines(e domain.PayrollEarnings) (int64, []domain.PayrollItemLine, error) {
	lines := make([]domain.PayrollItemLine, 0, 8)
	order := 0
	var gross int64

	add := func(code, name string, amount int64, taxable, fixed bool) error {
		if amount < 0 {
			return fmt.Errorf("%s: %w", code, domain.ErrNegativeAmount)
		}
		gross += amount
		if amount == 0 {
			return nil
		}
		order++
		lines = append(lines, domain.PayrollItemLine{
			ItemType:  domain.PayrollItemEarning,
			ItemCode:  code,
			ItemName:  name,
			Amount:    amount,
			IsTaxable: taxable,
			IsFixed:   fixed,
			SortOrder: order,
		})
		return nil
	}

	if err := add(domain.PayrollItemCodeBaseSalary, "기본급", e.BaseSalary, true, true); err != nil {
		return 0, nil, err
	}
	if err := add(domain.PayrollItemCodeOvertimePay, "연장근로수당", e.OvertimePay, true, false); err != nil {
		return 0, nil, err
	}
	if err := add(domain.PayrollItemCodeNightPay, "야간근로수당", e.NightPay, true, false); err != nil {
		return 0, nil, err
	}
	if err := add(domain.PayrollItemCodeHolidayPay, "휴일근로수당", e.HolidayPay, true, false); err != nil {
		return 0, nil, err
	}
	if err := add(domain.PayrollItemCodeBonus, "상여금", e.Bonus, true, false); err != nil {
		return 0, nil, err
	}

	for _, a := range e.Allowances {
		if a.Code == "" || a.Name == "" {
			return 0, nil, domain.ErrPayrollEmptyItemName
		}
		if err := add(a.Code, a.Name, a.Amount, !a.NonTaxable, true); err != nil {
			return 0, nil, err
		}
	}
	for _, a := range e.OtherEarnings {
		if a.Code == "" || a.Name == "" {
			return 0, nil, domain.ErrPayrollEmptyItemName
		}
		if err := add(a.Code, a.Name, a.Amount, !a.NonTaxable, false); err != nil {
			return 0, nil, err
		}
	}

	return gross, lines, nil
}

// earningsFromItems rebuilds the calculation input from the stored earning rows.
//
// This is the inverse of buildPayrollItems: the five statutory codes map back
// to their own fields, is_fixed separates 수당 from 기타지급, and is_taxable
// carries the 비과세 flag that the JSONB summary column cannot express.
func earningsFromItems(items []domain.PayrollItem) domain.PayrollEarnings {
	var e domain.PayrollEarnings
	for _, it := range items {
		if it.ItemType != domain.PayrollItemEarning {
			continue
		}
		switch it.ItemCode {
		case domain.PayrollItemCodeBaseSalary:
			e.BaseSalary = it.Amount
		case domain.PayrollItemCodeOvertimePay:
			e.OvertimePay = it.Amount
		case domain.PayrollItemCodeNightPay:
			e.NightPay = it.Amount
		case domain.PayrollItemCodeHolidayPay:
			e.HolidayPay = it.Amount
		case domain.PayrollItemCodeBonus:
			e.Bonus = it.Amount
		default:
			named := domain.PayrollNamedAmount{
				Code:       it.ItemCode,
				Name:       it.ItemName,
				Amount:     it.Amount,
				NonTaxable: !it.IsTaxable,
			}
			if it.IsFixed {
				e.Allowances = append(e.Allowances, named)
			} else {
				e.OtherEarnings = append(e.OtherEarnings, named)
			}
		}
	}
	return e
}

// namedLinesToMap collapses named amounts into the JSONB summary column.
func namedLinesToMap(lines []domain.PayrollNamedAmount) domain.PayrollAmountMap {
	out := domain.PayrollAmountMap{}
	for _, l := range lines {
		out[l.Code] += l.Amount
	}
	return out
}
