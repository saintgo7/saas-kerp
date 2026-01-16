package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
)

// Logger middleware logs request and response information
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Set start time in context
		appctx.SetStartTime(c, start)

		// Create request-scoped logger with request ID
		reqLogger := logger.With(
			zap.String("request_id", appctx.GetRequestID(c)),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
		)
		appctx.SetLogger(c, reqLogger)

		// Log request start
		reqLogger.Debug("request started",
			zap.String("query", query),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		status := c.Writer.Status()

		// Build log fields
		fields := []zap.Field{
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.Int("body_size", c.Writer.Size()),
		}

		// Add user info if authenticated
		if userID := appctx.GetUserID(c); userID.String() != "00000000-0000-0000-0000-000000000000" {
			fields = append(fields, zap.String("user_id", userID.String()))
		}

		// Add error if present
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		// Log based on status code
		switch {
		case status >= 500:
			reqLogger.Error("request completed", fields...)
		case status >= 400:
			reqLogger.Warn("request completed", fields...)
		default:
			reqLogger.Info("request completed", fields...)
		}
	}
}

// SkipLogging returns true if the path should skip logging
func SkipLogging(path string) bool {
	skipPaths := map[string]bool{
		"/health":       true,
		"/health/ready": true,
		"/health/live":  true,
		"/metrics":      true,
		"/favicon.ico":  true,
	}
	return skipPaths[path]
}
