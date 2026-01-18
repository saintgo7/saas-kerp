"""
Popbill API Client for Tax Invoice Operations

Popbill is a Korean ASP service provider for electronic tax invoice
management, integrating with the National Tax Service (NTS).

API Documentation: https://developers.popbill.com
"""

import base64
import hashlib
import hmac
import json
import time
import uuid
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from decimal import Decimal
from enum import Enum
from typing import Any, Optional
from urllib.parse import urljoin

import httpx
import structlog

logger = structlog.get_logger()


class PopbillError(Exception):
    """Exception for Popbill API errors."""

    def __init__(self, code: str, message: str) -> None:
        """Initialize Popbill error.

        Args:
            code: Popbill error code
            message: Error message
        """
        self.code = code
        self.message = message
        super().__init__(f"[{code}] {message}")


class PopbillInvoiceType(str, Enum):
    """Popbill invoice type codes."""

    SALES = "01"  # Sales tax invoice
    PURCHASE = "02"  # Purchase tax invoice
    COMMISSION = "03"  # Commission tax invoice


class PopbillInvoiceStatus(str, Enum):
    """Popbill invoice status codes."""

    DRAFT = "100"  # Draft (not submitted)
    SUBMITTED = "200"  # Submitted
    SENT_TO_NTS = "300"  # Sent to NTS
    NTS_CONFIRMED = "301"  # NTS confirmed
    NTS_REJECTED = "302"  # NTS rejected
    CANCELLED = "400"  # Cancelled


@dataclass
class PopbillConfig:
    """Configuration for Popbill API client."""

    link_id: str  # Popbill LinkID (provided by Popbill)
    secret_key: str  # Popbill SecretKey (provided by Popbill)
    is_test: bool = True  # Use test environment
    timeout: float = 30.0  # Request timeout in seconds
    retry_count: int = 3  # Number of retries for failed requests
    retry_delay: float = 1.0  # Delay between retries in seconds

    @property
    def base_url(self) -> str:
        """Get API base URL based on environment."""
        if self.is_test:
            return "https://popbill-test.linkhub.co.kr"
        return "https://popbill.linkhub.co.kr"


@dataclass
class PopbillTaxInvoice:
    """Tax invoice data structure for Popbill API."""

    # Invoice identification
    invoice_number: str  # Invoice number (mandatory)
    write_date: str  # Issue date YYYYMMDD (mandatory)

    # Supplier (seller) information
    invoicer_corp_num: str  # Supplier business number (10 digits)
    invoicer_corp_name: str  # Supplier company name
    invoicer_ceo_name: str  # Supplier CEO name
    invoicer_addr: str = ""  # Supplier address
    invoicer_biz_type: str = ""  # Business type
    invoicer_biz_class: str = ""  # Business item
    invoicer_email: str = ""  # Supplier email

    # Buyer (purchaser) information
    invoicee_corp_num: str = ""  # Buyer business number
    invoicee_corp_name: str = ""  # Buyer company name
    invoicee_ceo_name: str = ""  # Buyer CEO name
    invoicee_addr: str = ""  # Buyer address
    invoicee_biz_type: str = ""  # Business type
    invoicee_biz_class: str = ""  # Business item
    invoicee_email: str = ""  # Buyer email

    # Amount information
    supply_cost_total: int = 0  # Total supply amount (before tax)
    tax_total: int = 0  # Total VAT amount
    total_amount: int = 0  # Total amount (supply + tax)

    # Invoice details
    invoice_type: PopbillInvoiceType = PopbillInvoiceType.SALES
    tax_type: str = "01"  # 01: Taxable, 02: Zero-rated, 03: Exempt
    purpose_type: str = "02"  # 01: Request, 02: Claim

    # Items
    detail_list: list[dict[str, Any]] = field(default_factory=list)

    # Additional info
    remark1: str = ""  # Remark 1
    remark2: str = ""  # Remark 2
    remark3: str = ""  # Remark 3
    kwon: int = 1  # Number of booklets
    ho: int = 1  # Number within booklet

    def to_dict(self) -> dict[str, Any]:
        """Convert to Popbill API format dictionary."""
        return {
            "writeDate": self.write_date,
            "invoiceType": self.invoice_type.value,
            "taxType": self.tax_type,
            "purposeType": self.purpose_type,
            "invoicerCorpNum": self.invoicer_corp_num,
            "invoicerCorpName": self.invoicer_corp_name,
            "invoicerCEOName": self.invoicer_ceo_name,
            "invoicerAddr": self.invoicer_addr,
            "invoicerBizType": self.invoicer_biz_type,
            "invoicerBizClass": self.invoicer_biz_class,
            "invoicerEmail": self.invoicer_email,
            "invoiceeCorpNum": self.invoicee_corp_num,
            "invoiceeCorpName": self.invoicee_corp_name,
            "invoiceeCEOName": self.invoicee_ceo_name,
            "invoiceeAddr": self.invoicee_addr,
            "invoiceeBizType": self.invoicee_biz_type,
            "invoiceeBizClass": self.invoicee_biz_class,
            "invoiceeEmail": self.invoicee_email,
            "supplyCostTotal": str(self.supply_cost_total),
            "taxTotal": str(self.tax_total),
            "totalAmount": str(self.total_amount),
            "remark1": self.remark1,
            "remark2": self.remark2,
            "remark3": self.remark3,
            "kwon": self.kwon,
            "ho": self.ho,
            "detailList": self.detail_list,
        }


@dataclass
class PopbillIssueResult:
    """Result of tax invoice issuance."""

    success: bool
    ntsa_key: str = ""  # Popbill internal key
    invoice_number: str = ""
    nts_confirm_number: str = ""
    error_code: str = ""
    error_message: str = ""


class PopbillClient:
    """
    Popbill API client for tax invoice operations.

    Provides methods for:
    - Issuing tax invoices
    - Querying tax invoice status
    - Cancelling tax invoices
    - Checking API balance
    """

    def __init__(self, config: PopbillConfig) -> None:
        """Initialize Popbill client.

        Args:
            config: Popbill configuration
        """
        self.config = config
        self.log = logger.bind(component="PopbillClient")
        self._access_token: Optional[str] = None
        self._token_expires_at: Optional[datetime] = None
        self._client = httpx.AsyncClient(
            timeout=config.timeout,
            headers={
                "Content-Type": "application/json",
                "Accept": "application/json",
            },
        )

    async def close(self) -> None:
        """Close the HTTP client."""
        await self._client.aclose()

    async def _get_access_token(self, corp_num: str) -> str:
        """Get or refresh access token for API authentication.

        Args:
            corp_num: Business registration number

        Returns:
            Access token string
        """
        # Check if existing token is still valid
        if (
            self._access_token
            and self._token_expires_at
            and datetime.now() < self._token_expires_at - timedelta(minutes=5)
        ):
            return self._access_token

        # Request new token
        self.log.info("requesting_access_token", corp_num=corp_num[:6] + "****")

        timestamp = str(int(time.time()))
        service_id = "POPBILL"

        # Create signature
        sig_target = f"{self.config.link_id}\n{timestamp}\n"
        signature = base64.b64encode(
            hmac.new(
                self.config.secret_key.encode("utf-8"),
                sig_target.encode("utf-8"),
                hashlib.sha256,
            ).digest()
        ).decode("utf-8")

        headers = {
            "x-lh-linkID": self.config.link_id,
            "x-lh-timestamp": timestamp,
            "x-lh-signature": signature,
        }

        url = f"{self.config.base_url}/POPBILL/Token"
        payload = {
            "access_id": self.config.link_id,
            "scope": ["111"],  # Tax invoice scope
        }

        response = await self._client.post(url, headers=headers, json=payload)

        if response.status_code != 200:
            raise PopbillError("TOKEN_ERROR", f"Failed to get access token: {response.text}")

        data = response.json()
        self._access_token = data.get("access_token")
        expires_in = data.get("expires_in", 3600)
        self._token_expires_at = datetime.now() + timedelta(seconds=expires_in)

        self.log.info("access_token_obtained", expires_in=expires_in)
        return self._access_token

    async def _request(
        self,
        method: str,
        endpoint: str,
        corp_num: str,
        data: Optional[dict[str, Any]] = None,
        user_id: Optional[str] = None,
    ) -> dict[str, Any]:
        """Make authenticated API request.

        Args:
            method: HTTP method (GET, POST, etc.)
            endpoint: API endpoint path
            corp_num: Business registration number
            data: Request payload
            user_id: Optional user ID for audit

        Returns:
            Response data dictionary
        """
        token = await self._get_access_token(corp_num)

        headers = {
            "Authorization": f"Bearer {token}",
            "x-pb-userid": user_id or "",
        }

        url = urljoin(self.config.base_url, endpoint)

        for attempt in range(self.config.retry_count):
            try:
                if method.upper() == "GET":
                    response = await self._client.get(url, headers=headers, params=data)
                elif method.upper() == "POST":
                    response = await self._client.post(url, headers=headers, json=data)
                elif method.upper() == "DELETE":
                    response = await self._client.delete(url, headers=headers)
                else:
                    raise ValueError(f"Unsupported HTTP method: {method}")

                if response.status_code == 200:
                    return response.json()

                # Handle API errors
                error_data = response.json()
                error_code = str(error_data.get("code", "UNKNOWN"))
                error_msg = error_data.get("message", "Unknown error")

                self.log.warning(
                    "api_error",
                    endpoint=endpoint,
                    status_code=response.status_code,
                    error_code=error_code,
                    error_message=error_msg,
                )

                # Don't retry for client errors (4xx)
                if 400 <= response.status_code < 500:
                    raise PopbillError(error_code, error_msg)

            except httpx.RequestError as e:
                self.log.warning(
                    "request_error",
                    endpoint=endpoint,
                    attempt=attempt + 1,
                    error=str(e),
                )
                if attempt == self.config.retry_count - 1:
                    raise PopbillError("REQUEST_ERROR", f"Request failed: {str(e)}")

                await self._sleep_for_retry()

        raise PopbillError("MAX_RETRIES", "Maximum retry attempts exceeded")

    async def _sleep_for_retry(self) -> None:
        """Sleep before retry."""
        import asyncio

        await asyncio.sleep(self.config.retry_delay)

    async def issue_tax_invoice(
        self,
        corp_num: str,
        invoice: PopbillTaxInvoice,
        memo: str = "",
        force_send: bool = False,
        user_id: Optional[str] = None,
    ) -> PopbillIssueResult:
        """
        Issue a tax invoice via Popbill API.

        Args:
            corp_num: Business registration number (10 digits)
            invoice: Tax invoice data
            memo: Optional memo
            force_send: Send to NTS immediately
            user_id: Optional user ID for audit

        Returns:
            PopbillIssueResult with issuance details
        """
        self.log.info(
            "issuing_tax_invoice",
            corp_num=corp_num[:6] + "****",
            buyer=invoice.invoicee_corp_num[:6] + "****" if invoice.invoicee_corp_num else "N/A",
            amount=invoice.total_amount,
        )

        endpoint = f"/Taxinvoice/{corp_num}"
        payload = {
            **invoice.to_dict(),
            "memo": memo,
            "forceIssue": force_send,
        }

        try:
            response = await self._request("POST", endpoint, corp_num, payload, user_id)

            result = PopbillIssueResult(
                success=True,
                ntsa_key=response.get("ntsaKey", ""),
                invoice_number=response.get("invoiceNumber", invoice.invoice_number),
                nts_confirm_number=response.get("ntsconfirmNum", ""),
            )

            self.log.info(
                "tax_invoice_issued",
                ntsa_key=result.ntsa_key[:8] + "..." if result.ntsa_key else "",
                invoice_number=result.invoice_number,
            )

            return result

        except PopbillError as e:
            self.log.error(
                "issue_failed",
                error_code=e.code,
                error_message=e.message,
            )
            return PopbillIssueResult(
                success=False,
                error_code=e.code,
                error_message=e.message,
            )

    async def query_tax_invoice(
        self,
        corp_num: str,
        invoice_number: str,
        user_id: Optional[str] = None,
    ) -> dict[str, Any]:
        """
        Query tax invoice details from Popbill.

        Args:
            corp_num: Business registration number
            invoice_number: Invoice number to query
            user_id: Optional user ID for audit

        Returns:
            Tax invoice details dictionary
        """
        self.log.info(
            "querying_tax_invoice",
            corp_num=corp_num[:6] + "****",
            invoice_number=invoice_number,
        )

        endpoint = f"/Taxinvoice/{corp_num}/{invoice_number}"

        try:
            response = await self._request("GET", endpoint, corp_num, user_id=user_id)
            self.log.info("query_success", invoice_number=invoice_number)
            return response

        except PopbillError:
            raise

    async def cancel_tax_invoice(
        self,
        corp_num: str,
        invoice_number: str,
        cancel_reason: str = "",
        user_id: Optional[str] = None,
    ) -> bool:
        """
        Cancel an issued tax invoice.

        Args:
            corp_num: Business registration number
            invoice_number: Invoice number to cancel
            cancel_reason: Reason for cancellation
            user_id: Optional user ID for audit

        Returns:
            True if cancellation was successful
        """
        self.log.info(
            "cancelling_tax_invoice",
            corp_num=corp_num[:6] + "****",
            invoice_number=invoice_number,
        )

        endpoint = f"/Taxinvoice/{corp_num}/{invoice_number}/Cancel"
        payload = {"memo": cancel_reason}

        try:
            await self._request("POST", endpoint, corp_num, payload, user_id)
            self.log.info("cancel_success", invoice_number=invoice_number)
            return True

        except PopbillError as e:
            self.log.error(
                "cancel_failed",
                invoice_number=invoice_number,
                error_code=e.code,
                error_message=e.message,
            )
            raise

    async def get_balance(self, corp_num: str) -> int:
        """
        Get remaining API points balance.

        Args:
            corp_num: Business registration number

        Returns:
            Remaining API points
        """
        self.log.info("checking_balance", corp_num=corp_num[:6] + "****")

        endpoint = f"/Taxinvoice/{corp_num}/Balance"

        try:
            response = await self._request("GET", endpoint, corp_num)
            balance = int(response.get("balance", 0))
            self.log.info("balance_checked", balance=balance)
            return balance

        except PopbillError:
            raise

    async def search_tax_invoices(
        self,
        corp_num: str,
        invoice_type: str,
        start_date: str,
        end_date: str,
        state: Optional[list[str]] = None,
        page: int = 1,
        per_page: int = 50,
        user_id: Optional[str] = None,
    ) -> dict[str, Any]:
        """
        Search tax invoices with filters.

        Args:
            corp_num: Business registration number
            invoice_type: Invoice type (SELL, BUY, TRUSTEE)
            start_date: Start date (YYYYMMDD)
            end_date: End date (YYYYMMDD)
            state: List of status codes to filter
            page: Page number (1-based)
            per_page: Items per page
            user_id: Optional user ID

        Returns:
            Search results dictionary
        """
        self.log.info(
            "searching_tax_invoices",
            corp_num=corp_num[:6] + "****",
            invoice_type=invoice_type,
            date_range=f"{start_date}-{end_date}",
        )

        endpoint = f"/Taxinvoice/{corp_num}/Search"
        params = {
            "DType": "W",  # Search by write date
            "SDate": start_date,
            "EDate": end_date,
            "TaxType": invoice_type,
            "Page": page,
            "PerPage": per_page,
        }

        if state:
            params["State"] = ",".join(state)

        try:
            response = await self._request("GET", endpoint, corp_num, params, user_id)
            total = response.get("total", 0)
            self.log.info("search_complete", total_count=total)
            return response

        except PopbillError:
            raise

    async def register_webhook(
        self,
        corp_num: str,
        callback_url: str,
        user_id: Optional[str] = None,
    ) -> str:
        """
        Register webhook URL for invoice notifications.

        Args:
            corp_num: Business registration number
            callback_url: URL to receive webhook calls
            user_id: Optional user ID

        Returns:
            Webhook registration ID
        """
        self.log.info(
            "registering_webhook",
            corp_num=corp_num[:6] + "****",
            callback_url=callback_url,
        )

        endpoint = f"/Taxinvoice/{corp_num}/Webhook"
        payload = {
            "callbackURL": callback_url,
        }

        try:
            response = await self._request("POST", endpoint, corp_num, payload, user_id)
            webhook_id = response.get("webhookID", "")
            self.log.info("webhook_registered", webhook_id=webhook_id)
            return webhook_id

        except PopbillError:
            raise

    async def send_to_nts(
        self,
        corp_num: str,
        invoice_number: str,
        user_id: Optional[str] = None,
    ) -> bool:
        """
        Send issued invoice to National Tax Service.

        Args:
            corp_num: Business registration number
            invoice_number: Invoice number to send
            user_id: Optional user ID

        Returns:
            True if successful
        """
        self.log.info(
            "sending_to_nts",
            corp_num=corp_num[:6] + "****",
            invoice_number=invoice_number,
        )

        endpoint = f"/Taxinvoice/{corp_num}/{invoice_number}/Request"

        try:
            await self._request("POST", endpoint, corp_num, user_id=user_id)
            self.log.info("sent_to_nts", invoice_number=invoice_number)
            return True

        except PopbillError:
            raise


def create_popbill_client(
    link_id: str,
    secret_key: str,
    is_test: bool = True,
) -> PopbillClient:
    """
    Factory function to create PopbillClient.

    Args:
        link_id: Popbill LinkID
        secret_key: Popbill SecretKey
        is_test: Use test environment

    Returns:
        Configured PopbillClient instance
    """
    config = PopbillConfig(
        link_id=link_id,
        secret_key=secret_key,
        is_test=is_test,
    )
    return PopbillClient(config)
