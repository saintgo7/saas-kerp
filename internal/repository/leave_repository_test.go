package repository_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// TestLeaveTypeRepositoryScopesEveryStatementToTheCompany is the tenant
// isolation test for leave types.
func TestLeaveTypeRepositoryScopesEveryStatementToTheCompany(t *testing.T) {
	db, recorder := newHRDryRunDB(t)
	repo := repository.NewLeaveTypeRepositoryGorm(db)

	ctx := context.Background()
	companyID := uuid.New()
	typeID := uuid.New()
	active := true

	calls := []struct {
		name string
		run  func()
	}{
		{"Create", func() {
			_ = repo.Create(ctx, &domain.LeaveType{
				TenantModel: domain.TenantModel{CompanyID: companyID},
				Code:        "ANNUAL",
				Name:        "연차",
			})
		}},
		{"GetByID", func() { _, _ = repo.GetByID(ctx, companyID, typeID) }},
		{"GetByCode", func() { _, _ = repo.GetByCode(ctx, companyID, "ANNUAL") }},
		{"List", func() {
			_, _, _ = repo.List(ctx, &repository.LeaveTypeFilter{
				CompanyID:  companyID,
				IsActive:   &active,
				SearchTerm: "연",
				Page:       1,
				PageSize:   20,
			})
		}},
		{"Update", func() {
			_ = repo.Update(ctx, &domain.LeaveType{
				TenantModel: domain.TenantModel{
					BaseModel: domain.BaseModel{ID: typeID},
					CompanyID: companyID,
				},
				Code: "ANNUAL",
				Name: "연차",
			})
		}},
		{"Delete", func() { _ = repo.Delete(ctx, companyID, typeID) }},
		{"ExistsByCode", func() { _, _ = repo.ExistsByCode(ctx, companyID, "ANNUAL", &typeID) }},
		{"CountReferences", func() { _, _ = repo.CountReferences(ctx, companyID, typeID) }},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			call.run()
			assertEveryStatementIsTenantScoped(t, "LeaveTypeRepository."+call.name, recorder)
		})
	}
}

// TestLeaveRepositoryScopesEveryStatementToTheCompany is the tenant isolation
// test for leave requests and balances.
//
// WithTransaction is not exercised here: opening a transaction needs a real
// connection, and everything it wraps is one of the methods below.
func TestLeaveRepositoryScopesEveryStatementToTheCompany(t *testing.T) {
	db, recorder := newHRDryRunDB(t)
	repo := repository.NewLeaveRepositoryGorm(db)

	ctx := context.Background()
	companyID := uuid.New()
	leaveID := uuid.New()
	employeeID := uuid.New()
	typeID := uuid.New()
	year := 2026
	start := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)

	calls := []struct {
		name string
		run  func()
	}{
		{"Create", func() {
			_ = repo.Create(ctx, &domain.EmployeeLeave{
				TenantModel: domain.TenantModel{CompanyID: companyID},
				EmployeeID:  employeeID,
				LeaveTypeID: typeID,
				StartDate:   start,
				EndDate:     end,
				Days:        3,
				Status:      domain.LeaveStatusPending,
			})
		}},
		{"GetByID", func() { _, _ = repo.GetByID(ctx, companyID, leaveID) }},
		{"List", func() {
			_, _, _ = repo.List(ctx, &repository.LeaveFilter{
				CompanyID:   companyID,
				EmployeeID:  &employeeID,
				LeaveTypeID: &typeID,
				Status:      domain.LeaveStatusPending,
				DateFrom:    &start,
				DateTo:      &end,
				Page:        1,
				PageSize:    20,
			})
		}},
		{"Update", func() {
			_ = repo.Update(ctx, &domain.EmployeeLeave{
				TenantModel: domain.TenantModel{
					BaseModel: domain.BaseModel{ID: leaveID},
					CompanyID: companyID,
				},
				EmployeeID:  employeeID,
				LeaveTypeID: typeID,
				StartDate:   start,
				EndDate:     end,
				Days:        3,
				Status:      domain.LeaveStatusApproved,
			})
		}},
		{"Delete", func() { _ = repo.Delete(ctx, companyID, leaveID) }},
		{"CountOverlapping", func() {
			_, _ = repo.CountOverlapping(ctx, companyID, employeeID, start, end, &leaveID)
		}},
		{"GetBalance", func() { _, _ = repo.GetBalance(ctx, companyID, employeeID, typeID, year) }},
		{"ListBalances", func() {
			_, _, _ = repo.ListBalances(ctx, &repository.LeaveBalanceFilter{
				CompanyID:   companyID,
				EmployeeID:  &employeeID,
				LeaveTypeID: &typeID,
				FiscalYear:  &year,
				Page:        1,
				PageSize:    20,
			})
		}},
		{"UpsertBalance", func() {
			_ = repo.UpsertBalance(ctx, &domain.EmployeeLeaveBalance{
				CompanyID:    companyID,
				EmployeeID:   employeeID,
				LeaveTypeID:  typeID,
				FiscalYear:   year,
				EntitledDays: 15,
			})
		}},
		{"AdjustUsedDays", func() {
			_ = repo.AdjustUsedDays(ctx, companyID, employeeID, typeID, year, 3)
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			call.run()
			assertEveryStatementIsTenantScoped(t, "LeaveRepository."+call.name, recorder)
		})
	}
}

// TestUpsertBalanceNeverWritesUsedDaysOrRemainingDays pins two schema facts.
//
// remaining_days is GENERATED ALWAYS ... STORED, so writing it aborts the
// statement. used_days belongs to AdjustUsedDays, so an entitlement granted
// mid-year must not reset the leave somebody has already taken.
func TestUpsertBalanceNeverWritesUsedDaysOrRemainingDays(t *testing.T) {
	db, recorder := newHRDryRunDB(t)
	repo := repository.NewLeaveRepositoryGorm(db)

	_ = repo.UpsertBalance(context.Background(), &domain.EmployeeLeaveBalance{
		CompanyID:     uuid.New(),
		EmployeeID:    uuid.New(),
		LeaveTypeID:   uuid.New(),
		FiscalYear:    2026,
		EntitledDays:  15,
		CarryoverDays: 2,
		UsedDays:      99,
		RemainingDays: 99,
	})

	if len(recorder.statements) == 0 {
		t.Fatal("no SQL was built")
	}
	for _, sql := range recorder.statements {
		if strings.Contains(sql, "remaining_days") {
			t.Errorf("remaining_days is a generated column and must never be written:\n\t%s", sql)
		}
		upsert := strings.SplitN(sql, "ON CONFLICT", 2)
		if len(upsert) == 2 && strings.Contains(upsert[1], "used_days") {
			t.Errorf("an entitlement upsert must not overwrite used_days:\n\t%s", sql)
		}
	}
}

// TestAdjustUsedDaysRefusesToGoNegative pins the guard that lives in the WHERE
// clause rather than in Go, so two concurrent cancellations cannot both read
// the same used_days and drive it below zero.
func TestAdjustUsedDaysRefusesToGoNegative(t *testing.T) {
	db, recorder := newHRDryRunDB(t)
	repo := repository.NewLeaveRepositoryGorm(db)

	_ = repo.AdjustUsedDays(context.Background(), uuid.New(), uuid.New(), uuid.New(), 2026, -3)

	if len(recorder.statements) != 1 {
		t.Fatalf("expected exactly one statement, got %d: %v", len(recorder.statements), recorder.statements)
	}
	sql := recorder.statements[0]
	if !strings.Contains(sql, "used_days + ") {
		t.Errorf("the non-negative guard is not in the statement:\n\t%s", sql)
	}
}
