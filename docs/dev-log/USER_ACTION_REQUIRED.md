# 사용자 직접 작업 필요 항목

> 이 문서의 작업은 Claude가 수행할 수 없으며, 사용자가 직접 완료해야 합니다.

---

## 진행 체크리스트

- [x] DNS 설정 (Cloudflare Tunnel로 대체)
- [x] WSL2 SSH 서버 설정 (포트 5022)
- [x] Windows 방화벽 5022 포트 오픈
- [x] GitHub Repository 생성
- [x] WSL에 Docker/Docker Compose 설치
- [x] Cloudflare Tunnel 설정 (포트 80/443 차단 우회)
- [ ] GitHub Secrets 추가 (CI/CD용)
- [ ] SSH 키 쌍 생성 및 배포 (CI/CD용)

---

## 1. Cloudflare Tunnel 설정 (완료)

> ISP에서 포트 80/443을 차단하여 Cloudflare Tunnel로 우회

| 항목 | 값 |
|------|-----|
| Tunnel 이름 | `kerp-erp` |
| Tunnel ID | `10e537ac-f1e1-43e3-a2bd-3c86753378f2` |
| 도메인 | `erp.abada.kr` |
| 서비스 | `http://localhost:8080` |

**상세 설정:** `docs/dev-log/CLOUDFLARE_TUNNEL_SETUP.md` 참조

---

## 2. WSL2 서버 SSH 설정 (완료)

- SSH 서버 포트: 5022
- cloudflared 서비스 실행 중
- Docker/Docker Compose 설치 완료

---

## 3. GitHub Repository (완료)

- URL: https://github.com/saintgo7/saas-kerp.git
- Status: DONE

---

## 4. 남은 작업: GitHub Secrets 추가

Settings > Secrets and variables > Actions > New repository secret

| Name | Value |
|------|-------|
| WSL_HOST | 61.245.248.246 |
| WSL_PORT | 5022 |
| WSL_USER | blackpc |
| WSL_SSH_KEY | (SSH 개인키 전체 내용) |
| WSL_PATH | /home/blackpc/saas-kerp |

### SSH 키 생성 (Mac에서)
```bash
ssh-keygen -t ed25519 -C "kerp-deploy" -f ~/.ssh/id_kerp
cat ~/.ssh/id_kerp.pub  # WSL에 복사
cat ~/.ssh/id_kerp      # GitHub Secret에 복사
```

### WSL에 공개키 추가
```bash
# WSL에서
mkdir -p ~/.ssh
echo "ssh-ed25519 AAAA..." >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

---

## 현재 아키텍처

```
[인터넷]
    |
    v
[Cloudflare Tunnel] <-- erp.abada.kr (HTTPS, 무료)
    |
    v
[WSL2 Ubuntu]
    |-- cloudflared (커넥터, 상시 실행)
    |-- Docker + Docker Compose
    |-- Go API 서버 (포트 8080)
    |-- PostgreSQL (포트 5432, 내부)
    |-- Redis (포트 6379, 내부)
```

---

## 테스트 방법

```bash
# WSL에서 API 서버 실행
cd /home/blackpc/saas-kerp
go run cmd/api/main.go

# 브라우저에서 접속
https://erp.abada.kr/health
```

---

*Last Updated: 2026-01-17*
