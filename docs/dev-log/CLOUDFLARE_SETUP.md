# Cloudflare SSL 설정 가이드

> 무료 SSL + CDN + DDoS 방어

---

## 1. Cloudflare 계정 생성

1. https://dash.cloudflare.com/sign-up 접속
2. 이메일/비밀번호로 가입

---

## 2. 도메인 추가

1. **Add a Site** 클릭
2. 도메인 입력: `abada.kr`
3. **Free** 플랜 선택
4. **Continue**

---

## 3. DNS 레코드 설정

Cloudflare가 기존 DNS 레코드를 스캔합니다. 다음을 확인/추가:

| Type | Name | Content | Proxy |
|------|------|---------|-------|
| A | erp | 61.245.248.246 | Proxied (주황색 구름) |
| A | @ | (기존 값) | 선택 |

**중요**: `erp` 레코드의 Proxy 상태를 **Proxied** (주황색 구름)로 설정

---

## 4. 네임서버 변경

Cloudflare가 제공하는 네임서버로 변경:
```
예시:
ns1.cloudflare.com
ns2.cloudflare.com
```

**도메인 등록업체** (가비아, 카페24 등)에서:
1. 도메인 관리 > 네임서버 설정
2. Cloudflare 네임서버로 변경
3. 전파까지 최대 24시간 (보통 1-2시간)

---

## 5. SSL/TLS 설정

Cloudflare Dashboard > SSL/TLS:

1. **Overview** 탭:
   - SSL 모드: **Flexible** (원본 서버 HTTP만 필요)
   - 또는 **Full** (원본 서버 자체 서명 인증서)

2. **Edge Certificates** 탭:
   - Always Use HTTPS: **ON**
   - Automatic HTTPS Rewrites: **ON**
   - Minimum TLS Version: **TLS 1.2**

---

## 6. 캐시 설정 (선택)

Cloudflare Dashboard > Caching:

1. **Caching Level**: Standard
2. **Browser Cache TTL**: 4 hours
3. API 경로 캐시 제외 (Page Rules):
   - URL: `erp.abada.kr/api/*`
   - Setting: Cache Level = Bypass

---

## 7. 보안 설정 (선택)

Cloudflare Dashboard > Security:

1. **Security Level**: Medium
2. **Bot Fight Mode**: ON
3. **Browser Integrity Check**: ON

---

## 8. 설정 완료 후 테스트

```bash
# DNS 전파 확인
nslookup erp.abada.kr

# HTTPS 접속 테스트
curl -I https://erp.abada.kr
```

---

## Flexible vs Full SSL 모드

| 모드 | 설명 | 원본 서버 |
|------|------|-----------|
| **Flexible** | Cloudflare-브라우저만 암호화 | HTTP (80) |
| **Full** | 전 구간 암호화 (자체서명 OK) | HTTPS (443) |
| **Full (Strict)** | 전 구간 암호화 (유효 인증서) | HTTPS + 유효 인증서 |

**권장**: 처음엔 **Flexible**, 나중에 원본 인증서 설정 후 **Full**로 전환

---

## 트러블슈팅

### ERR_TOO_MANY_REDIRECTS
- SSL 모드가 **Full**인데 원본 서버가 HTTP만 지원
- 해결: SSL 모드를 **Flexible**로 변경

### 522 Connection Timed Out
- 원본 서버가 응답하지 않음
- 해결: 원본 서버 상태 확인, 방화벽 확인

### DNS 전파 지연
- 네임서버 변경 후 최대 48시간 소요
- 해결: 기다리거나 `nslookup` 으로 확인

---

*2026-01-16*
