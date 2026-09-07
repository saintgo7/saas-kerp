package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// insuranceRepositoryGorm implements InsuranceRepository with GORM.
type insuranceRepositoryGorm struct {
	db *gorm.DB
}

// NewInsuranceRepository creates a GORM-backed InsuranceRepository.
func NewInsuranceRepository(db *gorm.DB) InsuranceRepository {
	return &insuranceRepositoryGorm{db: db}
}

// ---------------------------------------------------------------------------
// Workplaces
// ---------------------------------------------------------------------------

// CreateWorkplace inserts an insurance workplace registration.
func (r *insuranceRepositoryGorm) CreateWorkplace(ctx context.Context, workplace *domain.InsuranceWorkplace) error {
	return r.db.WithContext(ctx).Create(workplace).Error
}

// GetWorkplaceByID loads one workplace belonging to the company.
func (r *insuranceRepositoryGorm) GetWorkplaceByID(ctx context.Context, companyID, id uuid.UUID) (*domain.InsuranceWorkplace, error) {
	var workplace domain.InsuranceWorkplace
	err := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		First(&workplace).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrInsuranceWorkplaceNotFound
		}
		return nil, err
	}
	return &workplace, nil
}

// GetWorkplaceByBusinessNumber loads a workplace by its 사업자등록번호.
func (r *insuranceRepositoryGorm) GetWorkplaceByBusinessNumber(ctx context.Context, companyID uuid.UUID, businessNumber string) (*domain.InsuranceWorkplace, error) {
	var workplace domain.InsuranceWorkplace
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND business_number = ?", companyID, businessNumber).
		First(&workplace).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrInsuranceWorkplaceNotFound
		}
		return nil, err
	}
	return &workplace, nil
}

// ListWorkplaces returns a page of workplace registrations.
func (r *insuranceRepositoryGorm) ListWorkplaces(ctx context.Context, filter *InsuranceWorkplaceFilter) ([]*domain.InsuranceWorkplace, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.InsuranceWorkplace{}).
		Where("company_id = ?", filter.CompanyID)
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var workplaces []*domain.InsuranceWorkplace
	err := query.
		Order("workplace_name ASC").
		Offset(offsetOf(filter.Page, filter.PageSize)).
		Limit(filter.PageSize).
		Find(&workplaces).Error
	if err != nil {
		return nil, 0, err
	}
	return workplaces, total, nil
}

// UpdateWorkplace persists a modified workplace registration.
func (r *insuranceRepositoryGorm) UpdateWorkplace(ctx context.Context, workplace *domain.InsuranceWorkplace) error {
	return r.db.WithContext(ctx).
		Model(&domain.InsuranceWorkplace{}).
		Where("id = ? AND company_id = ?", workplace.ID, workplace.CompanyID).
		Save(workplace).Error
}

// ---------------------------------------------------------------------------
// Employee qualifications
// ---------------------------------------------------------------------------

// GetEmployeeInsurance loads one employee's qualification record.
func (r *insuranceRepositoryGorm) GetEmployeeInsurance(ctx context.Context, companyID, employeeID uuid.UUID) (*domain.EmployeeInsurance, error) {
	var record domain.EmployeeInsurance
	err := r.db.WithContext(ctx).
		Where("employee_id = ? AND company_id = ?", employeeID, companyID).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrEmployeeInsuranceNotFound
		}
		return nil, err
	}
	return &record, nil
}

// employeeInsuranceUpsertColumns are the columns an upsert overwrites. company_id
// and employee_id are the identity and are never among them.
var employeeInsuranceUpsertColumns = []clause.Column{
	{Name: "nps_qualified"}, {Name: "nhis_qualified"}, {Name: "ei_qualified"}, {Name: "wci_qualified"},
	{Name: "nps_acquisition_date"}, {Name: "nhis_acquisition_date"}, {Name: "ei_acquisition_date"},
	{Name: "nps_loss_date"}, {Name: "nhis_loss_date"}, {Name: "ei_loss_date"},
	{Name: "nps_standard_remuneration"}, {Name: "nhis_standard_remuneration"},
	{Name: "nhis_grade"}, {Name: "dependents_count"},
	{Name: "nps_reduced"}, {Name: "nhis_reduced"},
	{Name: "nps_reduction_type"}, {Name: "nhis_reduction_type"},
	{Name: "updated_at"},
}

// UpsertEmployeeInsurance inserts or replaces one employee's qualification
// record. employee_insurance is UNIQUE(employee_id), so the conflict target is
// that column.
func (r *insuranceRepositoryGorm) UpsertEmployeeInsurance(ctx context.Context, record *domain.EmployeeInsurance) error {
	record.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "employee_id"}},
			DoUpdates: clause.AssignmentColumns(columnNames(employeeInsuranceUpsertColumns)),
		}).
		Create(record).Error
}

// employeeInsuranceSelect joins the qualification record with employee identity.
const employeeInsuranceSelect = `
	ei.*,
	COALESCE(e.employee_no, '') AS employee_no,
	COALESCE(e.name, '')        AS employee_name,
	COALESCE(d.name, '')        AS department_name`

// listEmployeeInsuranceQuery builds the filtered, joined base query.
func (r *insuranceRepositoryGorm) listEmployeeInsuranceQuery(ctx context.Context, filter *EmployeeInsuranceFilter) *gorm.DB {
	query := r.db.WithContext(ctx).
		Table("employee_insurance AS ei").
		Joins("LEFT JOIN employees AS e ON e.id = ei.employee_id AND e.company_id = ei.company_id").
		Joins("LEFT JOIN departments AS d ON d.id = e.department_id AND d.company_id = ei.company_id").
		Where("ei.company_id = ?", filter.CompanyID)

	if filter.EmployeeID != nil {
		query = query.Where("ei.employee_id = ?", *filter.EmployeeID)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		query = query.Where("e.name ILIKE ? OR e.employee_no ILIKE ?", like, like)
	}
	return query
}

// ListEmployeeInsurance returns a page of qualification records.
func (r *insuranceRepositoryGorm) ListEmployeeInsurance(ctx context.Context, filter *EmployeeInsuranceFilter) ([]*EmployeeInsuranceRow, int64, error) {
	var total int64
	if err := r.listEmployeeInsuranceQuery(ctx, filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []*EmployeeInsuranceRow
	err := r.listEmployeeInsuranceQuery(ctx, filter).
		Select(employeeInsuranceSelect).
		Order("e.employee_no ASC").
		Offset(offsetOf(filter.Page, filter.PageSize)).
		Limit(filter.PageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ---------------------------------------------------------------------------
// Monthly contributions
// ---------------------------------------------------------------------------

// contributionUpsertColumns are the columns a recalculation overwrites.
//
// total_employee and total_employer are GENERATED ALWAYS columns and are
// absent on purpose: naming them in an UPDATE makes PostgreSQL reject the
// statement.
var contributionUpsertColumns = []clause.Column{
	{Name: "nps_base"}, {Name: "nps_employee"}, {Name: "nps_employer"},
	{Name: "nhis_base"}, {Name: "nhis_employee"}, {Name: "nhis_employer"},
	{Name: "nhis_ltc_employee"}, {Name: "nhis_ltc_employer"},
	{Name: "ei_base"}, {Name: "ei_employee"}, {Name: "ei_employer"},
	{Name: "wci_base"}, {Name: "wci_employer"},
	{Name: "payroll_id"}, {Name: "updated_at"},
}

// UpsertContribution writes one employee's assessed contribution for one month.
//
// The conflict target is the natural key
// (company_id, employee_id, contribution_year, contribution_month), so
// recalculating a payroll updates the existing row instead of accumulating a
// second one for the same month.
func (r *insuranceRepositoryGorm) UpsertContribution(ctx context.Context, contribution *domain.InsuranceMonthlyContribution) error {
	contribution.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "company_id"}, {Name: "employee_id"},
				{Name: "contribution_year"}, {Name: "contribution_month"},
			},
			DoUpdates: clause.AssignmentColumns(columnNames(contributionUpsertColumns)),
		}).
		Create(contribution).Error
}

// contributionSelect joins a contribution row with employee identity.
const contributionSelect = `
	c.*,
	COALESCE(e.employee_no, '') AS employee_no,
	COALESCE(e.name, '')        AS employee_name,
	COALESCE(d.name, '')        AS department_name`

// listContributionsQuery builds the filtered, joined base query.
func (r *insuranceRepositoryGorm) listContributionsQuery(ctx context.Context, filter *InsuranceContributionFilter) *gorm.DB {
	query := r.db.WithContext(ctx).
		Table("insurance_monthly_contributions AS c").
		Joins("LEFT JOIN employees AS e ON e.id = c.employee_id AND e.company_id = c.company_id").
		Joins("LEFT JOIN departments AS d ON d.id = e.department_id AND d.company_id = c.company_id").
		Where("c.company_id = ?", filter.CompanyID)

	if filter.EmployeeID != nil {
		query = query.Where("c.employee_id = ?", *filter.EmployeeID)
	}
	if filter.Year != nil {
		query = query.Where("c.contribution_year = ?", *filter.Year)
	}
	if filter.Month != nil {
		query = query.Where("c.contribution_month = ?", *filter.Month)
	}
	return query
}

// ListContributions returns a page of monthly contribution records.
func (r *insuranceRepositoryGorm) ListContributions(ctx context.Context, filter *InsuranceContributionFilter) ([]*InsuranceContributionRow, int64, error) {
	var total int64
	if err := r.listContributionsQuery(ctx, filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []*InsuranceContributionRow
	err := r.listContributionsQuery(ctx, filter).
		Select(contributionSelect).
		Order("c.contribution_year DESC, c.contribution_month DESC, e.employee_no ASC").
		Offset(offsetOf(filter.Page, filter.PageSize)).
		Limit(filter.PageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// SummarizeContributions totals one month across the whole company.
func (r *insuranceRepositoryGorm) SummarizeContributions(ctx context.Context, companyID uuid.UUID, year, month int) (*InsuranceContributionSummary, error) {
	var summary InsuranceContributionSummary
	err := r.db.WithContext(ctx).
		Model(&domain.InsuranceMonthlyContribution{}).
		Select(`COUNT(*) AS employee_count,
			COALESCE(SUM(nps_employee), 0)      AS nps_employee,
			COALESCE(SUM(nps_employer), 0)      AS nps_employer,
			COALESCE(SUM(nhis_employee), 0)     AS nhis_employee,
			COALESCE(SUM(nhis_employer), 0)     AS nhis_employer,
			COALESCE(SUM(nhis_ltc_employee), 0) AS ltc_employee,
			COALESCE(SUM(nhis_ltc_employer), 0) AS ltc_employer,
			COALESCE(SUM(ei_employee), 0)       AS ei_employee,
			COALESCE(SUM(ei_employer), 0)       AS ei_employer,
			COALESCE(SUM(wci_employer), 0)      AS wci_employer,
			COALESCE(SUM(total_employee), 0)    AS total_employee,
			COALESCE(SUM(total_employer), 0)    AS total_employer`).
		Where("company_id = ? AND contribution_year = ? AND contribution_month = ?", companyID, year, month).
		Scan(&summary).Error
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

// ---------------------------------------------------------------------------
// Reports
// ---------------------------------------------------------------------------

// CreateReport inserts a 신고서.
func (r *insuranceRepositoryGorm) CreateReport(ctx context.Context, report *domain.InsuranceReport) error {
	return r.db.WithContext(ctx).Create(report).Error
}

// GetReportByID loads one report belonging to the company.
func (r *insuranceRepositoryGorm) GetReportByID(ctx context.Context, companyID, id uuid.UUID) (*domain.InsuranceReport, error) {
	var report domain.InsuranceReport
	err := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		First(&report).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrInsuranceReportNotFound
		}
		return nil, err
	}
	return &report, nil
}

// ListReports returns a page of 신고서 records.
func (r *insuranceRepositoryGorm) ListReports(ctx context.Context, filter *InsuranceReportFilter) ([]*domain.InsuranceReport, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.InsuranceReport{}).
		Where("company_id = ?", filter.CompanyID)

	if filter.AgencyType != nil {
		query = query.Where("agency_type = ?", *filter.AgencyType)
	}
	if filter.ReportType != nil {
		query = query.Where("report_type = ?", *filter.ReportType)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.EmployeeID != nil {
		query = query.Where("employee_id = ?", *filter.EmployeeID)
	}
	if filter.Year != nil {
		query = query.Where("report_year = ?", *filter.Year)
	}
	if filter.Month != nil {
		query = query.Where("report_month = ?", *filter.Month)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var reports []*domain.InsuranceReport
	err := query.
		Order("created_at DESC").
		Offset(offsetOf(filter.Page, filter.PageSize)).
		Limit(filter.PageSize).
		Find(&reports).Error
	if err != nil {
		return nil, 0, err
	}
	return reports, total, nil
}

// UpdateReport persists a modified report.
func (r *insuranceRepositoryGorm) UpdateReport(ctx context.Context, report *domain.InsuranceReport) error {
	return r.db.WithContext(ctx).
		Model(&domain.InsuranceReport{}).
		Where("id = ? AND company_id = ?", report.ID, report.CompanyID).
		Save(report).Error
}

// DeleteReport removes a report and, through the ON DELETE CASCADE on
// insurance_report_items.report_id, its lines.
func (r *insuranceRepositoryGorm) DeleteReport(ctx context.Context, companyID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		Delete(&domain.InsuranceReport{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrInsuranceReportNotFound
	}
	return nil
}

// TransitionReport applies a guarded status change.
func (r *insuranceRepositoryGorm) TransitionReport(ctx context.Context, companyID, id uuid.UUID,
	from, to domain.InsuranceReportStatus, updates map[string]interface{}) (int64, error) {

	patch := make(map[string]interface{}, len(updates)+2)
	for k, v := range updates {
		patch[k] = v
	}
	patch["status"] = to
	patch["updated_at"] = time.Now()

	res := r.db.WithContext(ctx).
		Model(&domain.InsuranceReport{}).
		Where("id = ? AND company_id = ? AND status = ?", id, companyID, from).
		Updates(patch)
	return res.RowsAffected, res.Error
}

// ReplaceReportItems swaps a report's lines for a new set. The caller runs it
// inside WithTransaction together with the header write.
func (r *insuranceRepositoryGorm) ReplaceReportItems(ctx context.Context, companyID, reportID uuid.UUID, items []domain.InsuranceReportItem) error {
	db := r.db.WithContext(ctx)

	// insurance_report_items has no company_id column of its own, so the
	// tenant predicate goes through the parent report. Deleting by report_id
	// alone would let a forged id clear another tenant's lines.
	var owned int64
	if err := db.Model(&domain.InsuranceReport{}).
		Where("id = ? AND company_id = ?", reportID, companyID).
		Count(&owned).Error; err != nil {
		return err
	}
	if owned == 0 {
		return domain.ErrInsuranceReportNotFound
	}

	if err := db.Where("report_id = ?", reportID).
		Delete(&domain.InsuranceReportItem{}).Error; err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}

	for i := range items {
		items[i].ReportID = reportID
	}
	return db.Create(&items).Error
}

// ListReportItems returns a report's lines in filing order.
func (r *insuranceRepositoryGorm) ListReportItems(ctx context.Context, companyID, reportID uuid.UUID) ([]domain.InsuranceReportItem, error) {
	var items []domain.InsuranceReportItem
	err := r.db.WithContext(ctx).
		Table("insurance_report_items AS i").
		Joins("JOIN insurance_reports AS rp ON rp.id = i.report_id").
		Select("i.*").
		Where("i.report_id = ? AND rp.company_id = ?", reportID, companyID).
		Order("i.line_no ASC").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// ---------------------------------------------------------------------------
// Credentials (status only)
// ---------------------------------------------------------------------------

// ListCredentialStatus reports which agencies have credentials registered and
// whether they are still valid. It cannot return the credentials themselves:
// domain.InsuranceCredentialStatus has no field for any of the *_enc columns.
func (r *insuranceRepositoryGorm) ListCredentialStatus(ctx context.Context, companyID uuid.UUID) ([]*domain.InsuranceCredentialStatus, error) {
	var rows []*domain.InsuranceCredentialStatus
	err := r.db.WithContext(ctx).
		Model(&domain.InsuranceCredentialStatus{}).
		Where("company_id = ?", companyID).
		Order("agency_type ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ---------------------------------------------------------------------------
// EDI jobs
// ---------------------------------------------------------------------------

// ListEDIJobs returns a page of EDI jobs.
func (r *insuranceRepositoryGorm) ListEDIJobs(ctx context.Context, filter *EDIJobFilter) ([]*domain.EDIJob, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.EDIJob{}).
		Where("company_id = ?", filter.CompanyID)
	if filter.ReportID != nil {
		query = query.Where("report_id = ?", *filter.ReportID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var jobs []*domain.EDIJob
	err := query.
		Order("created_at DESC").
		Offset(offsetOf(filter.Page, filter.PageSize)).
		Limit(filter.PageSize).
		Find(&jobs).Error
	if err != nil {
		return nil, 0, err
	}
	return jobs, total, nil
}

// GetEDIJobByID loads one EDI job belonging to the company.
func (r *insuranceRepositoryGorm) GetEDIJobByID(ctx context.Context, companyID, id uuid.UUID) (*domain.EDIJob, error) {
	var job domain.EDIJob
	err := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		First(&job).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrInsuranceReportNotFound
		}
		return nil, err
	}
	return &job, nil
}

// ---------------------------------------------------------------------------
// Transactions
// ---------------------------------------------------------------------------

// WithTransaction runs fn against a repository bound to one transaction.
func (r *insuranceRepositoryGorm) WithTransaction(ctx context.Context, fn func(repo InsuranceRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&insuranceRepositoryGorm{db: tx})
	})
}

// columnNames flattens clause columns into the plain names AssignmentColumns
// wants.
func columnNames(cols []clause.Column) []string {
	names := make([]string, 0, len(cols))
	for _, c := range cols {
		names = append(names, c.Name)
	}
	return names
}
