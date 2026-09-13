package handler

import (
	stderrors "errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/errors"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// PayrollHandlers groups the payroll and 4대보험 handlers.
//
// It mirrors HRHandlers: internal/router/payroll_routes.go is wired with one
// field on handler.Handlers instead of two, and this package owns the
// repository and service construction rather than spreading it across the
// router.
type PayrollHandlers struct {
	Payroll   *PayrollHandler
	Insurance *InsuranceHandler
}

// NewPayrollHandlers builds the payroll handlers with their dependencies.
//
// db may be nil: nothing here issues a query at construction time, which is
// what lets the router tests build the whole route tree without a database.
func NewPayrollHandlers(db *gorm.DB) *PayrollHandlers {
	payrollRepo := repository.NewPayrollRepository(db)
	insuranceRepo := repository.NewInsuranceRepository(db)

	payrollService := service.NewPayrollService(payrollRepo, insuranceRepo)
	insuranceService := service.NewInsuranceService(insuranceRepo)

	return &PayrollHandlers{
		Payroll:   NewPayrollHandler(payrollService),
		Insurance: NewInsuranceHandler(insuranceService),
	}
}

// PayrollHandler serves the 급여 endpoints.
type PayrollHandler struct {
	service *service.PayrollService
}

// NewPayrollHandler creates a PayrollHandler.
func NewPayrollHandler(svc *service.PayrollService) *PayrollHandler {
	return &PayrollHandler{service: svc}
}

// defaultPayrollPageSize is the page size a payroll list uses when the client
// does not ask for one. A monthly register is normally read a department at a
// time; the cap in internal/handler/response still applies.
const defaultPayrollPageSize = 20

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

// payrollErrorMapping is one client-safe answer for one known error.
type payrollErrorMapping struct {
	err     error
	status  int
	code    string
	message string
}

// payrollErrorMappings turns the payroll and insurance errors into responses.
//
// The table is explicit on purpose: err.Error() is never sent to the client.
// Anything not listed here may be a GORM or pgx error whose text carries table,
// column and constraint names - and in this domain the bind parameters of a
// statement that touched somebody's salary.
//
// The three "not loaded" entries are the important ones. internal/domain
// refuses to invent a 4대보험 rate for an unverified year, a 국민연금
// 기준소득월액 for an unpublished window, or a 근로소득 간이세액. Each of those
// failures is translated here into a sentence that tells the operator exactly
// what is missing, instead of surfacing as a 500.
var payrollErrorMappings = []payrollErrorMapping{
	// ---- 404 ---------------------------------------------------------------
	{domain.ErrPayrollNotFound, http.StatusNotFound, errors.CodeNotFound, "급여 자료를 찾을 수 없습니다"},
	{domain.ErrPayrollPeriodNotFound, http.StatusNotFound, errors.CodeNotFound, "급여기간을 찾을 수 없습니다"},
	{domain.ErrInsuranceWorkplaceNotFound, http.StatusNotFound, errors.CodeNotFound, "4대보험 사업장을 찾을 수 없습니다"},
	{domain.ErrEmployeeInsuranceNotFound, http.StatusNotFound, errors.CodeNotFound, "사원의 4대보험 자격 정보가 없습니다"},
	{domain.ErrInsuranceReportNotFound, http.StatusNotFound, errors.CodeNotFound, "4대보험 신고서를 찾을 수 없습니다"},

	// ---- 409 duplicates and races -----------------------------------------
	{domain.ErrPayrollPeriodExists, http.StatusConflict, errors.CodeAlreadyExists, "해당 연월의 급여기간이 이미 있습니다"},
	{domain.ErrPayrollExists, http.StatusConflict, errors.CodeAlreadyExists, "해당 사원의 해당 연월 급여가 이미 있습니다"},
	{domain.ErrInsuranceWorkplaceExists, http.StatusConflict, errors.CodeAlreadyExists, "같은 사업자등록번호의 사업장이 이미 있습니다"},
	{domain.ErrInsuranceReportSubmitted, http.StatusConflict, errors.CodeConflict, "이미 접수된 신고서입니다. 취소·정정은 별도 신고로 처리해야 합니다"},
	{service.ErrPayrollConcurrentChange, http.StatusConflict, errors.CodeConflict, "다른 요청이 먼저 상태를 변경했습니다. 새로 고친 뒤 다시 시도하십시오"},

	// ---- 422 state ---------------------------------------------------------
	{domain.ErrPayrollNotEditable, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction, "현재 상태에서는 급여를 변경할 수 없습니다"},
	{domain.ErrPayrollPeriodNotEditable, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction, "현재 상태에서는 급여기간을 변경할 수 없습니다"},
	{domain.ErrPayrollNotCalculated, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction, "계산되지 않은 급여가 있어 확정할 수 없습니다"},
	{service.ErrPayrollPeriodLocked, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction, "급여기간이 확정되어 개별 급여를 변경할 수 없습니다"},
	{service.ErrPayrollPeriodEmpty, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction, "급여기간에 계산 대상 급여 자료가 없습니다"},
	{domain.ErrInsuranceReportNotEditable, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction, "현재 상태에서는 신고서를 변경할 수 없습니다"},

	// A header that disagrees with its rows is never approved. The client is
	// told to recalculate rather than being handed the two numbers.
	{domain.ErrPayrollTotalsMismatch, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction,
		"급여 합계와 급여항목 합계가 일치하지 않습니다. 해당 급여를 다시 계산하십시오"},

	// ---- 422 missing statutory data ---------------------------------------
	{domain.ErrSocialInsuranceRatesUnavailable, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction,
		"해당 연도의 4대보험 요율이 등록되어 있지 않습니다. 확정 고시 요율을 등록한 뒤 다시 계산하십시오"},
	{domain.ErrPensionBaseUnavailable, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction,
		"해당 기간의 국민연금 기준소득월액 상·하한이 등록되어 있지 않습니다. 보건복지부 고시를 등록한 뒤 다시 계산하십시오"},
	{domain.ErrSimplifiedTaxTableUnavailable, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction,
		"근로소득 간이세액표가 적재되지 않아 소득세를 자동 계산할 수 없습니다. income_tax 를 직접 입력하십시오"},

	// ---- 400 input ---------------------------------------------------------
	{domain.ErrNegativeAmount, http.StatusBadRequest, errors.CodeInvalidInput, "금액은 음수일 수 없습니다"},
	{domain.ErrNonTaxableExceedsGross, http.StatusBadRequest, errors.CodeInvalidInput, "비과세 금액이 지급총액을 초과합니다"},
	{domain.ErrPayrollEmptyItemName, http.StatusBadRequest, errors.CodeMissingField, "급여항목에는 코드와 이름이 필요합니다"},
	{domain.ErrPayrollMonthInvalid, http.StatusBadRequest, errors.CodeOutOfRange, "급여월은 1에서 12 사이여야 합니다"},
	{domain.ErrPayrollPeriodDates, http.StatusBadRequest, errors.CodeInvalidInput, "급여기간 시작일이 종료일보다 늦을 수 없습니다"},
	{domain.ErrInsuranceContributionRange, http.StatusBadRequest, errors.CodeOutOfRange, "월은 1에서 12 사이여야 합니다"},
	{domain.ErrInsuranceAgencyInvalid, http.StatusBadRequest, errors.CodeInvalidInput, "알 수 없는 공단 구분입니다"},
	{domain.ErrInsuranceReportTypeInvalid, http.StatusBadRequest, errors.CodeInvalidInput, "알 수 없는 신고 종류입니다"},
	{dto.ErrInvalidPayrollDate, http.StatusBadRequest, errors.CodeInvalidFormat, "날짜는 YYYY-MM-DD 형식이어야 합니다"},

	// ---- 501 not implemented ----------------------------------------------
	//
	// The 공단 transmission path has no Go client. Answering 501 - rather than
	// recording the report as submitted - is the whole point: an operator who
	// sees a success here would stop chasing a statutory filing that never
	// happened.
	{domain.ErrInsuranceEDIUnavailable, http.StatusNotImplemented, errors.CodeUnavailable,
		"4대보험 EDI 전송이 아직 연결되지 않았습니다. 공단 포털에서 직접 신고한 뒤 접수번호를 기록하십시오"},
}

// respondPayrollError writes the client-safe response for a payroll error.
//
// Anything not in the table is a 500 whose cause is logged with the request id
// and never sent to the client.
func respondPayrollError(c *gin.Context, err error) {
	for _, mapping := range payrollErrorMappings {
		if stderrors.Is(err, mapping.err) {
			response.Error(c, mapping.status, mapping.code, mapping.message)
			return
		}
	}
	response.InternalErrorLogged(c, "Internal server error", err)
}

// payrollBindError answers a request body that failed binding. The validator's
// message names the offending field and rule, which is safe; it is not a
// database error.
func payrollBindError(c *gin.Context, err error) {
	response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "요청 본문이 올바르지 않습니다", err.Error())
}

// payrollInvalidID answers a path parameter that is not a UUID.
func payrollInvalidID(c *gin.Context, what string) {
	response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "잘못된 "+what+" ID 입니다")
}

// payrollPathUUID reads a UUID path parameter.
func payrollPathUUID(c *gin.Context, param, what string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		payrollInvalidID(c, what)
		return uuid.Nil, false
	}
	return id, true
}

// payrollQueryInt reads an optional integer query parameter.
func payrollQueryInt(c *gin.Context, key string) *int {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	return &v
}

// payrollQueryUUID reads an optional UUID query parameter. A malformed value is
// ignored rather than rejected, matching the other list endpoints.
func payrollQueryUUID(c *gin.Context, key string) *uuid.UUID {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}

// ---------------------------------------------------------------------------
// Payroll periods
// ---------------------------------------------------------------------------

// ListPeriods handles GET /payroll-periods
func (h *PayrollHandler) ListPeriods(c *gin.Context) {
	page, pageSize := parsePageParams(c, defaultPayrollPageSize)

	filter := &repository.PayrollPeriodFilter{
		CompanyID: appctx.GetCompanyID(c),
		PayYear:   payrollQueryInt(c, "pay_year"),
		PayMonth:  payrollQueryInt(c, "pay_month"),
		Page:      page,
		PageSize:  pageSize,
	}
	if status := c.Query("status"); status != "" {
		s := domain.PayrollPeriodStatus(status)
		filter.Status = &s
	}

	periods, total, err := h.service.ListPeriods(c.Request.Context(), filter)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.Paginated(c, dto.FromPayrollPeriods(periods), page, pageSize, total)
}

// CreatePeriod handles POST /payroll-periods
func (h *PayrollHandler) CreatePeriod(c *gin.Context) {
	var req dto.CreatePayrollPeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		payrollBindError(c, err)
		return
	}

	start, err := dto.ParseRequiredPayrollDate(req.PeriodStart)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	end, err := dto.ParseRequiredPayrollDate(req.PeriodEnd)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	paymentDate, err := dto.ParseOptionalPayrollDate(req.PaymentDate)
	if err != nil {
		respondPayrollError(c, err)
		return
	}

	period, err := h.service.CreatePeriod(c.Request.Context(), appctx.GetCompanyID(c), &service.CreatePayrollPeriodInput{
		PayYear:     req.PayYear,
		PayMonth:    req.PayMonth,
		PeriodName:  req.PeriodName,
		PeriodStart: start,
		PeriodEnd:   end,
		PaymentDate: paymentDate,
	})
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.Created(c, dto.FromPayrollPeriod(period))
}

// GetPeriod handles GET /payroll-periods/:id
func (h *PayrollHandler) GetPeriod(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "payroll period")
	if !ok {
		return
	}

	period, err := h.service.GetPeriod(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromPayrollPeriod(period))
}

// CalculatePeriod handles POST /payroll-periods/:id/calculate
func (h *PayrollHandler) CalculatePeriod(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "payroll period")
	if !ok {
		return
	}

	var req dto.CalculatePeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		payrollBindError(c, err)
		return
	}

	contributionDate, err := dto.ParseOptionalPayrollDate(req.ContributionDate)
	if err != nil {
		respondPayrollError(c, err)
		return
	}

	size := domain.WorkplaceUnder150
	if req.WorkplaceSize != "" {
		size = domain.WorkplaceSize(req.WorkplaceSize)
	}

	input := &service.CalculatePeriodInput{
		WorkplaceSize:          size,
		IndustrialAccidentRate: req.IndustrialAccidentRate,
		ContributionDate:       contributionDate,
		Employees:              map[uuid.UUID]service.CalculatePayrollInput{},
	}
	for _, emp := range req.Employees {
		employeeID, parseErr := uuid.Parse(emp.EmployeeID)
		if parseErr != nil {
			payrollInvalidID(c, "employee")
			return
		}
		perDate, perSize, convErr := emp.Parsed()
		if convErr != nil {
			respondPayrollError(c, convErr)
			return
		}
		input.Employees[employeeID] = service.CalculatePayrollInput{
			IncomeTax:              emp.IncomeTax,
			LocalIncomeTax:         emp.LocalIncomeTax,
			OtherDeductions:        emp.OtherDeductionLines(),
			WorkplaceSize:          perSize,
			IndustrialAccidentRate: emp.IndustrialAccidentRate,
			ContributionDate:       perDate,
		}
	}

	userID := appctx.GetUserID(c)
	result, err := h.service.CalculatePeriod(c.Request.Context(), appctx.GetCompanyID(c), id, input, &userID)
	if err != nil {
		respondPayrollError(c, err)
		return
	}

	// The skipped list is part of a successful answer: those payrolls were left
	// in draft on purpose (no 소득세 on file) and the operator has to see which.
	response.OK(c, gin.H{
		"period":    dto.FromPayrollPeriod(result.Period),
		"processed": result.Processed,
		"skipped":   result.Skipped,
	})
}

// ApprovePeriod handles POST /payroll-periods/:id/approve
func (h *PayrollHandler) ApprovePeriod(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "payroll period")
	if !ok {
		return
	}

	userID := appctx.GetUserID(c)
	period, err := h.service.ApprovePeriod(c.Request.Context(), appctx.GetCompanyID(c), id, &userID)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromPayrollPeriod(period))
}

// PayPeriod handles POST /payroll-periods/:id/pay
func (h *PayrollHandler) PayPeriod(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "payroll period")
	if !ok {
		return
	}

	var req dto.MarkPeriodPaidRequest
	// The body is optional: an empty payout request means "paid today".
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			payrollBindError(c, err)
			return
		}
	}

	paidOn, err := dto.ParseOptionalPayrollDate(req.PaymentDate)
	if err != nil {
		respondPayrollError(c, err)
		return
	}

	period, err := h.service.MarkPeriodPaid(c.Request.Context(), appctx.GetCompanyID(c), id, paidOn)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromPayrollPeriod(period))
}

// ClosePeriod handles POST /payroll-periods/:id/close
func (h *PayrollHandler) ClosePeriod(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "payroll period")
	if !ok {
		return
	}

	period, err := h.service.ClosePeriod(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromPayrollPeriod(period))
}

// ---------------------------------------------------------------------------
// Payroll records
// ---------------------------------------------------------------------------

// List handles GET /payrolls
func (h *PayrollHandler) List(c *gin.Context) {
	page, pageSize := parsePageParams(c, defaultPayrollPageSize)

	filter := &repository.PayrollFilter{
		CompanyID:  appctx.GetCompanyID(c),
		PeriodID:   payrollQueryUUID(c, "payroll_period_id"),
		EmployeeID: payrollQueryUUID(c, "employee_id"),
		PayYear:    payrollQueryInt(c, "pay_year"),
		PayMonth:   payrollQueryInt(c, "pay_month"),
		Search:     c.Query("search"),
		Page:       page,
		PageSize:   pageSize,
	}
	if status := c.Query("status"); status != "" {
		s := domain.PayrollStatus(status)
		filter.Status = &s
	}

	rows, total, err := h.service.ListPayrolls(c.Request.Context(), filter)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.Paginated(c, dto.FromPayrollListRows(rows), page, pageSize, total)
}

// Create handles POST /payrolls
func (h *PayrollHandler) Create(c *gin.Context) {
	var req dto.CreatePayrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		payrollBindError(c, err)
		return
	}

	employeeID, err := uuid.Parse(req.EmployeeID)
	if err != nil {
		payrollInvalidID(c, "employee")
		return
	}

	input := &service.CreatePayrollInput{
		EmployeeID: employeeID,
		PayYear:    req.PayYear,
		PayMonth:   req.PayMonth,
		PayrollEarningsInput: service.PayrollEarningsInput{
			Earnings:      req.Earnings.ToDomain(),
			WorkDays:      req.WorkDays,
			OvertimeHours: req.OvertimeHours,
			NightHours:    req.NightHours,
			HolidayHours:  req.HolidayHours,
			Notes:         req.Notes,
			BankCode:      req.BankCode,
		},
	}

	if req.PayrollPeriodID != "" {
		periodID, parseErr := uuid.Parse(req.PayrollPeriodID)
		if parseErr != nil {
			payrollInvalidID(c, "payroll period")
			return
		}
		input.PayrollPeriodID = &periodID
	}

	paymentDate, err := dto.ParseOptionalPayrollDate(req.PaymentDate)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	input.PaymentDate = paymentDate

	payroll, err := h.service.CreatePayroll(c.Request.Context(), appctx.GetCompanyID(c), input)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.Created(c, dto.FromPayroll(payroll))
}

// GetByID handles GET /payrolls/:id
func (h *PayrollHandler) GetByID(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "payroll")
	if !ok {
		return
	}

	payroll, err := h.service.GetPayroll(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromPayroll(payroll))
}

// Update handles PUT /payrolls/:id
func (h *PayrollHandler) Update(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "payroll")
	if !ok {
		return
	}

	var req dto.UpdatePayrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		payrollBindError(c, err)
		return
	}

	payroll, err := h.service.UpdatePayroll(c.Request.Context(), appctx.GetCompanyID(c), id, &service.PayrollEarningsInput{
		Earnings:      req.Earnings.ToDomain(),
		WorkDays:      req.WorkDays,
		OvertimeHours: req.OvertimeHours,
		NightHours:    req.NightHours,
		HolidayHours:  req.HolidayHours,
		Notes:         req.Notes,
		BankCode:      req.BankCode,
	})
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromPayroll(payroll))
}

// Delete handles DELETE /payrolls/:id
func (h *PayrollHandler) Delete(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "payroll")
	if !ok {
		return
	}

	if err := h.service.DeletePayroll(c.Request.Context(), appctx.GetCompanyID(c), id); err != nil {
		respondPayrollError(c, err)
		return
	}
	response.NoContent(c)
}

// Calculate handles POST /payrolls/:id/calculate
func (h *PayrollHandler) Calculate(c *gin.Context) {
	id, ok := payrollPathUUID(c, "id", "payroll")
	if !ok {
		return
	}

	var req dto.CalculatePayrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		payrollBindError(c, err)
		return
	}

	contributionDate, size, err := req.Parsed()
	if err != nil {
		respondPayrollError(c, err)
		return
	}

	payroll, err := h.service.CalculatePayroll(c.Request.Context(), appctx.GetCompanyID(c), id, &service.CalculatePayrollInput{
		IncomeTax:              req.IncomeTax,
		LocalIncomeTax:         req.LocalIncomeTax,
		OtherDeductions:        req.OtherDeductionLines(),
		WorkplaceSize:          size,
		IndustrialAccidentRate: req.IndustrialAccidentRate,
		ContributionDate:       contributionDate,
	})
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromPayroll(payroll))
}

// Preview handles POST /payrolls/preview
//
// It computes a payslip and stores nothing - not the payroll, not the item
// rows, not the 4대보험 register.
func (h *PayrollHandler) Preview(c *gin.Context) {
	var req dto.PayrollPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		payrollBindError(c, err)
		return
	}

	input, err := req.ToDomain()
	if err != nil {
		respondPayrollError(c, err)
		return
	}

	result, err := h.service.PreviewCalculation(c.Request.Context(), input)
	if err != nil {
		respondPayrollError(c, err)
		return
	}
	response.OK(c, dto.FromPayrollCalculation(result))
}
