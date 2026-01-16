# 사용자 직접 작업 필요 항목

> 이 문서의 작업은 Claude가 수행할 수 없으며, 사용자가 직접 완료해야 합니다.

---

## 1. DNS 설정 (필수)

도메인 관리자 (가비아, 카페24 등)에서:

```
타입: A
호스트: erp
값: 61.245.248.246
TTL: 300 (또는 기본값)
```

**확인 방법:**
```bash
nslookup erp.abada.kr
# 또는
dig erp.abada.kr
```

예상 결과:
```
erp.abada.kr.   300  IN  A  61.245.248.246
```

---

## 2. WSL2 서버 SSH 설정

### 2.1 SSH 서버 설치 (WSL Ubuntu 내에서)
```bash
sudo apt update
sudo apt install -y openssh-server
sudo systemctl enable ssh
sudo systemctl start ssh
```

### 2.2 SSH 포트 변경
```bash
sudo nano /etc/ssh/sshd_config
# Port 5022 로 변경
sudo systemctl restart ssh
```

### 2.3 Windows 방화벽 설정
1. Windows + R > `wf.msc`
2. Inbound Rules > New Rule
3. Port > TCP > 5022
4. Allow the connection
5. All profiles
6. Name: "SSH WSL2 5022"

### 2.4 WSL2 미러링 확인
```powershell
# PowerShell에서
cat ~/.wslconfig
```

내용:
```ini
[wsl2]
networkingMode=mirrored
```

---

## 3. GitHub Repository 설정

### 3.1 Repository (완료)
- URL: https://github.com/saintgo7/saas-kerp.git
- Status: DONE

### 3.2 Secrets 추가
Settings > Secrets and variables > Actions > New repository secret

| Name | Value |
|------|-------|
| WSL_HOST | 61.245.248.246 |
| WSL_PORT | 5022 |
| WSL_USER | blackpc |
| WSL_SSH_KEY | (SSH 개인키 전체 내용) |
| WSL_PATH | /home/blackpc/saas-kerp |

### 3.3 SSH 키 생성 (Mac에서)
```bash
ssh-keygen -t ed25519 -C "kerp-deploy" -f ~/.ssh/id_kerp
cat ~/.ssh/id_kerp.pub  # WSL에 복사
cat ~/.ssh/id_kerp      # GitHub Secret에 복사
```

### 3.4 WSL에 공개키 추가
```bash
# WSL에서
mkdir -p ~/.ssh
echo "ssh-ed25519 AAAA..." >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

---

## 4. .env_ssh 파일 생성

```bash
cd /Users/saint/01_DEV/SaaS_erp_clone-260116
cp docs/.env_ssh.example docs/.env_ssh
nano docs/.env_ssh
# WSL_PASS 값 입력
```

---

## 5. HTTPS 인증서 (WSL 서버에서)

DNS 설정 완료 후:
```bash
sudo apt install -y certbot
sudo certbot certonly --standalone -d erp.abada.kr --agree-tos -m your@email.com --non-interactive
```

인증서 위치:
- `/etc/letsencrypt/live/erp.abada.kr/fullchain.pem`
- `/etc/letsencrypt/live/erp.abada.kr/privkey.pem`

---

## 6. Docker 설치 (WSL 서버에서)

```bash
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
# 로그아웃 후 재로그인
docker --version
docker-compose --version
```

---

## 진행 체크리스트

- [ ] DNS A 레코드 추가 (erp.abada.kr -> 61.245.248.246)
- [ ] DNS 전파 확인 (보통 5-30분)
- [ ] WSL2 SSH 서버 설정 (포트 5022)
- [ ] Windows 방화벽 5022 포트 오픈
- [ ] GitHub Repository 생성
- [ ] GitHub Secrets 추가
- [ ] SSH 키 쌍 생성 및 배포
- [ ] docs/.env_ssh 파일 생성
- [ ] WSL에 Docker 설치
- [ ] Let's Encrypt 인증서 발급

---

## 문제 발생 시 연락처

작업 중 문제가 발생하면:
1. 에러 메시지 캡처
2. 실행한 명령어 기록
3. Claude에게 문의

---

*Last Updated: 2026-01-16*
