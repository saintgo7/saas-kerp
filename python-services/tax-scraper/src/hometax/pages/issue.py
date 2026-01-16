"""
Tax invoice issuance page object.
"""
import asyncio
from datetime import datetime
from typing import Optional

import structlog
from playwright.async_api import Page, TimeoutError as PlaywrightTimeout

from ..constants import (
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
        """Check the result of invoice issuance."""
        # Check for error alert
        alert = await self.page.query_selector(SELECTORS["alert_popup"])
        if alert:
            alert_visible = await alert.is_visible()
            if alert_visible:
                alert_text = await alert.text_content()
                # Check if it's a success message
                if "정상" in alert_text or "완료" in alert_text or "발급" in alert_text:
                    # Close alert
                    confirm_btn = await alert.query_selector(SELECTORS["confirm_btn"])
                    if confirm_btn:
                        await confirm_btn.click()
                else:
                    return IssuedInvoiceResult(
                        success=False,
                        invoice_number="",
                        issue_date=datetime.now(),
                        error_message=alert_text.strip() if alert_text else "Unknown error",
                    )

        # Try to get issued invoice info
        invoice_number = await self._get_element_value("#resultInvoiceNum, #taxInvoiceNum")
        nts_confirm = await self._get_element_value("#resultNtsConfirmNum, #ntsConfirmNum")

        if invoice_number:
            return IssuedInvoiceResult(
                success=True,
                invoice_number=invoice_number,
                issue_date=datetime.now(),
                nts_confirm_number=nts_confirm if nts_confirm else None,
            )

        # Default to success if no error detected
        return IssuedInvoiceResult(
            success=True,
            invoice_number="",
            issue_date=datetime.now(),
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
