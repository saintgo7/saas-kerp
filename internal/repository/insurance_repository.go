package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// InsuranceWorkplaceFilter selects insurance workplaces for a list query.
type InsuranceWorkplaceFilter struct {
	CompanyID uuid.UUID
	IsActive  *bool
	Page      int
	PageSize  int
}

// EmployeeInsuranceFilter selects employee qualification records.
type EmployeeInsuranceFilter struct {
	CompanyID  uuid.UUID
	EmployeeID *uuid.UUID
	Search     string
	Page       int
	PageSize   int
}

// InsuranceContributionFilter selects monthly contribution records.
type InsuranceContributionFilter struct {
	CompanyID  uuid.UUID
	EmployeeID *uuid.UUID
	Year       *int
	Month      *int
	Page       int
	PageSize   int
}

// InsuranceReportFilter selects 신고서 records.
type InsuranceReportFilter struct {
	CompanyID  uuid.UUID
	AgencyType *domain.InsuranceAgency
	ReportType *domain.InsuranceReportType
	Status     *domain.InsuranceReportStatus
	EmployeeID *uuid.UUID
	Year       *int
	Month      *int
	Page       int
	PageSize   int
}

// EDIJobFilter selects EDI jobs.
type EDIJobFilter struct {
	CompanyID uuid.UUID
	ReportID  *uuid.UUID
	Status    *domain.EDIJobStatus
	Page      int
	PageSize  int
}

// EmployeeInsuranceRow is one employee's qualification record joined with the
// identity fields the screen needs.
type EmployeeInsuranceRow struct {
	domain.EmployeeInsurance
	EmployeeNo     string `json:"employee_no"`
	EmployeeName   string `json:"employee_name"`
	DepartmentName string `json:"department_name"`
}

// InsuranceContributionRow is one month's assessed contribution joined with the
// employee's identity.
type InsuranceContributionRow struct {
	domain.InsuranceMonthlyContribution
	EmployeeNo     string `json:"employee_no"`
	EmployeeName   string `json:"employee_name"`
	DepartmentName string `json:"department_name"`
}

// InsuranceContributionSummary totals one month's 4대보험 across the company.
// The dashboard cards read this instead of summing a page of rows in the
// browser, which would only ever total the rows that happened to be on screen.
type InsuranceContributionSummary struct {
	EmployeeCount int   `json:"employee_count"`
	NPSEmployee   int64 `json:"nps_employee"`
	NPSEmployer   int64 `json:"nps_employer"`
	NHISEmployee  int64 `json:"nhis_employee"`
	NHISEmployer  int64 `json:"nhis_employer"`
	LTCEmployee   int64 `json:"ltc_employee"`
	LTCEmployer   int64 `json:"ltc_employer"`
	EIEmployee    int64 `json:"ei_employee"`
	EIEmployer    int64 `json:"ei_employer"`
	WCIEmployer   int64 `json:"wci_employer"`
	TotalEmployee int64 `json:"total_employee"`
	TotalEmployer int64 `json:"total_employer"`
}

// InsuranceRepository is data access for 4대보험 administration.
//
// Every method takes companyID and every query filters on it.
//
// There is deliberately NO method that reads insurance_credentials' encrypted
// columns. ListCredentialStatus returns domain.InsuranceCredentialStatus, whose
// struct has no field for cert_data_enc, login_password_enc or api_secret_enc,
// so GORM cannot select them and no caller can accidentally serialise them.
type InsuranceRepository interface {
	// Workplaces (사업장)
	CreateWorkplace(ctx context.Context, workplace *domain.InsuranceWorkplace) error
	GetWorkplaceByID(ctx context.Context, companyID, id uuid.UUID) (*domain.InsuranceWorkplace, error)
	GetWorkplaceByBusinessNumber(ctx context.Context, companyID uuid.UUID, businessNumber string) (*domain.InsuranceWorkplace, error)
	ListWorkplaces(ctx context.Context, filter *InsuranceWorkplaceFilter) ([]*domain.InsuranceWorkplace, int64, error)
	UpdateWorkplace(ctx context.Context, workplace *domain.InsuranceWorkplace) error

	// Employee qualifications (자격)
	GetEmployeeInsurance(ctx context.Context, companyID, employeeID uuid.UUID) (*domain.EmployeeInsurance, error)
	UpsertEmployeeInsurance(ctx context.Context, record *domain.EmployeeInsurance) error
	ListEmployeeInsurance(ctx context.Context, filter *EmployeeInsuranceFilter) ([]*EmployeeInsuranceRow, int64, error)

	// Monthly contributions (월별 보험료)
	//
	// UpsertContribution is called from inside the payroll transaction, so a
	// recalculated payroll and its 4대보험 register move together.
	UpsertContribution(ctx context.Context, contribution *domain.InsuranceMonthlyContribution) error
	ListContributions(ctx context.Context, filter *InsuranceContributionFilter) ([]*InsuranceContributionRow, int64, error)
	SummarizeContributions(ctx context.Context, companyID uuid.UUID, year, month int) (*InsuranceContributionSummary, error)

	// Reports (신고서)
	CreateReport(ctx context.Context, report *domain.InsuranceReport) error
	GetReportByID(ctx context.Context, companyID, id uuid.UUID) (*domain.InsuranceReport, error)
	ListReports(ctx context.Context, filter *InsuranceReportFilter) ([]*domain.InsuranceReport, int64, error)
	UpdateReport(ctx context.Context, report *domain.InsuranceReport) error
	DeleteReport(ctx context.Context, companyID, id uuid.UUID) error

	// TransitionReport is the guarded status change: the source status is in
	// the WHERE clause so two concurrent submissions cannot both win.
	TransitionReport(ctx context.Context, companyID, id uuid.UUID,
		from, to domain.InsuranceReportStatus, updates map[string]interface{}) (int64, error)

	ReplaceReportItems(ctx context.Context, companyID, reportID uuid.UUID, items []domain.InsuranceReportItem) error
	ListReportItems(ctx context.Context, companyID, reportID uuid.UUID) ([]domain.InsuranceReportItem, error)

	// Credentials - status only, never the secrets.
	ListCredentialStatus(ctx context.Context, companyID uuid.UUID) ([]*domain.InsuranceCredentialStatus, error)

	// EDI jobs
	ListEDIJobs(ctx context.Context, filter *EDIJobFilter) ([]*domain.EDIJob, int64, error)
	GetEDIJobByID(ctx context.Context, companyID, id uuid.UUID) (*domain.EDIJob, error)

	// WithTransaction runs fn against a repository bound to one transaction, so
	// a report and its item rows land together.
	WithTransaction(ctx context.Context, fn func(repo InsuranceRepository) error) error
}
