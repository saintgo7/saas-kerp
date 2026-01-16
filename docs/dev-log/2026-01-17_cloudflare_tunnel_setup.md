# 2026-01-17 Cloudflare Tunnel 설정 완료

## 요약

ISP 포트 80/443 차단 문제를 Cloudflare Tunnel (무료)로 해결

## 문제 상황

- ISP (SK 브로드밴드)에서 포트 80/443 차단
- Let's Encrypt 인증서 발급 불가
- 기존 A 레코드 방식으로는 접근 불가 (522 에러)

## 해결 방안

Cloudflare Tunnel 사용 (Zero Trust > Networks > Connectors)

## 진행 과정

### 1. Cloudflare Tunnel 생성
- Tunnel 이름: `kerp-erp`
- Tunnel ID: `10e537ac-f1e1-43e3-a2bd-3c86753378f2`

### 2. WSL에 cloudflared 설치
```bash
sudo cloudflared service install <TOKEN>
sudo systemctl start cloudflared
sudo systemctl enable cloudflared
```

### 3. DNS 레코드 충돌 해결
- 기존 erp CNAME 레코드 삭제 (Cloudflare DNS)
- Tunnel이 자동으로 CNAME 생성

### 4. Public Hostname 설정
- 도메인: `erp.abada.kr`
- 서비스: `http://localhost:8080`

## 결과

| 항목 | 상태 |
|------|------|
| Cloudflare Tunnel | 활성화 |
| cloudflared 서비스 | 실행 중 |
| DNS CNAME | 자동 생성됨 |
| HTTPS 접근 | erp.abada.kr 사용 가능 |

## 아키텍처

```
[인터넷] --> [Cloudflare Edge] --> [Tunnel] --> [WSL2:8080]
                  |
             HTTPS 자동 처리
```

## 장점

1. 포트 80/443 오픈 불필요
2. HTTPS 인증서 자동 관리
3. DDoS 보호 기본 제공
4. 무료

## 다음 단계

1. WSL에서 API 서버 실행
2. https://erp.abada.kr/health 테스트
3. CI/CD 파이프라인 구축

## 관련 문서

- `CLOUDFLARE_TUNNEL_SETUP.md` - 상세 설정 가이드
- `USER_ACTION_REQUIRED.md` - 체크리스트 업데이트

---

*작성: Claude Code*
