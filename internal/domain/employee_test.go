package domain_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
)

func hireDate() time.Time {
	return time.Date(2020, 3, 2, 0, 0, 0, 0, time.UTC)
}

func newValidEmployee() *domain.Employee {
	return &domain.Employee{
		TenantModel: domain.TenantModel{
			BaseModel: domain.BaseModel{ID: uuid.New()},
			CompanyID: uuid.New(),
		},
		EmployeeNo:     "EMP001",
		Name:           "홍길동",
		HireDate:       hireDate(),
		Status:         domain.EmployeeStatusActive,
		EmploymentType: domain.EmploymentTypeRegular,
	}
}

// TestEmployeeNeverSerialisesTheResidentNumber is the test that matters most
// here: a 주민등록번호 must not leave the process, not even as ciphertext.
//
// The column is BYTEA and the field is tagged json:"-". Without that tag,
// encoding/json emits the ciphertext base64-encoded under "ResidentNumberEnc"
// for every handler that marshals a domain object directly.
func TestEmployeeNeverSerialisesTheResidentNumber(t *testing.T) {
	employee := newValidEmployee()
	employee.ResidentNumberEnc = []byte("ciphertext-that-must-not-be-shipped")

	encoded, err := json.Marshal(employee)
	if err != nil {
		t.Fatalf("marshalling an employee: %v", err)
	}

	body := string(encoded)
	for _, forbidden := range []string{
		"resident",
		"ResidentNumber",
		"ciphertext-that-must-not-be-shipped",
	} {
		if strings.Contains(strings.ToLower(body), strings.ToLower(forbidden)) {
			t.Errorf("the JSON form of an employee contains %q:\n\t%s", forbidden, body)
		}
	}
}

// TestEmployeeResponseNeverCarriesTheResidentNumber checks the same thing one
// layer up: the API type reports only whether one is on file.
func TestEmployeeResponseNeverCarriesTheResidentNumber(t *testing.T) {
	employee := newValidEmployee()
	employee.ResidentNumberEnc = []byte("ciphertext")

	resp := dto.FromEmployee(employee)
	if !resp.HasResidentNumber {
		t.Error("has_resident_number should be true when ciphertext is stored")
	}

	encoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshalling the response: %v", err)
	}
	body := strings.ToLower(string(encoded))

	if strings.Contains(body, "ciphertext") {
		t.Errorf("the response leaks the stored value:\n\t%s", body)
	}
	// "has_resident_number" is the only field allowed to mention it.
	stripped := strings.ReplaceAll(body, "has_resident_number", "")
	if strings.Contains(stripped, "resident") {
		t.Errorf("the response has a resident number field beyond the flag:\n\t%s", body)
	}
}

// TestEmployeeCannotBeItsOwnManager pins the self-reference guard.
//
// The foreign key employees.manager_id -> employees(id) is satisfied by a row
// that points at itself, so nothing in the database rejects one. Left in
// place, every walk up the reporting line loops forever - the same defect that
// was fixed for departments.
func TestEmployeeCannotBeItsOwnManager(t *testing.T) {
	employee := newValidEmployee()
	employee.ManagerID = &employee.ID

	if err := employee.Validate(); err != domain.ErrEmployeeSelfManager {
		t.Fatalf("Validate() = %v, want ErrEmployeeSelfManager", err)
	}

	other := uuid.New()
	employee.ManagerID = &other
	if err := employee.Validate(); err != nil {
		t.Fatalf("a different manager must be accepted, got %v", err)
	}
}

func TestEmployeeValidate(t *testing.T) {
	resignation := hireDate().AddDate(2, 0, 0)
	beforeHire := hireDate().AddDate(-1, 0, 0)

	cases := []struct {
		name   string
		mutate func(*domain.Employee)
		want   error
	}{
		{"a complete employee is valid", func(*domain.Employee) {}, nil},
		{"employee number is required", func(e *domain.Employee) { e.EmployeeNo = "  " }, domain.ErrEmployeeNoEmpty},
		{"name is required", func(e *domain.Employee) { e.Name = "" }, domain.ErrEmployeeNameEmpty},
		{"hire date is required", func(e *domain.Employee) { e.HireDate = time.Time{} }, domain.ErrEmployeeHireDateEmpty},
		{"status must be one the database accepts", func(e *domain.Employee) { e.Status = "leave" }, domain.ErrEmployeeInvalidStatus},
		{"employment type must be one the database accepts", func(e *domain.Employee) { e.EmploymentType = "parttime" }, domain.ErrEmployeeInvalidType},
		{"gender must be one the database accepts", func(e *domain.Employee) { e.Gender = "M" }, domain.ErrEmployeeInvalidGender},
		{"gender may be empty", func(e *domain.Employee) { e.Gender = "" }, nil},
		{"a resigned employee needs a resignation date", func(e *domain.Employee) {
			e.Status = domain.EmployeeStatusResigned
		}, domain.ErrEmployeeResignationDate},
		{"a resigned employee with a date is valid", func(e *domain.Employee) {
			e.Status = domain.EmployeeStatusResigned
			e.ResignationDate = &resignation
		}, nil},
		{"resignation cannot precede the hire date", func(e *domain.Employee) {
			e.Status = domain.EmployeeStatusResigned
			e.ResignationDate = &beforeHire
		}, domain.ErrEmployeeResignedTooEarly},
		{"contract end cannot precede contract start", func(e *domain.Employee) {
			start := hireDate()
			end := beforeHire
			e.ContractStartDate = &start
			e.ContractEndDate = &end
		}, domain.ErrEmployeeContractDates},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			employee := newValidEmployee()
			tc.mutate(employee)
			if err := employee.Validate(); err != tc.want {
				t.Errorf("Validate() = %v, want %v", err, tc.want)
			}
		})
	}
}

// TestEmployeeStatusTransitions pins the separation rules. A terminated
// employee is final; a resignation may be corrected back to active.
func TestEmployeeStatusTransitions(t *testing.T) {
	cases := []struct {
		from, to domain.EmployeeStatus
		allowed  bool
	}{
		{domain.EmployeeStatusActive, domain.EmployeeStatusActive, true},
		{domain.EmployeeStatusActive, domain.EmployeeStatusOnLeave, true},
		{domain.EmployeeStatusActive, domain.EmployeeStatusResigned, true},
		{domain.EmployeeStatusActive, domain.EmployeeStatusTerminated, true},
		{domain.EmployeeStatusOnLeave, domain.EmployeeStatusActive, true},
		{domain.EmployeeStatusOnLeave, domain.EmployeeStatusResigned, true},
		{domain.EmployeeStatusResigned, domain.EmployeeStatusActive, true},
		{domain.EmployeeStatusResigned, domain.EmployeeStatusOnLeave, false},
		{domain.EmployeeStatusResigned, domain.EmployeeStatusTerminated, false},
		{domain.EmployeeStatusTerminated, domain.EmployeeStatusActive, false},
		{domain.EmployeeStatusActive, "leave", false},
	}

	for _, tc := range cases {
		if got := tc.from.CanTransitionTo(tc.to); got != tc.allowed {
			t.Errorf("%q -> %q = %v, want %v", tc.from, tc.to, got, tc.allowed)
		}
	}
}

// TestNormalizeResidentNumber checks the shape validation. The check digit is
// deliberately not verified: numbers issued from October 2020 have a random
// tail, so a checksum rule would reject valid new numbers.
func TestNormalizeResidentNumber(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{"hyphenated", "900315-1234567", "9003151234567", true},
		{"unhyphenated", "9003151234567", "9003151234567", true},
		{"surrounding whitespace", "  900315-1234567 ", "9003151234567", true},
		{"foreign resident series", "900315-5234567", "9003155234567", true},
		{"too short", "900315-123456", "", false},
		{"too long", "900315-12345678", "", false},
		{"not digits", "900315-12345A7", "", false},
		{"impossible month", "901315-1234567", "", false},
		{"impossible day", "900332-1234567", "", false},
		{"empty", "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.NormalizeResidentNumber(tc.input)
			if tc.ok {
				if err != nil {
					t.Fatalf("NormalizeResidentNumber(%q) failed: %v", tc.input, err)
				}
				if got != tc.want {
					t.Errorf("NormalizeResidentNumber(%q) = %q, want %q", tc.input, got, tc.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("NormalizeResidentNumber(%q) = %q, want an error", tc.input, got)
			}
			if got != "" {
				t.Errorf("a rejected value must not be returned, got %q", got)
			}
		})
	}
}
