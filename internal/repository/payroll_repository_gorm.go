package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// payrollRepositoryGorm implements PayrollRepository with GORM.
type payrollRepositoryGorm struct {
	db *gorm.DB
}

// NewPayrollRepository creates a GORM-backed PayrollRepository.
func NewPayrollRepository(db *gorm.DB) PayrollRepository {
	return &payrollRepositoryGorm{db: db}
}

// ---------------------------------------------------------------------------
// Periods
// ---------------------------------------------------------------------------

// CreatePeriod inserts a payroll period.
func (r *payrollRepositoryGorm) CreatePeriod(ctx context.Context, period *domain.PayrollPeriod) error {
	return r.db.WithContext(ctx).Create(period).Error
}

// GetPeriodByID loads one payroll period belonging to the company.
func (r *payrollRepositoryGorm) GetPeriodByID(ctx context.Context, companyID, id uuid.UUID) (*domain.PayrollPeriod, error) {
	var period domain.PayrollPeriod
	err := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		First(&period).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPayrollPeriodNotFound
		}
		return nil, err
	}
	return &period, nil
}

// GetPeriodByMonth loads the period for one pay month, if it exists.
func (r *payrollRepositoryGorm) GetPeriodByMonth(ctx context.Context, companyID uuid.UUID, year, month int) (*domain.PayrollPeriod, error) {
	var period domain.PayrollPeriod
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND pay_year = ? AND pay_month = ?", companyID, year, month).
		First(&period).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPayrollPeriodNotFound
		}
		return nil, err
	}
	return &period, nil
}

// ListPeriods returns a page of payroll periods.
func (r *payrollRepositoryGorm) ListPeriods(ctx context.Context, filter *PayrollPeriodFilter) ([]*domain.PayrollPeriod, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.PayrollPeriod{}).
		Where("company_id = ?", filter.CompanyID)

	if filter.PayYear != nil {
		query = query.Where("pay_year = ?", *filter.PayYear)
	}
	if filter.PayMonth != nil {
		query = query.Where("pay_month = ?", *filter.PayMonth)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var periods []*domain.PayrollPeriod
	err := query.
		Order("pay_year DESC, pay_month DESC").
		Offset(offsetOf(filter.Page, filter.PageSize)).
		Limit(filter.PageSize).
		Find(&periods).Error
	if err != nil {
		return nil, 0, err
	}
	return periods, total, nil
}

// UpdatePeriod persists a modified payroll period.
func (r *payrollRepositoryGorm) UpdatePeriod(ctx context.Context, period *domain.PayrollPeriod) error {
	return r.db.WithContext(ctx).
		Model(&domain.PayrollPeriod{}).
		Where("id = ? AND company_id = ?", period.ID, period.CompanyID).
		Save(period).Error
}

// TransitionPeriod applies a guarded status change. See the interface comment.
func (r *payrollRepositoryGorm) TransitionPeriod(ctx context.Context, companyID, id uuid.UUID,
	from, to domain.PayrollPeriodStatus, updates map[string]interface{}) (int64, error) {

	patch := make(map[string]interface{}, len(updates)+2)
	for k, v := range updates {
		patch[k] = v
	}
	patch["status"] = to
	patch["updated_at"] = time.Now()

	res := r.db.WithContext(ctx).
		Model(&domain.PayrollPeriod{}).
		Where("id = ? AND company_id = ? AND status = ?", id, companyID, from).
		Updates(patch)
	return res.RowsAffected, res.Error
}

// ---------------------------------------------------------------------------
// Payrolls
// ---------------------------------------------------------------------------

// CreatePayroll inserts a payroll record.
func (r *payrollRepositoryGorm) CreatePayroll(ctx context.Context, payroll *domain.Payroll) error {
	return r.db.WithContext(ctx).Create(payroll).Error
}

// GetPayrollByID loads one payroll record belonging to the company.
func (r *payrollRepositoryGorm) GetPayrollByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Payroll, error) {
	var payroll domain.Payroll
	err := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		First(&payroll).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPayrollNotFound
		}
		return nil, err
	}
	return &payroll, nil
}

// GetPayrollByEmployeeMonth loads one employee's payroll for a month.
func (r *payrollRepositoryGorm) GetPayrollByEmployeeMonth(ctx context.Context, companyID, employeeID uuid.UUID, year, month int) (*domain.Payroll, error) {
	var payroll domain.Payroll
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND employee_id = ? AND pay_year = ? AND pay_month = ?",
			companyID, employeeID, year, month).
		First(&payroll).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPayrollNotFound
		}
		return nil, err
	}
	return &payroll, nil
}

// payrollListSelect is the column list of a payroll list row.
//
// employees and departments are LEFT JOINed and the text columns are COALESCEd:
// migration 000019 deliberately kept payroll history alive across an employee
// deletion (근로기준법 제42조 - 3년 보존), so a payroll whose employee row is
// gone must still list, with a blank name rather than a failed scan.
const payrollListSelect = `
	p.id,
	p.employee_id,
	COALESCE(e.employee_no, '') AS employee_no,
	COALESCE(e.name, '')        AS employee_name,
	COALESCE(d.name, '')        AS department_name,
	p.payroll_period_id,
	p.pay_year,
	p.pay_month,
	p.status,
	p.base_salary,
	p.total_earnings,
	p.total_deductions,
	p.net_pay,
	p.payment_date`

// listPayrollsQuery builds the filtered, joined base query.
func (r *payrollRepositoryGorm) listPayrollsQuery(ctx context.Context, filter *PayrollFilter) *gorm.DB {
	// The company predicate is repeated on every joined table. Joining on the
	// id alone would let a payroll row reach an employee row of another tenant
	// if an id were ever reused or forged.
	query := r.db.WithContext(ctx).
		Table("payrolls AS p").
		Joins("LEFT JOIN employees AS e ON e.id = p.employee_id AND e.company_id = p.company_id").
		Joins("LEFT JOIN departments AS d ON d.id = e.department_id AND d.company_id = p.company_id").
		Where("p.company_id = ?", filter.CompanyID)

	if filter.PeriodID != nil {
		query = query.Where("p.payroll_period_id = ?", *filter.PeriodID)
	}
	if filter.EmployeeID != nil {
		query = query.Where("p.employee_id = ?", *filter.EmployeeID)
	}
	if filter.PayYear != nil {
		query = query.Where("p.pay_year = ?", *filter.PayYear)
	}
	if filter.PayMonth != nil {
		query = query.Where("p.pay_month = ?", *filter.PayMonth)
	}
	if filter.Status != nil {
		query = query.Where("p.status = ?", *filter.Status)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		query = query.Where("e.name ILIKE ? OR e.employee_no ILIKE ?", like, like)
	}
	return query
}

// ListPayrolls returns a page of payroll rows joined with employee identity.
func (r *payrollRepositoryGorm) ListPayrolls(ctx context.Context, filter *PayrollFilter) ([]*PayrollListRow, int64, error) {
	var total int64
	if err := r.listPayrollsQuery(ctx, filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []*PayrollListRow
	err := r.listPayrollsQuery(ctx, filter).
		Select(payrollListSelect).
		Order("p.pay_year DESC, p.pay_month DESC, e.employee_no ASC").
		Offset(offsetOf(filter.Page, filter.PageSize)).
		Limit(filter.PageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ListPayrollsByPeriod loads every payroll record of one period.
func (r *payrollRepositoryGorm) ListPayrollsByPeriod(ctx context.Context, companyID, periodID uuid.UUID) ([]*domain.Payroll, error) {
	var payrolls []*domain.Payroll
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND payroll_period_id = ?", companyID, periodID).
		Order("created_at ASC").
		Find(&payrolls).Error
	if err != nil {
		return nil, err
	}
	return payrolls, nil
}

// UpdatePayroll persists a modified payroll record.
func (r *payrollRepositoryGorm) UpdatePayroll(ctx context.Context, payroll *domain.Payroll) error {
	return r.db.WithContext(ctx).
		Model(&domain.Payroll{}).
		Where("id = ? AND company_id = ?", payroll.ID, payroll.CompanyID).
		Save(payroll).Error
}

// DeletePayroll removes a payroll record. Its items go with it through the
// ON DELETE CASCADE on payroll_items.payroll_id.
func (r *payrollRepositoryGorm) DeletePayroll(ctx context.Context, companyID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		Delete(&domain.Payroll{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrPayrollNotFound
	}
	return nil
}

// TransitionPayrollsByPeriod applies a guarded status change to a whole period.
func (r *payrollRepositoryGorm) TransitionPayrollsByPeriod(ctx context.Context, companyID, periodID uuid.UUID,
	from, to domain.PayrollStatus, updates map[string]interface{}) (int64, error) {

	patch := make(map[string]interface{}, len(updates)+2)
	for k, v := range updates {
		patch[k] = v
	}
	patch["status"] = to
	patch["updated_at"] = time.Now()

	res := r.db.WithContext(ctx).
		Model(&domain.Payroll{}).
		Where("company_id = ? AND payroll_period_id = ? AND status = ?", companyID, periodID, from).
		Updates(patch)
	return res.RowsAffected, res.Error
}

// ---------------------------------------------------------------------------
// Items
// ---------------------------------------------------------------------------

// ReplaceItems swaps a payroll's item rows for a new set.
//
// The delete and the insert are one statement pair with no commit between
// them; the caller runs this inside WithTransaction together with the header
// update, so a payroll is never briefly stored with the old rows and the new
// header.
func (r *payrollRepositoryGorm) ReplaceItems(ctx context.Context, companyID, payrollID uuid.UUID, items []domain.PayrollItem) error {
	db := r.db.WithContext(ctx)

	if err := db.
		Where("payroll_id = ? AND company_id = ?", payrollID, companyID).
		Delete(&domain.PayrollItem{}).Error; err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}

	for i := range items {
		items[i].PayrollID = payrollID
		items[i].CompanyID = companyID
	}
	return db.Create(&items).Error
}

// ListItems returns a payroll's item rows in display order.
func (r *payrollRepositoryGorm) ListItems(ctx context.Context, companyID, payrollID uuid.UUID) ([]domain.PayrollItem, error) {
	var items []domain.PayrollItem
	err := r.db.WithContext(ctx).
		Where("payroll_id = ? AND company_id = ?", payrollID, companyID).
		Order("sort_order ASC, item_code ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// SumItemsByPeriod totals the item rows of every payroll in a period, in one
// query, so that reconciling a 300-person payroll run does not issue 300
// selects.
func (r *payrollRepositoryGorm) SumItemsByPeriod(ctx context.Context, companyID, periodID uuid.UUID) ([]PayrollItemTotals, error) {
	var totals []PayrollItemTotals
	err := r.db.WithContext(ctx).
		Table("payroll_items AS i").
		Joins("JOIN payrolls AS p ON p.id = i.payroll_id AND p.company_id = i.company_id").
		Select(`i.payroll_id,
			COALESCE(SUM(CASE WHEN i.item_type = 'earning'   THEN i.amount ELSE 0 END), 0) AS earning_total,
			COALESCE(SUM(CASE WHEN i.item_type = 'deduction' THEN i.amount ELSE 0 END), 0) AS deduction_total`).
		Where("i.company_id = ? AND p.payroll_period_id = ?", companyID, periodID).
		Group("i.payroll_id").
		Scan(&totals).Error
	if err != nil {
		return nil, err
	}
	return totals, nil
}

// ---------------------------------------------------------------------------
// Aggregates
// ---------------------------------------------------------------------------

// SummarizePeriod totals the payrolls of one period.
//
// Cancelled records are excluded: they were withdrawn and paying them out again
// through the period total would be a real transfer of real money.
func (r *payrollRepositoryGorm) SummarizePeriod(ctx context.Context, companyID, periodID uuid.UUID) (*PayrollPeriodTotals, error) {
	var totals PayrollPeriodTotals
	err := r.db.WithContext(ctx).
		Model(&domain.Payroll{}).
		Select(`COUNT(*) AS total_employees,
			COALESCE(SUM(total_earnings), 0)   AS total_earnings,
			COALESCE(SUM(total_deductions), 0) AS total_deductions,
			COALESCE(SUM(net_pay), 0)          AS total_net_pay`).
		Where("company_id = ? AND payroll_period_id = ? AND status <> ?",
			companyID, periodID, domain.PayrollStatusCancelled).
		Scan(&totals).Error
	if err != nil {
		return nil, err
	}
	return &totals, nil
}

// ---------------------------------------------------------------------------
// Transactions
// ---------------------------------------------------------------------------

// WithTransaction runs fn against a payroll and an insurance repository that
// share one transaction.
func (r *payrollRepositoryGorm) WithTransaction(ctx context.Context, fn func(tx PayrollTx) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(PayrollTx{
			Payroll:   &payrollRepositoryGorm{db: tx},
			Insurance: &insuranceRepositoryGorm{db: tx},
		})
	})
}

// offsetOf converts a 1-based page number into a SQL OFFSET.
//
// The page size is clamped by internal/handler/pagination before it reaches
// here; this only guards against a zero or negative page number producing a
// negative offset, which PostgreSQL rejects outright.
func offsetOf(page, pageSize int) int {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		return 0
	}
	return (page - 1) * pageSize
}
