package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// PayrollPeriodFilter selects payroll periods for a list query.
type PayrollPeriodFilter struct {
	CompanyID uuid.UUID
	PayYear   *int
	PayMonth  *int
	Status    *domain.PayrollPeriodStatus
	Page      int
	PageSize  int
}

// PayrollFilter selects payroll records for a list query.
type PayrollFilter struct {
	CompanyID  uuid.UUID
	PeriodID   *uuid.UUID
	EmployeeID *uuid.UUID
	PayYear    *int
	PayMonth   *int
	Status     *domain.PayrollStatus
	// Search matches the employee number or name.
	Search   string
	Page     int
	PageSize int
}

// PayrollListRow is the projection returned by a payroll list query.
//
// It is deliberately NOT the full domain.Payroll. A payroll register is read by
// anyone who can open the HR screen, and the row-level detail (each statutory
// deduction, the employer's cost) belongs on the detail view, behind a request
// for one specific record. The bank account never appears in either.
type PayrollListRow struct {
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
	PaymentDate     *time.Time `json:"payment_date,omitempty"`
}

// PayrollItemTotals is one payroll's line totals, computed by the database.
//
// It exists so that reconciling a whole period costs one GROUP BY instead of
// one query per employee.
type PayrollItemTotals struct {
	PayrollID      uuid.UUID
	EarningTotal   int64
	DeductionTotal int64
}

// PayrollPeriodTotals is the aggregate of the payrolls in one period. The
// period header is written from this, never from figures accumulated in Go, so
// the header always equals what the rows actually say.
type PayrollPeriodTotals struct {
	TotalEmployees  int
	TotalEarnings   int64
	TotalDeductions int64
	TotalNetPay     int64
}

// PayrollTx is the set of repositories bound to one payroll transaction.
//
// Calculating a payroll writes the payroll header, replaces its item rows and
// updates the 4대보험 monthly register. Those are three tables and they must
// land together: a header committed without its rows is a payslip that
// reconciles against nothing, which is exactly the defect the voucher AddEntry
// path used to have.
type PayrollTx struct {
	Payroll   PayrollRepository
	Insurance InsuranceRepository
}

// PayrollRepository is data access for payroll periods, records and items.
//
// EVERY method takes companyID and every query filters on it. The RLS policies
// in db/migrations/000010 are the second line of defence, not the first: they
// only apply when the tenant session variable was set, and a repository that
// forgets the predicate is one connection-pool bug away from cross-tenant
// reads.
type PayrollRepository interface {
	// Periods
	CreatePeriod(ctx context.Context, period *domain.PayrollPeriod) error
	GetPeriodByID(ctx context.Context, companyID, id uuid.UUID) (*domain.PayrollPeriod, error)
	GetPeriodByMonth(ctx context.Context, companyID uuid.UUID, year, month int) (*domain.PayrollPeriod, error)
	ListPeriods(ctx context.Context, filter *PayrollPeriodFilter) ([]*domain.PayrollPeriod, int64, error)
	UpdatePeriod(ctx context.Context, period *domain.PayrollPeriod) error

	// TransitionPeriod moves a period from one status to another and applies
	// updates in the same statement.
	//
	// The current status is part of the WHERE clause, so two concurrent
	// approvals cannot both succeed: the second one matches no row and gets
	// rowsAffected 0. Checking the status in Go and then updating leaves a
	// window between the two in which both callers see "calculated".
	TransitionPeriod(ctx context.Context, companyID, id uuid.UUID,
		from, to domain.PayrollPeriodStatus, updates map[string]interface{}) (int64, error)

	// Payrolls
	CreatePayroll(ctx context.Context, payroll *domain.Payroll) error
	GetPayrollByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Payroll, error)
	GetPayrollByEmployeeMonth(ctx context.Context, companyID, employeeID uuid.UUID, year, month int) (*domain.Payroll, error)
	ListPayrolls(ctx context.Context, filter *PayrollFilter) ([]*PayrollListRow, int64, error)
	ListPayrollsByPeriod(ctx context.Context, companyID, periodID uuid.UUID) ([]*domain.Payroll, error)
	UpdatePayroll(ctx context.Context, payroll *domain.Payroll) error
	DeletePayroll(ctx context.Context, companyID, id uuid.UUID) error

	// TransitionPayrollsByPeriod moves every payroll of a period from one
	// status to another, guarded on the source status for the same reason
	// TransitionPeriod is.
	TransitionPayrollsByPeriod(ctx context.Context, companyID, periodID uuid.UUID,
		from, to domain.PayrollStatus, updates map[string]interface{}) (int64, error)

	// Items
	ReplaceItems(ctx context.Context, companyID, payrollID uuid.UUID, items []domain.PayrollItem) error
	ListItems(ctx context.Context, companyID, payrollID uuid.UUID) ([]domain.PayrollItem, error)
	SumItemsByPeriod(ctx context.Context, companyID, periodID uuid.UUID) ([]PayrollItemTotals, error)

	// Aggregates
	SummarizePeriod(ctx context.Context, companyID, periodID uuid.UUID) (*PayrollPeriodTotals, error)

	// WithTransaction runs fn against repositories that share one transaction.
	WithTransaction(ctx context.Context, fn func(tx PayrollTx) error) error
}
