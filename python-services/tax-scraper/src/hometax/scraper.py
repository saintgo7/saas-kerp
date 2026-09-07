"""
Hometax Web Scraper using Playwright

Handles authentication and tax invoice operations with Korean National Tax Service.

FAIL-CLOSED BY DEFAULT
----------------------
None of the scraping paths below have been validated against the live Hometax
site. The selectors in `constants.py` are guesses at a WebSquare DOM nobody has
verified, so every path that touches the site is gated behind
`settings.hometax_live_mode` (env `HOMETAX_LIVE_MODE`), which defaults to off.

With live mode off, each path raises `HometaxNotEnabledError`. That is the
correction: this module used to return `success=True` with a hardcoded
`invoice_number="20240115-12345678"` and `nts_confirm_number="NTS-CONFIRM-12345"`
while every line of actual scraping sat commented out, and to hand out valid
sessions for any credentials at all without contacting Hometax.

With live mode on, the real flow runs through the page objects in
`pages/`. Those page objects now fail closed too: an outcome that cannot be
confirmed is an error, not a success.
"""
import asyncio
import uuid
from dataclasses import dataclass
from datetime import date, datetime, timedelta
from typing import Any, Optional

import structlog
from playwright.async_api import (
    async_playwright,
    Browser,
    BrowserContext,
    Page,
    Playwright,
)

from config import get_settings
from .errors import (
    HometaxLoginError,
    HometaxNotEnabledError,
    HometaxNotImplementedError,
    HometaxResultUndeterminedError,
    HometaxSessionError,
)
from .models import (
    AuthType,
    HometaxSession,
    InvoiceType,
    IssuedInvoiceResult,
    TaxInvoice,
)
from .pages import LoginPage, TaxInvoiceIssuePage, TaxInvoiceSearchPage

logger = structlog.get_logger()


@dataclass
class _ScraperSession:
    """A live browser context plus the tenant and expiry that scope it."""

    company_id: str
    context: BrowserContext
    business_number: str
    expires_at: datetime

    def is_expired(self, now: Optional[datetime] = None) -> bool:
        """Whether this session has passed its expiry."""
        return (now or datetime.now()) >= self.expires_at


class HometaxScraper:
    """
    Hometax web scraper for tax invoice operations.

    Uses Playwright for browser automation to interact with the Korean
    National Tax Service (NTS) Hometax system.

    Sessions are keyed by (company_id, session_id). A session_id alone is not
    enough to reach a session: without the company, one tenant holding another
    tenant's session id could operate on their 세금계산서.
    """

    def __init__(self) -> None:
        """Initialize the scraper."""
        self.settings = get_settings()
        self.log = logger.bind(component="HometaxScraper")
        self._playwright: Optional[Playwright] = None
        self._browser: Optional[Browser] = None
        self._sessions: dict[tuple[str, str], _ScraperSession] = {}
        # Guards lazy browser startup: two concurrent requests could each launch
        # a browser, and one of them would be leaked with no reference to close.
        self._browser_lock = asyncio.Lock()
        self._session_lock = asyncio.Lock()

    def _require_live_mode(self, operation: str) -> None:
        """
        Refuse an operation unless live mode has been explicitly enabled.

        Args:
            operation: Human-readable operation name for the error message

        Raises:
            HometaxNotEnabledError: When live mode is off
        """
        if not self.settings.hometax_live_mode:
            raise HometaxNotEnabledError(
                f"{operation} is disabled: Hometax live mode is off. The scraping "
                f"selectors have not been validated against the live site, so this "
                f"path cannot report a trustworthy result. Set HOMETAX_LIVE_MODE=true "
                f"only after verifying them, or route this operation through the "
                f"Popbill provider."
            )

    async def _get_browser(self) -> Browser:
        """
        Get or create the browser instance.

        Returns:
            A connected Playwright browser
        """
        async with self._browser_lock:
            if self._browser is not None and self._browser.is_connected():
                return self._browser

            if self._playwright is None:
                # Keep the handle: without it `stop()` can never be called and
                # the node driver process outlives the browser.
                self._playwright = await async_playwright().start()

            self._browser = await self._playwright.chromium.launch(
                headless=self.settings.browser_headless,
                slow_mo=self.settings.browser_slow_mo,
            )
            self.log.info("browser_launched", headless=self.settings.browser_headless)
            return self._browser

    async def _create_context(self) -> BrowserContext:
        """Create a new browser context with Korean locale."""
        browser = await self._get_browser()
        context = await browser.new_context(
            locale="ko-KR",
            timezone_id="Asia/Seoul",
            viewport={"width": 1920, "height": 1080},
            user_agent=(
                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
                "AppleWebKit/537.36 (KHTML, like Gecko) "
                "Chrome/120.0.0.0 Safari/537.36"
            ),
        )
        return context

    async def login(
        self,
        business_number: str,
        auth_type: str,
        cert_password: Optional[str] = None,
        user_id: Optional[str] = None,
        password: Optional[str] = None,
        company_id: str = "",
    ) -> HometaxSession:
        """
        Login to Hometax with the specified authentication method.

        A session is created only after Hometax confirms the login. Previously
        both login helpers slept for a second and returned, so any business
        number with any password produced a valid session.

        Args:
            business_number: Business registration number (10 digits)
            auth_type: Authentication type (certificate, simple_auth, id_password)
            cert_password: Certificate password (for certificate auth)
            user_id: User ID (for ID/password auth)
            password: Password (for ID/password auth)
            company_id: Tenant that owns this session

        Returns:
            HometaxSession with session information

        Raises:
            HometaxNotEnabledError: If live mode is off
            HometaxNotImplementedError: For an authentication method with no
                implementation (certificate login needs NPKI/SEED handling)
            HometaxLoginError: If Hometax did not accept the credentials
        """
        self.log.info("login_started", auth_type=auth_type, company_id=company_id)

        self._require_live_mode("Hometax login")

        auth_type_enum = AuthType(auth_type)

        # Reject unimplemented methods before spending a browser context on them.
        if auth_type_enum == AuthType.CERTIFICATE:
            raise HometaxNotImplementedError(
                "Certificate (공동인증서/NPKI) login is not implemented. It requires "
                "certificate store access and SEED handling that this codebase does "
                "not have -- see shared/crypto/seed.py. Use id_password, or the "
                "Popbill provider."
            )
        if auth_type_enum == AuthType.SIMPLE_AUTH:
            raise HometaxNotImplementedError(
                "Simple authentication (간편인증) is not implemented: it requires an "
                "out-of-band approval flow on the user's device."
            )
        if auth_type_enum != AuthType.ID_PASSWORD:
            raise HometaxNotImplementedError(f"Unsupported auth type: {auth_type}")

        if not user_id or not password:
            raise HometaxLoginError("User ID and password are required")

        await self._evict_expired_sessions()
        await self._enforce_session_limit()

        context = await self._create_context()
        page = await context.new_page()

        try:
            login_page = LoginPage(page)
            await login_page.navigate()

            # The page object returns False for a rejected login and raises for
            # anything it could not determine. Neither is treated as success.
            success = await login_page.login_with_credentials(user_id, password)
            if not success:
                raise HometaxLoginError(
                    "Hometax rejected the credentials or the login could not be confirmed"
                )

            company_name = await self._get_company_name(login_page)

            session_id = str(uuid.uuid4())
            expires_at = datetime.now() + timedelta(
                minutes=self.settings.hometax_session_ttl_minutes
            )

            async with self._session_lock:
                self._sessions[(company_id, session_id)] = _ScraperSession(
                    company_id=company_id,
                    context=context,
                    business_number=business_number,
                    expires_at=expires_at,
                )

            session = HometaxSession(
                session_id=session_id,
                business_number=business_number,
                company_name=company_name,
                expires_at=expires_at,
                auth_type=auth_type_enum,
            )

            self.log.info(
                "login_success",
                session_id=session_id[:8] + "...",
                company_id=company_id,
            )

            return session

        except Exception as e:
            # Close the context on every failure path. It was never registered
            # in `_sessions`, so nothing else can be holding it.
            await context.close()
            self.log.error("login_failed", error=type(e).__name__)
            raise

    async def _get_company_name(self, login_page: LoginPage) -> str:
        """
        Read the logged-in company name from the page.

        Returns an empty string when the page does not expose it -- never a
        placeholder. This used to return the literal "Test Company", which then
        travelled into responses and storage as if it were real.

        Args:
            login_page: The authenticated login page object

        Returns:
            Company name, or "" if unavailable
        """
        info = await login_page.get_user_info()
        if not info:
            return ""
        return info.get("display_name") or ""

    async def _resolve_session(
        self,
        company_id: str,
        session_id: str,
    ) -> _ScraperSession:
        """
        Look up a session, enforcing tenant ownership and expiry.

        Args:
            company_id: Tenant making the request
            session_id: Session identifier

        Returns:
            The matching session

        Raises:
            HometaxSessionError: If unknown, expired, or owned by another tenant
        """
        async with self._session_lock:
            session = self._sessions.get((company_id, session_id))

            if session is None:
                # Deliberately the same message whether the session does not
                # exist or belongs to someone else: distinguishing them tells a
                # caller which session ids are real.
                raise HometaxSessionError("Invalid or expired session")

            if session.is_expired():
                self._sessions.pop((company_id, session_id), None)
                await session.context.close()
                raise HometaxSessionError("Invalid or expired session")

            return session

    async def _evict_expired_sessions(self) -> None:
        """Close and drop every session past its expiry."""
        now = datetime.now()
        async with self._session_lock:
            expired = [k for k, s in self._sessions.items() if s.is_expired(now)]
            for key in expired:
                session = self._sessions.pop(key, None)
                if session:
                    try:
                        await session.context.close()
                    except Exception:
                        self.log.warning("expired_session_close_failed")

        if expired:
            self.log.info("expired_sessions_evicted", count=len(expired))

    async def _enforce_session_limit(self) -> None:
        """
        Drop the oldest sessions once the configured ceiling is reached.

        Without a ceiling, a client that never calls logout accumulates browser
        contexts until the process runs out of memory or file descriptors.

        Raises:
            HometaxSessionError: If the limit cannot be honoured
        """
        limit = self.settings.hometax_max_sessions
        async with self._session_lock:
            if len(self._sessions) < limit:
                return

            victims = sorted(self._sessions.items(), key=lambda kv: kv[1].expires_at)
            drop = len(self._sessions) - limit + 1

            for key, session in victims[:drop]:
                self._sessions.pop(key, None)
                try:
                    await session.context.close()
                except Exception:
                    self.log.warning("session_evict_close_failed")

            self.log.warning("session_limit_reached", evicted=drop, limit=limit)

    async def get_tax_invoices(
        self,
        session_id: str,
        start_date: str,
        end_date: str,
        invoice_type: Optional[str] = None,
        company_id: str = "",
    ) -> list[TaxInvoice]:
        """
        Retrieve tax invoices from Hometax.

        Args:
            session_id: Active session ID
            start_date: Start date (YYYY-MM-DD)
            end_date: End date (YYYY-MM-DD)
            invoice_type: Filter by invoice type (sales/purchase)
            company_id: Tenant making the request

        Returns:
            List of tax invoices

        Raises:
            HometaxNotEnabledError: If live mode is off
            HometaxSessionError: If the session is invalid for this tenant
            HometaxScrapeError: If the query could not be completed. A failed
                query must never come back as an empty list: that reads as
                "no 세금계산서 in this period" and feeds 부가가치세 신고 directly.
        """
        self.log.info(
            "get_invoices_started",
            session_id=session_id[:8] + "...",
            company_id=company_id,
            start_date=start_date,
            end_date=end_date,
        )

        self._require_live_mode("Hometax invoice query")

        session = await self._resolve_session(company_id, session_id)
        page = await self._active_page(session.context)

        parsed_type = (
            InvoiceType(invoice_type) if invoice_type else InvoiceType.SALES
        )

        search_page = TaxInvoiceSearchPage(page)
        await search_page.navigate(parsed_type)

        invoices = await search_page.search(
            start_date=date.fromisoformat(start_date),
            end_date=date.fromisoformat(end_date),
            invoice_type=parsed_type,
        )

        self.log.info("get_invoices_success", count=len(invoices))
        return invoices

    async def issue_tax_invoice(
        self,
        session_id: str,
        invoice_data: dict[str, Any],
        company_id: str = "",
    ) -> IssuedInvoiceResult:
        """
        Issue a new tax invoice via Hometax.

        Args:
            session_id: Active session ID
            invoice_data: Invoice details
            company_id: Tenant making the request

        Returns:
            IssuedInvoiceResult carrying the 국세청 승인번호 obtained from the site

        Raises:
            HometaxNotEnabledError: If live mode is off
            HometaxSessionError: If the session is invalid for this tenant
            HometaxResultUndeterminedError: If issuance could not be confirmed
        """
        self.log.info(
            "issue_invoice_started",
            session_id=session_id[:8] + "...",
            company_id=company_id,
            buyer=str(invoice_data.get("buyer_business_number", ""))[:6] + "****",
        )

        # This is the path that used to return a hardcoded success with an
        # invented 국세청 승인번호 while transmitting nothing. Issuance carries a
        # 2% 미발급 가산세 for the supplier and denies the buyer their 매입세액
        # deduction, and a false success hides that until after the deadline.
        self._require_live_mode("Hometax tax invoice issuance")

        session = await self._resolve_session(company_id, session_id)
        page = await self._active_page(session.context)

        invoice = TaxInvoice(**invoice_data) if not isinstance(
            invoice_data, TaxInvoice
        ) else invoice_data

        issue_page = TaxInvoiceIssuePage(page)
        await issue_page.navigate()

        result = await issue_page.issue(invoice)

        # A success without a 국세청 승인번호 is not a success we can stand behind.
        if result.success and not result.nts_confirm_number:
            raise HometaxResultUndeterminedError(
                "Issuance reported success but no 국세청 승인번호 was returned; "
                "treat this filing as unconfirmed and reconcile against Hometax."
            )

        self.log.info(
            "issue_invoice_completed",
            success=result.success,
            has_confirm_number=bool(result.nts_confirm_number),
        )
        return result

    async def _active_page(self, context: BrowserContext) -> Page:
        """Return the context's page, creating one if needed."""
        return context.pages[0] if context.pages else await context.new_page()

    async def logout(self, session_id: str, company_id: str = "") -> None:
        """
        Logout and close a session.

        Args:
            session_id: Session ID to close
            company_id: Tenant that owns the session

        Raises:
            HometaxSessionError: If the session is not this tenant's
        """
        self.log.info("logout_started", session_id=session_id[:8] + "...")

        async with self._session_lock:
            session = self._sessions.pop((company_id, session_id), None)

        if session is None:
            # Do not delete by session_id alone: that let any caller terminate
            # another tenant's session.
            raise HometaxSessionError("Invalid or expired session")

        await session.context.close()
        self.log.info("logout_success")

    async def close(self) -> None:
        """Close all sessions, the browser, and the Playwright driver."""
        self.log.info("closing_scraper")

        async with self._session_lock:
            sessions = list(self._sessions.values())
            self._sessions.clear()

        for session in sessions:
            try:
                await session.context.close()
            except Exception:
                self.log.warning("context_close_failed")

        if self._browser is not None:
            try:
                if self._browser.is_connected():
                    await self._browser.close()
            except Exception:
                self.log.warning("browser_close_failed")
            finally:
                self._browser = None

        # Stopping Playwright terminates the node driver subprocess. Without
        # this the driver survives every browser restart and accumulates.
        if self._playwright is not None:
            try:
                await self._playwright.stop()
            except Exception:
                self.log.warning("playwright_stop_failed")
            finally:
                self._playwright = None

        self.log.info("scraper_closed")
