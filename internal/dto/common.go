package dto

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *MetaInfo   `json:"meta,omitempty"`
}

// ErrorInfo represents error details
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// MetaInfo represents metadata for paginated responses
type MetaInfo struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// SuccessResponse creates a success response
func SuccessResponse(data interface{}) Response {
	return Response{
		Success: true,
		Data:    data,
	}
}

// SuccessWithMeta creates a success response with metadata
func SuccessWithMeta(data interface{}, meta *MetaInfo) Response {
	return Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	}
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

// ErrorResponseWithDetails creates an error response with details
func ErrorResponseWithDetails(code, message, details string) Response {
	return Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// Common error codes
const (
	ErrCodeBadRequest          = "BAD_REQUEST"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeValidation          = "VALIDATION_ERROR"
	ErrCodeInternalServerError = "INTERNAL_SERVER_ERROR"
)

// SimpleErrorResponse is used for simple error responses without wrapper
type SimpleErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PaginatedResponse is used for paginated list responses
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalCount int64       `json:"total_count"`
	TotalPages int64       `json:"total_pages"`
}
