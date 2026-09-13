.PHONY: help build run test test-unit test-integration test-coverage test-security test-all \
        test-frontend test-e2e test-python generate-mocks proto proto-tools proto-check lint clean \
        dev-up dev-down dev-config prod-config test-up test-down \
        check-migrate-tool migrate-up migrate-down migrate-create scan-secrets \
        perf-auth perf-voucher perf-concurrent perf-all perf-smoke

# Variables
GO := go
GOTEST := $(GO) test
GOCOVER := $(GO) tool cover
MOCKERY := mockery

# Protobuf / gRPC code generation.
#
# buf is deliberately not used: it is not part of this repo's toolchain and the
# whole job is one protoc invocation with two plugins. The plugin versions are
# pinned to match go.mod (google.golang.org/protobuf v1.31.0,
# google.golang.org/grpc v1.60.0) so a regenerated file does not drag in a
# runtime the module does not have.
PROTOC := protoc
GO_MODULE := github.com/saintgo7/saas-kerp
PROTO_DIR := api/proto
PROTO_FILES := tax/v1/tax.proto common/v1/common.proto
PROTOC_GEN_GO_VERSION := v1.31.0
PROTOC_GEN_GO_GRPC_VERSION := v1.3.0
GO_BIN := $(shell $(GO) env GOPATH)/bin

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
	@echo "  make proto          - Generate Go protobuf/gRPC stubs from api/proto"
	@echo "  make proto-tools    - Install the pinned protoc plugins"
	@echo "  make proto-check    - Fail if the checked-in stubs are stale"
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
	@echo ""
	@echo "Configuration:"
	@echo "  make dev-config     - Validate the dev compose overlay"
	@echo "  make prod-config    - Validate the staging/production compose"
	@echo "  make scan-secrets   - Run the repository secret scanner"
	@echo ""
	@echo "Performance Testing:"
	@echo "  make perf-auth      - Run auth flow performance test"
	@echo "  make perf-voucher   - Run voucher API performance test"
	@echo "  make perf-concurrent- Run concurrent posting stress test"
	@echo "  make perf-all       - Run all performance tests"
	@echo "  make perf-smoke     - Run quick smoke test"

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
#
# Regenerate the Go stubs in api/proto. The generated *.pb.go files are checked
# in, as is usual for Go: `go build` and `go test` must work on a clean
# checkout without protoc installed.
#
# api/proto/tax/v1/tax.proto must stay wire compatible with
# python-services/tax-scraper/proto/tax.proto - same field numbers, same types.
# Changing a field number here silently corrupts every 세금계산서 that crosses
# the wire.
proto-tools:
	@echo "Installing protoc plugins..."
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)

proto: proto-tools
	@command -v $(PROTOC) >/dev/null 2>&1 || { echo "protoc not found. Install it (brew install protobuf) and retry."; exit 1; }
	@echo "Generating Go protobuf/gRPC stubs..."
	PATH="$(GO_BIN):$$PATH" $(PROTOC) -I $(PROTO_DIR) \
		--go_out=. --go_opt=module=$(GO_MODULE) \
		--go-grpc_out=. --go-grpc_opt=module=$(GO_MODULE) \
		--go-grpc_opt=require_unimplemented_servers=false \
		$(PROTO_FILES)
	@echo "Stubs generated under $(PROTO_DIR)"

# Regenerate into the working tree and fail if anything changed, so CI catches
# a .proto edit that was never compiled.
proto-check: proto
	@if ! git diff --quiet -- $(PROTO_DIR); then \
		echo "Generated protobuf stubs are stale. Run 'make proto' and commit the result:"; \
		git --no-pager diff --stat -- $(PROTO_DIR); \
		exit 1; \
	fi
	@echo "Protobuf stubs are up to date"

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
#
# The base compose file is the staging/production stack (Traefik, Let's
# Encrypt, the full monitoring stack) and every secret in it is guarded with
# `${VAR:?}`. `make dev-up` must therefore always add the dev override, which
# binds every port to 127.0.0.1 and mounts the demo seed.
DEV_COMPOSE := -f deployments/docker/docker-compose.yml -f deployments/docker/docker-compose.dev.yml

dev-up:
	docker compose $(DEV_COMPOSE) up -d

dev-down:
	docker compose $(DEV_COMPOSE) down

dev-config:
	docker compose $(DEV_COMPOSE) config -q && echo "dev compose config OK"

# Staging/production stack, for validating the real configuration locally.
# Never brings anything up - `config` only.
prod-config:
	docker compose -f deployments/docker/docker-compose.yml config -q && echo "prod compose config OK"

test-up:
	docker compose -f tests/docker-compose.test.yml up -d
	@echo "Waiting for services to be ready..."
	@sleep 5

test-down:
	docker compose -f tests/docker-compose.test.yml down

# Database
#
# NOTE: ./cmd/migrate does not exist yet (cmd/ holds only api/ and worker/), so
# these targets have never run. They are kept because CD calls the same binary;
# they now fail with an explanation instead of a bare "no Go files" error.
# Owner: Go side. Once cmd/migrate lands, delete the guard below.
MIGRATE_PKG := ./cmd/migrate

check-migrate-tool:
	@test -d $(MIGRATE_PKG) || { \
		echo "ERROR: $(MIGRATE_PKG) does not exist."; \
		echo "The migration binary has not been written yet."; \
		echo "For a throwaway test database use: scripts/apply-test-migrations.sh"; \
		exit 1; \
	}

migrate-up: check-migrate-tool
	$(GO) run $(MIGRATE_PKG) up

migrate-down: check-migrate-tool
	$(GO) run $(MIGRATE_PKG) down

migrate-create: check-migrate-tool
	@read -p "Migration name: " name; \
	$(GO) run $(MIGRATE_PKG) create $$name

# Secret hygiene
scan-secrets:
	python3 scripts/scan-secrets.py --self-test
	python3 scripts/scan-secrets.py

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
