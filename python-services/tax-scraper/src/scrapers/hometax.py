"""Hometax (Korean Tax Authority) scraper using Playwright."""

import asyncio
from dataclasses import dataclass
from datetime import date, datetime
from enum import Enum
from typing import Any

from loguru import logger
from playwright.async_api import Browser, Page, Playwright, async_playwright
from tenacity import retry, stop_after_attempt, wait_exponential

from config.settings import get_settings


class HometaxLoginType(Enum):
    """Hometax login types."""

    CERTIFICATE = "certificate"  # Digital certificate
    SIMPLE = "simple"  # Simple login (ID/PW)
    KAKAO = "kakao"  # Kakao authentication
    NAVER = "naver"  # Naver authentication


@dataclass
class HometaxCredentials:
    """Credentials for Hometax authentication."""

    login_type: HometaxLoginType
    cert_path: str | None = None
    cert_password: str | None = None
    user_id: str | None = None
    user_password: str | None = None


@dataclass
class TaxInvoiceSearchResult:
    """Tax invoice search result."""

    nts_confirm_number: str
    issue_date: date
    supplier_business_number: str
    supplier_name: str
    buyer_business_number: str
    buyer_name: str
    supply_amount: float
    tax_amount: float
    total_amount: float
    status: str


class HometaxScraper:
    """Hometax web scraper using Playwright."""

    BASE_URL = "https://www.hometax.go.kr"
    LOGIN_URL = f"{BASE_URL}/websquare/websquare.wq?w2xPath=/ui/pp/index_pp.xml"
    EINVOICE_URL = f"{BASE_URL}/websquare/websquare.wq?w2xPath=/ui/pp/UTXPPABA01.xml"

    def __init__(self, credentials: HometaxCredentials) -> None:
        """Initialize Hometax scraper."""
        self.credentials = credentials
        self.settings = get_settings()
        self._playwright: Playwright | None = None
        self._browser: Browser | None = None
        self._page: Page | None = None
        self._logged_in = False

    async def __aenter__(self) -> "HometaxScraper":
        """Async context manager entry."""
        await self.start()
        return self

    async def __aexit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        """Async context manager exit."""
        await self.close()

    async def start(self) -> None:
        """Start Playwright browser."""
        logger.info("Starting Playwright browser...")
        self._playwright = await async_playwright().start()
        self._browser = await self._playwright.chromium.launch(
            headless=self.settings.playwright_headless,
            slow_mo=self.settings.playwright_slow_mo,
        )
        self._page = await self._browser.new_page()
        await self._page.set_viewport_size({"width": 1920, "height": 1080})
        logger.info("Browser started successfully")

    async def close(self) -> None:
        """Close browser and cleanup."""
        if self._browser:
            await self._browser.close()
        if self._playwright:
            await self._playwright.stop()
        self._logged_in = False
        logger.info("Browser closed")

    @property
    def page(self) -> Page:
        """Get current page."""
        if not self._page:
            raise RuntimeError("Browser not started. Call start() first.")
        return self._page

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=2, max=10),
    )
    async def login(self) -> bool:
        """Login to Hometax."""
        logger.info(f"Logging into Hometax with {self.credentials.login_type.value}...")

        try:
            await self.page.goto(self.LOGIN_URL, timeout=self.settings.hometax_login_timeout)
            await self.page.wait_for_load_state("networkidle")

            if self.credentials.login_type == HometaxLoginType.CERTIFICATE:
                return await self._login_with_certificate()
            elif self.credentials.login_type == HometaxLoginType.SIMPLE:
                return await self._login_with_simple()
            else:
                raise NotImplementedError(
                    f"Login type {self.credentials.login_type} not implemented"
                )

        except Exception as e:
            logger.error(f"Login failed: {e}")
            raise

    async def _login_with_certificate(self) -> bool:
        """Login using digital certificate."""
        logger.info("Attempting certificate login...")

        # Click certificate login button
        await self.page.click('a:has-text("공동인증서")')
        await asyncio.sleep(2)

        # Certificate selection dialog handling would go here
        # This is a stub - actual implementation depends on certificate type
        # and requires handling native dialogs or browser extensions

        logger.warning("Certificate login requires manual interaction or extension")
        return False

    async def _login_with_simple(self) -> bool:
        """Login using ID and password."""
        if not self.credentials.user_id or not self.credentials.user_password:
            raise ValueError("User ID and password required for simple login")

        logger.info("Attempting simple login...")

        # Click simple login tab
        await self.page.click('a:has-text("아이디 로그인")')
        await asyncio.sleep(1)

        # Fill credentials
        await self.page.fill('input[name="userId"]', self.credentials.user_id)
        await self.page.fill('input[name="userPwd"]', self.credentials.user_password)

        # Click login button
        await self.page.click('button:has-text("로그인")')
        await self.page.wait_for_load_state("networkidle")

        # Check if login was successful
        if await self._check_login_success():
            self._logged_in = True
            logger.info("Login successful")
            return True

        logger.error("Login failed - invalid credentials or captcha required")
        return False

    async def _check_login_success(self) -> bool:
        """Check if login was successful."""
        try:
            # Look for logout button or user name indicator
            await self.page.wait_for_selector('a:has-text("로그아웃")', timeout=5000)
            return True
        except Exception:
            return False

    async def search_tax_invoices(
        self,
        start_date: date,
        end_date: date,
        direction: str = "sales",  # sales or purchase
    ) -> list[TaxInvoiceSearchResult]:
        """Search tax invoices within date range."""
        if not self._logged_in:
            raise RuntimeError("Not logged in. Call login() first.")

        logger.info(f"Searching tax invoices from {start_date} to {end_date}")

        # Navigate to e-invoice page
        await self.page.goto(self.EINVOICE_URL, timeout=self.settings.playwright_timeout)
        await self.page.wait_for_load_state("networkidle")

        # Select direction (sales/purchase)
        if direction == "sales":
            await self.page.click('input[value="01"]')  # Sales
        else:
            await self.page.click('input[value="02"]')  # Purchase

        # Fill date range
        await self.page.fill('input[name="startDt"]', start_date.strftime("%Y%m%d"))
        await self.page.fill('input[name="endDt"]', end_date.strftime("%Y%m%d"))

        # Click search
        await self.page.click('button:has-text("조회")')
        await self.page.wait_for_load_state("networkidle")

        # Parse results
        results = await self._parse_search_results()
        logger.info(f"Found {len(results)} tax invoices")

        return results

    async def _parse_search_results(self) -> list[TaxInvoiceSearchResult]:
        """Parse search results from the page."""
        results: list[TaxInvoiceSearchResult] = []

        # This is a stub - actual implementation would parse the HTML table
        # The structure depends on Hometax's current page layout

        try:
            rows = await self.page.query_selector_all("table.result tbody tr")

            for row in rows:
                cells = await row.query_selector_all("td")
                if len(cells) >= 9:
                    results.append(
                        TaxInvoiceSearchResult(
                            nts_confirm_number=await self._get_text(cells[0]),
                            issue_date=datetime.strptime(
                                await self._get_text(cells[1]), "%Y-%m-%d"
                            ).date(),
                            supplier_business_number=await self._get_text(cells[2]),
                            supplier_name=await self._get_text(cells[3]),
                            buyer_business_number=await self._get_text(cells[4]),
                            buyer_name=await self._get_text(cells[5]),
                            supply_amount=self._parse_amount(await self._get_text(cells[6])),
                            tax_amount=self._parse_amount(await self._get_text(cells[7])),
                            total_amount=self._parse_amount(await self._get_text(cells[8])),
                            status="issued",
                        )
                    )
        except Exception as e:
            logger.warning(f"Error parsing results: {e}")

        return results

    async def _get_text(self, element: Any) -> str:
        """Get text content from element."""
        return (await element.text_content() or "").strip()

    def _parse_amount(self, text: str) -> float:
        """Parse amount string to float."""
        try:
            return float(text.replace(",", "").replace("원", ""))
        except ValueError:
            return 0.0

    async def get_invoice_detail(self, nts_confirm_number: str) -> dict[str, Any] | None:
        """Get detailed tax invoice by NTS confirmation number."""
        if not self._logged_in:
            raise RuntimeError("Not logged in. Call login() first.")

        logger.info(f"Getting invoice detail for {nts_confirm_number}")

        # Navigate to detail page or search by confirmation number
        # This is a stub - actual implementation would navigate and parse

        return None

    async def download_invoice_pdf(
        self, nts_confirm_number: str, output_path: str
    ) -> str | None:
        """Download tax invoice as PDF."""
        if not self._logged_in:
            raise RuntimeError("Not logged in. Call login() first.")

        logger.info(f"Downloading PDF for {nts_confirm_number}")

        # This is a stub - actual implementation would trigger download

        return None
