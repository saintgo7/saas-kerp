package dto

import (
	"time"

	"github.com/saintgo7/saas-kerp/internal/errors"
)

// ---------------------------------------------------------------------------
// Canonical API response envelope
// ---------------------------------------------------------------------------
//
// Every JSON response this API produces has this shape. There is exactly one
// envelope and exactly one error-code vocabulary (internal/errors/codes.go,
// "{CATEGORY}_{NUMBER}": AUTH_xxx, VAL_xxx, RES_xxx, PERM_xxx, SRV_xxx, BIZ_xxx).
//
//	{
//	  "success": true|false,
//	  "data":    <payload>,                       // omitted on errors
//	  "error":   {                                // omitted on success
//	    "code":    "VAL_001",
//	    "message": "human readable, safe to show",
//	    "detail":  "optional free-form detail",
//	    "details": [ { "field": "email", "message": "is required" } ]
//	  },
//	  "meta": {
//	    "request_id": "…",
//	    "timestamp":  "2026-09-07T12:34:56Z",
//	    "pagination": { "page": 1, "per_page": 20, "total": 42, "total_pages": 3 }
//	  }
//	}
//
// Clients must read the error message from `error.message`, never from a
// top-level `message` field: there is none.
//
// internal/handler/response aliases these types, so the two historical envelopes
// in this codebase are now the same struct.

// Response is the standard API response envelope.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorInfo carries the machine-readable code and a client-safe message.
// It never carries raw driver or database error text.
type ErrorInfo struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Detail  string       `json:"detail,omitempty"`
	Details []FieldError `json:"details,omitempty"`
}

// FieldError describes a validation failure on one request field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Meta carries per-response metadata. Every field is optional so a response
// that only has pagination to report does not emit an empty request id and a
// zero timestamp.
type Meta struct {
	RequestID  string      `json:"request_id,omitempty"`
	Timestamp  *time.Time  `json:"timestamp,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// Pagination describes the page of a list response.
type Pagination struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// MetaInfo is the pagination input accepted by SuccessWithMeta. It is not a wire
// type: SuccessWithMeta converts it into Meta.Pagination.
type MetaInfo struct {
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// SuccessResponse creates a success response
func SuccessResponse(data interface{}) Response {
	return Response{
		Success: true,
		Data:    data,
	}
}

// SuccessWithMeta creates a success response carrying pagination metadata.
func SuccessWithMeta(data interface{}, meta *MetaInfo) Response {
	resp := Response{
		Success: true,
		Data:    data,
	}
	if meta != nil {
		resp.Meta = &Meta{
			Pagination: &Pagination{
				Page:       meta.Page,
				PerPage:    meta.PageSize,
				Total:      meta.Total,
				TotalPages: meta.TotalPages,
			},
		}
	}
	return resp
}

// ErrorResponse creates an error response
func ErrorResponse(code, message string) Response {
	return Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}
}

// ErrorResponseWithDetails creates an error response with a free-form detail.
func ErrorResponseWithDetails(code, message, detail string) Response {
	return Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Detail:  detail,
		},
	}
}

// Common error codes.
//
// These are aliases of the canonical vocabulary in internal/errors/codes.go.
// They used to be a second, incompatible set ("VALIDATION_ERROR", "NOT_FOUND",
// ...), which meant the same logical failure reached the client under two
// different codes depending on which handler produced it.
const (
	ErrCodeBadRequest          = errors.CodeValidation
	ErrCodeUnauthorized        = errors.CodeUnauthorized
	ErrCodeForbidden           = errors.CodeForbidden
	ErrCodeNotFound            = errors.CodeNotFound
	ErrCodeConflict            = errors.CodeConflict
	ErrCodeValidation          = errors.CodeValidation
	ErrCodeInternalServerError = errors.CodeInternal
)
