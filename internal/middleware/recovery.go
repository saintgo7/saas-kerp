package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
)

// Recovery middleware recovers from panics and logs the error
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get stack trace
				stack := debug.Stack()

				// Get request-scoped logger or use default
				reqLogger := appctx.GetLogger(c)
				if reqLogger == nil {
					reqLogger = logger
				}

				// Log the panic
				reqLogger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("stack", string(stack)),
					zap.String("request_id", appctx.GetRequestID(c)),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)

				// Return 500 error
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "SRV_001",
						"message": "Internal server error",
					},
					"meta": gin.H{
						"request_id": appctx.GetRequestID(c),
					},
				})
			}
		}()

		c.Next()
	}
}
