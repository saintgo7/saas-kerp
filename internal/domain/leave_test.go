package domain_test

import (
	"testing"
	"time"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestEmployeeLeaveValidate(t *testing.T) {
	cases := []struct {
		name  string
		leave domain.EmployeeLeave
		want  error
	}{
		{
			"a three-day request is valid",
			domain.EmployeeLeave{StartDate: day(2026, 3, 2), EndDate: day(2026, 3, 4), Days: 3, Status: domain.LeaveStatusPending},
			nil,
		},
		{
			"a half day inside one calendar day is valid",
			domain.EmployeeLeave{StartDate: day(2026, 3, 2), EndDate: day(2026, 3, 2), Days: 0.5, Status: domain.LeaveStatusPending},
			nil,
		},
		{
			"end before start is rejected",
			domain.EmployeeLeave{StartDate: day(2026, 3, 4), EndDate: day(2026, 3, 2), Days: 1, Status: domain.LeaveStatusPending},
			domain.ErrLeaveDateRange,
		},
		{
			"zero days is rejected",
			domain.EmployeeLeave{StartDate: day(2026, 3, 2), EndDate: day(2026, 3, 2), Days: 0, Status: domain.LeaveStatusPending},
			domain.ErrLeaveDaysNotPositive,
		},
		{
			"more days than the range spans is rejected",
			domain.EmployeeLeave{StartDate: day(2026, 3, 2), EndDate: day(2026, 3, 3), Days: 5, Status: domain.LeaveStatusPending},
			domain.ErrLeaveDaysExceedRange,
		},
		{
			"an unknown status is rejected",
			domain.EmployeeLeave{StartDate: day(2026, 3, 2), EndDate: day(2026, 3, 2), Days: 1, Status: "withdrawn"},
			domain.ErrLeaveInvalidStatus,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			leave := tc.leave
			if err := leave.Validate(); err != tc.want {
				t.Errorf("Validate() = %v, want %v", err, tc.want)
			}
		})
	}
}

// TestLeaveStatusTransitions pins the state machine the balance arithmetic
// depends on: a rejected or cancelled request is terminal, so used_days never
// has to be replayed over an ambiguous history.
func TestLeaveStatusTransitions(t *testing.T) {
	cases := []struct {
		from, to domain.LeaveStatus
		allowed  bool
	}{
		{domain.LeaveStatusPending, domain.LeaveStatusApproved, true},
		{domain.LeaveStatusPending, domain.LeaveStatusRejected, true},
		{domain.LeaveStatusPending, domain.LeaveStatusCancelled, true},
		{domain.LeaveStatusPending, domain.LeaveStatusPending, false},
		{domain.LeaveStatusApproved, domain.LeaveStatusCancelled, true},
		{domain.LeaveStatusApproved, domain.LeaveStatusRejected, false},
		{domain.LeaveStatusApproved, domain.LeaveStatusPending, false},
		{domain.LeaveStatusRejected, domain.LeaveStatusPending, false},
		{domain.LeaveStatusCancelled, domain.LeaveStatusPending, false},
	}

	for _, tc := range cases {
		if got := tc.from.CanTransitionTo(tc.to); got != tc.allowed {
			t.Errorf("%q -> %q = %v, want %v", tc.from, tc.to, got, tc.allowed)
		}
	}
}

func TestEmployeeLeaveConsumesBalance(t *testing.T) {
	for _, status := range []domain.LeaveStatus{
		domain.LeaveStatusPending, domain.LeaveStatusRejected, domain.LeaveStatusCancelled,
	} {
		leave := domain.EmployeeLeave{Status: status}
		if leave.ConsumesBalance() {
			t.Errorf("a %q request must not consume the balance", status)
		}
	}

	approved := domain.EmployeeLeave{Status: domain.LeaveStatusApproved}
	if !approved.ConsumesBalance() {
		t.Error("an approved request must consume the balance")
	}
}

func TestLeaveTypeValidate(t *testing.T) {
	valid := domain.LeaveType{Code: "ANNUAL", Name: "연차", DefaultDays: 15}
	if err := valid.Validate(); err != nil {
		t.Fatalf("a complete leave type must be valid, got %v", err)
	}

	noCode := valid
	noCode.Code = ""
	if err := noCode.Validate(); err != domain.ErrLeaveTypeCodeEmpty {
		t.Errorf("Validate() = %v, want ErrLeaveTypeCodeEmpty", err)
	}

	negative := valid
	negative.MaxCarryoverDays = -1
	if err := negative.Validate(); err != domain.ErrLeaveTypeDaysNegative {
		t.Errorf("Validate() = %v, want ErrLeaveTypeDaysNegative", err)
	}
}

func TestEmployeeLeaveBalanceAvailable(t *testing.T) {
	balance := domain.EmployeeLeaveBalance{
		FiscalYear:    2026,
		EntitledDays:  15,
		CarryoverDays: 2,
		UsedDays:      5,
	}
	if got := balance.Available(); got != 12 {
		t.Errorf("Available() = %v, want 12", got)
	}

	balance.FiscalYear = 0
	if err := balance.Validate(); err != domain.ErrLeaveBalanceFiscalYear {
		t.Errorf("Validate() = %v, want ErrLeaveBalanceFiscalYear", err)
	}

	balance.FiscalYear = 2026
	balance.UsedDays = -1
	if err := balance.Validate(); err != domain.ErrLeaveBalanceNegative {
		t.Errorf("Validate() = %v, want ErrLeaveBalanceNegative", err)
	}
}
