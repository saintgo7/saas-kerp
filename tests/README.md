# K-ERP SaaS Testing Guide

## Overview

This directory contains test infrastructure and cross-component tests for the K-ERP SaaS platform.

**Target Coverage:** 80%+

## Directory Structure

```
tests/
├── docker-compose.test.yml  # Test services (PostgreSQL, Redis, NATS)
├── fixtures/                # Shared test data
│   ├── companies.json
│   ├── accounts.json
│   ├── vouchers.json
│   └── users.json
├── e2e/                     # End-to-end tests
├── security/                # Security tests (RLS, auth)
└── performance/             # K6 load tests
    └── k6/scenarios/
```

## Quick Start

### 1. Start Test Services

```bash
docker compose -f tests/docker-compose.test.yml up -d
```

### 2. Run Tests

```bash
# Go unit tests
go test -v -short ./...

# Go integration tests
go test -v -tags=integration ./...

# Python tests
cd python-services && pytest -v --cov

# Frontend tests
cd web && npm run test:run

# E2E tests
cd web && npx playwright test
```

### 3. Stop Test Services

```bash
docker compose -f tests/docker-compose.test.yml down
```

## Test Categories

### Unit Tests
- **Go:** `internal/**/*_test.go` (without integration tag)
- **Python:** `python-services/*/tests/test_*.py`
- **Frontend:** `web/src/**/*.test.{ts,tsx}`

### Integration Tests
- **Go:** `internal/**/*_test.go` (with `//go:build integration` tag)
- Uses testcontainers for isolated database instances

### E2E Tests
- **Frontend:** `web/e2e/specs/*.spec.ts` (Playwright)
- **API:** `tests/e2e/` (cross-service scenarios)

### Security Tests
- RLS tenant isolation
- Authentication bypass prevention
- RBAC enforcement

### Performance Tests
- K6 load test scenarios
- Thresholds: p95 < 200ms, error rate < 1%

## Test Environment Variables

```bash
# Test Database
TEST_DATABASE_URL=postgres://kerp_test:kerp_test_password@localhost:5433/kerp_test?sslmode=disable

# Test Redis
TEST_REDIS_URL=redis://localhost:6380

# Test NATS
TEST_NATS_URL=nats://localhost:4223
```

## Makefile Commands

```bash
make test-unit        # Run unit tests
make test-integration # Run integration tests
make test-frontend    # Run frontend tests
make test-e2e         # Run E2E tests
make test-python      # Run Python tests
make test-all         # Run all tests
```

## Coverage Reports

| Component | Report Location |
|-----------|-----------------|
| Go | `coverage.html` |
| Python | `python-services/htmlcov/index.html` |
| Frontend | `web/coverage/index.html` |

## Critical Test Scenarios

### Voucher Workflow
1. Create balanced voucher (debit == credit)
2. Submit draft -> pending
3. Approve pending -> approved
4. Post approved -> posted
5. Reject any non-posted state

### Multi-Tenancy
1. Company A cannot access Company B data
2. All queries filter by company_id
3. RLS policies enforced at DB level

### Authentication
1. Valid JWT access
2. Expired token rejection
3. Invalid signature rejection
4. Token refresh flow
