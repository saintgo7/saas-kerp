package dto

import (
	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// ---------------------------------------------------------------------------
// Leave types
// ---------------------------------------------------------------------------

// LeaveTypeResponse represents a leave type in API responses.
type LeaveTypeResponse struct {
	ID               string  `json:"id"`
	Code             string  `json:"code"`
	Name             string  `json:"name"`
	IsPaid           bool    `json:"is_paid"`
	DefaultDays      float64 `json:"default_days"`
	MaxCarryoverDays float64 `json:"max_carryover_days"`
	RequiresApproval bool    `json:"requires_approval"`
	IsActive         bool    `json:"is_active"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// FromLeaveType converts domain.LeaveType to LeaveTypeResponse.
func FromLeaveType(leaveType *domain.LeaveType) LeaveTypeResponse {
	return LeaveTypeResponse{
		ID:               leaveType.ID.String(),
		Code:             leaveType.Code,
		Name:             leaveType.Name,
		IsPaid:           leaveType.IsPaid,
		DefaultDays:      leaveType.DefaultDays,
		MaxCarryoverDays: leaveType.MaxCarryoverDays,
		RequiresApproval: leaveType.RequiresApproval,
		IsActive:         leaveType.IsActive,
		CreatedAt:        leaveType.CreatedAt.Format(hrTimeFormat),
		UpdatedAt:        leaveType.UpdatedAt.Format(hrTimeFormat),
	}
}

// FromLeaveTypes converts []domain.LeaveType to []LeaveTypeResponse.
func FromLeaveTypes(types []domain.LeaveType) []LeaveTypeResponse {
	responses := make([]LeaveTypeResponse, len(types))
	for i := range types {
		responses[i] = FromLeaveType(&types[i])
	}
	return responses
}

// LeaveTypeRequest carries the writable fields of a leave type.
type LeaveTypeRequest struct {
	Code             string   `json:"code" binding:"required,max=20"`
	Name             string   `json:"name" binding:"required,max=100"`
	IsPaid           *bool    `json:"is_paid,omitempty"`
	DefaultDays      *float64 `json:"default_days,omitempty" binding:"omitempty,min=0,max=999"`
	MaxCarryoverDays *float64 `json:"max_carryover_days,omitempty" binding:"omitempty,min=0,max=999"`
	RequiresApproval *bool    `json:"requires_approval,omitempty"`
	IsActive         *bool    `json:"is_active,omitempty"`
}

// ApplyTo writes the request onto a leave type.
func (r *LeaveTypeRequest) ApplyTo(leaveType *domain.LeaveType) {
	leaveType.Code = r.Code
	leaveType.Name = r.Name
	if r.IsPaid != nil {
		leaveType.IsPaid = *r.IsPaid
	}
	if r.DefaultDays != nil {
		leaveType.DefaultDays = *r.DefaultDays
	}
	if r.MaxCarryoverDays != nil {
		leaveType.MaxCarryoverDays = *r.MaxCarryoverDays
	}
	if r.RequiresApproval != nil {
		leaveType.RequiresApproval = *r.RequiresApproval
	}
	if r.IsActive != nil {
		leaveType.IsActive = *r.IsActive
	}
}

// NewLeaveType builds a leave type with the schema defaults applied.
func (r *LeaveTypeRequest) NewLeaveType(companyID uuid.UUID) *domain.LeaveType {
	leaveType := &domain.LeaveType{
		TenantModel:      domain.TenantModel{CompanyID: companyID},
		IsPaid:           true,
		RequiresApproval: true,
		IsActive:         true,
	}
	r.ApplyTo(leaveType)
	return leaveType
}

// ---------------------------------------------------------------------------
// Leave requests
// ---------------------------------------------------------------------------

// LeaveResponse represents a leave request in API responses.
type LeaveResponse struct {
	ID            string `json:"id"`
	EmployeeID    string `json:"employee_id"`
	EmployeeNo    string `json:"employee_no,omitempty"`
	EmployeeName  string `json:"employee_name,omitempty"`
	LeaveTypeID   string `json:"leave_type_id"`
	LeaveTypeCode string `json:"leave_type_code,omitempty"`
	LeaveTypeName string `json:"leave_type_name,omitempty"`

	StartDate string  `json:"start_date"`
	EndDate   string  `json:"end_date"`
	Days      float64 `json:"days"`

	Reason          string `json:"reason,omitempty"`
	Status          string `json:"status"`
	ApprovedAt      string `json:"approved_at,omitempty"`
	ApprovedBy      string `json:"approved_by,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// FromLeave converts domain.EmployeeLeave to LeaveResponse.
func FromLeave(leave *domain.EmployeeLeave) LeaveResponse {
	resp := LeaveResponse{
		ID:              leave.ID.String(),
		EmployeeID:      leave.EmployeeID.String(),
		LeaveTypeID:     leave.LeaveTypeID.String(),
		StartDate:       leave.StartDate.Format(hrDateFormat),
		EndDate:         leave.EndDate.Format(hrDateFormat),
		Days:            leave.Days,
		Reason:          leave.Reason,
		Status:          string(leave.Status),
		RejectionReason: leave.RejectionReason,
		CreatedAt:       leave.CreatedAt.Format(hrTimeFormat),
		UpdatedAt:       leave.UpdatedAt.Format(hrTimeFormat),
	}

	// The employee association is a name and a number only. Nothing sensitive
	// from employees is projected here, and the resident number cannot be:
	// EmployeeResponse has no such field and this type has none either.
	if leave.Employee != nil {
		resp.EmployeeNo = leave.Employee.EmployeeNo
		resp.EmployeeName = leave.Employee.Name
	}
	if leave.LeaveType != nil {
		resp.LeaveTypeCode = leave.LeaveType.Code
		resp.LeaveTypeName = leave.LeaveType.Name
	}
	if leave.ApprovedAt != nil {
		resp.ApprovedAt = leave.ApprovedAt.Format(hrTimeFormat)
	}
	if leave.ApprovedBy != nil {
		resp.ApprovedBy = leave.ApprovedBy.String()
	}

	return resp
}

// FromLeaves converts []domain.EmployeeLeave to []LeaveResponse.
func FromLeaves(leaves []domain.EmployeeLeave) []LeaveResponse {
	responses := make([]LeaveResponse, len(leaves))
	for i := range leaves {
		responses[i] = FromLeave(&leaves[i])
	}
	return responses
}

// CreateLeaveRequest represents the request to file a leave request.
type CreateLeaveRequest struct {
	EmployeeID  string  `json:"employee_id" binding:"required,uuid"`
	LeaveTypeID string  `json:"leave_type_id" binding:"required,uuid"`
	StartDate   string  `json:"start_date" binding:"required"`
	EndDate     string  `json:"end_date" binding:"required"`
	Days        float64 `json:"days" binding:"required,gt=0"`
	Reason      string  `json:"reason,omitempty" binding:"max=500"`
}

// ToLeave converts the request to a domain.EmployeeLeave.
func (r *CreateLeaveRequest) ToLeave(companyID uuid.UUID) (*domain.EmployeeLeave, error) {
	employeeID, err := uuid.Parse(r.EmployeeID)
	if err != nil {
		return nil, err
	}
	leaveTypeID, err := uuid.Parse(r.LeaveTypeID)
	if err != nil {
		return nil, err
	}
	startDate, err := parseHRDate(r.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := parseHRDate(r.EndDate)
	if err != nil {
		return nil, err
	}
	if startDate == nil || endDate == nil {
		return nil, ErrInvalidHRDate
	}

	return &domain.EmployeeLeave{
		TenantModel: domain.TenantModel{CompanyID: companyID},
		EmployeeID:  employeeID,
		LeaveTypeID: leaveTypeID,
		StartDate:   *startDate,
		EndDate:     *endDate,
		Days:        r.Days,
		Reason:      r.Reason,
		Status:      domain.LeaveStatusPending,
	}, nil
}

// UpdateLeaveRequest represents the request to edit a pending leave request.
// employee_id is absent on purpose: a filed request cannot be moved to another
// person.
type UpdateLeaveRequest struct {
	LeaveTypeID string  `json:"leave_type_id" binding:"required,uuid"`
	StartDate   string  `json:"start_date" binding:"required"`
	EndDate     string  `json:"end_date" binding:"required"`
	Days        float64 `json:"days" binding:"required,gt=0"`
	Reason      string  `json:"reason,omitempty" binding:"max=500"`
}

// ApplyTo writes the request onto a leave request.
func (r *UpdateLeaveRequest) ApplyTo(leave *domain.EmployeeLeave) error {
	leaveTypeID, err := uuid.Parse(r.LeaveTypeID)
	if err != nil {
		return err
	}
	startDate, err := parseHRDate(r.StartDate)
	if err != nil {
		return err
	}
	endDate, err := parseHRDate(r.EndDate)
	if err != nil {
		return err
	}
	if startDate == nil || endDate == nil {
		return ErrInvalidHRDate
	}

	leave.LeaveTypeID = leaveTypeID
	leave.StartDate = *startDate
	leave.EndDate = *endDate
	leave.Days = r.Days
	leave.Reason = r.Reason
	return nil
}

// RejectLeaveRequest carries the mandatory rejection reason.
type RejectLeaveRequest struct {
	Reason string `json:"reason" binding:"required,max=500"`
}

// ---------------------------------------------------------------------------
// Balances
// ---------------------------------------------------------------------------

// LeaveBalanceResponse represents a leave balance in API responses.
type LeaveBalanceResponse struct {
	ID            string  `json:"id"`
	EmployeeID    string  `json:"employee_id"`
	EmployeeNo    string  `json:"employee_no,omitempty"`
	EmployeeName  string  `json:"employee_name,omitempty"`
	LeaveTypeID   string  `json:"leave_type_id"`
	LeaveTypeCode string  `json:"leave_type_code,omitempty"`
	LeaveTypeName string  `json:"leave_type_name,omitempty"`
	FiscalYear    int     `json:"fiscal_year"`
	EntitledDays  float64 `json:"entitled_days"`
	CarryoverDays float64 `json:"carryover_days"`
	UsedDays      float64 `json:"used_days"`
	RemainingDays float64 `json:"remaining_days"`
	UpdatedAt     string  `json:"updated_at"`
}

// FromLeaveBalance converts domain.EmployeeLeaveBalance to its response.
func FromLeaveBalance(balance *domain.EmployeeLeaveBalance) LeaveBalanceResponse {
	resp := LeaveBalanceResponse{
		ID:            balance.ID.String(),
		EmployeeID:    balance.EmployeeID.String(),
		LeaveTypeID:   balance.LeaveTypeID.String(),
		FiscalYear:    balance.FiscalYear,
		EntitledDays:  balance.EntitledDays,
		CarryoverDays: balance.CarryoverDays,
		UsedDays:      balance.UsedDays,
		RemainingDays: balance.RemainingDays,
		UpdatedAt:     balance.UpdatedAt.Format(hrTimeFormat),
	}
	if balance.Employee != nil {
		resp.EmployeeNo = balance.Employee.EmployeeNo
		resp.EmployeeName = balance.Employee.Name
	}
	if balance.LeaveType != nil {
		resp.LeaveTypeCode = balance.LeaveType.Code
		resp.LeaveTypeName = balance.LeaveType.Name
	}
	return resp
}

// FromLeaveBalances converts a slice of balances.
func FromLeaveBalances(balances []domain.EmployeeLeaveBalance) []LeaveBalanceResponse {
	responses := make([]LeaveBalanceResponse, len(balances))
	for i := range balances {
		responses[i] = FromLeaveBalance(&balances[i])
	}
	return responses
}

// GrantLeaveBalanceRequest sets an entitlement.
//
// used_days is not accepted: consumption is derived from approved requests, so
// letting a client set it directly would let leave be un-taken by hand.
type GrantLeaveBalanceRequest struct {
	EmployeeID    string  `json:"employee_id" binding:"required,uuid"`
	LeaveTypeID   string  `json:"leave_type_id" binding:"required,uuid"`
	FiscalYear    int     `json:"fiscal_year" binding:"required,min=1900,max=9999"`
	EntitledDays  float64 `json:"entitled_days" binding:"min=0,max=999"`
	CarryoverDays float64 `json:"carryover_days" binding:"min=0,max=999"`
}

// ToBalance converts the request to a domain.EmployeeLeaveBalance.
func (r *GrantLeaveBalanceRequest) ToBalance(companyID uuid.UUID) (*domain.EmployeeLeaveBalance, error) {
	employeeID, err := uuid.Parse(r.EmployeeID)
	if err != nil {
		return nil, err
	}
	leaveTypeID, err := uuid.Parse(r.LeaveTypeID)
	if err != nil {
		return nil, err
	}

	return &domain.EmployeeLeaveBalance{
		CompanyID:     companyID,
		EmployeeID:    employeeID,
		LeaveTypeID:   leaveTypeID,
		FiscalYear:    r.FiscalYear,
		EntitledDays:  r.EntitledDays,
		CarryoverDays: r.CarryoverDays,
	}, nil
}
