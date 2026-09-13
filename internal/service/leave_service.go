package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// Leave-related errors. They alias the domain errors so the repository, the
// service and the handler all compare the same values.
var (
	ErrLeaveTypeNotFound   = domain.ErrLeaveTypeNotFound
	ErrLeaveTypeCodeExists = domain.ErrLeaveTypeCodeExists
	ErrLeaveTypeInactive   = domain.ErrLeaveTypeInactive
	ErrLeaveTypeInUse      = domain.ErrLeaveTypeInUse

	ErrLeaveNotFound          = domain.ErrLeaveNotFound
	ErrLeaveInvalidTransition = domain.ErrLeaveInvalidTransition
	ErrLeaveNotPending        = domain.ErrLeaveNotPending
	ErrLeaveOverlaps          = domain.ErrLeaveOverlaps
	ErrLeaveInsufficient      = domain.ErrLeaveInsufficient
	ErrLeaveEmployeeInactive  = domain.ErrLeaveEmployeeInactive

	ErrLeaveBalanceNotFound = domain.ErrLeaveBalanceNotFound

	// ErrLeaveRejectionReason is returned when a rejection carries no reason.
	// A refusal without one is not reviewable later.
	ErrLeaveRejectionReason = errors.New("a rejection reason is required")
)

// Leave filters re-exported from repository
type (
	LeaveTypeFilter    = repository.LeaveTypeFilter
	LeaveFilter        = repository.LeaveFilter
	LeaveBalanceFilter = repository.LeaveBalanceFilter
)

// LeaveService defines the interface for leave business logic.
type LeaveService interface {
	// Leave types
	CreateType(ctx context.Context, leaveType *domain.LeaveType) error
	UpdateType(ctx context.Context, leaveType *domain.LeaveType) error
	DeleteType(ctx context.Context, companyID, id uuid.UUID) error
	GetType(ctx context.Context, companyID, id uuid.UUID) (*domain.LeaveType, error)
	ListTypes(ctx context.Context, filter *LeaveTypeFilter) ([]domain.LeaveType, int64, error)

	// Leave requests
	Create(ctx context.Context, leave *domain.EmployeeLeave) error
	Update(ctx context.Context, leave *domain.EmployeeLeave) error
	Approve(ctx context.Context, companyID, id, approverID uuid.UUID) error
	Reject(ctx context.Context, companyID, id, approverID uuid.UUID, reason string) error
	Cancel(ctx context.Context, companyID, id uuid.UUID) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.EmployeeLeave, error)
	List(ctx context.Context, filter *LeaveFilter) ([]domain.EmployeeLeave, int64, error)

	// Balances
	ListBalances(ctx context.Context, filter *LeaveBalanceFilter) ([]domain.EmployeeLeaveBalance, int64, error)
	GrantBalance(ctx context.Context, balance *domain.EmployeeLeaveBalance) error
}

// leaveService implements LeaveService.
type leaveService struct {
	repo         repository.LeaveRepository
	typeRepo     repository.LeaveTypeRepository
	employeeRepo repository.EmployeeRepository
}

// NewLeaveService creates a new LeaveService.
func NewLeaveService(
	repo repository.LeaveRepository,
	typeRepo repository.LeaveTypeRepository,
	employeeRepo repository.EmployeeRepository,
) LeaveService {
	return &leaveService{repo: repo, typeRepo: typeRepo, employeeRepo: employeeRepo}
}

// ---------------------------------------------------------------------------
// Leave types
// ---------------------------------------------------------------------------

// CreateType validates and stores a leave type.
func (s *leaveService) CreateType(ctx context.Context, leaveType *domain.LeaveType) error {
	leaveType.Code = strings.TrimSpace(leaveType.Code)
	leaveType.Name = strings.TrimSpace(leaveType.Name)

	if err := leaveType.Validate(); err != nil {
		return err
	}

	exists, err := s.typeRepo.ExistsByCode(ctx, leaveType.CompanyID, leaveType.Code, nil)
	if err != nil {
		return err
	}
	if exists {
		return ErrLeaveTypeCodeExists
	}

	return s.typeRepo.Create(ctx, leaveType)
}

// UpdateType validates and stores changes to a leave type.
func (s *leaveService) UpdateType(ctx context.Context, leaveType *domain.LeaveType) error {
	leaveType.Code = strings.TrimSpace(leaveType.Code)
	leaveType.Name = strings.TrimSpace(leaveType.Name)

	if err := leaveType.Validate(); err != nil {
		return err
	}

	if _, err := s.typeRepo.GetByID(ctx, leaveType.CompanyID, leaveType.ID); err != nil {
		return err
	}

	exists, err := s.typeRepo.ExistsByCode(ctx, leaveType.CompanyID, leaveType.Code, &leaveType.ID)
	if err != nil {
		return err
	}
	if exists {
		return ErrLeaveTypeCodeExists
	}

	return s.typeRepo.Update(ctx, leaveType)
}

// DeleteType removes a leave type once nothing references it.
//
// leave_types has no deleted_at, so this is a real DELETE and both
// employee_leaves and employee_leave_balances reference the row. Deactivating
// (is_active = false) is the way to retire a type that has been used.
func (s *leaveService) DeleteType(ctx context.Context, companyID, id uuid.UUID) error {
	if _, err := s.typeRepo.GetByID(ctx, companyID, id); err != nil {
		return err
	}

	refs, err := s.typeRepo.CountReferences(ctx, companyID, id)
	if err != nil {
		return err
	}
	if refs > 0 {
		return ErrLeaveTypeInUse
	}

	return s.typeRepo.Delete(ctx, companyID, id)
}

// GetType retrieves one leave type.
func (s *leaveService) GetType(ctx context.Context, companyID, id uuid.UUID) (*domain.LeaveType, error) {
	return s.typeRepo.GetByID(ctx, companyID, id)
}

// ListTypes retrieves leave types with filtering.
func (s *leaveService) ListTypes(ctx context.Context, filter *LeaveTypeFilter) ([]domain.LeaveType, int64, error) {
	filter.Page, filter.PageSize = clampHRListPage(filter.Page, filter.PageSize)
	return s.typeRepo.List(ctx, filter)
}

// ---------------------------------------------------------------------------
// Leave requests
// ---------------------------------------------------------------------------

// Create validates and stores a leave request.
//
// A type whose requires_approval is false is recorded as already approved and
// consumes its balance immediately; anything else starts as pending.
func (s *leaveService) Create(ctx context.Context, leave *domain.EmployeeLeave) error {
	leave.Status = domain.LeaveStatusPending
	leave.ApprovedAt = nil
	leave.ApprovedBy = nil
	leave.RejectionReason = ""

	if err := leave.Validate(); err != nil {
		return err
	}

	if err := s.requireActiveEmployee(ctx, leave.CompanyID, leave.EmployeeID); err != nil {
		return err
	}

	leaveType, err := s.requireActiveLeaveType(ctx, leave.CompanyID, leave.LeaveTypeID)
	if err != nil {
		return err
	}

	overlapping, err := s.repo.CountOverlapping(ctx, leave.CompanyID, leave.EmployeeID, leave.StartDate, leave.EndDate, nil)
	if err != nil {
		return err
	}
	if overlapping > 0 {
		return ErrLeaveOverlaps
	}

	if leaveType.RequiresApproval {
		return s.repo.Create(ctx, leave)
	}

	// No approval needed: store it approved and move the balance in the same
	// transaction, so a request can never be recorded as taken without the
	// balance that pays for it.
	now := time.Now().UTC()
	leave.Status = domain.LeaveStatusApproved
	leave.ApprovedAt = &now
	leave.ApprovedBy = leave.CreatedBy

	return s.repo.WithTransaction(ctx, func(tx repository.LeaveRepository) error {
		if err := tx.Create(ctx, leave); err != nil {
			return err
		}
		return s.consumeBalance(ctx, tx, leave, leaveType)
	})
}

// Update changes a pending leave request.
//
// Only a pending request may be edited. Editing an approved one would move
// days that have already been counted against the balance, and editing a
// rejected or cancelled one would rewrite a decision that has been recorded.
func (s *leaveService) Update(ctx context.Context, leave *domain.EmployeeLeave) error {
	stored, err := s.repo.GetByID(ctx, leave.CompanyID, leave.ID)
	if err != nil {
		return err
	}
	if stored.Status != domain.LeaveStatusPending {
		return ErrLeaveNotPending
	}

	// The identity of the request is fixed: only its content may change.
	leave.EmployeeID = stored.EmployeeID
	leave.Status = domain.LeaveStatusPending
	leave.ApprovedAt = nil
	leave.ApprovedBy = nil
	leave.RejectionReason = ""

	if err := leave.Validate(); err != nil {
		return err
	}

	if _, err := s.requireActiveLeaveType(ctx, leave.CompanyID, leave.LeaveTypeID); err != nil {
		return err
	}

	overlapping, err := s.repo.CountOverlapping(ctx, leave.CompanyID, leave.EmployeeID, leave.StartDate, leave.EndDate, &leave.ID)
	if err != nil {
		return err
	}
	if overlapping > 0 {
		return ErrLeaveOverlaps
	}

	return s.repo.Update(ctx, leave)
}

// Approve approves a pending leave request and consumes the balance.
func (s *leaveService) Approve(ctx context.Context, companyID, id, approverID uuid.UUID) error {
	leave, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return err
	}
	if !leave.Status.CanTransitionTo(domain.LeaveStatusApproved) {
		return ErrLeaveInvalidTransition
	}

	leaveType, err := s.typeRepo.GetByID(ctx, companyID, leave.LeaveTypeID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	leave.Status = domain.LeaveStatusApproved
	leave.ApprovedAt = &now
	if approverID != uuid.Nil {
		leave.ApprovedBy = &approverID
	}
	leave.RejectionReason = ""

	// The status change and the balance movement commit together: an approval
	// that is recorded without consuming the balance hands the days out twice.
	return s.repo.WithTransaction(ctx, func(tx repository.LeaveRepository) error {
		if err := tx.Update(ctx, leave); err != nil {
			return err
		}
		return s.consumeBalance(ctx, tx, leave, leaveType)
	})
}

// Reject refuses a pending leave request.
func (s *leaveService) Reject(ctx context.Context, companyID, id, approverID uuid.UUID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrLeaveRejectionReason
	}

	leave, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return err
	}
	if !leave.Status.CanTransitionTo(domain.LeaveStatusRejected) {
		return ErrLeaveInvalidTransition
	}

	now := time.Now().UTC()
	leave.Status = domain.LeaveStatusRejected
	leave.RejectionReason = reason
	leave.ApprovedAt = &now
	if approverID != uuid.Nil {
		leave.ApprovedBy = &approverID
	}

	return s.repo.Update(ctx, leave)
}

// Cancel withdraws a leave request, returning the balance if it was approved.
func (s *leaveService) Cancel(ctx context.Context, companyID, id uuid.UUID) error {
	leave, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return err
	}
	if !leave.Status.CanTransitionTo(domain.LeaveStatusCancelled) {
		return ErrLeaveInvalidTransition
	}

	consumed := leave.ConsumesBalance()
	fiscalYear := fiscalYearOfLeave(leave)
	leave.Status = domain.LeaveStatusCancelled

	return s.repo.WithTransaction(ctx, func(tx repository.LeaveRepository) error {
		if err := tx.Update(ctx, leave); err != nil {
			return err
		}
		if !consumed {
			return nil
		}
		// Give the days back. AdjustUsedDays refuses to drive used_days below
		// zero, so a double cancellation cannot invent leave.
		err := tx.AdjustUsedDays(ctx, leave.CompanyID, leave.EmployeeID, leave.LeaveTypeID, fiscalYear, -leave.Days)
		if errors.Is(err, domain.ErrLeaveBalanceNotFound) {
			// No balance is tracked for this type and year; there is nothing
			// to return and the cancellation still stands.
			return nil
		}
		return err
	})
}

// Delete removes a leave request that was never acted on.
func (s *leaveService) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	leave, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return err
	}
	if leave.Status != domain.LeaveStatusPending {
		// An approved, rejected or cancelled request is part of the record.
		return ErrLeaveNotPending
	}
	return s.repo.Delete(ctx, companyID, id)
}

// GetByID retrieves one leave request.
func (s *leaveService) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.EmployeeLeave, error) {
	return s.repo.GetByID(ctx, companyID, id)
}

// List retrieves leave requests with filtering.
func (s *leaveService) List(ctx context.Context, filter *LeaveFilter) ([]domain.EmployeeLeave, int64, error) {
	filter.Page, filter.PageSize = clampHRListPage(filter.Page, filter.PageSize)
	return s.repo.List(ctx, filter)
}

// ---------------------------------------------------------------------------
// Balances
// ---------------------------------------------------------------------------

// ListBalances retrieves leave balances with filtering.
func (s *leaveService) ListBalances(ctx context.Context, filter *LeaveBalanceFilter) ([]domain.EmployeeLeaveBalance, int64, error) {
	filter.Page, filter.PageSize = clampHRListPage(filter.Page, filter.PageSize)
	return s.repo.ListBalances(ctx, filter)
}

// GrantBalance sets the entitlement of one employee, leave type and year.
//
// It never writes used_days: consumption is maintained by approvals and
// cancellations, so re-granting an entitlement mid-year cannot erase the leave
// somebody has already taken.
func (s *leaveService) GrantBalance(ctx context.Context, balance *domain.EmployeeLeaveBalance) error {
	if err := balance.Validate(); err != nil {
		return err
	}

	exists, err := s.employeeRepo.Exists(ctx, balance.CompanyID, balance.EmployeeID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrEmployeeNotFound
	}

	if _, err := s.typeRepo.GetByID(ctx, balance.CompanyID, balance.LeaveTypeID); err != nil {
		return err
	}

	balance.UsedDays = 0
	return s.repo.UpsertBalance(ctx, balance)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// requireActiveEmployee checks that the employee belongs to this company and
// has not left it.
func (s *leaveService) requireActiveEmployee(ctx context.Context, companyID, employeeID uuid.UUID) error {
	employee, err := s.employeeRepo.GetByID(ctx, companyID, employeeID)
	if err != nil {
		return err
	}
	if !employee.IsActive() {
		return ErrLeaveEmployeeInactive
	}
	return nil
}

// requireActiveLeaveType checks that the leave type belongs to this company
// and is still offered.
func (s *leaveService) requireActiveLeaveType(ctx context.Context, companyID, leaveTypeID uuid.UUID) (*domain.LeaveType, error) {
	leaveType, err := s.typeRepo.GetByID(ctx, companyID, leaveTypeID)
	if err != nil {
		return nil, err
	}
	if !leaveType.IsActive {
		return nil, ErrLeaveTypeInactive
	}
	return leaveType, nil
}

// consumeBalance moves an approved request's days onto the balance row,
// creating the row from the type's default entitlement when it is missing.
//
// A type that grants nothing (default_days = 0 and no entitlement on file) is
// not balance-checked: public holidays and bereavement leave are recorded, not
// rationed.
func (s *leaveService) consumeBalance(
	ctx context.Context,
	tx repository.LeaveRepository,
	leave *domain.EmployeeLeave,
	leaveType *domain.LeaveType,
) error {
	fiscalYear := fiscalYearOfLeave(leave)

	balance, err := tx.GetBalance(ctx, leave.CompanyID, leave.EmployeeID, leave.LeaveTypeID, fiscalYear)
	if err != nil {
		if !errors.Is(err, domain.ErrLeaveBalanceNotFound) {
			return err
		}
		balance = &domain.EmployeeLeaveBalance{
			CompanyID:    leave.CompanyID,
			EmployeeID:   leave.EmployeeID,
			LeaveTypeID:  leave.LeaveTypeID,
			FiscalYear:   fiscalYear,
			EntitledDays: leaveType.DefaultDays,
		}
		if err := tx.UpsertBalance(ctx, balance); err != nil {
			return err
		}
	}

	granted := balance.EntitledDays + balance.CarryoverDays
	if granted > 0 && balance.Available() < leave.Days {
		return ErrLeaveInsufficient
	}

	return tx.AdjustUsedDays(ctx, leave.CompanyID, leave.EmployeeID, leave.LeaveTypeID, fiscalYear, leave.Days)
}

// fiscalYearOfLeave is the year a leave request is booked against.
//
// It is the calendar year the leave STARTS in. The companies table carries
// fiscal_year_start_month, but employee_leave_balances is keyed by a plain
// fiscal_year integer with no anchor month, so a company-specific year would
// make the same integer mean different windows for different tenants. Clients
// that grant entitlements pass fiscal_year explicitly.
func fiscalYearOfLeave(leave *domain.EmployeeLeave) int {
	return leave.StartDate.Year()
}
