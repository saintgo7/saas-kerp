package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/errors"
)

// Response is the standard API response structure
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorBody  `json:"error,omitempty"`
	Meta    Meta        `json:"meta"`
}

// ErrorBody contains error information
type ErrorBody struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

// FieldError represents a validation error on a specific field
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Meta contains metadata about the response
type Meta struct {
	RequestID  string      `json:"request_id"`
	Timestamp  time.Time   `json:"timestamp"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// Pagination contains pagination information
type Pagination struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// buildMeta builds the response metadata
func buildMeta(c *gin.Context) Meta {
	return Meta{
		RequestID: appctx.GetRequestID(c),
		Timestamp: time.Now().UTC(),
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

// Paginated sends a paginated response
func Paginated(c *gin.Context, data interface{}, page, perPage int, total int64) {
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

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

// FromError sends an error response from an AppError
func FromError(c *gin.Context, err error) {
	var appErr *errors.AppError
	if e, ok := err.(*errors.AppError); ok {
		appErr = e
	} else {
		appErr = errors.Wrap(errors.CodeInternal, err.Error(), err)
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
