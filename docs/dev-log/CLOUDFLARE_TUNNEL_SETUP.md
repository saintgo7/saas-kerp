# Cloudflare Tunnel 설정

> 포트 80/443 차단을 완전히 우회하는 방법 (무료)

---

## 현재 설정 정보

| 항목 | 값 |
|------|-----|
| Tunnel 이름 | `kerp-erp` |
| Tunnel ID | `10e537ac-f1e1-43e3-a2bd-3c86753378f2` |
| 도메인 | `erp.abada.kr` |
| 서비스 | `http://localhost:8080` |
| 상태 | 활성화됨 |

---

## 1. Cloudflare Dashboard에서 Tunnel 생성

1. https://one.dash.cloudflare.com 접속
2. **Networks** > **커넥터** (Connectors) 클릭
3. **Cloudflare Tunnel** > **Create a tunnel** 클릭
4. **Cloudflared** 선택 > Next
5. Tunnel 이름: `kerp-erp`
6. **Save tunnel**

---

## 2. 설치 토큰 복사

생성 후 나오는 화면에서:
```
cloudflared service install <TOKEN>
```
이 명령어의 `<TOKEN>` 부분을 복사

---

## 3. WSL에서 Tunnel 실행

```bash
# 토큰으로 서비스 설치
sudo cloudflared service install <YOUR_TOKEN>

# 서비스 시작
sudo systemctl start cloudflared
sudo systemctl enable cloudflared

# 상태 확인
sudo systemctl status cloudflared
```

---

## 4. Public Hostname 설정

Cloudflare Zero Trust Dashboard에서:
1. **Networks** > **커넥터** > Tunnel 클릭
2. **게시된 응용 프로그램 경로** (Public Hostnames) 탭
3. **게시된 응용 프로그램 경로 추가** 클릭:
   - Subdomain: `erp`
   - Domain: `abada.kr`
   - Service Type: `HTTP`
   - URL: `localhost:8080`
4. **저장**

---

## 5. DNS 자동 설정

Tunnel이 자동으로 DNS CNAME 레코드를 생성합니다:
```
erp.abada.kr -> 10e537ac-f1e1-43e3-a2bd-3c86753378f2.cfargotunnel.com
```

**중요:** 기존 DNS 레코드가 있으면 충돌 에러가 발생합니다.
- Cloudflare DNS에서 기존 A/CNAME 레코드 삭제 후 다시 시도

---

## 6. 테스트

```bash
# API 서버 실행 (WSL에서)
cd /path/to/project
go run cmd/api/main.go

# 다른 터미널에서 테스트
curl https://erp.abada.kr/health
```

예상 응답:
```json
{"status":"healthy","service":"kerp-api","version":"0.1.0"}
```

---

## 트러블슈팅

### DNS 레코드 충돌 에러
```
Error: An A, AAAA, or CNAME record with that host already exists.
```

해결: Cloudflare DNS 설정에서 해당 서브도메인의 기존 레코드 삭제

### 502 Bad Gateway
- WSL에서 API 서버가 실행 중인지 확인
- cloudflared 서비스 상태 확인: `sudo systemctl status cloudflared`

---

*2026-01-17 완료*
