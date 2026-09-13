package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// ---------------------------------------------------------------------------
// In-memory leave repositories
// ---------------------------------------------------------------------------

type balanceKey struct {
	company, employee, leaveType uuid.UUID
	fiscalYear                   int
}

type fakeLeaveRepo struct {
	leaves   map[uuid.UUID]domain.EmployeeLeave
	balances map[balanceKey]domain.EmployeeLeaveBalance
}

func newFakeLeaveRepo() *fakeLeaveRepo {
	return &fakeLeaveRepo{
		leaves:   map[uuid.UUID]domain.EmployeeLeave{},
		balances: map[balanceKey]domain.EmployeeLeaveBalance{},
	}
}

func (f *fakeLeaveRepo) Create(_ context.Context, leave *domain.EmployeeLeave) error {
	if leave.ID == uuid.Nil {
		leave.ID = uuid.New()
	}
	f.leaves[leave.ID] = *leave
	return nil
}

func (f *fakeLeaveRepo) GetByID(_ context.Context, companyID, id uuid.UUID) (*domain.EmployeeLeave, error) {
	leave, ok := f.leaves[id]
	if !ok || leave.CompanyID != companyID {
		return nil, domain.ErrLeaveNotFound
	}
	copied := leave
	return &copied, nil
}

func (f *fakeLeaveRepo) List(_ context.Context, filter *repository.LeaveFilter) ([]domain.EmployeeLeave, int64, error) {
	var out []domain.EmployeeLeave
	for _, leave := range f.leaves {
		if leave.CompanyID == filter.CompanyID {
			out = append(out, leave)
		}
	}
	return out, int64(len(out)), nil
}

func (f *fakeLeaveRepo) Update(_ context.Context, leave *domain.EmployeeLeave) error {
	stored, ok := f.leaves[leave.ID]
	if !ok || stored.CompanyID != leave.CompanyID {
		return domain.ErrLeaveNotFound
	}
	f.leaves[leave.ID] = *leave
	return nil
}

func (f *fakeLeaveRepo) Delete(_ context.Context, companyID, id uuid.UUID) error {
	leave, ok := f.leaves[id]
	if !ok || leave.CompanyID != companyID {
		return domain.ErrLeaveNotFound
	}
	delete(f.leaves, id)
	return nil
}

func (f *fakeLeaveRepo) CountOverlapping(_ context.Context, companyID, employeeID uuid.UUID, start, end time.Time, excludeID *uuid.UUID) (int64, error) {
	var count int64
	for id, leave := range f.leaves {
		if leave.CompanyID != companyID || leave.EmployeeID != employeeID {
			continue
		}
		if leave.Status != domain.LeaveStatusPending && leave.Status != domain.LeaveStatusApproved {
			continue
		}
		if excludeID != nil && *excludeID == id {
			continue
		}
		if !leave.StartDate.After(end) && !leave.EndDate.Before(start) {
			count++
		}
	}
	return count, nil
}

func (f *fakeLeaveRepo) GetBalance(_ context.Context, companyID, employeeID, leaveTypeID uuid.UUID, fiscalYear int) (*domain.EmployeeLeaveBalance, error) {
	balance, ok := f.balances[balanceKey{companyID, employeeID, leaveTypeID, fiscalYear}]
	if !ok {
		return nil, domain.ErrLeaveBalanceNotFound
	}
	copied := balance
	return &copied, nil
}

func (f *fakeLeaveRepo) ListBalances(_ context.Context, filter *repository.LeaveBalanceFilter) ([]domain.EmployeeLeaveBalance, int64, error) {
	var out []domain.EmployeeLeaveBalance
	for key, balance := range f.balances {
		if key.company == filter.CompanyID {
			out = append(out, balance)
		}
	}
	return out, int64(len(out)), nil
}

// UpsertBalance mirrors the real one: it writes the entitlement and never
// used_days, and it recomputes the generated remaining_days.
func (f *fakeLeaveRepo) UpsertBalance(_ context.Context, balance *domain.EmployeeLeaveBalance) error {
	key := balanceKey{balance.CompanyID, balance.EmployeeID, balance.LeaveTypeID, balance.FiscalYear}
	stored, exists := f.balances[key]
	if !exists {
		stored = *balance
		if stored.ID == uuid.Nil {
			stored.ID = uuid.New()
		}
		stored.UsedDays = 0
	} else {
		stored.EntitledDays = balance.EntitledDays
		stored.CarryoverDays = balance.CarryoverDays
	}
	stored.RemainingDays = stored.EntitledDays + stored.CarryoverDays - stored.UsedDays
	f.balances[key] = stored
	*balance = stored
	return nil
}

func (f *fakeLeaveRepo) AdjustUsedDays(_ context.Context, companyID, employeeID, leaveTypeID uuid.UUID, fiscalYear int, delta float64) error {
	key := balanceKey{companyID, employeeID, leaveTypeID, fiscalYear}
	balance, ok := f.balances[key]
	if !ok {
		return domain.ErrLeaveBalanceNotFound
	}
	if balance.UsedDays+delta < 0 {
		return domain.ErrLeaveBalanceNotFound
	}
	balance.UsedDays += delta
	balance.RemainingDays = balance.EntitledDays + balance.CarryoverDays - balance.UsedDays
	f.balances[key] = balance
	return nil
}

// WithTransaction snapshots both maps and restores them when fn fails, so the
// tests can tell an atomic approval from one that half-happened.
func (f *fakeLeaveRepo) WithTransaction(ctx context.Context, fn func(repo repository.LeaveRepository) error) error {
	leaves := make(map[uuid.UUID]domain.EmployeeLeave, len(f.leaves))
	for k, v := range f.leaves {
		leaves[k] = v
	}
	balances := make(map[balanceKey]domain.EmployeeLeaveBalance, len(f.balances))
	for k, v := range f.balances {
		balances[k] = v
	}

	if err := fn(f); err != nil {
		f.leaves = leaves
		f.balances = balances
		return err
	}
	return nil
}

type fakeLeaveTypeRepo struct {
	types map[uuid.UUID]domain.LeaveType
	refs  map[uuid.UUID]int64
}

func newFakeLeaveTypeRepo() *fakeLeaveTypeRepo {
	return &fakeLeaveTypeRepo{
		types: map[uuid.UUID]domain.LeaveType{},
		refs:  map[uuid.UUID]int64{},
	}
}

func (f *fakeLeaveTypeRepo) put(leaveType domain.LeaveType) domain.LeaveType {
	if leaveType.ID == uuid.Nil {
		leaveType.ID = uuid.New()
	}
	f.types[leaveType.ID] = leaveType
	return leaveType
}

func (f *fakeLeaveTypeRepo) Create(_ context.Context, leaveType *domain.LeaveType) error {
	if leaveType.ID == uuid.Nil {
		leaveType.ID = uuid.New()
	}
	f.types[leaveType.ID] = *leaveType
	return nil
}

func (f *fakeLeaveTypeRepo) GetByID(_ context.Context, companyID, id uuid.UUID) (*domain.LeaveType, error) {
	leaveType, ok := f.types[id]
	if !ok || leaveType.CompanyID != companyID {
		return nil, domain.ErrLeaveTypeNotFound
	}
	copied := leaveType
	return &copied, nil
}

func (f *fakeLeaveTypeRepo) GetByCode(_ context.Context, companyID uuid.UUID, code string) (*domain.LeaveType, error) {
	for _, leaveType := range f.types {
		if leaveType.CompanyID == companyID && leaveType.Code == code {
			copied := leaveType
			return &copied, nil
		}
	}
	return nil, domain.ErrLeaveTypeNotFound
}

func (f *fakeLeaveTypeRepo) List(_ context.Context, filter *repository.LeaveTypeFilter) ([]domain.LeaveType, int64, error) {
	var out []domain.LeaveType
	for _, leaveType := range f.types {
		if leaveType.CompanyID == filter.CompanyID {
			out = append(out, leaveType)
		}
	}
	return out, int64(len(out)), nil
}

func (f *fakeLeaveTypeRepo) Update(_ context.Context, leaveType *domain.LeaveType) error {
	stored, ok := f.types[leaveType.ID]
	if !ok || stored.CompanyID != leaveType.CompanyID {
		return domain.ErrLeaveTypeNotFound
	}
	f.types[leaveType.ID] = *leaveType
	return nil
}

func (f *fakeLeaveTypeRepo) Delete(_ context.Context, companyID, id uuid.UUID) error {
	leaveType, ok := f.types[id]
	if !ok || leaveType.CompanyID != companyID {
		return domain.ErrLeaveTypeNotFound
	}
	delete(f.types, id)
	return nil
}

func (f *fakeLeaveTypeRepo) ExistsByCode(_ context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	for id, leaveType := range f.types {
		if leaveType.CompanyID != companyID || leaveType.Code != code {
			continue
		}
		if excludeID != nil && *excludeID == id {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (f *fakeLeaveTypeRepo) CountReferences(_ context.Context, _, id uuid.UUID) (int64, error) {
	return f.refs[id], nil
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

type leaveFixture struct {
	svc        service.LeaveService
	leaves     *fakeLeaveRepo
	types      *fakeLeaveTypeRepo
	employees  *fakeEmployeeRepo
	companyID  uuid.UUID
	employeeID uuid.UUID
	annualID   uuid.UUID
}

func newLeaveFixture(t *testing.T) *leaveFixture {
	t.Helper()

	leaves := newFakeLeaveRepo()
	types := newFakeLeaveTypeRepo()
	employees := newFakeEmployeeRepo()

	companyID := uuid.New()
	employee := employees.put(domain.Employee{
		TenantModel:    domain.TenantModel{CompanyID: companyID},
		EmployeeNo:     "EMP001",
		Name:           "홍길동",
		HireDate:       time.Date(2020, 3, 2, 0, 0, 0, 0, time.UTC),
		Status:         domain.EmployeeStatusActive,
		EmploymentType: domain.EmploymentTypeRegular,
	})
	annual := types.put(domain.LeaveType{
		TenantModel:      domain.TenantModel{CompanyID: companyID},
		Code:             "ANNUAL",
		Name:             "연차",
		IsPaid:           true,
		DefaultDays:      15,
		RequiresApproval: true,
		IsActive:         true,
	})

	return &leaveFixture{
		svc:        service.NewLeaveService(leaves, types, employees),
		leaves:     leaves,
		types:      types,
		employees:  employees,
		companyID:  companyID,
		employeeID: employee.ID,
		annualID:   annual.ID,
	}
}

func (f *leaveFixture) request(days float64, start, end time.Time) *domain.EmployeeLeave {
	return &domain.EmployeeLeave{
		TenantModel: domain.TenantModel{CompanyID: f.companyID},
		EmployeeID:  f.employeeID,
		LeaveTypeID: f.annualID,
		StartDate:   start,
		EndDate:     end,
		Days:        days,
		Status:      domain.LeaveStatusPending,
	}
}

func march(d int) time.Time { return time.Date(2026, 3, d, 0, 0, 0, 0, time.UTC) }

func (f *leaveFixture) balance(t *testing.T) domain.EmployeeLeaveBalance {
	t.Helper()
	balance, err := f.leaves.GetBalance(context.Background(), f.companyID, f.employeeID, f.annualID, 2026)
	if err != nil {
		t.Fatalf("reading the balance: %v", err)
	}
	return *balance
}

// ---------------------------------------------------------------------------
// Approval and the balance
// ---------------------------------------------------------------------------

// TestApprovalConsumesTheBalance: approving is what turns a request into leave
// taken, so it is the only thing that may move used_days.
func TestApprovalConsumesTheBalance(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	leave := f.request(3, march(2), march(4))
	if err := f.svc.Create(ctx, leave); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// A pending request has not consumed anything yet.
	if _, err := f.leaves.GetBalance(ctx, f.companyID, f.employeeID, f.annualID, 2026); !errors.Is(err, domain.ErrLeaveBalanceNotFound) {
		t.Errorf("a pending request created a balance movement: %v", err)
	}

	if err := f.svc.Approve(ctx, f.companyID, leave.ID, uuid.New()); err != nil {
		t.Fatalf("Approve: %v", err)
	}

	balance := f.balance(t)
	if balance.UsedDays != 3 {
		t.Errorf("used_days = %v, want 3", balance.UsedDays)
	}
	if balance.EntitledDays != 15 {
		t.Errorf("entitled_days = %v, want the leave type default of 15", balance.EntitledDays)
	}
	if f.leaves.leaves[leave.ID].Status != domain.LeaveStatusApproved {
		t.Error("the request was not marked approved")
	}
}

// TestApprovalIsRefusedWhenTheBalanceIsShort, and the refusal leaves neither
// the status nor the balance changed - the two move together or not at all.
func TestApprovalIsRefusedWhenTheBalanceIsShort(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	if err := f.svc.GrantBalance(ctx, &domain.EmployeeLeaveBalance{
		CompanyID: f.companyID, EmployeeID: f.employeeID, LeaveTypeID: f.annualID,
		FiscalYear: 2026, EntitledDays: 2,
	}); err != nil {
		t.Fatalf("GrantBalance: %v", err)
	}

	leave := f.request(3, march(2), march(4))
	if err := f.svc.Create(ctx, leave); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := f.svc.Approve(ctx, f.companyID, leave.ID, uuid.New()); !errors.Is(err, domain.ErrLeaveInsufficient) {
		t.Fatalf("Approve() = %v, want ErrLeaveInsufficient", err)
	}
	if f.leaves.leaves[leave.ID].Status != domain.LeaveStatusPending {
		t.Error("the request was approved despite the refusal")
	}
	if used := f.balance(t).UsedDays; used != 0 {
		t.Errorf("used_days = %v after a refused approval, want 0", used)
	}
}

// TestCancellingAnApprovedRequestReturnsTheDays.
func TestCancellingAnApprovedRequestReturnsTheDays(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	leave := f.request(3, march(2), march(4))
	if err := f.svc.Create(ctx, leave); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := f.svc.Approve(ctx, f.companyID, leave.ID, uuid.New()); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if err := f.svc.Cancel(ctx, f.companyID, leave.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	if used := f.balance(t).UsedDays; used != 0 {
		t.Errorf("used_days = %v after cancelling, want 0", used)
	}
	if f.leaves.leaves[leave.ID].Status != domain.LeaveStatusCancelled {
		t.Error("the request was not marked cancelled")
	}
}

// TestCancellingAPendingRequestTouchesNoBalance: it never consumed anything.
func TestCancellingAPendingRequestTouchesNoBalance(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	leave := f.request(3, march(2), march(4))
	if err := f.svc.Create(ctx, leave); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := f.svc.Cancel(ctx, f.companyID, leave.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if len(f.leaves.balances) != 0 {
		t.Error("cancelling a pending request touched a balance")
	}
}

// TestGrantingAnEntitlementDoesNotResetUsedDays: re-granting mid-year must not
// erase the leave somebody has already taken.
func TestGrantingAnEntitlementDoesNotResetUsedDays(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	leave := f.request(3, march(2), march(4))
	if err := f.svc.Create(ctx, leave); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := f.svc.Approve(ctx, f.companyID, leave.ID, uuid.New()); err != nil {
		t.Fatalf("Approve: %v", err)
	}

	if err := f.svc.GrantBalance(ctx, &domain.EmployeeLeaveBalance{
		CompanyID: f.companyID, EmployeeID: f.employeeID, LeaveTypeID: f.annualID,
		FiscalYear: 2026, EntitledDays: 20, CarryoverDays: 2,
	}); err != nil {
		t.Fatalf("GrantBalance: %v", err)
	}

	balance := f.balance(t)
	if balance.UsedDays != 3 {
		t.Errorf("used_days = %v after re-granting, want the 3 already taken", balance.UsedDays)
	}
	if balance.EntitledDays != 20 || balance.CarryoverDays != 2 {
		t.Errorf("the entitlement was not updated: %+v", balance)
	}
}

// TestALeaveTypeThatNeedsNoApprovalIsRecordedAsTaken.
func TestALeaveTypeThatNeedsNoApprovalIsRecordedAsTaken(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	official := f.types.put(domain.LeaveType{
		TenantModel:      domain.TenantModel{CompanyID: f.companyID},
		Code:             "OFFICIAL",
		Name:             "공가",
		IsPaid:           true,
		RequiresApproval: false,
		IsActive:         true,
	})

	leave := f.request(1, march(10), march(10))
	leave.LeaveTypeID = official.ID
	if err := f.svc.Create(ctx, leave); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if f.leaves.leaves[leave.ID].Status != domain.LeaveStatusApproved {
		t.Error("a type that needs no approval must be recorded as approved")
	}
	balance, err := f.leaves.GetBalance(ctx, f.companyID, f.employeeID, official.ID, 2026)
	if err != nil {
		t.Fatalf("reading the balance: %v", err)
	}
	if balance.UsedDays != 1 {
		t.Errorf("used_days = %v, want 1", balance.UsedDays)
	}
	// default_days is 0 for this type, so it is recorded but not rationed.
	if balance.EntitledDays != 0 {
		t.Errorf("entitled_days = %v, want 0", balance.EntitledDays)
	}
}

// ---------------------------------------------------------------------------
// Request rules
// ---------------------------------------------------------------------------

// TestOverlappingRequestsAreRefused: one person cannot be on two leaves at
// once, and a double submission must not consume the balance twice.
func TestOverlappingRequestsAreRefused(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	if err := f.svc.Create(ctx, f.request(3, march(2), march(4))); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := f.svc.Create(ctx, f.request(1, march(4), march(4))); !errors.Is(err, domain.ErrLeaveOverlaps) {
		t.Fatalf("Create() = %v, want ErrLeaveOverlaps", err)
	}
	// A request that starts the day after is fine.
	if err := f.svc.Create(ctx, f.request(1, march(5), march(5))); err != nil {
		t.Fatalf("a non-overlapping request was refused: %v", err)
	}
}

// TestALeaveForASeparatedEmployeeIsRefused.
func TestALeaveForASeparatedEmployeeIsRefused(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	employee := f.employees.employees[f.employeeID]
	resigned := march(1)
	employee.Status = domain.EmployeeStatusResigned
	employee.ResignationDate = &resigned
	f.employees.employees[f.employeeID] = employee

	if err := f.svc.Create(ctx, f.request(1, march(10), march(10))); !errors.Is(err, domain.ErrLeaveEmployeeInactive) {
		t.Fatalf("Create() = %v, want ErrLeaveEmployeeInactive", err)
	}
}

// TestAnInactiveLeaveTypeIsRefused: deactivating a type is how a company
// retires it, so it must stop accepting new requests.
func TestAnInactiveLeaveTypeIsRefused(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	annual := f.types.types[f.annualID]
	annual.IsActive = false
	f.types.types[f.annualID] = annual

	if err := f.svc.Create(ctx, f.request(1, march(10), march(10))); !errors.Is(err, domain.ErrLeaveTypeInactive) {
		t.Fatalf("Create() = %v, want ErrLeaveTypeInactive", err)
	}
}

// TestOnlyAPendingRequestCanBeEditedOrDeleted.
func TestOnlyAPendingRequestCanBeEditedOrDeleted(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	leave := f.request(3, march(2), march(4))
	if err := f.svc.Create(ctx, leave); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := f.svc.Approve(ctx, f.companyID, leave.ID, uuid.New()); err != nil {
		t.Fatalf("Approve: %v", err)
	}

	loaded, err := f.svc.GetByID(ctx, f.companyID, leave.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	loaded.Days = 1
	if err := f.svc.Update(ctx, loaded); !errors.Is(err, domain.ErrLeaveNotPending) {
		t.Errorf("Update() = %v, want ErrLeaveNotPending", err)
	}
	if err := f.svc.Delete(ctx, f.companyID, leave.ID); !errors.Is(err, domain.ErrLeaveNotPending) {
		t.Errorf("Delete() = %v, want ErrLeaveNotPending", err)
	}
	if used := f.balance(t).UsedDays; used != 3 {
		t.Errorf("used_days = %v, want the approved 3 to be untouched", used)
	}
}

// TestRejectionNeedsAReason and is terminal.
func TestRejectionNeedsAReason(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	leave := f.request(1, march(2), march(2))
	if err := f.svc.Create(ctx, leave); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := f.svc.Reject(ctx, f.companyID, leave.ID, uuid.New(), "   "); !errors.Is(err, service.ErrLeaveRejectionReason) {
		t.Fatalf("Reject() = %v, want ErrLeaveRejectionReason", err)
	}
	if err := f.svc.Reject(ctx, f.companyID, leave.ID, uuid.New(), "인원 부족"); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if f.leaves.leaves[leave.ID].Status != domain.LeaveStatusRejected {
		t.Error("the request was not marked rejected")
	}
	if err := f.svc.Approve(ctx, f.companyID, leave.ID, uuid.New()); !errors.Is(err, domain.ErrLeaveInvalidTransition) {
		t.Errorf("approving a rejected request = %v, want ErrLeaveInvalidTransition", err)
	}
}

// TestApprovingTwiceIsRefused: the second approval would consume the balance a
// second time.
func TestApprovingTwiceIsRefused(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	leave := f.request(3, march(2), march(4))
	if err := f.svc.Create(ctx, leave); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := f.svc.Approve(ctx, f.companyID, leave.ID, uuid.New()); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if err := f.svc.Approve(ctx, f.companyID, leave.ID, uuid.New()); !errors.Is(err, domain.ErrLeaveInvalidTransition) {
		t.Fatalf("the second Approve() = %v, want ErrLeaveInvalidTransition", err)
	}
	if used := f.balance(t).UsedDays; used != 3 {
		t.Errorf("used_days = %v, want 3", used)
	}
}

// ---------------------------------------------------------------------------
// Tenant isolation
// ---------------------------------------------------------------------------

// TestLeaveIsInvisibleToOtherCompanies.
func TestLeaveIsInvisibleToOtherCompanies(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()
	intruder := uuid.New()

	leave := f.request(3, march(2), march(4))
	if err := f.svc.Create(ctx, leave); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := f.svc.GetByID(ctx, intruder, leave.ID); !errors.Is(err, domain.ErrLeaveNotFound) {
		t.Errorf("GetByID across tenants = %v, want ErrLeaveNotFound", err)
	}
	if err := f.svc.Approve(ctx, intruder, leave.ID, uuid.New()); !errors.Is(err, domain.ErrLeaveNotFound) {
		t.Errorf("Approve across tenants = %v, want ErrLeaveNotFound", err)
	}
	if err := f.svc.Reject(ctx, intruder, leave.ID, uuid.New(), "no"); !errors.Is(err, domain.ErrLeaveNotFound) {
		t.Errorf("Reject across tenants = %v, want ErrLeaveNotFound", err)
	}
	if err := f.svc.Cancel(ctx, intruder, leave.ID); !errors.Is(err, domain.ErrLeaveNotFound) {
		t.Errorf("Cancel across tenants = %v, want ErrLeaveNotFound", err)
	}
	if err := f.svc.Delete(ctx, intruder, leave.ID); !errors.Is(err, domain.ErrLeaveNotFound) {
		t.Errorf("Delete across tenants = %v, want ErrLeaveNotFound", err)
	}

	listed, _, err := f.svc.List(ctx, &service.LeaveFilter{CompanyID: intruder})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed) != 0 {
		t.Errorf("another tenant's list returned %d leave requests", len(listed))
	}

	if f.leaves.leaves[leave.ID].Status != domain.LeaveStatusPending {
		t.Error("the intruder changed the request")
	}
}

// TestALeaveCannotBeFiledForAnotherCompanysEmployee.
func TestALeaveCannotBeFiledForAnotherCompanysEmployee(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	foreign := f.employees.put(domain.Employee{
		TenantModel: domain.TenantModel{CompanyID: uuid.New()},
		EmployeeNo:  "OTHER", Name: "다른회사",
		HireDate: march(1), Status: domain.EmployeeStatusActive,
		EmploymentType: domain.EmploymentTypeRegular,
	})

	leave := f.request(1, march(10), march(10))
	leave.EmployeeID = foreign.ID

	if err := f.svc.Create(ctx, leave); !errors.Is(err, domain.ErrEmployeeNotFound) {
		t.Fatalf("Create() = %v, want ErrEmployeeNotFound", err)
	}
}

// TestALeaveCannotUseAnotherCompanysLeaveType.
func TestALeaveCannotUseAnotherCompanysLeaveType(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	foreign := f.types.put(domain.LeaveType{
		TenantModel: domain.TenantModel{CompanyID: uuid.New()},
		Code:        "ANNUAL", Name: "연차", IsActive: true, RequiresApproval: true,
	})

	leave := f.request(1, march(10), march(10))
	leave.LeaveTypeID = foreign.ID

	if err := f.svc.Create(ctx, leave); !errors.Is(err, domain.ErrLeaveTypeNotFound) {
		t.Fatalf("Create() = %v, want ErrLeaveTypeNotFound", err)
	}
}

// TestGrantingABalanceToAnotherCompanysEmployeeIsRefused.
func TestGrantingABalanceToAnotherCompanysEmployeeIsRefused(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	foreign := f.employees.put(domain.Employee{
		TenantModel: domain.TenantModel{CompanyID: uuid.New()},
		EmployeeNo:  "OTHER", Name: "다른회사", HireDate: march(1),
		Status: domain.EmployeeStatusActive, EmploymentType: domain.EmploymentTypeRegular,
	})

	err := f.svc.GrantBalance(ctx, &domain.EmployeeLeaveBalance{
		CompanyID: f.companyID, EmployeeID: foreign.ID, LeaveTypeID: f.annualID,
		FiscalYear: 2026, EntitledDays: 15,
	})
	if !errors.Is(err, domain.ErrEmployeeNotFound) {
		t.Fatalf("GrantBalance() = %v, want ErrEmployeeNotFound", err)
	}
	if len(f.leaves.balances) != 0 {
		t.Error("a balance was created anyway")
	}
}

// ---------------------------------------------------------------------------
// Leave types
// ---------------------------------------------------------------------------

// TestALeaveTypeInUseCannotBeDeleted: the table has no deleted_at, so the
// delete is real and both referencing tables would refuse it.
func TestALeaveTypeInUseCannotBeDeleted(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	f.types.refs[f.annualID] = 4
	if err := f.svc.DeleteType(ctx, f.companyID, f.annualID); !errors.Is(err, domain.ErrLeaveTypeInUse) {
		t.Fatalf("DeleteType() = %v, want ErrLeaveTypeInUse", err)
	}
	if _, ok := f.types.types[f.annualID]; !ok {
		t.Error("the leave type was deleted anyway")
	}

	f.types.refs[f.annualID] = 0
	if err := f.svc.DeleteType(ctx, f.companyID, f.annualID); err != nil {
		t.Fatalf("DeleteType() = %v, want nil once unused", err)
	}
}

// TestDuplicateLeaveTypeCodeIsRefused within one company.
func TestDuplicateLeaveTypeCodeIsRefused(t *testing.T) {
	f := newLeaveFixture(t)
	ctx := context.Background()

	err := f.svc.CreateType(ctx, &domain.LeaveType{
		TenantModel: domain.TenantModel{CompanyID: f.companyID},
		Code:        "ANNUAL", Name: "연차2", IsActive: true,
	})
	if !errors.Is(err, domain.ErrLeaveTypeCodeExists) {
		t.Fatalf("CreateType() = %v, want ErrLeaveTypeCodeExists", err)
	}
}
