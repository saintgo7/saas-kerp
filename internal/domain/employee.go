package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Employee errors
var (
	ErrEmployeeNotFound          = errors.New("employee not found")
	ErrEmployeeNoExists          = errors.New("employee number already exists")
	ErrEmployeeNoEmpty           = errors.New("employee number is required")
	ErrEmployeeNameEmpty         = errors.New("employee name is required")
	ErrEmployeeHireDateEmpty     = errors.New("hire date is required")
	ErrEmployeeSelfManager       = errors.New("an employee cannot be their own manager")
	ErrEmployeeManagerCycle      = errors.New("manager assignment creates a reporting cycle")
	ErrEmployeeManagerNotFound   = errors.New("manager not found")
	ErrEmployeeInvalidStatus     = errors.New("invalid employee status")
	ErrEmployeeInvalidType       = errors.New("invalid employment type")
	ErrEmployeeInvalidGender     = errors.New("invalid gender")
	ErrEmployeeInvalidTransition = errors.New("invalid employee status transition")
	ErrEmployeeResignationDate   = errors.New("resignation date is required when the employee is resigned or terminated")
	ErrEmployeeResignedTooEarly  = errors.New("resignation date cannot precede the hire date")
	ErrEmployeeContractDates     = errors.New("contract end date cannot precede the contract start date")
	ErrEmployeeHasHistory        = errors.New("employee has payroll or insurance history and cannot be deleted")
	ErrEmployeeDepartmentUnknown = errors.New("department not found")
	ErrEmployeePositionUnknown   = errors.New("position not found")

	// ErrResidentNumberFormat is returned for anything that is not 13 digits
	// with a plausible birth-date/gender prefix. Resident numbers issued after
	// 2020-10 have a random tail, so the historical check digit is NOT verified.
	ErrResidentNumberFormat = errors.New("invalid resident registration number")
)

// EmployeeStatus is the employment state of an employee.
//
// The values are exactly the CHECK constraint in
// db/migrations/000006_hr_tables.up.sql. The frontend mock data uses "leave"
// where the database says "on_leave"; the database is authoritative.
type EmployeeStatus string

const (
	EmployeeStatusActive     EmployeeStatus = "active"
	EmployeeStatusOnLeave    EmployeeStatus = "on_leave"
	EmployeeStatusResigned   EmployeeStatus = "resigned"
	EmployeeStatusTerminated EmployeeStatus = "terminated"
)

// IsValid reports whether the status is one the database accepts.
func (s EmployeeStatus) IsValid() bool {
	switch s {
	case EmployeeStatusActive, EmployeeStatusOnLeave, EmployeeStatusResigned, EmployeeStatusTerminated:
		return true
	}
	return false
}

// IsSeparated reports whether the employee has left the company.
func (s EmployeeStatus) IsSeparated() bool {
	return s == EmployeeStatusResigned || s == EmployeeStatusTerminated
}

// CanTransitionTo reports whether a move from s to next is allowed.
//
// A separation is never undone by editing the record: correcting a mistaken
// resignation goes back to "active" explicitly (and the service clears the
// resignation date with it), while "terminated" is final. Both separated states
// keep every payroll, salary and insurance row - those are retained for three
// years under the Labor Standards Act, and the schema enforces it with
// ON DELETE RESTRICT (db/migrations/000019_retention_and_amount_checks).
func (s EmployeeStatus) CanTransitionTo(next EmployeeStatus) bool {
	if !s.IsValid() || !next.IsValid() {
		return false
	}
	if s == next {
		return true
	}
	switch s {
	case EmployeeStatusActive:
		return next == EmployeeStatusOnLeave || next.IsSeparated()
	case EmployeeStatusOnLeave:
		return next == EmployeeStatusActive || next.IsSeparated()
	case EmployeeStatusResigned:
		return next == EmployeeStatusActive
	case EmployeeStatusTerminated:
		return false
	}
	return false
}

// EmploymentType is the contractual form of the employment relationship.
type EmploymentType string

const (
	EmploymentTypeRegular  EmploymentType = "regular"
	EmploymentTypeContract EmploymentType = "contract"
	EmploymentTypePartTime EmploymentType = "part_time"
	EmploymentTypeIntern   EmploymentType = "intern"
	EmploymentTypeDispatch EmploymentType = "dispatch"
)

// IsValid reports whether the employment type is one the database accepts.
func (t EmploymentType) IsValid() bool {
	switch t {
	case EmploymentTypeRegular, EmploymentTypeContract, EmploymentTypePartTime,
		EmploymentTypeIntern, EmploymentTypeDispatch:
		return true
	}
	return false
}

// Gender values accepted by the employees CHECK constraint.
const (
	GenderMale   = "male"
	GenderFemale = "female"
	GenderOther  = "other"
)

// IsValidGender reports whether gender is storable. The empty string is allowed
// because the column is nullable.
func IsValidGender(gender string) bool {
	switch gender {
	case "", GenderMale, GenderFemale, GenderOther:
		return true
	}
	return false
}

// MaxManagerChainDepth bounds the walk up the reporting line. A cycle that
// predates this code (or one introduced straight in SQL) must not turn a
// validation call into an infinite loop.
const MaxManagerChainDepth = 64

// Employee represents one person employed by a company.
//
// Table: employees. Every field maps to a column that exists in
// db/migrations/000006_hr_tables.up.sql; nothing here is invented.
type Employee struct {
	TenantModel

	// DeletedAt enables GORM soft delete. The table carries deleted_at and
	// db/migrations/000020_index_hygiene makes (company_id, employee_no)
	// unique only WHERE deleted_at IS NULL, so re-hiring a person with their
	// old employee number works. It must be gorm.DeletedAt and not
	// *time.Time: only that type makes Delete an UPDATE and appends
	// `deleted_at IS NULL` to every query. Tagged json:"-" because
	// gorm.DeletedAt marshals as {"Time":...,"Valid":...}.
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Optional link to a login account.
	UserID *uuid.UUID `gorm:"type:uuid" json:"user_id,omitempty"`

	// Identity
	EmployeeNo string `gorm:"type:varchar(20);not null" json:"employee_no"`
	Name       string `gorm:"type:varchar(50);not null" json:"name"`
	NameEn     string `gorm:"type:varchar(100)" json:"name_en,omitempty"`

	// ResidentNumberEnc holds the ENCRYPTED Korean resident registration
	// number (주민등록번호). It is a []byte of ciphertext, never a readable
	// value, and it is tagged json:"-" so that no handler can leak it by
	// serialising the domain object: the API answers with a boolean
	// (has_resident_number) instead. See service.ResidentNumberCipher for the
	// encryption, which is fail-closed - with no key configured the service
	// refuses to store a resident number rather than storing it in the clear.
	ResidentNumberEnc []byte `gorm:"column:resident_number_enc;type:bytea" json:"-"`

	BirthDate   *time.Time `gorm:"type:date" json:"birth_date,omitempty"`
	Gender      string     `gorm:"type:varchar(10)" json:"gender,omitempty"`
	Nationality string     `gorm:"type:varchar(50);default:'KR'" json:"nationality,omitempty"`

	// Contact
	Phone            string `gorm:"type:varchar(20)" json:"phone,omitempty"`
	Mobile           string `gorm:"type:varchar(20)" json:"mobile,omitempty"`
	Email            string `gorm:"type:varchar(100)" json:"email,omitempty"`
	EmergencyContact string `gorm:"type:varchar(100)" json:"emergency_contact,omitempty"`
	EmergencyPhone   string `gorm:"type:varchar(20)" json:"emergency_phone,omitempty"`

	// Address
	ZipCode       string `gorm:"type:varchar(10)" json:"zip_code,omitempty"`
	Address       string `gorm:"type:varchar(200)" json:"address,omitempty"`
	AddressDetail string `gorm:"type:varchar(100)" json:"address_detail,omitempty"`

	// Organization
	DepartmentID *uuid.UUID  `gorm:"type:uuid" json:"department_id,omitempty"`
	Department   *Department `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
	PositionID   *uuid.UUID  `gorm:"type:uuid" json:"position_id,omitempty"`
	Position     *Position   `gorm:"foreignKey:PositionID" json:"position,omitempty"`
	ManagerID    *uuid.UUID  `gorm:"type:uuid" json:"manager_id,omitempty"`
	Manager      *Employee   `gorm:"foreignKey:ManagerID" json:"manager,omitempty"`

	// Employment
	HireDate          time.Time  `gorm:"type:date;not null" json:"hire_date"`
	ProbationEndDate  *time.Time `gorm:"type:date" json:"probation_end_date,omitempty"`
	ResignationDate   *time.Time `gorm:"type:date" json:"resignation_date,omitempty"`
	ResignationReason string     `gorm:"type:varchar(200)" json:"resignation_reason,omitempty"`

	EmploymentType    EmploymentType `gorm:"type:varchar(20);default:'regular'" json:"employment_type"`
	ContractStartDate *time.Time     `gorm:"type:date" json:"contract_start_date,omitempty"`
	ContractEndDate   *time.Time     `gorm:"type:date" json:"contract_end_date,omitempty"`

	// Work
	WorkLocation string `gorm:"type:varchar(100)" json:"work_location,omitempty"`
	WorkEmail    string `gorm:"type:varchar(100)" json:"work_email,omitempty"`
	WorkPhone    string `gorm:"type:varchar(20)" json:"work_phone,omitempty"`

	// Status
	Status EmployeeStatus `gorm:"type:varchar(20);default:'active'" json:"status"`

	// Audit
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy *uuid.UUID `gorm:"type:uuid" json:"updated_by,omitempty"`
}

// TableName specifies the table name for GORM
func (Employee) TableName() string {
	return "employees"
}

// HasResidentNumber reports whether an encrypted resident number is on file.
// This is the only thing the API is allowed to say about it.
func (e *Employee) HasResidentNumber() bool {
	return len(e.ResidentNumberEnc) > 0
}

// IsActive reports whether the employee currently works for the company.
func (e *Employee) IsActive() bool {
	return e.Status == EmployeeStatusActive || e.Status == EmployeeStatusOnLeave
}

// Validate checks every invariant that neither the CHECK constraints nor the
// foreign keys can express on their own.
func (e *Employee) Validate() error {
	if strings.TrimSpace(e.EmployeeNo) == "" {
		return ErrEmployeeNoEmpty
	}
	if strings.TrimSpace(e.Name) == "" {
		return ErrEmployeeNameEmpty
	}
	if e.HireDate.IsZero() {
		return ErrEmployeeHireDateEmpty
	}
	if !e.Status.IsValid() {
		return ErrEmployeeInvalidStatus
	}
	if !e.EmploymentType.IsValid() {
		return ErrEmployeeInvalidType
	}
	if !IsValidGender(e.Gender) {
		return ErrEmployeeInvalidGender
	}

	// A self-referencing manager_id is accepted by the foreign key (the row
	// references itself and the constraint is satisfied), so nothing below the
	// service layer rejects it. Left in place it makes every walk up the
	// reporting line loop forever - the same defect that was fixed for
	// departments in internal/service/department_service.go.
	if e.ManagerID != nil && e.ID != uuid.Nil && *e.ManagerID == e.ID {
		return ErrEmployeeSelfManager
	}

	if e.Status.IsSeparated() && e.ResignationDate == nil {
		return ErrEmployeeResignationDate
	}
	if e.ResignationDate != nil && e.ResignationDate.Before(e.HireDate) {
		return ErrEmployeeResignedTooEarly
	}
	if e.ContractStartDate != nil && e.ContractEndDate != nil &&
		e.ContractEndDate.Before(*e.ContractStartDate) {
		return ErrEmployeeContractDates
	}

	return nil
}

// NormalizeResidentNumber strips the conventional hyphen and validates the
// shape of a Korean resident registration number.
//
// It returns the 13 digits with no separator. The value is a secret: callers
// must hand the result straight to the cipher and must never log it, store it,
// or place it in an error message.
func NormalizeResidentNumber(raw string) (string, error) {
	digits := strings.ReplaceAll(strings.TrimSpace(raw), "-", "")
	if len(digits) != 13 {
		return "", ErrResidentNumberFormat
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return "", ErrResidentNumberFormat
		}
	}

	// The seventh digit encodes century and gender. 0 and 9 are the 1800s
	// series, 1-4 the 1900s/2000s domestic series, 5-8 the foreign-resident
	// series. Nothing else is issued.
	switch digits[6] {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
	default:
		return "", ErrResidentNumberFormat
	}

	// Birth date part: MMDD must be plausible. The year is ambiguous without
	// the century digit, so only month and day are checked.
	month := int(digits[2]-'0')*10 + int(digits[3]-'0')
	day := int(digits[4]-'0')*10 + int(digits[5]-'0')
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return "", ErrResidentNumberFormat
	}

	return digits, nil
}
