package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/config"
	appctx "github.com/saintgo7/saas-kerp/internal/context"
)

// maxTrackedKeys caps the number of live token buckets. Without a cap, one
// client cycling through keys (many source IPs, many login emails) grows the map
// without bound between cleanup passes.
const maxTrackedKeys = 50000

// RateLimiter implements a simple in-memory rate limiter using token bucket algorithm
type RateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	rate      int           // tokens per second
	burst     int           // max tokens
	cleanup   time.Duration // cleanup interval
	lastClean time.Time
}

type bucket struct {
	tokens    float64
	lastCheck time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate, burst int) *RateLimiter {
	if rate < 1 {
		rate = 1
	}
	if burst < 1 {
		burst = 1
	}
	return &RateLimiter{
		buckets:   make(map[string]*bucket),
		rate:      rate,
		burst:     burst,
		cleanup:   time.Minute,
		lastClean: time.Now(),
	}
}

// Allow checks if a request from the given key should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Periodic cleanup of old entries
	if now.Sub(rl.lastClean) > rl.cleanup {
		rl.cleanupOldBuckets(now)
		rl.lastClean = now
	}

	b, exists := rl.buckets[key]
	if !exists {
		if len(rl.buckets) >= maxTrackedKeys {
			// The table is saturated. Force a sweep; if that does not free room,
			// refuse rather than growing memory without bound.
			rl.cleanupOldBuckets(now)
			if len(rl.buckets) >= maxTrackedKeys {
				return false
			}
		}
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

// cleanupOldBuckets removes buckets that have refilled to full and are therefore
// indistinguishable from a fresh one.
func (rl *RateLimiter) cleanupOldBuckets(now time.Time) {
	threshold := now.Add(-10 * time.Minute)
	for key, b := range rl.buckets {
		if b.lastCheck.Before(threshold) {
			delete(rl.buckets, key)
		}
	}
}

// RateLimit middleware applies rate limiting based on client IP.
//
// This runs at engine level, i.e. before any Auth middleware, so the identity is
// necessarily the network peer. c.ClientIP() only reflects X-Forwarded-For for
// peers listed in app.trusted_proxies (see internal/router/router.go); without
// that setting gin trusts every peer and the header can be forged.
func RateLimit(cfg *config.RateLimitConfig) gin.HandlerFunc {
	limiter := NewRateLimiter(cfg.RequestsPerSecond, cfg.Burst)

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		if !limiter.Allow(c.ClientIP()) {
			abortRateLimited(c)
			return
		}

		c.Next()
	}
}

// RateLimitByKey middleware applies rate limiting with a custom key function.
func RateLimitByKey(cfg *config.RateLimitConfig, rate, burst int, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, burst)

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		if !limiter.Allow(keyFunc(c)) {
			abortRateLimited(c)
			return
		}

		c.Next()
	}
}

// RateLimitAuthenticated limits per authenticated user. It only has an effect
// when registered *after* Auth; registered before it, the user ID is always
// uuid.Nil and every request shares one bucket.
func RateLimitAuthenticated(cfg *config.RateLimitConfig, rate, burst int) gin.HandlerFunc {
	return RateLimitByKey(cfg, rate, burst, func(c *gin.Context) string {
		if userID := appctx.GetUserID(c); userID != uuid.Nil {
			return "user:" + userID.String()
		}
		return "ip:" + c.ClientIP()
	})
}

// RateLimitCredentials applies a much tighter limit to the unauthenticated
// credential endpoints (login, register, forgot-password, refresh) than the
// global limiter does, keyed by client IP.
func RateLimitCredentials(cfg *config.RateLimitConfig, rate, burst int) gin.HandlerFunc {
	return RateLimitByKey(cfg, rate, burst, func(c *gin.Context) string {
		return "auth:" + c.ClientIP()
	})
}

func abortRateLimited(c *gin.Context) {
	c.Header("Retry-After", "1")
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "RATE_001",
			"message": "Rate limit exceeded",
		},
		"meta": gin.H{
			"request_id": appctx.GetRequestID(c),
			"timestamp":  time.Now().UTC(),
		},
	})
}
