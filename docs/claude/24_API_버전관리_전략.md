# 24. API 버전관리 전략

## 목차
1. [개요](#1-개요)
2. [버전관리 방식](#2-버전관리-방식)
3. [API 버전 정책](#3-api-버전-정책)
4. [하위 호환성 관리](#4-하위-호환성-관리)
5. [버전 마이그레이션](#5-버전-마이그레이션)
6. [API 변경 프로세스](#6-api-변경-프로세스)
7. [클라이언트 SDK 관리](#7-클라이언트-sdk-관리)
8. [문서화 및 커뮤니케이션](#8-문서화-및-커뮤니케이션)

---

## 1. 개요

### 1.1 버전관리 필요성

```yaml
api_versioning_goals:
  backward_compatibility:
    description: "기존 클라이언트 영향 최소화"
    importance: "프로덕션 시스템 안정성 유지"

  evolution:
    description: "API 지속적 개선 가능"
    importance: "새로운 기능 및 최적화 반영"

  clarity:
    description: "변경 사항 명확한 추적"
    importance: "개발자 경험 향상"

  deprecation:
    description: "구버전 점진적 제거"
    importance: "기술 부채 관리"

versioning_scope:
  rest_api: "HTTP REST API"
  grpc_api: "gRPC 내부 통신"
  webhook: "외부 이벤트 통지"
  sdk: "클라이언트 SDK"
```

### 1.2 버전관리 전략 개요

```
┌─────────────────────────────────────────────────────────────────┐
│                    API Version Lifecycle                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   Alpha     Beta      GA        Deprecated    Sunset           │
│     │         │        │             │           │              │
│     ▼         ▼        ▼             ▼           ▼              │
│  ┌─────┐  ┌─────┐  ┌─────┐     ┌─────────┐  ┌───────┐          │
│  │v2.0α│→ │v2.0β│→ │v2.0 │────→│v2.0 dep │→ │Removed│          │
│  └─────┘  └─────┘  └─────┘     └─────────┘  └───────┘          │
│     │         │        │             │                          │
│  Internal  Limited   Full      12 months    End of Life        │
│   Only     Partners  Public    Warning                          │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  Current Versions:                                              │
│  - v1 (GA) - Production stable                                  │
│  - v2 (Beta) - New features preview                            │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 버전관리 방식

### 2.1 URL Path 버전관리

```go
// internal/handler/router.go
package handler

import (
    "github.com/gin-gonic/gin"
)

// APIVersionConfig defines version-specific configuration
type APIVersionConfig struct {
    Version           string
    Deprecated        bool
    DeprecatedMessage string
    SunsetDate        string
}

func SetupRouter(
    v1Handlers *V1Handlers,
    v2Handlers *V2Handlers,
) *gin.Engine {
    r := gin.New()

    // Version 1 routes (stable)
    v1 := r.Group("/api/v1")
    {
        v1.Use(VersionMiddleware("v1", false, ""))
        setupV1Routes(v1, v1Handlers)
    }

    // Version 2 routes (beta - new features)
    v2 := r.Group("/api/v2")
    {
        v2.Use(VersionMiddleware("v2", false, ""))
        setupV2Routes(v2, v2Handlers)
    }

    // Deprecated version (will be removed)
    // v0 := r.Group("/api/v0")
    // {
    //     v0.Use(VersionMiddleware("v0", true, "v0 will be removed on 2024-06-01. Please migrate to v1."))
    //     setupV0Routes(v0, v0Handlers)
    // }

    return r
}

func setupV1Routes(rg *gin.RouterGroup, h *V1Handlers) {
    // Voucher endpoints
    vouchers := rg.Group("/vouchers")
    {
        vouchers.GET("", h.Voucher.List)
        vouchers.POST("", h.Voucher.Create)
        vouchers.GET("/:id", h.Voucher.Get)
        vouchers.PUT("/:id", h.Voucher.Update)
        vouchers.DELETE("/:id", h.Voucher.Delete)
        vouchers.POST("/:id/post", h.Voucher.Post)
    }

    // Account endpoints
    accounts := rg.Group("/accounts")
    {
        accounts.GET("", h.Account.List)
        accounts.GET("/:code", h.Account.Get)
    }
}

func setupV2Routes(rg *gin.RouterGroup, h *V2Handlers) {
    // V2 has enhanced voucher endpoints
    vouchers := rg.Group("/vouchers")
    {
        vouchers.GET("", h.Voucher.List)          // Enhanced filtering
        vouchers.POST("", h.Voucher.Create)        // Supports batch
        vouchers.GET("/:id", h.Voucher.Get)        // Includes audit trail
        vouchers.PUT("/:id", h.Voucher.Update)
        vouchers.DELETE("/:id", h.Voucher.Delete)
        vouchers.POST("/:id/post", h.Voucher.Post)
        vouchers.POST("/bulk", h.Voucher.BulkCreate) // New in v2
        vouchers.GET("/:id/history", h.Voucher.History) // New in v2
    }
}
```

### 2.2 버전 미들웨어

```go
// internal/middleware/version.go
package middleware

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
)

// VersionMiddleware adds version information to response headers
func VersionMiddleware(version string, deprecated bool, deprecationMessage string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Set version header
        c.Header("X-API-Version", version)

        // Handle deprecated versions
        if deprecated {
            c.Header("X-API-Deprecated", "true")
            c.Header("X-API-Deprecation-Message", deprecationMessage)

            // Log deprecated API usage for monitoring
            logDeprecatedUsage(c, version)
        }

        // Set API stability level
        switch version {
        case "v1":
            c.Header("X-API-Stability", "stable")
        case "v2":
            c.Header("X-API-Stability", "beta")
        default:
            c.Header("X-API-Stability", "unknown")
        }

        c.Next()
    }
}

func logDeprecatedUsage(c *gin.Context, version string) {
    // Log for analytics
    slog.Warn("deprecated API version used",
        slog.String("version", version),
        slog.String("path", c.Request.URL.Path),
        slog.String("client_id", c.GetString("client_id")),
        slog.String("user_agent", c.Request.UserAgent()),
    )
}

// VersionNegotiationMiddleware handles Accept-Version header
func VersionNegotiationMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        acceptVersion := c.GetHeader("Accept-Version")

        if acceptVersion != "" {
            // Validate requested version
            switch acceptVersion {
            case "v1", "v2":
                c.Set("requested_version", acceptVersion)
            default:
                c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
                    "error": "unsupported API version",
                    "supported_versions": []string{"v1", "v2"},
                })
                return
            }
        }

        c.Next()
    }
}
```

### 2.3 버전별 응답 형식

```go
// internal/dto/response/v1/voucher.go
package v1

import "time"

// VoucherResponse is the v1 response format
type VoucherResponse struct {
    ID          string `json:"id"`
    VoucherNo   string `json:"voucher_no"`
    VoucherDate string `json:"voucher_date"` // YYYY-MM-DD
    Description string `json:"description"`
    Status      string `json:"status"`
    TotalDebit  string `json:"total_debit"`  // String for precision
    TotalCredit string `json:"total_credit"` // String for precision
    CreatedAt   string `json:"created_at"`   // ISO 8601
}

// internal/dto/response/v2/voucher.go
package v2

import "time"

// VoucherResponse is the v2 response format with enhancements
type VoucherResponse struct {
    ID          string `json:"id"`
    VoucherNo   string `json:"voucher_no"`
    VoucherDate string `json:"voucher_date"`
    Description string `json:"description"`
    Status      string `json:"status"`

    // V2: Enhanced amount fields
    Amounts AmountDetails `json:"amounts"`

    // V2: Nested lines included by default
    Lines []VoucherLineResponse `json:"lines"`

    // V2: Audit information
    Audit AuditInfo `json:"audit"`

    // V2: Links for HATEOAS
    Links Links `json:"_links"`
}

type AmountDetails struct {
    TotalDebit  string `json:"total_debit"`
    TotalCredit string `json:"total_credit"`
    Currency    string `json:"currency"`
}

type AuditInfo struct {
    CreatedAt  string `json:"created_at"`
    CreatedBy  string `json:"created_by"`
    UpdatedAt  string `json:"updated_at,omitempty"`
    UpdatedBy  string `json:"updated_by,omitempty"`
    PostedAt   string `json:"posted_at,omitempty"`
    PostedBy   string `json:"posted_by,omitempty"`
}

type Links struct {
    Self    string `json:"self"`
    Lines   string `json:"lines,omitempty"`
    Post    string `json:"post,omitempty"`
    History string `json:"history,omitempty"`
}
```

---

## 3. API 버전 정책

### 3.1 Semantic Versioning

```yaml
versioning_scheme:
  format: "v{major}.{minor}"
  url_format: "v{major}"

  major:
    description: "호환성 깨지는 변경"
    url_change: true
    examples:
      - "응답 필드 제거"
      - "필수 파라미터 추가"
      - "인증 방식 변경"
      - "에러 코드 체계 변경"

  minor:
    description: "하위 호환 기능 추가"
    url_change: false
    examples:
      - "새 엔드포인트 추가"
      - "선택적 파라미터 추가"
      - "응답 필드 추가"
      - "새 필터 옵션"

version_policy:
  current_stable: "v1"
  current_beta: "v2"

  support_timeline:
    ga_support: "최소 24개월"
    deprecation_notice: "12개월 전 공지"
    sunset_period: "6개월 유예"

  concurrent_versions:
    max: 2
    description: "최대 2개 major 버전 동시 지원"
```

### 3.2 버전 라이프사이클

```go
// pkg/versioning/lifecycle.go
package versioning

import "time"

type VersionStatus string

const (
    StatusAlpha      VersionStatus = "alpha"
    StatusBeta       VersionStatus = "beta"
    StatusGA         VersionStatus = "ga"         // General Availability
    StatusDeprecated VersionStatus = "deprecated"
    StatusSunset     VersionStatus = "sunset"
)

type APIVersion struct {
    Version           string
    Status            VersionStatus
    ReleasedAt        time.Time
    DeprecatedAt      *time.Time
    SunsetAt          *time.Time
    DeprecationNotice string
    MigrationGuide    string
}

var Versions = map[string]APIVersion{
    "v1": {
        Version:    "v1",
        Status:     StatusGA,
        ReleasedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
    },
    "v2": {
        Version:    "v2",
        Status:     StatusBeta,
        ReleasedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
    },
}

// IsSupported checks if version is still supported
func (v *APIVersion) IsSupported() bool {
    return v.Status != StatusSunset
}

// IsStable checks if version is production-ready
func (v *APIVersion) IsStable() bool {
    return v.Status == StatusGA
}

// ShouldWarnDeprecation checks if deprecation warning should be shown
func (v *APIVersion) ShouldWarnDeprecation() bool {
    return v.Status == StatusDeprecated
}
```

### 3.3 버전 헤더 규격

```yaml
request_headers:
  accept_version:
    name: "Accept-Version"
    description: "선호 API 버전 지정"
    example: "Accept-Version: v2"
    required: false

  api_key:
    name: "X-API-Key"
    description: "API 인증키"

response_headers:
  api_version:
    name: "X-API-Version"
    description: "응답에 사용된 API 버전"
    example: "X-API-Version: v1"

  api_stability:
    name: "X-API-Stability"
    values:
      - "stable"
      - "beta"
      - "alpha"

  deprecation:
    name: "X-API-Deprecated"
    description: "true if version is deprecated"

  sunset:
    name: "Sunset"
    description: "RFC 8594 Sunset header"
    example: "Sunset: Sat, 01 Jun 2024 00:00:00 GMT"

  deprecation_link:
    name: "Link"
    description: "Migration guide link"
    example: 'Link: <https://api.example.com/docs/migration/v1-to-v2>; rel="deprecation"'
```

---

## 4. 하위 호환성 관리

### 4.1 호환성 유지 규칙

```yaml
backward_compatible_changes:
  allowed:
    - "새 엔드포인트 추가"
    - "응답에 선택적 필드 추가"
    - "요청에 선택적 파라미터 추가"
    - "새 HTTP 메서드 지원"
    - "새 필터/정렬 옵션"
    - "Rate limit 완화"
    - "응답 필드 값 범위 확장"

breaking_changes:
  not_allowed_without_major_version:
    - "응답 필드 제거"
    - "응답 필드 이름 변경"
    - "응답 필드 타입 변경"
    - "필수 파라미터 추가"
    - "URL 경로 변경"
    - "HTTP 메서드 변경"
    - "인증 방식 변경"
    - "에러 코드 변경"
    - "Rate limit 강화"

edge_cases:
  field_type_loosening:
    description: "문자열을 숫자 등으로 변경"
    policy: "Breaking - 클라이언트 파싱 실패 가능"

  null_to_default:
    description: "null 반환하던 것을 기본값으로 변경"
    policy: "Compatible - 단, 문서화 필요"

  enum_extension:
    description: "enum에 새 값 추가"
    policy: "Potentially breaking - 클라이언트 핸들링 필요"
```

### 4.2 필드 추가/제거 관리

```go
// internal/dto/response/voucher.go
package response

import (
    "encoding/json"
)

// VoucherResponse with version-aware serialization
type VoucherResponse struct {
    ID          string `json:"id"`
    VoucherNo   string `json:"voucher_no"`
    VoucherDate string `json:"voucher_date"`
    Description string `json:"description"`
    Status      string `json:"status"`
    TotalDebit  string `json:"total_debit"`
    TotalCredit string `json:"total_credit"`

    // V2+ fields (omitted in v1)
    Lines   []VoucherLineResponse `json:"lines,omitempty"`
    Audit   *AuditInfo            `json:"audit,omitempty"`

    // Internal version tracking
    apiVersion string `json:"-"`
}

// SetAPIVersion sets the API version for serialization
func (v *VoucherResponse) SetAPIVersion(version string) {
    v.apiVersion = version
}

// MarshalJSON customizes JSON output based on version
func (v *VoucherResponse) MarshalJSON() ([]byte, error) {
    type Alias VoucherResponse

    switch v.apiVersion {
    case "v1":
        // V1: Exclude v2+ fields
        return json.Marshal(&struct {
            *Alias
            Lines []VoucherLineResponse `json:"lines,omitempty"`
            Audit *AuditInfo            `json:"audit,omitempty"`
        }{
            Alias: (*Alias)(v),
            Lines: nil,
            Audit: nil,
        })
    default:
        // V2+: Include all fields
        return json.Marshal((*Alias)(v))
    }
}

// Deprecated field handling
type AccountResponse struct {
    Code        string `json:"code"`
    Name        string `json:"name"`
    AccountType string `json:"account_type"`

    // Deprecated: Use account_type instead
    // Will be removed in v3
    Type string `json:"type,omitempty"`
}

func (a *AccountResponse) SetAPIVersion(version string) {
    switch version {
    case "v1":
        // V1: Include deprecated field for compatibility
        a.Type = a.AccountType
    default:
        // V2+: Don't include deprecated field
        a.Type = ""
    }
}
```

### 4.3 열거형 확장 처리

```go
// internal/domain/voucher_status.go
package domain

type VoucherStatus string

const (
    VoucherStatusDraft     VoucherStatus = "draft"
    VoucherStatusPosted    VoucherStatus = "posted"
    VoucherStatusCancelled VoucherStatus = "cancelled"
    // V2: New status
    VoucherStatusPending   VoucherStatus = "pending" // Added in v2
)

// ValidStatuses returns valid statuses for the given API version
func ValidStatuses(apiVersion string) []VoucherStatus {
    base := []VoucherStatus{
        VoucherStatusDraft,
        VoucherStatusPosted,
        VoucherStatusCancelled,
    }

    switch apiVersion {
    case "v2":
        return append(base, VoucherStatusPending)
    default:
        return base
    }
}

// IsValidForVersion checks if status is valid for API version
func (s VoucherStatus) IsValidForVersion(apiVersion string) bool {
    for _, valid := range ValidStatuses(apiVersion) {
        if s == valid {
            return true
        }
    }
    return false
}
```

---

## 5. 버전 마이그레이션

### 5.1 마이그레이션 가이드 구조

```markdown
<!-- docs/api/migration/v1-to-v2.md -->

# API v1 to v2 Migration Guide

## Overview

This guide helps you migrate from API v1 to v2.

**Timeline:**
- v2 GA: 2024-06-01
- v1 Deprecation: 2024-12-01
- v1 Sunset: 2025-06-01

## Breaking Changes

### 1. Voucher Response Structure

**v1 Response:**
```json
{
  "id": "...",
  "total_debit": "1000.00",
  "total_credit": "1000.00",
  "created_at": "2024-01-15T09:00:00Z"
}
```

**v2 Response:**
```json
{
  "id": "...",
  "amounts": {
    "total_debit": "1000.00",
    "total_credit": "1000.00",
    "currency": "KRW"
  },
  "audit": {
    "created_at": "2024-01-15T09:00:00Z",
    "created_by": "user_id"
  }
}
```

**Migration Steps:**
1. Update response parsing to use `amounts.total_debit` instead of `total_debit`
2. Update timestamp handling to use `audit.created_at`

### 2. Error Response Format

**v1 Error:**
```json
{
  "error": "Validation failed",
  "details": ["Field required: voucher_date"]
}
```

**v2 Error:**
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed",
    "details": {
      "voucher_date": ["This field is required"]
    }
  }
}
```

## New Features in v2

- Bulk operations: POST /api/v2/vouchers/bulk
- Audit history: GET /api/v2/vouchers/{id}/history
- Enhanced filtering with date ranges and status
```

### 5.2 버전 어댑터

```go
// internal/handler/adapter/voucher_adapter.go
package adapter

import (
    domainv1 "github.com/company/kerp/internal/dto/response/v1"
    domainv2 "github.com/company/kerp/internal/dto/response/v2"
    "github.com/company/kerp/internal/domain"
)

// VoucherAdapter converts domain models to version-specific responses
type VoucherAdapter struct{}

// ToV1Response converts domain voucher to v1 response
func (a *VoucherAdapter) ToV1Response(v *domain.Voucher) *domainv1.VoucherResponse {
    return &domainv1.VoucherResponse{
        ID:          v.ID.String(),
        VoucherNo:   v.VoucherNo,
        VoucherDate: v.VoucherDate.Format("2006-01-02"),
        Description: v.Description,
        Status:      string(v.Status),
        TotalDebit:  v.TotalDebit.String(),
        TotalCredit: v.TotalCredit.String(),
        CreatedAt:   v.CreatedAt.Format(time.RFC3339),
    }
}

// ToV2Response converts domain voucher to v2 response
func (a *VoucherAdapter) ToV2Response(v *domain.Voucher, baseURL string) *domainv2.VoucherResponse {
    lines := make([]domainv2.VoucherLineResponse, len(v.Lines))
    for i, line := range v.Lines {
        lines[i] = domainv2.VoucherLineResponse{
            ID:          line.ID.String(),
            LineNo:      line.LineNo,
            AccountCode: line.AccountCode,
            DebitAmt:    line.DebitAmt.String(),
            CreditAmt:   line.CreditAmt.String(),
        }
    }

    return &domainv2.VoucherResponse{
        ID:          v.ID.String(),
        VoucherNo:   v.VoucherNo,
        VoucherDate: v.VoucherDate.Format("2006-01-02"),
        Description: v.Description,
        Status:      string(v.Status),
        Amounts: domainv2.AmountDetails{
            TotalDebit:  v.TotalDebit.String(),
            TotalCredit: v.TotalCredit.String(),
            Currency:    "KRW",
        },
        Lines: lines,
        Audit: domainv2.AuditInfo{
            CreatedAt: v.CreatedAt.Format(time.RFC3339),
            CreatedBy: v.CreatedBy.String(),
            UpdatedAt: formatOptionalTime(v.UpdatedAt),
            PostedAt:  formatOptionalTime(v.PostedAt),
        },
        Links: domainv2.Links{
            Self:    fmt.Sprintf("%s/api/v2/vouchers/%s", baseURL, v.ID),
            Lines:   fmt.Sprintf("%s/api/v2/vouchers/%s/lines", baseURL, v.ID),
            History: fmt.Sprintf("%s/api/v2/vouchers/%s/history", baseURL, v.ID),
        },
    }
}

func formatOptionalTime(t *time.Time) string {
    if t == nil {
        return ""
    }
    return t.Format(time.RFC3339)
}
```

### 5.3 점진적 마이그레이션 지원

```go
// internal/handler/v1/voucher_handler.go
package v1

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

type VoucherHandler struct {
    service *service.VoucherService
    adapter *adapter.VoucherAdapter
}

func (h *VoucherHandler) Get(c *gin.Context) {
    id := c.Param("id")

    voucher, err := h.service.GetVoucher(c.Request.Context(), uuid.MustParse(id))
    if err != nil {
        handleError(c, err)
        return
    }

    // Add deprecation hint header
    if c.GetHeader("X-Request-V2-Preview") == "true" {
        // Return v2 format in v1 endpoint for testing
        c.Header("X-Preview-Version", "v2")
        response := h.adapter.ToV2Response(voucher, getBaseURL(c))
        c.JSON(http.StatusOK, response)
        return
    }

    // Standard v1 response
    response := h.adapter.ToV1Response(voucher)
    c.JSON(http.StatusOK, response)
}

// Migration helper endpoint
func (h *VoucherHandler) GetMigrationPreview(c *gin.Context) {
    id := c.Param("id")

    voucher, err := h.service.GetVoucher(c.Request.Context(), uuid.MustParse(id))
    if err != nil {
        handleError(c, err)
        return
    }

    // Return both v1 and v2 format for comparison
    c.JSON(http.StatusOK, gin.H{
        "v1": h.adapter.ToV1Response(voucher),
        "v2": h.adapter.ToV2Response(voucher, getBaseURL(c)),
        "migration_guide": "https://api.kerp.io/docs/migration/v1-to-v2",
    })
}
```

---

## 6. API 변경 프로세스

### 6.1 변경 요청 워크플로우

```yaml
api_change_workflow:
  stages:
    1_proposal:
      name: "RFC 제안"
      activities:
        - "변경 사유 문서화"
        - "영향 범위 분석"
        - "호환성 평가"
      artifacts:
        - "API RFC 문서"

    2_review:
      name: "기술 검토"
      activities:
        - "아키텍처 리뷰"
        - "보안 검토"
        - "성능 영향 분석"
      reviewers:
        - "Tech Lead"
        - "API Owner"
        - "Security Team"

    3_implementation:
      name: "구현"
      activities:
        - "코드 구현"
        - "테스트 작성"
        - "문서 업데이트"
      requirements:
        - "OpenAPI 스펙 업데이트"
        - "마이그레이션 가이드 작성"
        - "SDK 업데이트 계획"

    4_beta:
      name: "베타 릴리스"
      activities:
        - "내부 테스트"
        - "파트너 프리뷰"
        - "피드백 수집"
      duration: "4-8 weeks"

    5_ga:
      name: "GA 릴리스"
      activities:
        - "공식 발표"
        - "문서 공개"
        - "구버전 deprecation 일정 공지"
```

### 6.2 API RFC 템플릿

```markdown
<!-- docs/api/rfc/RFC-XXXX-template.md -->

# RFC-XXXX: [Title]

## Status
- [ ] Draft
- [ ] Review
- [ ] Approved
- [ ] Implemented

## Summary
Brief description of the proposed change.

## Motivation
Why is this change needed?

## Detailed Design

### Endpoint Changes
```
GET /api/v2/vouchers/{id}/audit-trail  (NEW)
```

### Request Format
```json
{
  // New request fields
}
```

### Response Format
```json
{
  // New response fields
}
```

## Compatibility Impact

### Breaking Changes
- [ ] None
- [ ] Yes - describe below

### Migration Path
How will existing clients migrate?

## Security Considerations
Any security implications?

## Performance Considerations
Any performance implications?

## Alternatives Considered
What other approaches were considered?

## Timeline
- RFC Review: YYYY-MM-DD
- Beta Release: YYYY-MM-DD
- GA Release: YYYY-MM-DD
```

### 6.3 변경 영향 분석

```go
// tools/api-analyzer/main.go
package main

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/getkin/kin-openapi/openapi3"
)

type BreakingChange struct {
    Path        string `json:"path"`
    Method      string `json:"method"`
    ChangeType  string `json:"change_type"`
    Description string `json:"description"`
    Severity    string `json:"severity"`
}

func main() {
    oldSpec, _ := openapi3.NewLoader().LoadFromFile("api/v1/openapi.yaml")
    newSpec, _ := openapi3.NewLoader().LoadFromFile("api/v2/openapi.yaml")

    changes := compareSpecs(oldSpec, newSpec)

    report := AnalysisReport{
        TotalChanges:    len(changes),
        BreakingChanges: filterBreaking(changes),
        Additions:       filterAdditions(changes),
        Deprecations:    filterDeprecations(changes),
    }

    output, _ := json.MarshalIndent(report, "", "  ")
    fmt.Println(string(output))
}

func compareSpecs(old, new *openapi3.T) []BreakingChange {
    var changes []BreakingChange

    // Check for removed paths
    for path := range old.Paths.Map() {
        if new.Paths.Find(path) == nil {
            changes = append(changes, BreakingChange{
                Path:        path,
                ChangeType:  "path_removed",
                Description: "Endpoint removed",
                Severity:    "breaking",
            })
        }
    }

    // Check for removed/changed fields in responses
    for path, pathItem := range old.Paths.Map() {
        newPathItem := new.Paths.Find(path)
        if newPathItem == nil {
            continue
        }

        for method, op := range pathItem.Operations() {
            newOp := newPathItem.GetOperation(method)
            if newOp == nil {
                changes = append(changes, BreakingChange{
                    Path:        path,
                    Method:      method,
                    ChangeType:  "method_removed",
                    Description: fmt.Sprintf("%s method removed", method),
                    Severity:    "breaking",
                })
                continue
            }

            // Compare response schemas
            fieldChanges := compareResponses(op, newOp)
            changes = append(changes, fieldChanges...)
        }
    }

    return changes
}
```

---

## 7. 클라이언트 SDK 관리

### 7.1 SDK 버전 전략

```yaml
sdk_versioning:
  strategy: "SDK major version = API major version"

  versions:
    kerp-sdk-go:
      v1: "API v1 지원"
      v2: "API v2 지원 (v1 하위호환)"

    kerp-sdk-js:
      v1: "API v1 지원"
      v2: "API v2 지원 (v1 하위호환)"

    kerp-sdk-python:
      v1: "API v1 지원"
      v2: "API v2 지원 (v1 하위호환)"

  release_timeline:
    api_beta: "SDK alpha 릴리스"
    api_ga: "SDK stable 릴리스"
    api_deprecation: "SDK 업그레이드 안내"
```

### 7.2 Go SDK 예시

```go
// sdk/go/kerp/client.go
package kerp

import (
    "context"
    "net/http"
)

// ClientOption configures the client
type ClientOption func(*Client)

// WithAPIVersion sets the API version to use
func WithAPIVersion(version string) ClientOption {
    return func(c *Client) {
        c.apiVersion = version
    }
}

// WithBaseURL sets custom base URL
func WithBaseURL(url string) ClientOption {
    return func(c *Client) {
        c.baseURL = url
    }
}

type Client struct {
    httpClient *http.Client
    baseURL    string
    apiKey     string
    apiVersion string

    // Service clients
    Vouchers *VoucherService
    Accounts *AccountService
}

func NewClient(apiKey string, opts ...ClientOption) *Client {
    c := &Client{
        httpClient: &http.Client{},
        baseURL:    "https://api.kerp.io",
        apiKey:     apiKey,
        apiVersion: "v1", // Default to stable
    }

    for _, opt := range opts {
        opt(c)
    }

    c.Vouchers = &VoucherService{client: c}
    c.Accounts = &AccountService{client: c}

    return c
}

// VoucherService handles voucher operations
type VoucherService struct {
    client *Client
}

// Get retrieves a voucher by ID
func (s *VoucherService) Get(ctx context.Context, id string) (*Voucher, error) {
    path := fmt.Sprintf("/api/%s/vouchers/%s", s.client.apiVersion, id)

    var voucher Voucher
    if err := s.client.get(ctx, path, &voucher); err != nil {
        return nil, err
    }

    return &voucher, nil
}

// List retrieves vouchers with optional filters
func (s *VoucherService) List(ctx context.Context, opts *ListOptions) (*VoucherList, error) {
    path := fmt.Sprintf("/api/%s/vouchers", s.client.apiVersion)

    if opts != nil {
        path += "?" + opts.Encode()
    }

    var list VoucherList
    if err := s.client.get(ctx, path, &list); err != nil {
        return nil, err
    }

    return &list, nil
}

// Create creates a new voucher
func (s *VoucherService) Create(ctx context.Context, req *CreateVoucherRequest) (*Voucher, error) {
    path := fmt.Sprintf("/api/%s/vouchers", s.client.apiVersion)

    var voucher Voucher
    if err := s.client.post(ctx, path, req, &voucher); err != nil {
        return nil, err
    }

    return &voucher, nil
}
```

### 7.3 SDK 마이그레이션 가이드

```markdown
<!-- sdk/go/MIGRATION.md -->

# K-ERP Go SDK Migration Guide (v1 to v2)

## Installation

```bash
# Upgrade to v2
go get github.com/company/kerp-sdk-go/v2
```

## Import Changes

```go
// v1
import "github.com/company/kerp-sdk-go"

// v2
import kerp "github.com/company/kerp-sdk-go/v2"
```

## Client Initialization

```go
// v1
client := kerp.NewClient("api_key")

// v2 - explicit version selection
client := kerp.NewClient("api_key", kerp.WithAPIVersion("v2"))
```

## Response Changes

```go
// v1
voucher, _ := client.Vouchers.Get(ctx, "id")
fmt.Println(voucher.TotalDebit)

// v2
voucher, _ := client.Vouchers.Get(ctx, "id")
fmt.Println(voucher.Amounts.TotalDebit)
fmt.Println(voucher.Audit.CreatedAt)
```

## New Features in v2

```go
// Bulk create (v2 only)
vouchers, _ := client.Vouchers.BulkCreate(ctx, []*CreateVoucherRequest{...})

// Get history (v2 only)
history, _ := client.Vouchers.GetHistory(ctx, "voucher_id")
```
```

---

## 8. 문서화 및 커뮤니케이션

### 8.1 API 문서 구조

```yaml
documentation_structure:
  api_reference:
    location: "docs/api/"
    format: "OpenAPI 3.0"
    files:
      - "openapi-v1.yaml"
      - "openapi-v2.yaml"

  guides:
    location: "docs/guides/"
    files:
      - "getting-started.md"
      - "authentication.md"
      - "pagination.md"
      - "error-handling.md"
      - "rate-limiting.md"
      - "webhooks.md"

  migration:
    location: "docs/migration/"
    files:
      - "v1-to-v2.md"

  changelog:
    location: "CHANGELOG.md"
    format: "Keep a Changelog"

  sdk_docs:
    location: "sdk/{language}/README.md"
```

### 8.2 Changelog 형식

```markdown
<!-- CHANGELOG.md -->

# API Changelog

All notable changes to the K-ERP API will be documented in this file.

## [v2.0.0] - 2024-06-01

### Added
- Bulk voucher creation endpoint: `POST /api/v2/vouchers/bulk`
- Voucher audit history endpoint: `GET /api/v2/vouchers/{id}/history`
- HATEOAS links in voucher responses
- Enhanced filtering with date ranges

### Changed
- **BREAKING**: Voucher response structure reorganized
  - `total_debit` → `amounts.total_debit`
  - `total_credit` → `amounts.total_credit`
  - Added `amounts.currency` field
- **BREAKING**: Error response format changed to include error codes
- Timestamps now include timezone information

### Deprecated
- v1 API deprecated (sunset: 2025-06-01)

## [v1.5.0] - 2024-03-01

### Added
- Account balance endpoint: `GET /api/v1/accounts/{code}/balance`
- Voucher status filter parameter

### Fixed
- Pagination count accuracy for filtered results
```

### 8.3 Deprecation 공지

```go
// internal/handler/deprecation.go
package handler

import (
    "fmt"
    "time"
)

type DeprecationNotice struct {
    Feature      string
    Message      string
    DeprecatedAt time.Time
    SunsetAt     time.Time
    Alternative  string
    DocsURL      string
}

var deprecations = []DeprecationNotice{
    {
        Feature:      "API v1",
        Message:      "API v1 is deprecated and will be removed",
        DeprecatedAt: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
        SunsetAt:     time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
        Alternative:  "Migrate to API v2",
        DocsURL:      "https://api.kerp.io/docs/migration/v1-to-v2",
    },
    {
        Feature:      "type field in AccountResponse",
        Message:      "The 'type' field is deprecated, use 'account_type' instead",
        DeprecatedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
        SunsetAt:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
        Alternative:  "Use 'account_type' field",
        DocsURL:      "https://api.kerp.io/docs/migration/account-type-field",
    },
}

// GenerateDeprecationEmail generates email content for deprecation notice
func GenerateDeprecationEmail(notice DeprecationNotice) string {
    return fmt.Sprintf(`
Subject: [K-ERP API] %s Deprecation Notice

Dear API User,

We're writing to inform you about an upcoming change to the K-ERP API.

**What's changing:**
%s

**Timeline:**
- Deprecated: %s
- Sunset (removal): %s

**What you need to do:**
%s

**Resources:**
- Migration Guide: %s
- API Documentation: https://api.kerp.io/docs

If you have any questions, please contact api-support@kerp.io.

Best regards,
K-ERP API Team
`, notice.Feature, notice.Message,
        notice.DeprecatedAt.Format("January 2, 2006"),
        notice.SunsetAt.Format("January 2, 2006"),
        notice.Alternative, notice.DocsURL)
}
```

### 8.4 API 상태 페이지

```typescript
// web/src/app/api-status/page.tsx
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';

interface APIVersion {
  version: string;
  status: 'stable' | 'beta' | 'deprecated' | 'sunset';
  releasedAt: string;
  deprecatedAt?: string;
  sunsetAt?: string;
  features: string[];
}

const apiVersions: APIVersion[] = [
  {
    version: 'v2',
    status: 'beta',
    releasedAt: '2024-06-01',
    features: [
      'Bulk operations',
      'Audit history',
      'Enhanced filtering',
      'HATEOAS links',
    ],
  },
  {
    version: 'v1',
    status: 'stable',
    releasedAt: '2024-01-01',
    features: [
      'CRUD operations',
      'Basic filtering',
      'Pagination',
    ],
  },
];

export default function APIStatusPage() {
  return (
    <div className="container mx-auto py-8">
      <h1 className="text-3xl font-bold mb-8">API Version Status</h1>

      <div className="grid gap-6">
        {apiVersions.map((version) => (
          <Card key={version.version}>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>{version.version.toUpperCase()}</CardTitle>
                <StatusBadge status={version.status} />
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div>
                  <span className="text-sm text-muted-foreground">Released:</span>
                  <span className="ml-2">{version.releasedAt}</span>
                </div>

                {version.deprecatedAt && (
                  <div className="text-yellow-600">
                    <span className="text-sm">Deprecated:</span>
                    <span className="ml-2">{version.deprecatedAt}</span>
                  </div>
                )}

                {version.sunsetAt && (
                  <div className="text-red-600">
                    <span className="text-sm">Sunset:</span>
                    <span className="ml-2">{version.sunsetAt}</span>
                  </div>
                )}

                <div>
                  <span className="text-sm text-muted-foreground">Features:</span>
                  <ul className="mt-2 list-disc list-inside">
                    {version.features.map((feature) => (
                      <li key={feature}>{feature}</li>
                    ))}
                  </ul>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status: APIVersion['status'] }) {
  const variants = {
    stable: 'bg-green-100 text-green-800',
    beta: 'bg-blue-100 text-blue-800',
    deprecated: 'bg-yellow-100 text-yellow-800',
    sunset: 'bg-red-100 text-red-800',
  };

  return (
    <Badge className={variants[status]}>
      {status.toUpperCase()}
    </Badge>
  );
}
```

---

## 버전 이력

| 버전 | 날짜 | 작성자 | 변경 내용 |
|------|------|--------|-----------|
| 0.2 | 2024-01-15 | DevTeam | Go+Python 하이브리드 아키텍처 반영 |
| 0.1 | 2024-01-01 | DevTeam | 초안 작성 |
