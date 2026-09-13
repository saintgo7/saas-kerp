package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/errors"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	*BaseHandler
	version string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *gorm.DB, redis *redis.Client, logger *zap.Logger, version string) *HealthHandler {
	return &HealthHandler{
		BaseHandler: NewBaseHandler(db, redis, logger),
		version:     version,
	}
}

// Service status strings. They are intentionally coarse: this endpoint is
// unauthenticated.
const (
	statusHealthy   = "healthy"
	statusUnhealthy = "unhealthy"
)

// HealthStatus represents the health check response
type HealthStatus struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services,omitempty"`
}

// Check performs a basic health check
func (h *HealthHandler) Check(c *gin.Context) {
	response.OK(c, HealthStatus{
		Status:    statusHealthy,
		Version:   h.version,
		Timestamp: time.Now().UTC(),
	})
}

// Ready performs a readiness check (checks all dependencies).
//
// The response deliberately carries no error text. This endpoint is
// unauthenticated, and the connection errors it would otherwise echo contain the
// database host, port, database name and account name. The cause is logged
// instead, with the request ID, so operators can still correlate the two.
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	logger := h.GetLogger(c)
	services := make(map[string]string)
	healthy := true

	// Check database
	sqlDB, err := h.DB.DB()
	if err == nil {
		err = sqlDB.PingContext(ctx)
	}
	if err != nil {
		if logger != nil {
			logger.Error("readiness: database unhealthy", zap.Error(err))
		}
		services["database"] = statusUnhealthy
		healthy = false
	} else {
		services["database"] = statusHealthy
	}

	// Check Redis
	if h.Redis != nil {
		if err := h.Redis.Ping(ctx).Err(); err != nil {
			if logger != nil {
				logger.Error("readiness: redis unhealthy", zap.Error(err))
			}
			services["redis"] = statusUnhealthy
			healthy = false
		} else {
			services["redis"] = statusHealthy
		}
	}

	status := statusHealthy
	statusCode := http.StatusOK
	if !healthy {
		status = statusUnhealthy
		statusCode = http.StatusServiceUnavailable
	}

	payload := HealthStatus{
		Status:    status,
		Version:   h.version,
		Timestamp: time.Now().UTC(),
		Services:  services,
	}

	if healthy {
		response.OK(c, payload)
		return
	}
	response.ErrorWithPayload(c, statusCode, errors.CodeUnavailable, "Service dependencies are unhealthy", payload)
}

// Live performs a liveness check (just confirms the service is running).
func (h *HealthHandler) Live(c *gin.Context) {
	response.OK(c, gin.H{"status": "alive"})
}
