package response

import (
	stderrors "errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/errors"
)

// The response envelope is defined once, in internal/dto. These aliases keep the
// existing response.* call sites working while guaranteeing that every handler,
// whichever helper it reaches for, emits the identical JSON shape.
type (
	// Response is the standard API response structure.
	Response = dto.Response
	// ErrorBody contains error information.
	ErrorBody = dto.ErrorInfo
	// FieldError represents a validation error on a specific field.
	FieldError = dto.FieldError
	// Meta contains metadata about the response.
	Meta = dto.Meta
	// Pagination contains pagination information.
	Pagination = dto.Pagination
)

// buildMeta builds the response metadata
func buildMeta(c *gin.Context) *Meta {
	now := time.Now().UTC()
	return &Meta{
		RequestID: appctx.GetRequestID(c),
		Timestamp: &now,
	}
}

// OK sends a successful response with data
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    buildMeta(c),
	})
}

// Created sends a 201 response with data
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
		Meta:    buildMeta(c),
	})
}

// NoContent sends a 204 response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Accepted sends a 202 response with data
func Accepted(c *gin.Context, data interface{}) {
	c.JSON(http.StatusAccepted, Response{
		Success: true,
		Data:    data,
		Meta:    buildMeta(c),
	})
}

// Paginated sends a paginated response.
//
// perPage is clamped to [1, MaxPerPage] before it is used, so a client-supplied
// page size of 0 can never reach the division below.
func Paginated(c *gin.Context, data interface{}, page, perPage int, total int64) {
	page, perPage = ClampPagination(page, perPage)
	totalPages := TotalPages(total, perPage)

	meta := buildMeta(c)
	meta.Pagination = &Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Error sends an error response with the given status code
func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, Response{
		Success: false,
		Error: &ErrorBody{
			Code:    code,
			Message: message,
		},
		Meta: buildMeta(c),
	})
}

// ErrorWithDetails sends an error response with field-level details
func ErrorWithDetails(c *gin.Context, status int, code, message string, details []FieldError) {
	c.JSON(status, Response{
		Success: false,
		Error: &ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: buildMeta(c),
	})
}

// ErrorWithDetail sends an error response with a free-form detail string. Use it
// for binding/validation feedback, never for driver or database error text.
func ErrorWithDetail(c *gin.Context, status int, code, message, detail string) {
	c.JSON(status, Response{
		Success: false,
		Error: &ErrorBody{
			Code:    code,
			Message: message,
			Detail:  detail,
		},
		Meta: buildMeta(c),
	})
}

// ErrorWithPayload sends an error response that also carries a data payload,
// for endpoints whose failure body is still structured (readiness probes).
func ErrorWithPayload(c *gin.Context, status int, code, message string, data interface{}) {
	c.JSON(status, Response{
		Success: false,
		Data:    data,
		Error: &ErrorBody{
			Code:    code,
			Message: message,
		},
		Meta: buildMeta(c),
	})
}

// ErrorLogged writes a client-safe error response and logs the underlying cause
// with the request-scoped logger. Use it wherever the cause is a service,
// repository or driver error: those strings carry SQL fragments, table, column
// and constraint names, and sometimes parameter values.
func ErrorLogged(c *gin.Context, status int, code, message string, cause error) {
	if cause != nil {
		if logger := appctx.GetLogger(c); logger != nil {
			logger.Error(message,
				zap.Error(cause),
				zap.String("code", code),
				zap.Int("status", status),
				zap.String("route", c.FullPath()),
			)
		}
	}
	Error(c, status, code, message)
}

// InternalErrorLogged is ErrorLogged for the 500 case.
func InternalErrorLogged(c *gin.Context, message string, cause error) {
	ErrorLogged(c, http.StatusInternalServerError, errors.CodeInternal, message, cause)
}

// FromError sends an error response from an AppError.
//
// Errors that are not AppErrors are reported as a generic internal error: their
// text comes from GORM/pgx and leaks table, column and constraint names. Log the
// original with the request ID instead of shipping it to the client.
func FromError(c *gin.Context, err error) {
	var appErr *errors.AppError
	if !stderrors.As(err, &appErr) {
		appErr = errors.New(errors.CodeInternal, "Internal server error")
	}

	c.JSON(appErr.HTTPStatus(), Response{
		Success: false,
		Error: &ErrorBody{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
		Meta: buildMeta(c),
	})
}

// BadRequest sends a 400 response
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, errors.CodeValidation, message)
}

// Unauthorized sends a 401 response
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, errors.CodeUnauthorized, message)
}

// Forbidden sends a 403 response
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, errors.CodeForbidden, message)
}

// NotFound sends a 404 response
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, errors.CodeNotFound, message)
}

// Conflict sends a 409 response
func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, errors.CodeConflict, message)
}

// UnprocessableEntity sends a 422 response
func UnprocessableEntity(c *gin.Context, code, message string) {
	Error(c, http.StatusUnprocessableEntity, code, message)
}

// InternalError sends a 500 response
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, errors.CodeInternal, message)
}

// ValidationError sends a 400 response with validation details
func ValidationError(c *gin.Context, details []FieldError) {
	ErrorWithDetails(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", details)
}
