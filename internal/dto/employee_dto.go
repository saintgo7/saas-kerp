package dto

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// Layouts used by the HR payloads. Dates are calendar dates with no time of
// day; timestamps are RFC 3339.
const (
	hrTimeFormat = "2006-01-02T15:04:05Z07:00"
	hrDateFormat = "2006-01-02"
)

// ErrInvalidHRDate is returned when a date field is present but unparseable.
// Silently dropping it (the older handlers' habit) turns a client typo into a
// record with a missing hire date.
var ErrInvalidHRDate = errors.New("date must be formatted as YYYY-MM-DD")

// parseHRDate parses an optional calendar date.
func parseHRDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(hrDateFormat, value)
	if err != nil {
		return nil, ErrInvalidHRDate
	}
	return &parsed, nil
}

// formatHRDate renders an optional calendar date.
func formatHRDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(hrDateFormat)
	return &formatted
}

// parseOptionalUUID parses an optional identifier.
func parseOptionalUUID(value string) (*uuid.UUID, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// EmployeeResponse represents an employee in API responses.
//
// There is no resident number field and there never may be. The stored value
// is ciphertext (employees.resident_number_enc) and the API says only whether
// one is on file. A client that needs to show a masked identity renders it
// from birth_date and gender, which are separate, far less sensitive columns.
type EmployeeResponse struct {
	ID         string `json:"id"`
	EmployeeNo string `json:"employee_no"`
	Name       string `json:"name"`
	NameEn     string `json:"name_en,omitempty"`
	UserID     string `json:"user_id,omitempty"`

	HasResidentNumber bool `json:"has_resident_number"`

	BirthDate   *string `json:"birth_date,omitempty"`
	Gender      string  `json:"gender,omitempty"`
	Nationality string  `json:"nationality,omitempty"`

	Phone            string `json:"phone,omitempty"`
	Mobile           string `json:"mobile,omitempty"`
	Email            string `json:"email,omitempty"`
	EmergencyContact string `json:"emergency_contact,omitempty"`
	EmergencyPhone   string `json:"emergency_phone,omitempty"`

	ZipCode       string `json:"zip_code,omitempty"`
	Address       string `json:"address,omitempty"`
	AddressDetail string `json:"address_detail,omitempty"`

	DepartmentID   string `json:"department_id,omitempty"`
	DepartmentName string `json:"department_name,omitempty"`
	PositionID     string `json:"position_id,omitempty"`
	PositionName   string `json:"position_name,omitempty"`
	ManagerID      string `json:"manager_id,omitempty"`
	ManagerName    string `json:"manager_name,omitempty"`

	HireDate          string  `json:"hire_date"`
	ProbationEndDate  *string `json:"probation_end_date,omitempty"`
	ResignationDate   *string `json:"resignation_date,omitempty"`
	ResignationReason string  `json:"resignation_reason,omitempty"`

	EmploymentType    string  `json:"employment_type"`
	ContractStartDate *string `json:"contract_start_date,omitempty"`
	ContractEndDate   *string `json:"contract_end_date,omitempty"`

	WorkLocation string `json:"work_location,omitempty"`
	WorkEmail    string `json:"work_email,omitempty"`
	WorkPhone    string `json:"work_phone,omitempty"`

	Status string `json:"status"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// FromEmployee converts domain.Employee to EmployeeResponse.
func FromEmployee(employee *domain.Employee) EmployeeResponse {
	resp := EmployeeResponse{
		ID:                employee.ID.String(),
		EmployeeNo:        employee.EmployeeNo,
		Name:              employee.Name,
		NameEn:            employee.NameEn,
		HasResidentNumber: employee.HasResidentNumber(),
		BirthDate:         formatHRDate(employee.BirthDate),
		Gender:            employee.Gender,
		Nationality:       employee.Nationality,
		Phone:             employee.Phone,
		Mobile:            employee.Mobile,
		Email:             employee.Email,
		EmergencyContact:  employee.EmergencyContact,
		EmergencyPhone:    employee.EmergencyPhone,
		ZipCode:           employee.ZipCode,
		Address:           employee.Address,
		AddressDetail:     employee.AddressDetail,
		HireDate:          employee.HireDate.Format(hrDateFormat),
		ProbationEndDate:  formatHRDate(employee.ProbationEndDate),
		ResignationDate:   formatHRDate(employee.ResignationDate),
		ResignationReason: employee.ResignationReason,
		EmploymentType:    string(employee.EmploymentType),
		ContractStartDate: formatHRDate(employee.ContractStartDate),
		ContractEndDate:   formatHRDate(employee.ContractEndDate),
		WorkLocation:      employee.WorkLocation,
		WorkEmail:         employee.WorkEmail,
		WorkPhone:         employee.WorkPhone,
		Status:            string(employee.Status),
		CreatedAt:         employee.CreatedAt.Format(hrTimeFormat),
		UpdatedAt:         employee.UpdatedAt.Format(hrTimeFormat),
	}

	if employee.UserID != nil {
		resp.UserID = employee.UserID.String()
	}
	if employee.DepartmentID != nil {
		resp.DepartmentID = employee.DepartmentID.String()
	}
	if employee.Department != nil {
		resp.DepartmentName = employee.Department.Name
	}
	if employee.PositionID != nil {
		resp.PositionID = employee.PositionID.String()
	}
	if employee.Position != nil {
		resp.PositionName = employee.Position.Name
	}
	if employee.ManagerID != nil {
		resp.ManagerID = employee.ManagerID.String()
	}
	if employee.Manager != nil {
		resp.ManagerName = employee.Manager.Name
	}

	return resp
}

// FromEmployees converts []domain.Employee to []EmployeeResponse.
func FromEmployees(employees []domain.Employee) []EmployeeResponse {
	responses := make([]EmployeeResponse, len(employees))
	for i := range employees {
		responses[i] = FromEmployee(&employees[i])
	}
	return responses
}

// EmployeeRequest carries the writable fields of an employee.
//
// ResidentNumber is write-only: it is accepted here, encrypted by the service
// and never returned by any response type. Handlers must not echo this struct
// back to the client.
type EmployeeRequest struct {
	EmployeeNo string `json:"employee_no" binding:"required,max=20"`
	Name       string `json:"name" binding:"required,max=50"`
	NameEn     string `json:"name_en,omitempty" binding:"max=100"`
	UserID     string `json:"user_id,omitempty" binding:"omitempty,uuid"`

	// ResidentNumber is the plaintext 주민등록번호, with or without the hyphen.
	// Omit the field to leave the stored value alone; send "" to erase it.
	ResidentNumber *string `json:"resident_number,omitempty" binding:"omitempty,max=14"`

	BirthDate   string `json:"birth_date,omitempty"`
	Gender      string `json:"gender,omitempty" binding:"omitempty,oneof=male female other"`
	Nationality string `json:"nationality,omitempty" binding:"max=50"`

	Phone            string `json:"phone,omitempty" binding:"max=20"`
	Mobile           string `json:"mobile,omitempty" binding:"max=20"`
	Email            string `json:"email,omitempty" binding:"omitempty,email,max=100"`
	EmergencyContact string `json:"emergency_contact,omitempty" binding:"max=100"`
	EmergencyPhone   string `json:"emergency_phone,omitempty" binding:"max=20"`

	ZipCode       string `json:"zip_code,omitempty" binding:"max=10"`
	Address       string `json:"address,omitempty" binding:"max=200"`
	AddressDetail string `json:"address_detail,omitempty" binding:"max=100"`

	DepartmentID string `json:"department_id,omitempty" binding:"omitempty,uuid"`
	PositionID   string `json:"position_id,omitempty" binding:"omitempty,uuid"`
	ManagerID    string `json:"manager_id,omitempty" binding:"omitempty,uuid"`

	HireDate          string `json:"hire_date" binding:"required"`
	ProbationEndDate  string `json:"probation_end_date,omitempty"`
	ResignationDate   string `json:"resignation_date,omitempty"`
	ResignationReason string `json:"resignation_reason,omitempty" binding:"max=200"`

	EmploymentType    string `json:"employment_type,omitempty" binding:"omitempty,oneof=regular contract part_time intern dispatch"`
	ContractStartDate string `json:"contract_start_date,omitempty"`
	ContractEndDate   string `json:"contract_end_date,omitempty"`

	WorkLocation string `json:"work_location,omitempty" binding:"max=100"`
	WorkEmail    string `json:"work_email,omitempty" binding:"omitempty,email,max=100"`
	WorkPhone    string `json:"work_phone,omitempty" binding:"max=20"`

	Status string `json:"status,omitempty" binding:"omitempty,oneof=active on_leave resigned terminated"`
}

// ApplyTo writes the request onto an employee.
//
// It never touches ResidentNumberEnc: the plaintext travels separately, so a
// mapping mistake here cannot overwrite an encrypted column with a readable
// value. It also never touches CompanyID.
func (r *EmployeeRequest) ApplyTo(employee *domain.Employee) error {
	hireDate, err := parseHRDate(r.HireDate)
	if err != nil {
		return err
	}
	if hireDate == nil {
		return ErrInvalidHRDate
	}
	birthDate, err := parseHRDate(r.BirthDate)
	if err != nil {
		return err
	}
	probationEnd, err := parseHRDate(r.ProbationEndDate)
	if err != nil {
		return err
	}
	resignationDate, err := parseHRDate(r.ResignationDate)
	if err != nil {
		return err
	}
	contractStart, err := parseHRDate(r.ContractStartDate)
	if err != nil {
		return err
	}
	contractEnd, err := parseHRDate(r.ContractEndDate)
	if err != nil {
		return err
	}

	userID, err := parseOptionalUUID(r.UserID)
	if err != nil {
		return err
	}
	departmentID, err := parseOptionalUUID(r.DepartmentID)
	if err != nil {
		return err
	}
	positionID, err := parseOptionalUUID(r.PositionID)
	if err != nil {
		return err
	}
	managerID, err := parseOptionalUUID(r.ManagerID)
	if err != nil {
		return err
	}

	employee.EmployeeNo = r.EmployeeNo
	employee.Name = r.Name
	employee.NameEn = r.NameEn
	employee.UserID = userID
	employee.BirthDate = birthDate
	employee.Gender = r.Gender
	employee.Nationality = r.Nationality
	employee.Phone = r.Phone
	employee.Mobile = r.Mobile
	employee.Email = r.Email
	employee.EmergencyContact = r.EmergencyContact
	employee.EmergencyPhone = r.EmergencyPhone
	employee.ZipCode = r.ZipCode
	employee.Address = r.Address
	employee.AddressDetail = r.AddressDetail
	employee.DepartmentID = departmentID
	employee.PositionID = positionID
	employee.ManagerID = managerID
	employee.HireDate = *hireDate
	employee.ProbationEndDate = probationEnd
	employee.ResignationDate = resignationDate
	employee.ResignationReason = r.ResignationReason
	employee.ContractStartDate = contractStart
	employee.ContractEndDate = contractEnd
	employee.WorkLocation = r.WorkLocation
	employee.WorkEmail = r.WorkEmail
	employee.WorkPhone = r.WorkPhone

	if r.EmploymentType != "" {
		employee.EmploymentType = domain.EmploymentType(r.EmploymentType)
	}
	if r.Status != "" {
		employee.Status = domain.EmployeeStatus(r.Status)
	}

	return nil
}

// EmployeeStatusRequest changes an employee's employment status.
type EmployeeStatusRequest struct {
	Status          string `json:"status" binding:"required,oneof=active on_leave resigned terminated"`
	ResignationDate string `json:"resignation_date,omitempty"`
	Reason          string `json:"reason,omitempty" binding:"max=200"`
}

// ResignationDateValue parses the optional resignation date.
func (r *EmployeeStatusRequest) ResignationDateValue() (*time.Time, error) {
	return parseHRDate(r.ResignationDate)
}

// EmployeeStatsResponse represents headcount statistics.
type EmployeeStatsResponse struct {
	TotalCount      int64 `json:"total_count"`
	ActiveCount     int64 `json:"active_count"`
	OnLeaveCount    int64 `json:"on_leave_count"`
	ResignedCount   int64 `json:"resigned_count"`
	TerminatedCount int64 `json:"terminated_count"`
}
