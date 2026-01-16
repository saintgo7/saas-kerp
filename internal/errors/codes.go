package errors

// Error codes follow the pattern: {CATEGORY}_{NUMBER}
// Categories:
//   AUTH - Authentication/Authorization
//   VAL  - Validation
//   RES  - Resource
//   PERM - Permission
//   SRV  - Server/Internal

const (
	// Authentication errors (AUTH_)
	CodeUnauthorized       = "AUTH_001"
	CodeTokenExpired       = "AUTH_002"
	CodeInvalidCredentials = "AUTH_003"
	CodeAccountLocked      = "AUTH_004"
	CodeAccountInactive    = "AUTH_005"
	CodeTokenInvalid       = "AUTH_006"
	CodeRefreshTokenInvalid = "AUTH_007"
	CodeMFARequired        = "AUTH_008"
	CodeMFAInvalid         = "AUTH_009"

	// Validation errors (VAL_)
	CodeValidation    = "VAL_001"
	CodeInvalidInput  = "VAL_002"
	CodeMissingField  = "VAL_003"
	CodeInvalidFormat = "VAL_004"
	CodeOutOfRange    = "VAL_005"

	// Resource errors (RES_)
	CodeNotFound      = "RES_001"
	CodeAlreadyExists = "RES_002"
	CodeConflict      = "RES_003"
	CodeEmailExists   = "RES_004"
	CodeBusinessNumberExists = "RES_005"

	// Permission errors (PERM_)
	CodeForbidden        = "PERM_001"
	CodeInsufficientRole = "PERM_002"
	CodeTenantMismatch   = "PERM_003"

	// Server errors (SRV_)
	CodeInternal        = "SRV_001"
	CodeDatabase        = "SRV_002"
	CodeExternalService = "SRV_003"
	CodeTimeout         = "SRV_004"
	CodeUnavailable     = "SRV_005"

	// Business logic errors (BIZ_)
	CodeVoucherUnbalanced   = "BIZ_001"
	CodePeriodClosed        = "BIZ_002"
	CodeInsufficientBalance = "BIZ_003"
	CodeInvalidTransaction  = "BIZ_004"
)

// HTTP status code mapping
var HTTPStatusCodes = map[string]int{
	CodeUnauthorized:       401,
	CodeTokenExpired:       401,
	CodeInvalidCredentials: 401,
	CodeAccountLocked:      403,
	CodeAccountInactive:    403,
	CodeTokenInvalid:       401,
	CodeRefreshTokenInvalid: 401,
	CodeMFARequired:        401,
	CodeMFAInvalid:         401,

	CodeValidation:    400,
	CodeInvalidInput:  400,
	CodeMissingField:  400,
	CodeInvalidFormat: 400,
	CodeOutOfRange:    400,

	CodeNotFound:      404,
	CodeAlreadyExists: 409,
	CodeConflict:      409,
	CodeEmailExists:   409,
	CodeBusinessNumberExists: 409,

	CodeForbidden:        403,
	CodeInsufficientRole: 403,
	CodeTenantMismatch:   403,

	CodeInternal:        500,
	CodeDatabase:        500,
	CodeExternalService: 502,
	CodeTimeout:         504,
	CodeUnavailable:     503,

	CodeVoucherUnbalanced:   422,
	CodePeriodClosed:        422,
	CodeInsufficientBalance: 422,
	CodeInvalidTransaction:  422,
}

// GetHTTPStatus returns the HTTP status code for an error code
func GetHTTPStatus(code string) int {
	if status, ok := HTTPStatusCodes[code]; ok {
		return status
	}
	return 500
}
