package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newBodyLimitRouter(limit int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(BodyLimit(limit))
	r.POST("/echo", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		c.String(http.StatusOK, "%d", len(body))
	})
	return r
}

func TestBodyLimit_RejectsDeclaredOversizedBody(t *testing.T) {
	r := newBodyLimitRouter(1024)

	req := httptest.NewRequest("POST", "/echo", bytes.NewReader(make([]byte, 4096)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}
	if strings.Contains(w.Body.String(), "\"success\":true") {
		t.Error("the rejection body claims success")
	}
}

func TestBodyLimit_AllowsBodiesWithinTheLimit(t *testing.T) {
	r := newBodyLimitRouter(1024)

	req := httptest.NewRequest("POST", "/echo", bytes.NewReader(make([]byte, 512)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	if w.Body.String() != "512" {
		t.Errorf("handler read %s bytes, want 512", w.Body.String())
	}
}

// A body sent without a Content-Length (chunked) is truncated by MaxBytesReader,
// which makes the handler's read fail rather than letting it buffer without
// bound.
func TestBodyLimit_TruncatesUndeclaredOversizedBody(t *testing.T) {
	r := newBodyLimitRouter(64)

	req := httptest.NewRequest("POST", "/echo", bytes.NewReader(make([]byte, 4096)))
	req.ContentLength = -1
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK && w.Body.String() != "64" {
		t.Fatalf("an undeclared oversized body was accepted whole: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestBodyLimit_ZeroUsesTheDefault(t *testing.T) {
	r := newBodyLimitRouter(0)

	req := httptest.NewRequest("POST", "/echo", bytes.NewReader(make([]byte, 1024)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: a zero limit must fall back to the default, not block everything", w.Code)
	}
}
