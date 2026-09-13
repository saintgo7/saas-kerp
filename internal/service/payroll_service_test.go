package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// The tests below use an in-memory repository whose WithTransaction really
// rolls back. That matters: the point of most of these cases is that a failed
// calculation leaves NOTHING behind, and a fake that committed each write as it
// happened would pass while the real code was broken.

var (
	payrollTestCompanyID  = uuid.MustParse("0192f3a0-0000-7000-8000-00000000c0c0")
	payrollTestEmployeeID = uuid.MustParse("0192f3a0-0000-7000-8000-00000000e001")
	payrollTestUserID     = uuid.MustParse("0192f3a0-0000-7000-8000-0000000000a1")
)

func payrollInt64p(v int64) *int64 { return &v }

// ---------------------------------------------------------------------------
// Fake store
// ---------------------------------------------------------------------------

type payrollFakeStore struct {
	periods       map[uuid.UUID]*domain.PayrollPeriod
	payrolls      map[uuid.UUID]*domain.Payroll
	items         map[uuid.UUID][]domain.PayrollItem
	contributions map[string]*domain.InsuranceMonthlyContribution
	qualification map[uuid.UUID]*domain.EmployeeInsurance
	reports       map[uuid.UUID]*domain.InsuranceReport
	reportItems   map[uuid.UUID][]domain.InsuranceReportItem

	// inTx guards against nested snapshots, which the service never does.
	inTx bool
}

func newPayrollFakeStore() *payrollFakeStore {
	return &payrollFakeStore{
		periods:       map[uuid.UUID]*domain.PayrollPeriod{},
		payrolls:      map[uuid.UUID]*domain.Payroll{},
		items:         map[uuid.UUID][]domain.PayrollItem{},
		contributions: map[string]*domain.InsuranceMonthlyContribution{},
		qualification: map[uuid.UUID]*domain.EmployeeInsurance{},
		reports:       map[uuid.UUID]*domain.InsuranceReport{},
		reportItems:   map[uuid.UUID][]domain.InsuranceReportItem{},
	}
}

// snapshot deep-copies everything a transaction could modify.
func (s *payrollFakeStore) snapshot() *payrollFakeStore {
	copySnap := newPayrollFakeStore()
	for k, v := range s.periods {
		c := *v
		copySnap.periods[k] = &c
	}
	for k, v := range s.payrolls {
		c := *v
		copySnap.payrolls[k] = &c
	}
	for k, v := range s.items {
		rows := make([]domain.PayrollItem, len(v))
		copy(rows, v)
		copySnap.items[k] = rows
	}
	for k, v := range s.contributions {
		c := *v
		copySnap.contributions[k] = &c
	}
	for k, v := range s.qualification {
		c := *v
		copySnap.qualification[k] = &c
	}
	for k, v := range s.reports {
		c := *v
		copySnap.reports[k] = &c
	}
	for k, v := range s.reportItems {
		rows := make([]domain.InsuranceReportItem, len(v))
		copy(rows, v)
		copySnap.reportItems[k] = rows
	}
	return copySnap
}

// restore puts back a snapshot, which is what a rollback does.
func (s *payrollFakeStore) restore(from *payrollFakeStore) {
	s.periods = from.periods
	s.payrolls = from.payrolls
	s.items = from.items
	s.contributions = from.contributions
	s.qualification = from.qualification
	s.reports = from.reports
	s.reportItems = from.reportItems
}

func payrollContributionKey(companyID, employeeID uuid.UUID, year, month int) string {
	return companyID.String() + "|" + employeeID.String() + "|" +
		time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
}

// ---------------------------------------------------------------------------
// Fake payroll repository
// ---------------------------------------------------------------------------

type payrollFakeRepo struct{ store *payrollFakeStore }

func (r *payrollFakeRepo) CreatePeriod(_ context.Context, p *domain.PayrollPeriod) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	c := *p
	r.store.periods[p.ID] = &c
	return nil
}

func (r *payrollFakeRepo) GetPeriodByID(_ context.Context, companyID, id uuid.UUID) (*domain.PayrollPeriod, error) {
	p, ok := r.store.periods[id]
	if !ok || p.CompanyID != companyID {
		return nil, domain.ErrPayrollPeriodNotFound
	}
	c := *p
	return &c, nil
}

func (r *payrollFakeRepo) GetPeriodByMonth(_ context.Context, companyID uuid.UUID, year, month int) (*domain.PayrollPeriod, error) {
	for _, p := range r.store.periods {
		if p.CompanyID == companyID && p.PayYear == year && p.PayMonth == month {
			c := *p
			return &c, nil
		}
	}
	return nil, domain.ErrPayrollPeriodNotFound
}

func (r *payrollFakeRepo) ListPeriods(_ context.Context, f *repository.PayrollPeriodFilter) ([]*domain.PayrollPeriod, int64, error) {
	var out []*domain.PayrollPeriod
	for _, p := range r.store.periods {
		if p.CompanyID == f.CompanyID {
			c := *p
			out = append(out, &c)
		}
	}
	return out, int64(len(out)), nil
}

func (r *payrollFakeRepo) UpdatePeriod(_ context.Context, p *domain.PayrollPeriod) error {
	c := *p
	r.store.periods[p.ID] = &c
	return nil
}

func (r *payrollFakeRepo) TransitionPeriod(_ context.Context, companyID, id uuid.UUID,
	from, to domain.PayrollPeriodStatus, updates map[string]interface{}) (int64, error) {

	p, ok := r.store.periods[id]
	// The status is part of the match, exactly as the SQL WHERE clause is.
	if !ok || p.CompanyID != companyID || p.Status != from {
		return 0, nil
	}
	p.Status = to
	applyPayrollPeriodUpdates(p, updates)
	return 1, nil
}

func applyPayrollPeriodUpdates(p *domain.PayrollPeriod, updates map[string]interface{}) {
	for k, v := range updates {
		switch k {
		case "total_employees":
			p.TotalEmployees = v.(int)
		case "total_earnings":
			p.TotalEarnings = v.(int64)
		case "total_deductions":
			p.TotalDeductions = v.(int64)
		case "total_net_pay":
			p.TotalNetPay = v.(int64)
		case "approved_at":
			t := v.(time.Time)
			p.ApprovedAt = &t
		case "calculated_at":
			t := v.(time.Time)
			p.CalculatedAt = &t
		case "paid_at":
			t := v.(time.Time)
			p.PaidAt = &t
		case "payment_date":
			t := v.(time.Time)
			p.PaymentDate = &t
		case "approved_by":
			p.ApprovedBy = v.(*uuid.UUID)
		case "calculated_by":
			p.CalculatedBy = v.(*uuid.UUID)
		}
	}
}

func (r *payrollFakeRepo) CreatePayroll(_ context.Context, p *domain.Payroll) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	c := *p
	r.store.payrolls[p.ID] = &c
	return nil
}

func (r *payrollFakeRepo) GetPayrollByID(_ context.Context, companyID, id uuid.UUID) (*domain.Payroll, error) {
	p, ok := r.store.payrolls[id]
	if !ok || p.CompanyID != companyID {
		return nil, domain.ErrPayrollNotFound
	}
	c := *p
	return &c, nil
}

func (r *payrollFakeRepo) GetPayrollByEmployeeMonth(_ context.Context, companyID, employeeID uuid.UUID, year, month int) (*domain.Payroll, error) {
	for _, p := range r.store.payrolls {
		if p.CompanyID == companyID && p.EmployeeID == employeeID && p.PayYear == year && p.PayMonth == month {
			c := *p
			return &c, nil
		}
	}
	return nil, domain.ErrPayrollNotFound
}

func (r *payrollFakeRepo) ListPayrolls(_ context.Context, f *repository.PayrollFilter) ([]*repository.PayrollListRow, int64, error) {
	var out []*repository.PayrollListRow
	for _, p := range r.store.payrolls {
		if p.CompanyID != f.CompanyID {
			continue
		}
		out = append(out, &repository.PayrollListRow{
			ID: p.ID, EmployeeID: p.EmployeeID, PayYear: p.PayYear, PayMonth: p.PayMonth,
			Status: string(p.Status), TotalEarnings: p.TotalEarnings,
			TotalDeductions: p.TotalDeductions, NetPay: p.NetPay,
		})
	}
	return out, int64(len(out)), nil
}

func (r *payrollFakeRepo) ListPayrollsByPeriod(_ context.Context, companyID, periodID uuid.UUID) ([]*domain.Payroll, error) {
	var out []*domain.Payroll
	for _, p := range r.store.payrolls {
		if p.CompanyID == companyID && p.PayrollPeriodID != nil && *p.PayrollPeriodID == periodID {
			c := *p
			out = append(out, &c)
		}
	}
	return out, nil
}

func (r *payrollFakeRepo) UpdatePayroll(_ context.Context, p *domain.Payroll) error {
	existing, ok := r.store.payrolls[p.ID]
	if !ok || existing.CompanyID != p.CompanyID {
		return domain.ErrPayrollNotFound
	}
	c := *p
	r.store.payrolls[p.ID] = &c
	return nil
}

func (r *payrollFakeRepo) DeletePayroll(_ context.Context, companyID, id uuid.UUID) error {
	p, ok := r.store.payrolls[id]
	if !ok || p.CompanyID != companyID {
		return domain.ErrPayrollNotFound
	}
	delete(r.store.payrolls, id)
	delete(r.store.items, id)
	return nil
}

func (r *payrollFakeRepo) TransitionPayrollsByPeriod(_ context.Context, companyID, periodID uuid.UUID,
	from, to domain.PayrollStatus, updates map[string]interface{}) (int64, error) {

	var n int64
	for _, p := range r.store.payrolls {
		if p.CompanyID != companyID || p.PayrollPeriodID == nil || *p.PayrollPeriodID != periodID {
			continue
		}
		if p.Status != from {
			continue
		}
		p.Status = to
		for k, v := range updates {
			switch k {
			case "paid_at":
				t := v.(time.Time)
				p.PaidAt = &t
			case "payment_date":
				t := v.(time.Time)
				p.PaymentDate = &t
			}
		}
		n++
	}
	return n, nil
}

func (r *payrollFakeRepo) ReplaceItems(_ context.Context, companyID, payrollID uuid.UUID, items []domain.PayrollItem) error {
	p, ok := r.store.payrolls[payrollID]
	if !ok || p.CompanyID != companyID {
		return domain.ErrPayrollNotFound
	}
	rows := make([]domain.PayrollItem, len(items))
	copy(rows, items)
	for i := range rows {
		rows[i].PayrollID = payrollID
		rows[i].CompanyID = companyID
	}
	r.store.items[payrollID] = rows
	return nil
}

func (r *payrollFakeRepo) ListItems(_ context.Context, companyID, payrollID uuid.UUID) ([]domain.PayrollItem, error) {
	p, ok := r.store.payrolls[payrollID]
	if !ok || p.CompanyID != companyID {
		return nil, domain.ErrPayrollNotFound
	}
	rows := make([]domain.PayrollItem, len(r.store.items[payrollID]))
	copy(rows, r.store.items[payrollID])
	return rows, nil
}

func (r *payrollFakeRepo) SumItemsByPeriod(_ context.Context, companyID, periodID uuid.UUID) ([]repository.PayrollItemTotals, error) {
	var out []repository.PayrollItemTotals
	for id, rows := range r.store.items {
		p, ok := r.store.payrolls[id]
		if !ok || p.CompanyID != companyID || p.PayrollPeriodID == nil || *p.PayrollPeriodID != periodID {
			continue
		}
		totals := repository.PayrollItemTotals{PayrollID: id}
		for _, row := range rows {
			switch row.ItemType {
			case domain.PayrollItemEarning:
				totals.EarningTotal += row.Amount
			case domain.PayrollItemDeduction:
				totals.DeductionTotal += row.Amount
			}
		}
		out = append(out, totals)
	}
	return out, nil
}

func (r *payrollFakeRepo) SummarizePeriod(_ context.Context, companyID, periodID uuid.UUID) (*repository.PayrollPeriodTotals, error) {
	totals := &repository.PayrollPeriodTotals{}
	for _, p := range r.store.payrolls {
		if p.CompanyID != companyID || p.PayrollPeriodID == nil || *p.PayrollPeriodID != periodID {
			continue
		}
		if p.Status == domain.PayrollStatusCancelled {
			continue
		}
		totals.TotalEmployees++
		totals.TotalEarnings += p.TotalEarnings
		totals.TotalDeductions += p.TotalDeductions
		totals.TotalNetPay += p.NetPay
	}
	return totals, nil
}

// WithTransaction snapshots the store, runs fn, and restores on error. That is
// what makes the "nothing was written" assertions in these tests meaningful.
func (r *payrollFakeRepo) WithTransaction(ctx context.Context, fn func(tx repository.PayrollTx) error) error {
	if r.store.inTx {
		return fn(repository.PayrollTx{
			Payroll:   r,
			Insurance: &payrollFakeInsuranceRepo{store: r.store},
		})
	}

	before := r.store.snapshot()
	r.store.inTx = true
	err := fn(repository.PayrollTx{
		Payroll:   r,
		Insurance: &payrollFakeInsuranceRepo{store: r.store},
	})
	r.store.inTx = false
	if err != nil {
		r.store.restore(before)
	}
	return err
}

// ---------------------------------------------------------------------------
// Fake insurance repository
// ---------------------------------------------------------------------------

type payrollFakeInsuranceRepo struct{ store *payrollFakeStore }

func (r *payrollFakeInsuranceRepo) GetEmployeeInsurance(_ context.Context, companyID, employeeID uuid.UUID) (*domain.EmployeeInsurance, error) {
	rec, ok := r.store.qualification[employeeID]
	if !ok || rec.CompanyID != companyID {
		return nil, domain.ErrEmployeeInsuranceNotFound
	}
	c := *rec
	return &c, nil
}

func (r *payrollFakeInsuranceRepo) UpsertContribution(_ context.Context, c *domain.InsuranceMonthlyContribution) error {
	key := payrollContributionKey(c.CompanyID, c.EmployeeID, c.ContributionYear, c.ContributionMonth)
	copied := *c
	r.store.contributions[key] = &copied
	return nil
}

// The remaining InsuranceRepository methods are not exercised by the payroll
// service; they exist so the fake satisfies the interface.
func (r *payrollFakeInsuranceRepo) CreateWorkplace(context.Context, *domain.InsuranceWorkplace) error {
	return nil
}
func (r *payrollFakeInsuranceRepo) GetWorkplaceByID(context.Context, uuid.UUID, uuid.UUID) (*domain.InsuranceWorkplace, error) {
	return nil, domain.ErrInsuranceWorkplaceNotFound
}
func (r *payrollFakeInsuranceRepo) GetWorkplaceByBusinessNumber(context.Context, uuid.UUID, string) (*domain.InsuranceWorkplace, error) {
	return nil, domain.ErrInsuranceWorkplaceNotFound
}
func (r *payrollFakeInsuranceRepo) ListWorkplaces(context.Context, *repository.InsuranceWorkplaceFilter) ([]*domain.InsuranceWorkplace, int64, error) {
	return nil, 0, nil
}
func (r *payrollFakeInsuranceRepo) UpdateWorkplace(context.Context, *domain.InsuranceWorkplace) error {
	return nil
}
func (r *payrollFakeInsuranceRepo) UpsertEmployeeInsurance(_ context.Context, rec *domain.EmployeeInsurance) error {
	c := *rec
	r.store.qualification[rec.EmployeeID] = &c
	return nil
}
func (r *payrollFakeInsuranceRepo) ListEmployeeInsurance(context.Context, *repository.EmployeeInsuranceFilter) ([]*repository.EmployeeInsuranceRow, int64, error) {
	return nil, 0, nil
}
func (r *payrollFakeInsuranceRepo) ListContributions(context.Context, *repository.InsuranceContributionFilter) ([]*repository.InsuranceContributionRow, int64, error) {
	return nil, 0, nil
}
func (r *payrollFakeInsuranceRepo) SummarizeContributions(context.Context, uuid.UUID, int, int) (*repository.InsuranceContributionSummary, error) {
	return &repository.InsuranceContributionSummary{}, nil
}
func (r *payrollFakeInsuranceRepo) CreateReport(_ context.Context, rep *domain.InsuranceReport) error {
	if rep.ID == uuid.Nil {
		rep.ID = uuid.New()
	}
	c := *rep
	r.store.reports[rep.ID] = &c
	return nil
}
func (r *payrollFakeInsuranceRepo) GetReportByID(_ context.Context, companyID, id uuid.UUID) (*domain.InsuranceReport, error) {
	rep, ok := r.store.reports[id]
	if !ok || rep.CompanyID != companyID {
		return nil, domain.ErrInsuranceReportNotFound
	}
	c := *rep
	return &c, nil
}
func (r *payrollFakeInsuranceRepo) ListReports(context.Context, *repository.InsuranceReportFilter) ([]*domain.InsuranceReport, int64, error) {
	return nil, 0, nil
}
func (r *payrollFakeInsuranceRepo) UpdateReport(context.Context, *domain.InsuranceReport) error {
	return nil
}
func (r *payrollFakeInsuranceRepo) DeleteReport(_ context.Context, companyID, id uuid.UUID) error {
	rep, ok := r.store.reports[id]
	if !ok || rep.CompanyID != companyID {
		return domain.ErrInsuranceReportNotFound
	}
	delete(r.store.reports, id)
	delete(r.store.reportItems, id)
	return nil
}
func (r *payrollFakeInsuranceRepo) TransitionReport(_ context.Context, companyID, id uuid.UUID,
	from, to domain.InsuranceReportStatus, updates map[string]interface{}) (int64, error) {

	rep, ok := r.store.reports[id]
	// The source status is part of the match, exactly as the SQL WHERE is.
	if !ok || rep.CompanyID != companyID || rep.Status != from {
		return 0, nil
	}
	rep.Status = to
	if reason, found := updates["rejection_reason"]; found {
		rep.RejectionReason = reason.(string)
	}
	return 1, nil
}
func (r *payrollFakeInsuranceRepo) ReplaceReportItems(_ context.Context, companyID, reportID uuid.UUID, items []domain.InsuranceReportItem) error {
	rep, ok := r.store.reports[reportID]
	if !ok || rep.CompanyID != companyID {
		return domain.ErrInsuranceReportNotFound
	}
	rows := make([]domain.InsuranceReportItem, len(items))
	copy(rows, items)
	for i := range rows {
		rows[i].ReportID = reportID
	}
	r.store.reportItems[reportID] = rows
	return nil
}
func (r *payrollFakeInsuranceRepo) ListReportItems(_ context.Context, companyID, reportID uuid.UUID) ([]domain.InsuranceReportItem, error) {
	rep, ok := r.store.reports[reportID]
	if !ok || rep.CompanyID != companyID {
		return nil, domain.ErrInsuranceReportNotFound
	}
	rows := make([]domain.InsuranceReportItem, len(r.store.reportItems[reportID]))
	copy(rows, r.store.reportItems[reportID])
	return rows, nil
}
func (r *payrollFakeInsuranceRepo) ListCredentialStatus(context.Context, uuid.UUID) ([]*domain.InsuranceCredentialStatus, error) {
	return nil, nil
}
func (r *payrollFakeInsuranceRepo) ListEDIJobs(context.Context, *repository.EDIJobFilter) ([]*domain.EDIJob, int64, error) {
	return nil, 0, nil
}
func (r *payrollFakeInsuranceRepo) GetEDIJobByID(context.Context, uuid.UUID, uuid.UUID) (*domain.EDIJob, error) {
	return nil, domain.ErrInsuranceReportNotFound
}
func (r *payrollFakeInsuranceRepo) WithTransaction(ctx context.Context, fn func(repo repository.InsuranceRepository) error) error {
	return fn(r)
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// newPayrollFixture builds a service over a fresh store with one open period.
func newPayrollFixture(t *testing.T) (*service.PayrollService, *payrollFakeStore, uuid.UUID) {
	t.Helper()

	store := newPayrollFakeStore()
	payrollRepo := &payrollFakeRepo{store: store}
	insuranceRepo := &payrollFakeInsuranceRepo{store: store}
	svc := service.NewPayrollService(payrollRepo, insuranceRepo)

	period, err := svc.CreatePeriod(context.Background(), payrollTestCompanyID, &service.CreatePayrollPeriodInput{
		PayYear:     2025,
		PayMonth:    8,
		PeriodStart: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2025, 8, 31, 0, 0, 0, 0, time.UTC),
		PaymentDate: payrollTimePtr(time.Date(2025, 8, 25, 0, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)

	return svc, store, period.ID
}

func payrollTimePtr(t time.Time) *time.Time { return &t }

// createTestPayroll adds a 320만원 draft payroll for one employee.
func createTestPayroll(t *testing.T, svc *service.PayrollService, periodID uuid.UUID, employeeID uuid.UUID) *domain.Payroll {
	t.Helper()

	payroll, err := svc.CreatePayroll(context.Background(), payrollTestCompanyID, &service.CreatePayrollInput{
		EmployeeID:      employeeID,
		PayYear:         2025,
		PayMonth:        8,
		PayrollPeriodID: &periodID,
		PaymentDate:     payrollTimePtr(time.Date(2025, 8, 25, 0, 0, 0, 0, time.UTC)),
		PayrollEarningsInput: service.PayrollEarningsInput{
			Earnings: domain.PayrollEarnings{
				BaseSalary: 3_000_000,
				Allowances: []domain.PayrollNamedAmount{
					{Code: "meal", Name: "식대", Amount: 200_000, NonTaxable: true},
				},
			},
			WorkDays: 22,
		},
	})
	require.NoError(t, err)
	return payroll
}

// calcInput is the standard calculation request used below.
func calcInput() *service.CalculatePayrollInput {
	return &service.CalculatePayrollInput{
		IncomeTax:              payrollInt64p(84_850),
		WorkplaceSize:          domain.WorkplaceUnder150,
		IndustrialAccidentRate: payrollInt64p(1530),
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCreatePayrollWritesHeaderAndLinesTogether(t *testing.T) {
	svc, store, periodID := newPayrollFixture(t)
	payroll := createTestPayroll(t, svc, periodID, payrollTestEmployeeID)

	// Even a draft reconciles: gross equals the sum of its rows and net equals
	// gross while nothing has been deducted.
	assert.Equal(t, int64(3_200_000), payroll.TotalEarnings)
	assert.Equal(t, int64(0), payroll.TotalDeductions)
	assert.Equal(t, int64(3_200_000), payroll.NetPay)

	rows := store.items[payroll.ID]
	require.Len(t, rows, 2, "기본급과 식대 두 줄이 저장되어야 한다")
	assert.NoError(t, domain.ValidatePayrollTotals(store.payrolls[payroll.ID], rows))
}

func TestCreatePayrollRejectsDuplicate(t *testing.T) {
	svc, _, periodID := newPayrollFixture(t)
	createTestPayroll(t, svc, periodID, payrollTestEmployeeID)

	_, err := svc.CreatePayroll(context.Background(), payrollTestCompanyID, &service.CreatePayrollInput{
		EmployeeID:      payrollTestEmployeeID,
		PayYear:         2025,
		PayMonth:        8,
		PayrollPeriodID: &periodID,
	})
	assert.ErrorIs(t, err, domain.ErrPayrollExists)
}

func TestCalculatePayrollWritesEverythingInOneTransaction(t *testing.T) {
	svc, store, periodID := newPayrollFixture(t)
	payroll := createTestPayroll(t, svc, periodID, payrollTestEmployeeID)

	calculated, err := svc.CalculatePayroll(context.Background(), payrollTestCompanyID, payroll.ID, calcInput())
	require.NoError(t, err)

	// Hand-checked in internal/domain/payroll_test.go for the same inputs.
	assert.Equal(t, domain.PayrollStatusCalculated, calculated.Status)
	assert.Equal(t, int64(375_450), calculated.TotalDeductions)
	assert.Equal(t, int64(2_824_550), calculated.NetPay)

	stored := store.payrolls[payroll.ID]
	rows := store.items[payroll.ID]
	assert.NoError(t, domain.ValidatePayrollTotals(stored, rows),
		"저장된 헤더와 행이 어긋나면 급여명세서가 아무것과도 대사되지 않는다")

	// The 4대보험 register is written by the same transaction, so the two
	// registers cannot disagree about what was withheld.
	contribution := store.contributions[payrollContributionKey(payrollTestCompanyID, payrollTestEmployeeID, 2025, 8)]
	require.NotNil(t, contribution, "4대보험 월별 내역이 함께 기록되어야 한다")
	assert.Equal(t, int64(135_000), contribution.NPSEmployee)
	assert.Equal(t, int64(106_350), contribution.NHISEmployee)
	assert.Equal(t, int64(13_770), contribution.NHISLTCEmployee)
	assert.Equal(t, int64(27_000), contribution.EIEmployee)
	assert.Equal(t, int64(34_500), contribution.EIEmployer, "사업주 고용보험료는 고용안정분을 포함한다")
	assert.Equal(t, int64(45_900), contribution.WCIEmployer)
	assert.Equal(t, payroll.ID, *contribution.PayrollID)
}

func TestCalculatePayrollWithoutIncomeTaxWritesNothing(t *testing.T) {
	svc, store, periodID := newPayrollFixture(t)
	payroll := createTestPayroll(t, svc, periodID, payrollTestEmployeeID)

	before := len(store.contributions)

	_, err := svc.CalculatePayroll(context.Background(), payrollTestCompanyID, payroll.ID, &service.CalculatePayrollInput{
		WorkplaceSize: domain.WorkplaceUnder150,
	})
	assert.ErrorIs(t, err, domain.ErrSimplifiedTaxTableUnavailable)

	// The transaction rolled back: the payroll is still a draft with no
	// deductions and no 4대보험 row was created.
	stored := store.payrolls[payroll.ID]
	assert.Equal(t, domain.PayrollStatusDraft, stored.Status)
	assert.Zero(t, stored.TotalDeductions)
	assert.Len(t, store.contributions, before)
}

func TestCalculatePayrollWithUnverifiedRatesWritesNothing(t *testing.T) {
	svc, store, periodID := newPayrollFixture(t)
	payroll := createTestPayroll(t, svc, periodID, payrollTestEmployeeID)

	in := calcInput()
	// 2026 rates are deliberately absent from internal/domain.
	in.ContributionDate = payrollTimePtr(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))

	_, err := svc.CalculatePayroll(context.Background(), payrollTestCompanyID, payroll.ID, in)
	assert.ErrorIs(t, err, domain.ErrSocialInsuranceRatesUnavailable)

	stored := store.payrolls[payroll.ID]
	assert.Equal(t, domain.PayrollStatusDraft, stored.Status)
	assert.Zero(t, stored.TotalDeductions)
	assert.Empty(t, store.contributions)
}

func TestCalculatePayrollHonoursQualification(t *testing.T) {
	svc, store, periodID := newPayrollFixture(t)
	payroll := createTestPayroll(t, svc, periodID, payrollTestEmployeeID)

	// 국민연금 자격이 없는 사원 (60세 이상 등).
	store.qualification[payrollTestEmployeeID] = &domain.EmployeeInsurance{
		CompanyID:     payrollTestCompanyID,
		EmployeeID:    payrollTestEmployeeID,
		NPSQualified:  false,
		NHISQualified: true,
		EIQualified:   true,
		WCIQualified:  true,
	}

	calculated, err := svc.CalculatePayroll(context.Background(), payrollTestCompanyID, payroll.ID, calcInput())
	require.NoError(t, err)

	assert.Zero(t, calculated.NPSEmployee, "자격이 없는 보험은 공제하지 않는다")
	assert.Equal(t, int64(106_350), calculated.NHISEmployee)
	assert.Equal(t, int64(375_450-135_000), calculated.TotalDeductions)
}

func TestUpdatePayrollResetsDeductions(t *testing.T) {
	svc, store, periodID := newPayrollFixture(t)
	payroll := createTestPayroll(t, svc, periodID, payrollTestEmployeeID)

	_, err := svc.CalculatePayroll(context.Background(), payrollTestCompanyID, payroll.ID, calcInput())
	require.NoError(t, err)

	updated, err := svc.UpdatePayroll(context.Background(), payrollTestCompanyID, payroll.ID, &service.PayrollEarningsInput{
		Earnings: domain.PayrollEarnings{BaseSalary: 4_000_000},
	})
	require.NoError(t, err)

	// Deductions computed from the old gross must not survive onto a new one.
	assert.Equal(t, domain.PayrollStatusDraft, updated.Status)
	assert.Equal(t, int64(4_000_000), updated.TotalEarnings)
	assert.Zero(t, updated.TotalDeductions)
	assert.Zero(t, updated.NPSEmployee)
	assert.Zero(t, updated.IncomeTax)
	assert.Equal(t, int64(4_000_000), updated.NetPay)

	assert.NoError(t, domain.ValidatePayrollTotals(store.payrolls[payroll.ID], store.items[payroll.ID]))
}

func TestApprovePeriodRejectsHeaderLineMismatch(t *testing.T) {
	svc, store, periodID := newPayrollFixture(t)
	payroll := createTestPayroll(t, svc, periodID, payrollTestEmployeeID)

	_, err := svc.CalculatePayroll(context.Background(), payrollTestCompanyID, payroll.ID, calcInput())
	require.NoError(t, err)

	_, err = svc.CalculatePeriod(context.Background(), payrollTestCompanyID, periodID,
		&service.CalculatePeriodInput{WorkplaceSize: domain.WorkplaceUnder150}, &payrollTestUserID)
	require.NoError(t, err)

	// Corrupt one stored line, the way a partially applied write would.
	rows := store.items[payroll.ID]
	rows[0].Amount += 1000
	store.items[payroll.ID] = rows

	_, err = svc.ApprovePeriod(context.Background(), payrollTestCompanyID, periodID, &payrollTestUserID)
	assert.ErrorIs(t, err, domain.ErrPayrollTotalsMismatch,
		"행 합계와 헤더가 어긋난 급여는 확정되어서는 안 된다")

	// Nothing moved: the run is still merely calculated.
	assert.Equal(t, domain.PayrollPeriodStatusCalculated, store.periods[periodID].Status)
	assert.Equal(t, domain.PayrollStatusCalculated, store.payrolls[payroll.ID].Status)
}

func TestApprovePeriodIsIdempotent(t *testing.T) {
	svc, store, periodID := newPayrollFixture(t)
	payroll := createTestPayroll(t, svc, periodID, payrollTestEmployeeID)

	_, err := svc.CalculatePayroll(context.Background(), payrollTestCompanyID, payroll.ID, calcInput())
	require.NoError(t, err)
	_, err = svc.CalculatePeriod(context.Background(), payrollTestCompanyID, periodID,
		&service.CalculatePeriodInput{WorkplaceSize: domain.WorkplaceUnder150}, &payrollTestUserID)
	require.NoError(t, err)

	approved, err := svc.ApprovePeriod(context.Background(), payrollTestCompanyID, periodID, &payrollTestUserID)
	require.NoError(t, err)
	assert.Equal(t, domain.PayrollPeriodStatusApproved, approved.Status)
	assert.Equal(t, 1, approved.TotalEmployees)
	// The header is written from what the rows say, not from a running total.
	assert.Equal(t, int64(3_200_000), approved.TotalEarnings)
	assert.Equal(t, int64(375_450), approved.TotalDeductions)
	assert.Equal(t, int64(2_824_550), approved.TotalNetPay)
	assert.Equal(t, domain.PayrollStatusApproved, store.payrolls[payroll.ID].Status)

	// A second approval must not confirm the run twice.
	_, err = svc.ApprovePeriod(context.Background(), payrollTestCompanyID, periodID, &payrollTestUserID)
	assert.ErrorIs(t, err, domain.ErrPayrollPeriodNotEditable)
}

func TestApprovedPeriodLocksItsPayrolls(t *testing.T) {
	svc, _, periodID := newPayrollFixture(t)
	payroll := createTestPayroll(t, svc, periodID, payrollTestEmployeeID)

	_, err := svc.CalculatePayroll(context.Background(), payrollTestCompanyID, payroll.ID, calcInput())
	require.NoError(t, err)
	_, err = svc.CalculatePeriod(context.Background(), payrollTestCompanyID, periodID,
		&service.CalculatePeriodInput{WorkplaceSize: domain.WorkplaceUnder150}, &payrollTestUserID)
	require.NoError(t, err)
	_, err = svc.ApprovePeriod(context.Background(), payrollTestCompanyID, periodID, &payrollTestUserID)
	require.NoError(t, err)

	// Recalculating one employee after the run was confirmed would silently
	// make the period header disagree with the sum of its payrolls.
	_, err = svc.CalculatePayroll(context.Background(), payrollTestCompanyID, payroll.ID, calcInput())
	assert.Error(t, err)

	_, err = svc.UpdatePayroll(context.Background(), payrollTestCompanyID, payroll.ID, &service.PayrollEarningsInput{
		Earnings: domain.PayrollEarnings{BaseSalary: 9_000_000},
	})
	assert.Error(t, err)

	assert.Error(t, svc.DeletePayroll(context.Background(), payrollTestCompanyID, payroll.ID))
}

func TestPeriodLifecycleOrder(t *testing.T) {
	svc, store, periodID := newPayrollFixture(t)
	payroll := createTestPayroll(t, svc, periodID, payrollTestEmployeeID)

	_, err := svc.CalculatePayroll(context.Background(), payrollTestCompanyID, payroll.ID, calcInput())
	require.NoError(t, err)

	// Paying before approval is refused.
	_, err = svc.MarkPeriodPaid(context.Background(), payrollTestCompanyID, periodID, nil)
	assert.ErrorIs(t, err, domain.ErrPayrollPeriodNotEditable)

	_, err = svc.CalculatePeriod(context.Background(), payrollTestCompanyID, periodID,
		&service.CalculatePeriodInput{WorkplaceSize: domain.WorkplaceUnder150}, &payrollTestUserID)
	require.NoError(t, err)

	// Closing before paying is refused.
	_, err = svc.ClosePeriod(context.Background(), payrollTestCompanyID, periodID)
	assert.ErrorIs(t, err, domain.ErrPayrollPeriodNotEditable)

	_, err = svc.ApprovePeriod(context.Background(), payrollTestCompanyID, periodID, &payrollTestUserID)
	require.NoError(t, err)

	paid, err := svc.MarkPeriodPaid(context.Background(), payrollTestCompanyID, periodID, nil)
	require.NoError(t, err)
	assert.Equal(t, domain.PayrollPeriodStatusPaid, paid.Status)
	assert.Equal(t, domain.PayrollStatusPaid, store.payrolls[payroll.ID].Status)

	// Paying twice is refused.
	_, err = svc.MarkPeriodPaid(context.Background(), payrollTestCompanyID, periodID, nil)
	assert.ErrorIs(t, err, domain.ErrPayrollPeriodNotEditable)

	closed, err := svc.ClosePeriod(context.Background(), payrollTestCompanyID, periodID)
	require.NoError(t, err)
	assert.Equal(t, domain.PayrollPeriodStatusClosed, closed.Status)
}

func TestCalculatePeriodSkipsPayrollsWithoutIncomeTax(t *testing.T) {
	svc, store, periodID := newPayrollFixture(t)

	withTax := createTestPayroll(t, svc, periodID, payrollTestEmployeeID)
	_, err := svc.CalculatePayroll(context.Background(), payrollTestCompanyID, withTax.ID, calcInput())
	require.NoError(t, err)

	otherEmployee := uuid.MustParse("0192f3a0-0000-7000-8000-00000000e002")
	withoutTax := createTestPayroll(t, svc, periodID, otherEmployee)

	result, err := svc.CalculatePeriod(context.Background(), payrollTestCompanyID, periodID,
		&service.CalculatePeriodInput{WorkplaceSize: domain.WorkplaceUnder150}, &payrollTestUserID)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Processed)
	require.Len(t, result.Skipped, 1, "소득세가 없는 급여는 건너뛰고 보고되어야 한다")
	assert.Equal(t, withoutTax.ID, result.Skipped[0].PayrollID)
	assert.Contains(t, result.Skipped[0].Reason, "소득세")

	// The skipped record stays a draft rather than being confirmed with zero
	// withholding.
	assert.Equal(t, domain.PayrollStatusDraft, store.payrolls[withoutTax.ID].Status)

	// And a run containing an uncalculated payroll cannot be approved.
	_, err = svc.ApprovePeriod(context.Background(), payrollTestCompanyID, periodID, &payrollTestUserID)
	assert.ErrorIs(t, err, domain.ErrPayrollNotCalculated)
}

func TestCalculatePeriodAppliesPerEmployeeOverrides(t *testing.T) {
	svc, store, periodID := newPayrollFixture(t)
	payroll := createTestPayroll(t, svc, periodID, payrollTestEmployeeID)

	result, err := svc.CalculatePeriod(context.Background(), payrollTestCompanyID, periodID, &service.CalculatePeriodInput{
		WorkplaceSize:          domain.WorkplaceUnder150,
		IndustrialAccidentRate: payrollInt64p(1530),
		Employees: map[uuid.UUID]service.CalculatePayrollInput{
			payrollTestEmployeeID: {IncomeTax: payrollInt64p(84_850)},
		},
	}, &payrollTestUserID)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Processed)
	assert.Empty(t, result.Skipped)

	stored := store.payrolls[payroll.ID]
	assert.Equal(t, int64(84_850), stored.IncomeTax)
	assert.Equal(t, int64(8_480), stored.LocalIncomeTax)
	// The 산재보험료율 falls through from the batch defaults.
	assert.Equal(t, int64(45_900), stored.WCIEmployer)
	assert.NoError(t, domain.ValidatePayrollTotals(stored, store.items[payroll.ID]))
}

func TestCrossTenantAccessIsRefused(t *testing.T) {
	svc, _, periodID := newPayrollFixture(t)
	payroll := createTestPayroll(t, svc, periodID, payrollTestEmployeeID)

	otherCompany := uuid.MustParse("0192f3a0-0000-7000-8000-00000000c0c1")

	_, err := svc.GetPayroll(context.Background(), otherCompany, payroll.ID)
	assert.ErrorIs(t, err, domain.ErrPayrollNotFound)

	_, err = svc.GetPeriod(context.Background(), otherCompany, periodID)
	assert.ErrorIs(t, err, domain.ErrPayrollPeriodNotFound)

	_, err = svc.CalculatePayroll(context.Background(), otherCompany, payroll.ID, calcInput())
	assert.ErrorIs(t, err, domain.ErrPayrollNotFound)
}

func TestCreatePeriodRejectsDuplicateMonth(t *testing.T) {
	svc, _, _ := newPayrollFixture(t)

	_, err := svc.CreatePeriod(context.Background(), payrollTestCompanyID, &service.CreatePayrollPeriodInput{
		PayYear:     2025,
		PayMonth:    8,
		PeriodStart: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2025, 8, 31, 0, 0, 0, 0, time.UTC),
	})
	assert.ErrorIs(t, err, domain.ErrPayrollPeriodExists)
}

func TestCreatePeriodValidatesDates(t *testing.T) {
	svc, _, _ := newPayrollFixture(t)

	_, err := svc.CreatePeriod(context.Background(), payrollTestCompanyID, &service.CreatePayrollPeriodInput{
		PayYear:     2025,
		PayMonth:    9,
		PeriodStart: time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC),
	})
	assert.ErrorIs(t, err, domain.ErrPayrollPeriodDates)

	_, err = svc.CreatePeriod(context.Background(), payrollTestCompanyID, &service.CreatePayrollPeriodInput{
		PayYear:     2025,
		PayMonth:    13,
		PeriodStart: time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC),
	})
	assert.ErrorIs(t, err, domain.ErrPayrollMonthInvalid)
}
