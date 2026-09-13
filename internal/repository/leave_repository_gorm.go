package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// ---------------------------------------------------------------------------
// Leave types
// ---------------------------------------------------------------------------

// leaveTypeRepositoryGorm implements LeaveTypeRepository using GORM.
type leaveTypeRepositoryGorm struct {
	db *gorm.DB
}

// NewLeaveTypeRepositoryGorm creates a new LeaveTypeRepository with GORM.
func NewLeaveTypeRepositoryGorm(db *gorm.DB) LeaveTypeRepository {
	return &leaveTypeRepositoryGorm{db: db}
}

// Create inserts a leave type.
func (r *leaveTypeRepositoryGorm) Create(ctx context.Context, leaveType *domain.LeaveType) error {
	return r.db.WithContext(ctx).Create(leaveType).Error
}

// GetByID retrieves one leave type of one company.
func (r *leaveTypeRepositoryGorm) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.LeaveType, error) {
	var leaveType domain.LeaveType
	err := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		First(&leaveType).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrLeaveTypeNotFound
		}
		return nil, err
	}
	return &leaveType, nil
}

// GetByCode retrieves one leave type of one company by code.
func (r *leaveTypeRepositoryGorm) GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.LeaveType, error) {
	var leaveType domain.LeaveType
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND code = ?", companyID, code).
		First(&leaveType).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrLeaveTypeNotFound
		}
		return nil, err
	}
	return &leaveType, nil
}

// List retrieves leave types with filtering and pagination.
func (r *leaveTypeRepositoryGorm) List(ctx context.Context, filter *LeaveTypeFilter) ([]domain.LeaveType, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.LeaveType{}).
		Where("company_id = ?", filter.CompanyID)

	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.SearchTerm != "" {
		pattern := "%" + filter.SearchTerm + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var types []domain.LeaveType
	if err := query.Order("code ASC").Find(&types).Error; err != nil {
		return nil, 0, err
	}
	return types, total, nil
}

// Update writes a leave type back, scoped to its company.
func (r *leaveTypeRepositoryGorm) Update(ctx context.Context, leaveType *domain.LeaveType) error {
	result := r.db.WithContext(ctx).Model(&domain.LeaveType{}).
		Where("id = ? AND company_id = ?", leaveType.ID, leaveType.CompanyID).
		Select("code", "name", "is_paid", "default_days", "max_carryover_days",
			"requires_approval", "is_active", "updated_at").
		Updates(leaveType)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrLeaveTypeNotFound
	}
	return nil
}

// Delete removes a leave type (the table has no deleted_at).
func (r *leaveTypeRepositoryGorm) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		Delete(&domain.LeaveType{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrLeaveTypeNotFound
	}
	return nil
}

// ExistsByCode reports whether the company already uses this leave type code.
func (r *leaveTypeRepositoryGorm) ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).Model(&domain.LeaveType{}).
		Where("company_id = ? AND code = ?", companyID, code)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountReferences counts the leave requests and balances that use the type.
func (r *leaveTypeRepositoryGorm) CountReferences(ctx context.Context, companyID, id uuid.UUID) (int64, error) {
	var leaves, balances int64

	if err := r.db.WithContext(ctx).Model(&domain.EmployeeLeave{}).
		Where("company_id = ? AND leave_type_id = ?", companyID, id).
		Count(&leaves).Error; err != nil {
		return 0, err
	}
	if err := r.db.WithContext(ctx).Model(&domain.EmployeeLeaveBalance{}).
		Where("company_id = ? AND leave_type_id = ?", companyID, id).
		Count(&balances).Error; err != nil {
		return 0, err
	}

	return leaves + balances, nil
}

// ---------------------------------------------------------------------------
// Leave requests and balances
// ---------------------------------------------------------------------------

// leaveRepositoryGorm implements LeaveRepository using GORM.
type leaveRepositoryGorm struct {
	db *gorm.DB
}

// NewLeaveRepositoryGorm creates a new LeaveRepository with GORM.
func NewLeaveRepositoryGorm(db *gorm.DB) LeaveRepository {
	return &leaveRepositoryGorm{db: db}
}

// Create inserts a leave request.
func (r *leaveRepositoryGorm) Create(ctx context.Context, leave *domain.EmployeeLeave) error {
	return r.db.WithContext(ctx).
		Omit("Employee", "LeaveType").
		Create(leave).Error
}

// GetByID retrieves one leave request of one company.
func (r *leaveRepositoryGorm) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.EmployeeLeave, error) {
	var leave domain.EmployeeLeave
	err := r.db.WithContext(ctx).
		Where("employee_leaves.id = ? AND employee_leaves.company_id = ?", id, companyID).
		Preload("Employee", "company_id = ?", companyID).
		Preload("LeaveType", "company_id = ?", companyID).
		First(&leave).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrLeaveNotFound
		}
		return nil, err
	}
	return &leave, nil
}

// List retrieves leave requests with filtering and pagination.
func (r *leaveRepositoryGorm) List(ctx context.Context, filter *LeaveFilter) ([]domain.EmployeeLeave, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.EmployeeLeave{}).
		Where("employee_leaves.company_id = ?", filter.CompanyID)

	if filter.EmployeeID != nil {
		query = query.Where("employee_leaves.employee_id = ?", *filter.EmployeeID)
	}
	if filter.LeaveTypeID != nil {
		query = query.Where("employee_leaves.leave_type_id = ?", *filter.LeaveTypeID)
	}
	if filter.Status != "" {
		query = query.Where("employee_leaves.status = ?", string(filter.Status))
	}
	// Overlap, not containment: a request that starts before the window and
	// ends inside it is part of that window.
	if filter.DateFrom != nil {
		query = query.Where("employee_leaves.end_date >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("employee_leaves.start_date <= ?", *filter.DateTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	query = query.
		Preload("Employee", "company_id = ?", filter.CompanyID).
		Preload("LeaveType", "company_id = ?", filter.CompanyID).
		Order("employee_leaves.start_date DESC")

	var leaves []domain.EmployeeLeave
	if err := query.Find(&leaves).Error; err != nil {
		return nil, 0, err
	}
	return leaves, total, nil
}

// Update writes a leave request back, scoped to its company.
func (r *leaveRepositoryGorm) Update(ctx context.Context, leave *domain.EmployeeLeave) error {
	result := r.db.WithContext(ctx).Model(&domain.EmployeeLeave{}).
		Where("id = ? AND company_id = ?", leave.ID, leave.CompanyID).
		Select("leave_type_id", "start_date", "end_date", "days", "reason",
			"status", "approved_at", "approved_by", "rejection_reason", "updated_at").
		Updates(leave)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrLeaveNotFound
	}
	return nil
}

// Delete removes a leave request (the table has no deleted_at).
func (r *leaveRepositoryGorm) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		Delete(&domain.EmployeeLeave{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrLeaveNotFound
	}
	return nil
}

// CountOverlapping counts the pending or approved requests of one employee
// that overlap the given range.
func (r *leaveRepositoryGorm) CountOverlapping(ctx context.Context, companyID, employeeID uuid.UUID, start, end time.Time, excludeID *uuid.UUID) (int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.EmployeeLeave{}).
		Where("company_id = ? AND employee_id = ?", companyID, employeeID).
		Where("status IN ?", []string{string(domain.LeaveStatusPending), string(domain.LeaveStatusApproved)}).
		Where("start_date <= ? AND end_date >= ?", end, start)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	err := query.Count(&count).Error
	return count, err
}

// GetBalance retrieves one balance row.
func (r *leaveRepositoryGorm) GetBalance(ctx context.Context, companyID, employeeID, leaveTypeID uuid.UUID, fiscalYear int) (*domain.EmployeeLeaveBalance, error) {
	var balance domain.EmployeeLeaveBalance
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND employee_id = ? AND leave_type_id = ? AND fiscal_year = ?",
			companyID, employeeID, leaveTypeID, fiscalYear).
		First(&balance).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrLeaveBalanceNotFound
		}
		return nil, err
	}
	return &balance, nil
}

// ListBalances retrieves balance rows with filtering and pagination.
func (r *leaveRepositoryGorm) ListBalances(ctx context.Context, filter *LeaveBalanceFilter) ([]domain.EmployeeLeaveBalance, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.EmployeeLeaveBalance{}).
		Where("employee_leave_balances.company_id = ?", filter.CompanyID)

	if filter.EmployeeID != nil {
		query = query.Where("employee_leave_balances.employee_id = ?", *filter.EmployeeID)
	}
	if filter.LeaveTypeID != nil {
		query = query.Where("employee_leave_balances.leave_type_id = ?", *filter.LeaveTypeID)
	}
	if filter.FiscalYear != nil {
		query = query.Where("employee_leave_balances.fiscal_year = ?", *filter.FiscalYear)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	query = query.
		Preload("Employee", "company_id = ?", filter.CompanyID).
		Preload("LeaveType", "company_id = ?", filter.CompanyID).
		Order("employee_leave_balances.fiscal_year DESC")

	var balances []domain.EmployeeLeaveBalance
	if err := query.Find(&balances).Error; err != nil {
		return nil, 0, err
	}
	return balances, total, nil
}

// UpsertBalance creates or updates the entitlement of one balance row.
//
// used_days is deliberately absent from DoUpdates: it belongs to
// AdjustUsedDays, so granting an entitlement mid-year cannot silently reset
// the leave a person has already taken. remaining_days is generated by
// PostgreSQL and is never written.
func (r *leaveRepositoryGorm) UpsertBalance(ctx context.Context, balance *domain.EmployeeLeaveBalance) error {
	return r.db.WithContext(ctx).
		Omit("Employee", "LeaveType").
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "employee_id"}, {Name: "leave_type_id"}, {Name: "fiscal_year"},
			},
			DoUpdates: clause.AssignmentColumns([]string{"entitled_days", "carryover_days", "updated_at"}),
		}).
		Create(balance).Error
}

// AdjustUsedDays adds delta to used_days.
//
// The guard is in the WHERE clause rather than in Go so that two concurrent
// cancellations cannot both read the same used_days and drive it negative.
func (r *leaveRepositoryGorm) AdjustUsedDays(ctx context.Context, companyID, employeeID, leaveTypeID uuid.UUID, fiscalYear int, delta float64) error {
	result := r.db.WithContext(ctx).Model(&domain.EmployeeLeaveBalance{}).
		Where("company_id = ? AND employee_id = ? AND leave_type_id = ? AND fiscal_year = ?",
			companyID, employeeID, leaveTypeID, fiscalYear).
		Where("used_days + ? >= 0", delta).
		UpdateColumns(map[string]interface{}{
			"used_days":  gorm.Expr("used_days + ?", delta),
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// Either no such balance row, or the update would have made used_days
		// negative. Both are refusals, not silent no-ops.
		return domain.ErrLeaveBalanceNotFound
	}
	return nil
}

// WithTransaction executes fn inside one transaction.
func (r *leaveRepositoryGorm) WithTransaction(ctx context.Context, fn func(repo LeaveRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&leaveRepositoryGorm{db: tx})
	})
}
