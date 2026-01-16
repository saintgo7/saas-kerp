# K-ERP 녹화용 Quick Start Guide

> 빠른 개발 데모를 위한 가이드

---

## 사전 준비 (녹화 전)

### 1. DNS 설정 (직접 작업)
DNS 관리자에서 다음 레코드 추가:
```
erp.abada.kr    A    61.245.248.246
```

### 2. WSL2 서버 준비 (SSH 접속 후)
```bash
# SSH 접속
ssh -p 5022 blackpc@61.245.248.246

# 초기 설정 실행
curl -fsSL https://raw.githubusercontent.com/username/saas-kerp/main/scripts/wsl-setup.sh | bash

# 프로젝트 클론
cd /home/blackpc
git clone https://github.com/saintgo7/saas-kerp.git
cd saas-kerp

# 환경변수 설정
cp .env.example .env
nano .env  # 비밀번호 등 설정

# Docker 시작
docker-compose up -d
```

### 3. GitHub 설정
1. Repository 생성: `saas-kerp`
2. Secrets 추가 (Settings > Secrets):
   - `WSL_HOST`: 61.245.248.246
   - `WSL_PORT`: 5022
   - `WSL_USER`: blackpc
   - `WSL_SSH_KEY`: SSH 개인키

### 4. 로컬 환경
```bash
# .env_ssh 파일 생성
cp docs/.env_ssh.example docs/.env_ssh
nano docs/.env_ssh  # 비밀번호 입력
```

---

## 녹화 시나리오

### Part 1: 환경 소개 (2분)
```
- 프로젝트 구조 설명
- 기술 스택 소개 (Go + Python + React)
- 아키텍처 다이어그램 보여주기
```

### Part 2: Claude Code 자동화 데모 (5분)
```bash
# Agent 호출
claude /go "Create user registration API"

# Skill 사용
claude /qf backend user-registration

# 빠른 커밋
claude /qc feat "Add user registration"
```

### Part 3: 실시간 개발 (10분)
```bash
# 1. API 엔드포인트 생성
# 2. 테스트 작성
# 3. 프론트엔드 연동

# 테스트 실행
make test

# 로컬 확인
make run
curl localhost:8080/api/v1/health
```

### Part 4: 배포 (5분)
```bash
# 스테이징 배포
claude /dw

# 또는 수동
git push origin develop

# CI/CD 확인
gh run watch

# 배포 확인
curl https://erp.abada.kr/api/health
```

### Part 5: 결과 확인 (3분)
```
- https://erp.abada.kr 접속
- API 테스트
- 로그 확인
```

---

## 빠른 명령어 모음

### Claude Code Shortcuts
| 명령어 | 설명 |
|--------|------|
| `/go <task>` | Go 백엔드 개발 |
| `/react <task>` | React 프론트엔드 개발 |
| `/ops <task>` | DevOps 작업 |
| `/dw` | WSL 배포 |
| `/qf <module> <name>` | 기능 빠른 생성 |
| `/db create <name>` | 마이그레이션 생성 |

### Make 명령어
```bash
make qb      # 빠른 빌드
make qt      # 빠른 테스트
make qd      # 빠른 배포
make qs      # 상태 확인
make qssh    # SSH 접속
make qlogs   # 로그 보기
```

### Git Aliases
```bash
alias kerp-ssh='ssh -p 5022 blackpc@61.245.248.246'
alias kerp-deploy='git push origin develop'
alias kerp-status='curl -s https://erp.abada.kr/api/health | jq'
```

---

## 에러 대응

### SSH 접속 실패
```bash
# 포트 확인
nc -zv 61.245.248.246 5022

# Windows 방화벽 확인 (WSL 호스트에서)
# Windows + R > wf.msc > Inbound Rules > New Rule > Port 5022
```

### Docker 컨테이너 문제
```bash
kerp-ssh
cd saas-kerp
docker-compose logs api
docker-compose restart api
```

### HTTPS 인증서 문제
```bash
kerp-ssh
sudo certbot renew --force-renewal
docker-compose restart traefik
```

---

## 체크리스트

### 녹화 전
- [ ] 화면 해상도 1920x1080
- [ ] 터미널 폰트 18pt
- [ ] 알림 끄기
- [ ] .env 파일 가리기
- [ ] DNS 전파 확인: `nslookup erp.abada.kr`
- [ ] WSL 서버 상태 확인
- [ ] GitHub Actions 초록불 확인

### 녹화 중
- [ ] 천천히 타이핑 (시청자 이해)
- [ ] 명령어 실행 전 설명
- [ ] 에러 발생 시 당황하지 않기

### 녹화 후
- [ ] 영상 편집
- [ ] 코드 정리
- [ ] 문서 업데이트

---

*Recording Guide v1.0 - 2026-01-16*
