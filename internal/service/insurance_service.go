package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// InsuranceService is the application layer for 4대보험 administration.
//
// It does NOT calculate contributions. Those are produced by the payroll
// calculation (domain.CalculatePayroll -> social_insurance.go) and written into
// insurance_monthly_contributions inside the payroll transaction, so the
// payroll register and the 4대보험 register can never disagree about what was
// withheld from whom. This service reads that register and manages the
// surrounding paperwork: workplace registrations, qualifications and 신고서.
//
// TRANSMISSION IS FAIL-CLOSED. There is no Go client for the insurance-edi
// service, so SubmitReport returns domain.ErrInsuranceEDIUnavailable and
// changes nothing. Marking a report "submitted" - or queuing an edi_jobs row
// that no worker consumes - would tell an operator a statutory filing was made
// when nothing left the building.
type InsuranceService struct {
	repo repository.InsuranceRepository
}

// NewInsuranceService creates an InsuranceService.
func NewInsuranceService(repo repository.InsuranceRepository) *InsuranceService {
	return &InsuranceService{repo: repo}
}

// ---------------------------------------------------------------------------
// Inputs
// ---------------------------------------------------------------------------

// InsuranceWorkplaceInput describes a workplace registration.
type InsuranceWorkplaceInput struct {
	BusinessNumber string
	WorkplaceName  string

	NPSWorkplaceNumber  string
	NHISWorkplaceNumber string
	EIWorkplaceNumber   string
	WCIWorkplaceNumber  string

	NPSRegistered  bool
	NHISRegistered bool
	EIRegistered   bool
	WCIRegistered  bool

	Representative string
	Phone          string
	Fax            string
	Email          string

	ZipCode       string
	Address       string
	AddressDetail string

	BusinessTypeCode string
	BusinessTypeName string

	IsActive bool
}

// EmployeeInsuranceInput describes one employee's qualification record.
type EmployeeInsuranceInput struct {
	NPSQualified  bool
	NHISQualified bool
	EIQualified   bool
	WCIQualified  bool

	NPSAcquisitionDate  *time.Time
	NHISAcquisitionDate *time.Time
	EIAcquisitionDate   *time.Time

	NPSLossDate  *time.Time
	NHISLossDate *time.Time
	EILossDate   *time.Time

	NPSStandardRemuneration  *int64
	NHISStandardRemuneration *int64

	NHISGrade       string
	DependentsCount int

	NPSReduced        bool
	NHISReduced       bool
	NPSReductionType  string
	NHISReductionType string
}

// InsuranceReportItemInput is one employee line of a 신고서.
type InsuranceReportItemInput struct {
	EmployeeID   uuid.UUID
	LineNo       int
	EmployeeName string
	// ResidentNumberMasked must already be masked by the caller. Nothing in
	// this codebase masks it for you - see the HANDOFF note in
	// internal/domain/employee_insurance.go.
	ResidentNumberMasked string
	BaseAmount           int64
	EmployeeAmount       int64
	EmployerAmount       int64
	TotalAmount          int64
	ItemData             domain.InsuranceJSON
}

// CreateInsuranceReportInput describes a new 신고서.
type CreateInsuranceReportInput struct {
	WorkplaceID   *uuid.UUID
	ReportType    domain.InsuranceReportType
	AgencyType    domain.InsuranceAgency
	EmployeeID    *uuid.UUID
	IsBatch       bool
	ReportYear    *int
	ReportMonth   *int
	EffectiveDate *time.Time
	ReportData    domain.InsuranceJSON
	Items         []InsuranceReportItemInput
}

// ---------------------------------------------------------------------------
// Workplaces
// ---------------------------------------------------------------------------

// CreateWorkplace registers a workplace with the agencies.
func (s *InsuranceService) CreateWorkplace(ctx context.Context, companyID uuid.UUID, in *InsuranceWorkplaceInput) (*domain.InsuranceWorkplace, error) {
	if in.BusinessNumber == "" || in.WorkplaceName == "" {
		return nil, fmt.Errorf("%w: business_number, workplace_name", domain.ErrInsuranceWorkplaceNotFound)
	}

	existing, err := s.repo.GetWorkplaceByBusinessNumber(ctx, companyID, in.BusinessNumber)
	if err != nil && !errors.Is(err, domain.ErrInsuranceWorkplaceNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrInsuranceWorkplaceExists
	}

	workplace := &domain.InsuranceWorkplace{CompanyID: companyID}
	applyWorkplaceInput(workplace, in)

	if err := s.repo.CreateWorkplace(ctx, workplace); err != nil {
		return nil, err
	}
	return workplace, nil
}

// GetWorkplace loads one workplace registration.
func (s *InsuranceService) GetWorkplace(ctx context.Context, companyID, id uuid.UUID) (*domain.InsuranceWorkplace, error) {
	return s.repo.GetWorkplaceByID(ctx, companyID, id)
}

// ListWorkplaces returns a page of workplace registrations.
func (s *InsuranceService) ListWorkplaces(ctx context.Context, filter *repository.InsuranceWorkplaceFilter) ([]*domain.InsuranceWorkplace, int64, error) {
	return s.repo.ListWorkplaces(ctx, filter)
}

// UpdateWorkplace modifies a workplace registration.
func (s *InsuranceService) UpdateWorkplace(ctx context.Context, companyID, id uuid.UUID, in *InsuranceWorkplaceInput) (*domain.InsuranceWorkplace, error) {
	workplace, err := s.repo.GetWorkplaceByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}

	// The business number is the identity the agencies key on, so a change to
	// it must not collide with another registration.
	if in.BusinessNumber != "" && in.BusinessNumber != workplace.BusinessNumber {
		other, err := s.repo.GetWorkplaceByBusinessNumber(ctx, companyID, in.BusinessNumber)
		if err != nil && !errors.Is(err, domain.ErrInsuranceWorkplaceNotFound) {
			return nil, err
		}
		if other != nil && other.ID != workplace.ID {
			return nil, domain.ErrInsuranceWorkplaceExists
		}
	}

	applyWorkplaceInput(workplace, in)
	if err := s.repo.UpdateWorkplace(ctx, workplace); err != nil {
		return nil, err
	}
	return workplace, nil
}

// applyWorkplaceInput copies the input onto a workplace record.
func applyWorkplaceInput(w *domain.InsuranceWorkplace, in *InsuranceWorkplaceInput) {
	w.BusinessNumber = in.BusinessNumber
	w.WorkplaceName = in.WorkplaceName
	w.NPSWorkplaceNumber = in.NPSWorkplaceNumber
	w.NHISWorkplaceNumber = in.NHISWorkplaceNumber
	w.EIWorkplaceNumber = in.EIWorkplaceNumber
	w.WCIWorkplaceNumber = in.WCIWorkplaceNumber
	w.NPSRegistered = in.NPSRegistered
	w.NHISRegistered = in.NHISRegistered
	w.EIRegistered = in.EIRegistered
	w.WCIRegistered = in.WCIRegistered
	w.Representative = in.Representative
	w.Phone = in.Phone
	w.Fax = in.Fax
	w.Email = in.Email
	w.ZipCode = in.ZipCode
	w.Address = in.Address
	w.AddressDetail = in.AddressDetail
	w.BusinessTypeCode = in.BusinessTypeCode
	w.BusinessTypeName = in.BusinessTypeName
	w.IsActive = in.IsActive
}

// ---------------------------------------------------------------------------
// Employee qualifications
// ---------------------------------------------------------------------------

// GetEmployeeInsurance loads one employee's qualification record.
func (s *InsuranceService) GetEmployeeInsurance(ctx context.Context, companyID, employeeID uuid.UUID) (*domain.EmployeeInsurance, error) {
	return s.repo.GetEmployeeInsurance(ctx, companyID, employeeID)
}

// UpsertEmployeeInsurance creates or replaces a qualification record.
func (s *InsuranceService) UpsertEmployeeInsurance(ctx context.Context, companyID, employeeID uuid.UUID, in *EmployeeInsuranceInput) (*domain.EmployeeInsurance, error) {
	if employeeID == uuid.Nil {
		return nil, domain.ErrEmployeeInsuranceNotFound
	}
	if in.DependentsCount < 0 {
		return nil, fmt.Errorf("dependents_count: %w", domain.ErrNegativeAmount)
	}

	record := &domain.EmployeeInsurance{
		CompanyID:                companyID,
		EmployeeID:               employeeID,
		NPSQualified:             in.NPSQualified,
		NHISQualified:            in.NHISQualified,
		EIQualified:              in.EIQualified,
		WCIQualified:             in.WCIQualified,
		NPSAcquisitionDate:       in.NPSAcquisitionDate,
		NHISAcquisitionDate:      in.NHISAcquisitionDate,
		EIAcquisitionDate:        in.EIAcquisitionDate,
		NPSLossDate:              in.NPSLossDate,
		NHISLossDate:             in.NHISLossDate,
		EILossDate:               in.EILossDate,
		NPSStandardRemuneration:  in.NPSStandardRemuneration,
		NHISStandardRemuneration: in.NHISStandardRemuneration,
		NHISGrade:                in.NHISGrade,
		DependentsCount:          in.DependentsCount,
		NPSReduced:               in.NPSReduced,
		NHISReduced:              in.NHISReduced,
		NPSReductionType:         in.NPSReductionType,
		NHISReductionType:        in.NHISReductionType,
	}

	if err := s.repo.UpsertEmployeeInsurance(ctx, record); err != nil {
		return nil, err
	}
	return s.repo.GetEmployeeInsurance(ctx, companyID, employeeID)
}

// ListEmployeeInsurance returns a page of qualification records.
func (s *InsuranceService) ListEmployeeInsurance(ctx context.Context, filter *repository.EmployeeInsuranceFilter) ([]*repository.EmployeeInsuranceRow, int64, error) {
	return s.repo.ListEmployeeInsurance(ctx, filter)
}

// ---------------------------------------------------------------------------
// Contributions
// ---------------------------------------------------------------------------

// ListContributions returns a page of assessed monthly contributions.
func (s *InsuranceService) ListContributions(ctx context.Context, filter *repository.InsuranceContributionFilter) ([]*repository.InsuranceContributionRow, int64, error) {
	if filter.Month != nil && (*filter.Month < 1 || *filter.Month > 12) {
		return nil, 0, domain.ErrInsuranceContributionRange
	}
	return s.repo.ListContributions(ctx, filter)
}

// SummarizeContributions totals one month across the company.
func (s *InsuranceService) SummarizeContributions(ctx context.Context, companyID uuid.UUID, year, month int) (*repository.InsuranceContributionSummary, error) {
	if month < 1 || month > 12 {
		return nil, domain.ErrInsuranceContributionRange
	}
	return s.repo.SummarizeContributions(ctx, companyID, year, month)
}

// PreviewContribution computes what one employee's 4대보험 would be for a
// monthly wage, without storing anything.
//
// It calls the same domain function the payroll calculation does, so the
// preview and the payslip can never disagree. A year whose rates have not been
// verified fails here exactly as it does there.
func (s *InsuranceService) PreviewContribution(ctx context.Context, monthlyWage int64, on time.Time,
	industryRate *int64, size domain.WorkplaceSize) (*domain.SocialInsuranceResult, error) {

	result, err := domain.CalculateSocialInsurance(monthlyWage, on)
	if err != nil {
		return nil, err
	}

	if industryRate != nil {
		wci, err := domain.CalculateIndustrialAccident(monthlyWage, *industryRate)
		if err != nil {
			return nil, err
		}
		result.IndustrialAccident = wci
	}

	return &result, nil
}

// ---------------------------------------------------------------------------
// Reports
// ---------------------------------------------------------------------------

// CreateReport opens a draft 신고서.
func (s *InsuranceService) CreateReport(ctx context.Context, companyID uuid.UUID,
	in *CreateInsuranceReportInput, userID *uuid.UUID) (*domain.InsuranceReport, error) {

	if !domain.IsValidReportAgency(in.AgencyType) {
		return nil, fmt.Errorf("%w: %s", domain.ErrInsuranceAgencyInvalid, in.AgencyType)
	}
	if !domain.IsValidReportType(in.ReportType) {
		return nil, fmt.Errorf("%w: %s", domain.ErrInsuranceReportTypeInvalid, in.ReportType)
	}
	if in.ReportMonth != nil && (*in.ReportMonth < 1 || *in.ReportMonth > 12) {
		return nil, domain.ErrInsuranceContributionRange
	}
	if in.WorkplaceID != nil {
		// Resolve the workplace through the tenant-scoped repository so a
		// workplace id belonging to another company cannot be attached.
		if _, err := s.repo.GetWorkplaceByID(ctx, companyID, *in.WorkplaceID); err != nil {
			return nil, err
		}
	}

	data := in.ReportData
	if data == nil {
		data = domain.InsuranceJSON{}
	}

	report := &domain.InsuranceReport{
		CompanyID:     companyID,
		WorkplaceID:   in.WorkplaceID,
		ReportType:    in.ReportType,
		AgencyType:    in.AgencyType,
		EmployeeID:    in.EmployeeID,
		IsBatch:       in.IsBatch,
		ReportYear:    in.ReportYear,
		ReportMonth:   in.ReportMonth,
		EffectiveDate: in.EffectiveDate,
		Status:        domain.InsuranceReportStatusDraft,
		ReportData:    data,
		CreatedBy:     userID,
	}

	items := make([]domain.InsuranceReportItem, 0, len(in.Items))
	for i, item := range in.Items {
		lineNo := item.LineNo
		if lineNo == 0 {
			lineNo = i + 1
		}
		items = append(items, domain.InsuranceReportItem{
			EmployeeID:           item.EmployeeID,
			LineNo:               lineNo,
			EmployeeName:         item.EmployeeName,
			ResidentNumberMasked: item.ResidentNumberMasked,
			BaseAmount:           item.BaseAmount,
			EmployeeAmount:       item.EmployeeAmount,
			EmployerAmount:       item.EmployerAmount,
			TotalAmount:          item.TotalAmount,
			ItemData:             item.ItemData,
			Status:               "pending",
		})
	}

	// The header and its lines land together: a 신고서 whose lines are missing
	// is a filing that would be transmitted short of the employees it covers.
	err := s.repo.WithTransaction(ctx, func(repo repository.InsuranceRepository) error {
		if err := repo.CreateReport(ctx, report); err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		return repo.ReplaceReportItems(ctx, companyID, report.ID, items)
	})
	if err != nil {
		return nil, err
	}

	report.Items = items
	return report, nil
}

// GetReport loads one 신고서 with its lines.
func (s *InsuranceService) GetReport(ctx context.Context, companyID, id uuid.UUID) (*domain.InsuranceReport, error) {
	report, err := s.repo.GetReportByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.ListReportItems(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	report.Items = items
	return report, nil
}

// ListReports returns a page of 신고서 records.
func (s *InsuranceService) ListReports(ctx context.Context, filter *repository.InsuranceReportFilter) ([]*domain.InsuranceReport, int64, error) {
	if filter.Month != nil && (*filter.Month < 1 || *filter.Month > 12) {
		return nil, 0, domain.ErrInsuranceContributionRange
	}
	return s.repo.ListReports(ctx, filter)
}

// DeleteReport removes a 신고서 that has not been filed.
func (s *InsuranceService) DeleteReport(ctx context.Context, companyID, id uuid.UUID) error {
	report, err := s.repo.GetReportByID(ctx, companyID, id)
	if err != nil {
		return err
	}
	if report.ReceiptNumber != "" {
		return fmt.Errorf("%w: 접수번호 %s", domain.ErrInsuranceReportSubmitted, report.ReceiptNumber)
	}
	if !report.CanBeModified() {
		return fmt.Errorf("%w: status=%s", domain.ErrInsuranceReportNotEditable, report.Status)
	}
	return s.repo.DeleteReport(ctx, companyID, id)
}

// SubmitReport would transmit a 신고서 to the 공단 over EDI.
//
// IT DOES NOT, AND IT SAYS SO. The state guards run first, so a report that has
// already been filed still reports that rather than the missing-transport
// error; then the call fails with domain.ErrInsuranceEDIUnavailable having
// written nothing at all - no status change, no edi_jobs row.
//
// What is missing, concretely:
//   - api/proto/insurance/v1/*.proto and the generated Go stubs. Only tax/v1
//     exists today.
//   - internal/grpcclient/insurance_client.go against python-services/
//     insurance-edi on :50052.
//   - A decrypt path for insurance_credentials (ARIA), which deliberately has
//     no reader in this codebase.
//
// When that client lands, the success test must include a non-empty 접수번호:
// recording an empty one is precisely how the tax invoice idempotency guard was
// defeated, letting a retry file the same document twice.
func (s *InsuranceService) SubmitReport(ctx context.Context, companyID, id uuid.UUID, userID *uuid.UUID) (*domain.InsuranceReport, error) {
	report, err := s.repo.GetReportByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if err := report.ValidateForSubmission(); err != nil {
		return nil, err
	}
	return nil, domain.ErrInsuranceEDIUnavailable
}

// CancelReport withdraws a 신고서 that has not reached a 공단.
func (s *InsuranceService) CancelReport(ctx context.Context, companyID, id uuid.UUID, reason string) (*domain.InsuranceReport, error) {
	report, err := s.repo.GetReportByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if report.ReceiptNumber != "" {
		// Withdrawing an accepted filing is itself a filing (취소신고), not a
		// status flip in this database.
		return nil, fmt.Errorf("%w: 접수번호 %s", domain.ErrInsuranceReportSubmitted, report.ReceiptNumber)
	}
	if !report.CanBeCancelled() {
		return nil, fmt.Errorf("%w: status=%s", domain.ErrInsuranceReportNotEditable, report.Status)
	}

	rows, err := s.repo.TransitionReport(ctx, companyID, id,
		report.Status, domain.InsuranceReportStatusCancelled, map[string]interface{}{
			"rejection_reason": reason,
		})
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrPayrollConcurrentChange
	}
	return s.GetReport(ctx, companyID, id)
}

// ---------------------------------------------------------------------------
// Credentials and EDI jobs
// ---------------------------------------------------------------------------

// ListCredentialStatus reports which agency credentials are registered and
// whether they are still valid.
//
// It can only ever return status. The repository projects
// insurance_credentials onto domain.InsuranceCredentialStatus, which has no
// field for any encrypted column, so there is no code path from here to a
// certificate, a password or an API secret.
func (s *InsuranceService) ListCredentialStatus(ctx context.Context, companyID uuid.UUID) ([]*domain.InsuranceCredentialStatus, error) {
	return s.repo.ListCredentialStatus(ctx, companyID)
}

// ListEDIJobs returns a page of EDI jobs.
func (s *InsuranceService) ListEDIJobs(ctx context.Context, filter *repository.EDIJobFilter) ([]*domain.EDIJob, int64, error) {
	return s.repo.ListEDIJobs(ctx, filter)
}

// GetEDIJob loads one EDI job.
func (s *InsuranceService) GetEDIJob(ctx context.Context, companyID, id uuid.UUID) (*domain.EDIJob, error) {
	return s.repo.GetEDIJobByID(ctx, companyID, id)
}
