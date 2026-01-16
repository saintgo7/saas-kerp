package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

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
		Status:    "healthy",
		Version:   h.version,
		Timestamp: time.Now().UTC(),
	})
}

// Ready performs a readiness check (checks all dependencies)
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	services := make(map[string]string)
	healthy := true

	// Check database
	sqlDB, err := h.DB.DB()
	if err != nil {
		services["database"] = "unhealthy: " + err.Error()
		healthy = false
	} else if err := sqlDB.PingContext(ctx); err != nil {
		services["database"] = "unhealthy: " + err.Error()
		healthy = false
	} else {
		services["database"] = "healthy"
	}

	// Check Redis
	if h.Redis != nil {
		if err := h.Redis.Ping(ctx).Err(); err != nil {
			services["redis"] = "unhealthy: " + err.Error()
			healthy = false
		} else {
			services["redis"] = "healthy"
		}
	}

	status := "healthy"
	statusCode := http.StatusOK
	if !healthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response.Response{
		Success: healthy,
		Data: HealthStatus{
			Status:    status,
			Version:   h.version,
			Timestamp: time.Now().UTC(),
			Services:  services,
		},
		Meta: response.Meta{
			RequestID: c.GetString("request_id"),
			Timestamp: time.Now().UTC(),
		},
	})
}

// Live performs a liveness check (just confirms the service is running)
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}
