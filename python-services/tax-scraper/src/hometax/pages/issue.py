"""
Tax invoice issuance page object.
"""
import asyncio
from datetime import datetime

import structlog
from playwright.async_api import Page, TimeoutError as PlaywrightTimeout

from ..constants import (
    ERROR_MESSAGES,
    MENU_TAX_INVOICE_SALES,
    SELECTORS,
    TIMEOUTS,
)
from ..models import IssuedInvoiceResult, TaxInvoice

logger = structlog.get_logger()


class TaxInvoiceIssuePage:
    """
    Page object for tax invoice issuance operations.
    """

    def __init__(self, page: Page) -> None:
        """Initialize issue page with Playwright page instance."""
        self.page = page
        self.log = logger.bind(component="TaxInvoiceIssuePage")

    async def navigate(self) -> None:
        """Navigate to tax invoice issuance page."""
        self.log.info("navigating_to_issue_page")

        await self.page.evaluate(f"wqRunMenu('{MENU_TAX_INVOICE_SALES}')")
        await self._wait_for_loading()

        self.log.info("issue_page_loaded")

    async def issue(self, invoice: TaxInvoice) -> IssuedInvoiceResult:
        """
        Issue a new tax invoice.

        Args:
            invoice: Tax invoice data to issue

        Returns:
            IssuedInvoiceResult with confirmation or error
        """
        self.log.info(
            "issue_started",
            buyer=invoice.buyer_business_number[:6] + "****",
            amount=invoice.total_amount,
        )

        try:
            # Fill in invoice form
            await self._fill_invoice_form(invoice)

            # Submit
            await self.page.click(SELECTORS["form_submit"])
            await self._wait_for_loading()

            # Check for success/error
            result = await self._check_issue_result()

            if result.success:
                self.log.info(
                    "issue_success",
                    invoice_number=result.invoice_number,
                    nts_confirm=result.nts_confirm_number,
                )
            else:
                self.log.warning("issue_failed", error=result.error_message)

            return result

        except PlaywrightTimeout as e:
            self.log.error("issue_timeout", error=str(e))
            return IssuedInvoiceResult(
                success=False,
                invoice_number="",
                issue_date=datetime.now(),
                error_message=f"Timeout: {str(e)}",
            )
        except Exception as e:
            self.log.error("issue_error", error=str(e))
            return IssuedInvoiceResult(
                success=False,
                invoice_number="",
                issue_date=datetime.now(),
                error_message=str(e),
            )

    async def _fill_invoice_form(self, invoice: TaxInvoice) -> None:
        """Fill in the tax invoice issuance form."""
        # Issue date
        await self.page.fill(
            SELECTORS["form_issue_date"],
            invoice.issue_date.strftime("%Y%m%d"),
        )

        # Supplier info (should be pre-filled, but verify)
        supplier_brn = await self.page.input_value(SELECTORS["form_supplier_brn"])
        if not supplier_brn:
            await self.page.fill(
                SELECTORS["form_supplier_brn"],
                invoice.supplier_business_number,
            )
            await self.page.fill(SELECTORS["form_supplier_name"], invoice.supplier_name)

        # Buyer info
        await self.page.fill(SELECTORS["form_buyer_brn"], invoice.buyer_business_number)
        await self._wait_for_loading()  # BRN lookup may trigger

        await self.page.fill(SELECTORS["form_buyer_name"], invoice.buyer_name)

        # Amounts
        await self.page.fill(
            SELECTORS["form_supply_amount"],
            str(invoice.supply_amount),
        )
        await self.page.fill(
            SELECTORS["form_tax_amount"],
            str(invoice.tax_amount),
        )

        # Fill items if present
        if invoice.items:
            await self._fill_items(invoice.items)

        # Remarks
        if invoice.remarks:
            remark_selector = "#remark1, #remark"
            await self.page.fill(remark_selector, invoice.remarks)

    async def _fill_items(self, items: list) -> None:
        """Fill in invoice line items."""
        for i, item in enumerate(items):
            # Add row if needed
            if i > 0:
                add_btn = await self.page.query_selector("#btn_add_row, .add_row")
                if add_btn:
                    await add_btn.click()
                    await asyncio.sleep(0.2)

            # Fill item fields (row index based selectors)
            row_prefix = f"#item_{i}_" if i > 0 else "#item_0_"

            if item.supply_date:
                await self.page.fill(
                    f"{row_prefix}purchaseDT",
                    item.supply_date.strftime("%Y%m%d"),
                )

            await self.page.fill(f"{row_prefix}itemName", item.description)

            if item.specification:
                await self.page.fill(f"{row_prefix}spec", item.specification)

            await self.page.fill(f"{row_prefix}qty", str(item.quantity))
            await self.page.fill(f"{row_prefix}unitCost", str(item.unit_price))
            await self.page.fill(f"{row_prefix}supplyCost", str(item.amount))
            await self.page.fill(f"{row_prefix}tax", str(item.tax_amount))

    async def _check_issue_result(self) -> IssuedInvoiceResult:
        """
        Determine the outcome of an issuance attempt.

        Classification is failure-first and evidence-based:

        1. If the alert text matches a known failure message, it is a failure.
           The previous logic asked whether the alert contained "발급" and called
           that success -- but `ERROR_MESSAGES["ISSUE_FAILED"]` is
           "발급에 실패했습니다", which contains "발급". Explicit rejections from
           the 국세청 were therefore recorded as successful issuances, and the
           alert was dismissed, erasing the evidence.
        2. Success requires a 국세청 승인번호. Nothing else proves the filing
           reached the NTS.
        3. Anything else is undetermined, and undetermined is not success.

        Returns:
            IssuedInvoiceResult reflecting what was actually observed
        """
        alert_text = await self._read_alert_text()

        if alert_text:
            # Failure first: check against the known error strings before
            # considering any success wording.
            if self._is_failure_alert(alert_text):
                self.log.warning("issue_rejected", alert=alert_text[:120])
                return IssuedInvoiceResult(
                    success=False,
                    invoice_number="",
                    issue_date=datetime.now(),
                    error_message=alert_text.strip(),
                )

            if self._is_success_alert(alert_text):
                await self._dismiss_alert()
            else:
                # An alert we do not recognise is not evidence of success.
                self.log.warning("issue_unrecognised_alert", alert=alert_text[:120])
                return IssuedInvoiceResult(
                    success=False,
                    invoice_number="",
                    issue_date=datetime.now(),
                    error_message=(
                        f"RESULT_UNDETERMINED: unrecognised response from Hometax: "
                        f"{alert_text.strip()[:200]}"
                    ),
                )

        invoice_number = await self._get_element_value("#resultInvoiceNum, #taxInvoiceNum")
        nts_confirm = await self._get_element_value("#resultNtsConfirmNum, #ntsConfirmNum")

        # The 국세청 승인번호 is the only proof the invoice was transmitted.
        if invoice_number and nts_confirm:
            return IssuedInvoiceResult(
                success=True,
                invoice_number=invoice_number,
                issue_date=datetime.now(),
                nts_confirm_number=nts_confirm,
            )

        if invoice_number and not nts_confirm:
            return IssuedInvoiceResult(
                success=False,
                invoice_number=invoice_number,
                issue_date=datetime.now(),
                error_message=(
                    "RESULT_UNDETERMINED: an invoice number was returned but no "
                    "국세청 승인번호. The filing may not have reached the NTS -- "
                    "reconcile against Hometax before treating it as issued."
                ),
            )

        # No alert, no invoice number, no confirmation number. Previously this
        # returned success=True with an empty invoice number, so a selector
        # mismatch or a page that never loaded was recorded as a successful
        # issuance.
        return IssuedInvoiceResult(
            success=False,
            invoice_number="",
            issue_date=datetime.now(),
            error_message=(
                "RESULT_UNDETERMINED: no confirmation and no error was found on the "
                "page. The selectors may not match the current Hometax DOM. The "
                "issuance status is unknown and must be reconciled."
            ),
        )

    async def _read_alert_text(self) -> str:
        """Return the visible alert text, or an empty string when there is none."""
        alert = await self.page.query_selector(SELECTORS["alert_popup"])
        if not alert:
            return ""

        if not await alert.is_visible():
            return ""

        text = await alert.text_content()
        return text or ""

    async def _dismiss_alert(self) -> None:
        """Close the alert popup if it has a confirm button."""
        alert = await self.page.query_selector(SELECTORS["alert_popup"])
        if not alert:
            return
        confirm_btn = await alert.query_selector(SELECTORS["confirm_btn"])
        if confirm_btn:
            await confirm_btn.click()

    @staticmethod
    def _is_failure_alert(alert_text: str) -> bool:
        """
        Whether the alert text matches a known failure message.

        Args:
            alert_text: Text read from the alert popup

        Returns:
            True if this is a rejection
        """
        text = alert_text.strip()
        if any(msg in text for msg in ERROR_MESSAGES.values()):
            return True
        # Generic failure wording, checked before any success wording.
        return any(token in text for token in ("실패", "오류", "에러", "불가", "거부"))

    @staticmethod
    def _is_success_alert(alert_text: str) -> bool:
        """
        Whether the alert text unambiguously reports success.

        Note the absence of a bare "발급" match: it appears in
        "발급에 실패했습니다" too.

        Args:
            alert_text: Text read from the alert popup

        Returns:
            True if this is a success notice
        """
        text = alert_text.strip()
        return any(
            token in text
            for token in ("정상적으로 발급", "발급이 완료", "정상 처리", "정상처리")
        )

    async def _get_element_value(self, selector: str) -> str:
        """Get value from element."""
        element = await self.page.query_selector(selector)
        if not element:
            return ""

        tag = await element.evaluate("el => el.tagName")
        if tag.lower() in ("input", "select"):
            return await element.input_value() or ""
        else:
            text = await element.text_content()
            return text.strip() if text else ""

    async def _wait_for_loading(self) -> None:
        """Wait for loading indicator to disappear."""
        try:
            await self.page.wait_for_selector(
                SELECTORS["loading_indicator"],
                state="hidden",
                timeout=TIMEOUTS["navigation"],
            )
        except PlaywrightTimeout:
            pass
        await asyncio.sleep(TIMEOUTS["animation"] / 1000)
