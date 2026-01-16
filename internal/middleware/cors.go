package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/saintgo7/saas-kerp/internal/config"
)

// CORS middleware handles Cross-Origin Resource Sharing
func CORS(cfg *config.CORSConfig) gin.HandlerFunc {
	// Pre-compute allowed origins set for faster lookup
	allowedOriginsSet := make(map[string]bool)
	allowAll := false
	for _, origin := range cfg.AllowedOrigins {
		if origin == "*" {
			allowAll = true
			break
		}
		allowedOriginsSet[origin] = true
	}

	// Pre-compute header values
	methodsHeader := strings.Join(cfg.AllowedMethods, ", ")
	headersHeader := strings.Join(cfg.AllowedHeaders, ", ")
	maxAgeHeader := strconv.Itoa(cfg.MaxAge)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Check if origin is allowed
		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if allowedOriginsSet[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}

		// Set CORS headers
		c.Header("Access-Control-Allow-Methods", methodsHeader)
		c.Header("Access-Control-Allow-Headers", headersHeader)
		c.Header("Access-Control-Max-Age", maxAgeHeader)
		c.Header("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// DefaultCORSConfig returns a default CORS configuration for development
func DefaultCORSConfig() *config.CORSConfig {
	return &config.CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type", "X-Request-ID"},
		MaxAge:         86400,
	}
}
