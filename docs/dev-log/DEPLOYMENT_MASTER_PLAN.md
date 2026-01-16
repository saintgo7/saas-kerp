# K-ERP SaaS 배포 마스터 플랜

> 개발: macOS | 배포: Windows 11 WSL2 Ubuntu | 최종: AWS

---

## 1. 환경 개요

### 1.1 서버 정보

| 항목 | 값 |
|------|---|
| **공인 IP** | 61.245.248.246 |
| **도메인** | erp.abada.kr |
| **SSH 포트** | 5022 |
| **작업 디렉토리** | /home/blackpc/saas-kerp |
| **OS** | Windows 11 + WSL2 Ubuntu |
| **인증** | docs/.env_ssh (비밀번호) |

### 1.2 아키텍처 흐름

```
[Mac 개발] --> [GitHub] --> [WSL2 Ubuntu] --> [AWS 이전]
     |              |              |
   코딩         CI/CD         온프레미스 배포
```

---

## 2. 사전 작업 체크리스트

### 2.1 DNS 설정 (사용자 작업 필요)

DNS 제공업체에서 다음 레코드 추가:

```
# A 레코드
erp.abada.kr    A    61.245.248.246

# 와일드카드 (서브도메인용)
*.erp.abada.kr  A    61.245.248.246

# CAA 레코드 (Let's Encrypt용)
erp.abada.kr    CAA  0 issue "letsencrypt.org"
```

### 2.2 WSL2 포트 포워딩 확인

WSL2 미러링 모드이므로 추가 설정 불필요하나, 다음 포트 오픈 필요:

| 포트 | 용도 |
|------|------|
| 80 | HTTP (Let's Encrypt 검증) |
| 443 | HTTPS |
| 5022 | SSH |
| 5432 | PostgreSQL (내부만) |
| 6379 | Redis (내부만) |

### 2.3 .env_ssh 파일 형식

```bash
# docs/.env_ssh
WSL_HOST=61.245.248.246
WSL_PORT=5022
WSL_USER=blackpc
WSL_PASS=tele9088
WSL_PATH=/home/blackpc/saas-kerp
```

---

## 3. 개발 워크플로우

### 3.1 Git 브랜치 전략

```
main (production)
  └── develop (staging)
       ├── feature/module-name
       ├── fix/bug-description
       └── hotfix/critical-fix
```

### 3.2 커밋 컨벤션

```
feat: Add new feature
fix: Bug fix
docs: Documentation
refactor: Code refactoring
test: Add tests
chore: Maintenance
perf: Performance improvement
```

### 3.3 개발 사이클

```
1. feature 브랜치 생성
2. 코드 작성 + 테스트
3. PR 생성 -> develop
4. CI 통과 확인
5. develop 머지
6. staging 배포 (자동)
7. QA 완료 후 main 머지
8. production 배포 (수동 승인)
```

---

## 4. GitHub Repository 설정

### 4.1 Repository 구조

```
saas-kerp/
├── .github/
│   ├── workflows/
│   │   ├── ci.yml           # PR 빌드/테스트
│   │   ├── cd-staging.yml   # develop -> WSL 배포
│   │   └── cd-production.yml # main -> AWS 배포
│   ├── CODEOWNERS
│   └── pull_request_template.md
├── cmd/
├── internal/
├── web/
├── deployments/
│   ├── docker/
│   ├── kubernetes/
│   └── scripts/
└── docs/
```

### 4.2 GitHub Secrets 설정

```yaml
# Repository Settings > Secrets and variables > Actions

# WSL 배포용
WSL_HOST: 61.245.248.246
WSL_PORT: 5022
WSL_USER: blackpc
WSL_SSH_KEY: <SSH private key>
WSL_PATH: /home/blackpc/saas-kerp

# Docker Registry
DOCKER_REGISTRY: ghcr.io/username/saas-kerp
DOCKER_USERNAME: username
DOCKER_PASSWORD: <GitHub PAT>

# AWS 배포용 (추후)
AWS_ACCESS_KEY_ID:
AWS_SECRET_ACCESS_KEY:
AWS_REGION: ap-northeast-2
```

### 4.3 CI Workflow (ci.yml)

```yaml
name: CI

on:
  pull_request:
    branches: [develop, main]
  push:
    branches: [develop]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v4

  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_DB: kerp_test
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
        ports:
          - 5432:5432
      redis:
        image: redis:7
        ports:
          - 6379:6379
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Run tests
        run: make test
        env:
          DATABASE_URL: postgres://test:test@localhost:5432/kerp_test

  build:
    runs-on: ubuntu-latest
    needs: [lint, test]
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/build-push-action@v5
        with:
          context: .
          push: false
          tags: saas-kerp:${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

### 4.4 CD Staging Workflow (cd-staging.yml)

```yaml
name: CD Staging (WSL)

on:
  push:
    branches: [develop]

jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: staging
    steps:
      - uses: actions/checkout@v4

      - name: Build Docker images
        run: |
          docker build -t ghcr.io/${{ github.repository }}/api:staging .
          docker build -t ghcr.io/${{ github.repository }}/worker:staging -f Dockerfile.worker .

      - name: Push to Registry
        run: |
          echo "${{ secrets.DOCKER_PASSWORD }}" | docker login ghcr.io -u ${{ secrets.DOCKER_USERNAME }} --password-stdin
          docker push ghcr.io/${{ github.repository }}/api:staging
          docker push ghcr.io/${{ github.repository }}/worker:staging

      - name: Deploy to WSL
        uses: appleboy/ssh-action@v1.0.3
        with:
          host: ${{ secrets.WSL_HOST }}
          username: ${{ secrets.WSL_USER }}
          key: ${{ secrets.WSL_SSH_KEY }}
          port: ${{ secrets.WSL_PORT }}
          script: |
            cd ${{ secrets.WSL_PATH }}
            git pull origin develop
            docker-compose -f docker-compose.staging.yml pull
            docker-compose -f docker-compose.staging.yml up -d
            docker system prune -f
```

---

## 5. WSL2 서버 초기 설정

### 5.1 설치 스크립트 (wsl-setup.sh)

```bash
#!/bin/bash
set -e

echo "=== K-ERP WSL2 Setup ==="

# 기본 패키지
sudo apt update && sudo apt upgrade -y
sudo apt install -y curl wget git build-essential

# Docker
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER

# Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Go 1.22
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

# Node.js 20 LTS
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# 프로젝트 디렉토리
mkdir -p /home/blackpc/saas-kerp
cd /home/blackpc/saas-kerp

# SSL 인증서 (Certbot)
sudo apt install -y certbot
sudo certbot certonly --standalone -d erp.abada.kr --agree-tos -m admin@abada.kr --non-interactive

# 인증서 자동 갱신
echo "0 0 * * * root certbot renew --quiet" | sudo tee /etc/cron.d/certbot-renew

echo "=== Setup Complete ==="
```

### 5.2 Docker Compose (docker-compose.staging.yml)

```yaml
version: '3.8'

services:
  traefik:
    image: traefik:v3.0
    command:
      - "--api.insecure=true"
      - "--providers.docker=true"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge=true"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web"
      - "--certificatesresolvers.letsencrypt.acme.email=admin@abada.kr"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./letsencrypt:/letsencrypt
    networks:
      - kerp-network

  api:
    image: ghcr.io/username/saas-kerp/api:staging
    environment:
      - DATABASE_URL=postgres://kerp:${DB_PASSWORD}@postgres:5432/kerp_staging
      - REDIS_URL=redis://redis:6379
      - NATS_URL=nats://nats:4222
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.api.rule=Host(`erp.abada.kr`) && PathPrefix(`/api`)"
      - "traefik.http.routers.api.entrypoints=websecure"
      - "traefik.http.routers.api.tls.certresolver=letsencrypt"
    depends_on:
      - postgres
      - redis
      - nats
    networks:
      - kerp-network

  worker:
    image: ghcr.io/username/saas-kerp/worker:staging
    environment:
      - DATABASE_URL=postgres://kerp:${DB_PASSWORD}@postgres:5432/kerp_staging
      - NATS_URL=nats://nats:4222
    depends_on:
      - postgres
      - nats
    networks:
      - kerp-network

  web:
    image: ghcr.io/username/saas-kerp/web:staging
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.web.rule=Host(`erp.abada.kr`)"
      - "traefik.http.routers.web.entrypoints=websecure"
      - "traefik.http.routers.web.tls.certresolver=letsencrypt"
    networks:
      - kerp-network

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: kerp_staging
      POSTGRES_USER: kerp
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - kerp-network

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
    networks:
      - kerp-network

  nats:
    image: nats:2-alpine
    command: ["--jetstream", "--store_dir=/data"]
    volumes:
      - nats_data:/data
    networks:
      - kerp-network

  minio:
    image: minio/minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: kerp_admin
      MINIO_ROOT_PASSWORD: ${MINIO_PASSWORD}
    volumes:
      - minio_data:/data
    networks:
      - kerp-network

volumes:
  postgres_data:
  redis_data:
  nats_data:
  minio_data:

networks:
  kerp-network:
    driver: bridge
```

---

## 6. Claude Code 자동화 설정

### 6.1 Skills 설정 (.claude/skills/)

```yaml
# .claude/skills/deploy-staging.yaml
name: deploy-staging
description: Deploy to WSL2 staging server
trigger: "/deploy-staging"
steps:
  - name: Check branch
    run: git branch --show-current
    expect: develop
  - name: Run tests
    run: make test
  - name: Push to develop
    run: git push origin develop
  - name: Wait for CI
    run: gh run watch
  - name: Verify deployment
    run: curl -s https://erp.abada.kr/api/health | jq .status
```

```yaml
# .claude/skills/quick-commit.yaml
name: quick-commit
description: Fast commit with conventional format
trigger: "/qc"
args:
  - name: type
    options: [feat, fix, docs, refactor, test, chore]
  - name: message
    required: true
steps:
  - run: git add -A
  - run: git commit -m "${type}: ${message}"
  - run: git push
```

### 6.2 Agents 설정 (.claude/agents/)

```yaml
# .claude/agents/backend-fast.yaml
name: backend-fast
description: Fast backend development agent
model: claude-sonnet-4-20250514
tools:
  - Bash
  - Read
  - Write
  - Edit
  - Grep
  - Glob
context:
  - path: internal/
  - path: cmd/
  - path: pkg/
instructions: |
  You are a Go backend specialist for K-ERP.
  - Follow Clean Architecture
  - Use GORM for simple CRUD, sqlc for complex queries
  - Always include company_id for multi-tenancy
  - Write table-driven tests
  - Be fast and concise
```

```yaml
# .claude/agents/frontend-fast.yaml
name: frontend-fast
description: Fast frontend development agent
model: claude-sonnet-4-20250514
tools:
  - Bash
  - Read
  - Write
  - Edit
context:
  - path: web/
instructions: |
  You are a React/Next.js frontend specialist.
  - Use TypeScript strictly
  - Follow component patterns in existing code
  - Use TanStack Query for data fetching
  - Keep components small and focused
```

### 6.3 MCP 설정 (.claude/mcp.json)

```json
{
  "servers": {
    "supabase": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-supabase"],
      "env": {
        "SUPABASE_URL": "${SUPABASE_URL}",
        "SUPABASE_KEY": "${SUPABASE_KEY}"
      }
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    },
    "postgres": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-postgres"],
      "env": {
        "DATABASE_URL": "${DATABASE_URL}"
      }
    },
    "ssh-deploy": {
      "command": "node",
      "args": ["scripts/mcp-ssh-deploy.js"],
      "env": {
        "SSH_HOST": "61.245.248.246",
        "SSH_PORT": "5022",
        "SSH_USER": "blackpc"
      }
    }
  }
}
```

### 6.4 Hooks 설정 (.claude/hooks/)

```yaml
# .claude/hooks/pre-commit.yaml
name: pre-commit
trigger: before_commit
steps:
  - name: Lint Go
    run: golangci-lint run ./...
  - name: Lint Frontend
    run: cd web && npm run lint
  - name: Check secrets
    run: |
      if grep -rn "password\|secret\|api_key" --include="*.go" --include="*.ts" | grep -v "_test.go"; then
        echo "Potential secret found!"
        exit 1
      fi
```

```yaml
# .claude/hooks/post-push.yaml
name: post-push
trigger: after_push
conditions:
  branch: develop
steps:
  - name: Notify Slack
    run: |
      curl -X POST $SLACK_WEBHOOK -d '{
        "text": "New push to develop by ${GIT_AUTHOR}"
      }'
  - name: Start monitoring
    run: |
      echo "Watching GitHub Actions..."
      gh run watch --exit-status
```

---

## 7. 백업 전략

### 7.1 자동 백업 스케줄

| 백업 대상 | 주기 | 보관 기간 | 저장 위치 |
|-----------|------|-----------|-----------|
| PostgreSQL | 매일 02:00 | 30일 | /backup/postgres/ |
| Redis | 매일 03:00 | 7일 | /backup/redis/ |
| MinIO | 매일 04:00 | 30일 | /backup/minio/ |
| 전체 Docker volumes | 매주 일요일 | 4주 | 외부 NAS |

### 7.2 백업 스크립트 (backup.sh)

```bash
#!/bin/bash
BACKUP_DIR=/home/blackpc/backups
DATE=$(date +%Y%m%d_%H%M%S)

# PostgreSQL
docker exec kerp-postgres pg_dump -U kerp kerp_staging | gzip > $BACKUP_DIR/postgres/kerp_$DATE.sql.gz

# Redis
docker exec kerp-redis redis-cli BGSAVE
docker cp kerp-redis:/data/dump.rdb $BACKUP_DIR/redis/dump_$DATE.rdb

# Cleanup old backups (30 days)
find $BACKUP_DIR -type f -mtime +30 -delete

# Sync to NAS (optional)
# rsync -avz $BACKUP_DIR nas:/kerp-backups/
```

### 7.3 복구 절차

```bash
# PostgreSQL 복구
gunzip -c /backup/postgres/kerp_20260116.sql.gz | docker exec -i kerp-postgres psql -U kerp kerp_staging

# Redis 복구
docker stop kerp-redis
docker cp /backup/redis/dump_20260116.rdb kerp-redis:/data/dump.rdb
docker start kerp-redis
```

---

## 8. AWS 이전 로드맵

### Phase 1: 준비 (WSL 안정화 후)

```
- AWS 계정 및 IAM 설정
- VPC/Subnet 설계
- Terraform 코드 작성
- 비용 예측
```

### Phase 2: 인프라 구축

```
- EKS 클러스터 생성
- RDS PostgreSQL 생성
- ElastiCache Redis 생성
- S3 버킷 생성
- Route 53 도메인 이전
```

### Phase 3: 마이그레이션

```
1. 데이터베이스 마이그레이션 (pg_dump/restore)
2. 파일 스토리지 마이그레이션 (MinIO -> S3)
3. DNS 전환 (erp.abada.kr -> AWS ALB)
4. WSL 서버 DR용으로 유지 또는 종료
```

---

## 9. 빠른 개발을 위한 단축 명령어

### 9.1 Makefile 추가

```makefile
# Quick development commands
.PHONY: qb qt qd qs

qb: ## Quick build
	@go build -o bin/api ./cmd/api

qt: ## Quick test (current package)
	@go test -v ./...

qd: ## Quick deploy to staging
	@git add -A && git commit -m "wip: quick deploy" && git push origin develop

qs: ## Quick status
	@curl -s https://erp.abada.kr/api/health | jq

qssh: ## Quick SSH to WSL
	@ssh -p 5022 blackpc@61.245.248.246

qlogs: ## Quick logs from staging
	@ssh -p 5022 blackpc@61.245.248.246 'docker-compose -f saas-kerp/docker-compose.staging.yml logs -f --tail=100'
```

### 9.2 Shell Aliases (~/.zshrc)

```bash
# K-ERP shortcuts
alias kerp-ssh='ssh -p 5022 blackpc@61.245.248.246'
alias kerp-deploy='git add -A && git commit -m "deploy: quick update" && git push origin develop'
alias kerp-logs='kerp-ssh "cd saas-kerp && docker-compose logs -f"'
alias kerp-status='curl -s https://erp.abada.kr/api/health | jq'
```

---

## 10. 녹화용 데모 시나리오

### 10.1 시나리오: 신규 기능 개발부터 배포까지

```
1. [00:00] 프로젝트 개요 설명
2. [02:00] Claude Code로 기능 개발 시작
   - /agent backend-fast 호출
   - API 엔드포인트 생성
3. [10:00] 테스트 작성 및 실행
4. [15:00] /qc feat "Add new feature" 커밋
5. [17:00] GitHub Actions CI 확인
6. [20:00] /deploy-staging 자동 배포
7. [23:00] https://erp.abada.kr 에서 확인
8. [25:00] 마무리
```

### 10.2 녹화 체크리스트

- [ ] 터미널 폰트 크기 확대 (18pt+)
- [ ] 화면 해상도 1920x1080
- [ ] 불필요한 알림 끄기
- [ ] .env 파일 숨기기
- [ ] 에러 시나리오도 준비

---

## 11. 작업 순서 (우선순위)

### Week 1: 기반 구축
1. GitHub Repository 생성 및 설정
2. WSL2 서버 초기 설정
3. DNS 설정 (erp.abada.kr)
4. Docker Compose 배포 테스트
5. HTTPS 인증서 발급

### Week 2: CI/CD 파이프라인
1. GitHub Actions CI 워크플로우
2. CD Staging 워크플로우
3. Claude Code skills/agents 설정
4. 백업 자동화

### Week 3: 핵심 모듈 개발
1. 인증/인가 모듈
2. 회사/테넌트 관리
3. 사용자 관리
4. 기초 API 테스트

### Week 4+: 비즈니스 모듈
1. 회계 모듈
2. 세금계산서 연동
3. 인사/급여
4. 재고/판매

---

## 12. 연락처 및 참고

- **도메인 관리**: (DNS 설정 필요)
- **GitHub**: https://github.com/saintgo7/saas-kerp
- **Staging URL**: https://erp.abada.kr
- **모니터링**: https://erp.abada.kr/grafana (추후)

---

*Last Updated: 2026-01-16*
