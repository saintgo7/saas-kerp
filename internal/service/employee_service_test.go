package service_test

import (
	"bytes"
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
// In-memory repositories
// ---------------------------------------------------------------------------
//
// They are keyed by id but every lookup also checks company_id, exactly as the
// SQL does. That is what makes a cross-tenant test meaningful here: a fake
// that ignored the company id would pass a test the real repository fails.

type fakeEmployeeRepo struct {
	employees   map[uuid.UUID]domain.Employee
	departments map[uuid.UUID]uuid.UUID // department id -> owning company
	retained    map[uuid.UUID]repository.EmployeeRetainedRecords
	reports     map[uuid.UUID]int64
	deleted     []uuid.UUID
}

func newFakeEmployeeRepo() *fakeEmployeeRepo {
	return &fakeEmployeeRepo{
		employees:   map[uuid.UUID]domain.Employee{},
		departments: map[uuid.UUID]uuid.UUID{},
		retained:    map[uuid.UUID]repository.EmployeeRetainedRecords{},
		reports:     map[uuid.UUID]int64{},
	}
}

func (f *fakeEmployeeRepo) put(employee domain.Employee) domain.Employee {
	if employee.ID == uuid.Nil {
		employee.ID = uuid.New()
	}
	f.employees[employee.ID] = employee
	return employee
}

func (f *fakeEmployeeRepo) lookup(companyID, id uuid.UUID) (domain.Employee, bool) {
	employee, ok := f.employees[id]
	if !ok || employee.CompanyID != companyID {
		return domain.Employee{}, false
	}
	return employee, true
}

func (f *fakeEmployeeRepo) Create(_ context.Context, employee *domain.Employee) error {
	if employee.ID == uuid.Nil {
		employee.ID = uuid.New()
	}
	f.employees[employee.ID] = *employee
	return nil
}

func (f *fakeEmployeeRepo) GetByID(_ context.Context, companyID, id uuid.UUID) (*domain.Employee, error) {
	employee, ok := f.lookup(companyID, id)
	if !ok {
		return nil, domain.ErrEmployeeNotFound
	}
	copied := employee
	return &copied, nil
}

func (f *fakeEmployeeRepo) GetByEmployeeNo(_ context.Context, companyID uuid.UUID, employeeNo string) (*domain.Employee, error) {
	for _, employee := range f.employees {
		if employee.CompanyID == companyID && employee.EmployeeNo == employeeNo {
			copied := employee
			return &copied, nil
		}
	}
	return nil, domain.ErrEmployeeNotFound
}

func (f *fakeEmployeeRepo) List(_ context.Context, filter *repository.EmployeeFilter) ([]domain.Employee, int64, error) {
	var out []domain.Employee
	for _, employee := range f.employees {
		if employee.CompanyID == filter.CompanyID {
			out = append(out, employee)
		}
	}
	return out, int64(len(out)), nil
}

func (f *fakeEmployeeRepo) Update(_ context.Context, employee *domain.Employee) error {
	if _, ok := f.lookup(employee.CompanyID, employee.ID); !ok {
		return domain.ErrEmployeeNotFound
	}
	f.employees[employee.ID] = *employee
	return nil
}

func (f *fakeEmployeeRepo) Delete(_ context.Context, companyID, id uuid.UUID) error {
	if _, ok := f.lookup(companyID, id); !ok {
		return domain.ErrEmployeeNotFound
	}
	f.deleted = append(f.deleted, id)
	delete(f.employees, id)
	return nil
}

func (f *fakeEmployeeRepo) ExistsByEmployeeNo(_ context.Context, companyID uuid.UUID, employeeNo string, excludeID *uuid.UUID) (bool, error) {
	for id, employee := range f.employees {
		if employee.CompanyID != companyID || employee.EmployeeNo != employeeNo {
			continue
		}
		if excludeID != nil && *excludeID == id {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (f *fakeEmployeeRepo) Exists(_ context.Context, companyID, id uuid.UUID) (bool, error) {
	_, ok := f.lookup(companyID, id)
	return ok, nil
}

func (f *fakeEmployeeRepo) DepartmentExists(_ context.Context, companyID, departmentID uuid.UUID) (bool, error) {
	owner, ok := f.departments[departmentID]
	return ok && owner == companyID, nil
}

func (f *fakeEmployeeRepo) GetManagerID(_ context.Context, companyID, id uuid.UUID) (*uuid.UUID, error) {
	employee, ok := f.lookup(companyID, id)
	if !ok {
		return nil, domain.ErrEmployeeNotFound
	}
	return employee.ManagerID, nil
}

func (f *fakeEmployeeRepo) CountRetainedRecords(_ context.Context, _, id uuid.UUID) (repository.EmployeeRetainedRecords, error) {
	return f.retained[id], nil
}

func (f *fakeEmployeeRepo) CountDirectReports(_ context.Context, _, id uuid.UUID) (int64, error) {
	return f.reports[id], nil
}

func (f *fakeEmployeeRepo) CountByStatus(_ context.Context, companyID uuid.UUID) (repository.EmployeeStatusCounts, error) {
	var counts repository.EmployeeStatusCounts
	for _, employee := range f.employees {
		if employee.CompanyID != companyID {
			continue
		}
		counts.Total++
		switch employee.Status {
		case domain.EmployeeStatusActive:
			counts.Active++
		case domain.EmployeeStatusOnLeave:
			counts.OnLeave++
		case domain.EmployeeStatusResigned:
			counts.Resigned++
		case domain.EmployeeStatusTerminated:
			counts.Terminated++
		}
	}
	return counts, nil
}

type fakePositionRepo struct {
	positions map[uuid.UUID]domain.Position
	employees map[uuid.UUID]int64
}

func newFakePositionRepo() *fakePositionRepo {
	return &fakePositionRepo{
		positions: map[uuid.UUID]domain.Position{},
		employees: map[uuid.UUID]int64{},
	}
}

func (f *fakePositionRepo) put(position domain.Position) domain.Position {
	if position.ID == uuid.Nil {
		position.ID = uuid.New()
	}
	f.positions[position.ID] = position
	return position
}

func (f *fakePositionRepo) Create(_ context.Context, position *domain.Position) error {
	if position.ID == uuid.Nil {
		position.ID = uuid.New()
	}
	f.positions[position.ID] = *position
	return nil
}

func (f *fakePositionRepo) GetByID(_ context.Context, companyID, id uuid.UUID) (*domain.Position, error) {
	position, ok := f.positions[id]
	if !ok || position.CompanyID != companyID {
		return nil, domain.ErrPositionNotFound
	}
	copied := position
	return &copied, nil
}

func (f *fakePositionRepo) GetByCode(_ context.Context, companyID uuid.UUID, code string) (*domain.Position, error) {
	for _, position := range f.positions {
		if position.CompanyID == companyID && position.Code == code {
			copied := position
			return &copied, nil
		}
	}
	return nil, domain.ErrPositionNotFound
}

func (f *fakePositionRepo) List(_ context.Context, filter *repository.PositionFilter) ([]domain.Position, int64, error) {
	var out []domain.Position
	for _, position := range f.positions {
		if position.CompanyID == filter.CompanyID {
			out = append(out, position)
		}
	}
	return out, int64(len(out)), nil
}

func (f *fakePositionRepo) Update(_ context.Context, position *domain.Position) error {
	stored, ok := f.positions[position.ID]
	if !ok || stored.CompanyID != position.CompanyID {
		return domain.ErrPositionNotFound
	}
	f.positions[position.ID] = *position
	return nil
}

func (f *fakePositionRepo) Delete(_ context.Context, companyID, id uuid.UUID) error {
	position, ok := f.positions[id]
	if !ok || position.CompanyID != companyID {
		return domain.ErrPositionNotFound
	}
	delete(f.positions, id)
	return nil
}

func (f *fakePositionRepo) ExistsByCode(_ context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	for id, position := range f.positions {
		if position.CompanyID != companyID || position.Code != code {
			continue
		}
		if excludeID != nil && *excludeID == id {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (f *fakePositionRepo) CountEmployees(_ context.Context, _, positionID uuid.UUID) (int64, error) {
	return f.employees[positionID], nil
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// testKey is a 32-byte AES key. It exists only in this test binary.
var testKey = bytes.Repeat([]byte{0x2a}, 32)

func testCipher(t *testing.T) service.ResidentNumberCipher {
	t.Helper()
	cipher, err := service.NewResidentNumberCipher(testKey)
	if err != nil {
		t.Fatalf("building the test cipher: %v", err)
	}
	return cipher
}

func newEmployeeFixture(companyID uuid.UUID) *domain.Employee {
	return &domain.Employee{
		TenantModel:    domain.TenantModel{CompanyID: companyID},
		EmployeeNo:     "EMP001",
		Name:           "홍길동",
		HireDate:       time.Date(2020, 3, 2, 0, 0, 0, 0, time.UTC),
		Status:         domain.EmployeeStatusActive,
		EmploymentType: domain.EmploymentTypeRegular,
	}
}

// ---------------------------------------------------------------------------
// Resident number handling
// ---------------------------------------------------------------------------

// TestCreateRefusesAResidentNumberWithNoEncryptionKey is the fail-closed test.
//
// With no key configured the only two options are to refuse the request or to
// write a 주민등록번호 into the database in the clear. It must refuse, and it
// must leave nothing behind.
func TestCreateRefusesAResidentNumberWithNoEncryptionKey(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), service.NewUnavailableResidentNumberCipher())

	companyID := uuid.New()
	employee := newEmployeeFixture(companyID)

	err := svc.Create(context.Background(), employee, "900315-1234567")
	if !errors.Is(err, service.ErrResidentNumberEncryptionUnavailable) {
		t.Fatalf("Create() = %v, want ErrResidentNumberEncryptionUnavailable", err)
	}
	if len(repo.employees) != 0 {
		t.Fatalf("a refused create must store nothing, stored %d rows", len(repo.employees))
	}
	if employee.ResidentNumberEnc != nil {
		t.Error("the domain object must not keep the resident number after a refusal")
	}
	if svc.ResidentNumberAvailable() {
		t.Error("ResidentNumberAvailable() must report the feature as unavailable")
	}
}

// TestCreateWithoutAResidentNumberWorksWithNoKey: the rest of HR must keep
// working when field encryption is not configured.
func TestCreateWithoutAResidentNumberWorksWithNoKey(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), service.NewUnavailableResidentNumberCipher())

	employee := newEmployeeFixture(uuid.New())
	if err := svc.Create(context.Background(), employee, ""); err != nil {
		t.Fatalf("Create() = %v, want nil", err)
	}
	if len(repo.employees) != 1 {
		t.Fatalf("expected the employee to be stored, got %d rows", len(repo.employees))
	}
}

// TestCreateStoresTheResidentNumberEncrypted checks that what reaches the
// repository is ciphertext, not the digits.
func TestCreateStoresTheResidentNumberEncrypted(t *testing.T) {
	repo := newFakeEmployeeRepo()
	cipher := testCipher(t)
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), cipher)

	const plaintext = "900315-1234567"
	const digits = "9003151234567"

	employee := newEmployeeFixture(uuid.New())
	if err := svc.Create(context.Background(), employee, plaintext); err != nil {
		t.Fatalf("Create() = %v, want nil", err)
	}

	stored := repo.employees[employee.ID]
	if len(stored.ResidentNumberEnc) == 0 {
		t.Fatal("no ciphertext was stored")
	}
	if bytes.Contains(stored.ResidentNumberEnc, []byte(digits)) ||
		bytes.Contains(stored.ResidentNumberEnc, []byte(plaintext)) {
		t.Fatalf("the stored bytes contain the plaintext: %q", stored.ResidentNumberEnc)
	}

	decrypted, err := cipher.Decrypt(stored.ResidentNumberEnc)
	if err != nil {
		t.Fatalf("the stored ciphertext does not decrypt: %v", err)
	}
	if decrypted != digits {
		t.Errorf("round trip = %q, want the hyphen-free digits", decrypted)
	}
}

// TestEncryptionIsNotDeterministic: two employees with the same resident
// number must not produce the same ciphertext, otherwise the column becomes a
// lookup table for "are these two people the same person".
func TestEncryptionIsNotDeterministic(t *testing.T) {
	cipher := testCipher(t)

	first, err := cipher.Encrypt("9003151234567")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	second, err := cipher.Encrypt("9003151234567")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if bytes.Equal(first, second) {
		t.Error("encrypting the same value twice produced identical ciphertext")
	}
}

// TestDecryptRejectsTamperedCiphertext: AES-GCM is authenticated, so a flipped
// byte must fail rather than yield different digits.
func TestDecryptRejectsTamperedCiphertext(t *testing.T) {
	cipher := testCipher(t)

	sealed, err := cipher.Encrypt("9003151234567")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	sealed[len(sealed)-1] ^= 0xff

	if _, err := cipher.Decrypt(sealed); !errors.Is(err, service.ErrResidentNumberCiphertext) {
		t.Errorf("Decrypt(tampered) = %v, want ErrResidentNumberCiphertext", err)
	}
}

// TestCreateRejectsAMalformedResidentNumber: a key being available does not
// make any 13 characters acceptable.
func TestCreateRejectsAMalformedResidentNumber(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	err := svc.Create(context.Background(), newEmployeeFixture(uuid.New()), "900315-12345")
	if !errors.Is(err, domain.ErrResidentNumberFormat) {
		t.Fatalf("Create() = %v, want ErrResidentNumberFormat", err)
	}
	if len(repo.employees) != 0 {
		t.Error("a rejected create must store nothing")
	}
}

// TestUpdateKeepsTheStoredResidentNumberWhenNoneIsSupplied: an ordinary edit
// (a new phone number) must not silently erase the encrypted column.
func TestUpdateKeepsTheStoredResidentNumberWhenNoneIsSupplied(t *testing.T) {
	repo := newFakeEmployeeRepo()
	cipher := testCipher(t)
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), cipher)

	ctx := context.Background()
	companyID := uuid.New()
	employee := newEmployeeFixture(companyID)
	if err := svc.Create(ctx, employee, "900315-1234567"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	original := repo.employees[employee.ID].ResidentNumberEnc

	loaded, err := svc.GetByID(ctx, companyID, employee.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	loaded.Phone = "010-1111-2222"
	// The handler's Update path hands the service nil when the request body
	// carries no resident_number field at all.
	if err := svc.Update(ctx, loaded, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if !bytes.Equal(repo.employees[employee.ID].ResidentNumberEnc, original) {
		t.Error("an update with no resident number changed the stored ciphertext")
	}
}

// TestUpdateErasesTheResidentNumberOnAnEmptyString gives the client a way to
// remove the number without a second endpoint.
func TestUpdateErasesTheResidentNumberOnAnEmptyString(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	ctx := context.Background()
	companyID := uuid.New()
	employee := newEmployeeFixture(companyID)
	if err := svc.Create(ctx, employee, "900315-1234567"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	loaded, err := svc.GetByID(ctx, companyID, employee.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	empty := ""
	if err := svc.Update(ctx, loaded, &empty); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if len(repo.employees[employee.ID].ResidentNumberEnc) != 0 {
		t.Error("the resident number was not erased")
	}
}

// TestCipherFromEnvIsFailClosed: no variable means the fail-closed cipher, not
// a silently disabled check.
func TestCipherFromEnvIsFailClosed(t *testing.T) {
	t.Setenv(service.ResidentNumberKeyEnv, "")

	cipher, err := service.NewResidentNumberCipherFromEnv()
	if err != nil {
		t.Fatalf("an unset key must not be an error, got %v", err)
	}
	if cipher.Available() {
		t.Fatal("an unset key must not yield a working cipher")
	}
	if _, err := cipher.Encrypt("9003151234567"); !errors.Is(err, service.ErrResidentNumberEncryptionUnavailable) {
		t.Errorf("Encrypt = %v, want ErrResidentNumberEncryptionUnavailable", err)
	}
}

// TestCipherFromEnvRejectsAShortKey: a misconfigured key is reported, never
// stretched or truncated into something that "works".
func TestCipherFromEnvRejectsAShortKey(t *testing.T) {
	t.Setenv(service.ResidentNumberKeyEnv, "0011223344")

	cipher, err := service.NewResidentNumberCipherFromEnv()
	if !errors.Is(err, service.ErrResidentNumberKeySize) {
		t.Fatalf("NewResidentNumberCipherFromEnv() error = %v, want ErrResidentNumberKeySize", err)
	}
	if cipher.Available() {
		t.Error("a rejected key must leave the cipher unavailable")
	}
}

// ---------------------------------------------------------------------------
// Tenant isolation
// ---------------------------------------------------------------------------

// TestEmployeesAreInvisibleToOtherCompanies covers every read and write path
// with the id of another tenant's employee.
func TestEmployeesAreInvisibleToOtherCompanies(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	ctx := context.Background()
	ownerCompany := uuid.New()
	intruderCompany := uuid.New()

	employee := newEmployeeFixture(ownerCompany)
	if err := svc.Create(ctx, employee, ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := svc.GetByID(ctx, intruderCompany, employee.ID); !errors.Is(err, domain.ErrEmployeeNotFound) {
		t.Errorf("GetByID across tenants = %v, want ErrEmployeeNotFound", err)
	}
	if _, err := svc.GetByEmployeeNo(ctx, intruderCompany, "EMP001"); !errors.Is(err, domain.ErrEmployeeNotFound) {
		t.Errorf("GetByEmployeeNo across tenants = %v, want ErrEmployeeNotFound", err)
	}
	if err := svc.Delete(ctx, intruderCompany, employee.ID); !errors.Is(err, domain.ErrEmployeeNotFound) {
		t.Errorf("Delete across tenants = %v, want ErrEmployeeNotFound", err)
	}
	change := service.EmployeeStatusChange{Status: domain.EmployeeStatusOnLeave}
	if err := svc.ChangeStatus(ctx, intruderCompany, employee.ID, change); !errors.Is(err, domain.ErrEmployeeNotFound) {
		t.Errorf("ChangeStatus across tenants = %v, want ErrEmployeeNotFound", err)
	}

	listed, _, err := svc.List(ctx, &service.EmployeeFilter{CompanyID: intruderCompany})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed) != 0 {
		t.Errorf("another tenant's list returned %d employees", len(listed))
	}

	if _, ok := repo.employees[employee.ID]; !ok {
		t.Error("the owning tenant's employee was affected by the intruder's calls")
	}
}

// TestCreateRejectsAnotherTenantsDepartment: the foreign keys are
// company-blind, so filing an employee under another tenant's department is
// something only the service can refuse.
func TestCreateRejectsAnotherTenantsDepartment(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	companyID := uuid.New()
	foreignDepartment := uuid.New()
	repo.departments[foreignDepartment] = uuid.New() // owned by somebody else

	employee := newEmployeeFixture(companyID)
	employee.DepartmentID = &foreignDepartment

	if err := svc.Create(context.Background(), employee, ""); !errors.Is(err, domain.ErrEmployeeDepartmentUnknown) {
		t.Fatalf("Create() = %v, want ErrEmployeeDepartmentUnknown", err)
	}
}

// TestCreateRejectsAnotherTenantsPosition is the same check for positions.
func TestCreateRejectsAnotherTenantsPosition(t *testing.T) {
	repo := newFakeEmployeeRepo()
	positions := newFakePositionRepo()
	svc := service.NewEmployeeService(repo, positions, testCipher(t))

	companyID := uuid.New()
	foreign := positions.put(domain.Position{
		TenantModel: domain.TenantModel{CompanyID: uuid.New()},
		Code:        "MGR", Name: "과장",
	})

	employee := newEmployeeFixture(companyID)
	employee.PositionID = &foreign.ID

	if err := svc.Create(context.Background(), employee, ""); !errors.Is(err, domain.ErrEmployeePositionUnknown) {
		t.Fatalf("Create() = %v, want ErrEmployeePositionUnknown", err)
	}
}

// TestCreateRejectsAnotherTenantsManager keeps the reporting line inside one
// company.
func TestCreateRejectsAnotherTenantsManager(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	foreignManager := repo.put(domain.Employee{
		TenantModel: domain.TenantModel{CompanyID: uuid.New()},
		EmployeeNo:  "OTHER",
		Name:        "다른회사",
	})

	employee := newEmployeeFixture(uuid.New())
	employee.ManagerID = &foreignManager.ID

	if err := svc.Create(context.Background(), employee, ""); !errors.Is(err, domain.ErrEmployeeManagerNotFound) {
		t.Fatalf("Create() = %v, want ErrEmployeeManagerNotFound", err)
	}
}

// ---------------------------------------------------------------------------
// Reporting line
// ---------------------------------------------------------------------------

// TestUpdateRejectsSelfManagement is the self-reference guard at the service
// boundary, where a real request arrives.
func TestUpdateRejectsSelfManagement(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	ctx := context.Background()
	companyID := uuid.New()
	employee := newEmployeeFixture(companyID)
	if err := svc.Create(ctx, employee, ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	loaded, err := svc.GetByID(ctx, companyID, employee.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	loaded.ManagerID = &loaded.ID

	if err := svc.Update(ctx, loaded, nil); !errors.Is(err, domain.ErrEmployeeSelfManager) {
		t.Fatalf("Update() = %v, want ErrEmployeeSelfManager", err)
	}
	if repo.employees[employee.ID].ManagerID != nil {
		t.Error("the self-reference was stored anyway")
	}
}

// TestUpdateRejectsAReportingCycle covers the indirect case: A reports to B,
// B reports to C, and C is then made to report to A.
func TestUpdateRejectsAReportingCycle(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	ctx := context.Background()
	companyID := uuid.New()

	c := repo.put(domain.Employee{
		TenantModel: domain.TenantModel{CompanyID: companyID},
		EmployeeNo:  "C", Name: "씨", HireDate: time.Now(),
		Status:         domain.EmployeeStatusActive,
		EmploymentType: domain.EmploymentTypeRegular,
	})
	b := repo.put(domain.Employee{
		TenantModel: domain.TenantModel{CompanyID: companyID},
		EmployeeNo:  "B", Name: "비", HireDate: time.Now(),
		Status:         domain.EmployeeStatusActive,
		EmploymentType: domain.EmploymentTypeRegular,
		ManagerID:      &c.ID,
	})
	a := repo.put(domain.Employee{
		TenantModel: domain.TenantModel{CompanyID: companyID},
		EmployeeNo:  "A", Name: "에이", HireDate: time.Now(),
		Status:         domain.EmployeeStatusActive,
		EmploymentType: domain.EmploymentTypeRegular,
		ManagerID:      &b.ID,
	})

	loaded, err := svc.GetByID(ctx, companyID, c.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	loaded.ManagerID = &a.ID

	if err := svc.Update(ctx, loaded, nil); !errors.Is(err, domain.ErrEmployeeManagerCycle) {
		t.Fatalf("Update() = %v, want ErrEmployeeManagerCycle", err)
	}
}

// TestUpdateAcceptsAManagerOutsideTheOwnChain confirms the cycle check does
// not reject legitimate assignments.
func TestUpdateAcceptsAManagerOutsideTheOwnChain(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	ctx := context.Background()
	companyID := uuid.New()

	boss := repo.put(domain.Employee{
		TenantModel: domain.TenantModel{CompanyID: companyID},
		EmployeeNo:  "BOSS", Name: "사장", HireDate: time.Now(),
		Status:         domain.EmployeeStatusActive,
		EmploymentType: domain.EmploymentTypeRegular,
	})

	employee := newEmployeeFixture(companyID)
	if err := svc.Create(ctx, employee, ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	loaded, err := svc.GetByID(ctx, companyID, employee.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	loaded.ManagerID = &boss.ID

	if err := svc.Update(ctx, loaded, nil); err != nil {
		t.Fatalf("Update() = %v, want nil", err)
	}
	stored := repo.employees[employee.ID]
	if stored.ManagerID == nil || *stored.ManagerID != boss.ID {
		t.Error("the manager was not stored")
	}
}

// ---------------------------------------------------------------------------
// Separation and retention
// ---------------------------------------------------------------------------

// TestDeleteIsRefusedWhenWageHistoryExists is the retention rule.
//
// payrolls, employee_salaries, employee_insurance and
// insurance_monthly_contributions all reference employees with ON DELETE
// RESTRICT because the wage ledger must be kept for three years. The service
// has to answer with something a user can act on, and point at the
// resignation path instead of at a foreign key violation.
func TestDeleteIsRefusedWhenWageHistoryExists(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	ctx := context.Background()
	companyID := uuid.New()
	employee := newEmployeeFixture(companyID)
	if err := svc.Create(ctx, employee, ""); err != nil {
		t.Fatalf("Create: %v", err)
	}
	repo.retained[employee.ID] = repository.EmployeeRetainedRecords{Payrolls: 12}

	canDelete, reason, err := svc.CanDelete(ctx, companyID, employee.ID)
	if err != nil {
		t.Fatalf("CanDelete: %v", err)
	}
	if canDelete {
		t.Error("an employee with payroll history must not be deletable")
	}
	if reason == "" {
		t.Error("CanDelete must explain why not")
	}

	if err := svc.Delete(ctx, companyID, employee.ID); !errors.Is(err, domain.ErrEmployeeHasHistory) {
		t.Fatalf("Delete() = %v, want ErrEmployeeHasHistory", err)
	}
	if len(repo.deleted) != 0 {
		t.Error("the repository was asked to delete anyway")
	}
	if _, ok := repo.employees[employee.ID]; !ok {
		t.Error("the employee row was removed despite the refusal")
	}
}

// TestDeleteIsRefusedWhileOthersReportToTheEmployee: manager_id would be left
// pointing at a soft-deleted row.
func TestDeleteIsRefusedWhileOthersReportToTheEmployee(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	ctx := context.Background()
	companyID := uuid.New()
	employee := newEmployeeFixture(companyID)
	if err := svc.Create(ctx, employee, ""); err != nil {
		t.Fatalf("Create: %v", err)
	}
	repo.reports[employee.ID] = 3

	if err := svc.Delete(ctx, companyID, employee.ID); err == nil {
		t.Fatal("Delete() = nil, want a refusal")
	}
	if len(repo.deleted) != 0 {
		t.Error("the repository was asked to delete anyway")
	}
}

// TestDeleteSucceedsForAnEmployeeWithNoHistory keeps the refusals honest.
func TestDeleteSucceedsForAnEmployeeWithNoHistory(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	ctx := context.Background()
	companyID := uuid.New()
	employee := newEmployeeFixture(companyID)
	if err := svc.Create(ctx, employee, ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Delete(ctx, companyID, employee.ID); err != nil {
		t.Fatalf("Delete() = %v, want nil", err)
	}
	if len(repo.deleted) != 1 {
		t.Error("the repository was not asked to delete")
	}
}

// TestResignationKeepsTheRecord: the separation path changes a status and
// records a date. It must not delete anything.
func TestResignationKeepsTheRecord(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	ctx := context.Background()
	companyID := uuid.New()
	employee := newEmployeeFixture(companyID)
	if err := svc.Create(ctx, employee, ""); err != nil {
		t.Fatalf("Create: %v", err)
	}
	repo.retained[employee.ID] = repository.EmployeeRetainedRecords{Payrolls: 12, Salaries: 3}

	resigned := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	change := service.EmployeeStatusChange{
		Status:          domain.EmployeeStatusResigned,
		ResignationDate: &resigned,
		Reason:          "개인 사정",
		ActorID:         uuid.New(),
	}
	if err := svc.ChangeStatus(ctx, companyID, employee.ID, change); err != nil {
		t.Fatalf("ChangeStatus() = %v, want nil", err)
	}

	stored := repo.employees[employee.ID]
	if stored.Status != domain.EmployeeStatusResigned {
		t.Errorf("status = %q, want resigned", stored.Status)
	}
	if stored.ResignationDate == nil || !stored.ResignationDate.Equal(resigned) {
		t.Error("the resignation date was not recorded")
	}
	if stored.ResignationReason != "개인 사정" {
		t.Errorf("reason = %q", stored.ResignationReason)
	}
	if len(repo.deleted) != 0 {
		t.Error("a resignation must not delete anything")
	}
}

// TestResignationRequiresADate: a separation with no date is a record nobody
// can use for a severance or an insurance loss report.
func TestResignationRequiresADate(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	ctx := context.Background()
	companyID := uuid.New()
	employee := newEmployeeFixture(companyID)
	if err := svc.Create(ctx, employee, ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	change := service.EmployeeStatusChange{Status: domain.EmployeeStatusResigned}
	if err := svc.ChangeStatus(ctx, companyID, employee.ID, change); !errors.Is(err, domain.ErrEmployeeResignationDate) {
		t.Fatalf("ChangeStatus() = %v, want ErrEmployeeResignationDate", err)
	}
	if repo.employees[employee.ID].Status != domain.EmployeeStatusActive {
		t.Error("the status changed despite the refusal")
	}
}

// TestReinstatementClearsTheResignation: a row must not claim both to be
// employed and to have left.
func TestReinstatementClearsTheResignation(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	ctx := context.Background()
	companyID := uuid.New()
	employee := newEmployeeFixture(companyID)
	if err := svc.Create(ctx, employee, ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	resigned := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	if err := svc.ChangeStatus(ctx, companyID, employee.ID, service.EmployeeStatusChange{
		Status: domain.EmployeeStatusResigned, ResignationDate: &resigned, Reason: "이직",
	}); err != nil {
		t.Fatalf("resignation: %v", err)
	}

	if err := svc.ChangeStatus(ctx, companyID, employee.ID, service.EmployeeStatusChange{
		Status: domain.EmployeeStatusActive,
	}); err != nil {
		t.Fatalf("reinstatement: %v", err)
	}

	stored := repo.employees[employee.ID]
	if stored.Status != domain.EmployeeStatusActive {
		t.Errorf("status = %q, want active", stored.Status)
	}
	if stored.ResignationDate != nil || stored.ResignationReason != "" {
		t.Error("the resignation was not cleared on reinstatement")
	}
}

// TestTerminationIsFinal: "terminated" has no way back, so the transition
// table has to refuse it rather than leaving it to a caller's discipline.
func TestTerminationIsFinal(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	ctx := context.Background()
	companyID := uuid.New()
	employee := newEmployeeFixture(companyID)
	if err := svc.Create(ctx, employee, ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	terminated := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	if err := svc.ChangeStatus(ctx, companyID, employee.ID, service.EmployeeStatusChange{
		Status: domain.EmployeeStatusTerminated, ResignationDate: &terminated, Reason: "징계해고",
	}); err != nil {
		t.Fatalf("termination: %v", err)
	}

	err := svc.ChangeStatus(ctx, companyID, employee.ID, service.EmployeeStatusChange{
		Status: domain.EmployeeStatusActive,
	})
	if !errors.Is(err, domain.ErrEmployeeInvalidTransition) {
		t.Fatalf("ChangeStatus() = %v, want ErrEmployeeInvalidTransition", err)
	}
}

// TestDuplicateEmployeeNumberIsRefused within one company, while another
// company may use the same number.
func TestDuplicateEmployeeNumberIsRefused(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	ctx := context.Background()
	companyID := uuid.New()

	if err := svc.Create(ctx, newEmployeeFixture(companyID), ""); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Create(ctx, newEmployeeFixture(companyID), ""); !errors.Is(err, domain.ErrEmployeeNoExists) {
		t.Fatalf("Create() = %v, want ErrEmployeeNoExists", err)
	}
	if err := svc.Create(ctx, newEmployeeFixture(uuid.New()), ""); err != nil {
		t.Fatalf("another company must be free to use EMP001, got %v", err)
	}
}

// TestListClampsThePageSize: page_size=0 must never reach a LIMIT or the
// total-pages division.
func TestListClampsThePageSize(t *testing.T) {
	repo := newFakeEmployeeRepo()
	svc := service.NewEmployeeService(repo, newFakePositionRepo(), testCipher(t))

	filter := &service.EmployeeFilter{CompanyID: uuid.New(), Page: 0, PageSize: 0}
	if _, _, err := svc.List(context.Background(), filter); err != nil {
		t.Fatalf("List: %v", err)
	}
	if filter.Page < 1 || filter.PageSize < 1 {
		t.Errorf("List left page=%d page_size=%d unclamped", filter.Page, filter.PageSize)
	}

	filter = &service.EmployeeFilter{CompanyID: uuid.New(), Page: 1, PageSize: 100000}
	if _, _, err := svc.List(context.Background(), filter); err != nil {
		t.Fatalf("List: %v", err)
	}
	if filter.PageSize > 100 {
		t.Errorf("List left an unbounded page size of %d", filter.PageSize)
	}
}
