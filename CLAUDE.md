# K-ERP SaaS v0.2 Development Environment

## Project Overview

| Item | Value |
|------|-------|
| Name | K-ERP SaaS Platform |
| Version | 0.2.0 |
| Architecture | Go + Python Hybrid |
| Target | Korean SMBs |

## Quick Start

```bash
# Check development status
/status

# Deploy to remote
/deploy staging
```

## 8 Phase Parallel Development

| Phase | Name | Agent | Terminal |
|-------|------|-------|----------|
| 1 | Infrastructure & DevOps | @ops | T1 |
| 2 | Database & Schema | @db | T2 |
| 3 | Go Core API | @go | T3 |
| 4 | Go Business Modules | @acc | T4 |
| 5 | Python Tax Scraper | @tax | T5 |
| 6 | Python Insurance EDI | @hr | T6 |
| 7 | Frontend | @react | T7 |
| 8 | Integration & Testing | - | T8 |

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

## Skills Reference

| Command | Description |
|---------|-------------|
| `/api {name}` | Go REST API scaffolding |
| `/grpc {name}` | gRPC Proto/code generation |
| `/provider {name}` | Provider pattern implementation |
| `/sqlc {name}` | sqlc query generation |
| `/py {name}` | Python gRPC service |
| `/test {name}` | Test code generation |
| `/db {name}` | DB migration |
| `/deploy {env}` | Deploy to environment |
| `/status` | Show development status |

## Agents Reference

| Agent | Specialty |
|-------|-----------|
| @go | Go development |
| @py | Python gRPC |
| @react | React frontend |
| @acc | Accounting domain |
| @tax | Tax invoice |
| @hr | HR/Payroll |
| @db | Database design |
| @ops | DevOps |

## Rules

1. All responses in Korean (code comments in English)
2. No emojis allowed
3. Multi-tenancy: All tables must have `company_id`
4. Double-entry: Voucher debit must equal credit
5. Security: SEED/ARIA encryption for government systems
6. Testing: 80%+ coverage required

## Documentation

- [Development Plan](docs/DEVELOPMENT_PLAN.md)
- [Architecture](docs/claude/01_아키텍처_설계.md)
- [API Design](docs/claude/12_API_설계_상세.md)
- [DB Schema](docs/claude/13_DB_스키마_상세.md)

## Remote Commands

```bash
# SSH connect
ssh wsl-48-246

# Deploy
ssh wsl-48-246 "cd ~/saas-kerp && git pull && docker compose up -d --build"

# Logs
ssh wsl-48-246 "cd ~/saas-kerp && docker compose logs -f"

# Status
ssh wsl-48-246 "cd ~/saas-kerp && docker compose ps"
```
