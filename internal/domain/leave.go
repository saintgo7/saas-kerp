package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Leave errors
var (
	ErrLeaveTypeNotFound     = errors.New("leave type not found")
	ErrLeaveTypeCodeExists   = errors.New("leave type code already exists")
	ErrLeaveTypeCodeEmpty    = errors.New("leave type code is required")
	ErrLeaveTypeNameEmpty    = errors.New("leave type name is required")
	ErrLeaveTypeInactive     = errors.New("leave type is not active")
	ErrLeaveTypeInUse        = errors.New("leave type is referenced by leave records and cannot be deleted")
	ErrLeaveTypeDaysNegative = errors.New("leave type day counts cannot be negative")

	ErrLeaveNotFound          = errors.New("leave request not found")
	ErrLeaveInvalidStatus     = errors.New("invalid leave status")
	ErrLeaveInvalidTransition = errors.New("invalid leave status transition")
	ErrLeaveDateRange         = errors.New("leave end date cannot precede the start date")
	ErrLeaveDaysNotPositive   = errors.New("leave days must be greater than zero")
	ErrLeaveDaysExceedRange   = errors.New("leave days exceed the requested date range")
	ErrLeaveOverlaps          = errors.New("leave overlaps an existing request for this employee")
	ErrLeaveNotPending        = errors.New("only a pending leave request can be changed")
	ErrLeaveInsufficient      = errors.New("remaining leave balance is insufficient")
	ErrLeaveEmployeeInactive  = errors.New("leave cannot be recorded for a separated employee")

	ErrLeaveBalanceNotFound     = errors.New("leave balance not found")
	ErrLeaveBalanceNegative     = errors.New("leave balance days cannot be negative")
	ErrLeaveBalanceFiscalYear   = errors.New("invalid fiscal year")
	ErrLeaveBalanceWouldGoBelow = errors.New("used days cannot fall below zero")
)

// LeaveStatus is the state of one leave request. The values are the CHECK
// constraint on employee_leaves.status.
type LeaveStatus string

const (
	LeaveStatusPending   LeaveStatus = "pending"
	LeaveStatusApproved  LeaveStatus = "approved"
	LeaveStatusRejected  LeaveStatus = "rejected"
	LeaveStatusCancelled LeaveStatus = "cancelled"
)

// IsValid reports whether the status is one the database accepts.
func (s LeaveStatus) IsValid() bool {
	switch s {
	case LeaveStatusPending, LeaveStatusApproved, LeaveStatusRejected, LeaveStatusCancelled:
		return true
	}
	return false
}

// CanTransitionTo reports whether a leave request may move from s to next.
//
// Rejection and cancellation are terminal: a rejected request is re-submitted
// as a new one, so that the balance arithmetic never has to be replayed
// backwards over a record whose history is ambiguous.
func (s LeaveStatus) CanTransitionTo(next LeaveStatus) bool {
	if !s.IsValid() || !next.IsValid() {
		return false
	}
	switch s {
	case LeaveStatusPending:
		return next == LeaveStatusApproved || next == LeaveStatusRejected || next == LeaveStatusCancelled
	case LeaveStatusApproved:
		return next == LeaveStatusCancelled
	}
	return false
}

// LeaveType is a category of leave a company grants (annual, sick, ...).
//
// Table: leave_types.
type LeaveType struct {
	TenantModel

	Code string `gorm:"type:varchar(20);not null" json:"code"`
	Name string `gorm:"type:varchar(100);not null" json:"name"`

	// IsPaid drives payroll, not this package: an unpaid leave still consumes
	// a balance if the company grants one.
	IsPaid           bool    `gorm:"default:true" json:"is_paid"`
	DefaultDays      float64 `gorm:"type:decimal(5,1);default:0" json:"default_days"`
	MaxCarryoverDays float64 `gorm:"type:decimal(5,1);default:0" json:"max_carryover_days"`
	RequiresApproval bool    `gorm:"default:true" json:"requires_approval"`

	IsActive bool `gorm:"default:true" json:"is_active"`
}

// TableName specifies the table name for GORM
func (LeaveType) TableName() string {
	return "leave_types"
}

// Validate checks the invariants of a leave type.
func (t *LeaveType) Validate() error {
	if t.Code == "" {
		return ErrLeaveTypeCodeEmpty
	}
	if t.Name == "" {
		return ErrLeaveTypeNameEmpty
	}
	if t.DefaultDays < 0 || t.MaxCarryoverDays < 0 {
		return ErrLeaveTypeDaysNegative
	}
	return nil
}

// EmployeeLeave is one leave request/record.
//
// Table: employee_leaves. The table has created_by but no updated_by, and no
// deleted_at - a withdrawn request is cancelled, not deleted, so that the
// balance arithmetic stays auditable.
type EmployeeLeave struct {
	TenantModel

	EmployeeID  uuid.UUID  `gorm:"type:uuid;not null" json:"employee_id"`
	Employee    *Employee  `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	LeaveTypeID uuid.UUID  `gorm:"type:uuid;not null" json:"leave_type_id"`
	LeaveType   *LeaveType `gorm:"foreignKey:LeaveTypeID" json:"leave_type,omitempty"`

	StartDate time.Time `gorm:"type:date;not null" json:"start_date"`
	EndDate   time.Time `gorm:"type:date;not null" json:"end_date"`
	Days      float64   `gorm:"type:decimal(5,1);not null" json:"days"`

	Reason string      `gorm:"type:varchar(500)" json:"reason,omitempty"`
	Status LeaveStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`

	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	ApprovedBy      *uuid.UUID `gorm:"type:uuid" json:"approved_by,omitempty"`
	RejectionReason string     `gorm:"type:varchar(500)" json:"rejection_reason,omitempty"`

	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
}

// TableName specifies the table name for GORM
func (EmployeeLeave) TableName() string {
	return "employee_leaves"
}

// Validate checks the invariants of a leave request.
func (l *EmployeeLeave) Validate() error {
	if !l.Status.IsValid() {
		return ErrLeaveInvalidStatus
	}
	if l.EndDate.Before(l.StartDate) {
		return ErrLeaveDateRange
	}
	if l.Days <= 0 {
		return ErrLeaveDaysNotPositive
	}
	if l.Days > l.CalendarSpanDays() {
		return ErrLeaveDaysExceedRange
	}
	return nil
}

// CalendarSpanDays is the inclusive number of calendar days the request spans.
// It is the ceiling for Days: half-days make Days smaller, never larger.
func (l *EmployeeLeave) CalendarSpanDays() float64 {
	start := time.Date(l.StartDate.Year(), l.StartDate.Month(), l.StartDate.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(l.EndDate.Year(), l.EndDate.Month(), l.EndDate.Day(), 0, 0, 0, 0, time.UTC)
	return end.Sub(start).Hours()/24 + 1
}

// ConsumesBalance reports whether the request currently counts against the
// employee's balance. Only an approved request does.
func (l *EmployeeLeave) ConsumesBalance() bool {
	return l.Status == LeaveStatusApproved
}

// EmployeeLeaveBalance is the per-year entitlement and consumption for one
// employee and one leave type.
//
// Table: employee_leave_balances. It does NOT embed BaseModel: the table has
// updated_at but no created_at, so an embedded CreatedAt would make GORM write
// a column that does not exist. remaining_days is a stored generated column
// (entitled + carryover - used) and is therefore read-only here.
type EmployeeLeaveBalance struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v7()" json:"id"`
	CompanyID uuid.UUID `gorm:"type:uuid;not null;index" json:"company_id"`

	EmployeeID  uuid.UUID  `gorm:"type:uuid;not null" json:"employee_id"`
	Employee    *Employee  `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	LeaveTypeID uuid.UUID  `gorm:"type:uuid;not null" json:"leave_type_id"`
	LeaveType   *LeaveType `gorm:"foreignKey:LeaveTypeID" json:"leave_type,omitempty"`

	FiscalYear int `gorm:"not null" json:"fiscal_year"`

	EntitledDays  float64 `gorm:"type:decimal(5,1);not null;default:0" json:"entitled_days"`
	CarryoverDays float64 `gorm:"type:decimal(5,1);not null;default:0" json:"carryover_days"`
	UsedDays      float64 `gorm:"type:decimal(5,1);not null;default:0" json:"used_days"`

	// RemainingDays is GENERATED ALWAYS ... STORED in PostgreSQL. The "->" tag
	// makes it read-only for GORM; writing to it aborts the statement.
	RemainingDays float64 `gorm:"->;type:decimal(5,1)" json:"remaining_days"`

	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (EmployeeLeaveBalance) TableName() string {
	return "employee_leave_balances"
}

// Validate checks the invariants of a balance row.
func (b *EmployeeLeaveBalance) Validate() error {
	if b.FiscalYear < 1900 || b.FiscalYear > 9999 {
		return ErrLeaveBalanceFiscalYear
	}
	if b.EntitledDays < 0 || b.CarryoverDays < 0 || b.UsedDays < 0 {
		return ErrLeaveBalanceNegative
	}
	return nil
}

// Available is entitled + carryover - used, computed in Go so that a balance
// built in memory (before the generated column has been read back) still
// answers correctly.
func (b *EmployeeLeaveBalance) Available() float64 {
	return b.EntitledDays + b.CarryoverDays - b.UsedDays
}
