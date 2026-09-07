package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// EmployeeFilter defines filter criteria for listing employees.
//
// CompanyID is mandatory. Every statement this repository issues carries a
// company_id predicate; a zero CompanyID therefore matches nothing, which is
// the safe direction for a multi-tenant query.
type EmployeeFilter struct {
	CompanyID      uuid.UUID
	DepartmentID   *uuid.UUID
	PositionID     *uuid.UUID
	ManagerID      *uuid.UUID
	Status         domain.EmployeeStatus
	EmploymentType domain.EmploymentType
	SearchTerm     string // matched against employee_no, name, email, work_email
	HiredFrom      *time.Time
	HiredTo        *time.Time
	Page           int
	PageSize       int
	SortBy         string // employee_no | name | hire_date
	SortDir        string // asc | desc
}

// EmployeeStatusCounts is the per-status headcount of one company.
type EmployeeStatusCounts struct {
	Total      int64 `json:"total"`
	Active     int64 `json:"active"`
	OnLeave    int64 `json:"on_leave"`
	Resigned   int64 `json:"resigned"`
	Terminated int64 `json:"terminated"`
}

// EmployeeRetainedRecords counts the rows that must outlive the employee.
//
// Every foreign key listed here was changed to ON DELETE RESTRICT in
// db/migrations/000019_retention_and_amount_checks: the wage ledger has to be
// kept for three years under the Labor Standards Act, so a person who has ever
// been paid is separated (resigned/terminated), never deleted.
type EmployeeRetainedRecords struct {
	Payrolls               int64
	Salaries               int64
	Insurance              int64
	InsuranceContributions int64
	Leaves                 int64
}

// Any reports whether anything at all references the employee.
func (r EmployeeRetainedRecords) Any() bool {
	return r.Payrolls > 0 || r.Salaries > 0 || r.Insurance > 0 ||
		r.InsuranceContributions > 0 || r.Leaves > 0
}

// EmployeeRepository defines the interface for employee data access.
type EmployeeRepository interface {
	// CRUD operations
	Create(ctx context.Context, employee *domain.Employee) error
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Employee, error)
	GetByEmployeeNo(ctx context.Context, companyID uuid.UUID, employeeNo string) (*domain.Employee, error)
	List(ctx context.Context, filter *EmployeeFilter) ([]domain.Employee, int64, error)
	Update(ctx context.Context, employee *domain.Employee) error

	// Delete soft-deletes the employee (employees.deleted_at). It is never a
	// hard delete: the retention foreign keys would refuse one, and the
	// partial unique index on (company_id, employee_no) already accounts for
	// soft-deleted rows so the number can be reused on re-hire.
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Validation
	ExistsByEmployeeNo(ctx context.Context, companyID uuid.UUID, employeeNo string, excludeID *uuid.UUID) (bool, error)
	Exists(ctx context.Context, companyID, id uuid.UUID) (bool, error)

	// DepartmentExists checks a department id against the SAME company, so an
	// employee can never be filed under another tenant's department. It lives
	// here rather than on DepartmentRepository because that interface reports
	// "not found" as an opaque error that errors.Is cannot match.
	DepartmentExists(ctx context.Context, companyID, departmentID uuid.UUID) (bool, error)

	// GetManagerID returns the manager of one employee, so the service can
	// walk the reporting line and reject a cycle before it is stored.
	GetManagerID(ctx context.Context, companyID, id uuid.UUID) (*uuid.UUID, error)

	// CountRetainedRecords counts everything that a delete must not destroy.
	CountRetainedRecords(ctx context.Context, companyID, id uuid.UUID) (EmployeeRetainedRecords, error)

	// CountDirectReports counts the live employees whose manager is id.
	CountDirectReports(ctx context.Context, companyID, id uuid.UUID) (int64, error)

	// Statistics
	CountByStatus(ctx context.Context, companyID uuid.UUID) (EmployeeStatusCounts, error)
}
