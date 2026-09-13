package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// newInsuranceFixture builds an InsuranceService over the same in-memory store
// the payroll tests use, so a rolled-back transaction really rolls back.
func newInsuranceFixture(t *testing.T) (*service.InsuranceService, *payrollFakeStore) {
	t.Helper()
	store := newPayrollFakeStore()
	return service.NewInsuranceService(&payrollFakeInsuranceRepo{store: store}), store
}

// draftReport puts one draft 자격취득신고 into the store.
func draftReport(t *testing.T, svc *service.InsuranceService) *domain.InsuranceReport {
	t.Helper()

	report, err := svc.CreateReport(context.Background(), payrollTestCompanyID, &service.CreateInsuranceReportInput{
		ReportType:    domain.InsuranceReportAcquisition,
		AgencyType:    domain.InsuranceAgencyNPS,
		EmployeeID:    &payrollTestEmployeeID,
		EffectiveDate: payrollTimePtr(time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)),
		ReportData:    domain.InsuranceJSON{"standard_remuneration": 3000000},
	}, &payrollTestUserID)
	require.NoError(t, err)
	return report
}

// TestSubmitReportFailsClosed is the important one.
//
// There is no Go client for the insurance-edi service, so submitting must fail
// LOUDLY and change nothing. The failure mode being guarded against is the one
// this repository has already been bitten by: a transmit path that marks the
// document sent without anything having left the building, so the operator
// stops chasing a statutory filing that never happened.
func TestSubmitReportFailsClosed(t *testing.T) {
	svc, store := newInsuranceFixture(t)
	report := draftReport(t, svc)

	_, err := svc.SubmitReport(context.Background(), payrollTestCompanyID, report.ID, &payrollTestUserID)
	assert.ErrorIs(t, err, domain.ErrInsuranceEDIUnavailable)

	stored := store.reports[report.ID]
	assert.Equal(t, domain.InsuranceReportStatusDraft, stored.Status,
		"전송 경로가 없는데 상태가 바뀌면 신고했다고 오인하게 된다")
	assert.Empty(t, stored.ReceiptNumber)
	assert.Nil(t, stored.SubmittedAt)
}

// TestSubmitReportChecksIdempotencyBeforeTransport verifies the ordering of the
// two guards: a report that already carries a 접수번호 must report THAT, not the
// missing transport, so an operator is never told to retry a filing that has
// already been accepted.
func TestSubmitReportChecksIdempotencyBeforeTransport(t *testing.T) {
	svc, store := newInsuranceFixture(t)
	report := draftReport(t, svc)

	// Simulate a filing that was made through the 공단 portal and recorded.
	store.reports[report.ID].ReceiptNumber = "2025-NPS-000123"
	store.reports[report.ID].Status = domain.InsuranceReportStatusSubmitted

	_, err := svc.SubmitReport(context.Background(), payrollTestCompanyID, report.ID, &payrollTestUserID)
	assert.ErrorIs(t, err, domain.ErrInsuranceReportSubmitted)
	assert.NotErrorIs(t, err, domain.ErrInsuranceEDIUnavailable)
}

// TestSubmittedReportCannotBeCancelledOrDeleted checks that an accepted filing
// is not withdrawn by flipping a status column: withdrawing it is itself a
// filing (취소신고).
func TestSubmittedReportCannotBeCancelledOrDeleted(t *testing.T) {
	svc, store := newInsuranceFixture(t)
	report := draftReport(t, svc)

	store.reports[report.ID].ReceiptNumber = "2025-NPS-000123"
	store.reports[report.ID].Status = domain.InsuranceReportStatusAccepted

	_, err := svc.CancelReport(context.Background(), payrollTestCompanyID, report.ID, "실수")
	assert.ErrorIs(t, err, domain.ErrInsuranceReportSubmitted)

	err = svc.DeleteReport(context.Background(), payrollTestCompanyID, report.ID)
	assert.ErrorIs(t, err, domain.ErrInsuranceReportSubmitted)

	assert.Equal(t, domain.InsuranceReportStatusAccepted, store.reports[report.ID].Status)
}

func TestCancelDraftReport(t *testing.T) {
	svc, store := newInsuranceFixture(t)
	report := draftReport(t, svc)

	cancelled, err := svc.CancelReport(context.Background(), payrollTestCompanyID, report.ID, "중복 작성")
	require.NoError(t, err)
	assert.Equal(t, domain.InsuranceReportStatusCancelled, cancelled.Status)
	assert.Equal(t, "중복 작성", store.reports[report.ID].RejectionReason)
}

func TestCreateReportValidatesAgencyAndType(t *testing.T) {
	svc, _ := newInsuranceFixture(t)

	_, err := svc.CreateReport(context.Background(), payrollTestCompanyID, &service.CreateInsuranceReportInput{
		ReportType: domain.InsuranceReportAcquisition,
		// comwel is a credentials/EDI agency, not one the report CHECK accepts.
		AgencyType: domain.InsuranceAgencyComwel,
	}, &payrollTestUserID)
	assert.ErrorIs(t, err, domain.ErrInsuranceAgencyInvalid)

	_, err = svc.CreateReport(context.Background(), payrollTestCompanyID, &service.CreateInsuranceReportInput{
		ReportType: domain.InsuranceReportType("something_else"),
		AgencyType: domain.InsuranceAgencyNPS,
	}, &payrollTestUserID)
	assert.ErrorIs(t, err, domain.ErrInsuranceReportTypeInvalid)

	month := 13
	_, err = svc.CreateReport(context.Background(), payrollTestCompanyID, &service.CreateInsuranceReportInput{
		ReportType:  domain.InsuranceReportMonthly,
		AgencyType:  domain.InsuranceAgencyNHIS,
		ReportMonth: &month,
	}, &payrollTestUserID)
	assert.ErrorIs(t, err, domain.ErrInsuranceContributionRange)
}

func TestCreateReportStoresHeaderAndLinesTogether(t *testing.T) {
	svc, store := newInsuranceFixture(t)

	report, err := svc.CreateReport(context.Background(), payrollTestCompanyID, &service.CreateInsuranceReportInput{
		ReportType: domain.InsuranceReportMonthly,
		AgencyType: domain.InsuranceAgencyNHIS,
		IsBatch:    true,
		Items: []service.InsuranceReportItemInput{
			{EmployeeID: payrollTestEmployeeID, EmployeeName: "홍길동", BaseAmount: 3_000_000,
				EmployeeAmount: 106_350, EmployerAmount: 106_350, TotalAmount: 212_700},
			{EmployeeID: uuid.New(), EmployeeName: "김철수", BaseAmount: 2_500_000,
				EmployeeAmount: 88_620, EmployerAmount: 88_620, TotalAmount: 177_240},
		},
	}, &payrollTestUserID)
	require.NoError(t, err)

	rows := store.reportItems[report.ID]
	require.Len(t, rows, 2)
	// Line numbers are assigned when the caller leaves them at zero, so the
	// filing order is deterministic.
	assert.Equal(t, 1, rows[0].LineNo)
	assert.Equal(t, 2, rows[1].LineNo)
}

func TestReportCrossTenantAccessIsRefused(t *testing.T) {
	svc, _ := newInsuranceFixture(t)
	report := draftReport(t, svc)

	other := uuid.MustParse("0192f3a0-0000-7000-8000-00000000c0c1")

	_, err := svc.GetReport(context.Background(), other, report.ID)
	assert.ErrorIs(t, err, domain.ErrInsuranceReportNotFound)

	_, err = svc.SubmitReport(context.Background(), other, report.ID, &payrollTestUserID)
	assert.ErrorIs(t, err, domain.ErrInsuranceReportNotFound)

	assert.ErrorIs(t, svc.DeleteReport(context.Background(), other, report.ID), domain.ErrInsuranceReportNotFound)
}

// TestPreviewContributionMatchesPayroll checks that the standalone 4대보험
// preview and the payroll calculation agree, because both call the same domain
// function. Two independent implementations would drift.
func TestPreviewContributionMatchesPayroll(t *testing.T) {
	svc, _ := newInsuranceFixture(t)

	on := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.PreviewContribution(context.Background(), 3_000_000, on, payrollInt64p(1530), domain.WorkplaceUnder150)
	require.NoError(t, err)

	// Same hand-checked figures as internal/domain/payroll_test.go.
	assert.Equal(t, int64(135_000), result.NationalPension.Employee)
	assert.Equal(t, int64(106_350), result.HealthInsurance.Employee)
	assert.Equal(t, int64(13_770), result.LongTermCare.Employee)
	assert.Equal(t, int64(27_000), result.EmploymentInsurance.Employee)
	assert.Equal(t, int64(45_900), result.IndustrialAccident.Employer)
	assert.Equal(t, int64(282_120), result.EmployeeTotal())
	assert.NotEmpty(t, result.RatesSource)
}

func TestPreviewContributionRefusesUnverifiedYear(t *testing.T) {
	svc, _ := newInsuranceFixture(t)

	_, err := svc.PreviewContribution(context.Background(), 3_000_000,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), nil, domain.WorkplaceUnder150)
	assert.ErrorIs(t, err, domain.ErrSocialInsuranceRatesUnavailable)
}

func TestSummarizeContributionsValidatesMonth(t *testing.T) {
	svc, _ := newInsuranceFixture(t)

	_, err := svc.SummarizeContributions(context.Background(), payrollTestCompanyID, 2025, 0)
	assert.ErrorIs(t, err, domain.ErrInsuranceContributionRange)

	_, err = svc.SummarizeContributions(context.Background(), payrollTestCompanyID, 2025, 13)
	assert.ErrorIs(t, err, domain.ErrInsuranceContributionRange)
}

// TestEmployeeInsuranceQualificationRoundTrip checks the conversion the payroll
// calculation depends on.
func TestEmployeeInsuranceQualificationRoundTrip(t *testing.T) {
	record := &domain.EmployeeInsurance{
		NPSQualified:  false,
		NHISQualified: true,
		EIQualified:   true,
		WCIQualified:  false,
	}
	q := record.Qualification()
	assert.False(t, q.NationalPension)
	assert.True(t, q.HealthInsurance)
	assert.True(t, q.EmploymentInsurance)
	assert.False(t, q.IndustrialAccident)

	// No record on file means a regular full-time hire covered by all four.
	var missing *domain.EmployeeInsurance
	assert.Equal(t, domain.AllQualified(), missing.Qualification())
}
