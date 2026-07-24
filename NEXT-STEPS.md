<!-- K-ERP SaaS(Go+Python 멀티테넌트 ERP)의 P1 우선순위 수정 계획 -->
# NEXT-STEPS — K-ERP SaaS Clone (saas-kerp-clone)

> 작성: 2026-07-02 (P1 production-safe pass). 브랜치 `claude/maxburn-cleanup-2026-06-14`.
> 기준 사실: `STATUS.md`(2026-06-14) + 이번 점검(직접 확인). 미검증 항목은 그대로 표기한다.

## 이번 점검에서 직접 확인한 사실 (Go 1.26, darwin/arm64)
- `go build ./...` → **성공 (exit 0)**.
- `go test ./internal/domain/...` → **통과 (ok)**.
- **tax_invoice는 이미 와이어링됨** — `internal/handler/handlers.go`가 repo/service/handler를 생성하고 `internal/router/v1.go:60`이 `h.TaxInvoice.RegisterRoutes(tenant)` 등록. (STATUS.md의 "미연결" 기술은 이후 커밋으로 해소됨.)
- **department는 여전히 미와이어링** — `internal/service/department_service.go`·`internal/repository/department_repository*.go`·`internal/domain/department.go`는 존재하나 `handlers.go`/`v1.go`에 `Department` 참조 0건.

## 🔴 P1-BLOCKER (보안) — 커밋 금지 자산
- **`docs/SERVER_ACCESS_GUIDE.md` (untracked)에 실제 OpenSSH PRIVATE KEY 2종이 평문 삽입돼 있다.**
  - `-----BEGIN OPENSSH PRIVATE KEY-----` 블록: `id_kerp`(kerp-deploy), `k-erp-servers`(k-erp-servers-key). 프로덕션 `erp.abada.kr`(61.245.248.246:5022) 접속 키.
  - 조치(이번 pass): git 커밋하지 않음. `.gitignore`에 `docs/SERVER_ACCESS_GUIDE.md`를 추가해 실수 커밋을 차단했다.
  - **필요 후속(heldForReview):** (a) 이 문서에서 private key 블록을 제거하고 키는 1Password/암호화 저장소로 이동, (b) `git log --all -- docs/SERVER_ACCESS_GUIDE.md`로 과거 유입 이력 확인, 유입 시 **해당 SSH 키 즉시 로테이션**. 히스토리 재작성/키 교체는 P1 프로덕션 pass 금지 범위.

## 우선순위 1 — department 도메인 라우트 연결 (기능 미완)
- 근본원인: service/repo/domain은 구현됐으나 `Handlers` 구조체와 라우터에 등록 누락. handler 파일도 부재.
- 필요 작업(다중 파일, 런타임 동작 변경 → **heldForReview**):
  1. `internal/handler/department_handler.go` 신규(tax_invoice_handler.go 패턴 준용: `RegisterRoutes`, Create/List/Get/Update/Delete).
  2. `internal/handler/handlers.go`에 `Department *DepartmentHandler` 필드 + repo/service 생성 + `NewHandlers` 반환에 추가.
  3. `internal/router/v1.go` tenant 그룹에 `h.Department.RegisterRoutes(tenant)` 추가.
  4. handler 단위 테스트 추가.

## 우선순위 2 — tax_invoice ↔ NTS(홈택스) 실연동
- 현재: `handlers.go`가 `service.NewTaxInvoiceService(taxInvoiceRepo, nil)`로 **gRPC 클라이언트에 nil**을 주입. `TransmitToNTS`는 nil일 때 fail-closed(커밋 30955ef)로 안전하지만, **실제 국세청 전송은 미연동**.
- 필요: `python-services/tax-scraper/`(SEED)와의 gRPC 경계 기동 → `grpcclient.TaxInvoiceClient` 생성해 주입. 통합은 `docker-compose.test.yml`/`make test-up` 환경에서 검증. (외부 의존·미검증, 프로덕션 pass 범위 밖.)

## 우선순위 3 — 프론트/파이썬 미검증 해소
- `web/`: node_modules 미설치 → `pnpm i && pnpm build` 상태 미검증.
- `python-services/`: venv 미구성 → pytest 미검증.
- `tests/`(통합/E2E/perf/security): `make test-up`으로 docker-compose 기동 후 실행 필요(외부 의존).

## 이번 pass에서 적용한 안전 변경
- `NEXT-STEPS.md` 신규(본 파일).
- `.gitignore`에 `docs/SERVER_ACCESS_GUIDE.md` 추가(개인키 포함 문서 실수 커밋 차단).
- 코드 변경 0. department 와이어링/NTS 연동은 다중 파일·런타임 변경이라 문서화만 함.

## 책/논문 가능성
HIGH — "한국 SMB용 멀티테넌트 ERP를 Go+Python 하이브리드로 설계/구현한 사례 연구". 절별 매핑은 STATUS.md §6 참조. 5~7장 코드는 미검증이므로 "구현됨/미검증" 구분 표기, 성능·커버리지 수치는 실측 후에만 기재(§11).
