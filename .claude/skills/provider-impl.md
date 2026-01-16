# Provider Pattern Implementation

외부 서비스 연동을 위한 Provider 패턴을 구현합니다.
세금계산서(Scraper/NTS/Popbill), 4대보험 EDI 등에 사용됩니다.

## Trigger
`/provider <type> <name>` 또는 `/provider`

## Arguments
- `type`: invoice (세금계산서), insurance (4대보험)
- `name`: provider 이름 (scraper, nts, popbill, nps, nhis, etc.)

## Process

### 1. Provider Interface 정의
```
internal/provider/${type}/interface.go
```
```go
package ${type}

import (
    "context"
    "github.com/kerp/internal/domain"
)

// Provider 인터페이스
type Provider interface {
    Name() string
    Priority() int
    IsAvailable(ctx context.Context) bool
    // Type-specific methods
}

// InvoiceProvider 세금계산서 Provider
type InvoiceProvider interface {
    Provider
    Issue(ctx context.Context, req *IssueRequest) (*IssueResponse, error)
    Search(ctx context.Context, req *SearchRequest) ([]*domain.Invoice, error)
    GetDetail(ctx context.Context, ntsNum string) (*domain.Invoice, error)
}

// InsuranceProvider 4대보험 Provider
type InsuranceProvider interface {
    Provider
    SubmitReport(ctx context.Context, req *ReportRequest) (*ReportResponse, error)
    GetStatus(ctx context.Context, reportID string) (*StatusResponse, error)
}
```

### 2. Provider Chain 구현
```
internal/provider/${type}/chain.go
```
```go
package ${type}

import (
    "context"
    "sort"
    "go.uber.org/zap"
)

// ProviderChain 우선순위 기반 Provider 실행
type ProviderChain struct {
    providers []Provider
    logger    *zap.Logger
}

func NewProviderChain(providers []Provider, logger *zap.Logger) *ProviderChain {
    // 우선순위 정렬 (낮을수록 우선)
    sort.Slice(providers, func(i, j int) bool {
        return providers[i].Priority() < providers[j].Priority()
    })
    return &ProviderChain{providers: providers, logger: logger}
}

func (c *ProviderChain) Execute(ctx context.Context, fn func(Provider) error) error {
    var lastErr error

    for _, p := range c.providers {
        if !p.IsAvailable(ctx) {
            c.logger.Info("provider not available", zap.String("name", p.Name()))
            continue
        }

        c.logger.Info("trying provider",
            zap.String("name", p.Name()),
            zap.Int("priority", p.Priority()))

        if err := fn(p); err != nil {
            c.logger.Warn("provider failed",
                zap.String("name", p.Name()),
                zap.Error(err))
            lastErr = err
            continue
        }

        return nil
    }

    return fmt.Errorf("all providers failed: %w", lastErr)
}
```

### 3. 개별 Provider 구현
```
internal/provider/${type}/${name}.go
```
```go
package ${type}

type ${Name}Provider struct {
    client  *grpc.${Name}Client  // or HTTP client
    logger  *zap.Logger
    config  *${Name}Config
}

func New${Name}Provider(client *grpc.${Name}Client, cfg *${Name}Config, logger *zap.Logger) *${Name}Provider {
    return &${Name}Provider{
        client: client,
        config: cfg,
        logger: logger,
    }
}

func (p *${Name}Provider) Name() string { return "${name}" }
func (p *${Name}Provider) Priority() int { return ${priority} }

func (p *${Name}Provider) IsAvailable(ctx context.Context) bool {
    if err := p.client.HealthCheck(ctx); err != nil {
        p.logger.Warn("${name} health check failed", zap.Error(err))
        return false
    }
    return true
}

// Implement type-specific methods...
```

## Provider Priority (Invoice)

| Provider | Priority | Cost | Note |
|----------|----------|------|------|
| Scraper (Python) | 1 | Free | 홈택스 스크래핑 |
| NTS API | 2 | Free | 국세청 API |
| Popbill | 3 | Paid | ASP (Fallback) |

## Provider Priority (Insurance)

| Provider | Priority | Agency | Note |
|----------|----------|--------|------|
| NPS | 1 | 국민연금 | SEED-CBC |
| NHIS | 2 | 건강보험 | ARIA-CBC |
| EI | 3 | 고용보험 | SEED-CBC |
| WCI | 4 | 산재보험 | SEED-CBC |

## Output Structure
```
internal/provider/${type}/
├── interface.go      # Provider 인터페이스
├── chain.go          # Provider Chain
├── ${name}.go        # 개별 Provider
└── config.go         # 설정
```
