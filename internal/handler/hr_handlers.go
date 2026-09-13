package handler

import (
	stderrors "errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/errors"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// HRHandlers groups the human-resources handlers.
//
// It exists so that internal/router/hr_routes.go can be wired with one field
// on handler.Handlers instead of four, and so that this package owns the
// repository and service construction for HR rather than spreading it.
type HRHandlers struct {
	Employee   *EmployeeHandler
	Department *DepartmentHandler
	Position   *PositionHandler
	Leave      *LeaveHandler
}

// NewHRHandlers builds the HR handlers with their services and repositories.
//
// db may be nil: nothing here issues a query at construction time, which is
// what lets the router tests build the whole route tree without a database.
func NewHRHandlers(db *gorm.DB, logger *zap.Logger) *HRHandlers {
	employeeRepo := repository.NewEmployeeRepositoryGorm(db)
	departmentRepo := repository.NewDepartmentRepositoryGorm(db)
	positionRepo := repository.NewPositionRepositoryGorm(db)
	leaveRepo := repository.NewLeaveRepositoryGorm(db)
	leaveTypeRepo := repository.NewLeaveTypeRepositoryGorm(db)

	// Field encryption is fail-closed. With no key the cipher refuses every
	// operation and the employee service rejects any request carrying a
	// resident number, rather than writing 주민등록번호 to the database in the
	// clear. The condition is logged loudly at startup because the feature is
	// then partially unavailable.
	cipher, err := service.NewResidentNumberCipherFromEnv()
	if logger != nil {
		switch {
		case err != nil:
			logger.Error("resident number encryption disabled: configured key is unusable",
				zap.String("env", service.ResidentNumberKeyEnv),
				zap.Error(err))
		case !cipher.Available():
			logger.Warn("resident number encryption disabled: no key configured; employee resident numbers will be refused",
				zap.String("env", service.ResidentNumberKeyEnv))
		}
	}

	employeeService := service.NewEmployeeService(employeeRepo, positionRepo, cipher)
	departmentService := service.NewDepartmentService(departmentRepo)
	positionService := service.NewPositionService(positionRepo)
	leaveService := service.NewLeaveService(leaveRepo, leaveTypeRepo, employeeRepo)

	return &HRHandlers{
		Employee:   NewEmployeeHandler(employeeService),
		Department: NewDepartmentHandler(departmentService),
		Position:   NewPositionHandler(positionService),
		Leave:      NewLeaveHandler(leaveService),
	}
}

// hrErrorMapping is one client-safe answer for one known error.
type hrErrorMapping struct {
	err     error
	status  int
	code    string
	message string
}

// hrErrorMappings turns the HR domain and service errors into responses.
//
// The table is explicit on purpose: err.Error() is never sent to the client,
// because anything that is not in this list may be a GORM or pgx error whose
// text carries table, column and constraint names - and, for this domain, the
// bind parameters of a statement that touched a resident number.
var hrErrorMappings = []hrErrorMapping{
	// ---- 404 ---------------------------------------------------------------
	{domain.ErrEmployeeNotFound, http.StatusNotFound, errors.CodeNotFound, "Employee not found"},
	// The department errors are declared twice, once in internal/domain and
	// once in internal/service, as two distinct values that happen to share a
	// message. errors.Is compares identity, not text, so both have to be
	// listed or half the paths fall through to a 500.
	{domain.ErrDepartmentNotFound, http.StatusNotFound, errors.CodeNotFound, "Department not found"},
	{service.ErrDepartmentNotFound, http.StatusNotFound, errors.CodeNotFound, "Department not found"},
	{domain.ErrPositionNotFound, http.StatusNotFound, errors.CodeNotFound, "Position not found"},
	{domain.ErrLeaveNotFound, http.StatusNotFound, errors.CodeNotFound, "Leave request not found"},
	{domain.ErrLeaveTypeNotFound, http.StatusNotFound, errors.CodeNotFound, "Leave type not found"},
	{domain.ErrLeaveBalanceNotFound, http.StatusNotFound, errors.CodeNotFound, "Leave balance not found"},

	// ---- 409 duplicates ----------------------------------------------------
	{domain.ErrEmployeeNoExists, http.StatusConflict, errors.CodeAlreadyExists, "Employee number already exists"},
	{domain.ErrDepartmentCodeExists, http.StatusConflict, errors.CodeAlreadyExists, "Department code already exists"},
	{service.ErrDepartmentCodeExists, http.StatusConflict, errors.CodeAlreadyExists, "Department code already exists"},
	{domain.ErrPositionCodeExists, http.StatusConflict, errors.CodeAlreadyExists, "Position code already exists"},
	{domain.ErrLeaveTypeCodeExists, http.StatusConflict, errors.CodeAlreadyExists, "Leave type code already exists"},

	// ---- 409 references ----------------------------------------------------
	{domain.ErrEmployeeHasHistory, http.StatusConflict, errors.CodeConflict,
		"Employee has payroll, insurance or leave history; record a resignation instead of deleting"},
	{service.ErrEmployeeHasReports, http.StatusConflict, errors.CodeConflict, "Employee still manages other employees"},
	{domain.ErrDepartmentHasChildren, http.StatusConflict, errors.CodeConflict,
		"Department has sub-departments; move or remove them first"},
	{service.ErrDepartmentHasChildren, http.StatusConflict, errors.CodeConflict,
		"Department has sub-departments; move or remove them first"},
	{service.ErrDepartmentHasTransactions, http.StatusConflict, errors.CodeConflict,
		"Department is referenced by voucher entries; deactivate it instead of deleting"},
	{domain.ErrPositionInUse, http.StatusConflict, errors.CodeConflict, "Position is assigned to employees"},
	{domain.ErrLeaveTypeInUse, http.StatusConflict, errors.CodeConflict, "Leave type is in use; deactivate it instead"},
	{domain.ErrLeaveOverlaps, http.StatusConflict, errors.CodeConflict, "Leave overlaps an existing request for this employee"},

	// ---- 422 state ---------------------------------------------------------
	{domain.ErrEmployeeInvalidTransition, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction, "Invalid employee status transition"},
	{domain.ErrLeaveInvalidTransition, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction, "Invalid leave status transition"},
	{domain.ErrLeaveNotPending, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction, "Only a pending leave request can be changed"},
	{domain.ErrLeaveInsufficient, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction, "Remaining leave balance is insufficient"},
	{domain.ErrLeaveTypeInactive, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction, "Leave type is not active"},
	{domain.ErrLeaveEmployeeInactive, http.StatusUnprocessableEntity, errors.CodeInvalidTransaction, "Leave cannot be recorded for a separated employee"},

	// ---- 400 input ---------------------------------------------------------
	{domain.ErrEmployeeSelfManager, http.StatusBadRequest, errors.CodeInvalidInput, "An employee cannot be their own manager"},
	{domain.ErrEmployeeManagerCycle, http.StatusBadRequest, errors.CodeInvalidInput, "Manager assignment creates a reporting cycle"},
	{domain.ErrEmployeeManagerNotFound, http.StatusBadRequest, errors.CodeInvalidInput, "Manager not found in this company"},
	// The same rule one level up the org chart: a department may not be its
	// own parent, nor a child of one of its own descendants. Both would make
	// every recursive walk over the hierarchy loop forever.
	{service.ErrDepartmentCircularRef, http.StatusBadRequest, errors.CodeInvalidInput,
		"A department cannot be its own parent or be moved beneath one of its own sub-departments"},
	{domain.ErrEmployeeDepartmentUnknown, http.StatusBadRequest, errors.CodeInvalidInput, "Department not found in this company"},
	{domain.ErrEmployeePositionUnknown, http.StatusBadRequest, errors.CodeInvalidInput, "Position not found in this company"},
	{domain.ErrEmployeeInvalidStatus, http.StatusBadRequest, errors.CodeInvalidInput, "Invalid employee status"},
	{domain.ErrEmployeeInvalidType, http.StatusBadRequest, errors.CodeInvalidInput, "Invalid employment type"},
	{domain.ErrEmployeeInvalidGender, http.StatusBadRequest, errors.CodeInvalidInput, "Invalid gender"},
	{domain.ErrEmployeeNoEmpty, http.StatusBadRequest, errors.CodeMissingField, "Employee number is required"},
	{domain.ErrEmployeeNameEmpty, http.StatusBadRequest, errors.CodeMissingField, "Employee name is required"},
	{domain.ErrEmployeeHireDateEmpty, http.StatusBadRequest, errors.CodeMissingField, "Hire date is required"},
	{domain.ErrEmployeeResignationDate, http.StatusBadRequest, errors.CodeMissingField, "A resignation date is required for a resigned or terminated employee"},
	{domain.ErrEmployeeResignedTooEarly, http.StatusBadRequest, errors.CodeInvalidInput, "Resignation date cannot precede the hire date"},
	{domain.ErrEmployeeContractDates, http.StatusBadRequest, errors.CodeInvalidInput, "Contract end date cannot precede the contract start date"},
	{domain.ErrPositionCodeEmpty, http.StatusBadRequest, errors.CodeMissingField, "Position code is required"},
	{domain.ErrPositionNameEmpty, http.StatusBadRequest, errors.CodeMissingField, "Position name is required"},
	{domain.ErrPositionSalaryRange, http.StatusBadRequest, errors.CodeInvalidInput, "Minimum salary cannot exceed maximum salary"},
	{domain.ErrPositionSalaryNegative, http.StatusBadRequest, errors.CodeInvalidInput, "Salary bounds cannot be negative"},
	{domain.ErrPositionRankNegative, http.StatusBadRequest, errors.CodeInvalidInput, "Rank level cannot be negative"},
	{domain.ErrLeaveTypeCodeEmpty, http.StatusBadRequest, errors.CodeMissingField, "Leave type code is required"},
	{domain.ErrLeaveTypeNameEmpty, http.StatusBadRequest, errors.CodeMissingField, "Leave type name is required"},
	{domain.ErrLeaveTypeDaysNegative, http.StatusBadRequest, errors.CodeInvalidInput, "Leave type day counts cannot be negative"},
	{domain.ErrLeaveDateRange, http.StatusBadRequest, errors.CodeInvalidInput, "Leave end date cannot precede the start date"},
	{domain.ErrLeaveDaysNotPositive, http.StatusBadRequest, errors.CodeInvalidInput, "Leave days must be greater than zero"},
	{domain.ErrLeaveDaysExceedRange, http.StatusBadRequest, errors.CodeInvalidInput, "Leave days exceed the requested date range"},
	{domain.ErrLeaveInvalidStatus, http.StatusBadRequest, errors.CodeInvalidInput, "Invalid leave status"},
	{domain.ErrLeaveBalanceNegative, http.StatusBadRequest, errors.CodeInvalidInput, "Leave balance days cannot be negative"},
	{domain.ErrLeaveBalanceFiscalYear, http.StatusBadRequest, errors.CodeInvalidInput, "Invalid fiscal year"},
	{service.ErrLeaveRejectionReason, http.StatusBadRequest, errors.CodeMissingField, "A rejection reason is required"},
	{dto.ErrInvalidHRDate, http.StatusBadRequest, errors.CodeInvalidFormat, "Dates must be formatted as YYYY-MM-DD"},

	// The message deliberately repeats nothing of the submitted value.
	{domain.ErrResidentNumberFormat, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid resident registration number"},

	// ---- 503 configuration -------------------------------------------------
	{service.ErrResidentNumberEncryptionUnavailable, http.StatusServiceUnavailable, errors.CodeUnavailable,
		"Resident number encryption is not configured; the resident number cannot be stored"},
}

// respondHRError writes the client-safe response for an HR error.
//
// Anything not in the table is a 500 whose cause is logged with the request id
// and never sent to the client.
func respondHRError(c *gin.Context, err error) {
	for _, mapping := range hrErrorMappings {
		if stderrors.Is(err, mapping.err) {
			// Client errors are answered without a log line; the 4xx body
			// already tells the caller everything they may know.
			response.Error(c, mapping.status, mapping.code, mapping.message)
			return
		}
	}
	response.InternalErrorLogged(c, "Internal server error", err)
}

// hrBindError answers a request body that failed binding.
//
// The validator's message names the offending field and rule, which is safe
// and useful; it is not a database error.
func hrBindError(c *gin.Context, err error) {
	response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Invalid request body", err.Error())
}

// hrInvalidID answers a path parameter that is not a UUID.
func hrInvalidID(c *gin.Context, what string) {
	response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid "+what+" ID")
}
