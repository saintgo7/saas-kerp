package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// 4대보험 사업장·자격·보험료·신고 (Korean social insurance administration).
//
// The contribution ARITHMETIC is not here - it is in social_insurance.go and is
// invoked through CalculatePayroll. This file is the record-keeping side:
// which workplace is registered with which agency, which employee has acquired
// which qualification, what was assessed in a given month, and what has been
// reported to the four 공단.
//
// Amounts are int64 whole won, matching the DECIMAL(18,0) columns.

// Insurance errors.
var (
	ErrInsuranceWorkplaceNotFound = errors.New("insurance workplace not found")
	ErrInsuranceWorkplaceExists   = errors.New("a workplace with this business number already exists")
	ErrEmployeeInsuranceNotFound  = errors.New("employee insurance record not found")
	ErrInsuranceReportNotFound    = errors.New("insurance report not found")
	ErrInsuranceReportNotEditable = errors.New("insurance report cannot be modified in its current status")
	ErrInsuranceReportSubmitted   = errors.New("insurance report has already been submitted")
	ErrInsuranceAgencyInvalid     = errors.New("unknown insurance agency")
	ErrInsuranceReportTypeInvalid = errors.New("unknown insurance report type")
	ErrInsuranceContributionRange = errors.New("contribution month must be between 1 and 12")

	// ErrInsuranceEDIUnavailable is returned by every path that would have to
	// talk to a 공단. There is no Go client for the insurance-edi service, so
	// the alternative to this error is marking a report "submitted" when
	// nothing left the building. See the HANDOFF block at the bottom.
	ErrInsuranceEDIUnavailable = errors.New("4대보험 EDI 전송 경로가 아직 연결되지 않았습니다")
)

// InsuranceAgency identifies one of the four 공단.
//
// comwel (근로복지공단) administers 고용보험 and 산재보험 together, which is why
// the credentials and EDI tables accept it alongside ei and wci.
type InsuranceAgency string

const (
	InsuranceAgencyNPS    InsuranceAgency = "nps"    // 국민연금공단
	InsuranceAgencyNHIS   InsuranceAgency = "nhis"   // 국민건강보험공단
	InsuranceAgencyEI     InsuranceAgency = "ei"     // 고용보험
	InsuranceAgencyWCI    InsuranceAgency = "wci"    // 산재보험
	InsuranceAgencyComwel InsuranceAgency = "comwel" // 근로복지공단
)

// IsValidReportAgency reports whether the agency may appear on a report. The
// insurance_reports CHECK constraint does not accept comwel.
func IsValidReportAgency(a InsuranceAgency) bool {
	switch a {
	case InsuranceAgencyNPS, InsuranceAgencyNHIS, InsuranceAgencyEI, InsuranceAgencyWCI:
		return true
	default:
		return false
	}
}

// IsValidCredentialAgency reports whether the agency may hold credentials.
func IsValidCredentialAgency(a InsuranceAgency) bool {
	return IsValidReportAgency(a) || a == InsuranceAgencyComwel
}

// InsuranceReportType is the kind of 신고 being filed.
//
// The column has no CHECK constraint, so this list is enforced in Go instead:
// an unrecognised report type reaches the 공단 as an unfileable document.
type InsuranceReportType string

const (
	InsuranceReportAcquisition InsuranceReportType = "acquisition" // 자격취득신고
	InsuranceReportLoss        InsuranceReportType = "loss"        // 자격상실신고
	InsuranceReportChange      InsuranceReportType = "change"      // 내용변경신고
	InsuranceReportMonthly     InsuranceReportType = "monthly"     // 월별보험료 신고
	InsuranceReportAnnual      InsuranceReportType = "annual"      // 보수총액신고
)

// IsValidReportType reports whether a report type is one this system files.
func IsValidReportType(t InsuranceReportType) bool {
	switch t {
	case InsuranceReportAcquisition, InsuranceReportLoss, InsuranceReportChange,
		InsuranceReportMonthly, InsuranceReportAnnual:
		return true
	default:
		return false
	}
}

// InsuranceReportStatus is the lifecycle of a 신고서.
type InsuranceReportStatus string

const (
	InsuranceReportStatusDraft     InsuranceReportStatus = "draft"
	InsuranceReportStatusPending   InsuranceReportStatus = "pending"
	InsuranceReportStatusSubmitted InsuranceReportStatus = "submitted"
	InsuranceReportStatusAccepted  InsuranceReportStatus = "accepted"
	InsuranceReportStatusRejected  InsuranceReportStatus = "rejected"
	InsuranceReportStatusCancelled InsuranceReportStatus = "cancelled"
)

// EDIJobStatus is the lifecycle of an asynchronous EDI job.
type EDIJobStatus string

const (
	EDIJobStatusPending    EDIJobStatus = "pending"
	EDIJobStatusQueued     EDIJobStatus = "queued"
	EDIJobStatusProcessing EDIJobStatus = "processing"
	EDIJobStatusCompleted  EDIJobStatus = "completed"
	EDIJobStatusFailed     EDIJobStatus = "failed"
	EDIJobStatusCancelled  EDIJobStatus = "cancelled"
)

// ---------------------------------------------------------------------------
// JSONB mapping
// ---------------------------------------------------------------------------

// InsuranceJSON maps a JSONB document column (report_data, item_data,
// parameters, result). It is a free-form object because each 신고 type carries
// a different set of fields.
type InsuranceJSON map[string]interface{}

// Value implements driver.Valuer.
func (d InsuranceJSON) Value() (driver.Value, error) {
	if d == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(d)
}

// Scan implements sql.Scanner.
func (d *InsuranceJSON) Scan(value interface{}) error {
	if value == nil {
		*d = InsuranceJSON{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("insurance json: unsupported source type %T", value)
	}
	if len(data) == 0 {
		*d = InsuranceJSON{}
		return nil
	}
	return json.Unmarshal(data, d)
}

// GormDataType tells GORM the column type.
func (InsuranceJSON) GormDataType() string { return "jsonb" }

// ---------------------------------------------------------------------------
// Persistence models
// ---------------------------------------------------------------------------

// InsuranceWorkplace is a company's registration with the four agencies.
//
// insurance_workplaces.representative_resident_enc is deliberately NOT mapped:
// it is the representative's encrypted 주민등록번호, and a field that does not
// exist cannot be selected, serialised or logged.
type InsuranceWorkplace struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()" json:"id"`
	CompanyID uuid.UUID `gorm:"type:uuid;not null;index" json:"company_id"`

	BusinessNumber string `gorm:"type:varchar(12);not null" json:"business_number"`
	WorkplaceName  string `gorm:"type:varchar(100);not null" json:"workplace_name"`

	NPSWorkplaceNumber  string `gorm:"column:nps_workplace_number;type:varchar(20)" json:"nps_workplace_number,omitempty"`
	NHISWorkplaceNumber string `gorm:"column:nhis_workplace_number;type:varchar(20)" json:"nhis_workplace_number,omitempty"`
	EIWorkplaceNumber   string `gorm:"column:ei_workplace_number;type:varchar(20)" json:"ei_workplace_number,omitempty"`
	WCIWorkplaceNumber  string `gorm:"column:wci_workplace_number;type:varchar(20)" json:"wci_workplace_number,omitempty"`

	NPSRegistered  bool `gorm:"column:nps_registered;default:false" json:"nps_registered"`
	NHISRegistered bool `gorm:"column:nhis_registered;default:false" json:"nhis_registered"`
	EIRegistered   bool `gorm:"column:ei_registered;default:false" json:"ei_registered"`
	WCIRegistered  bool `gorm:"column:wci_registered;default:false" json:"wci_registered"`

	Representative string `gorm:"type:varchar(50)" json:"representative,omitempty"`

	Phone string `gorm:"type:varchar(20)" json:"phone,omitempty"`
	Fax   string `gorm:"type:varchar(20)" json:"fax,omitempty"`
	Email string `gorm:"type:varchar(100)" json:"email,omitempty"`

	ZipCode       string `gorm:"type:varchar(10)" json:"zip_code,omitempty"`
	Address       string `gorm:"type:varchar(200)" json:"address,omitempty"`
	AddressDetail string `gorm:"type:varchar(100)" json:"address_detail,omitempty"`

	BusinessTypeCode string `gorm:"type:varchar(10)" json:"business_type_code,omitempty"`
	BusinessTypeName string `gorm:"type:varchar(100)" json:"business_type_name,omitempty"`

	IsActive bool `gorm:"default:true" json:"is_active"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName specifies the table name for GORM.
func (InsuranceWorkplace) TableName() string { return "insurance_workplaces" }

// EmployeeInsurance is one employee's 4대보험 qualification record.
type EmployeeInsurance struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()" json:"id"`
	EmployeeID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"employee_id"`
	CompanyID  uuid.UUID `gorm:"type:uuid;not null;index" json:"company_id"`

	NPSQualified  bool `gorm:"column:nps_qualified;default:true" json:"nps_qualified"`
	NHISQualified bool `gorm:"column:nhis_qualified;default:true" json:"nhis_qualified"`
	EIQualified   bool `gorm:"column:ei_qualified;default:true" json:"ei_qualified"`
	WCIQualified  bool `gorm:"column:wci_qualified;default:true" json:"wci_qualified"`

	NPSAcquisitionDate  *time.Time `gorm:"column:nps_acquisition_date;type:date" json:"nps_acquisition_date,omitempty"`
	NHISAcquisitionDate *time.Time `gorm:"column:nhis_acquisition_date;type:date" json:"nhis_acquisition_date,omitempty"`
	EIAcquisitionDate   *time.Time `gorm:"column:ei_acquisition_date;type:date" json:"ei_acquisition_date,omitempty"`

	NPSLossDate  *time.Time `gorm:"column:nps_loss_date;type:date" json:"nps_loss_date,omitempty"`
	NHISLossDate *time.Time `gorm:"column:nhis_loss_date;type:date" json:"nhis_loss_date,omitempty"`
	EILossDate   *time.Time `gorm:"column:ei_loss_date;type:date" json:"ei_loss_date,omitempty"`

	// 기준소득월액 / 보수월액 as reported to each agency. They can differ from
	// the payroll month's actual wage: the reported figure is fixed at
	// acquisition and revised once a year (정기결정).
	NPSStandardRemuneration  *int64 `gorm:"column:nps_standard_remuneration;type:decimal(18,0)" json:"nps_standard_remuneration,omitempty"`
	NHISStandardRemuneration *int64 `gorm:"column:nhis_standard_remuneration;type:decimal(18,0)" json:"nhis_standard_remuneration,omitempty"`

	NHISGrade string `gorm:"column:nhis_grade;type:varchar(10)" json:"nhis_grade,omitempty"`

	DependentsCount int `gorm:"default:0" json:"dependents_count"`

	NPSReduced        bool   `gorm:"column:nps_reduced;default:false" json:"nps_reduced"`
	NHISReduced       bool   `gorm:"column:nhis_reduced;default:false" json:"nhis_reduced"`
	NPSReductionType  string `gorm:"column:nps_reduction_type;type:varchar(20)" json:"nps_reduction_type,omitempty"`
	NHISReductionType string `gorm:"column:nhis_reduction_type;type:varchar(20)" json:"nhis_reduction_type,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName specifies the table name for GORM.
func (EmployeeInsurance) TableName() string { return "employee_insurance" }

// Qualification converts the stored flags into the shape CalculatePayroll wants.
//
// A nil record means "no qualification record on file". That is treated as
// fully qualified, which is what a regular full-time hire is, but the service
// layer says so in the response so the operator can see the assumption.
func (e *EmployeeInsurance) Qualification() SocialInsuranceQualification {
	if e == nil {
		return AllQualified()
	}
	return SocialInsuranceQualification{
		NationalPension:     e.NPSQualified,
		HealthInsurance:     e.NHISQualified,
		EmploymentInsurance: e.EIQualified,
		IndustrialAccident:  e.WCIQualified,
	}
}

// InsuranceReport is a 신고서 filed with one agency.
type InsuranceReport struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()" json:"id"`
	CompanyID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"company_id"`
	WorkplaceID *uuid.UUID `gorm:"type:uuid" json:"workplace_id,omitempty"`

	ReportType InsuranceReportType `gorm:"type:varchar(30);not null" json:"report_type"`
	AgencyType InsuranceAgency     `gorm:"type:varchar(10);not null" json:"agency_type"`

	EmployeeID *uuid.UUID `gorm:"type:uuid" json:"employee_id,omitempty"`
	IsBatch    bool       `gorm:"default:false" json:"is_batch"`

	ReportYear    *int       `json:"report_year,omitempty"`
	ReportMonth   *int       `json:"report_month,omitempty"`
	EffectiveDate *time.Time `gorm:"type:date" json:"effective_date,omitempty"`

	Status InsuranceReportStatus `gorm:"type:varchar(20);not null;default:draft" json:"status"`

	ReportData InsuranceJSON `gorm:"type:jsonb;not null" json:"report_data"`

	ReceiptNumber    string     `gorm:"type:varchar(50)" json:"receipt_number,omitempty"`
	SubmissionMethod string     `gorm:"type:varchar(20)" json:"submission_method,omitempty"`
	SubmittedAt      *time.Time `json:"submitted_at,omitempty"`
	SubmittedBy      *uuid.UUID `gorm:"type:uuid" json:"submitted_by,omitempty"`

	ResultCode      string     `gorm:"type:varchar(20)" json:"result_code,omitempty"`
	ResultMessage   string     `json:"result_message,omitempty"`
	AcceptedAt      *time.Time `json:"accepted_at,omitempty"`
	RejectionReason string     `json:"rejection_reason,omitempty"`

	// EDI payload columns are mapped but never written by this codebase yet.
	// They are excluded from the response DTO: edi_message carries the
	// employees' 주민등록번호 in cleartext once a generator exists.
	EDIStandard string `gorm:"column:edi_standard;type:varchar(20)" json:"-"`
	EDIMessage  string `gorm:"column:edi_message" json:"-"`
	EDIResponse string `gorm:"column:edi_response" json:"-"`
	EDIFilePath string `gorm:"column:edi_file_path;type:varchar(500)" json:"-"`

	CreatedAt time.Time  `gorm:"not null;default:now()" json:"created_at"`
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedAt time.Time  `gorm:"not null;default:now()" json:"updated_at"`

	// Items is filled by the repository on a detail read, not by a GORM
	// association.
	Items []InsuranceReportItem `gorm:"-" json:"items,omitempty"`
}

// TableName specifies the table name for GORM.
func (InsuranceReport) TableName() string { return "insurance_reports" }

// InsuranceReportItem is one employee line inside a batch report.
type InsuranceReportItem struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()" json:"id"`
	ReportID   uuid.UUID `gorm:"type:uuid;not null;index" json:"report_id"`
	EmployeeID uuid.UUID `gorm:"type:uuid;not null" json:"employee_id"`

	LineNo int `gorm:"not null" json:"line_no"`

	EmployeeName string `gorm:"type:varchar(50);not null" json:"employee_name"`
	// ResidentNumberMasked is stored already masked (e.g. 900101-1******).
	// Nothing in this codebase writes an unmasked value into it.
	ResidentNumberMasked string `gorm:"type:varchar(14)" json:"resident_number_masked,omitempty"`

	BaseAmount     int64 `gorm:"type:decimal(18,0)" json:"base_amount"`
	EmployeeAmount int64 `gorm:"type:decimal(18,0)" json:"employee_amount"`
	EmployerAmount int64 `gorm:"type:decimal(18,0)" json:"employer_amount"`
	TotalAmount    int64 `gorm:"type:decimal(18,0)" json:"total_amount"`

	ItemData InsuranceJSON `gorm:"type:jsonb" json:"item_data,omitempty"`

	Status       string `gorm:"type:varchar(20);default:pending" json:"status"`
	ErrorMessage string `gorm:"type:varchar(500)" json:"error_message,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName specifies the table name for GORM.
func (InsuranceReportItem) TableName() string { return "insurance_report_items" }

// InsuranceMonthlyContribution is one employee's assessed 보험료 for one month.
//
// It is written as a by-product of the payroll calculation, inside the same
// transaction, so the 4대보험 register and the payroll register cannot
// disagree about what was withheld.
type InsuranceMonthlyContribution struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()" json:"id"`
	CompanyID  uuid.UUID `gorm:"type:uuid;not null;index" json:"company_id"`
	EmployeeID uuid.UUID `gorm:"type:uuid;not null;index" json:"employee_id"`

	ContributionYear  int `gorm:"not null" json:"contribution_year"`
	ContributionMonth int `gorm:"not null" json:"contribution_month"`

	NPSBase     int64 `gorm:"column:nps_base;type:decimal(18,0);default:0" json:"nps_base"`
	NPSEmployee int64 `gorm:"column:nps_employee;type:decimal(18,0);default:0" json:"nps_employee"`
	NPSEmployer int64 `gorm:"column:nps_employer;type:decimal(18,0);default:0" json:"nps_employer"`

	NHISBase     int64 `gorm:"column:nhis_base;type:decimal(18,0);default:0" json:"nhis_base"`
	NHISEmployee int64 `gorm:"column:nhis_employee;type:decimal(18,0);default:0" json:"nhis_employee"`
	NHISEmployer int64 `gorm:"column:nhis_employer;type:decimal(18,0);default:0" json:"nhis_employer"`

	NHISLTCEmployee int64 `gorm:"column:nhis_ltc_employee;type:decimal(18,0);default:0" json:"nhis_ltc_employee"`
	NHISLTCEmployer int64 `gorm:"column:nhis_ltc_employer;type:decimal(18,0);default:0" json:"nhis_ltc_employer"`

	EIBase     int64 `gorm:"column:ei_base;type:decimal(18,0);default:0" json:"ei_base"`
	EIEmployee int64 `gorm:"column:ei_employee;type:decimal(18,0);default:0" json:"ei_employee"`
	EIEmployer int64 `gorm:"column:ei_employer;type:decimal(18,0);default:0" json:"ei_employer"`

	WCIBase     int64 `gorm:"column:wci_base;type:decimal(18,0);default:0" json:"wci_base"`
	WCIEmployer int64 `gorm:"column:wci_employer;type:decimal(18,0);default:0" json:"wci_employer"`

	// TotalEmployee and TotalEmployer are GENERATED ALWAYS columns. They are
	// tagged read-only ("->"): letting GORM include them in an INSERT makes
	// PostgreSQL reject the statement outright.
	TotalEmployee int64 `gorm:"->;type:decimal(18,0)" json:"total_employee"`
	TotalEmployer int64 `gorm:"->;type:decimal(18,0)" json:"total_employer"`

	PayrollID *uuid.UUID `gorm:"type:uuid" json:"payroll_id,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName specifies the table name for GORM.
func (InsuranceMonthlyContribution) TableName() string {
	return "insurance_monthly_contributions"
}

// InsuranceCredentialStatus is the SAFE projection of insurance_credentials.
//
// The table holds a company's login to the 공단 portals: cert_data_enc,
// cert_password_enc, login_id_enc, login_password_enc, api_key_enc,
// api_secret_enc. NONE of them are fields on this struct, so GORM never selects
// them, they never reach a response body and they never reach a log line. If a
// future EDI worker needs the secrets it must read them through its own type,
// in its own package, and must not reuse this one.
type InsuranceCredentialStatus struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	CompanyID   uuid.UUID  `gorm:"type:uuid;not null" json:"company_id"`
	WorkplaceID *uuid.UUID `gorm:"type:uuid" json:"workplace_id,omitempty"`

	AgencyType     InsuranceAgency `gorm:"type:varchar(10);not null" json:"agency_type"`
	CredentialType string          `gorm:"type:varchar(20);not null" json:"credential_type"`

	CertExpiresAt *time.Time `json:"cert_expires_at,omitempty"`

	IsActive       bool       `gorm:"default:true" json:"is_active"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	LastVerifiedAt *time.Time `json:"last_verified_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name for GORM.
func (InsuranceCredentialStatus) TableName() string { return "insurance_credentials" }

// EDIJob is an asynchronous job against a 공단.
type EDIJob struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()" json:"id"`
	CompanyID uuid.UUID  `gorm:"type:uuid;not null;index" json:"company_id"`
	ReportID  *uuid.UUID `gorm:"type:uuid" json:"report_id,omitempty"`

	JobType    string          `gorm:"type:varchar(30);not null" json:"job_type"`
	AgencyType InsuranceAgency `gorm:"type:varchar(10);not null" json:"agency_type"`
	Status     EDIJobStatus    `gorm:"type:varchar(20);not null;default:pending" json:"status"`

	Parameters InsuranceJSON `gorm:"type:jsonb;not null" json:"parameters"`

	TotalCount     int `gorm:"default:0" json:"total_count"`
	ProcessedCount int `gorm:"default:0" json:"processed_count"`
	SuccessCount   int `gorm:"default:0" json:"success_count"`
	FailCount      int `gorm:"default:0" json:"fail_count"`

	Result       InsuranceJSON `gorm:"type:jsonb" json:"result,omitempty"`
	ErrorCode    string        `gorm:"type:varchar(50)" json:"error_code,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`

	RetryCount int `gorm:"default:0" json:"retry_count"`
	MaxRetries int `gorm:"default:3" json:"max_retries"`

	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	WorkerID string `gorm:"type:varchar(100)" json:"worker_id,omitempty"`

	CreatedAt time.Time  `gorm:"not null;default:now()" json:"created_at"`
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
}

// TableName specifies the table name for GORM.
func (EDIJob) TableName() string { return "edi_jobs" }

// ---------------------------------------------------------------------------
// State transitions
// ---------------------------------------------------------------------------

// CanBeModified reports whether the report may still be edited or deleted.
func (r *InsuranceReport) CanBeModified() bool {
	return r.Status == InsuranceReportStatusDraft || r.Status == InsuranceReportStatusPending
}

// ValidateForSubmission is the guard in front of every path that would send the
// report to a 공단.
//
// The receipt number check is the idempotency guard and it is checked FIRST,
// before the status: a report that carries a 접수번호 has been filed, whatever
// its status column says. The tax invoice code learned this the hard way -
// recording an empty confirmation number let a retry file the same document
// twice.
func (r *InsuranceReport) ValidateForSubmission() error {
	if r.ReceiptNumber != "" {
		return fmt.Errorf("%w: 접수번호 %s", ErrInsuranceReportSubmitted, r.ReceiptNumber)
	}
	switch r.Status {
	case InsuranceReportStatusSubmitted, InsuranceReportStatusAccepted:
		return ErrInsuranceReportSubmitted
	case InsuranceReportStatusDraft, InsuranceReportStatusPending, InsuranceReportStatusRejected:
		// A rejected report may be corrected and re-filed.
	default:
		return fmt.Errorf("%w: status=%s", ErrInsuranceReportNotEditable, r.Status)
	}
	if !IsValidReportAgency(r.AgencyType) {
		return fmt.Errorf("%w: %s", ErrInsuranceAgencyInvalid, r.AgencyType)
	}
	if !IsValidReportType(r.ReportType) {
		return fmt.Errorf("%w: %s", ErrInsuranceReportTypeInvalid, r.ReportType)
	}
	return nil
}

// CanBeCancelled reports whether the report may be withdrawn. Once a 공단 has
// accepted a filing, cancelling it is a new filing, not a status change.
func (r *InsuranceReport) CanBeCancelled() bool {
	switch r.Status {
	case InsuranceReportStatusDraft, InsuranceReportStatusPending, InsuranceReportStatusRejected:
		return true
	default:
		return false
	}
}

// ApplyPayrollContribution copies a payroll calculation into the monthly
// contribution register. Both come from one CalculatePayroll call, so the two
// registers report the same figures by construction.
func (c *InsuranceMonthlyContribution) ApplyPayrollContribution(r PayrollCalculationResult) {
	c.NPSBase = r.SocialInsurance.PensionBase
	c.NPSEmployee = r.SocialInsurance.NationalPension.Employee
	c.NPSEmployer = r.SocialInsurance.NationalPension.Employer

	c.NHISBase = r.TaxableWage
	c.NHISEmployee = r.SocialInsurance.HealthInsurance.Employee
	c.NHISEmployer = r.SocialInsurance.HealthInsurance.Employer

	c.NHISLTCEmployee = r.SocialInsurance.LongTermCare.Employee
	c.NHISLTCEmployer = r.SocialInsurance.LongTermCare.Employer

	c.EIBase = r.TaxableWage
	c.EIEmployee = r.SocialInsurance.EmploymentInsurance.Employee
	// The employer's 고용보험료 includes 고용안정·직업능력개발사업, which is why
	// this reads the employer cost rather than the insurance split.
	c.EIEmployer = r.EmployerCost.EmploymentInsurance

	if r.IndustrialAccidentApplied {
		c.WCIBase = r.TaxableWage
		c.WCIEmployer = r.EmployerCost.IndustrialAccident
	} else {
		c.WCIBase = 0
		c.WCIEmployer = 0
	}
}

// ============================================================================
// HANDOFF / 확인 필요
// ============================================================================
//
//  1. EDI 전송 경로가 없다.
//     python-services/insurance-edi (gRPC :50052) 는 존재하지만 Go 클라이언트가
//     없다 - internal/grpcclient 에는 tax_client.go 뿐이고 api/proto 에도
//     insurance 용 .proto 가 없다. 그래서 신고 전송은 상태를 건드리지 않고
//     ErrInsuranceEDIUnavailable 로 실패한다. edi_jobs 행조차 만들지 않는다:
//     아무도 소비하지 않는 작업 행은 "접수됨" 으로 보이는 가짜 성공이다.
//
//     필요한 것: api/proto/insurance/v1/insurance.proto, 그로부터 생성한 Go
//     스텁, internal/grpcclient/insurance_client.go. 전송 성공 판정은 반드시
//     "접수번호가 비어 있지 않을 것" 을 포함해야 한다 (세금계산서에서 빈
//     승인번호가 멱등성 가드를 무력화한 전례가 있다).
//
//  2. insurance_credentials 의 복호화.
//     ARIA 로 암호화된 인증서·비밀번호를 다루는 코드는 이 패키지에 없고,
//     InsuranceCredentialStatus 는 그 컬럼들을 아예 매핑하지 않는다. 전송
//     워커가 생길 때 별도 타입으로 읽되 응답 DTO 에는 절대 넣지 말 것.
//
//  3. resident_number_masked.
//     이 코드베이스에는 주민등록번호를 마스킹하는 공용 함수가 없다. 지금은
//     호출자가 마스킹된 문자열을 넣는 것을 전제로 한다. 신고서 생성기를 만들 때
//     마스킹을 도메인 함수로 올려 강제할 것.
