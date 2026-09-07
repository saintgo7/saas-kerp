package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// LeaveTypeFilter defines filter criteria for listing leave types.
type LeaveTypeFilter struct {
	CompanyID  uuid.UUID
	IsActive   *bool
	SearchTerm string
	Page       int
	PageSize   int
}

// LeaveFilter defines filter criteria for listing leave requests.
//
// DateFrom/DateTo select every request that OVERLAPS the window, not only the
// ones fully inside it: a leave that starts in March and ends in April belongs
// to both months' listings.
type LeaveFilter struct {
	CompanyID   uuid.UUID
	EmployeeID  *uuid.UUID
	LeaveTypeID *uuid.UUID
	Status      domain.LeaveStatus
	DateFrom    *time.Time
	DateTo      *time.Time
	Page        int
	PageSize    int
}

// LeaveBalanceFilter defines filter criteria for listing leave balances.
type LeaveBalanceFilter struct {
	CompanyID   uuid.UUID
	EmployeeID  *uuid.UUID
	LeaveTypeID *uuid.UUID
	FiscalYear  *int
	Page        int
	PageSize    int
}

// LeaveTypeRepository defines the interface for leave type data access.
type LeaveTypeRepository interface {
	Create(ctx context.Context, leaveType *domain.LeaveType) error
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.LeaveType, error)
	GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.LeaveType, error)
	List(ctx context.Context, filter *LeaveTypeFilter) ([]domain.LeaveType, int64, error)
	Update(ctx context.Context, leaveType *domain.LeaveType) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)

	// CountReferences counts leave requests and balances pointing at the type.
	// leave_types has no deleted_at, so deleting one is a real DELETE and both
	// referencing tables would refuse it.
	CountReferences(ctx context.Context, companyID, id uuid.UUID) (int64, error)
}

// LeaveRepository defines the interface for leave request and balance access.
type LeaveRepository interface {
	// Leave requests
	Create(ctx context.Context, leave *domain.EmployeeLeave) error
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.EmployeeLeave, error)
	List(ctx context.Context, filter *LeaveFilter) ([]domain.EmployeeLeave, int64, error)
	Update(ctx context.Context, leave *domain.EmployeeLeave) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// CountOverlapping counts the pending or approved requests of one employee
	// that overlap [start, end]. excludeID skips the request being edited.
	CountOverlapping(ctx context.Context, companyID, employeeID uuid.UUID, start, end time.Time, excludeID *uuid.UUID) (int64, error)

	// Balances
	GetBalance(ctx context.Context, companyID, employeeID, leaveTypeID uuid.UUID, fiscalYear int) (*domain.EmployeeLeaveBalance, error)
	ListBalances(ctx context.Context, filter *LeaveBalanceFilter) ([]domain.EmployeeLeaveBalance, int64, error)

	// UpsertBalance creates or replaces the entitlement of one
	// (employee, leave type, fiscal year). It never writes used_days: that
	// number is maintained by AdjustUsedDays as requests are approved and
	// cancelled, and remaining_days is a generated column.
	UpsertBalance(ctx context.Context, balance *domain.EmployeeLeaveBalance) error

	// AdjustUsedDays adds delta to used_days for one balance row. It refuses to
	// drive the value below zero and reports domain.ErrLeaveBalanceNotFound
	// when no row matches, so that a cancellation can never invent leave.
	AdjustUsedDays(ctx context.Context, companyID, employeeID, leaveTypeID uuid.UUID, fiscalYear int, delta float64) error

	// WithTransaction runs fn against a repository bound to one transaction, so
	// that a status change and the balance it moves commit together.
	WithTransaction(ctx context.Context, fn func(repo LeaveRepository) error) error
}
