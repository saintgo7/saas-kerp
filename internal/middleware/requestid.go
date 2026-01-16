package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
)

const (
	// HeaderRequestID is the header name for request ID
	HeaderRequestID = "X-Request-ID"
)

// RequestID generates or extracts a unique request ID for each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get request ID from header
		requestID := c.GetHeader(HeaderRequestID)

		// Generate new one if not provided
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set in context and response header
		appctx.SetRequestID(c, requestID)
		c.Header(HeaderRequestID, requestID)

		c.Next()
	}
}
