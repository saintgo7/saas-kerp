# 11. CI/CD 파이프라인

## 개요

K-ERP 시스템의 지속적 통합(CI) 및 지속적 배포(CD) 파이프라인 설계.
GitHub Actions를 기반으로 빌드, 테스트, 배포 자동화.

### 파이프라인 아키텍처

```
┌──────────────────────────────────────────────────────────────────┐
│                         GitHub Actions                            │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐         │
│  │  Lint   │──▶│  Test   │──▶│  Build  │──▶│  Push   │         │
│  └─────────┘   └─────────┘   └─────────┘   └─────────┘         │
│       │             │             │             │                │
│       ▼             ▼             ▼             ▼                │
│  golangci-lint  go test     Docker Build    Docker Hub          │
│  eslint         jest        ARM64/AMD64     / ECR               │
│                                                                   │
├──────────────────────────────────────────────────────────────────┤
│                         Deployment                                │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐           │
│  │ Development │──▶│   Staging   │──▶│ Production  │           │
│  │   (auto)    │   │   (auto)    │   │  (manual)   │           │
│  └─────────────┘   └─────────────┘   └─────────────┘           │
│                                                                   │
└──────────────────────────────────────────────────────────────────┘
```

---

## 1. 브랜치 전략

### Git Flow 기반 브랜치 모델

```
main ─────────────────────────────────────────────▶ Production
  │
  └─── release/v1.2.0 ────────────────────────────▶ Staging
         │
         └─── develop ────────────────────────────▶ Development
                │
                ├─── feature/ACC-123-voucher-api
                ├─── feature/HR-456-payroll
                └─── hotfix/fix-login-bug
```

### 브랜치별 배포 환경

| 브랜치 | 환경 | 배포 방식 | 승인 |
|--------|------|----------|------|
| `feature/*` | - | PR 빌드만 | - |
| `develop` | Development | 자동 | - |
| `release/*` | Staging | 자동 | - |
| `main` | Production | 수동 | 필요 |
| `hotfix/*` | Staging → Production | 반자동 | 필요 |

---

## 2. GitHub Actions 워크플로우

### 2.1 PR 검증 워크플로우

```yaml
# .github/workflows/pr-check.yml
name: PR Check

on:
  pull_request:
    branches: [develop, main, 'release/**']
    types: [opened, synchronize, reopened]

concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

env:
  GO_VERSION: '1.22'
  NODE_VERSION: '20'

jobs:
  # ==================== Go Backend ====================
  backend-lint:
    name: Backend Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: v1.57
          args: --timeout=5m
          working-directory: ./backend

  backend-test:
    name: Backend Test
    runs-on: ubuntu-latest
    needs: backend-lint
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: kerp_test
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
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Run migrations
        working-directory: ./backend
        run: |
          go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
          migrate -path db/migrations -database "postgres://test:test@localhost:5432/kerp_test?sslmode=disable" up
        env:
          DATABASE_URL: postgres://test:test@localhost:5432/kerp_test?sslmode=disable

      - name: Run tests
        working-directory: ./backend
        run: |
          go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
        env:
          DATABASE_URL: postgres://test:test@localhost:5432/kerp_test?sslmode=disable
          REDIS_URL: redis://localhost:6379
          GO_ENV: test

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: ./backend/coverage.out
          flags: backend
          fail_ci_if_error: false

  backend-security:
    name: Backend Security Scan
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Run Gosec Security Scanner
        uses: securego/gosec@master
        with:
          args: -exclude-generated ./backend/...

      - name: Run govulncheck
        uses: golang/govulncheck-action@v1
        with:
          go-version-input: ${{ env.GO_VERSION }}
          work-dir: ./backend

  # ==================== Frontend ====================
  frontend-lint:
    name: Frontend Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: 'npm'
          cache-dependency-path: ./web/package-lock.json

      - name: Install dependencies
        working-directory: ./web
        run: npm ci

      - name: ESLint
        working-directory: ./web
        run: npm run lint

      - name: TypeScript Check
        working-directory: ./web
        run: npm run type-check

  frontend-test:
    name: Frontend Test
    runs-on: ubuntu-latest
    needs: frontend-lint
    steps:
      - uses: actions/checkout@v4

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: 'npm'
          cache-dependency-path: ./web/package-lock.json

      - name: Install dependencies
        working-directory: ./web
        run: npm ci

      - name: Run tests
        working-directory: ./web
        run: npm run test:coverage

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: ./web/coverage/lcov.info
          flags: frontend
          fail_ci_if_error: false

  frontend-build:
    name: Frontend Build
    runs-on: ubuntu-latest
    needs: [frontend-lint, frontend-test]
    steps:
      - uses: actions/checkout@v4

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: 'npm'
          cache-dependency-path: ./web/package-lock.json

      - name: Install dependencies
        working-directory: ./web
        run: npm ci

      - name: Build
        working-directory: ./web
        run: npm run build
        env:
          VITE_API_URL: /api

      - name: Upload build artifacts
        uses: actions/upload-artifact@v4
        with:
          name: frontend-build
          path: ./web/dist
          retention-days: 1
```

### 2.2 CI/CD 메인 파이프라인

```yaml
# .github/workflows/ci-cd.yml
name: CI/CD Pipeline

on:
  push:
    branches:
      - develop
      - 'release/**'
      - main
  workflow_dispatch:
    inputs:
      environment:
        description: 'Deployment environment'
        required: true
        type: choice
        options:
          - development
          - staging
          - production

env:
  GO_VERSION: '1.22'
  NODE_VERSION: '20'
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  # ==================== Build ====================
  build-backend:
    name: Build Backend
    runs-on: ubuntu-latest
    outputs:
      version: ${{ steps.version.outputs.version }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Generate version
        id: version
        run: |
          if [[ "${{ github.ref }}" == "refs/heads/main" ]]; then
            VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
          else
            VERSION=$(git describe --tags --always --dirty)
          fi
          echo "version=${VERSION}" >> $GITHUB_OUTPUT

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Build binary
        working-directory: ./backend
        run: |
          CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
            -ldflags="-s -w -X main.version=${{ steps.version.outputs.version }}" \
            -o ../bin/api-amd64 ./cmd/api

          CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
            -ldflags="-s -w -X main.version=${{ steps.version.outputs.version }}" \
            -o ../bin/api-arm64 ./cmd/api

          CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
            -ldflags="-s -w -X main.version=${{ steps.version.outputs.version }}" \
            -o ../bin/worker-amd64 ./cmd/worker

          CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
            -ldflags="-s -w -X main.version=${{ steps.version.outputs.version }}" \
            -o ../bin/worker-arm64 ./cmd/worker

      - name: Upload binaries
        uses: actions/upload-artifact@v4
        with:
          name: backend-binaries
          path: ./bin/
          retention-days: 1

  build-frontend:
    name: Build Frontend
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: 'npm'
          cache-dependency-path: ./web/package-lock.json

      - name: Install dependencies
        working-directory: ./web
        run: npm ci

      - name: Build
        working-directory: ./web
        run: npm run build
        env:
          VITE_API_URL: /api

      - name: Upload build
        uses: actions/upload-artifact@v4
        with:
          name: frontend-build
          path: ./web/dist
          retention-days: 1

  # ==================== Docker ====================
  docker-build:
    name: Build Docker Images
    runs-on: ubuntu-latest
    needs: [build-backend, build-frontend]
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4

      - name: Download backend binaries
        uses: actions/download-artifact@v4
        with:
          name: backend-binaries
          path: ./bin/

      - name: Download frontend build
        uses: actions/download-artifact@v4
        with:
          name: frontend-build
          path: ./web/dist/

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=ref,event=branch
            type=sha,prefix=
            type=raw,value=${{ needs.build-backend.outputs.version }}

      # API Server Image
      - name: Build and push API image
        uses: docker/build-push-action@v5
        with:
          context: .
          file: ./deployments/docker/Dockerfile.api
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}-api:${{ needs.build-backend.outputs.version }}
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}-api:${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

      # Worker Image
      - name: Build and push Worker image
        uses: docker/build-push-action@v5
        with:
          context: .
          file: ./deployments/docker/Dockerfile.worker
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}-worker:${{ needs.build-backend.outputs.version }}
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}-worker:${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  # ==================== Deploy ====================
  deploy-development:
    name: Deploy to Development
    runs-on: ubuntu-latest
    needs: docker-build
    if: github.ref == 'refs/heads/develop'
    environment:
      name: development
      url: https://dev.kerp.example.com
    steps:
      - uses: actions/checkout@v4

      - name: Set up kubectl
        uses: azure/setup-kubectl@v3
        with:
          version: 'v1.29.0'

      - name: Configure kubectl
        run: |
          echo "${{ secrets.KUBE_CONFIG_DEV }}" | base64 -d > $HOME/.kube/config
          chmod 600 $HOME/.kube/config

      - name: Deploy to Kubernetes
        run: |
          kubectl set image deployment/kerp-api \
            api=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}-api:${{ github.sha }} \
            -n kerp-dev

          kubectl set image deployment/kerp-worker \
            worker=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}-worker:${{ github.sha }} \
            -n kerp-dev

          kubectl rollout status deployment/kerp-api -n kerp-dev --timeout=300s
          kubectl rollout status deployment/kerp-worker -n kerp-dev --timeout=300s

      - name: Notify Slack
        uses: 8398a7/action-slack@v3
        with:
          status: ${{ job.status }}
          fields: repo,message,commit,author,action,eventName,workflow
        env:
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK }}
        if: always()

  deploy-staging:
    name: Deploy to Staging
    runs-on: ubuntu-latest
    needs: docker-build
    if: startsWith(github.ref, 'refs/heads/release/')
    environment:
      name: staging
      url: https://staging.kerp.example.com
    steps:
      - uses: actions/checkout@v4

      - name: Set up kubectl
        uses: azure/setup-kubectl@v3
        with:
          version: 'v1.29.0'

      - name: Configure kubectl
        run: |
          echo "${{ secrets.KUBE_CONFIG_STAGING }}" | base64 -d > $HOME/.kube/config
          chmod 600 $HOME/.kube/config

      - name: Deploy to Kubernetes
        run: |
          kubectl set image deployment/kerp-api \
            api=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}-api:${{ github.sha }} \
            -n kerp-staging

          kubectl set image deployment/kerp-worker \
            worker=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}-worker:${{ github.sha }} \
            -n kerp-staging

          kubectl rollout status deployment/kerp-api -n kerp-staging --timeout=300s
          kubectl rollout status deployment/kerp-worker -n kerp-staging --timeout=300s

      - name: Run smoke tests
        run: |
          chmod +x ./scripts/smoke-test.sh
          ./scripts/smoke-test.sh https://staging.kerp.example.com

  deploy-production:
    name: Deploy to Production
    runs-on: ubuntu-latest
    needs: docker-build
    if: github.ref == 'refs/heads/main'
    environment:
      name: production
      url: https://kerp.example.com
    steps:
      - uses: actions/checkout@v4

      - name: Set up kubectl
        uses: azure/setup-kubectl@v3
        with:
          version: 'v1.29.0'

      - name: Configure kubectl
        run: |
          echo "${{ secrets.KUBE_CONFIG_PROD }}" | base64 -d > $HOME/.kube/config
          chmod 600 $HOME/.kube/config

      - name: Blue-Green Deployment
        run: |
          # 새 버전 배포 (Green)
          kubectl apply -f deployments/k8s/production/deployment-green.yaml

          # 헬스체크 대기
          kubectl rollout status deployment/kerp-api-green -n kerp-prod --timeout=300s

          # 트래픽 전환
          kubectl patch service kerp-api -n kerp-prod \
            -p '{"spec":{"selector":{"version":"green"}}}'

          # 이전 버전 정리 (Blue)
          kubectl scale deployment/kerp-api-blue --replicas=0 -n kerp-prod

      - name: Notify on success
        uses: 8398a7/action-slack@v3
        with:
          status: success
          fields: repo,message,commit,author
          text: '🚀 Production deployment successful!'
        env:
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK }}

      - name: Notify on failure
        uses: 8398a7/action-slack@v3
        with:
          status: failure
          fields: repo,message,commit,author
          text: '❌ Production deployment failed!'
        env:
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK }}
        if: failure()
```

### 2.3 핫픽스 워크플로우

```yaml
# .github/workflows/hotfix.yml
name: Hotfix Pipeline

on:
  push:
    branches:
      - 'hotfix/**'

jobs:
  hotfix-deploy:
    name: Hotfix Deployment
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Extract hotfix version
        id: version
        run: |
          BRANCH_NAME=${GITHUB_REF#refs/heads/}
          VERSION=${BRANCH_NAME#hotfix/}
          echo "version=${VERSION}" >> $GITHUB_OUTPUT

      - name: Build and test
        run: |
          cd backend && go test ./...
          cd ../web && npm ci && npm run test

      - name: Build Docker images
        run: |
          docker build -t kerp-api:hotfix-${{ steps.version.outputs.version }} \
            -f deployments/docker/Dockerfile.api .

      - name: Deploy to Staging
        run: |
          echo "Deploying hotfix to staging..."
          # Staging deployment logic

      - name: Create PR to main
        uses: peter-evans/create-pull-request@v6
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          branch: hotfix/${{ steps.version.outputs.version }}
          base: main
          title: "Hotfix: ${{ steps.version.outputs.version }}"
          body: |
            ## Hotfix Release
            Version: ${{ steps.version.outputs.version }}

            ### Changes
            ${{ github.event.head_commit.message }}

            ### Checklist
            - [ ] Staging tested
            - [ ] Production approved
```

---

## 3. Docker 구성

### 3.1 API 서버 Dockerfile

```dockerfile
# deployments/docker/Dockerfile.api
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.source="https://github.com/your-org/k-erp"
LABEL org.opencontainers.image.description="K-ERP API Server"

# 아키텍처에 따른 바이너리 선택
ARG TARGETARCH
COPY bin/api-${TARGETARCH} /app/api
COPY web/dist /app/static

WORKDIR /app

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/api"]
```

### 3.2 Worker Dockerfile

```dockerfile
# deployments/docker/Dockerfile.worker
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.source="https://github.com/your-org/k-erp"
LABEL org.opencontainers.image.description="K-ERP Background Worker"

ARG TARGETARCH
COPY bin/worker-${TARGETARCH} /app/worker

WORKDIR /app

USER nonroot:nonroot

ENTRYPOINT ["/app/worker"]
```

### 3.3 Docker Compose (로컬 개발)

```yaml
# docker-compose.yml
version: '3.8'

services:
  api:
    build:
      context: .
      dockerfile: deployments/docker/Dockerfile.api
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://kerp:kerp@postgres:5432/kerp?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=${JWT_SECRET}
      - POPBILL_LINK_ID=${POPBILL_LINK_ID}
      - POPBILL_SECRET_KEY=${POPBILL_SECRET_KEY}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  worker:
    build:
      context: .
      dockerfile: deployments/docker/Dockerfile.worker
    environment:
      - DATABASE_URL=postgres://kerp:kerp@postgres:5432/kerp?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - NATS_URL=nats://nats:4222
    depends_on:
      - postgres
      - redis
      - nats

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: kerp
      POSTGRES_PASSWORD: kerp
      POSTGRES_DB: kerp
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U kerp"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    volumes:
      - redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

  nats:
    image: nats:2.10-alpine
    command: ["--jetstream", "--store_dir=/data"]
    volumes:
      - nats-data:/data

volumes:
  postgres-data:
  redis-data:
  nats-data:
```

---

## 4. Kubernetes 매니페스트

### 4.1 Deployment

```yaml
# deployments/k8s/base/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kerp-api
  labels:
    app: kerp
    component: api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: kerp
      component: api
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: kerp
        component: api
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: kerp-api
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
        fsGroup: 65534

      containers:
        - name: api
          image: ghcr.io/your-org/k-erp-api:latest
          imagePullPolicy: Always

          ports:
            - name: http
              containerPort: 8080
              protocol: TCP

          env:
            - name: GO_ENV
              value: production
            - name: PORT
              value: "8080"
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: kerp-secrets
                  key: database-url
            - name: REDIS_URL
              valueFrom:
                secretKeyRef:
                  name: kerp-secrets
                  key: redis-url
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: kerp-secrets
                  key: jwt-secret

          resources:
            requests:
              cpu: 100m
              memory: 256Mi
            limits:
              cpu: 500m
              memory: 512Mi

          readinessProbe:
            httpGet:
              path: /health/ready
              port: http
            initialDelaySeconds: 5
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3

          livenessProbe:
            httpGet:
              path: /health/live
              port: http
            initialDelaySeconds: 15
            periodSeconds: 20
            timeoutSeconds: 5
            failureThreshold: 3

          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop:
                - ALL

      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchLabels:
                    app: kerp
                    component: api
                topologyKey: kubernetes.io/hostname

      topologySpreadConstraints:
        - maxSkew: 1
          topologyKey: topology.kubernetes.io/zone
          whenUnsatisfiable: ScheduleAnyway
          labelSelector:
            matchLabels:
              app: kerp
              component: api
```

### 4.2 Service

```yaml
# deployments/k8s/base/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: kerp-api
  labels:
    app: kerp
    component: api
spec:
  type: ClusterIP
  ports:
    - name: http
      port: 80
      targetPort: http
      protocol: TCP
  selector:
    app: kerp
    component: api
---
apiVersion: v1
kind: Service
metadata:
  name: kerp-api-headless
  labels:
    app: kerp
    component: api
spec:
  type: ClusterIP
  clusterIP: None
  ports:
    - name: http
      port: 80
      targetPort: http
  selector:
    app: kerp
    component: api
```

### 4.3 Ingress

```yaml
# deployments/k8s/base/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: kerp-ingress
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "50m"
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/rate-limit-window: "1m"
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
    - hosts:
        - kerp.example.com
      secretName: kerp-tls
  rules:
    - host: kerp.example.com
      http:
        paths:
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: kerp-api
                port:
                  number: 80
          - path: /
            pathType: Prefix
            backend:
              service:
                name: kerp-web
                port:
                  number: 80
```

### 4.4 HPA (Horizontal Pod Autoscaler)

```yaml
# deployments/k8s/base/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: kerp-api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: kerp-api
  minReplicas: 3
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 10
          periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
        - type: Percent
          value: 100
          periodSeconds: 15
        - type: Pods
          value: 4
          periodSeconds: 15
      selectPolicy: Max
```

### 4.5 PodDisruptionBudget

```yaml
# deployments/k8s/base/pdb.yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: kerp-api-pdb
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: kerp
      component: api
```

---

## 5. 환경별 구성 (Kustomize)

### 5.1 Development 오버레이

```yaml
# deployments/k8s/overlays/development/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: kerp-dev

resources:
  - ../../base

replicas:
  - name: kerp-api
    count: 1

images:
  - name: ghcr.io/your-org/k-erp-api
    newTag: develop

patchesStrategicMerge:
  - deployment-patch.yaml

configMapGenerator:
  - name: kerp-config
    literals:
      - LOG_LEVEL=debug
      - ENABLE_SWAGGER=true
```

```yaml
# deployments/k8s/overlays/development/deployment-patch.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kerp-api
spec:
  template:
    spec:
      containers:
        - name: api
          resources:
            requests:
              cpu: 50m
              memory: 128Mi
            limits:
              cpu: 200m
              memory: 256Mi
```

### 5.2 Production 오버레이

```yaml
# deployments/k8s/overlays/production/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: kerp-prod

resources:
  - ../../base
  - network-policy.yaml

replicas:
  - name: kerp-api
    count: 5

images:
  - name: ghcr.io/your-org/k-erp-api
    newTag: v1.0.0

patchesStrategicMerge:
  - deployment-patch.yaml

configMapGenerator:
  - name: kerp-config
    literals:
      - LOG_LEVEL=info
      - ENABLE_SWAGGER=false
```

---

## 6. 시크릿 관리

### 6.1 Sealed Secrets 사용

```yaml
# deployments/k8s/secrets/sealed-secret.yaml
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  name: kerp-secrets
  namespace: kerp-prod
spec:
  encryptedData:
    database-url: AgBy8h...  # 암호화된 값
    redis-url: AgCT9x...
    jwt-secret: AgDK2m...
    popbill-link-id: AgEQ5n...
    popbill-secret-key: AgFR3p...
```

### 6.2 External Secrets Operator 사용

```yaml
# deployments/k8s/secrets/external-secret.yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: kerp-secrets
  namespace: kerp-prod
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: aws-secrets-manager
  target:
    name: kerp-secrets
    creationPolicy: Owner
  data:
    - secretKey: database-url
      remoteRef:
        key: kerp/production
        property: database_url
    - secretKey: redis-url
      remoteRef:
        key: kerp/production
        property: redis_url
    - secretKey: jwt-secret
      remoteRef:
        key: kerp/production
        property: jwt_secret
```

---

## 7. 품질 게이트

### 7.1 품질 기준

```yaml
# .github/quality-gates.yml
quality_gates:
  code_coverage:
    backend:
      minimum: 80
      target: 90
    frontend:
      minimum: 70
      target: 85

  security:
    critical_vulnerabilities: 0
    high_vulnerabilities: 0
    medium_vulnerabilities: 5

  performance:
    build_time_seconds: 300
    docker_image_size_mb: 100

  lint:
    errors: 0
    warnings: 10
```

### 7.2 품질 게이트 검증 스크립트

```bash
#!/bin/bash
# scripts/quality-gate.sh

set -e

echo "Running Quality Gates..."

# 1. 코드 커버리지 체크
BACKEND_COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
if (( $(echo "$BACKEND_COVERAGE < 80" | bc -l) )); then
    echo "❌ Backend coverage ($BACKEND_COVERAGE%) is below threshold (80%)"
    exit 1
fi
echo "✅ Backend coverage: $BACKEND_COVERAGE%"

# 2. 보안 취약점 체크
CRITICAL=$(cat security-report.json | jq '.Critical // 0')
HIGH=$(cat security-report.json | jq '.High // 0')
if [ "$CRITICAL" -gt 0 ] || [ "$HIGH" -gt 0 ]; then
    echo "❌ Security vulnerabilities found: Critical=$CRITICAL, High=$HIGH"
    exit 1
fi
echo "✅ No critical/high security vulnerabilities"

# 3. Docker 이미지 크기 체크
IMAGE_SIZE=$(docker images kerp-api --format "{{.Size}}" | head -1)
echo "✅ Docker image size: $IMAGE_SIZE"

echo "All quality gates passed! ✅"
```

---

## 8. 롤백 전략

### 8.1 자동 롤백 스크립트

```bash
#!/bin/bash
# scripts/rollback.sh

set -e

NAMESPACE=${1:-kerp-prod}
DEPLOYMENT=${2:-kerp-api}

echo "Starting rollback for $DEPLOYMENT in $NAMESPACE..."

# 현재 버전 기록
CURRENT_IMAGE=$(kubectl get deployment $DEPLOYMENT -n $NAMESPACE \
    -o jsonpath='{.spec.template.spec.containers[0].image}')
echo "Current image: $CURRENT_IMAGE"

# 이전 버전으로 롤백
kubectl rollout undo deployment/$DEPLOYMENT -n $NAMESPACE

# 롤백 완료 대기
kubectl rollout status deployment/$DEPLOYMENT -n $NAMESPACE --timeout=300s

# 새 버전 확인
NEW_IMAGE=$(kubectl get deployment $DEPLOYMENT -n $NAMESPACE \
    -o jsonpath='{.spec.template.spec.containers[0].image}')
echo "Rolled back to: $NEW_IMAGE"

# Slack 알림
curl -X POST -H 'Content-type: application/json' \
    --data "{\"text\":\"🔄 Rollback completed: $DEPLOYMENT from $CURRENT_IMAGE to $NEW_IMAGE\"}" \
    $SLACK_WEBHOOK_URL

echo "Rollback completed successfully!"
```

### 8.2 GitHub Actions 롤백 워크플로우

```yaml
# .github/workflows/rollback.yml
name: Production Rollback

on:
  workflow_dispatch:
    inputs:
      revision:
        description: 'Revision to rollback to (leave empty for previous)'
        required: false
      reason:
        description: 'Reason for rollback'
        required: true

jobs:
  rollback:
    name: Rollback Production
    runs-on: ubuntu-latest
    environment:
      name: production-rollback
    steps:
      - uses: actions/checkout@v4

      - name: Configure kubectl
        run: |
          echo "${{ secrets.KUBE_CONFIG_PROD }}" | base64 -d > $HOME/.kube/config

      - name: Perform rollback
        run: |
          if [ -n "${{ github.event.inputs.revision }}" ]; then
            kubectl rollout undo deployment/kerp-api \
              --to-revision=${{ github.event.inputs.revision }} \
              -n kerp-prod
          else
            kubectl rollout undo deployment/kerp-api -n kerp-prod
          fi

          kubectl rollout status deployment/kerp-api -n kerp-prod --timeout=300s

      - name: Notify Slack
        uses: 8398a7/action-slack@v3
        with:
          status: custom
          custom_payload: |
            {
              "text": "🔄 Production Rollback Executed",
              "attachments": [{
                "color": "warning",
                "fields": [
                  {"title": "Triggered by", "value": "${{ github.actor }}", "short": true},
                  {"title": "Reason", "value": "${{ github.event.inputs.reason }}", "short": true}
                ]
              }]
            }
        env:
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK }}
```

---

## 9. 스크립트

### 9.1 스모크 테스트

```bash
#!/bin/bash
# scripts/smoke-test.sh

BASE_URL=${1:-http://localhost:8080}
MAX_RETRIES=30
RETRY_INTERVAL=5

echo "Running smoke tests against $BASE_URL"

# 헬스체크 대기
for i in $(seq 1 $MAX_RETRIES); do
    if curl -sf "$BASE_URL/health/ready" > /dev/null; then
        echo "✅ Health check passed"
        break
    fi

    if [ $i -eq $MAX_RETRIES ]; then
        echo "❌ Health check failed after $MAX_RETRIES retries"
        exit 1
    fi

    echo "Waiting for service... ($i/$MAX_RETRIES)"
    sleep $RETRY_INTERVAL
done

# API 엔드포인트 테스트
test_endpoint() {
    local endpoint=$1
    local expected_status=${2:-200}

    status=$(curl -sf -o /dev/null -w "%{http_code}" "$BASE_URL$endpoint")

    if [ "$status" -eq "$expected_status" ]; then
        echo "✅ $endpoint -> $status"
        return 0
    else
        echo "❌ $endpoint -> $status (expected $expected_status)"
        return 1
    fi
}

FAILED=0

test_endpoint "/health/live" 200 || FAILED=$((FAILED+1))
test_endpoint "/health/ready" 200 || FAILED=$((FAILED+1))
test_endpoint "/api/v1/version" 200 || FAILED=$((FAILED+1))
test_endpoint "/api/v1/auth/login" 401 || FAILED=$((FAILED+1))  # 인증 필요 확인

if [ $FAILED -gt 0 ]; then
    echo "❌ Smoke tests failed: $FAILED failures"
    exit 1
fi

echo "✅ All smoke tests passed!"
```

### 9.2 DB 마이그레이션 스크립트

```bash
#!/bin/bash
# scripts/migrate.sh

set -e

ACTION=${1:-up}
DATABASE_URL=${DATABASE_URL:-"postgres://kerp:kerp@localhost:5432/kerp?sslmode=disable"}

# migrate 도구 설치 확인
if ! command -v migrate &> /dev/null; then
    echo "Installing golang-migrate..."
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
fi

echo "Running migrations ($ACTION)..."

case $ACTION in
    up)
        migrate -path db/migrations -database "$DATABASE_URL" up
        ;;
    down)
        migrate -path db/migrations -database "$DATABASE_URL" down 1
        ;;
    drop)
        migrate -path db/migrations -database "$DATABASE_URL" drop -f
        ;;
    version)
        migrate -path db/migrations -database "$DATABASE_URL" version
        ;;
    create)
        NAME=${2:-"migration"}
        migrate create -ext sql -dir db/migrations -seq "$NAME"
        ;;
    *)
        echo "Usage: $0 {up|down|drop|version|create <name>}"
        exit 1
        ;;
esac

echo "Migration completed!"
```

---

## 10. 체크리스트

### CI/CD 구축 체크리스트

- [ ] GitHub Actions 워크플로우 설정
- [ ] PR 검증 파이프라인 (lint, test, security)
- [ ] Docker 멀티 아키텍처 빌드 (amd64, arm64)
- [ ] 컨테이너 레지스트리 설정 (GHCR)
- [ ] Kubernetes 매니페스트 작성
- [ ] Kustomize 환경별 오버레이
- [ ] Sealed Secrets / External Secrets 설정
- [ ] HPA 설정 (오토스케일링)
- [ ] PodDisruptionBudget 설정
- [ ] 롤백 워크플로우
- [ ] 스모크 테스트 스크립트
- [ ] Slack 알림 연동

### 환경별 확인사항

| 환경 | URL | 배포 방식 | 승인 |
|------|-----|----------|------|
| Development | dev.kerp.example.com | 자동 (develop 브랜치) | 불필요 |
| Staging | staging.kerp.example.com | 자동 (release 브랜치) | 불필요 |
| Production | kerp.example.com | 수동 (main 브랜치) | 필요 |
