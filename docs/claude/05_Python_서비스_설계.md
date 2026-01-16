# K-ERP v0.2 - Python 서비스 설계

**문서 버전**: 0.2.0
**작성일**: 2026-01-16
**상태**: 검토 대기

---

## 목차

1. [개요](#1-개요)
2. [프로젝트 구조](#2-프로젝트-구조)
3. [Tax Scraper 서비스](#3-tax-scraper-서비스)
4. [Insurance EDI 서비스](#4-insurance-edi-서비스)
5. [공통 모듈](#5-공통-모듈)
6. [테스트 전략](#6-테스트-전략)

---

## 1. 개요

### 1.1 Python 서비스 역할

| 서비스 | 역할 | 포트 |
|--------|------|------|
| Tax Scraper | 홈택스 스크래핑, SEED 암호화 | 50051 |
| Insurance EDI | 4대보험 EDI, SEED/ARIA 암호화 | 50052 |

### 1.2 기술 스택

```
Python 3.11+
├── grpcio (1.60+)         # gRPC 서버/클라이언트
├── grpcio-tools           # Proto 컴파일
├── pycryptodome (3.20+)   # SEED/ARIA 암호화
├── cryptography (42+)     # PKCS#7 서명
├── playwright (1.40+)     # 웹 스크래핑
├── httpx (0.26+)          # 비동기 HTTP
├── pydantic (2.5+)        # 데이터 검증
├── structlog (24+)        # 구조화 로깅
└── pytest (8+)            # 테스트
```

---

## 2. 프로젝트 구조

```
python-services/
├── shared/                         # 공유 모듈
│   ├── proto/                     # 생성된 gRPC 코드
│   ├── crypto/                    # 암호화 모듈
│   │   ├── __init__.py
│   │   ├── seed.py               # SEED-CBC
│   │   ├── aria.py               # ARIA-CBC
│   │   └── pkcs7.py              # 전자서명
│   └── utils/                     # 유틸리티
│       ├── __init__.py
│       ├── config.py
│       └── logging.py
│
├── tax-scraper/                   # 세금계산서 스크래퍼
│   ├── src/
│   │   ├── __init__.py
│   │   ├── main.py               # 진입점
│   │   ├── server.py             # gRPC 서버
│   │   ├── scraper/
│   │   │   ├── __init__.py
│   │   │   ├── hometax.py        # 홈택스 스크래퍼
│   │   │   ├── auth.py           # 인증 처리
│   │   │   └── invoice.py        # 세금계산서 처리
│   │   └── models/
│   │       ├── __init__.py
│   │       └── invoice.py
│   ├── tests/
│   │   ├── test_scraper.py
│   │   └── test_crypto.py
│   ├── Dockerfile
│   ├── pyproject.toml
│   └── requirements.txt
│
└── insurance-edi/                 # 4대보험 EDI
    ├── src/
    │   ├── __init__.py
    │   ├── main.py
    │   ├── server.py
    │   ├── edi/
    │   │   ├── __init__.py
    │   │   ├── generator.py      # EDI 메시지 생성
    │   │   ├── parser.py         # EDI 응답 파싱
    │   │   └── sender.py         # EDI 전송
    │   ├── agency/
    │   │   ├── __init__.py
    │   │   ├── nps.py            # 국민연금
    │   │   ├── nhis.py           # 건강보험
    │   │   ├── ei.py             # 고용보험
    │   │   └── wci.py            # 산재보험
    │   └── models/
    │       └── insurance.py
    ├── tests/
    ├── Dockerfile
    └── requirements.txt
```

---

## 3. Tax Scraper 서비스

### 3.1 설정 모듈

```python
# tax-scraper/src/config.py
"""설정 모듈"""

from pydantic_settings import BaseSettings
from pydantic import Field


class Settings(BaseSettings):
    """서비스 설정"""

    # gRPC 서버
    grpc_port: int = Field(default=50051, env="GRPC_PORT")
    grpc_max_workers: int = Field(default=10, env="GRPC_MAX_WORKERS")

    # 홈택스
    hometax_timeout: int = Field(default=60, env="HOMETAX_TIMEOUT")
    hometax_headless: bool = Field(default=True, env="HOMETAX_HEADLESS")

    # 로깅
    log_level: str = Field(default="INFO", env="LOG_LEVEL")
    log_format: str = Field(default="json", env="LOG_FORMAT")

    # 재시도
    max_retries: int = Field(default=3, env="MAX_RETRIES")
    retry_delay: float = Field(default=1.0, env="RETRY_DELAY")

    class Config:
        env_file = ".env"


settings = Settings()
```

### 3.2 메인 진입점

```python
# tax-scraper/src/main.py
"""Tax Scraper 서비스 진입점"""

import asyncio
import signal
import sys
from concurrent import futures

import grpc
import structlog

from .config import settings
from .server import ScraperServicer
from shared.proto import scraper_pb2_grpc

logger = structlog.get_logger()


async def serve():
    """gRPC 서버 시작"""
    server = grpc.aio.server(
        futures.ThreadPoolExecutor(max_workers=settings.grpc_max_workers),
        options=[
            ("grpc.max_send_message_length", 50 * 1024 * 1024),
            ("grpc.max_receive_message_length", 50 * 1024 * 1024),
            ("grpc.keepalive_time_ms", 10000),
            ("grpc.keepalive_timeout_ms", 5000),
            ("grpc.keepalive_permit_without_calls", True),
        ],
    )

    # 서비스 등록
    scraper_pb2_grpc.add_ScraperServiceServicer_to_server(
        ScraperServicer(), server
    )

    # 리플렉션 활성화 (개발용)
    from grpc_reflection.v1alpha import reflection
    SERVICE_NAMES = (
        scraper_pb2_grpc.DESCRIPTOR.services_by_name["ScraperService"].full_name,
        reflection.SERVICE_NAME,
    )
    reflection.enable_server_reflection(SERVICE_NAMES, server)

    # 서버 시작
    listen_addr = f"[::]:{settings.grpc_port}"
    server.add_insecure_port(listen_addr)

    logger.info("starting_server", port=settings.grpc_port)
    await server.start()

    # 종료 시그널 처리
    async def shutdown():
        logger.info("shutting_down")
        await server.stop(grace=10)

    loop = asyncio.get_event_loop()
    for sig in (signal.SIGTERM, signal.SIGINT):
        loop.add_signal_handler(sig, lambda: asyncio.create_task(shutdown()))

    await server.wait_for_termination()


def main():
    """메인 함수"""
    structlog.configure(
        processors=[
            structlog.processors.add_log_level,
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.JSONRenderer()
            if settings.log_format == "json"
            else structlog.dev.ConsoleRenderer(),
        ],
        wrapper_class=structlog.make_filtering_bound_logger(
            getattr(structlog, settings.log_level.upper())
        ),
    )

    try:
        asyncio.run(serve())
    except KeyboardInterrupt:
        logger.info("interrupted")
        sys.exit(0)


if __name__ == "__main__":
    main()
```

### 3.3 홈택스 스크래퍼 상세 구현

```python
# tax-scraper/src/scraper/hometax.py
"""홈택스 세금계산서 스크래퍼"""

import asyncio
from dataclasses import dataclass
from datetime import date, datetime
from typing import Optional, List
from contextlib import asynccontextmanager

from playwright.async_api import (
    async_playwright,
    Browser,
    BrowserContext,
    Page,
    PlaywrightContextManager,
)
import structlog

from shared.crypto.seed import SEEDCipher
from ..config import settings

logger = structlog.get_logger()


@dataclass
class HometaxCredential:
    """홈택스 인증 정보"""
    cert_der: bytes          # 인증서 DER
    key_encrypted: bytes     # 암호화된 개인키
    password: str            # 비밀번호
    business_number: str     # 사업자번호


@dataclass
class SearchCriteria:
    """조회 조건"""
    start_date: date
    end_date: date
    invoice_type: str = "all"  # "sales" | "purchase" | "all"
    page: int = 1
    page_size: int = 100


class HometaxScraper:
    """홈택스 스크래퍼"""

    BASE_URL = "https://www.hometax.go.kr"
    LOGIN_URL = f"{BASE_URL}/websquare/websquare.wq?w2xPath=/ui/pp/index_pp.xml"

    def __init__(self, credential: HometaxCredential):
        self.credential = credential
        self._playwright: Optional[PlaywrightContextManager] = None
        self._browser: Optional[Browser] = None
        self._context: Optional[BrowserContext] = None
        self._page: Optional[Page] = None
        self._logged_in = False

    @asynccontextmanager
    async def session(self):
        """브라우저 세션 컨텍스트"""
        try:
            await self._start_browser()
            yield self
        finally:
            await self._close_browser()

    async def _start_browser(self):
        """브라우저 시작"""
        self._playwright = await async_playwright().start()
        self._browser = await self._playwright.chromium.launch(
            headless=settings.hometax_headless,
            args=[
                "--no-sandbox",
                "--disable-dev-shm-usage",
                "--disable-gpu",
                "--disable-web-security",
            ],
        )
        self._context = await self._browser.new_context(
            viewport={"width": 1920, "height": 1080},
            locale="ko-KR",
        )
        self._page = await self._context.new_page()

        # 타임아웃 설정
        self._page.set_default_timeout(settings.hometax_timeout * 1000)

        logger.info("browser_started", headless=settings.hometax_headless)

    async def _close_browser(self):
        """브라우저 종료"""
        if self._page:
            await self._page.close()
        if self._context:
            await self._context.close()
        if self._browser:
            await self._browser.close()
        if self._playwright:
            await self._playwright.stop()

        self._logged_in = False
        logger.info("browser_closed")

    async def login(self) -> bool:
        """공인인증서 로그인"""
        if self._logged_in:
            return True

        logger.info("login_attempt", business_number=self.credential.business_number)

        try:
            # 로그인 페이지 이동
            await self._page.goto(self.LOGIN_URL)
            await self._page.wait_for_load_state("networkidle")

            # 공인인증서 로그인 탭 선택
            await self._page.click("text=공동인증서(구 공인인증서) 로그인")
            await asyncio.sleep(1)

            # 인증서 선택 및 로그인 실행
            await self._execute_cert_login()

            # 로그인 완료 확인
            await self._page.wait_for_selector(
                "#utl_head_menu_area",
                timeout=30000
            )

            self._logged_in = True
            logger.info("login_success")
            return True

        except Exception as e:
            logger.error("login_failed", error=str(e))
            raise HometaxLoginError(f"Login failed: {e}")

    async def _execute_cert_login(self):
        """인증서 로그인 실행"""
        # 인증서 데이터 준비
        cert_data = self._prepare_certificate()

        # JavaScript로 인증서 로그인 실행
        await self._page.evaluate(
            """
            async (certData) => {
                // 홈택스 내부 인증서 처리 함수 호출
                // 실제 구현은 홈택스 JavaScript API 분석 필요
                return await window.NX_CERTIFICATE_LOGIN(certData);
            }
            """,
            cert_data,
        )

        await asyncio.sleep(2)

    def _prepare_certificate(self) -> dict:
        """인증서 데이터 준비"""
        # 개인키 복호화
        key = self._derive_key(self.credential.password)
        cipher = SEEDCipher(key)
        decrypted_key = cipher.decrypt(self.credential.key_encrypted)

        return {
            "cert": self.credential.cert_der.hex(),
            "key": decrypted_key.hex(),
            "password": self.credential.password,
        }

    def _derive_key(self, password: str) -> bytes:
        """비밀번호에서 키 파생"""
        import hashlib
        return hashlib.pbkdf2_hmac(
            "sha256",
            password.encode("utf-8"),
            b"hometax_salt",
            10000,
            dklen=16
        )

    async def search_invoices(self, criteria: SearchCriteria) -> List[dict]:
        """세금계산서 목록 조회"""
        if not self._logged_in:
            await self.login()

        logger.info(
            "search_invoices",
            start_date=criteria.start_date.isoformat(),
            end_date=criteria.end_date.isoformat(),
            invoice_type=criteria.invoice_type,
        )

        # 세금계산서 조회 페이지 이동
        invoice_url = f"{self.BASE_URL}/websquare/websquare.wq?w2xPath=/ui/ab/a/a/UTECAAB001.xml"
        await self._page.goto(invoice_url)
        await self._page.wait_for_load_state("networkidle")

        # 조회 조건 입력
        await self._set_search_criteria(criteria)

        # 조회 실행
        await self._page.click("#btnSearch")
        await self._page.wait_for_selector("#grid1", timeout=30000)

        # 결과 파싱
        return await self._parse_invoice_list()

    async def _set_search_criteria(self, criteria: SearchCriteria):
        """조회 조건 설정"""
        # 시작일
        await self._page.fill(
            "#startDate",
            criteria.start_date.strftime("%Y%m%d")
        )
        # 종료일
        await self._page.fill(
            "#endDate",
            criteria.end_date.strftime("%Y%m%d")
        )

        # 매출/매입 구분
        if criteria.invoice_type == "sales":
            await self._page.click("#rbSales")
        elif criteria.invoice_type == "purchase":
            await self._page.click("#rbPurchase")

    async def _parse_invoice_list(self) -> List[dict]:
        """조회 결과 파싱"""
        invoices = []

        rows = await self._page.query_selector_all("#grid1 tbody tr")
        for row in rows:
            cells = await row.query_selector_all("td")
            if len(cells) < 6:
                continue

            invoice = {
                "nts_confirm_num": await self._get_text(cells[0]),
                "issue_date": await self._get_text(cells[1]),
                "supplier_brn": await self._get_text(cells[2]),
                "supplier_name": await self._get_text(cells[3]),
                "supply_amount": self._parse_amount(await self._get_text(cells[4])),
                "tax_amount": self._parse_amount(await self._get_text(cells[5])),
            }
            invoices.append(invoice)

        logger.info("invoices_found", count=len(invoices))
        return invoices

    async def _get_text(self, element) -> str:
        """요소에서 텍스트 추출"""
        text = await element.inner_text()
        return text.strip() if text else ""

    def _parse_amount(self, text: str) -> int:
        """금액 문자열 파싱"""
        cleaned = text.replace(",", "").replace(" ", "")
        return int(cleaned) if cleaned.isdigit() else 0

    async def issue_invoice(self, invoice_data: dict) -> dict:
        """세금계산서 발행"""
        if not self._logged_in:
            await self.login()

        logger.info(
            "issue_invoice",
            buyer_brn=invoice_data.get("buyer_brn"),
            amount=invoice_data.get("total_amount"),
        )

        # 발행 페이지 이동
        issue_url = f"{self.BASE_URL}/websquare/websquare.wq?w2xPath=/ui/ab/a/b/UTECAAB002.xml"
        await self._page.goto(issue_url)
        await self._page.wait_for_load_state("networkidle")

        # 발행 양식 작성
        await self._fill_issue_form(invoice_data)

        # 전자서명 및 발행
        nts_confirm_num = await self._sign_and_issue()

        return {
            "success": True,
            "nts_confirm_num": nts_confirm_num,
            "issued_at": datetime.now().isoformat(),
        }

    async def _fill_issue_form(self, data: dict):
        """발행 양식 작성"""
        # 공급받는자 정보
        await self._page.fill("#buyerBrn", data["buyer_brn"])
        await self._page.fill("#buyerName", data["buyer_name"])
        await self._page.fill("#buyerEmail", data.get("buyer_email", ""))

        # 금액
        await self._page.fill("#supplyAmount", str(data["supply_amount"]))
        await self._page.fill("#taxAmount", str(data["tax_amount"]))

        # 품목
        for idx, item in enumerate(data.get("items", [])):
            await self._add_invoice_item(idx, item)

    async def _add_invoice_item(self, idx: int, item: dict):
        """품목 추가"""
        prefix = f"#item_{idx}_"
        await self._page.fill(f"{prefix}name", item["item_name"])
        await self._page.fill(f"{prefix}qty", str(item["quantity"]))
        await self._page.fill(f"{prefix}price", str(item["unit_price"]))
        await self._page.fill(f"{prefix}supply", str(item["supply_amount"]))
        await self._page.fill(f"{prefix}tax", str(item["tax_amount"]))

    async def _sign_and_issue(self) -> str:
        """전자서명 및 발행"""
        # 발행 버튼 클릭
        await self._page.click("#btnIssue")

        # 전자서명 대화상자 처리
        await self._page.wait_for_selector("#signDialog", timeout=10000)
        await self._page.click("#btnSign")

        # 발행 완료 대기
        await self._page.wait_for_selector("#issueResult", timeout=60000)

        # 승인번호 추출
        nts_num_element = await self._page.query_selector("#ntsConfirmNum")
        nts_confirm_num = await nts_num_element.inner_text()

        logger.info("invoice_issued", nts_confirm_num=nts_confirm_num)
        return nts_confirm_num.strip()


class HometaxLoginError(Exception):
    """홈택스 로그인 오류"""
    pass


class HometaxScrapingError(Exception):
    """홈택스 스크래핑 오류"""
    pass
```

---

## 4. Insurance EDI 서비스

### 4.1 국민연금 클라이언트

```python
# insurance-edi/src/agency/nps.py
"""국민연금공단 클라이언트"""

import asyncio
from datetime import datetime
from typing import Optional

import httpx
import structlog

from shared.crypto.seed import SEEDCipher
from shared.crypto.pkcs7 import PKCSSignature
from ..edi.generator import NPSEDIGenerator
from ..edi.parser import NPSEDIParser

logger = structlog.get_logger()


class NPSClient:
    """국민연금공단 EDI 클라이언트"""

    EDI_URL = "https://edi.nps.or.kr/edi/receive"

    def __init__(self, config: dict):
        self.config = config
        self.generator = NPSEDIGenerator()
        self.parser = NPSEDIParser()
        self._http_client: Optional[httpx.AsyncClient] = None

    async def __aenter__(self):
        self._http_client = httpx.AsyncClient(
            timeout=60.0,
            verify=True,
        )
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self._http_client:
            await self._http_client.aclose()

    async def report_acquisition(self, request) -> dict:
        """취득신고"""
        logger.info(
            "nps_acquisition",
            employee=request.employee.name,
            date=request.acquisition_date,
        )

        # EDI 메시지 생성
        edi_message = self.generator.generate_acquisition(
            workplace_code=request.business.workplace_code,
            employees=[{
                "resident_number": request.employee.resident_number,
                "name": request.employee.name,
                "acquisition_date": request.acquisition_date,
                "monthly_salary": request.monthly_salary,
            }],
        )

        # 암호화
        encrypted = await self._encrypt_message(edi_message, request.certificate)

        # 전자서명
        signed = await self._sign_message(encrypted, request.certificate)

        # 전송
        response = await self._send_edi(signed)

        # 응답 파싱
        result = self.parser.parse_response(response)

        return {
            "success": result["success"],
            "transaction_id": result.get("transaction_id"),
            "processed_at": datetime.now().isoformat(),
        }

    async def report_loss(self, request) -> dict:
        """상실신고"""
        logger.info(
            "nps_loss",
            employee=request.employee.name,
            date=request.loss_date,
        )

        edi_message = self.generator.generate_loss(
            workplace_code=request.business.workplace_code,
            employees=[{
                "resident_number": request.employee.resident_number,
                "name": request.employee.name,
                "loss_date": request.loss_date,
                "loss_reason": request.reason,
                "total_salary": request.total_salary,
            }],
        )

        encrypted = await self._encrypt_message(edi_message, request.certificate)
        signed = await self._sign_message(encrypted, request.certificate)
        response = await self._send_edi(signed)
        result = self.parser.parse_response(response)

        return {
            "success": result["success"],
            "transaction_id": result.get("transaction_id"),
            "processed_at": datetime.now().isoformat(),
        }

    async def get_premium(self, request) -> dict:
        """보험료 조회"""
        # 웹 스크래핑 또는 API 호출
        # ... 구현
        pass

    async def _encrypt_message(self, message: bytes, cert_info) -> bytes:
        """EDI 메시지 암호화 (SEED-CBC)"""
        key = self._derive_encryption_key(cert_info.password)
        cipher = SEEDCipher(key)
        encrypted, iv = cipher.encrypt(message)

        # IV + 암호문 결합
        return iv + encrypted

    async def _sign_message(self, data: bytes, cert_info) -> bytes:
        """전자서명 (PKCS#7)"""
        signer = PKCSSignature(
            cert_der=cert_info.cert_der_base64,
            key_encrypted=cert_info.key_encrypted_base64,
            password=cert_info.password,
        )

        signature = signer.sign(data)

        # 서명 + 데이터 결합
        return signature + data

    async def _send_edi(self, data: bytes) -> bytes:
        """EDI 전송"""
        headers = {
            "Content-Type": "application/octet-stream",
            "X-EDI-Version": "1.0",
        }

        response = await self._http_client.post(
            self.EDI_URL,
            content=data,
            headers=headers,
        )

        if response.status_code != 200:
            raise EDISendError(f"EDI send failed: {response.status_code}")

        return response.content

    def _derive_encryption_key(self, password: str) -> bytes:
        """암호화 키 파생"""
        import hashlib
        return hashlib.pbkdf2_hmac(
            "sha256",
            password.encode("utf-8"),
            b"nps_edi_salt",
            10000,
            dklen=16
        )


class EDISendError(Exception):
    """EDI 전송 오류"""
    pass
```

---

## 5. 공통 모듈

### 5.1 의존성 파일

```txt
# shared/requirements.txt
grpcio>=1.60.0
grpcio-tools>=1.60.0
grpcio-reflection>=1.60.0
pycryptodome>=3.20.0
cryptography>=42.0.0
pydantic>=2.5.0
pydantic-settings>=2.1.0
structlog>=24.1.0
httpx>=0.26.0
```

```txt
# tax-scraper/requirements.txt
-r ../shared/requirements.txt
playwright>=1.40.0
```

```txt
# insurance-edi/requirements.txt
-r ../shared/requirements.txt
```

---

## 6. 테스트 전략

### 6.1 단위 테스트

```python
# tax-scraper/tests/test_crypto.py
"""암호화 테스트"""

import pytest
from shared.crypto.seed import SEEDCipher


class TestSEEDCipher:
    """SEED 암호화 테스트"""

    def test_encrypt_decrypt(self):
        """암호화/복호화 테스트"""
        key = b"0123456789abcdef"
        iv = b"fedcba9876543210"

        cipher = SEEDCipher(key, iv)
        plaintext = b"Hello, World! This is a test message."

        # 암호화
        ciphertext, returned_iv = cipher.encrypt(plaintext)
        assert ciphertext != plaintext
        assert returned_iv == iv

        # 복호화
        decrypted = cipher.decrypt(ciphertext, iv)
        assert decrypted == plaintext

    def test_invalid_key_length(self):
        """잘못된 키 길이 테스트"""
        with pytest.raises(ValueError):
            SEEDCipher(b"short_key", b"1234567890123456")

    def test_pkcs7_padding(self):
        """PKCS7 패딩 테스트"""
        key = b"0123456789abcdef"
        cipher = SEEDCipher(key)

        # 다양한 길이의 입력
        for length in [1, 15, 16, 17, 31, 32, 33]:
            plaintext = b"x" * length
            ciphertext, iv = cipher.encrypt(plaintext)
            decrypted = cipher.decrypt(ciphertext, iv)
            assert decrypted == plaintext
```

### 6.2 통합 테스트

```python
# tax-scraper/tests/test_integration.py
"""통합 테스트"""

import pytest
from unittest.mock import AsyncMock, patch

from src.scraper.hometax import HometaxScraper, HometaxCredential


@pytest.fixture
def mock_credential():
    return HometaxCredential(
        cert_der=b"mock_cert",
        key_encrypted=b"mock_key",
        password="test_password",
        business_number="1234567890",
    )


class TestHometaxScraper:
    """홈택스 스크래퍼 테스트"""

    @pytest.mark.asyncio
    async def test_login_success(self, mock_credential):
        """로그인 성공 테스트"""
        with patch("playwright.async_api.async_playwright") as mock_pw:
            # Mock 설정
            mock_browser = AsyncMock()
            mock_page = AsyncMock()
            mock_page.goto = AsyncMock()
            mock_page.click = AsyncMock()
            mock_page.wait_for_selector = AsyncMock()
            mock_page.evaluate = AsyncMock()
            mock_browser.new_context.return_value = AsyncMock(
                new_page=AsyncMock(return_value=mock_page)
            )
            mock_pw.return_value.__aenter__.return_value.chromium.launch = (
                AsyncMock(return_value=mock_browser)
            )

            scraper = HometaxScraper(mock_credential)
            async with scraper.session():
                result = await scraper.login()
                assert result is True
```

---

**다음 문서**: [06_배포_설정.md](./06_배포_설정.md)
