# 24. API 버전관리 전략

## 1. 개요

### 1.1 목적
K-ERP API의 하위 호환성을 유지하면서 지속적인 기능 개선과 변경을 관리하기 위한 버전관리 전략을 정의한다.

### 1.2 버전관리 원칙
- **하위 호환성 우선**: 기존 클라이언트의 동작을 보장
- **명시적 버전 표기**: URL 경로에 버전을 명시
- **점진적 폐기**: 충분한 마이그레이션 기간 제공
- **문서화 필수**: 모든 변경사항 문서화

## 2. 버전 체계

### 2.1 시맨틱 버전 규칙
```
MAJOR.MINOR.PATCH

예: v1.2.3
- MAJOR (1): 하위 호환성이 깨지는 변경
- MINOR (2): 하위 호환 기능 추가
- PATCH (3): 하위 호환 버그 수정
```

### 2.2 API 버전 vs 서비스 버전
| 구분 | API 버전 | 서비스 버전 |
|------|----------|-------------|
| **표기** | v1, v2, v3 | 1.0.0, 1.1.0 |
| **변경 주기** | 연 1~2회 | 주 1~2회 |
| **호환성** | Major 변경 시 | 항상 호환 |
| **URL 반영** | O | X |

### 2.3 URL 버전 규칙
```
# URL Path 방식 (권장)
GET /api/v1/vouchers
GET /api/v2/vouchers

# Header 방식 (내부 마이크로서비스)
GET /api/vouchers
X-API-Version: 2

# Query Parameter (비권장)
GET /api/vouchers?version=1
```

## 3. Go 라우터 구현

### 3.1 버전별 라우터 구조
```go
// cmd/api/router.go
package main

import (
    "github.com/gin-gonic/gin"

    v1 "k-erp/internal/handler/v1"
    v2 "k-erp/internal/handler/v2"
)

func SetupRouter(
    v1Handler *v1.Handler,
    v2Handler *v2.Handler,
) *gin.Engine {
    r := gin.New()

    // 공통 미들웨어
    r.Use(gin.Recovery())
    r.Use(middleware.Logger())
    r.Use(middleware.CORS())
    r.Use(middleware.RequestID())

    // API v1 라우트
    apiV1 := r.Group("/api/v1")
    apiV1.Use(middleware.Auth())
    apiV1.Use(middleware.Tenant())
    {
        setupV1Routes(apiV1, v1Handler)
    }

    // API v2 라우트
    apiV2 := r.Group("/api/v2")
    apiV2.Use(middleware.Auth())
    apiV2.Use(middleware.Tenant())
    {
        setupV2Routes(apiV2, v2Handler)
    }

    return r
}

func setupV1Routes(rg *gin.RouterGroup, h *v1.Handler) {
    // 회계
    vouchers := rg.Group("/vouchers")
    {
        vouchers.GET("", h.ListVouchers)
        vouchers.POST("", h.CreateVoucher)
        vouchers.GET("/:id", h.GetVoucher)
        vouchers.PUT("/:id", h.UpdateVoucher)
        vouchers.DELETE("/:id", h.DeleteVoucher)
    }

    // 세금계산서
    taxInvoices := rg.Group("/tax-invoices")
    {
        taxInvoices.GET("", h.ListTaxInvoices)
        taxInvoices.POST("", h.CreateTaxInvoice)
        taxInvoices.POST("/:id/issue", h.IssueTaxInvoice)
    }
}

func setupV2Routes(rg *gin.RouterGroup, h *v2.Handler) {
    // v2에서 개선된 전표 API
    vouchers := rg.Group("/vouchers")
    {
        vouchers.GET("", h.ListVouchers)      // 페이지네이션 개선
        vouchers.POST("", h.CreateVoucher)    // 배치 생성 지원
        vouchers.POST("/batch", h.BatchCreateVouchers)
        vouchers.GET("/:id", h.GetVoucher)
        vouchers.PATCH("/:id", h.PatchVoucher) // PATCH 지원 추가
    }

    // v2 신규 기능
    analytics := rg.Group("/analytics")
    {
        analytics.GET("/dashboard", h.GetDashboard)
        analytics.GET("/reports/:type", h.GetReport)
    }
}
```

### 3.2 버전별 핸들러 구조
```go
// internal/handler/v1/voucher.go
package v1

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "k-erp/internal/service"
)

type Handler struct {
    voucherService service.VoucherService
}

// V1 응답 형식
type VoucherResponse struct {
    ID          uuid.UUID `json:"id"`
    VoucherNo   string    `json:"voucher_no"`
    VoucherDate string    `json:"voucher_date"`
    Description string    `json:"description"`
    TotalDebit  float64   `json:"total_debit"`
    TotalCredit float64   `json:"total_credit"`
    Status      string    `json:"status"`
    CreatedAt   string    `json:"created_at"`
}

func (h *Handler) GetVoucher(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }

    voucher, err := h.voucherService.GetByID(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "voucher not found"})
        return
    }

    // V1 형식으로 응답
    c.JSON(http.StatusOK, toV1Response(voucher))
}

func toV1Response(v *domain.Voucher) VoucherResponse {
    return VoucherResponse{
        ID:          v.ID,
        VoucherNo:   v.VoucherNo,
        VoucherDate: v.VoucherDate.Format("2006-01-02"),
        Description: v.Description,
        TotalDebit:  v.TotalDebit.InexactFloat64(),
        TotalCredit: v.TotalCredit.InexactFloat64(),
        Status:      string(v.Status),
        CreatedAt:   v.CreatedAt.Format(time.RFC3339),
    }
}
```

```go
// internal/handler/v2/voucher.go
package v2

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/shopspring/decimal"
)

// V2 응답 형식 (개선됨)
type VoucherResponse struct {
    ID          uuid.UUID           `json:"id"`
    VoucherNo   string              `json:"voucher_no"`
    VoucherDate string              `json:"voucher_date"`
    Description string              `json:"description"`
    Amount      AmountResponse      `json:"amount"`      // 구조화
    Status      StatusResponse      `json:"status"`      // 상세 상태
    Metadata    MetadataResponse    `json:"metadata"`    // 메타데이터 추가
    Lines       []VoucherLineResponse `json:"lines,omitempty"` // 포함 옵션
    Links       LinksResponse       `json:"_links"`      // HATEOAS
}

type AmountResponse struct {
    Debit    string `json:"debit"`     // decimal string
    Credit   string `json:"credit"`    // decimal string
    Currency string `json:"currency"`  // KRW
}

type StatusResponse struct {
    Code        string `json:"code"`
    Label       string `json:"label"`
    IsEditable  bool   `json:"is_editable"`
    IsReversible bool  `json:"is_reversible"`
}

type MetadataResponse struct {
    CreatedAt   string `json:"created_at"`
    CreatedBy   string `json:"created_by"`
    UpdatedAt   string `json:"updated_at,omitempty"`
    UpdatedBy   string `json:"updated_by,omitempty"`
    Version     int    `json:"version"`
}

type LinksResponse struct {
    Self    string `json:"self"`
    Lines   string `json:"lines"`
    Reverse string `json:"reverse,omitempty"`
    PDF     string `json:"pdf,omitempty"`
}

func (h *Handler) GetVoucher(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        respondError(c, ErrInvalidID)
        return
    }

    // include 파라미터 처리 (v2 신규)
    includeLines := c.Query("include") == "lines"

    voucher, err := h.voucherService.GetByID(c.Request.Context(), id)
    if err != nil {
        respondError(c, err)
        return
    }

    response := toV2Response(voucher, includeLines)
    c.JSON(http.StatusOK, response)
}
```

## 4. 변경 유형별 전략

### 4.1 하위 호환 변경 (Minor)
**동일 버전 내 처리:**
```go
// 필드 추가 (기존 클라이언트 영향 없음)
type VoucherResponse struct {
    ID          uuid.UUID `json:"id"`
    VoucherNo   string    `json:"voucher_no"`
    // ... 기존 필드

    // v1.2.0 추가 - 클라이언트는 무시 가능
    Tags        []string  `json:"tags,omitempty"`
    Attachments int       `json:"attachments_count,omitempty"`
}

// 선택적 파라미터 추가
// GET /api/v1/vouchers?include_tags=true (신규)
// GET /api/v1/vouchers (기존 - 변경 없음)
func (h *Handler) ListVouchers(c *gin.Context) {
    includeTags := c.Query("include_tags") == "true"

    vouchers, err := h.service.List(c.Request.Context(), ListOptions{
        IncludeTags: includeTags,
    })
    // ...
}
```

### 4.2 하위 비호환 변경 (Major)
**새 버전 생성 필수:**
```go
// v1: amount는 숫자
type VoucherV1 struct {
    Amount float64 `json:"amount"` // 부동소수점
}

// v2: amount는 객체로 변경 (Breaking Change)
type VoucherV2 struct {
    Amount struct {
        Value    string `json:"value"`    // decimal string
        Currency string `json:"currency"`
    } `json:"amount"`
}
```

### 4.3 변경 유형 분류표
| 변경 유형 | 호환성 | 버전 변경 | 예시 |
|-----------|--------|-----------|------|
| 필드 추가 | O | Minor | `tags` 필드 추가 |
| 선택 파라미터 추가 | O | Minor | `?include=lines` |
| 새 엔드포인트 | O | Minor | `GET /reports` 추가 |
| 필드 제거 | X | Major | `legacy_id` 제거 |
| 필드 타입 변경 | X | Major | `float → string` |
| 필드 이름 변경 | X | Major | `amt → amount` |
| 필수 파라미터 추가 | X | Major | `company_id` 필수화 |
| URL 구조 변경 | X | Major | `/invoice → /tax-invoices` |

## 5. API 폐기(Deprecation) 정책

### 5.1 폐기 프로세스
```
1. 폐기 예고 (D-day)
   └─ 문서 업데이트, 헤더 추가

2. 경고 기간 (6개월)
   └─ Deprecated 헤더 반환
   └─ 사용량 모니터링

3. 마이그레이션 지원 (3개월)
   └─ 마이그레이션 가이드 제공
   └─ 고객 개별 연락

4. 종료 (D+9개월)
   └─ 410 Gone 반환
   └─ 완전 제거 (D+12개월)
```

### 5.2 Deprecation 헤더
```go
// internal/middleware/deprecation.go
package middleware

import (
    "fmt"
    "time"

    "github.com/gin-gonic/gin"
)

type DeprecationConfig struct {
    Endpoint    string
    Deprecated  time.Time
    Sunset      time.Time
    Alternative string
}

var deprecatedEndpoints = map[string]DeprecationConfig{
    "/api/v1/invoices": {
        Deprecated:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
        Sunset:      time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
        Alternative: "/api/v2/tax-invoices",
    },
}

func Deprecation() gin.HandlerFunc {
    return func(c *gin.Context) {
        path := c.Request.URL.Path

        if config, exists := deprecatedEndpoints[path]; exists {
            // RFC 8594 Deprecation Header
            c.Header("Deprecation", config.Deprecated.Format(time.RFC1123))
            c.Header("Sunset", config.Sunset.Format(time.RFC1123))
            c.Header("Link", fmt.Sprintf("<%s>; rel=\"successor-version\"",
                config.Alternative))

            // 경고 로그
            log.Warn("deprecated API called",
                "endpoint", path,
                "client_ip", c.ClientIP(),
                "sunset", config.Sunset,
            )

            // 메트릭 기록
            deprecatedAPICallsTotal.WithLabelValues(path).Inc()
        }

        c.Next()
    }
}
```

### 5.3 버전 종료 응답
```go
// internal/middleware/version_check.go
func VersionCheck() gin.HandlerFunc {
    sunsetVersions := map[string]time.Time{
        "v0": time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
    }

    return func(c *gin.Context) {
        // URL에서 버전 추출
        version := extractVersion(c.Request.URL.Path)

        if sunset, exists := sunsetVersions[version]; exists {
            if time.Now().After(sunset) {
                c.AbortWithStatusJSON(http.StatusGone, gin.H{
                    "error": "api_version_sunset",
                    "message": fmt.Sprintf("API %s is no longer available", version),
                    "sunset_date": sunset.Format(time.RFC3339),
                    "migration_guide": "https://docs.k-erp.io/migration/v0-to-v1",
                    "current_version": "v2",
                })
                return
            }
        }

        c.Next()
    }
}
```

## 6. API 변경 문서화

### 6.1 CHANGELOG 형식
```markdown
# API Changelog

## [v2.0.0] - 2025-03-01

### Breaking Changes
- `GET /vouchers/:id` 응답의 `amount` 필드가 객체로 변경
  - Before: `"amount": 10000.00`
  - After: `"amount": {"value": "10000.00", "currency": "KRW"}`
- `POST /vouchers` 요청 시 `lines` 필드 필수화

### Migration Guide
```go
// Before (v1)
type Amount = float64

// After (v2)
type Amount struct {
    Value    string `json:"value"`
    Currency string `json:"currency"`
}
```

### Added
- `PATCH /vouchers/:id` 부분 업데이트 지원
- `POST /vouchers/batch` 배치 생성 API
- `GET /analytics/*` 분석 API 추가

### Deprecated
- `GET /api/v1/invoices` → `GET /api/v2/tax-invoices` 사용

## [v1.5.0] - 2025-02-01

### Added
- `tags` 필드 추가 (선택)
- `include=lines` 쿼리 파라미터 지원

### Fixed
- 페이지네이션 cursor 인코딩 버그 수정
```

### 6.2 OpenAPI 버전 관리
```yaml
# openapi/v1.yaml
openapi: 3.0.3
info:
  title: K-ERP API
  version: "1.5.0"
  description: |
    K-ERP REST API v1

    ## Deprecation Notice
    This version will be sunset on 2025-12-01.
    Please migrate to v2.
  x-api-status: deprecated
  x-sunset-date: "2025-12-01"

servers:
  - url: https://api.k-erp.io/api/v1
    description: Production v1

# openapi/v2.yaml
openapi: 3.0.3
info:
  title: K-ERP API
  version: "2.0.0"
  description: K-ERP REST API v2 (Current)
  x-api-status: stable

servers:
  - url: https://api.k-erp.io/api/v2
    description: Production v2
```

## 7. 버전 테스트 전략

### 7.1 버전 호환성 테스트
```go
// internal/handler/v1/voucher_test.go
package v1_test

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestV1VoucherResponse_BackwardCompatibility(t *testing.T) {
    // v1 클라이언트가 기대하는 응답 구조
    type ExpectedV1Response struct {
        ID          string  `json:"id"`
        VoucherNo   string  `json:"voucher_no"`
        VoucherDate string  `json:"voucher_date"`
        TotalDebit  float64 `json:"total_debit"`
        TotalCredit float64 `json:"total_credit"`
        Status      string  `json:"status"`
        CreatedAt   string  `json:"created_at"`
    }

    // API 호출
    req := httptest.NewRequest(http.MethodGet, "/api/v1/vouchers/"+testVoucherID, nil)
    req.Header.Set("Authorization", "Bearer "+testToken)

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    require.Equal(t, http.StatusOK, w.Code)

    // v1 형식으로 파싱 가능해야 함
    var response ExpectedV1Response
    err := json.Unmarshal(w.Body.Bytes(), &response)
    require.NoError(t, err, "Response should be parseable as v1 format")

    // 필수 필드 존재 확인
    assert.NotEmpty(t, response.ID)
    assert.NotEmpty(t, response.VoucherNo)
    assert.NotEmpty(t, response.VoucherDate)
    assert.NotEmpty(t, response.Status)
}

func TestV1VoucherResponse_NewFieldsIgnored(t *testing.T) {
    // v1.5.0에서 추가된 tags 필드가 있어도 v1 클라이언트 동작에 영향 없음
    type OldV1Client struct {
        ID     string `json:"id"`
        Status string `json:"status"`
        // tags 필드 모름
    }

    req := httptest.NewRequest(http.MethodGet, "/api/v1/vouchers/"+testVoucherID, nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    var response OldV1Client
    err := json.Unmarshal(w.Body.Bytes(), &response)
    require.NoError(t, err, "Old client should ignore unknown fields")
}
```

### 7.2 계약 테스트 (Contract Testing)
```go
// test/contract/v1_contract_test.go
package contract_test

import (
    "testing"

    "github.com/pact-foundation/pact-go/v2/consumer"
)

func TestVoucherAPIContract_V1(t *testing.T) {
    mockProvider, err := consumer.NewV4Pact(consumer.MockHTTPProviderConfig{
        Consumer: "web-frontend",
        Provider: "k-erp-api-v1",
    })
    require.NoError(t, err)

    // v1 계약 정의
    err = mockProvider.
        AddInteraction().
        Given("a voucher exists").
        UponReceiving("a request to get voucher").
        WithRequest("GET", "/api/v1/vouchers/123").
        WillRespondWith(200).
        WithJSONBody(map[string]interface{}{
            "id":           consumer.Like("123"),
            "voucher_no":   consumer.Like("V-2025-0001"),
            "voucher_date": consumer.Regex("2025-01-15", `\d{4}-\d{2}-\d{2}`),
            "total_debit":  consumer.Like(10000.0),
            "total_credit": consumer.Like(10000.0),
            "status":       consumer.Term("approved", "draft|approved|posted"),
        }).
        ExecuteTest(t, func(config consumer.MockServerConfig) error {
            // 테스트 실행
            client := NewAPIClient(config.URL)
            voucher, err := client.GetVoucher("123")
            assert.NoError(t, err)
            assert.Equal(t, "approved", voucher.Status)
            return nil
        })
}
```

## 8. 클라이언트 SDK 버전 관리

### 8.1 SDK 구조
```
k-erp-sdk/
├── go/
│   ├── v1/           # v1 SDK
│   │   ├── client.go
│   │   └── types.go
│   └── v2/           # v2 SDK
│       ├── client.go
│       └── types.go
├── typescript/
│   ├── v1/
│   └── v2/
└── python/
    ├── v1/
    └── v2/
```

### 8.2 TypeScript SDK 예시
```typescript
// sdk/typescript/v2/client.ts
import { Configuration, VouchersApi, VoucherResponse } from './generated';

export class KERPClient {
  private vouchersApi: VouchersApi;

  constructor(config: { baseUrl: string; accessToken: string }) {
    const configuration = new Configuration({
      basePath: config.baseUrl + '/api/v2',  // v2 고정
      accessToken: config.accessToken,
    });

    this.vouchersApi = new VouchersApi(configuration);
  }

  async getVoucher(id: string, options?: { includeLines?: boolean }): Promise<VoucherResponse> {
    const response = await this.vouchersApi.getVoucher({
      id,
      include: options?.includeLines ? 'lines' : undefined,
    });
    return response;
  }
}

// sdk/typescript/v1/client.ts (deprecated)
/**
 * @deprecated Use v2 client instead. This will be removed on 2025-12-01.
 */
export class KERPClientV1 {
  // ...
}
```

## 9. 버전 모니터링

### 9.1 버전별 사용량 메트릭
```go
// internal/middleware/metrics.go
var (
    apiRequestsByVersion = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "kerp_api_requests_by_version_total",
            Help: "Total API requests by version",
        },
        []string{"version", "endpoint", "method", "status"},
    )

    deprecatedAPIUsage = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "kerp_deprecated_api_usage_total",
            Help: "Usage of deprecated API endpoints",
        },
        []string{"endpoint", "client_id"},
    )
)

func VersionMetrics() gin.HandlerFunc {
    return func(c *gin.Context) {
        version := extractVersion(c.Request.URL.Path)

        c.Next()

        apiRequestsByVersion.WithLabelValues(
            version,
            c.Request.URL.Path,
            c.Request.Method,
            fmt.Sprintf("%d", c.Writer.Status()),
        ).Inc()
    }
}
```

### 9.2 Grafana 대시보드
```yaml
# grafana/dashboards/api-versions.json
{
  "title": "API Version Usage",
  "panels": [
    {
      "title": "Requests by Version",
      "type": "piechart",
      "targets": [
        {
          "expr": "sum(rate(kerp_api_requests_by_version_total[5m])) by (version)"
        }
      ]
    },
    {
      "title": "Deprecated API Usage Trend",
      "type": "timeseries",
      "targets": [
        {
          "expr": "sum(rate(kerp_deprecated_api_usage_total[5m])) by (endpoint)"
        }
      ]
    },
    {
      "title": "Top Deprecated API Users",
      "type": "table",
      "targets": [
        {
          "expr": "topk(10, sum(kerp_deprecated_api_usage_total) by (client_id))"
        }
      ]
    }
  ]
}
```

## 10. 버전 마이그레이션 가이드

### 10.1 v1 → v2 마이그레이션 체크리스트
```markdown
## V1 → V2 Migration Checklist

### 필수 변경사항

- [ ] Base URL 변경: `/api/v1` → `/api/v2`
- [ ] `amount` 필드 처리 변경
  ```typescript
  // Before
  const total = voucher.total_debit;

  // After
  const total = parseFloat(voucher.amount.debit);
  ```
- [ ] 에러 응답 형식 변경 (RFC 7807)
  ```typescript
  // Before
  if (error.message) { ... }

  // After
  if (error.detail) { ... }  // RFC 7807 Problem Details
  ```

### 선택적 개선사항

- [ ] `include=lines` 활용하여 N+1 문제 해결
- [ ] `PATCH` 메서드 활용하여 부분 업데이트
- [ ] 배치 API 활용 (`POST /vouchers/batch`)
- [ ] HATEOAS `_links` 활용

### 테스트 체크리스트

- [ ] 전표 생성 테스트
- [ ] 전표 조회 테스트 (include 옵션 포함)
- [ ] 전표 수정 테스트 (PUT/PATCH)
- [ ] 에러 처리 테스트
- [ ] 페이지네이션 테스트
```

## 11. 관련 문서

| 문서 | 설명 |
|------|------|
| [12_API_설계_상세.md](./12_API_설계_상세.md) | API 설계 원칙 |
| [18_에러_핸들링_가이드.md](./18_에러_핸들링_가이드.md) | RFC 7807 에러 형식 |
| [11_CI_CD_파이프라인.md](./11_CI_CD_파이프라인.md) | API 배포 파이프라인 |
| [17_테스트_전략.md](./17_테스트_전략.md) | API 테스트 전략 |

---

*문서 버전: 1.0.0*
*최종 수정: 2025년 1월*
*작성자: K-ERP 개발팀*
