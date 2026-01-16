# K-ERP Claude Code Configuration

K-ERP SaaS 프로젝트를 위한 Claude Code Skills/Agents 설정입니다.

---

## Quick Reference

### Skills (슬래시 명령어)
| 명령어 | 용도 | 예시 |
|--------|------|------|
| `/api` | Go REST API 스캐폴딩 | `/api partners` |
| `/grpc` | gRPC Proto/코드 생성 | `/grpc tax-scraper` |
| `/provider` | Provider 패턴 구현 | `/provider invoice` |
| `/sqlc` | sqlc 쿼리 생성 | `/sqlc voucher-summary` |
| `/py` | Python gRPC 서비스 | `/py edi-sender` |
| `/test` | 테스트 코드 생성 | `/test voucher-service` |
| `/db` | DB 마이그레이션 | `/db add-partners-table` |
| `/qf` | 빠른 기능 구현 | `/qf login-api` |
| `/deploy` | WSL 배포 | `/deploy staging` |

### Agents (도메인 전문가)
| 호출 | 전문 분야 | 사용 시점 |
|------|----------|----------|
| `@go` | Go 개발 | Go 코드 작성/리뷰 |
| `@py` | Python gRPC | 스크래퍼/EDI 개발 |
| `@react` | React 프론트엔드 | UI 컴포넌트 개발 |
| `@acc` | 회계 도메인 | 전표/원장/재무제표 |
| `@tax` | 세금계산서 | 홈택스/Popbill 연동 |
| `@hr` | 인사/급여 | 4대보험/급여계산 |
| `@db` | DB 설계 | 스키마/인덱스/쿼리 최적화 |
| `@ops` | DevOps | Docker/K8s 배포 |

---

## Skills 상세 가이드

### /api - Go REST API 스캐폴딩

Clean Architecture 패턴으로 전체 API 레이어를 생성합니다.

```
/api partners
```

**생성되는 파일:**
```
internal/
├── domain/partner.go           # 도메인 모델
├── dto/partner_dto.go          # Request/Response DTO
├── repository/partner_repo.go  # 데이터 접근 계층
├── service/partner_service.go  # 비즈니스 로직
└── handler/partner_handler.go  # HTTP 핸들러
```

**특징:**
- 모든 모델에 `company_id` 자동 포함 (멀티테넌시)
- Soft delete 패턴 적용 (`deleted_at`)
- GORM 태그 자동 생성

---

### /grpc - gRPC Proto/코드 생성

Go-Python 간 gRPC 통신을 위한 Proto 정의와 코드를 생성합니다.

```
/grpc tax-scraper
```

**생성되는 파일:**
```
proto/
└── tax_scraper.proto          # Proto 정의

internal/grpc/
└── tax_scraper_client.go      # Go 클라이언트

python/services/
└── tax_scraper_server.py      # Python 서버
```

**사용 예시:**
```go
// Go에서 Python 서비스 호출
client := grpc.NewTaxScraperClient(conn)
resp, err := client.ScrapInvoices(ctx, &pb.ScrapeRequest{
    BusinessNumber: "1234567890",
    StartDate:      "2024-01-01",
})
```

---

### /provider - Provider 패턴 구현

외부 서비스 연동을 위한 Provider Chain을 생성합니다.

```
/provider invoice
```

**생성 구조:**
```go
// Provider 인터페이스
type InvoiceProvider interface {
    Name() string
    IsAvailable(ctx context.Context) bool
    Issue(ctx context.Context, req *IssueRequest) (*IssueResponse, error)
}

// Provider Chain (우선순위 기반 폴백)
// 1. ScraperProvider (무료)
// 2. NTSAPIProvider (무료)
// 3. PopbillProvider (유료 - 최후의 수단)
```

---

### /sqlc - sqlc 쿼리 생성

복잡한 SQL 쿼리를 sqlc 형식으로 생성합니다.

```
/sqlc voucher-summary
```

**생성 위치:** `db/queries/voucher_summary.sql`

**예시 쿼리:**
```sql
-- name: GetVoucherSummary :many
SELECT
    v.voucher_date,
    COUNT(*) as voucher_count,
    SUM(v.total_debit) as total_amount
FROM vouchers v
WHERE v.company_id = $1
  AND v.voucher_date BETWEEN $2 AND $3
  AND v.deleted_at IS NULL
GROUP BY v.voucher_date
ORDER BY v.voucher_date;
```

---

### /py - Python gRPC 서비스

Python 기반 gRPC 서비스를 생성합니다.

```
/py edi-sender
```

**생성되는 파일:**
```
python/services/
├── edi_sender/
│   ├── __init__.py
│   ├── server.py           # gRPC 서버
│   ├── service.py          # 비즈니스 로직
│   └── encryption.py       # SEED/ARIA 암호화
└── requirements.txt        # 의존성 추가
```

**특징:**
- SEED-CBC, ARIA-CBC 암호화 (한국 정부 시스템용)
- Playwright 기반 웹 스크래핑
- gRPC 서버 템플릿

---

### /test - 테스트 코드 생성

Go/Python 테스트 코드를 생성합니다.

```
/test voucher-service
```

**Go 테스트 (Table-driven):**
```go
func TestVoucherService_Create(t *testing.T) {
    tests := []struct {
        name    string
        input   *CreateVoucherInput
        wantErr bool
    }{
        {"valid voucher", validInput, false},
        {"unbalanced voucher", unbalancedInput, true},
    }
    // ...
}
```

**Python 테스트 (pytest):**
```python
@pytest.mark.asyncio
async def test_scrape_invoices():
    service = TaxScraperService()
    result = await service.scrape("1234567890", "2024-01-01")
    assert result.success
```

---

## Agents 상세 가이드

### @acc - 회계 도메인 전문가

K-IFRS 복식부기, 전표, 원장, 재무제표를 담당합니다.

**활용 시점:**
- 계정과목 체계 설계
- 전표 검증 로직 구현
- 재무제표 쿼리 작성
- 차대변 균형 검증

**예시 질문:**
```
@acc 매출채권 대손충당금 설정 전표 분개 예시 알려줘
@acc 시산표 조회 쿼리 작성해줘
```

---

### @tax - 세금계산서 전문가

홈택스, Popbill, 국세청 API 연동을 담당합니다.

**활용 시점:**
- 세금계산서 발행/조회 API
- Provider Chain 구현
- 부가세 신고 자료 생성
- 사업자번호 검증

**예시 질문:**
```
@tax 역발행 세금계산서 처리 플로우 설명해줘
@tax Popbill API로 세금계산서 발행하는 코드 작성해줘
```

---

### @hr - 인사/급여 전문가

급여계산, 4대보험, 연말정산을 담당합니다.

**활용 시점:**
- 급여 계산 로직
- 4대보험 EDI 메시지 생성
- 소득세 원천징수 계산
- 연말정산 처리

**예시 질문:**
```
@hr 2024년 4대보험 요율로 급여 공제 계산해줘
@hr 국민연금 자격취득 EDI 메시지 생성 코드 작성해줘
```

---

### @db - DB 설계 전문가

PostgreSQL 스키마, 인덱스, 쿼리 최적화를 담당합니다.

**활용 시점:**
- 테이블 설계
- RLS 정책 작성
- 인덱스 최적화
- 쿼리 성능 분석

**예시 질문:**
```
@db partners 테이블 마이그레이션 작성해줘
@db 이 쿼리 EXPLAIN ANALYZE 결과 분석해줘
```

---

### @py - Python gRPC 전문가

Python 기반 스크래퍼/EDI 서비스를 담당합니다.

**활용 시점:**
- Playwright 스크래핑 구현
- gRPC 서버 개발
- SEED/ARIA 암호화
- 4대보험 EDI 연동

**예시 질문:**
```
@py 홈택스 로그인 스크래핑 코드 작성해줘
@py SEED-CBC 암호화로 EDI 메시지 서명하는 코드 작성해줘
```

---

## 워크플로우 예시

### 1. 새로운 API 엔드포인트 추가

```bash
# 1. API 스캐폴딩
/api purchase-orders

# 2. DB 마이그레이션 생성
/db create-purchase-orders-table

# 3. 복잡 쿼리 추가
/sqlc purchase-order-summary

# 4. 테스트 생성
/test purchase-order-service
```

### 2. 외부 서비스 연동 추가

```bash
# 1. Provider 패턴 구현
/provider bank-statement

# 2. gRPC 서비스 생성 (스크래핑용)
/grpc bank-scraper

# 3. Python 서비스 구현
/py bank-scraper
```

### 3. 회계 기능 개발

```bash
# 1. 도메인 전문가와 상담
@acc 매입매출장 조회 요구사항 정리해줘

# 2. DB 설계
@db 매입매출장 테이블 설계해줘

# 3. API 구현
/api purchase-sales-ledger

# 4. 쿼리 최적화
/sqlc purchase-sales-summary
```

---

## 설정 커스터마이징

### settings.json 구조

```json
{
  "project": {
    "name": "K-ERP SaaS",
    "version": "0.2.0",
    "language": "ko"
  },
  "shortcuts": {
    "skills": {
      "/api": "api-scaffold",
      "/custom": "my-custom-skill"
    },
    "agents": {
      "@custom": "my-custom-agent"
    }
  }
}
```

### 새 Skill 추가

1. `.claude/skills/my-skill.md` 파일 생성
2. `settings.json`의 `shortcuts.skills`에 추가
3. `/my-skill` 명령어로 사용

### 새 Agent 추가

1. `.claude/agents/my-agent.md` 파일 생성
2. `settings.json`의 `shortcuts.agents`에 추가
3. `@my-agent` 호출로 사용

---

## 파일 구조

```
.claude/
├── README.md              # 이 문서
├── settings.json          # 메인 설정
├── settings.local.json    # 로컬 오버라이드 (gitignore)
├── skills/                # Skill 정의
│   ├── api-scaffold.md
│   ├── grpc-gen.md
│   ├── provider-impl.md
│   ├── sqlc-query.md
│   ├── py-service.md
│   ├── test-gen.md
│   ├── db-migrate.md
│   ├── quick-feature.md
│   └── deploy-wsl.md
├── agents/                # Agent 정의
│   ├── go-fast.md
│   ├── react-fast.md
│   ├── python-grpc.md
│   ├── accounting.md
│   ├── tax-invoice.md
│   ├── hr-payroll.md
│   ├── db-design.md
│   └── devops-deploy.md
└── hooks/                 # Git hooks
    └── ...
```

---

## 주의사항

1. **멀티테넌시**: 모든 테이블에 `company_id` 필수
2. **복식부기**: 전표는 반드시 차대변 균형
3. **암호화**: 정부 시스템 연동 시 SEED/ARIA 사용
4. **Provider 우선순위**: 무료 서비스 우선, 유료는 폴백

---

## 관련 문서

- [프로젝트 아키텍처](../docs/claude/01_overview.md)
- [Go 서비스 가이드](../docs/claude/03_go_api_server.md)
- [Python 서비스 가이드](../docs/claude/04_python_grpc_services.md)
- [DB 설계 가이드](../docs/claude/05_database_design.md)
