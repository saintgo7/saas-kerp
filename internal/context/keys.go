package context

// Context keys for storing values in request context
const (
	// Request metadata
	KeyRequestID  = "request_id"
	KeyStartTime  = "start_time"
	KeyClientIP   = "client_ip"
	KeyUserAgent  = "user_agent"

	// Authentication
	KeyUserID    = "user_id"
	KeyCompanyID = "company_id"
	KeyEmail     = "email"
	KeyUserName  = "user_name"
	KeyRoles     = "roles"

	// Logging
	KeyLogger = "logger"

	// Database
	KeyDB    = "db"
	KeyTx    = "tx"
	KeyRedis = "redis"
)
