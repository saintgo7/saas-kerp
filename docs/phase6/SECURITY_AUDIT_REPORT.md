# Phase 6 Security Audit Report

**Date:** 2026-01-18
**Auditor:** Claude Security Agent
**Scope:** python-services/insurance-edi/, python-services/shared/crypto/
**Status:** Pending Remediation

---

## Executive Summary

Phase 6 보험 EDI 구현에서 **18개 보안 취약점** 발견됨.
- Critical: 3건
- High: 6건
- Medium: 7건
- Low: 2건

**프로덕션 배포 전 Critical/High 취약점 반드시 수정 필요**

---

## Critical Findings (즉시 수정)

### C-1. SEED Cipher AES Placeholder
- **File:** `python-services/shared/crypto/seed.py:56-88`
- **Issue:** SEED 암호화가 AES로 대체되어 있음
- **Impact:** 정부 시스템 통신 불가, 데이터 손상
- **Fix:**
```python
# KISA 공식 SEED 라이브러리 사용 또는 직접 구현
# 프로덕션 가드 추가
if settings.environment == "production":
    raise RuntimeError("SEED cipher not implemented - cannot use in production")
```

### C-2. No TLS for EDI Connections
- **File:** `python-services/insurance-edi/edi/client.py:32`
- **Issue:** `ssl_enabled: bool = False` 기본값
- **Impact:** 주민번호, 급여정보 평문 전송 (MITM 취약)
- **Fix:**
```python
@dataclass
class ConnectionConfig:
    ssl_enabled: bool = True  # Change default
    ssl_verify_cert: bool = True
    ssl_ca_cert: Optional[str] = None
```

### C-3. Zero-Key Encryption Fallback
- **Files:**
  - `python-services/insurance-edi/providers/nps.py:84-90`
  - `python-services/insurance-edi/providers/nhis.py:77-82`
  - `python-services/insurance-edi/providers/ei.py:78-83`
  - `python-services/insurance-edi/services/insurance_service.py:51-57`
- **Issue:** 키 미설정시 `bytes(16)` 제로 키 사용
- **Impact:** 암호화 무효화
- **Fix:**
```python
def _get_encryption_key(self) -> bytes:
    key_hex = settings.crypto.aria_key
    if not key_hex:
        if settings.environment == "production":
            raise RuntimeError("ARIA_ENCRYPTION_KEY must be configured")
        logger.warning("Using placeholder key - NOT FOR PRODUCTION")
    return bytes.fromhex(key_hex) if key_hex else bytes(16)
```

---

## High Findings (배포 전 수정)

### H-1. Resident Number Logged in Plain Text
- **Files:**
  - `python-services/insurance-edi/providers/nps.py:104, 177`
  - `python-services/insurance-edi/providers/nhis.py:96, 166`
  - `python-services/insurance-edi/providers/ei.py:99, 198`
- **Issue:** `logger.info(..., data=data)` - data에 resident_no 포함
- **Impact:** 개인정보보호법 위반, 로그 유출시 주민번호 노출
- **Fix:**
```python
def _sanitize_log_data(self, data: Dict) -> Dict:
    sanitized = copy.deepcopy(data)
    if emp := sanitized.get("employee"):
        if "resident_no" in emp:
            emp["resident_no"] = emp["resident_no"][:6] + "-*******"
    return sanitized

logger.info("Submitting", data=self._sanitize_log_data(data))
```

### H-2. IV Reuse Vulnerability
- **File:** `python-services/shared/crypto/aria.py:390-391`
- **Issue:** `self._iv = iv or bytes(16)` - 제로 IV 기본값
- **Impact:** 동일 평문 블록 탐지 가능 (패턴 분석)
- **Fix:**
```python
def __init__(self, key: Union[bytes, str], iv: bytes = None):
    self._cipher = ARIACipher(key)
    if iv is None:
        from .utils import generate_iv
        iv = generate_iv(16)
    self._iv = iv
```

### H-3. No Certificate Validation
- **File:** `python-services/insurance-edi/edi/client.py:104-110`
- **Issue:** SSL 활성화시에도 인증서 검증 로직 없음
- **Impact:** MITM 공격으로 위조 인증서 사용 가능
- **Fix:**
```python
if self.config.ssl_enabled:
    ssl_context = ssl.create_default_context()
    if self.config.ssl_ca_cert:
        ssl_context.load_verify_locations(self.config.ssl_ca_cert)
    if not self.config.ssl_verify_cert:
        ssl_context.check_hostname = False
        ssl_context.verify_mode = ssl.CERT_NONE
        logger.warning("Certificate verification disabled!")
```

### H-4. gRPC Without Authentication
- **File:** `python-services/insurance-edi/main.py:62-63`
- **Issue:** `self.server.add_insecure_port(listen_addr)`
- **Impact:** 미인가 접근, 허위 보험 신고 가능
- **Fix:**
```python
# mTLS 또는 JWT 인증 추가
credentials = grpc.ssl_server_credentials(
    private_key_certificate_chain_pairs=[(private_key, certificate)]
)
self.server.add_secure_port(listen_addr, credentials)
```

### H-5. Exception Messages Leak Data
- **Files:** 모든 provider의 except 블록
- **Issue:** `error_message=str(e)` - 내부 오류 정보 클라이언트에 노출
- **Impact:** 시스템 정보 유출, 디버깅 정보 노출
- **Fix:**
```python
except Exception as e:
    logger.exception("Submission failed", error=str(e))
    return SubmissionResult(
        success=False,
        error_code="SUBMISSION_ERROR",
        error_message="An error occurred. Contact support.",  # Generic message
    ).to_dict()
```

### H-6. Encryption Key in Environment Variable
- **File:** `python-services/insurance-edi/config.py:31-32`
- **Issue:** `aria_key: Optional[str] = Field(env="ARIA_ENCRYPTION_KEY")`
- **Impact:** 환경변수는 프로세스 목록, 크래시 덤프에서 노출 가능
- **Fix:** HashiCorp Vault, AWS Secrets Manager 등 시크릿 관리 도구 사용

---

## Medium Findings (단기 수정)

### M-1. Padding Oracle Vulnerability
- **File:** `python-services/shared/crypto/pkcs7.py:72-88`
- **Issue:** 패딩 오류별 다른 에러 메시지 반환
- **Fix:** 모든 패딩 오류에 동일한 메시지 반환 + Encrypt-then-MAC 패턴 적용

### M-2. Key Not Zeroed from Memory
- **File:** `python-services/shared/crypto/aria.py:180-197`
- **Issue:** `self._key` 메모리에서 미삭제
- **Fix:** bytearray 사용 + `__del__` 메서드에서 secure_zero() 호출

### M-3. Insufficient RRN Validation
- **File:** `python-services/insurance-edi/forms/base.py:357-380`
- **Issue:** 주민번호 형식만 검증, 체크섬 미검증
- **Fix:** `validators.py:60-86`의 체크섬 검증 로직 사용

### M-4. Business Number Validation Bypass
- **File:** `python-services/insurance-edi/providers/base.py:191-212`
- **Issue:** 사업자번호 길이만 검증, 체크섬 미검증
- **Fix:** `validators.py:7-36`의 체크섬 검증 로직 사용

### M-5. Connection Timeout Too Long
- **File:** `python-services/insurance-edi/config.py:15-25`
- **Issue:** 기본 타임아웃 30초 (DoS 취약)
- **Fix:** 10초로 감소 + Circuit Breaker 패턴 구현

### M-6. No Rate Limiting
- **File:** `python-services/insurance-edi/services/insurance_service.py`
- **Issue:** gRPC 엔드포인트에 Rate Limiting 없음
- **Fix:** gRPC Interceptor로 Rate Limiting 구현

### M-7. gRPC Binds to 0.0.0.0
- **File:** `python-services/insurance-edi/config.py:48`
- **Issue:** `grpc_host: str = Field(default="0.0.0.0")`
- **Fix:** 기본값 `127.0.0.1`로 변경, 방화벽 규칙 설정

---

## Low Findings (장기 개선)

### L-1. HKDF Implementation Non-Standard
- **File:** `python-services/shared/crypto/utils.py:148-178`
- **Issue:** RFC 5869 미준수 HKDF 구현
- **Fix:** `cryptography.hazmat.primitives.kdf.hkdf.HKDF` 사용

### L-2. Debug Mode Available
- **File:** `python-services/insurance-edi/config.py:54`
- **Issue:** 프로덕션에서 DEBUG 모드 활성화 가능
- **Fix:** 프로덕션 환경에서 DEBUG 강제 비활성화 validator 추가

---

## Remediation Priority

### 1. Immediate (프로덕션 배포 차단)
- [ ] C-1: SEED 플레이스홀더 가드 추가
- [ ] C-2: TLS 기본 활성화
- [ ] C-3: 제로 키 프로덕션 차단

### 2. Short-term (1주일 내)
- [ ] H-1: 주민번호 로그 마스킹
- [ ] H-2: IV 자동 생성
- [ ] H-3: 인증서 검증 구현
- [ ] H-4: gRPC 인증 추가
- [ ] H-5: 에러 메시지 일반화
- [ ] H-6: 시크릿 관리 도구 연동

### 3. Medium-term (1개월 내)
- [ ] M-1 ~ M-7 모든 Medium 취약점

### 4. Long-term (분기 내)
- [ ] L-1 ~ L-2 모든 Low 취약점
- [ ] 키 로테이션 메커니즘
- [ ] 보안 모니터링/알림
- [ ] 침투 테스트

---

## Test Cases Required

```python
# tests/test_security.py

def test_no_resident_number_in_logs(caplog):
    """RRN이 로그에 마스킹되어 출력되는지 검증"""
    pass

def test_tls_required_in_production():
    """프로덕션에서 TLS 비활성화 불가 검증"""
    pass

def test_zero_key_rejected_in_production():
    """프로덕션에서 제로 키 사용 불가 검증"""
    pass

def test_padding_oracle_resistance():
    """모든 패딩 오류가 동일 메시지 반환 검증"""
    pass

def test_grpc_auth_required():
    """미인증 요청 거부 검증"""
    pass

def test_iv_uniqueness():
    """매 암호화마다 새 IV 생성 검증"""
    pass
```

---

## References

- OWASP Top 10 2021: https://owasp.org/Top10/
- Korean PIPA (개인정보보호법): https://www.law.go.kr
- KISA SEED/ARIA 표준: https://seed.kisa.or.kr
- gRPC Authentication: https://grpc.io/docs/guides/auth/

---

**Next Review Date:** Phase 6 수정 완료 후
