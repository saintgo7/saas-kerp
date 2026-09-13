"""
Tax Invoice Service Implementation

Provides business logic for tax invoice operations, coordinating between
Hometax scraper and Popbill API provider.
"""

import asyncio
import uuid
from datetime import datetime
from typing import Any, Optional

import structlog

from config import get_settings
from providers.popbill import PopbillClient, PopbillConfig, PopbillTaxInvoice
from src.hometax.errors import (
    HometaxError,
    HometaxSessionError,
)
from src.hometax.models import (
    HometaxSession,
    TaxInvoice,
)
from src.hometax.scraper import HometaxScraper
from src.utils.validators import (
    validate_business_number,
    validate_date_range,
    validate_invoice_number,
)

logger = structlog.get_logger()


class TaxInvoiceService:
    """
    Service for tax invoice operations.

    Coordinates between:
    - HometaxScraper for direct Hometax operations
    - PopbillClient for ASP API operations
    """

    def __init__(self) -> None:
        """Initialize the service."""
        self.settings = get_settings()
        self.log = logger.bind(component="TaxInvoiceService")
        self._scraper: Optional[HometaxScraper] = None
        self._popbill: Optional[PopbillClient] = None
        # Keyed by (company_id, session_id). A session_id alone must never be
        # enough to reach a session: that let one tenant cancel another tenant's
        # 세금계산서 using nothing but a leaked id.
        self._sessions: dict[tuple[str, str], HometaxSession] = {}
        self._scraper_lock = asyncio.Lock()
        self._popbill_lock = asyncio.Lock()
        self._session_lock = asyncio.Lock()

        if self.settings.popbill_is_test:
            self.log.warning(
                "popbill_test_endpoint",
                message=(
                    "POPBILL_IS_TEST is on: 세금계산서 go to the Popbill test "
                    "endpoint and are NOT transmitted to the 국세청."
                ),
            )

    @staticmethod
    def _require_company_id(company_id: str) -> str:
        """
        Ensure a tenant identifier was supplied.

        Args:
            company_id: Tenant identifier from the request

        Returns:
            The identifier

        Raises:
            HometaxSessionError: If it is missing
        """
        if not company_id:
            raise HometaxSessionError(
                "company_id is required: session lookups are scoped per tenant"
            )
        return company_id

    @staticmethod
    def _client_safe_error(exc: Exception) -> tuple[str, str]:
        """
        Map an internal exception to a caller-safe (error_code, message).

        Raw `str(e)` used to travel to gRPC clients carrying file paths,
        internal hostnames and upstream Popbill response bodies.

        Args:
            exc: The exception that occurred

        Returns:
            Tuple of (error_code, error_message)
        """
        if isinstance(exc, HometaxError):
            # These are our own, deliberately non-revealing messages.
            return exc.error_code, str(exc)
        if isinstance(exc, (ConnectionError, TimeoutError)):
            return "UPSTREAM_UNAVAILABLE", "외부 시스템과 통신하지 못했습니다"
        return "INTERNAL_ERROR", "처리 중 오류가 발생했습니다"

    async def _get_scraper(self) -> HometaxScraper:
        """Get or create the Hometax scraper instance."""
        # Lock the lazy init: two concurrent requests could otherwise each build
        # a scraper, each launching a browser, and leak one of them.
        async with self._scraper_lock:
            if self._scraper is None:
                self._scraper = HometaxScraper()
            return self._scraper

    async def _get_popbill(self) -> PopbillClient:
        """Get or create the Popbill client instance."""
        async with self._popbill_lock:
            if self._popbill is None:
                config = PopbillConfig(
                    link_id=self.settings.popbill_link_id,
                    secret_key=self.settings.popbill_secret_key,
                    is_test=self.settings.popbill_is_test,
                )
                self._popbill = PopbillClient(config)
            return self._popbill

    async def _resolve_session(
        self,
        company_id: str,
        session_id: str,
    ) -> HometaxSession:
        """
        Look up a session by exact (company_id, session_id) and check expiry.

        Args:
            company_id: Tenant making the request
            session_id: Session identifier

        Returns:
            The session

        Raises:
            HometaxSessionError: If unknown, expired, or owned by another tenant
        """
        self._require_company_id(company_id)

        async with self._session_lock:
            session = self._sessions.get((company_id, session_id))

            if session is None:
                # Same message for "no such session" and "not yours", so the
                # response cannot be used to probe which ids exist.
                raise HometaxSessionError("Invalid or expired session")

            # `expires_at` existed on the model but was never checked, so a
            # session stayed valid forever.
            if datetime.now() >= session.expires_at:
                self._sessions.pop((company_id, session_id), None)
                raise HometaxSessionError("Invalid or expired session")

            return session

    async def login(
        self,
        company_id: str,
        business_number: str,
        auth_type: str,
        cert_password: Optional[str] = None,
        cert_data: Optional[bytes] = None,
        user_id: Optional[str] = None,
        password: Optional[str] = None,
    ) -> dict[str, Any]:
        """
        Login to Hometax.

        Args:
            company_id: Company ID for multi-tenant isolation
            business_number: Business registration number
            auth_type: Authentication type
            cert_password: Certificate password
            cert_data: Certificate data
            user_id: User ID for ID/password auth
            password: Password for ID/password auth

        Returns:
            Login result with session info
        """
        self.log.info(
            "login_request",
            company_id=company_id,
            business_number=business_number[:6] + "****",
            auth_type=auth_type,
        )

        # Validate business number
        is_valid, error_msg = validate_business_number(business_number)
        if not is_valid:
            return {
                "success": False,
                "error_message": error_msg,
                "error_code": "INVALID_BUSINESS_NUMBER",
            }

        try:
            self._require_company_id(company_id)

            scraper = await self._get_scraper()
            session = await scraper.login(
                business_number=business_number,
                auth_type=auth_type,
                cert_password=cert_password,
                user_id=user_id,
                password=password,
                company_id=company_id,
            )

            # Store the session under the tenant that owns it.
            async with self._session_lock:
                self._sessions[(company_id, session.session_id)] = session

            self.log.info(
                "login_success",
                session_id=session.session_id[:8] + "...",
                company_id=company_id,
            )

            return {
                "success": True,
                "session_id": session.session_id,
                "expires_at": session.expires_at.isoformat(),
                "company_name": session.company_name,
            }

        except Exception as e:
            # Log the detail; return only a stable code. A login error carrying
            # upstream text can distinguish "no such user" from "wrong password".
            self.log.error("login_failed", error_type=type(e).__name__, detail=str(e))
            code, message = self._client_safe_error(e)
            return {
                "success": False,
                "error_message": message,
                "error_code": code if code != "INTERNAL_ERROR" else "LOGIN_FAILED",
            }

    async def logout(self, session_id: str, company_id: str = "") -> dict[str, Any]:
        """
        Logout from Hometax.

        Args:
            session_id: Session ID to invalidate
            company_id: Tenant that owns the session

        Returns:
            Logout result
        """
        self.log.info(
            "logout_request",
            session_id=session_id[:8] + "...",
            company_id=company_id,
        )

        try:
            # Confirms the session belongs to this tenant before touching it.
            # The old loop deleted by session_id suffix alone, so any caller
            # could terminate another tenant's session.
            await self._resolve_session(company_id, session_id)

            scraper = await self._get_scraper()
            await scraper.logout(session_id, company_id=company_id)

            async with self._session_lock:
                self._sessions.pop((company_id, session_id), None)

            self.log.info("logout_success")
            return {"success": True}

        except Exception as e:
            self.log.error("logout_failed", error_type=type(e).__name__, detail=str(e))
            code, message = self._client_safe_error(e)
            return {"success": False, "error_message": message, "error_code": code}

    async def get_tax_invoices(
        self,
        session_id: str,
        start_date: str,
        end_date: str,
        invoice_type: Optional[str] = None,
        business_number: Optional[str] = None,
        page: int = 1,
        page_size: int = 50,
        company_id: str = "",
    ) -> dict[str, Any]:
        """
        Get tax invoices from Hometax.

        Args:
            session_id: Active session ID
            start_date: Start date (YYYY-MM-DD)
            end_date: End date (YYYY-MM-DD)
            invoice_type: Filter by invoice type
            business_number: Filter by counterparty
            page: Page number
            page_size: Items per page

        Returns:
            Search results with invoices
        """
        self.log.info(
            "get_invoices_request",
            session_id=session_id[:8] + "...",
            date_range=f"{start_date} to {end_date}",
        )

        # Validate date range
        is_valid, error_msg, parsed_start, parsed_end = validate_date_range(
            start_date, end_date
        )
        if not is_valid:
            return {
                "success": False,
                "error_message": error_msg,
            }

        try:
            await self._resolve_session(company_id, session_id)

            scraper = await self._get_scraper()
            invoices = await scraper.get_tax_invoices(
                session_id=session_id,
                start_date=start_date,
                end_date=end_date,
                invoice_type=invoice_type,
                company_id=company_id,
            )

            # Apply pagination
            total_count = len(invoices)
            start_idx = (page - 1) * page_size
            end_idx = start_idx + page_size
            paginated = invoices[start_idx:end_idx]

            self.log.info(
                "get_invoices_success",
                total_count=total_count,
                returned_count=len(paginated),
            )

            return {
                "success": True,
                "invoices": [self._invoice_to_dict(inv) for inv in paginated],
                "total_count": total_count,
                "page": page,
                "page_size": page_size,
            }

        except Exception as e:
            # A failed query must never surface as `success: True, total_count: 0`.
            # That reads downstream as "no 세금계산서 in this period" and feeds
            # 부가가치세 신고 as missing 매출/매입.
            self.log.error(
                "get_invoices_failed", error_type=type(e).__name__, detail=str(e)
            )
            code, message = self._client_safe_error(e)
            return {
                "success": False,
                "error_code": code,
                "error_message": message,
            }

    async def issue_tax_invoice(
        self,
        session_id: str,
        invoice_data: dict[str, Any],
        provider: str = "hometax",
        transmit_immediately: bool = False,
        company_id: str = "",
        idempotency_key: str = "",
    ) -> dict[str, Any]:
        """
        Issue a new tax invoice.

        Args:
            session_id: Active session ID
            invoice_data: Invoice details
            provider: Provider to use (hometax or popbill)
            transmit_immediately: Send to NTS immediately

        Returns:
            Issue result with invoice number
        """
        self.log.info(
            "issue_invoice_request",
            session_id=session_id[:8] + "...",
            provider=provider,
            buyer=invoice_data.get("buyer_business_number", "")[:6] + "****",
        )

        try:
            if provider == "popbill":
                result = await self._issue_via_popbill(
                    invoice_data,
                    transmit_immediately,
                    idempotency_key=idempotency_key,
                )
            else:
                result = await self._issue_via_hometax(
                    session_id, invoice_data, company_id=company_id
                )

            if result.get("success"):
                self.log.info(
                    "issue_invoice_success",
                    invoice_number=result.get("invoice_number"),
                )

            return result

        except Exception as e:
            self.log.error(
                "issue_invoice_failed", error_type=type(e).__name__, detail=str(e)
            )
            code, message = self._client_safe_error(e)
            return {
                "success": False,
                "error_message": message,
                "error_code": code if code != "INTERNAL_ERROR" else "ISSUE_FAILED",
            }

    async def _issue_via_hometax(
        self,
        session_id: str,
        invoice_data: dict[str, Any],
        company_id: str = "",
    ) -> dict[str, Any]:
        """
        Issue an invoice via the Hometax scraper.

        Raises rather than returning a fabricated success when the scraping path
        is not enabled -- see `HometaxScraper.issue_tax_invoice`.
        """
        await self._resolve_session(company_id, session_id)

        scraper = await self._get_scraper()
        result = await scraper.issue_tax_invoice(
            session_id=session_id,
            invoice_data=invoice_data,
            company_id=company_id,
        )

        return {
            "success": result.success,
            "invoice_number": result.invoice_number,
            "issue_date": result.issue_date.isoformat(),
            "nts_confirm_number": result.nts_confirm_number or "",
            "error_message": result.error_message or "",
        }

    async def _issue_via_popbill(
        self,
        invoice_data: dict[str, Any],
        force_send: bool = False,
        idempotency_key: str = "",
    ) -> dict[str, Any]:
        """
        Issue an invoice via the Popbill API.

        `idempotency_key` becomes the 관리번호 when supplied, so a retry of the
        same logical issuance reuses the identifier instead of minting a new
        random one and creating a second 세금계산서.
        """
        popbill = await self._get_popbill()

        # Convert to Popbill format
        popbill_invoice = PopbillTaxInvoice(
            invoice_number=(
                invoice_data.get("invoice_number")
                or idempotency_key
                or str(uuid.uuid4())[:8]
            ),
            write_date=datetime.now().strftime("%Y%m%d"),
            invoicer_corp_num=invoice_data["supplier_business_number"],
            invoicer_corp_name=invoice_data["supplier_name"],
            invoicer_ceo_name=invoice_data.get("supplier_ceo_name", ""),
            invoicer_addr=invoice_data.get("supplier_address", ""),
            invoicer_email=invoice_data.get("supplier_email", ""),
            invoicee_corp_num=invoice_data["buyer_business_number"],
            invoicee_corp_name=invoice_data["buyer_name"],
            invoicee_ceo_name=invoice_data.get("buyer_ceo_name", ""),
            invoicee_addr=invoice_data.get("buyer_address", ""),
            invoicee_email=invoice_data.get("buyer_email", ""),
            supply_cost_total=int(invoice_data["supply_amount"]),
            tax_total=int(invoice_data["tax_amount"]),
            total_amount=int(invoice_data["total_amount"]),
        )

        result = await popbill.issue_tax_invoice(
            corp_num=invoice_data["supplier_business_number"],
            invoice=popbill_invoice,
            force_send=force_send,
        )

        return {
            "success": result.success,
            "invoice_number": result.invoice_number,
            "nts_confirm_number": result.nts_confirm_number,
            "error_code": result.error_code,
            "error_message": result.error_message,
        }

    async def cancel_tax_invoice(
        self,
        session_id: str,
        invoice_number: str,
        cancel_reason: str = "",
        company_id: str = "",
    ) -> dict[str, Any]:
        """
        Cancel an issued tax invoice.

        Args:
            session_id: Active session ID
            invoice_number: Invoice number to cancel
            cancel_reason: Reason for cancellation

        Returns:
            Cancellation result
        """
        self.log.info(
            "cancel_invoice_request",
            session_id=session_id[:8] + "...",
            invoice_number=invoice_number,
        )

        # Validate invoice number
        is_valid, error_msg = validate_invoice_number(invoice_number)
        if not is_valid:
            return {
                "success": False,
                "error_message": error_msg,
            }

        try:
            # For now, use Popbill for cancellation
            popbill = await self._get_popbill()

            # Exact (company_id, session_id) match. The old suffix scan ignored
            # the company entirely, so company B could pass company A's session
            # id and cancel A's 세금계산서 -- the Popbill call below runs under
            # `session.business_number`, which would be A's 사업자번호.
            session = await self._resolve_session(company_id, session_id)

            success = await popbill.cancel_tax_invoice(
                corp_num=session.business_number,
                invoice_number=invoice_number,
                cancel_reason=cancel_reason,
            )

            if success:
                self.log.info("cancel_invoice_success", invoice_number=invoice_number)

            return {
                "success": success,
                "cancelled_at": datetime.now().isoformat() if success else "",
            }

        except Exception as e:
            self.log.error(
                "cancel_invoice_failed", error_type=type(e).__name__, detail=str(e)
            )
            code, message = self._client_safe_error(e)
            return {
                "success": False,
                "error_code": code,
                "error_message": message,
            }

    async def get_invoice_status(
        self,
        session_id: str,
        invoice_number: str,
        company_id: str = "",
    ) -> dict[str, Any]:
        """
        Get status of a specific invoice.

        Args:
            session_id: Active session ID
            invoice_number: Invoice number to query

        Returns:
            Invoice status information
        """
        self.log.info(
            "get_status_request",
            session_id=session_id[:8] + "...",
            invoice_number=invoice_number,
        )

        # This path used to skip validation entirely and pass the raw value
        # into a Popbill URL path segment.
        is_valid, error_msg = validate_invoice_number(invoice_number)
        if not is_valid:
            return {
                "success": False,
                "error_code": "INVALID_INVOICE_NUMBER",
                "error_message": error_msg,
            }

        try:
            popbill = await self._get_popbill()

            session = await self._resolve_session(company_id, session_id)

            invoice_data = await popbill.query_tax_invoice(
                corp_num=session.business_number,
                invoice_number=invoice_number,
            )

            return {
                "success": True,
                "invoice_number": invoice_number,
                "status": invoice_data.get("stateCode", ""),
                "nts_confirm_number": invoice_data.get("ntsconfirmNum", ""),
                "last_updated": invoice_data.get("modifyDT", ""),
            }

        except Exception as e:
            self.log.error(
                "get_status_failed", error_type=type(e).__name__, detail=str(e)
            )
            code, message = self._client_safe_error(e)
            return {
                "success": False,
                "error_code": code,
                "error_message": message,
            }

    async def sync_from_hometax(
        self,
        session_id: str,
        company_id: str,
        start_date: str,
        end_date: str,
        invoice_type: Optional[str] = None,
    ) -> dict[str, Any]:
        """
        Sync tax invoices from Hometax to local database.

        Args:
            session_id: Active session ID
            company_id: Company ID
            start_date: Start date
            end_date: End date
            invoice_type: Filter by invoice type

        Returns:
            Sync result with statistics
        """
        self.log.info(
            "sync_request",
            session_id=session_id[:8] + "...",
            company_id=company_id,
            date_range=f"{start_date} to {end_date}",
        )

        try:
            await self._resolve_session(company_id, session_id)

            # Fetch the invoices. This raises if the query failed, so a failure
            # can no longer be reported as a successful sync of zero records.
            scraper = await self._get_scraper()
            invoices = await scraper.get_tax_invoices(
                session_id=session_id,
                start_date=start_date,
                end_date=end_date,
                invoice_type=invoice_type,
                company_id=company_id,
            )

            # Persistence is not implemented here. Reporting
            # `success: True, synced_count: N` while writing nothing to the
            # database claimed a sync that never happened -- and the caller had
            # no way to tell. Say so instead.
            self.log.warning(
                "sync_persistence_not_implemented",
                retrieved=len(invoices),
                company_id=company_id,
            )

            return {
                "success": False,
                "error_code": "NOT_IMPLEMENTED",
                "error_message": (
                    "Hometax 동기화의 저장 단계가 구현되지 않았습니다. "
                    f"{len(invoices)}건을 조회했으나 저장하지 않았습니다."
                ),
                "synced_count": 0,
                "new_count": 0,
                "updated_count": 0,
                "retrieved_count": len(invoices),
                "errors": ["persistence_not_implemented"],
            }

        except Exception as e:
            self.log.error("sync_failed", error_type=type(e).__name__, detail=str(e))
            code, message = self._client_safe_error(e)
            return {
                "success": False,
                "error_code": code,
                "error_message": message,
                "synced_count": 0,
                "new_count": 0,
                "updated_count": 0,
                "errors": [code],
            }

    def _invoice_to_dict(self, invoice: TaxInvoice) -> dict[str, Any]:
        """Convert TaxInvoice model to dictionary."""
        return {
            "invoice_number": invoice.invoice_number,
            "issue_date": invoice.issue_date.isoformat(),
            "invoice_type": invoice.invoice_type.value,
            "status": invoice.status.value if hasattr(invoice.status, "value") else invoice.status,
            "supplier_business_number": invoice.supplier_business_number,
            "supplier_name": invoice.supplier_name,
            "supplier_ceo_name": invoice.supplier_ceo_name,
            "supplier_address": invoice.supplier_address,
            "buyer_business_number": invoice.buyer_business_number,
            "buyer_name": invoice.buyer_name,
            "buyer_ceo_name": invoice.buyer_ceo_name,
            "buyer_address": invoice.buyer_address,
            "supply_amount": int(invoice.supply_amount),
            "tax_amount": int(invoice.tax_amount),
            "total_amount": int(invoice.total_amount),
            "nts_confirm_number": invoice.nts_confirm_number or "",
            "remarks": invoice.remarks,
        }

    async def dependency_health(self) -> dict[str, bool]:
        """
        Report the real state of each dependency.

        The health RPC used to answer with both dependencies hardcoded to True,
        so a crashed browser or an expired Popbill token still looked healthy
        and no orchestrator ever restarted the pod.

        A dependency that has not been constructed yet is reported healthy: it
        is lazily created, and "not started" is not "broken".

        Returns:
            Mapping of dependency name to health
        """
        services: dict[str, bool] = {}

        scraper = self._scraper
        if scraper is None:
            services["hometax_scraper"] = True
        else:
            browser = scraper._browser
            services["hometax_scraper"] = browser is None or browser.is_connected()

        popbill = self._popbill
        if popbill is None:
            services["popbill_client"] = True
        else:
            services["popbill_client"] = not popbill.is_closed

        return services

    async def close(self) -> None:
        """Close all resources."""
        if self._scraper:
            await self._scraper.close()
        if self._popbill:
            await self._popbill.close()
        async with self._session_lock:
            self._sessions.clear()
        self.log.info("service_closed")
