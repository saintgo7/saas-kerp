# K-ERP SaaS v0.2 Development Plan

**Version**: 0.2.0
**Created**: 2026-01-17
**Status**: Active Development

---

## Table of Contents

1. [Overview](#1-overview)
2. [Development Environment](#2-development-environment)
3. [8 Phase Parallel Development](#3-8-phase-parallel-development)
4. [Security Checklist](#4-security-checklist)
5. [Terminal Assignment](#5-terminal-assignment)
6. [Commands Reference](#6-commands-reference)

---

## 1. Overview

### 1.1 Project Summary

| Item | Description |
|------|-------------|
| Project | K-ERP SaaS Platform |
| Target | Korean SMBs (Revenue 1B~100B KRW) |
| Architecture | Go + Python Hybrid |
| Duration | 12 months (MVP: 6 months) |

### 1.2 Tech Stack

```
Backend:   Go 1.22+ (Gin) + Python 3.11+ (gRPC)
Frontend:  React 18 + TypeScript + Vite
Database:  PostgreSQL 16 + Redis 7 + NATS JetStream
Container: Docker + Kubernetes
```

### 1.3 Service Ports

| Service | Port | Language |
|---------|------|----------|
| Go API Server | 8080 | Go |
| Tax Scraper | 50051 | Python |
| Insurance EDI | 50052 | Python |
| PostgreSQL | 5432 | - |
| Redis | 6379 | - |
| NATS | 4222 | - |

---

## 2. Development Environment

### 2.1 Local (Mac)

```bash
# Claude Code with 8 terminals
# Each terminal handles one phase
```

### 2.2 Remote Server (wsl-48-246)

| Item | Value |
|------|-------|
| Host | 61.245.248.246 |
| Port | 5022 |
| User | blackpc |
| Path | /home/blackpc/saas-kerp |
| Domain | erp.abada.kr |

**Server Specs:**
- OS: WSL2 Ubuntu
- RAM: 94GB
- Disk: 950GB available
- Docker: 29.1.4
- Docker Compose: 5.0.1
- Python: 3.12.3
- Go: Not installed (use Docker)

### 2.3 SSH Config

```bash
# ~/.ssh/config
Host wsl-48-246
    HostName 61.245.248.246
    Port 5022
    User blackpc
    IdentityFile ~/.ssh/id_kerp
```

---

## 3. 8 Phase Parallel Development

### Phase Overview

```
+------------------+------------------+------------------+------------------+
|    PHASE 1       |    PHASE 2       |    PHASE 3       |    PHASE 4       |
|   Infrastructure |    Database      |   Go Core API    |  Go Business     |
|   @ops           |    @db           |   @go            |  @acc            |
|   Terminal 1     |    Terminal 2    |   Terminal 3     |  Terminal 4      |
+------------------+------------------+------------------+------------------+
|    PHASE 5       |    PHASE 6       |    PHASE 7       |    PHASE 8       |
|   Tax Scraper    |  Insurance EDI   |    Frontend      |   Integration    |
|   @tax           |    @hr           |   @react         |  Testing         |
|   Terminal 5     |    Terminal 6    |   Terminal 7     |  Terminal 8      |
+------------------+------------------+------------------+------------------+
```

---

### PHASE 1: Infrastructure & DevOps

**Terminal**: 1
**Agent**: @ops
**Duration**: Week 1-2

#### Tasks

| ID | Task | Priority | Status |
|----|------|----------|--------|
| 1.1 | Docker Compose setup | P0 | Pending |
| 1.2 | Go installation on remote | P0 | Pending |
| 1.3 | CI/CD pipeline (GitHub Actions) | P1 | Pending |
| 1.4 | Nginx reverse proxy | P1 | Pending |
| 1.5 | SSL/TLS certificate (Let's Encrypt) | P1 | Pending |
| 1.6 | Monitoring setup (Prometheus/Grafana) | P2 | Pending |

#### Key Files

```
deployments/
├── docker/
│   ├── docker-compose.yml
│   ├── docker-compose.dev.yml
│   ├── Dockerfile.api
│   ├── Dockerfile.worker
│   └── nginx/
│       └── nginx.conf
└── k8s/
    ├── namespace.yaml
    ├── api-deployment.yaml
    └── services.yaml
```

#### Commands

```bash
# Deploy to remote
ssh wsl-48-246 "cd ~/saas-kerp && docker compose up -d"

# Check logs
ssh wsl-48-246 "cd ~/saas-kerp && docker compose logs -f api"
```

---

### PHASE 2: Database & Schema

**Terminal**: 2
**Agent**: @db
**Duration**: Week 1-3

#### Tasks

| ID | Task | Priority | Status |
|----|------|----------|--------|
| 2.1 | PostgreSQL 16 container setup | P0 | Pending |
| 2.2 | Multi-tenancy schema (RLS) | P0 | Pending |
| 2.3 | Core tables migration | P0 | Pending |
| 2.4 | Accounting tables | P1 | Pending |
| 2.5 | Tax invoice tables | P1 | Pending |
| 2.6 | HR/Payroll tables | P1 | Pending |
| 2.7 | Index optimization | P2 | Pending |
| 2.8 | Audit log trigger | P2 | Pending |

#### Schema Structure

```sql
-- Core Tables
companies, users, roles, permissions

-- Accounting
accounts, vouchers, voucher_entries, ledger_balances

-- Tax Invoice
invoices, invoice_items, scrape_jobs

-- HR/Payroll
departments, employees, employee_salaries, payrolls, insurance_reports
```

#### Commands

```bash
# Run migration
/db create-{table_name}

# Generate sqlc
/sqlc {query_name}
```

---

### PHASE 3: Go Core API

**Terminal**: 3
**Agent**: @go
**Duration**: Week 2-4

#### Tasks

| ID | Task | Priority | Status |
|----|------|----------|--------|
| 3.1 | Project structure setup | P0 | Pending |
| 3.2 | Gin router configuration | P0 | Pending |
| 3.3 | JWT authentication middleware | P0 | Pending |
| 3.4 | Tenant middleware (company_id) | P0 | Pending |
| 3.5 | GORM configuration | P1 | Pending |
| 3.6 | sqlc integration | P1 | Pending |
| 3.7 | Error handling middleware | P1 | Pending |
| 3.8 | Request validation | P1 | Pending |
| 3.9 | Logging (zap) | P2 | Pending |
| 3.10 | gRPC client setup | P2 | Pending |

#### Directory Structure

```
internal/
├── domain/          # Domain models
├── dto/             # Request/Response DTOs
├── repository/      # Data access layer
├── service/         # Business logic
├── handler/         # HTTP handlers
└── middleware/      # Auth, tenant, logging
```

#### Commands

```bash
# Scaffold new API
/api {resource_name}

# Generate tests
/test {service_name}
```

---

### PHASE 4: Go Business Modules

**Terminal**: 4
**Agent**: @acc
**Duration**: Week 3-8

#### Tasks

| ID | Task | Priority | Status |
|----|------|----------|--------|
| 4.1 | Account management API | P0 | Pending |
| 4.2 | Voucher CRUD | P0 | Pending |
| 4.3 | Double-entry validation | P0 | Pending |
| 4.4 | Ledger calculation | P1 | Pending |
| 4.5 | Trial balance report | P1 | Pending |
| 4.6 | Financial statements | P1 | Pending |
| 4.7 | Tax invoice API | P1 | Pending |
| 4.8 | Provider chain implementation | P1 | Pending |
| 4.9 | Partner management | P2 | Pending |
| 4.10 | Master data APIs | P2 | Pending |

#### Key Patterns

```go
// Double-entry validation (MUST)
if voucher.TotalDebit != voucher.TotalCredit {
    return ErrUnbalancedVoucher
}

// Provider chain (cost optimization)
providers := []Provider{
    scraperProvider,  // Free
    ntsAPIProvider,   // Free
    popbillProvider,  // Paid (fallback)
}
```

#### Commands

```bash
# Create provider implementation
/provider {service_name}

# Complex query
/sqlc {report_name}
```

---

### PHASE 5: Python Tax Scraper

**Terminal**: 5
**Agent**: @tax
**Duration**: Week 4-8

#### Tasks

| ID | Task | Priority | Status |
|----|------|----------|--------|
| 5.1 | Python project structure | P0 | Pending |
| 5.2 | gRPC server setup | P0 | Pending |
| 5.3 | Playwright browser setup | P0 | Pending |
| 5.4 | SEED-CBC encryption | P0 | Pending |
| 5.5 | Certificate handling | P0 | Pending |
| 5.6 | Hometax login | P1 | Pending |
| 5.7 | Invoice scraping | P1 | Pending |
| 5.8 | Invoice issuance | P1 | Pending |
| 5.9 | Error handling | P2 | Pending |
| 5.10 | Health check | P2 | Pending |

#### Directory Structure

```
python-services/
├── shared/
│   ├── proto/           # Generated gRPC code
│   └── crypto/          # SEED, ARIA encryption
└── tax-scraper/
    └── src/
        ├── main.py      # gRPC server entry
        ├── server.py    # ScraperServicer
        └── scraper/
            ├── hometax.py
            └── auth.py
```

#### Commands

```bash
# Create Python service
/py {service_name}

# Generate gRPC code
/grpc {service_name}
```

---

### PHASE 6: Python Insurance EDI

**Terminal**: 6
**Agent**: @hr
**Duration**: Week 5-10

#### Tasks

| ID | Task | Priority | Status |
|----|------|----------|--------|
| 6.1 | EDI message generator | P0 | Pending |
| 6.2 | ARIA-CBC encryption | P0 | Pending |
| 6.3 | PKCS#7 signing | P0 | Pending |
| 6.4 | NPS client (National Pension) | P1 | Pending |
| 6.5 | NHIS client (Health Insurance) | P1 | Pending |
| 6.6 | EI client (Employment Insurance) | P1 | Pending |
| 6.7 | WCI client (Industrial Accident) | P2 | Pending |
| 6.8 | Response parser | P2 | Pending |
| 6.9 | gRPC server | P1 | Pending |
| 6.10 | Integration tests | P2 | Pending |

#### EDI Message Format

```
Header (50 bytes)
├── Record Type (2)
├── Business Code (4)
├── Date (8)
├── Time (6)
├── Sender (10)
└── Company No (10)

Data (200 bytes)
├── Resident Number (13) - Encrypted
├── Name (30)
├── Date (8)
└── ...
```

#### Commands

```bash
# Create EDI service
/py {agency_name}-edi

# Test encryption
/test crypto-seed
```

---

### PHASE 7: Frontend

**Terminal**: 7
**Agent**: @react
**Duration**: Week 4-12

#### Tasks

| ID | Task | Priority | Status |
|----|------|----------|--------|
| 7.1 | Vite + React 18 setup | P0 | Pending |
| 7.2 | TypeScript configuration | P0 | Pending |
| 7.3 | shadcn/ui components | P0 | Pending |
| 7.4 | Authentication pages | P1 | Pending |
| 7.5 | Dashboard layout | P1 | Pending |
| 7.6 | Voucher management UI | P1 | Pending |
| 7.7 | Tax invoice UI | P1 | Pending |
| 7.8 | HR/Payroll UI | P2 | Pending |
| 7.9 | Report viewer | P2 | Pending |
| 7.10 | WebSocket integration | P2 | Pending |

#### Directory Structure

```
web/
├── src/
│   ├── components/
│   │   ├── ui/          # shadcn components
│   │   └── features/    # Feature components
│   ├── pages/
│   ├── hooks/
│   ├── services/        # API clients
│   ├── stores/          # State management
│   └── types/
├── package.json
└── vite.config.ts
```

---

### PHASE 8: Integration & Testing

**Terminal**: 8
**Agent**: Testing
**Duration**: Week 6-12

#### Tasks

| ID | Task | Priority | Status |
|----|------|----------|--------|
| 8.1 | Unit test setup (Go) | P0 | Pending |
| 8.2 | Unit test setup (Python) | P0 | Pending |
| 8.3 | Integration tests | P1 | Pending |
| 8.4 | E2E tests (Playwright) | P1 | Pending |
| 8.5 | Security audit | P0 | Pending |
| 8.6 | Performance testing | P2 | Pending |
| 8.7 | Load testing | P2 | Pending |
| 8.8 | API documentation | P1 | Pending |
| 8.9 | Coverage report | P2 | Pending |
| 8.10 | CI test automation | P1 | Pending |

#### Test Commands

```bash
# Run all Go tests
make test

# Run Python tests
pytest python-services/

# Security scan
make security-scan

# E2E tests
npm run test:e2e
```

---

## 4. Security Checklist

### 4.1 Authentication & Authorization

| Item | Requirement | Status |
|------|-------------|--------|
| JWT Token | Access: 15min, Refresh: 7days | Pending |
| MFA | TOTP support | Pending |
| Password | bcrypt hashing | Pending |
| RBAC | Role-based permissions | Pending |

### 4.2 Data Security

| Item | Requirement | Status |
|------|-------------|--------|
| TLS | 1.3 for all connections | Pending |
| mTLS | Go-Python gRPC | Pending |
| Encryption | AES-256 for PII | Pending |
| RLS | Row-level security | Pending |

### 4.3 Government Integration Security

| Item | Requirement | Status |
|------|-------------|--------|
| SEED-CBC | Hometax, NPS, EI, WCI | Pending |
| ARIA-CBC | NHIS (Health Insurance) | Pending |
| PKCS#7 | Digital signature | Pending |
| Certificate | Secure storage | Pending |

### 4.4 API Security

| Item | Requirement | Status |
|------|-------------|--------|
| Rate Limiting | Redis token bucket | Pending |
| Input Validation | All endpoints | Pending |
| SQL Injection | Parameterized queries | Pending |
| XSS | Output encoding | Pending |
| CORS | Whitelist domains | Pending |

### 4.5 Infrastructure Security

| Item | Requirement | Status |
|------|-------------|--------|
| Secrets | AWS Secrets Manager | Pending |
| Container | Non-root user | Pending |
| Network | Pod security policies | Pending |
| Logging | Audit trail | Pending |

---

## 5. Terminal Assignment

### Quick Reference

| Terminal | Phase | Agent | Focus |
|----------|-------|-------|-------|
| T1 | Infrastructure | @ops | Docker, CI/CD, Deploy |
| T2 | Database | @db | Schema, Migration, Index |
| T3 | Go Core | @go | Auth, Middleware, Router |
| T4 | Go Business | @acc | Accounting, Tax Invoice |
| T5 | Tax Scraper | @tax | Python, Hometax, gRPC |
| T6 | Insurance EDI | @hr | Python, 4대보험, EDI |
| T7 | Frontend | @react | React, TypeScript, UI |
| T8 | Testing | - | Tests, Security, Docs |

### Starting Each Terminal

```bash
# Terminal 1 - Infrastructure
cd ~/01_DEV/SaaS_erp_clone-260116
claude --phase 1

# Terminal 2 - Database
cd ~/01_DEV/SaaS_erp_clone-260116
claude --phase 2

# ... (repeat for T3-T8)
```

---

## 6. Commands Reference

### Skills

| Command | Description |
|---------|-------------|
| `/api {name}` | Go REST API scaffolding |
| `/grpc {name}` | gRPC Proto generation |
| `/provider {name}` | Provider pattern implementation |
| `/sqlc {name}` | sqlc query generation |
| `/py {name}` | Python gRPC service |
| `/test {name}` | Test code generation |
| `/db {name}` | DB migration |
| `/deploy {env}` | Deploy to environment |

### Agents

| Command | Description |
|---------|-------------|
| `@go` | Go development specialist |
| `@py` | Python gRPC specialist |
| `@react` | React frontend specialist |
| `@acc` | Accounting domain expert |
| `@tax` | Tax invoice specialist |
| `@hr` | HR/Payroll specialist |
| `@db` | Database architect |
| `@ops` | DevOps specialist |

### Remote Commands

```bash
# SSH to remote
ssh wsl-48-246

# Deploy
ssh wsl-48-246 "cd ~/saas-kerp && git pull && docker compose up -d --build"

# View logs
ssh wsl-48-246 "cd ~/saas-kerp && docker compose logs -f"

# Check status
ssh wsl-48-246 "cd ~/saas-kerp && docker compose ps"
```

---

## Appendix

### A. Dependencies Between Phases

```
Phase 1 (Infra) ─────┐
                     ├──→ Phase 3 (Go Core) ──→ Phase 4 (Go Business)
Phase 2 (Database) ──┘           │
                                 ├──→ Phase 5 (Tax Scraper)
                                 ├──→ Phase 6 (Insurance EDI)
                                 └──→ Phase 7 (Frontend)
                                           │
Phase 8 (Testing) ←────────────────────────┘
```

### B. Estimated Timeline

| Week | Phase 1 | Phase 2 | Phase 3 | Phase 4 | Phase 5 | Phase 6 | Phase 7 | Phase 8 |
|------|---------|---------|---------|---------|---------|---------|---------|---------|
| 1-2  | Active  | Active  | -       | -       | -       | -       | -       | -       |
| 3-4  | Done    | Active  | Active  | Start   | Start   | -       | Start   | -       |
| 5-6  | -       | Done    | Active  | Active  | Active  | Start   | Active  | Start   |
| 7-8  | -       | -       | Done    | Active  | Active  | Active  | Active  | Active  |
| 9-10 | -       | -       | -       | Active  | Done    | Active  | Active  | Active  |
| 11-12| -       | -       | -       | Done    | -       | Done    | Done    | Active  |

---

**Document End**
