package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// employeeUpdatableColumns is the exact column set an update may write.
//
// It is spelled out rather than derived so that a column added to the model
// later cannot silently become writable, and so that GORM updates zero values
// (clearing a manager, a resignation date or an address) instead of skipping
// them the way it does for a bare struct update.
//
// resident_number_enc is in the list on purpose: the service always loads the
// stored row first and carries the existing ciphertext forward, so an update
// that does not mention a resident number keeps the one on file.
var employeeUpdatableColumns = []string{
	"user_id", "employee_no", "name", "name_en", "resident_number_enc",
	"birth_date", "gender", "nationality",
	"phone", "mobile", "email", "emergency_contact", "emergency_phone",
	"zip_code", "address", "address_detail",
	"department_id", "position_id", "manager_id",
	"hire_date", "probation_end_date", "resignation_date", "resignation_reason",
	"employment_type", "contract_start_date", "contract_end_date",
	"work_location", "work_email", "work_phone",
	"status", "updated_by", "updated_at",
}

// employeeSortColumns maps the API's sort keys to real columns. A client-chosen
// value never reaches the ORDER BY clause directly.
var employeeSortColumns = map[string]string{
	"employee_no": "employee_no",
	"name":        "name",
	"hire_date":   "hire_date",
	"status":      "status",
}

// employeeRepositoryGorm implements EmployeeRepository using GORM.
type employeeRepositoryGorm struct {
	db *gorm.DB
}

// NewEmployeeRepositoryGorm creates a new EmployeeRepository with GORM.
func NewEmployeeRepositoryGorm(db *gorm.DB) EmployeeRepository {
	return &employeeRepositoryGorm{db: db}
}

// Create inserts an employee.
func (r *employeeRepositoryGorm) Create(ctx context.Context, employee *domain.Employee) error {
	// Omit the associations: Department, Position and Manager are read-only
	// projections here, and letting GORM upsert them would write another
	// tenant's row through a nested object supplied by the client.
	return r.db.WithContext(ctx).
		Omit("Department", "Position", "Manager").
		Create(employee).Error
}

// GetByID retrieves one employee of one company, with its department, position
// and manager.
func (r *employeeRepositoryGorm) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.scoped(ctx, companyID).
		Where("employees.id = ?", id).
		First(&employee).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrEmployeeNotFound
		}
		return nil, err
	}
	return &employee, nil
}

// GetByEmployeeNo retrieves one employee of one company by employee number.
func (r *employeeRepositoryGorm) GetByEmployeeNo(ctx context.Context, companyID uuid.UUID, employeeNo string) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.scoped(ctx, companyID).
		Where("employees.employee_no = ?", employeeNo).
		First(&employee).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrEmployeeNotFound
		}
		return nil, err
	}
	return &employee, nil
}

// scoped builds a query restricted to one company, with the related rows
// preloaded under the same restriction.
//
// The preload conditions matter: GORM's default preload is a bare
// `WHERE id IN (...)`, which reaches across tenants if a foreign key ever
// points at another company's row. Row Level Security would also stop that,
// but the application must not depend on it alone.
func (r *employeeRepositoryGorm) scoped(ctx context.Context, companyID uuid.UUID) *gorm.DB {
	return r.db.WithContext(ctx).
		Where("employees.company_id = ?", companyID).
		Preload("Department", "company_id = ?", companyID).
		Preload("Position", "company_id = ?", companyID).
		Preload("Manager", "company_id = ?", companyID)
}

// List retrieves employees with filtering and pagination.
func (r *employeeRepositoryGorm) List(ctx context.Context, filter *EmployeeFilter) ([]domain.Employee, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Employee{}).
		Where("employees.company_id = ?", filter.CompanyID)

	if filter.DepartmentID != nil {
		query = query.Where("employees.department_id = ?", *filter.DepartmentID)
	}
	if filter.PositionID != nil {
		query = query.Where("employees.position_id = ?", *filter.PositionID)
	}
	if filter.ManagerID != nil {
		query = query.Where("employees.manager_id = ?", *filter.ManagerID)
	}
	if filter.Status != "" {
		query = query.Where("employees.status = ?", string(filter.Status))
	}
	if filter.EmploymentType != "" {
		query = query.Where("employees.employment_type = ?", string(filter.EmploymentType))
	}
	if filter.HiredFrom != nil {
		query = query.Where("employees.hire_date >= ?", *filter.HiredFrom)
	}
	if filter.HiredTo != nil {
		query = query.Where("employees.hire_date <= ?", *filter.HiredTo)
	}
	if filter.SearchTerm != "" {
		pattern := "%" + filter.SearchTerm + "%"
		query = query.Where(
			"employees.employee_no ILIKE ? OR employees.name ILIKE ? OR employees.email ILIKE ? OR employees.work_email ILIKE ?",
			pattern, pattern, pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	column, ok := employeeSortColumns[filter.SortBy]
	if !ok {
		column = "employee_no"
	}
	direction := "ASC"
	if filter.SortDir == "desc" {
		direction = "DESC"
	}

	query = query.
		Preload("Department", "company_id = ?", filter.CompanyID).
		Preload("Position", "company_id = ?", filter.CompanyID).
		Order("employees." + column + " " + direction)

	var employees []domain.Employee
	if err := query.Find(&employees).Error; err != nil {
		return nil, 0, err
	}
	return employees, total, nil
}

// Update writes an employee back. The company_id predicate keeps a caller from
// updating another tenant's row by supplying its id.
func (r *employeeRepositoryGorm) Update(ctx context.Context, employee *domain.Employee) error {
	result := r.db.WithContext(ctx).Model(&domain.Employee{}).
		Where("id = ? AND company_id = ?", employee.ID, employee.CompanyID).
		Select(employeeUpdatableColumns).
		Updates(employee)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrEmployeeNotFound
	}
	return nil
}

// Delete soft-deletes an employee. domain.Employee carries gorm.DeletedAt, so
// GORM turns this into `UPDATE employees SET deleted_at = now()` and the row -
// with every payroll and insurance record hanging off it - survives.
func (r *employeeRepositoryGorm) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		Delete(&domain.Employee{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrEmployeeNotFound
	}
	return nil
}

// ExistsByEmployeeNo reports whether the company already uses this number.
func (r *employeeRepositoryGorm) ExistsByEmployeeNo(ctx context.Context, companyID uuid.UUID, employeeNo string, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).Model(&domain.Employee{}).
		Where("company_id = ? AND employee_no = ?", companyID, employeeNo)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// Exists reports whether the company has a live employee with this id.
func (r *employeeRepositoryGorm) Exists(ctx context.Context, companyID, id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Employee{}).
		Where("company_id = ? AND id = ?", companyID, id).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DepartmentExists reports whether the company owns this department.
func (r *employeeRepositoryGorm) DepartmentExists(ctx context.Context, companyID, departmentID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Department{}).
		Where("company_id = ? AND id = ?", companyID, departmentID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetManagerID returns the manager of one employee.
func (r *employeeRepositoryGorm) GetManagerID(ctx context.Context, companyID, id uuid.UUID) (*uuid.UUID, error) {
	var row struct {
		ManagerID *uuid.UUID
	}
	err := r.db.WithContext(ctx).Model(&domain.Employee{}).
		Select("manager_id").
		Where("company_id = ? AND id = ?", companyID, id).
		Take(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrEmployeeNotFound
		}
		return nil, err
	}
	return row.ManagerID, nil
}

// CountRetainedRecords counts every row whose foreign key to employees is
// ON DELETE RESTRICT, plus the leave records.
func (r *employeeRepositoryGorm) CountRetainedRecords(ctx context.Context, companyID, id uuid.UUID) (EmployeeRetainedRecords, error) {
	var counts EmployeeRetainedRecords

	tables := []struct {
		name string
		into *int64
	}{
		{"payrolls", &counts.Payrolls},
		{"employee_salaries", &counts.Salaries},
		{"employee_insurance", &counts.Insurance},
		{"insurance_monthly_contributions", &counts.InsuranceContributions},
		{"employee_leaves", &counts.Leaves},
	}

	for _, t := range tables {
		if err := r.db.WithContext(ctx).
			Table(t.name).
			Where("company_id = ? AND employee_id = ?", companyID, id).
			Count(t.into).Error; err != nil {
			return EmployeeRetainedRecords{}, err
		}
	}

	return counts, nil
}

// CountDirectReports counts the live employees managed by id.
func (r *employeeRepositoryGorm) CountDirectReports(ctx context.Context, companyID, id uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Employee{}).
		Where("company_id = ? AND manager_id = ?", companyID, id).
		Count(&count).Error
	return count, err
}

// CountByStatus returns the per-status headcount in one round trip.
func (r *employeeRepositoryGorm) CountByStatus(ctx context.Context, companyID uuid.UUID) (EmployeeStatusCounts, error) {
	var rows []struct {
		Status string
		Count  int64
	}

	err := r.db.WithContext(ctx).Model(&domain.Employee{}).
		Select("status, COUNT(*) AS count").
		Where("company_id = ?", companyID).
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return EmployeeStatusCounts{}, err
	}

	var counts EmployeeStatusCounts
	for _, row := range rows {
		counts.Total += row.Count
		switch domain.EmployeeStatus(row.Status) {
		case domain.EmployeeStatusActive:
			counts.Active = row.Count
		case domain.EmployeeStatusOnLeave:
			counts.OnLeave = row.Count
		case domain.EmployeeStatusResigned:
			counts.Resigned = row.Count
		case domain.EmployeeStatusTerminated:
			counts.Terminated = row.Count
		}
	}
	return counts, nil
}
