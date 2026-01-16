package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/config"
)

// RateLimiter implements a simple in-memory rate limiter using token bucket algorithm
type RateLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*bucket
	rate     int           // tokens per second
	burst    int           // max tokens
	cleanup  time.Duration // cleanup interval
	lastClean time.Time
}

type bucket struct {
	tokens    float64
	lastCheck time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate, burst int) *RateLimiter {
	return &RateLimiter{
		buckets:   make(map[string]*bucket),
		rate:      rate,
		burst:     burst,
		cleanup:   5 * time.Minute,
		lastClean: time.Now(),
	}
}

// Allow checks if a request from the given key should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Periodic cleanup of old entries
	if time.Since(rl.lastClean) > rl.cleanup {
		rl.cleanupOldBuckets()
		rl.lastClean = time.Now()
	}

	now := time.Now()

	b, exists := rl.buckets[key]
	if !exists {
		rl.buckets[key] = &bucket{
			tokens:    float64(rl.burst) - 1,
			lastCheck: now,
		}
		return true
	}

	// Add tokens based on time elapsed
	elapsed := now.Sub(b.lastCheck).Seconds()
	b.tokens += elapsed * float64(rl.rate)
	if b.tokens > float64(rl.burst) {
		b.tokens = float64(rl.burst)
	}
	b.lastCheck = now

	// Check if we have tokens
	if b.tokens >= 1 {
		b.tokens--
		return true
	}

	return false
}

// cleanupOldBuckets removes buckets that haven't been used recently
func (rl *RateLimiter) cleanupOldBuckets() {
	threshold := time.Now().Add(-10 * time.Minute)
	for key, b := range rl.buckets {
		if b.lastCheck.Before(threshold) {
			delete(rl.buckets, key)
		}
	}
}

// RateLimit middleware applies rate limiting based on client IP
func RateLimit(cfg *config.RateLimitConfig) gin.HandlerFunc {
	limiter := NewRateLimiter(cfg.RequestsPerSecond, cfg.Burst)

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		// Use client IP as the rate limit key
		key := c.ClientIP()

		// Add user ID if authenticated for per-user rate limiting
		if userID := appctx.GetUserID(c); userID.String() != "00000000-0000-0000-0000-000000000000" {
			key = userID.String()
		}

		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_001",
					"message": "Rate limit exceeded",
				},
				"meta": gin.H{
					"request_id": appctx.GetRequestID(c),
					"retry_after": 1,
				},
			})
			return
		}

		c.Next()
	}
}

// RateLimitByKey middleware applies rate limiting with a custom key function
func RateLimitByKey(cfg *config.RateLimitConfig, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	limiter := NewRateLimiter(cfg.RequestsPerSecond, cfg.Burst)

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		key := keyFunc(c)
		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_001",
					"message": "Rate limit exceeded",
				},
				"meta": gin.H{
					"request_id": appctx.GetRequestID(c),
					"retry_after": 1,
				},
			})
			return
		}

		c.Next()
	}
}
