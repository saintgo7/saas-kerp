package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError represents an application-level error with code and message
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// HTTPStatus returns the HTTP status code for this error
func (e *AppError) HTTPStatus() int {
	return GetHTTPStatus(e.Code)
}

// New creates a new AppError
func New(code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Wrap wraps an existing error with an AppError
func Wrap(code, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

// Newf creates a new AppError with formatted message
func Newf(code, format string, args ...interface{}) *AppError {
	return &AppError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Wrapf wraps an error with formatted message
func Wrapf(code string, err error, format string, args ...interface{}) *AppError {
	return &AppError{Code: code, Message: fmt.Sprintf(format, args...), Err: err}
}

// Is checks if an error matches an AppError by code
func Is(err error, target *AppError) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == target.Code
	}
	return false
}

// Code extracts the error code from an error
func Code(err error) string {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return CodeInternal
}

// StatusCode returns the HTTP status code for an error
func StatusCode(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.HTTPStatus()
	}
	return http.StatusInternalServerError
}

// Predefined errors for common cases
var (
	// Authentication
	ErrUnauthorized       = New(CodeUnauthorized, "Authentication required")
	ErrTokenExpired       = New(CodeTokenExpired, "Token has expired")
	ErrInvalidCredentials = New(CodeInvalidCredentials, "Invalid email or password")
	ErrAccountLocked      = New(CodeAccountLocked, "Account is locked")
	ErrAccountInactive    = New(CodeAccountInactive, "Account is inactive")
	ErrTokenInvalid       = New(CodeTokenInvalid, "Invalid token")
	ErrRefreshTokenInvalid = New(CodeRefreshTokenInvalid, "Invalid refresh token")

	// Validation
	ErrValidation   = New(CodeValidation, "Validation failed")
	ErrInvalidInput = New(CodeInvalidInput, "Invalid input")
	ErrMissingField = New(CodeMissingField, "Required field is missing")

	// Resource
	ErrNotFound      = New(CodeNotFound, "Resource not found")
	ErrAlreadyExists = New(CodeAlreadyExists, "Resource already exists")
	ErrConflict      = New(CodeConflict, "Resource conflict")
	ErrEmailExists   = New(CodeEmailExists, "Email already exists")

	// Permission
	ErrForbidden        = New(CodeForbidden, "Access denied")
	ErrInsufficientRole = New(CodeInsufficientRole, "Insufficient permissions")
	ErrTenantMismatch   = New(CodeTenantMismatch, "Tenant mismatch")

	// Server
	ErrInternal        = New(CodeInternal, "Internal server error")
	ErrDatabase        = New(CodeDatabase, "Database error")
	ErrExternalService = New(CodeExternalService, "External service error")

	// Business
	ErrVoucherUnbalanced   = New(CodeVoucherUnbalanced, "Voucher debit and credit must be equal")
	ErrPeriodClosed        = New(CodePeriodClosed, "Accounting period is closed")
	ErrInsufficientBalance = New(CodeInsufficientBalance, "Insufficient balance")
)

// FieldError represents a validation error for a specific field
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError represents multiple field validation errors
type ValidationError struct {
	AppError
	Fields []FieldError `json:"fields,omitempty"`
}

// NewValidationError creates a validation error with field details
func NewValidationError(fields []FieldError) *ValidationError {
	return &ValidationError{
		AppError: *New(CodeValidation, "Validation failed"),
		Fields:   fields,
	}
}

// AddField adds a field error
func (e *ValidationError) AddField(field, message string) {
	e.Fields = append(e.Fields, FieldError{Field: field, Message: message})
}

// HasErrors returns true if there are validation errors
func (e *ValidationError) HasErrors() bool {
	return len(e.Fields) > 0
}
