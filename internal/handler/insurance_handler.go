package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// InsuranceHandler serves the 4대보험 endpoints.
//
// It shares respondPayrollError with the payroll handler: the two domains fail
// in the same ways (a missing rate year, a locked record, an unverified
// statutory input) and one table keeps those answers identical whichever screen
// asked.
type InsuranceHandler struct {
	service *service.InsuranceService
}

// NewInsuranceHandler creates an InsuranceHandler.
func NewInsuranceHandler(svc *service.InsuranceService) *InsuranceHandler {
	return &InsuranceHandler{service: svc}
}

// ---------------------------------------------------------------------------
// Workplaces
// ---------------------------------------------------------------------------

// ListWorkplaces handles GET /insurance/workplaces
func (h *InsuranceHandler) ListWorkplaces(c *gin.Context) {
	page, pageSize := parsePageParams(c, defaultPayrollPageSize)

	filter := &repository.InsuranceWorkplaceFilter{
		CompanyID: appctx.GetCompanyID(c),
		Page:      page,
		PageSize:  pageSize,
	}
	if raw := c.Query("is_active"); raw != "" {
		active := raw == "true" || raw == "1"
		filter.IsActive = &active
	}

	workplaces, total, err := h.service.ListWorkplaces(c.Request.Context(), filter)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.Paginated(c, dto.FromInsuranceWorkplaces(workplaces), page, pageSize, total)
}

// CreateWorkplace handles POST /insurance/workplaces
func (h *InsuranceHandler) CreateWorkplace(c *gin.Context) {
	var req dto.InsuranceWorkplaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		payrollBindError(c, err)
		return
	}

	workplace, err := h.service.CreateWorkplace(c.Request.Context(), appctx.GetCompanyID(c), workplaceInput(&req))
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.Created(c, dto.FromInsuranceWorkplace(workplace))
}

// GetWorkplace handles GET /insurance/workplaces/:id
func (h *InsuranceHandler) GetWorkplace(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "insurance workplace")
	if !ok {
		return
	}

	workplace, err := h.service.GetWorkplace(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromInsuranceWorkplace(workplace))
}

// UpdateWorkplace handles PUT /insurance/workplaces/:id
func (h *InsuranceHandler) UpdateWorkplace(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "insurance workplace")
	if !ok {
		return
	}

	var req dto.InsuranceWorkplaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		payrollBindError(c, err)
		return
	}

	workplace, err := h.service.UpdateWorkplace(c.Request.Context(), appctx.GetCompanyID(c), id, workplaceInput(&req))
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromInsuranceWorkplace(workplace))
}

// workplaceInput copies the request into the service input.
func workplaceInput(req *dto.InsuranceWorkplaceRequest) *service.InsuranceWorkplaceInput {
	return &service.InsuranceWorkplaceInput{
		BusinessNumber:      req.BusinessNumber,
		WorkplaceName:       req.WorkplaceName,
		NPSWorkplaceNumber:  req.NPSWorkplaceNumber,
		NHISWorkplaceNumber: req.NHISWorkplaceNumber,
		EIWorkplaceNumber:   req.EIWorkplaceNumber,
		WCIWorkplaceNumber:  req.WCIWorkplaceNumber,
		NPSRegistered:       req.NPSRegistered,
		NHISRegistered:      req.NHISRegistered,
		EIRegistered:        req.EIRegistered,
		WCIRegistered:       req.WCIRegistered,
		Representative:      req.Representative,
		Phone:               req.Phone,
		Fax:                 req.Fax,
		Email:               req.Email,
		ZipCode:             req.ZipCode,
		Address:             req.Address,
		AddressDetail:       req.AddressDetail,
		BusinessTypeCode:    req.BusinessTypeCode,
		BusinessTypeName:    req.BusinessTypeName,
		IsActive:            req.IsActive,
	}
}

// ---------------------------------------------------------------------------
// Employee qualifications
// ---------------------------------------------------------------------------

// ListEmployeeInsurance handles GET /insurance/employees
func (h *InsuranceHandler) ListEmployeeInsurance(c *gin.Context) {
	page, pageSize := parsePageParams(c, defaultPayrollPageSize)

	filter := &repository.EmployeeInsuranceFilter{
		CompanyID:  appctx.GetCompanyID(c),
		EmployeeID: payrollQueryUUID(c, "employee_id"),
		Search:     c.Query("search"),
		Page:       page,
		PageSize:   pageSize,
	}

	rows, total, err := h.service.ListEmployeeInsurance(c.Request.Context(), filter)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.Paginated(c, dto.FromEmployeeInsuranceRows(rows), page, pageSize, total)
}

// GetEmployeeInsurance handles GET /insurance/employees/:employee_id
func (h *InsuranceHandler) GetEmployeeInsurance(c *gin.Context) {
	employeeID, ok := payrollPathUUID(c, "employee_id", "employee")
	if !ok {
		return
	}

	record, err := h.service.GetEmployeeInsurance(c.Request.Context(), appctx.GetCompanyID(c), employeeID)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromEmployeeInsurance(record))
}

// UpsertEmployeeInsurance handles PUT /insurance/employees/:employee_id
func (h *InsuranceHandler) UpsertEmployeeInsurance(c *gin.Context) {
	employeeID, ok := payrollPathUUID(c, "employee_id", "employee")
	if !ok {
		return
	}

	var req dto.EmployeeInsuranceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		payrollBindError(c, err)
		return
	}

	dates, err := req.ParseDates()
	if err != nil {
		respondPayrollError(c, err)
		return
	}

	record, err := h.service.UpsertEmployeeInsurance(c.Request.Context(), appctx.GetCompanyID(c), employeeID,
		&service.EmployeeInsuranceInput{
			NPSQualified:             req.NPSQualified,
			NHISQualified:            req.NHISQualified,
			EIQualified:              req.EIQualified,
			WCIQualified:             req.WCIQualified,
			NPSAcquisitionDate:       dates.NPSAcquisition,
			NHISAcquisitionDate:      dates.NHISAcquisition,
			EIAcquisitionDate:        dates.EIAcquisition,
			NPSLossDate:              dates.NPSLoss,
			NHISLossDate:             dates.NHISLoss,
			EILossDate:               dates.EILoss,
			NPSStandardRemuneration:  req.NPSStandardRemuneration,
			NHISStandardRemuneration: req.NHISStandardRemuneration,
			NHISGrade:                req.NHISGrade,
			DependentsCount:          req.DependentsCount,
			NPSReduced:               req.NPSReduced,
			NHISReduced:              req.NHISReduced,
			NPSReductionType:         req.NPSReductionType,
			NHISReductionType:        req.NHISReductionType,
		})
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromEmployeeInsurance(record))
}

// ---------------------------------------------------------------------------
// Contributions
// ---------------------------------------------------------------------------

// ListContributions handles GET /insurance/contributions
func (h *InsuranceHandler) ListContributions(c *gin.Context) {
	page, pageSize := parsePageParams(c, defaultPayrollPageSize)

	filter := &repository.InsuranceContributionFilter{
		CompanyID:  appctx.GetCompanyID(c),
		EmployeeID: payrollQueryUUID(c, "employee_id"),
		Year:       payrollQueryInt(c, "year"),
		Month:      payrollQueryInt(c, "month"),
		Page:       page,
		PageSize:   pageSize,
	}

	rows, total, err := h.service.ListContributions(c.Request.Context(), filter)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.Paginated(c, dto.FromInsuranceContributionRows(rows), page, pageSize, total)
}

// GetSummary handles GET /insurance/summary
//
// The dashboard cards read this instead of totalling a page of rows, which
// would only ever add up the rows that happened to be on screen.
func (h *InsuranceHandler) GetSummary(c *gin.Context) {
	year := payrollQueryInt(c, "year")
	month := payrollQueryInt(c, "month")
	if year == nil || month == nil {
		response.BadRequest(c, "year 와 month 를 지정해야 합니다")
		return
	}

	summary, err := h.service.SummarizeContributions(c.Request.Context(), appctx.GetCompanyID(c), *year, *month)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromInsuranceSummary(summary, *year, *month))
}

// PreviewContribution handles POST /insurance/contributions/preview
//
// It calls the same domain function the payroll calculation does, so the
// preview and the payslip cannot disagree, and it stores nothing.
func (h *InsuranceHandler) PreviewContribution(c *gin.Context) {
	var req dto.ContributionPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		payrollBindError(c, err)
		return
	}

	on, size, err := req.Parsed()
	if err != nil {
		respondPayrollError(c, err)
		return
	}

	result, err := h.service.PreviewContribution(c.Request.Context(), req.MonthlyWage, on, req.IndustrialAccidentRate, size)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromSocialInsuranceResult(result))
}

// ---------------------------------------------------------------------------
// Reports
// ---------------------------------------------------------------------------

// ListReports handles GET /insurance/reports
func (h *InsuranceHandler) ListReports(c *gin.Context) {
	page, pageSize := parsePageParams(c, defaultPayrollPageSize)

	filter := &repository.InsuranceReportFilter{
		CompanyID:  appctx.GetCompanyID(c),
		EmployeeID: payrollQueryUUID(c, "employee_id"),
		Year:       payrollQueryInt(c, "year"),
		Month:      payrollQueryInt(c, "month"),
		Page:       page,
		PageSize:   pageSize,
	}
	if raw := c.Query("agency_type"); raw != "" {
		a := domain.InsuranceAgency(raw)
		filter.AgencyType = &a
	}
	if raw := c.Query("report_type"); raw != "" {
		t := domain.InsuranceReportType(raw)
		filter.ReportType = &t
	}
	if raw := c.Query("status"); raw != "" {
		s := domain.InsuranceReportStatus(raw)
		filter.Status = &s
	}

	reports, total, err := h.service.ListReports(c.Request.Context(), filter)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.Paginated(c, dto.FromInsuranceReports(reports), page, pageSize, total)
}

// CreateReport handles POST /insurance/reports
func (h *InsuranceHandler) CreateReport(c *gin.Context) {
	var req dto.CreateInsuranceReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		payrollBindError(c, err)
		return
	}

	effectiveDate, err := dto.ParseOptionalPayrollDate(req.EffectiveDate)
	if err != nil {
		respondPayrollError(c, err)
		return
	}

	input := &service.CreateInsuranceReportInput{
		ReportType:    domain.InsuranceReportType(req.ReportType),
		AgencyType:    domain.InsuranceAgency(req.AgencyType),
		IsBatch:       req.IsBatch,
		ReportYear:    req.ReportYear,
		ReportMonth:   req.ReportMonth,
		EffectiveDate: effectiveDate,
		ReportData:    domain.InsuranceJSON(req.ReportData),
	}

	if req.WorkplaceID != "" {
		workplaceID, parseErr := uuid.Parse(req.WorkplaceID)
		if parseErr != nil {
			payrollInvalidID(c, "insurance workplace")
			return
		}
		input.WorkplaceID = &workplaceID
	}
	if req.EmployeeID != "" {
		employeeID, parseErr := uuid.Parse(req.EmployeeID)
		if parseErr != nil {
			payrollInvalidID(c, "employee")
			return
		}
		input.EmployeeID = &employeeID
	}

	for _, item := range req.Items {
		employeeID, parseErr := uuid.Parse(item.EmployeeID)
		if parseErr != nil {
			payrollInvalidID(c, "employee")
			return
		}
		input.Items = append(input.Items, service.InsuranceReportItemInput{
			EmployeeID:           employeeID,
			LineNo:               item.LineNo,
			EmployeeName:         item.EmployeeName,
			ResidentNumberMasked: item.ResidentNumberMasked,
			BaseAmount:           item.BaseAmount,
			EmployeeAmount:       item.EmployeeAmount,
			EmployerAmount:       item.EmployerAmount,
			TotalAmount:          item.TotalAmount,
			ItemData:             domain.InsuranceJSON(item.ItemData),
		})
	}

	userID := appctx.GetUserID(c)
	report, err := h.service.CreateReport(c.Request.Context(), appctx.GetCompanyID(c), input, &userID)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.Created(c, dto.FromInsuranceReport(report))
}

// GetReport handles GET /insurance/reports/:id
func (h *InsuranceHandler) GetReport(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "insurance report")
	if !ok {
		return
	}

	report, err := h.service.GetReport(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromInsuranceReport(report))
}

// DeleteReport handles DELETE /insurance/reports/:id
func (h *InsuranceHandler) DeleteReport(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "insurance report")
	if !ok {
		return
	}

	if err := h.service.DeleteReport(c.Request.Context(), appctx.GetCompanyID(c), id); err != nil {
		respondPayrollError(c, err)
		return
	}
	response.NoContent(c)
}

// SubmitReport handles POST /insurance/reports/:id/submit
//
// It answers 501. The 공단 EDI transport is not wired, and reporting success
// here would tell an operator a statutory filing had been made when nothing
// left the building. The service writes no status change and creates no
// edi_jobs row before failing.
func (h *InsuranceHandler) SubmitReport(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "insurance report")
	if !ok {
		return
	}

	userID := appctx.GetUserID(c)
	report, err := h.service.SubmitReport(c.Request.Context(), appctx.GetCompanyID(c), id, &userID)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromInsuranceReport(report))
}

// CancelReport handles POST /insurance/reports/:id/cancel
func (h *InsuranceHandler) CancelReport(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "insurance report")
	if !ok {
		return
	}

	var req dto.CancelInsuranceReportRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			payrollBindError(c, err)
			return
		}
	}

	report, err := h.service.CancelReport(c.Request.Context(), appctx.GetCompanyID(c), id, req.Reason)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromInsuranceReport(report))
}

// ---------------------------------------------------------------------------
// Credentials and EDI jobs
// ---------------------------------------------------------------------------

// ListCredentialStatus handles GET /insurance/credentials
//
// It returns status only - which agency has a credential on file, whether it is
// active and when the certificate expires. The certificate, its password, the
// portal login and the API secret are ARIA-encrypted columns that the domain
// model does not map, so no code path leads from this endpoint to them.
func (h *InsuranceHandler) ListCredentialStatus(c *gin.Context) {
	rows, err := h.service.ListCredentialStatus(c.Request.Context(), appctx.GetCompanyID(c))
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromInsuranceCredentialStatuses(rows))
}

// ListEDIJobs handles GET /insurance/edi-jobs
func (h *InsuranceHandler) ListEDIJobs(c *gin.Context) {
	page, pageSize := parsePageParams(c, defaultPayrollPageSize)

	filter := &repository.EDIJobFilter{
		CompanyID: appctx.GetCompanyID(c),
		ReportID:  payrollQueryUUID(c, "report_id"),
		Page:      page,
		PageSize:  pageSize,
	}
	if raw := c.Query("status"); raw != "" {
		s := domain.EDIJobStatus(raw)
		filter.Status = &s
	}

	jobs, total, err := h.service.ListEDIJobs(c.Request.Context(), filter)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.Paginated(c, dto.FromEDIJobs(jobs), page, pageSize, total)
}

// GetEDIJob handles GET /insurance/edi-jobs/:id
func (h *InsuranceHandler) GetEDIJob(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "EDI job")
	if !ok {
		return
	}

	job, err := h.service.GetEDIJob(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromEDIJob(job))
}
