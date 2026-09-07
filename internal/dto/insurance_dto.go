package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// ---------------------------------------------------------------------------
// Workplaces
// ---------------------------------------------------------------------------

// InsuranceWorkplaceRequest registers or updates a 4대보험 사업장.
type InsuranceWorkplaceRequest struct {
	BusinessNumber string `json:"business_number" binding:"required,min=10,max=12"`
	WorkplaceName  string `json:"workplace_name" binding:"required,max=100"`

	NPSWorkplaceNumber  string `json:"nps_workplace_number" binding:"max=20"`
	NHISWorkplaceNumber string `json:"nhis_workplace_number" binding:"max=20"`
	EIWorkplaceNumber   string `json:"ei_workplace_number" binding:"max=20"`
	WCIWorkplaceNumber  string `json:"wci_workplace_number" binding:"max=20"`

	NPSRegistered  bool `json:"nps_registered"`
	NHISRegistered bool `json:"nhis_registered"`
	EIRegistered   bool `json:"ei_registered"`
	WCIRegistered  bool `json:"wci_registered"`

	Representative string `json:"representative" binding:"max=50"`
	Phone          string `json:"phone" binding:"max=20"`
	Fax            string `json:"fax" binding:"max=20"`
	Email          string `json:"email" binding:"omitempty,email,max=100"`

	ZipCode       string `json:"zip_code" binding:"max=10"`
	Address       string `json:"address" binding:"max=200"`
	AddressDetail string `json:"address_detail" binding:"max=100"`

	BusinessTypeCode string `json:"business_type_code" binding:"max=10"`
	BusinessTypeName string `json:"business_type_name" binding:"max=100"`

	IsActive bool `json:"is_active"`
}

// InsuranceWorkplaceResponse is one workplace registration.
//
// There is no representative resident number here, and there cannot be:
// domain.InsuranceWorkplace does not map representative_resident_enc.
type InsuranceWorkplaceResponse struct {
	ID             uuid.UUID `json:"id"`
	BusinessNumber string    `json:"business_number"`
	WorkplaceName  string    `json:"workplace_name"`

	NPSWorkplaceNumber  string `json:"nps_workplace_number,omitempty"`
	NHISWorkplaceNumber string `json:"nhis_workplace_number,omitempty"`
	EIWorkplaceNumber   string `json:"ei_workplace_number,omitempty"`
	WCIWorkplaceNumber  string `json:"wci_workplace_number,omitempty"`

	NPSRegistered  bool `json:"nps_registered"`
	NHISRegistered bool `json:"nhis_registered"`
	EIRegistered   bool `json:"ei_registered"`
	WCIRegistered  bool `json:"wci_registered"`

	Representative string `json:"representative,omitempty"`
	Phone          string `json:"phone,omitempty"`
	Fax            string `json:"fax,omitempty"`
	Email          string `json:"email,omitempty"`

	ZipCode       string `json:"zip_code,omitempty"`
	Address       string `json:"address,omitempty"`
	AddressDetail string `json:"address_detail,omitempty"`

	BusinessTypeCode string `json:"business_type_code,omitempty"`
	BusinessTypeName string `json:"business_type_name,omitempty"`

	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FromInsuranceWorkplace converts a workplace registration.
func FromInsuranceWorkplace(w *domain.InsuranceWorkplace) InsuranceWorkplaceResponse {
	return InsuranceWorkplaceResponse{
		ID:                  w.ID,
		BusinessNumber:      w.BusinessNumber,
		WorkplaceName:       w.WorkplaceName,
		NPSWorkplaceNumber:  w.NPSWorkplaceNumber,
		NHISWorkplaceNumber: w.NHISWorkplaceNumber,
		EIWorkplaceNumber:   w.EIWorkplaceNumber,
		WCIWorkplaceNumber:  w.WCIWorkplaceNumber,
		NPSRegistered:       w.NPSRegistered,
		NHISRegistered:      w.NHISRegistered,
		EIRegistered:        w.EIRegistered,
		WCIRegistered:       w.WCIRegistered,
		Representative:      w.Representative,
		Phone:               w.Phone,
		Fax:                 w.Fax,
		Email:               w.Email,
		ZipCode:             w.ZipCode,
		Address:             w.Address,
		AddressDetail:       w.AddressDetail,
		BusinessTypeCode:    w.BusinessTypeCode,
		BusinessTypeName:    w.BusinessTypeName,
		IsActive:            w.IsActive,
		CreatedAt:           w.CreatedAt,
		UpdatedAt:           w.UpdatedAt,
	}
}

// FromInsuranceWorkplaces converts a slice of workplace registrations.
func FromInsuranceWorkplaces(workplaces []*domain.InsuranceWorkplace) []InsuranceWorkplaceResponse {
	out := make([]InsuranceWorkplaceResponse, 0, len(workplaces))
	for _, w := range workplaces {
		out = append(out, FromInsuranceWorkplace(w))
	}
	return out
}

// ---------------------------------------------------------------------------
// Employee qualifications
// ---------------------------------------------------------------------------

// EmployeeInsuranceRequest sets one employee's 4대보험 자격.
type EmployeeInsuranceRequest struct {
	NPSQualified  bool `json:"nps_qualified"`
	NHISQualified bool `json:"nhis_qualified"`
	EIQualified   bool `json:"ei_qualified"`
	WCIQualified  bool `json:"wci_qualified"`

	NPSAcquisitionDate  string `json:"nps_acquisition_date"`
	NHISAcquisitionDate string `json:"nhis_acquisition_date"`
	EIAcquisitionDate   string `json:"ei_acquisition_date"`

	NPSLossDate  string `json:"nps_loss_date"`
	NHISLossDate string `json:"nhis_loss_date"`
	EILossDate   string `json:"ei_loss_date"`

	NPSStandardRemuneration  *int64 `json:"nps_standard_remuneration" binding:"omitempty,min=0"`
	NHISStandardRemuneration *int64 `json:"nhis_standard_remuneration" binding:"omitempty,min=0"`

	NHISGrade       string `json:"nhis_grade" binding:"max=10"`
	DependentsCount int    `json:"dependents_count" binding:"min=0"`

	NPSReduced        bool   `json:"nps_reduced"`
	NHISReduced       bool   `json:"nhis_reduced"`
	NPSReductionType  string `json:"nps_reduction_type" binding:"max=20"`
	NHISReductionType string `json:"nhis_reduction_type" binding:"max=20"`
}

// InsuranceDates is the set of parsed acquisition and loss dates.
type InsuranceDates struct {
	NPSAcquisition  *time.Time
	NHISAcquisition *time.Time
	EIAcquisition   *time.Time
	NPSLoss         *time.Time
	NHISLoss        *time.Time
	EILoss          *time.Time
}

// ParseDates parses every date field, failing on the first malformed one.
func (r *EmployeeInsuranceRequest) ParseDates() (*InsuranceDates, error) {
	var d InsuranceDates
	var err error

	if d.NPSAcquisition, err = parsePayrollDate(r.NPSAcquisitionDate); err != nil {
		return nil, err
	}
	if d.NHISAcquisition, err = parsePayrollDate(r.NHISAcquisitionDate); err != nil {
		return nil, err
	}
	if d.EIAcquisition, err = parsePayrollDate(r.EIAcquisitionDate); err != nil {
		return nil, err
	}
	if d.NPSLoss, err = parsePayrollDate(r.NPSLossDate); err != nil {
		return nil, err
	}
	if d.NHISLoss, err = parsePayrollDate(r.NHISLossDate); err != nil {
		return nil, err
	}
	if d.EILoss, err = parsePayrollDate(r.EILossDate); err != nil {
		return nil, err
	}
	return &d, nil
}

// EmployeeInsuranceResponse is one employee's qualification record.
type EmployeeInsuranceResponse struct {
	ID         uuid.UUID `json:"id"`
	EmployeeID uuid.UUID `json:"employee_id"`

	EmployeeNo     string `json:"employee_no,omitempty"`
	EmployeeName   string `json:"employee_name,omitempty"`
	DepartmentName string `json:"department_name,omitempty"`

	NPSQualified  bool `json:"nps_qualified"`
	NHISQualified bool `json:"nhis_qualified"`
	EIQualified   bool `json:"ei_qualified"`
	WCIQualified  bool `json:"wci_qualified"`

	NPSAcquisitionDate  string `json:"nps_acquisition_date,omitempty"`
	NHISAcquisitionDate string `json:"nhis_acquisition_date,omitempty"`
	EIAcquisitionDate   string `json:"ei_acquisition_date,omitempty"`

	NPSLossDate  string `json:"nps_loss_date,omitempty"`
	NHISLossDate string `json:"nhis_loss_date,omitempty"`
	EILossDate   string `json:"ei_loss_date,omitempty"`

	NPSStandardRemuneration  *int64 `json:"nps_standard_remuneration,omitempty"`
	NHISStandardRemuneration *int64 `json:"nhis_standard_remuneration,omitempty"`

	NHISGrade       string `json:"nhis_grade,omitempty"`
	DependentsCount int    `json:"dependents_count"`

	NPSReduced        bool   `json:"nps_reduced"`
	NHISReduced       bool   `json:"nhis_reduced"`
	NPSReductionType  string `json:"nps_reduction_type,omitempty"`
	NHISReductionType string `json:"nhis_reduction_type,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FromEmployeeInsurance converts a qualification record.
func FromEmployeeInsurance(e *domain.EmployeeInsurance) EmployeeInsuranceResponse {
	return EmployeeInsuranceResponse{
		ID:                       e.ID,
		EmployeeID:               e.EmployeeID,
		NPSQualified:             e.NPSQualified,
		NHISQualified:            e.NHISQualified,
		EIQualified:              e.EIQualified,
		WCIQualified:             e.WCIQualified,
		NPSAcquisitionDate:       payrollDatePtrString(e.NPSAcquisitionDate),
		NHISAcquisitionDate:      payrollDatePtrString(e.NHISAcquisitionDate),
		EIAcquisitionDate:        payrollDatePtrString(e.EIAcquisitionDate),
		NPSLossDate:              payrollDatePtrString(e.NPSLossDate),
		NHISLossDate:             payrollDatePtrString(e.NHISLossDate),
		EILossDate:               payrollDatePtrString(e.EILossDate),
		NPSStandardRemuneration:  e.NPSStandardRemuneration,
		NHISStandardRemuneration: e.NHISStandardRemuneration,
		NHISGrade:                e.NHISGrade,
		DependentsCount:          e.DependentsCount,
		NPSReduced:               e.NPSReduced,
		NHISReduced:              e.NHISReduced,
		NPSReductionType:         e.NPSReductionType,
		NHISReductionType:        e.NHISReductionType,
		CreatedAt:                e.CreatedAt,
		UpdatedAt:                e.UpdatedAt,
	}
}

// FromEmployeeInsuranceRows converts joined qualification rows.
func FromEmployeeInsuranceRows(rows []*repository.EmployeeInsuranceRow) []EmployeeInsuranceResponse {
	out := make([]EmployeeInsuranceResponse, 0, len(rows))
	for _, r := range rows {
		resp := FromEmployeeInsurance(&r.EmployeeInsurance)
		resp.EmployeeNo = r.EmployeeNo
		resp.EmployeeName = r.EmployeeName
		resp.DepartmentName = r.DepartmentName
		out = append(out, resp)
	}
	return out
}

// ---------------------------------------------------------------------------
// Contributions
// ---------------------------------------------------------------------------

// InsuranceContributionResponse is one employee's assessed 보험료 for a month.
type InsuranceContributionResponse struct {
	ID         uuid.UUID `json:"id"`
	EmployeeID uuid.UUID `json:"employee_id"`

	EmployeeNo     string `json:"employee_no,omitempty"`
	EmployeeName   string `json:"employee_name,omitempty"`
	DepartmentName string `json:"department_name,omitempty"`

	ContributionYear  int `json:"contribution_year"`
	ContributionMonth int `json:"contribution_month"`

	NPSBase     int64 `json:"nps_base"`
	NPSEmployee int64 `json:"nps_employee"`
	NPSEmployer int64 `json:"nps_employer"`

	NHISBase     int64 `json:"nhis_base"`
	NHISEmployee int64 `json:"nhis_employee"`
	NHISEmployer int64 `json:"nhis_employer"`

	NHISLTCEmployee int64 `json:"nhis_ltc_employee"`
	NHISLTCEmployer int64 `json:"nhis_ltc_employer"`

	EIBase     int64 `json:"ei_base"`
	EIEmployee int64 `json:"ei_employee"`
	EIEmployer int64 `json:"ei_employer"`

	WCIBase     int64 `json:"wci_base"`
	WCIEmployer int64 `json:"wci_employer"`

	TotalEmployee int64 `json:"total_employee"`
	TotalEmployer int64 `json:"total_employer"`

	PayrollID *uuid.UUID `json:"payroll_id,omitempty"`

	UpdatedAt time.Time `json:"updated_at"`
}

// FromInsuranceContributionRows converts joined contribution rows.
func FromInsuranceContributionRows(rows []*repository.InsuranceContributionRow) []InsuranceContributionResponse {
	out := make([]InsuranceContributionResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, InsuranceContributionResponse{
			ID:                r.ID,
			EmployeeID:        r.EmployeeID,
			EmployeeNo:        r.EmployeeNo,
			EmployeeName:      r.EmployeeName,
			DepartmentName:    r.DepartmentName,
			ContributionYear:  r.ContributionYear,
			ContributionMonth: r.ContributionMonth,
			NPSBase:           r.NPSBase,
			NPSEmployee:       r.NPSEmployee,
			NPSEmployer:       r.NPSEmployer,
			NHISBase:          r.NHISBase,
			NHISEmployee:      r.NHISEmployee,
			NHISEmployer:      r.NHISEmployer,
			NHISLTCEmployee:   r.NHISLTCEmployee,
			NHISLTCEmployer:   r.NHISLTCEmployer,
			EIBase:            r.EIBase,
			EIEmployee:        r.EIEmployee,
			EIEmployer:        r.EIEmployer,
			WCIBase:           r.WCIBase,
			WCIEmployer:       r.WCIEmployer,
			TotalEmployee:     r.TotalEmployee,
			TotalEmployer:     r.TotalEmployer,
			PayrollID:         r.PayrollID,
			UpdatedAt:         r.UpdatedAt,
		})
	}
	return out
}

// InsuranceSummaryResponse totals one month across the company.
type InsuranceSummaryResponse struct {
	Year          int   `json:"year"`
	Month         int   `json:"month"`
	EmployeeCount int   `json:"employee_count"`
	NPSEmployee   int64 `json:"nps_employee"`
	NPSEmployer   int64 `json:"nps_employer"`
	NHISEmployee  int64 `json:"nhis_employee"`
	NHISEmployer  int64 `json:"nhis_employer"`
	LTCEmployee   int64 `json:"ltc_employee"`
	LTCEmployer   int64 `json:"ltc_employer"`
	EIEmployee    int64 `json:"ei_employee"`
	EIEmployer    int64 `json:"ei_employer"`
	WCIEmployer   int64 `json:"wci_employer"`
	TotalEmployee int64 `json:"total_employee"`
	TotalEmployer int64 `json:"total_employer"`
}

// FromInsuranceSummary converts a monthly summary.
func FromInsuranceSummary(s *repository.InsuranceContributionSummary, year, month int) InsuranceSummaryResponse {
	return InsuranceSummaryResponse{
		Year:          year,
		Month:         month,
		EmployeeCount: s.EmployeeCount,
		NPSEmployee:   s.NPSEmployee,
		NPSEmployer:   s.NPSEmployer,
		NHISEmployee:  s.NHISEmployee,
		NHISEmployer:  s.NHISEmployer,
		LTCEmployee:   s.LTCEmployee,
		LTCEmployer:   s.LTCEmployer,
		EIEmployee:    s.EIEmployee,
		EIEmployer:    s.EIEmployer,
		WCIEmployer:   s.WCIEmployer,
		TotalEmployee: s.TotalEmployee,
		TotalEmployer: s.TotalEmployer,
	}
}

// ContributionPreviewRequest computes 4대보험 for a monthly wage.
type ContributionPreviewRequest struct {
	MonthlyWage int64 `json:"monthly_wage" binding:"min=0"`

	// ContributionDate selects the rate year and the 국민연금 기준소득월액
	// window. Required, because both change on their own schedules and picking
	// "today" would silently apply this year's rates to last year's payroll.
	ContributionDate string `json:"contribution_date" binding:"required"`

	IndustrialAccidentRate *int64 `json:"industrial_accident_rate" binding:"omitempty,min=0"`
	WorkplaceSize          string `json:"workplace_size" binding:"omitempty,oneof=under_150 priority_150_plus from_150_to_999 over_1000"`
}

// Parsed returns the contribution date and workplace-size band.
func (r *ContributionPreviewRequest) Parsed() (time.Time, domain.WorkplaceSize, error) {
	on, err := parsePayrollDate(r.ContributionDate)
	if err != nil {
		return time.Time{}, "", err
	}
	if on == nil {
		return time.Time{}, "", ErrInvalidPayrollDate
	}
	size := domain.WorkplaceUnder150
	if r.WorkplaceSize != "" {
		size = domain.WorkplaceSize(r.WorkplaceSize)
	}
	return *on, size, nil
}

// ContributionPreviewResponse is a computed 4대보험 assessment.
type ContributionPreviewResponse struct {
	MonthlyWage int64 `json:"monthly_wage"`
	PensionBase int64 `json:"pension_base"`

	NationalPension     SocialInsuranceSplitResponse `json:"national_pension"`
	HealthInsurance     SocialInsuranceSplitResponse `json:"health_insurance"`
	LongTermCare        SocialInsuranceSplitResponse `json:"long_term_care"`
	EmploymentInsurance SocialInsuranceSplitResponse `json:"employment_insurance"`
	IndustrialAccident  SocialInsuranceSplitResponse `json:"industrial_accident"`

	EmployeeTotal int64 `json:"employee_total"`
	EmployerTotal int64 `json:"employer_total"`

	// RatesSource names the authority behind the rate row that was applied.
	RatesSource string `json:"rates_source"`
}

// FromSocialInsuranceResult converts a 4대보험 calculation.
func FromSocialInsuranceResult(r *domain.SocialInsuranceResult) ContributionPreviewResponse {
	return ContributionPreviewResponse{
		MonthlyWage:         r.MonthlyWage,
		PensionBase:         r.PensionBase,
		NationalPension:     splitResponse(r.NationalPension),
		HealthInsurance:     splitResponse(r.HealthInsurance),
		LongTermCare:        splitResponse(r.LongTermCare),
		EmploymentInsurance: splitResponse(r.EmploymentInsurance),
		IndustrialAccident:  splitResponse(r.IndustrialAccident),
		EmployeeTotal:       r.EmployeeTotal(),
		EmployerTotal:       r.EmployerTotal(),
		RatesSource:         r.RatesSource,
	}
}

// ---------------------------------------------------------------------------
// Reports
// ---------------------------------------------------------------------------

// InsuranceReportItemRequest is one employee line of a 신고서.
type InsuranceReportItemRequest struct {
	EmployeeID   string `json:"employee_id" binding:"required,uuid"`
	LineNo       int    `json:"line_no" binding:"min=0"`
	EmployeeName string `json:"employee_name" binding:"required,max=50"`

	// ResidentNumberMasked must ALREADY be masked (900101-1******). Nothing on
	// the server masks it for you; sending a full 주민등록번호 here stores a
	// full 주민등록번호.
	ResidentNumberMasked string `json:"resident_number_masked" binding:"max=14"`

	BaseAmount     int64 `json:"base_amount" binding:"min=0"`
	EmployeeAmount int64 `json:"employee_amount" binding:"min=0"`
	EmployerAmount int64 `json:"employer_amount" binding:"min=0"`
	TotalAmount    int64 `json:"total_amount" binding:"min=0"`

	ItemData map[string]interface{} `json:"item_data"`
}

// CreateInsuranceReportRequest opens a draft 신고서.
type CreateInsuranceReportRequest struct {
	WorkplaceID string `json:"workplace_id" binding:"omitempty,uuid"`
	ReportType  string `json:"report_type" binding:"required,oneof=acquisition loss change monthly annual"`
	AgencyType  string `json:"agency_type" binding:"required,oneof=nps nhis ei wci"`
	EmployeeID  string `json:"employee_id" binding:"omitempty,uuid"`
	IsBatch     bool   `json:"is_batch"`

	ReportYear    *int   `json:"report_year" binding:"omitempty,min=2000,max=2100"`
	ReportMonth   *int   `json:"report_month" binding:"omitempty,min=1,max=12"`
	EffectiveDate string `json:"effective_date"`

	ReportData map[string]interface{} `json:"report_data"`

	Items []InsuranceReportItemRequest `json:"items" binding:"omitempty,dive"`
}

// CancelInsuranceReportRequest withdraws a draft 신고서.
type CancelInsuranceReportRequest struct {
	Reason string `json:"reason" binding:"max=500"`
}

// InsuranceReportItemResponse is one line of a 신고서.
type InsuranceReportItemResponse struct {
	ID                   uuid.UUID              `json:"id"`
	EmployeeID           uuid.UUID              `json:"employee_id"`
	LineNo               int                    `json:"line_no"`
	EmployeeName         string                 `json:"employee_name"`
	ResidentNumberMasked string                 `json:"resident_number_masked,omitempty"`
	BaseAmount           int64                  `json:"base_amount"`
	EmployeeAmount       int64                  `json:"employee_amount"`
	EmployerAmount       int64                  `json:"employer_amount"`
	TotalAmount          int64                  `json:"total_amount"`
	ItemData             map[string]interface{} `json:"item_data,omitempty"`
	Status               string                 `json:"status"`
	ErrorMessage         string                 `json:"error_message,omitempty"`
}

// InsuranceReportResponse is one 신고서.
//
// The EDI payload columns (edi_message, edi_response, edi_file_path) are absent
// on purpose: once a generator exists they carry the covered employees'
// 주민등록번호 in the clear, and a report list is a screen many people can open.
type InsuranceReportResponse struct {
	ID          uuid.UUID  `json:"id"`
	WorkplaceID *uuid.UUID `json:"workplace_id,omitempty"`

	ReportType string `json:"report_type"`
	AgencyType string `json:"agency_type"`

	EmployeeID *uuid.UUID `json:"employee_id,omitempty"`
	IsBatch    bool       `json:"is_batch"`

	ReportYear    *int   `json:"report_year,omitempty"`
	ReportMonth   *int   `json:"report_month,omitempty"`
	EffectiveDate string `json:"effective_date,omitempty"`

	Status string `json:"status"`

	ReportData map[string]interface{} `json:"report_data,omitempty"`

	ReceiptNumber    string     `json:"receipt_number,omitempty"`
	SubmissionMethod string     `json:"submission_method,omitempty"`
	SubmittedAt      *time.Time `json:"submitted_at,omitempty"`

	ResultCode      string     `json:"result_code,omitempty"`
	ResultMessage   string     `json:"result_message,omitempty"`
	AcceptedAt      *time.Time `json:"accepted_at,omitempty"`
	RejectionReason string     `json:"rejection_reason,omitempty"`

	Items []InsuranceReportItemResponse `json:"items,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FromInsuranceReport converts a 신고서.
func FromInsuranceReport(r *domain.InsuranceReport) InsuranceReportResponse {
	resp := InsuranceReportResponse{
		ID:               r.ID,
		WorkplaceID:      r.WorkplaceID,
		ReportType:       string(r.ReportType),
		AgencyType:       string(r.AgencyType),
		EmployeeID:       r.EmployeeID,
		IsBatch:          r.IsBatch,
		ReportYear:       r.ReportYear,
		ReportMonth:      r.ReportMonth,
		EffectiveDate:    payrollDatePtrString(r.EffectiveDate),
		Status:           string(r.Status),
		ReportData:       r.ReportData,
		ReceiptNumber:    r.ReceiptNumber,
		SubmissionMethod: r.SubmissionMethod,
		SubmittedAt:      r.SubmittedAt,
		ResultCode:       r.ResultCode,
		ResultMessage:    r.ResultMessage,
		AcceptedAt:       r.AcceptedAt,
		RejectionReason:  r.RejectionReason,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
	for _, it := range r.Items {
		resp.Items = append(resp.Items, InsuranceReportItemResponse{
			ID:                   it.ID,
			EmployeeID:           it.EmployeeID,
			LineNo:               it.LineNo,
			EmployeeName:         it.EmployeeName,
			ResidentNumberMasked: it.ResidentNumberMasked,
			BaseAmount:           it.BaseAmount,
			EmployeeAmount:       it.EmployeeAmount,
			EmployerAmount:       it.EmployerAmount,
			TotalAmount:          it.TotalAmount,
			ItemData:             it.ItemData,
			Status:               it.Status,
			ErrorMessage:         it.ErrorMessage,
		})
	}
	return resp
}

// FromInsuranceReports converts a slice of 신고서 records.
func FromInsuranceReports(reports []*domain.InsuranceReport) []InsuranceReportResponse {
	out := make([]InsuranceReportResponse, 0, len(reports))
	for _, r := range reports {
		out = append(out, FromInsuranceReport(r))
	}
	return out
}

// ---------------------------------------------------------------------------
// Credentials and EDI jobs
// ---------------------------------------------------------------------------

// InsuranceCredentialStatusResponse says whether an agency credential is on
// file and usable.
//
// IT CARRIES NO CREDENTIAL. The certificate, its password, the portal login and
// the API secret are all ARIA-encrypted columns that the domain model does not
// map, so there is no field here they could be copied into.
type InsuranceCredentialStatusResponse struct {
	ID             uuid.UUID  `json:"id"`
	WorkplaceID    *uuid.UUID `json:"workplace_id,omitempty"`
	AgencyType     string     `json:"agency_type"`
	CredentialType string     `json:"credential_type"`

	CertExpiresAt *time.Time `json:"cert_expires_at,omitempty"`
	// CertExpired is derived so the screen can warn without doing date maths.
	CertExpired bool `json:"cert_expired"`

	IsActive       bool       `json:"is_active"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	LastVerifiedAt *time.Time `json:"last_verified_at,omitempty"`
}

// FromInsuranceCredentialStatuses converts credential status rows.
func FromInsuranceCredentialStatuses(rows []*domain.InsuranceCredentialStatus) []InsuranceCredentialStatusResponse {
	now := time.Now()
	out := make([]InsuranceCredentialStatusResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, InsuranceCredentialStatusResponse{
			ID:             r.ID,
			WorkplaceID:    r.WorkplaceID,
			AgencyType:     string(r.AgencyType),
			CredentialType: r.CredentialType,
			CertExpiresAt:  r.CertExpiresAt,
			CertExpired:    r.CertExpiresAt != nil && r.CertExpiresAt.Before(now),
			IsActive:       r.IsActive,
			LastUsedAt:     r.LastUsedAt,
			LastVerifiedAt: r.LastVerifiedAt,
		})
	}
	return out
}

// EDIJobResponse is one asynchronous EDI job.
type EDIJobResponse struct {
	ID       uuid.UUID  `json:"id"`
	ReportID *uuid.UUID `json:"report_id,omitempty"`

	JobType    string `json:"job_type"`
	AgencyType string `json:"agency_type"`
	Status     string `json:"status"`

	TotalCount     int `json:"total_count"`
	ProcessedCount int `json:"processed_count"`
	SuccessCount   int `json:"success_count"`
	FailCount      int `json:"fail_count"`

	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`

	RetryCount int `json:"retry_count"`
	MaxRetries int `json:"max_retries"`

	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// FromEDIJob converts an EDI job.
//
// parameters and result are not exposed: they are free-form JSONB that an
// eventual transmitter will fill with filing payloads.
func FromEDIJob(j *domain.EDIJob) EDIJobResponse {
	return EDIJobResponse{
		ID:             j.ID,
		ReportID:       j.ReportID,
		JobType:        j.JobType,
		AgencyType:     string(j.AgencyType),
		Status:         string(j.Status),
		TotalCount:     j.TotalCount,
		ProcessedCount: j.ProcessedCount,
		SuccessCount:   j.SuccessCount,
		FailCount:      j.FailCount,
		ErrorCode:      j.ErrorCode,
		ErrorMessage:   j.ErrorMessage,
		RetryCount:     j.RetryCount,
		MaxRetries:     j.MaxRetries,
		StartedAt:      j.StartedAt,
		CompletedAt:    j.CompletedAt,
		CreatedAt:      j.CreatedAt,
	}
}

// FromEDIJobs converts a slice of EDI jobs.
func FromEDIJobs(jobs []*domain.EDIJob) []EDIJobResponse {
	out := make([]EDIJobResponse, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, FromEDIJob(j))
	}
	return out
}
