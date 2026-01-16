# Development Status Skill

현재 개발 환경 상태를 표시합니다.

## Trigger
`/status` or startup

## Actions

### 1. Show Phase Status

Read `.claude/phases/phase-config.json` and display:

```
================================================================================
                        K-ERP SaaS v0.2 Development Status
================================================================================

PHASES (8 Parallel Terminals)
--------------------------------------------------------------------------------
Phase | Name                  | Agent  | Terminal | Status    | Focus
--------------------------------------------------------------------------------
  1   | Infrastructure        | @ops   |    T1    | [status]  | Docker, CI/CD
  2   | Database & Schema     | @db    |    T2    | [status]  | PostgreSQL, RLS
  3   | Go Core API           | @go    |    T3    | [status]  | Auth, Middleware
  4   | Go Business Modules   | @acc   |    T4    | [status]  | Accounting, Invoice
  5   | Python Tax Scraper    | @tax   |    T5    | [status]  | Hometax, SEED
  6   | Python Insurance EDI  | @hr    |    T6    | [status]  | 4대보험, EDI
  7   | Frontend              | @react |    T7    | [status]  | React, TypeScript
  8   | Integration & Testing | test   |    T8    | [status]  | Tests, Security
--------------------------------------------------------------------------------

REMOTE SERVER (wsl-48-246)
--------------------------------------------------------------------------------
Host: 61.245.248.246:5022 | User: blackpc | Path: /home/blackpc/saas-kerp
Domain: erp.abada.kr

SERVICES
--------------------------------------------------------------------------------
Service         | Port  | Status
--------------------------------------------------------------------------------
Go API Server   | 8080  | [status]
Tax Scraper     | 50051 | [status]
Insurance EDI   | 50052 | [status]
PostgreSQL      | 5432  | [status]
Redis           | 6379  | [status]
NATS            | 4222  | [status]
--------------------------------------------------------------------------------

QUICK COMMANDS
--------------------------------------------------------------------------------
/api {name}      - Go REST API scaffolding
/db {name}       - Database migration
/grpc {name}     - gRPC Proto generation
/py {name}       - Python service
/test {name}     - Test generation
/deploy staging  - Deploy to staging
--------------------------------------------------------------------------------

AGENTS: @go @py @react @acc @tax @hr @db @ops
================================================================================
```

### 2. Check Remote Server

```bash
ssh wsl-48-246 "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'" 2>/dev/null
```

### 3. Update Status

When a phase task is completed, update the phase-config.json status.

## Output Format

- No emojis
- ASCII table format
- Clear status indicators: [pending], [active], [done], [error]
