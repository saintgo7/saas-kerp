.PHONY: help build run test test-unit test-integration test-coverage test-security test-all \
        test-frontend test-e2e test-python generate-mocks lint clean \
        dev-up dev-down test-up test-down migrate-up migrate-down \
        perf-auth perf-voucher perf-concurrent perf-all perf-smoke

# Variables
GO := go
GOTEST := $(GO) test
GOCOVER := $(GO) tool cover
MOCKERY := mockery

# Help
help:
	@echo "K-ERP SaaS Development Commands"
	@echo ""
	@echo "Build & Run:"
	@echo "  make build          - Build API server"
	@echo "  make run            - Run API server"
	@echo "  make run-worker     - Run background worker"
	@echo ""
	@echo "Testing:"
	@echo "  make test           - Run all Go tests"
	@echo "  make test-unit      - Run unit tests only"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make test-security  - Run security tests"
	@echo "  make test-frontend  - Run frontend tests"
	@echo "  make test-e2e       - Run E2E tests"
	@echo "  make test-python    - Run Python tests"
	@echo "  make test-all       - Run all tests"
	@echo ""
	@echo "Code Generation:"
	@echo "  make generate-mocks - Generate mock files"
	@echo ""
	@echo "Infrastructure:"
	@echo "  make dev-up         - Start development services"
	@echo "  make dev-down       - Stop development services"
	@echo "  make test-up        - Start test services"
	@echo "  make test-down      - Stop test services"
	@echo ""
	@echo "Database:"
	@echo "  make migrate-up     - Run database migrations"
	@echo "  make migrate-down   - Rollback database migrations"

# Build
build:
	$(GO) build -o bin/api ./cmd/api
	$(GO) build -o bin/worker ./cmd/worker

run:
	$(GO) run ./cmd/api

run-worker:
	$(GO) run ./cmd/worker

# Testing - Go
test:
	$(GOTEST) -v ./...

test-unit:
	$(GOTEST) -v -short -race ./...

test-integration:
	$(GOTEST) -v -tags=integration -race ./...

test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCOVER) -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-security:
	$(GOTEST) -v -tags=security ./tests/security/...

# Testing - Frontend
test-frontend:
	cd web && npm run test:run

test-e2e:
	cd web && npx playwright test

# Testing - Python
test-python:
	cd python-services && pytest -v --cov=. --cov-report=html

# Testing - All
test-all: test-unit test-integration test-frontend test-python

# Code Generation
generate-mocks:
	@echo "Generating mocks..."
	$(MOCKERY) --dir=internal/repository --name=VoucherRepository --output=internal/mocks --outpkg=mocks
	$(MOCKERY) --dir=internal/repository --name=AccountRepository --output=internal/mocks --outpkg=mocks
	$(MOCKERY) --dir=internal/repository --name=LedgerRepository --output=internal/mocks --outpkg=mocks
	$(MOCKERY) --dir=internal/repository --name=PartnerRepository --output=internal/mocks --outpkg=mocks
	$(MOCKERY) --dir=internal/repository --name=TaxInvoiceRepository --output=internal/mocks --outpkg=mocks
	$(MOCKERY) --dir=internal/service --name=VoucherService --output=internal/mocks --outpkg=mocks
	$(MOCKERY) --dir=internal/service --name=AccountService --output=internal/mocks --outpkg=mocks
	$(MOCKERY) --dir=internal/service --name=LedgerService --output=internal/mocks --outpkg=mocks
	@echo "Mocks generated successfully"

# Lint
lint:
	golangci-lint run ./...

# Infrastructure
dev-up:
	docker compose -f deployments/docker/docker-compose.yml up -d

dev-down:
	docker compose -f deployments/docker/docker-compose.yml down

test-up:
	docker compose -f tests/docker-compose.test.yml up -d
	@echo "Waiting for services to be ready..."
	@sleep 5

test-down:
	docker compose -f tests/docker-compose.test.yml down

# Database
migrate-up:
	$(GO) run ./cmd/migrate up

migrate-down:
	$(GO) run ./cmd/migrate down

migrate-create:
	@read -p "Migration name: " name; \
	$(GO) run ./cmd/migrate create $$name

# Performance Testing
perf-auth:
	k6 run tests/performance/k6/scenarios/auth-flow.js

perf-voucher:
	k6 run tests/performance/k6/scenarios/voucher-api.js

perf-concurrent:
	k6 run tests/performance/k6/scenarios/concurrent-posting.js

perf-all: perf-auth perf-voucher perf-concurrent

perf-smoke:
	k6 run --vus 5 --duration 30s tests/performance/k6/scenarios/auth-flow.js

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -rf web/coverage/
	rm -rf python-services/htmlcov/
