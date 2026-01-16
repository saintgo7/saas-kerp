# K-ERP SaaS v0.2 Development Environment

## STARTUP INSTRUCTIONS

**Claude 시작 시 자동 실행:**
1. 현재 Git 브랜치 확인: `git branch --show-current`
2. 브랜치가 `phase/*`이면 해당 Phase 모드로 자동 진입
3. 브랜치가 `develop`이면 대시보드 표시

**Phase 진입 명령 (다음 중 아무거나 인식):**
- `saas 1` ~ `saas 8`
- `phase 1` ~ `phase 8`
- `1번` ~ `8번`
- `Phase 1 시작`

**Phase 진입 시 자동 실행:**
```bash
git checkout phase/{n}-{name}
```

---

## Phase Configuration

| Phase | Branch | Agent | Focus |
|-------|--------|-------|-------|
| 1 | `phase/1-infra` | @ops | Docker, CI/CD, Nginx, SSL |
| 2 | `phase/2-database` | @db | PostgreSQL, Migrations, RLS |
| 3 | `phase/3-go-core` | @go | Router, Middleware, Auth |
| 4 | `phase/4-go-biz` | @acc | Accounting, Voucher, Invoice |
| 5 | `phase/5-tax` | @tax | Hometax, gRPC, SEED, Playwright |
| 6 | `phase/6-edi` | @hr | 4대보험, EDI, ARIA, PKCS7 |
| 7 | `phase/7-frontend` | @react | React, TypeScript, Vite, shadcn |
| 8 | `phase/8-testing` | test | Unit, E2E, Security, Performance |

## Phase Commands

| Command | Action |
|---------|--------|
| `saas` / `status` | 전체 상태 표시 |
| `saas {n}` / `phase {n}` | Phase N 진입 + 브랜치 전환 |
| `merge` / `saas merge` | 현재 Phase를 develop에 merge |
| `update` / `saas update` | develop에서 최신 가져오기 |
| `done` / `saas done` | Phase 완료 처리 |
| `deploy` | 원격 서버 배포 |

---

## Project Overview

| Item | Value |
|------|-------|
| Name | K-ERP SaaS Platform |
| Version | 0.2.0 |
| Architecture | Go + Python Hybrid |
| Target | Korean SMBs |

## Remote Server

```
Host: wsl-48-246 (61.245.248.246:5022)
User: blackpc
Path: /home/blackpc/saas-kerp
Domain: erp.abada.kr
```

## Service Ports

| Service | Port |
|---------|------|
| Go API | 8080 |
| Tax Scraper (Python) | 50051 |
| Insurance EDI (Python) | 50052 |
| PostgreSQL | 5432 |
| Redis | 6379 |
| NATS | 4222 |

## Directory Ownership (Conflict Prevention)

```
Phase 1: deployments/, .github/, docker-compose*.yml
Phase 2: db/migrations/, db/queries/, db/seed/
Phase 3: cmd/api/, internal/middleware/, internal/handler/, go.mod, go.sum
Phase 4: internal/domain/, internal/service/, internal/repository/, internal/dto/
Phase 5: python-services/tax-scraper/, python-services/shared/proto/
Phase 6: python-services/insurance-edi/, python-services/shared/crypto/
Phase 7: web/, package.json
Phase 8: tests/, **/*_test.go, **/*_test.py
```

## Skills Reference

| Command | Description |
|---------|-------------|
| `api {name}` | Go REST API scaffolding |
| `grpc {name}` | gRPC Proto/code generation |
| `provider {name}` | Provider pattern implementation |
| `sqlc {name}` | sqlc query generation |
| `py {name}` | Python gRPC service |
| `test {name}` | Test code generation |
| `db {name}` | DB migration |

## Rules

1. All responses in Korean (code comments in English)
2. No emojis allowed
3. Multi-tenancy: All tables must have `company_id`
4. Double-entry: Voucher debit must equal credit
5. Security: SEED/ARIA encryption for government systems
6. Testing: 80%+ coverage required
7. Branch per Phase: 각 Phase는 자신의 브랜치에서 작업

## Git Workflow

```
main (production)
  └── develop (integration)
        ├── phase/1-infra
        ├── phase/2-database
        ├── phase/3-go-core
        ├── phase/4-go-biz
        ├── phase/5-tax
        ├── phase/6-edi
        ├── phase/7-frontend
        └── phase/8-testing
```

**Merge Order:**
1. Phase 1, 2 완료 -> develop merge
2. Phase 3 시작 가능
3. Phase 3 진행 중 -> Phase 4, 5, 6, 7 병렬 시작 가능
4. 모든 Phase 완료 -> Phase 8 통합 테스트
5. develop -> main (릴리즈)

## Documentation

- [Development Plan](docs/DEVELOPMENT_PLAN.md)
- [Phase Config](.claude/phases/phase-config.json)
- [Architecture](docs/claude/01_아키텍처_설계.md)
