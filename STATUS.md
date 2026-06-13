<!-- 프로젝트 현황·검증 결과·잔여 작업·논문 개요를 기록하는 스윕 산출물 -->
# STATUS — K-ERP SaaS Clone

> 자동 포트폴리오 스윕(claude/sweep-2026-06-13)이 생성/갱신한 현황 문서. 본문은 한국어, 마지막에 English summary.

## 1. 프로젝트 개요

- 이름: K-ERP SaaS Platform (한국 중소기업 대상 ERP, 더존 kERP 류 클론).
- 아키텍처: Go(Gin) 백엔드 + Python(gRPC) 세무·4대보험 서비스 + React/TypeScript(Vite) 프론트엔드의 하이브리드.
- 데이터: PostgreSQL 16 + Redis 7 + NATS JetStream (멀티테넌시, 모든 테이블 `company_id` 보유 설계).
- 핵심 도메인: 복식부기 전표(차변=대변), 계정과목, 원장, 세금계산서, 거래처, 프로젝트, 부서, 사용자/권한, 회사 설정.
- 개발 모델: `docs/DEVELOPMENT_PLAN.md` 기준 8개 Phase 병렬 개발(인프라/DB/Go코어/Go비즈/세무/EDI/프론트/테스트). 현재 작업 브랜치는 `phase/4-go-biz`였다.

## 2. 검증 결과 (이번 스윕에서 실제 실행)

- 환경: Go 1.26.3 (go.mod 선언은 go 1.21), darwin/arm64.
- `go build ./...` -> 성공 (exit 0, 의존성 다운로드 후 전체 빌드 통과).
- `go test ./...` -> 성공 (exit 0). 통과 패키지: `internal/domain`, `internal/handler`, `internal/middleware`, `internal/service`. 나머지 패키지는 테스트 파일 없음(no test files).
- Go 테스트 파일 8개 존재(도메인·핸들러·서비스 단위 테스트 중심).
- 프론트엔드(web/): `node_modules` 미설치 상태라 이번 스윕에서 빌드/테스트 미실행(시간 제한으로 설치 스킵). 상태 unverified.
- Python 서비스(python-services/): venv 미구성이라 pytest 미실행. 상태 unverified.

요약: **Go 백엔드는 빌드·단위 테스트가 모두 통과하는 건강한 상태.** 프론트/파이썬은 이번 스윕에서 검증하지 못했다(미설치).

## 3. 동작하는 것 vs 안 되는 것 (검증 근거 명시)

동작 확인됨 (검증됨):
- Go 전체 빌드 (`go build ./...` exit 0).
- Go 도메인/서비스/핸들러/미들웨어 단위 테스트 (`go test ./...` exit 0).
- 라우팅 구성: 인증(public), 인증필요(protected), 테넌트 스코프(tenant) 3단 그룹. 테넌트 라우트에 Account/Partner/Voucher/Ledger/User/Role/Company/Project 핸들러가 `RegisterRoutes`로 등록됨(`internal/router/v1.go`).
- DB 마이그레이션 13쌍(up/down) 존재(`db/migrations/`, extensions·uuid_v7·core_tables·partners_projects·accounting_tables 등).

미검증/미완성 (코드 검토로 확인된 갭):
- **세금계산서(tax_invoice) 미연결**: 도메인(`internal/domain/tax_invoice.go`)·서비스(`tax_invoice_service.go`)·핸들러(`tax_invoice_handler.go`)는 존재하나 `Handlers` 구조체와 라우터에 와이어링되어 있지 않다. `TaxInvoice.RegisterRoutes` 호출 없음(검증됨: grep 결과 NOT registered). Phase 5(세무) 미완 영역으로 보인다.
- **부서(department) 핸들러 부재**: `department_service.go`는 있으나 핸들러/라우트 등록이 없다(검증됨).
- 프론트엔드 빌드/테스트 결과 unverified (node_modules 미설치).
- Python 세무 스크래퍼·4대보험 EDI 서비스 동작 unverified (venv 미구성).
- 통합/E2E/성능/보안 테스트(`tests/`)는 docker-compose 기반 외부 서비스 의존이라 이번 스윕 범위 밖(미실행).

## 4. 끝내기까지 남은 구체 작업

1. tax_invoice를 `internal/handler/handlers.go`의 `Handlers` 구조체·`NewHandlers`·`v1.go` 테넌트 라우트에 와이어링. (repo->service->handler 체인이 이미 존재하므로 등록만 하면 되나, 테스트 추가 후 진행 권장 — 다중 파일 변경이라 이번 스윕에서는 보류.)
2. department 핸들러 추가 및 라우트 등록(서비스 계층은 이미 존재).
3. 프론트엔드: `npm ci` 후 `npm run build` / `vitest` 통과 확인. 백엔드 `/v1` prefix 정합성 재확인(최근 커밋들이 API_BASE_URL /v1 prefix를 다룸).
4. Python 서비스: venv 구성 후 `pytest` 통과 확인, gRPC proto 재생성 점검.
5. go.mod의 `go 1.21` 선언과 실제 빌드 환경(1.26) 정합성 검토(필수 아님, 빌드는 통과).
6. 통합/E2E/보안 테스트는 `make test-up`으로 테스트 서비스 기동 후 실행(외부 의존 필요).

## 5. 이번 스윕에서 변경한 것

- STATUS.md 신규 작성(본 파일).
- 코드 변경 없음. tax_invoice/department 와이어링은 다중 파일·미검증 위험으로 의도적으로 보류하고 문서화만 했다.
- 기존 WIP(pre-existing) 체크포인트 포함: `.serena/project.yml` 수정, `docs/SERVER_ACCESS_GUIDE.md`, `docs/html/`(dev-log HTML 리포트 산출물), `scripts/generate-*.py`·`scripts/parse-devlog.py`·`scripts/update-html.sh`(개발 로그 시각화 도구). 민감 파일 없음(스캔 결과 NONE).

## 6. 논문/책 집필 개요 (Book/Paper Outline)

이 프로젝트는 "한국 중소기업용 멀티테넌트 ERP를 Go+Python 하이브리드로 설계·구현한 사례 연구"로 기술 서적 또는 시스템 논문화가 가능하다. 현재 확보된 자료를 절별로 매핑한다.

- 1장. 서론 — 한국 SMB ERP 시장과 더존 kERP류 요구사항. (자료: `docs/DEVELOPMENT_PLAN.md` 1.1 Overview, 타깃 매출 10억~1000억 KRW.)
- 2장. 아키텍처 설계 — Go(Gin) + Python(gRPC) 하이브리드 선택 근거, 서비스 포트 분리. (자료: `docs/claude/01_아키텍처_설계.md`, CLAUDE.md Service Ports, DEVELOPMENT_PLAN 1.2 Tech Stack.)
- 3장. 멀티테넌시 데이터 모델 — 전 테이블 `company_id`, RLS 설계, 마이그레이션 전략. (자료: `db/migrations/` 13쌍, core_tables/accounting_tables.)
- 4장. 복식부기 회계 엔진 — 전표 차변=대변 불변식, 계정과목·원장 설계. (자료: `internal/domain/voucher.go`·`account.go`·`ledger.go`, `voucher_test.go`·`account_test.go` — 검증된 단위 테스트 존재.)
- 5장. API 계층과 인증/테넌트 미들웨어 — 3단 라우트 그룹(public/protected/tenant), JWT. (자료: `internal/router/v1.go`, `internal/middleware/`, 핸들러 테스트.)
- 6장. 정부 시스템 연동 — 홈택스 세무 스크래핑(SEED), 4대보험 EDI(ARIA/PKCS7), gRPC 경계. (자료: `python-services/tax-scraper/`, `python-services/insurance-edi/`. 단, 동작 unverified — 논문에 "구현 존재, 미검증"으로 정직 표기 필요.)
- 7장. 프론트엔드 — React/TS/Vite, shadcn 컴포넌트(65개 tsx). (자료: `web/src/`. 빌드 unverified.)
- 8장. 검증·테스트 전략 — 단위/통합/E2E/성능/보안 4층 테스트, 80% 커버리지 목표. (자료: `tests/`, `Makefile.test`, `docker-compose.test.yml`.)
- 9장. 8-Phase 병렬 개발 방법론 — 디렉토리 소유권 기반 충돌 방지, 브랜치 전략. (자료: CLAUDE.md Directory Ownership, Git Workflow, `docs/dev-log/`.)
- 10장. 결론·한계 — MVP 6개월 목표, 미완 영역(tax_invoice 미연결, 프론트/파이썬 미검증).

정직성 주의(§11): 5·6·7장은 코드가 존재하나 이번 스윕에서 실행 검증되지 않았다. 논문화 시 "구현됨/미검증"을 구분해 표기하고, 성능·커버리지 수치는 실제 측정 후에만 기재할 것. 시뮬/추정을 실측으로 적지 말 것.

---

## English Summary

K-ERP SaaS clone: a multi-tenant ERP for Korean SMBs, built as a Go(Gin) backend + Python(gRPC) tax/insurance services + React/TypeScript(Vite) frontend.

Verified this sweep (Go 1.26.3, darwin/arm64):
- `go build ./...` -> success (exit 0).
- `go test ./...` -> success (exit 0); domain, handler, middleware, service packages all pass; 8 Go test files.

Not verified this sweep (skipped to honor the time-box):
- Frontend (web/): no node_modules installed -> build/test status unverified.
- Python services: no venv -> pytest status unverified.
- Integration/E2E/perf/security tests under `tests/` depend on docker-compose services -> not run.

Known gaps found by code review:
- `tax_invoice` domain/service/handler exist but are NOT wired into the `Handlers` struct or routes (Phase 5 tax work incomplete).
- `department` service exists but has no handler/route registration.

Changes made this sweep: added this STATUS.md only. No code changed — tax_invoice/department wiring was deliberately deferred (multi-file, unverified risk) and documented instead. The commit also checkpoints pre-existing WIP (serena config edit, dev-log HTML report generator scripts and output, server access guide doc); no secrets detected.

Book/paper potential: HIGH — a case study on designing/implementing a multi-tenant Korean SMB ERP with a Go+Python hybrid; section-to-material mapping is in section 6 above. Sections 5-7 contain code but are unverified; mark them as such per integrity rules and never report unmeasured performance/coverage numbers.
