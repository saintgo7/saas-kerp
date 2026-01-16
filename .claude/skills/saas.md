# K-ERP SaaS Development Orchestrator

8-Phase 병렬 개발 환경을 관리하는 메인 스킬입니다.

## Trigger

`/saas` or `/saas {phase}` or `/saas status`

## Commands

| Command | Description |
|---------|-------------|
| `/saas` | 전체 상태 표시 및 가이드 |
| `/saas 1` ~ `/saas 8` | 특정 Phase로 진입 (브랜치 자동 전환) |
| `/saas status` | 모든 Phase 상태 확인 |
| `/saas deploy` | 원격 서버 배포 |
| `/saas sync` | Git 동기화 |
| `/saas merge` | 현재 Phase 브랜치를 develop에 merge |
| `/saas update` | develop에서 최신 변경사항 가져오기 |
| `/saas done` | Phase 완료 처리 (merge + status update) |

## Phase Entry

When `/saas {phase_number}` is called:

1. Read `.claude/phases/phase-config.json`
2. Load the corresponding agent
3. Show phase-specific context and tasks
4. Set focus to phase files

### Phase Quick Reference

```
Phase | Agent  | Focus                          | Terminal
------|--------|--------------------------------|---------
  1   | @ops   | Docker, CI/CD, Nginx, SSL      | T1
  2   | @db    | PostgreSQL, Migrations, RLS    | T2
  3   | @go    | Router, Middleware, Auth       | T3
  4   | @acc   | Accounting, Voucher, Invoice   | T4
  5   | @tax   | Hometax, gRPC, SEED, Playwright| T5
  6   | @hr    | 4대보험, EDI, ARIA, PKCS7      | T6
  7   | @react | React, TypeScript, Vite, UI    | T7
  8   | test   | Unit, E2E, Security, Perf      | T8
```

## Actions

### On `/saas` (no args)

Display full dashboard:

```
================================================================================
                    K-ERP SaaS v0.2 - Development Dashboard
================================================================================

CURRENT STATUS
--------------------------------------------------------------------------------
Phase | Name                  | Agent  | Status    | Last Updated
--------------------------------------------------------------------------------
  1   | Infrastructure        | @ops   | [status]  | [date]
  2   | Database & Schema     | @db    | [status]  | [date]
  3   | Go Core API           | @go    | [status]  | [date]
  4   | Go Business Modules   | @acc   | [status]  | [date]
  5   | Python Tax Scraper    | @tax   | [status]  | [date]
  6   | Python Insurance EDI  | @hr    | [status]  | [date]
  7   | Frontend              | @react | [status]  | [date]
  8   | Integration & Testing | test   | [status]  | [date]
--------------------------------------------------------------------------------

REMOTE SERVER: wsl-48-246 (61.245.248.246:5022)
DOMAIN: erp.abada.kr

QUICK START
--------------------------------------------------------------------------------
/saas 1        - Infrastructure (Docker, CI/CD)
/saas 2        - Database (Schema, Migrations)
/saas 3        - Go Core (Auth, Middleware)
/saas 4        - Go Business (Accounting)
/saas 5        - Tax Scraper (Python, Hometax)
/saas 6        - Insurance EDI (Python, 4대보험)
/saas 7        - Frontend (React, TypeScript)
/saas 8        - Testing (Unit, E2E, Security)
--------------------------------------------------------------------------------

AVAILABLE SKILLS
--------------------------------------------------------------------------------
/api {name}     - Go REST API scaffolding
/grpc {name}    - gRPC Proto generation
/provider {name}- Provider pattern implementation
/sqlc {name}    - sqlc query generation
/py {name}      - Python gRPC service
/test {name}    - Test code generation
/db {name}      - DB migration
/deploy {env}   - Deploy to environment
--------------------------------------------------------------------------------
```

### On `/saas {phase_number}`

1. **Switch to phase branch**: `git checkout phase/{n}-{name}`
2. **Pull latest**: `git pull origin develop --rebase` (if needed)
3. Update phase-config.json status to "active"
4. Load agent personality from `.claude/agents/{agent}.md`
5. Display phase-specific tasks from DEVELOPMENT_PLAN.md
6. Show relevant files and directories
7. Enter phase-focused mode

**Branch per Phase:**
```
Phase 1 -> phase/1-infra
Phase 2 -> phase/2-database
Phase 3 -> phase/3-go-core
Phase 4 -> phase/4-go-biz
Phase 5 -> phase/5-tax
Phase 6 -> phase/6-edi
Phase 7 -> phase/7-frontend
Phase 8 -> phase/8-testing
```

Example output for `/saas 3`:

```
================================================================================
                    PHASE 3: Go Core API - Active
================================================================================
Agent: @go (Go Backend Specialist)
Terminal: T3
Status: ACTIVE

PENDING TASKS (from DEVELOPMENT_PLAN.md)
--------------------------------------------------------------------------------
[ ] 3.1 Project structure setup (P0)
[ ] 3.2 Gin router configuration (P0)
[ ] 3.3 JWT authentication middleware (P0)
[ ] 3.4 Tenant middleware (company_id) (P0)
[ ] 3.5 GORM configuration (P1)
[ ] 3.6 sqlc integration (P1)
[ ] 3.7 Error handling middleware (P1)
[ ] 3.8 Request validation (P1)
[ ] 3.9 Logging (zap) (P2)
[ ] 3.10 gRPC client setup (P2)
--------------------------------------------------------------------------------

FOCUS DIRECTORIES
--------------------------------------------------------------------------------
cmd/api/
internal/middleware/
internal/handler/
--------------------------------------------------------------------------------

RELATED SKILLS: /api, /test
================================================================================

Ready to work on Phase 3. What task would you like to start?
```

### On `/saas deploy`

Execute deployment sequence:

```bash
# 1. Check local changes
git status

# 2. Commit if needed (prompt user)

# 3. Push to remote
git push origin main

# 4. Deploy to server
ssh wsl-48-246 "cd ~/saas-kerp && git pull && docker compose up -d --build"

# 5. Check status
ssh wsl-48-246 "cd ~/saas-kerp && docker compose ps"
```

### On `/saas sync`

Sync development status:

```bash
# 1. Fetch remote status
ssh wsl-48-246 "cd ~/saas-kerp && docker compose ps"

# 2. Update phase-config.json with service status

# 3. Display sync result
```

### On `/saas merge`

Merge current phase branch to develop:

```bash
# 1. Commit any pending changes
git add -A && git commit -m "feat(phase-N): work in progress"

# 2. Switch to develop and pull latest
git checkout develop
git pull origin develop

# 3. Merge phase branch
git merge phase/{n}-{name} --no-ff -m "Merge phase/{n}-{name} into develop"

# 4. Push develop
git push origin develop

# 5. Switch back to phase branch
git checkout phase/{n}-{name}

# 6. Rebase on develop for future work
git rebase develop
```

### On `/saas update`

Pull latest changes from develop:

```bash
# 1. Stash any uncommitted changes
git stash

# 2. Fetch and rebase on develop
git fetch origin develop
git rebase origin/develop

# 3. Pop stashed changes
git stash pop
```

### On `/saas done`

Mark phase as complete and merge:

```bash
# 1. Run tests for this phase
# 2. Commit all changes
# 3. Merge to develop
# 4. Update phase-config.json status to "done"
# 5. Notify other phases (optional)
```

## Multi-Terminal Setup Guide

To run 8 terminals simultaneously:

```bash
# Terminal 1 - Infrastructure
cd ~/01_DEV/SaaS_erp_clone-260116 && claude
# Then type: /saas 1

# Terminal 2 - Database
cd ~/01_DEV/SaaS_erp_clone-260116 && claude
# Then type: /saas 2

# Terminal 3 - Go Core
cd ~/01_DEV/SaaS_erp_clone-260116 && claude
# Then type: /saas 3

# ... and so on for T4-T8
```

## Rules

1. All responses in Korean (code comments in English)
2. No emojis allowed
3. Update phase-config.json when status changes
4. Always check dependencies before starting a phase
5. Coordinate with other phases through Git commits

## Phase Dependencies

```
Phase 1 (Infra) + Phase 2 (DB) --> Phase 3 (Go Core)
Phase 3 --> Phase 4, 5, 6, 7
All Phases --> Phase 8 (Testing)
```
