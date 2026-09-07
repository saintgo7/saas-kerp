package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saintgo7/saas-kerp/internal/errors"
)

// DefaultMaxRequestBody is the fallback body cap: 1 MiB.
const DefaultMaxRequestBody int64 = 1 << 20

// BodyLimit caps the number of bytes any handler will read from a request body.
//
// Every handler binds JSON with c.ShouldBindJSON, which reads the whole body
// into memory before validating anything. Without a cap, a single unauthenticated
// POST to /auth/login with a multi-hundred-megabyte body, or a POST /vouchers
// carrying a million entries, is enough to exhaust the process. ReadTimeout does
// not help: it bounds how long a read may take, not how much it may return.
//
// Requests that declare an oversized Content-Length are rejected before the body
// is read at all; chunked bodies are truncated by MaxBytesReader, which makes the
// subsequent bind fail with 400.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxRequestBody
	}

	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			abortWithError(c, http.StatusRequestEntityTooLarge, errors.CodeValidation, "Request body too large")
			return
		}

		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}

		c.Next()
	}
}
