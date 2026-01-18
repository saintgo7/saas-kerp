"""
Tax Invoice Service Implementation

Provides business logic for tax invoice operations, coordinating between
Hometax scraper and Popbill API provider.
"""

import uuid
from datetime import datetime, timedelta
from typing import Any, Optional

import structlog

from config import get_settings
from providers.popbill import PopbillClient, PopbillConfig, PopbillTaxInvoice
from src.hometax.models import (
    AuthType,
    HometaxSession,
    InvoiceType,
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
        self._sessions: dict[str, HometaxSession] = {}

    async def _get_scraper(self) -> HometaxScraper:
        """Get or create Hometax scraper instance."""
        if self._scraper is None:
            self._scraper = HometaxScraper()
        return self._scraper

    async def _get_popbill(self) -> PopbillClient:
        """Get or create Popbill client instance."""
        if self._popbill is None:
            config = PopbillConfig(
                link_id=self.settings.popbill_link_id,
                secret_key=self.settings.popbill_secret_key,
                is_test=self.settings.popbill_is_test,
            )
            self._popbill = PopbillClient(config)
        return self._popbill

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
            scraper = await self._get_scraper()
            session = await scraper.login(
                business_number=business_number,
                auth_type=auth_type,
                cert_password=cert_password,
                user_id=user_id,
                password=password,
            )

            # Store session with company context
            session_key = f"{company_id}:{session.session_id}"
            self._sessions[session_key] = session

            self.log.info(
                "login_success",
                session_id=session.session_id[:8] + "...",
                company_name=session.company_name,
            )

            return {
                "success": True,
                "session_id": session.session_id,
                "expires_at": session.expires_at.isoformat(),
                "company_name": session.company_name,
            }

        except Exception as e:
            self.log.error("login_failed", error=str(e))
            return {
                "success": False,
                "error_message": str(e),
                "error_code": "LOGIN_FAILED",
            }

    async def logout(self, session_id: str) -> dict[str, Any]:
        """
        Logout from Hometax.

        Args:
            session_id: Session ID to invalidate

        Returns:
            Logout result
        """
        self.log.info("logout_request", session_id=session_id[:8] + "...")

        try:
            scraper = await self._get_scraper()
            await scraper.logout(session_id)

            # Remove session from cache
            for key in list(self._sessions.keys()):
                if key.endswith(f":{session_id}"):
                    del self._sessions[key]
                    break

            self.log.info("logout_success")
            return {"success": True}

        except Exception as e:
            self.log.error("logout_failed", error=str(e))
            return {"success": False, "error_message": str(e)}

    async def get_tax_invoices(
        self,
        session_id: str,
        start_date: str,
        end_date: str,
        invoice_type: Optional[str] = None,
        business_number: Optional[str] = None,
        page: int = 1,
        page_size: int = 50,
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
            scraper = await self._get_scraper()
            invoices = await scraper.get_tax_invoices(
                session_id=session_id,
                start_date=start_date,
                end_date=end_date,
                invoice_type=invoice_type,
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
            self.log.error("get_invoices_failed", error=str(e))
            return {
                "success": False,
                "error_message": str(e),
            }

    async def issue_tax_invoice(
        self,
        session_id: str,
        invoice_data: dict[str, Any],
        provider: str = "hometax",
        transmit_immediately: bool = False,
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
                result = await self._issue_via_popbill(invoice_data, transmit_immediately)
            else:
                result = await self._issue_via_hometax(session_id, invoice_data)

            if result.get("success"):
                self.log.info(
                    "issue_invoice_success",
                    invoice_number=result.get("invoice_number"),
                )

            return result

        except Exception as e:
            self.log.error("issue_invoice_failed", error=str(e))
            return {
                "success": False,
                "error_message": str(e),
                "error_code": "ISSUE_FAILED",
            }

    async def _issue_via_hometax(
        self,
        session_id: str,
        invoice_data: dict[str, Any],
    ) -> dict[str, Any]:
        """Issue invoice via Hometax scraper."""
        scraper = await self._get_scraper()
        result = await scraper.issue_tax_invoice(
            session_id=session_id,
            invoice_data=invoice_data,
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
    ) -> dict[str, Any]:
        """Issue invoice via Popbill API."""
        popbill = await self._get_popbill()

        # Convert to Popbill format
        popbill_invoice = PopbillTaxInvoice(
            invoice_number=invoice_data.get("invoice_number", str(uuid.uuid4())[:8]),
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

            # Get company info from session
            session = None
            for key, sess in self._sessions.items():
                if key.endswith(f":{session_id}"):
                    session = sess
                    break

            if not session:
                return {
                    "success": False,
                    "error_message": "Invalid session",
                }

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
            self.log.error("cancel_invoice_failed", error=str(e))
            return {
                "success": False,
                "error_message": str(e),
            }

    async def get_invoice_status(
        self,
        session_id: str,
        invoice_number: str,
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

        try:
            popbill = await self._get_popbill()

            # Get session info
            session = None
            for key, sess in self._sessions.items():
                if key.endswith(f":{session_id}"):
                    session = sess
                    break

            if not session:
                return {
                    "success": False,
                    "error_message": "Invalid session",
                }

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
            self.log.error("get_status_failed", error=str(e))
            return {
                "success": False,
                "error_message": str(e),
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
            # Get invoices from Hometax
            scraper = await self._get_scraper()
            invoices = await scraper.get_tax_invoices(
                session_id=session_id,
                start_date=start_date,
                end_date=end_date,
                invoice_type=invoice_type,
            )

            # TODO: Save to database
            # For now, just return the count

            self.log.info(
                "sync_success",
                synced_count=len(invoices),
            )

            return {
                "success": True,
                "synced_count": len(invoices),
                "new_count": len(invoices),  # TODO: Calculate actual new
                "updated_count": 0,  # TODO: Calculate actual updates
                "errors": [],
            }

        except Exception as e:
            self.log.error("sync_failed", error=str(e))
            return {
                "success": False,
                "error_message": str(e),
                "synced_count": 0,
                "errors": [str(e)],
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

    async def close(self) -> None:
        """Close all resources."""
        if self._scraper:
            await self._scraper.close()
        if self._popbill:
            await self._popbill.close()
        self._sessions.clear()
        self.log.info("service_closed")
