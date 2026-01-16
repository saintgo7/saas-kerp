# K-ERP v0.2 - CI/CD 파이프라인

**문서 버전**: 0.2.0
**작성일**: 2026-01-16
**상태**: 검토 대기

---

## 목차

1. [CI/CD 개요](#1-cicd-개요)
2. [GitHub Actions 워크플로우](#2-github-actions-워크플로우)
3. [Go 빌드 파이프라인](#3-go-빌드-파이프라인)
4. [Python 빌드 파이프라인](#4-python-빌드-파이프라인)
5. [배포 파이프라인](#5-배포-파이프라인)

---

## 1. CI/CD 개요

### 1.1 전체 파이프라인 아키텍처

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         CI/CD Pipeline Architecture                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Developer                                                                  │
│      │                                                                      │
│      ▼                                                                      │
│  ┌─────────┐    ┌─────────────────────────────────────────────────────┐   │
│  │  Push   │───▶│                  GitHub Actions                      │   │
│  │  to Git │    │  ┌─────────────────────────────────────────────────┐│   │
│  └─────────┘    │  │ PR Checks (feature/*, bugfix/*)                 ││   │
│                 │  │  - Lint (Go + Python)                            ││   │
│                 │  │  - Unit Tests                                    ││   │
│                 │  │  - Security Scan                                 ││   │
│                 │  └─────────────────────────────────────────────────┘│   │
│                 │                         │                            │   │
│                 │                         ▼                            │   │
│                 │  ┌─────────────────────────────────────────────────┐│   │
│                 │  │ Merge to develop                                ││   │
│                 │  │  - Build Docker Images                          ││   │
│                 │  │  - Integration Tests                            ││   │
│                 │  │  - Push to Dev Registry                         ││   │
│                 │  │  - Deploy to Dev                                ││   │
│                 │  └─────────────────────────────────────────────────┘│   │
│                 │                         │                            │   │
│                 │                         ▼                            │   │
│                 │  ┌─────────────────────────────────────────────────┐│   │
│                 │  │ Merge to main                                   ││   │
│                 │  │  - Build Production Images                      ││   │
│                 │  │  - E2E Tests                                    ││   │
│                 │  │  - Push to Prod Registry                        ││   │
│                 │  │  - Deploy to Staging                            ││   │
│                 │  │  - Manual Approval                              ││   │
│                 │  │  - Deploy to Production                         ││   │
│                 │  └─────────────────────────────────────────────────┘│   │
│                 └─────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 브랜치 전략

| 브랜치 | 용도 | 배포 대상 | 보호 규칙 |
|--------|------|-----------|-----------|
| main | 운영 코드 | Production | PR 필수, 2인 승인 |
| develop | 개발 통합 | Development | PR 필수, 1인 승인 |
| feature/* | 기능 개발 | - | - |
| bugfix/* | 버그 수정 | - | - |
| hotfix/* | 긴급 수정 | Production | PR 필수, 1인 승인 |
| release/* | 릴리스 준비 | Staging | PR 필수, 2인 승인 |

---

## 2. GitHub Actions 워크플로우

### 2.1 PR Checks 워크플로우

```yaml
# .github/workflows/pr-checks.yml
name: PR Checks

on:
  pull_request:
    branches: [main, develop]
    types: [opened, synchronize, reopened]

env:
  GO_VERSION: '1.22'
  PYTHON_VERSION: '3.11'

jobs:
  # Go 린트 및 테스트
  go-checks:
    name: Go Checks
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Install dependencies
        run: go mod download

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: v1.55.2
          args: --timeout=5m

      - name: Run tests
        run: |
          go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: ./coverage.out
          flags: go

  # Python 린트 및 테스트
  python-checks:
    name: Python Checks
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: python-services
    steps:
      - uses: actions/checkout@v4

      - name: Setup Python
        uses: actions/setup-python@v5
        with:
          python-version: ${{ env.PYTHON_VERSION }}
          cache: 'pip'

      - name: Install dependencies
        run: |
          python -m pip install --upgrade pip
          pip install -r requirements.txt
          pip install -r requirements-dev.txt

      - name: Run ruff (linter)
        run: ruff check .

      - name: Run mypy (type check)
        run: mypy .

      - name: Run pytest
        run: |
          pytest --cov=. --cov-report=xml -v

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: python-services/coverage.xml
          flags: python

  # 보안 스캔
  security-scan:
    name: Security Scan
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          severity: 'CRITICAL,HIGH'
          exit-code: '1'

      - name: Run gosec
        uses: securego/gosec@master
        with:
          args: ./...

      - name: Run Bandit (Python)
        run: |
          pip install bandit
          bandit -r python-services/ -ll

  # Proto 파일 검증
  proto-check:
    name: Proto Check
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Buf
        uses: bufbuild/buf-setup-action@v1
        with:
          version: 'latest'

      - name: Buf lint
        run: buf lint

      - name: Buf breaking
        run: buf breaking --against '.git#branch=main'
```

### 2.2 Build 워크플로우

```yaml
# .github/workflows/build.yml
name: Build

on:
  push:
    branches: [develop, main]

env:
  REGISTRY: ghcr.io
  GO_VERSION: '1.22'
  PYTHON_VERSION: '3.11'

jobs:
  build-go:
    name: Build Go Services
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    strategy:
      matrix:
        service: [api-server, worker]
    steps:
      - uses: actions/checkout@v4

      - name: Setup Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ github.repository }}/${{ matrix.service }}
          tags: |
            type=ref,event=branch
            type=sha,prefix=
            type=raw,value=latest,enable=${{ github.ref == 'refs/heads/main' }}

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          file: deployments/docker/${{ matrix.service }}/Dockerfile
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  build-python:
    name: Build Python Services
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    strategy:
      matrix:
        service: [tax-scraper, insurance-edi]
    steps:
      - uses: actions/checkout@v4

      - name: Setup Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ github.repository }}/${{ matrix.service }}
          tags: |
            type=ref,event=branch
            type=sha,prefix=
            type=raw,value=latest,enable=${{ github.ref == 'refs/heads/main' }}

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: python-services
          file: python-services/${{ matrix.service }}/Dockerfile
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  integration-tests:
    name: Integration Tests
    needs: [build-go, build-python]
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_DB: kerp_test
          POSTGRES_USER: kerp
          POSTGRES_PASSWORD: kerp_test
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
      nats:
        image: nats:2.10-alpine
        ports:
          - 4222:4222
        options: --jetstream
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Run integration tests
        env:
          DATABASE_URL: postgres://kerp:kerp_test@localhost:5432/kerp_test?sslmode=disable
          REDIS_URL: redis://localhost:6379
          NATS_URL: nats://localhost:4222
        run: |
          go test -v -tags=integration ./tests/integration/...
```

---

## 3. Go 빌드 파이프라인

### 3.1 Dockerfile

```dockerfile
# deployments/docker/api-server/Dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev} -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /bin/api ./cmd/api

# Runtime stage
FROM alpine:3.19

# Security: non-root user
RUN addgroup -g 1000 kerp && \
    adduser -u 1000 -G kerp -s /bin/sh -D kerp

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/api /app/api
COPY --from=builder /app/db/migrations /app/migrations

RUN chown -R kerp:kerp /app

USER kerp

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/api"]
```

### 3.2 Makefile

```makefile
# Makefile
.PHONY: all build test lint clean docker-build docker-push

VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-w -s -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

GO_FILES := $(shell find . -name '*.go' -not -path "./vendor/*")

# Build
build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/api ./cmd/api
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/worker ./cmd/worker

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/api-linux ./cmd/api
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/worker-linux ./cmd/worker

# Test
test:
	go test -v -race -cover ./...

test-integration:
	go test -v -tags=integration ./tests/integration/...

test-coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	golangci-lint run ./...

# Code generation
generate:
	go generate ./...

sqlc:
	sqlc generate

proto:
	buf generate

swagger:
	swag init -g cmd/api/main.go -o api/docs

# Database
migrate-up:
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	migrate create -ext sql -dir db/migrations -seq $(name)

# Docker
docker-build:
	docker build -f deployments/docker/api-server/Dockerfile -t kerp/api:$(VERSION) .
	docker build -f deployments/docker/worker/Dockerfile -t kerp/worker:$(VERSION) .

docker-push:
	docker push kerp/api:$(VERSION)
	docker push kerp/worker:$(VERSION)

# Development
dev-up:
	docker-compose up -d

dev-down:
	docker-compose down

dev-logs:
	docker-compose logs -f

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
```

---

## 4. Python 빌드 파이프라인

### 4.1 Dockerfile

```dockerfile
# python-services/tax-scraper/Dockerfile
FROM python:3.11-slim AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Create virtual environment
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Runtime stage
FROM python:3.11-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    chromium \
    chromium-driver \
    fonts-nanum \
    fonts-nanum-coding \
    && rm -rf /var/lib/apt/lists/*

# Security: non-root user
RUN groupadd -g 1000 kerp && \
    useradd -u 1000 -g kerp -s /bin/bash -m kerp

# Copy virtual environment
COPY --from=builder /opt/venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Install Playwright browsers
RUN playwright install chromium

# Copy source
COPY shared/ /app/shared/
COPY tax-scraper/src/ /app/src/

ENV PYTHONPATH=/app
ENV PYTHONUNBUFFERED=1
ENV PLAYWRIGHT_BROWSERS_PATH=/root/.cache/ms-playwright

RUN chown -R kerp:kerp /app

USER kerp

EXPOSE 50051

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD python -c "import grpc; channel = grpc.insecure_channel('localhost:50051'); grpc.channel_ready_future(channel).result(timeout=5)" || exit 1

CMD ["python", "-m", "src.main"]
```

### 4.2 pyproject.toml

```toml
# python-services/pyproject.toml
[build-system]
requires = ["setuptools>=61.0"]
build-backend = "setuptools.build_meta"

[project]
name = "kerp-python-services"
version = "0.2.0"
description = "K-ERP Python microservices"
requires-python = ">=3.11"

dependencies = [
    "grpcio>=1.60.0",
    "grpcio-tools>=1.60.0",
    "playwright>=1.40.0",
    "pycryptodome>=3.19.0",
    "cryptography>=41.0.0",
    "httpx>=0.26.0",
    "pydantic>=2.5.0",
    "python-dotenv>=1.0.0",
    "structlog>=24.1.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=7.4.0",
    "pytest-cov>=4.1.0",
    "pytest-asyncio>=0.23.0",
    "ruff>=0.1.9",
    "mypy>=1.8.0",
    "bandit>=1.7.6",
]

[tool.ruff]
line-length = 100
target-version = "py311"
select = ["E", "F", "I", "N", "W", "UP", "B", "C4", "SIM"]

[tool.mypy]
python_version = "3.11"
strict = true
ignore_missing_imports = true

[tool.pytest.ini_options]
asyncio_mode = "auto"
testpaths = ["tests"]
addopts = "-v --cov --cov-report=term-missing"

[tool.coverage.run]
source = ["src", "shared"]
omit = ["tests/*", "*_pb2.py", "*_pb2_grpc.py"]
```

---

## 5. 배포 파이프라인

### 5.1 Deploy 워크플로우

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  workflow_run:
    workflows: ["Build"]
    types: [completed]
    branches: [develop, main]

env:
  REGISTRY: ghcr.io
  CLUSTER_NAME: kerp-cluster

jobs:
  deploy-dev:
    name: Deploy to Development
    if: ${{ github.event.workflow_run.conclusion == 'success' && github.event.workflow_run.head_branch == 'develop' }}
    runs-on: ubuntu-latest
    environment: development
    steps:
      - uses: actions/checkout@v4

      - name: Configure kubectl
        uses: azure/k8s-set-context@v3
        with:
          kubeconfig: ${{ secrets.KUBE_CONFIG_DEV }}

      - name: Deploy to dev
        run: |
          kubectl set image deployment/api-server \
            api-server=${{ env.REGISTRY }}/${{ github.repository }}/api-server:${{ github.sha }} \
            -n kerp-dev
          kubectl set image deployment/worker \
            worker=${{ env.REGISTRY }}/${{ github.repository }}/worker:${{ github.sha }} \
            -n kerp-dev
          kubectl set image deployment/tax-scraper \
            tax-scraper=${{ env.REGISTRY }}/${{ github.repository }}/tax-scraper:${{ github.sha }} \
            -n kerp-dev
          kubectl set image deployment/insurance-edi \
            insurance-edi=${{ env.REGISTRY }}/${{ github.repository }}/insurance-edi:${{ github.sha }} \
            -n kerp-dev

      - name: Verify deployment
        run: |
          kubectl rollout status deployment/api-server -n kerp-dev --timeout=5m
          kubectl rollout status deployment/worker -n kerp-dev --timeout=5m
          kubectl rollout status deployment/tax-scraper -n kerp-dev --timeout=5m
          kubectl rollout status deployment/insurance-edi -n kerp-dev --timeout=5m

  deploy-staging:
    name: Deploy to Staging
    if: ${{ github.event.workflow_run.conclusion == 'success' && github.event.workflow_run.head_branch == 'main' }}
    runs-on: ubuntu-latest
    environment: staging
    steps:
      - uses: actions/checkout@v4

      - name: Configure kubectl
        uses: azure/k8s-set-context@v3
        with:
          kubeconfig: ${{ secrets.KUBE_CONFIG_STAGING }}

      - name: Deploy to staging
        run: |
          kubectl apply -k deployments/k8s/overlays/staging/

      - name: Run E2E tests
        run: |
          npm install
          npx playwright test --project=staging

  deploy-prod:
    name: Deploy to Production
    needs: deploy-staging
    runs-on: ubuntu-latest
    environment:
      name: production
      url: https://api.kerp.io
    steps:
      - uses: actions/checkout@v4

      - name: Configure kubectl
        uses: azure/k8s-set-context@v3
        with:
          kubeconfig: ${{ secrets.KUBE_CONFIG_PROD }}

      - name: Blue-Green deployment
        run: |
          # Create new (green) deployment
          kubectl apply -k deployments/k8s/overlays/prod/

          # Wait for green to be ready
          kubectl rollout status deployment/api-server-green -n kerp --timeout=10m

          # Switch traffic to green
          kubectl patch service api-server -n kerp \
            -p '{"spec":{"selector":{"version":"green"}}}'

          # Verify green is healthy
          sleep 30
          curl -sf https://api.kerp.io/health || exit 1

          # Scale down blue
          kubectl scale deployment/api-server-blue --replicas=0 -n kerp

      - name: Notify Slack
        if: always()
        uses: slackapi/slack-github-action@v1.24.0
        with:
          payload: |
            {
              "text": "Production deployment ${{ job.status }}",
              "blocks": [
                {
                  "type": "section",
                  "text": {
                    "type": "mrkdwn",
                    "text": "*Production Deployment*\nStatus: ${{ job.status }}\nCommit: ${{ github.sha }}"
                  }
                }
              ]
            }
        env:
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
```

### 5.2 Helm Chart 구조

```yaml
# deployments/helm/kerp/Chart.yaml
apiVersion: v2
name: kerp
description: K-ERP Helm chart
type: application
version: 0.2.0
appVersion: "0.2.0"

dependencies:
  - name: postgresql
    version: "14.0.0"
    repository: "https://charts.bitnami.com/bitnami"
    condition: postgresql.enabled
  - name: redis
    version: "18.0.0"
    repository: "https://charts.bitnami.com/bitnami"
    condition: redis.enabled
```

```yaml
# deployments/helm/kerp/values.yaml
global:
  imageRegistry: ghcr.io
  imagePullSecrets:
    - name: ghcr-secret

apiServer:
  replicaCount: 3
  image:
    repository: kerp/api-server
    tag: latest
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 20
    targetCPUUtilizationPercentage: 70

worker:
  replicaCount: 2
  image:
    repository: kerp/worker
    tag: latest
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi

taxScraper:
  replicaCount: 2
  image:
    repository: kerp/tax-scraper
    tag: latest
  resources:
    requests:
      cpu: 200m
      memory: 512Mi
    limits:
      cpu: 1000m
      memory: 2Gi

insuranceEdi:
  replicaCount: 2
  image:
    repository: kerp/insurance-edi
    tag: latest
  resources:
    requests:
      cpu: 100m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 1Gi

postgresql:
  enabled: false  # Use external in production

redis:
  enabled: false  # Use external in production

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: api.kerp.io
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: kerp-tls
      hosts:
        - api.kerp.io
```

### 5.3 Rollback 워크플로우

```yaml
# .github/workflows/rollback.yml
name: Rollback

on:
  workflow_dispatch:
    inputs:
      environment:
        description: 'Environment to rollback'
        required: true
        type: choice
        options:
          - development
          - staging
          - production
      version:
        description: 'Version to rollback to (e.g., v0.2.1)'
        required: true
        type: string

jobs:
  rollback:
    name: Rollback ${{ github.event.inputs.environment }}
    runs-on: ubuntu-latest
    environment: ${{ github.event.inputs.environment }}
    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ github.event.inputs.version }}

      - name: Configure kubectl
        uses: azure/k8s-set-context@v3
        with:
          kubeconfig: ${{ secrets[format('KUBE_CONFIG_{0}', upper(github.event.inputs.environment))] }}

      - name: Rollback deployment
        run: |
          kubectl rollout undo deployment/api-server -n kerp-${{ github.event.inputs.environment }}
          kubectl rollout undo deployment/worker -n kerp-${{ github.event.inputs.environment }}
          kubectl rollout undo deployment/tax-scraper -n kerp-${{ github.event.inputs.environment }}
          kubectl rollout undo deployment/insurance-edi -n kerp-${{ github.event.inputs.environment }}

      - name: Verify rollback
        run: |
          kubectl rollout status deployment/api-server -n kerp-${{ github.event.inputs.environment }} --timeout=5m

      - name: Notify team
        uses: slackapi/slack-github-action@v1.24.0
        with:
          payload: |
            {
              "text": "Rollback completed for ${{ github.event.inputs.environment }} to ${{ github.event.inputs.version }}"
            }
        env:
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
```

---

**다음 문서**: [12_API_설계_상세.md](./12_API_설계_상세.md)
