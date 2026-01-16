"""
Tax invoice search page object.
"""
import asyncio
from datetime import date, datetime
from decimal import Decimal
from typing import Optional

import structlog
from playwright.async_api import Page, TimeoutError as PlaywrightTimeout

from ..constants import (
    MENU_TAX_INVOICE_SEARCH,
    SELECTORS,
    STATUS_MAP,
    TIMEOUTS,
)
from ..models import InvoiceType, TaxInvoice

logger = structlog.get_logger()


class TaxInvoiceSearchPage:
    """
    Page object for tax invoice search operations.
    """

    def __init__(self, page: Page) -> None:
        """Initialize search page with Playwright page instance."""
        self.page = page
        self.log = logger.bind(component="TaxInvoiceSearchPage")

    async def navigate(self, invoice_type: InvoiceType = InvoiceType.SALES) -> None:
        """
        Navigate to tax invoice search page.

        Args:
            invoice_type: Type of invoices to search (sales/purchase)
        """
        self.log.info("navigating_to_search", invoice_type=invoice_type.value)

        # Navigate to appropriate menu
        menu_id = MENU_TAX_INVOICE_SEARCH
        await self.page.evaluate(f"wqRunMenu('{menu_id}')")
        await self._wait_for_loading()

        self.log.info("search_page_loaded")

    async def search(
        self,
        start_date: date,
        end_date: date,
        invoice_type: Optional[InvoiceType] = None,
        business_number: Optional[str] = None,
    ) -> list[TaxInvoice]:
        """
        Search for tax invoices.

        Args:
            start_date: Search start date
            end_date: Search end date
            invoice_type: Filter by invoice type
            business_number: Filter by counterparty business number

        Returns:
            List of matching tax invoices
        """
        self.log.info(
            "search_started",
            start_date=start_date.isoformat(),
            end_date=end_date.isoformat(),
        )

        try:
            # Set search criteria
            await self._set_date_range(start_date, end_date)

            if business_number:
                await self._set_business_number(business_number)

            # Execute search
            await self.page.click(SELECTORS["search_btn"])
            await self._wait_for_loading()

            # Parse results
            invoices = await self._parse_search_results()

            self.log.info("search_complete", count=len(invoices))
            return invoices

        except PlaywrightTimeout as e:
            self.log.error("search_timeout", error=str(e))
            return []
        except Exception as e:
            self.log.error("search_error", error=str(e))
            return []

    async def _set_date_range(self, start_date: date, end_date: date) -> None:
        """Set search date range."""
        start_str = start_date.strftime("%Y%m%d")
        end_str = end_date.strftime("%Y%m%d")

        await self.page.fill(SELECTORS["search_start_date"], start_str)
        await self.page.fill(SELECTORS["search_end_date"], end_str)

    async def _set_business_number(self, business_number: str) -> None:
        """Set business number filter."""
        # Selector depends on search type (sales/purchase)
        selector = "#schCorpNum, #invoiceeCorpNum"
        await self.page.fill(selector, business_number)

    async def _parse_search_results(self) -> list[TaxInvoice]:
        """Parse search result table."""
        invoices: list[TaxInvoice] = []

        try:
            # Wait for result table
            await self.page.wait_for_selector(
                SELECTORS["result_table"],
                timeout=TIMEOUTS["element_wait"],
            )

            rows = await self.page.query_selector_all(SELECTORS["result_rows"])

            for row in rows:
                invoice = await self._parse_row(row)
                if invoice:
                    invoices.append(invoice)

        except PlaywrightTimeout:
            self.log.warning("no_search_results")
        except Exception as e:
            self.log.error("parse_results_error", error=str(e))

        return invoices

    async def _parse_row(self, row) -> Optional[TaxInvoice]:
        """Parse a single result row into TaxInvoice."""
        try:
            cells = await row.query_selector_all("td")
            if len(cells) < 8:
                return None

            # Extract cell values (order depends on Hometax table structure)
            invoice_number = await self._get_cell_text(cells[0])
            issue_date_str = await self._get_cell_text(cells[1])
            supplier_brn = await self._get_cell_text(cells[2])
            supplier_name = await self._get_cell_text(cells[3])
            buyer_brn = await self._get_cell_text(cells[4])
            buyer_name = await self._get_cell_text(cells[5])
            supply_amount_str = await self._get_cell_text(cells[6])
            tax_amount_str = await self._get_cell_text(cells[7])
            status_code = await self._get_cell_text(cells[8]) if len(cells) > 8 else "04"
            nts_confirm = await self._get_cell_text(cells[9]) if len(cells) > 9 else ""

            # Parse date
            issue_date = datetime.strptime(issue_date_str, "%Y-%m-%d")

            # Parse amounts
            supply_amount = Decimal(supply_amount_str.replace(",", ""))
            tax_amount = Decimal(tax_amount_str.replace(",", ""))

            return TaxInvoice(
                invoice_number=invoice_number,
                issue_date=issue_date,
                invoice_type=InvoiceType.SALES,
                status=STATUS_MAP.get(status_code, "confirmed"),
                supplier_business_number=supplier_brn.replace("-", ""),
                supplier_name=supplier_name,
                buyer_business_number=buyer_brn.replace("-", ""),
                buyer_name=buyer_name,
                supply_amount=supply_amount,
                tax_amount=tax_amount,
                total_amount=supply_amount + tax_amount,
                nts_confirm_number=nts_confirm if nts_confirm else None,
            )

        except Exception as e:
            self.log.warning("parse_row_error", error=str(e))
            return None

    async def _get_cell_text(self, cell) -> str:
        """Get text content from a table cell."""
        text = await cell.text_content()
        return text.strip() if text else ""

    async def get_invoice_detail(self, invoice_number: str) -> Optional[TaxInvoice]:
        """
        Get detailed information for a specific invoice.

        Args:
            invoice_number: Tax invoice number

        Returns:
            TaxInvoice with full details
        """
        self.log.info("get_invoice_detail", invoice_number=invoice_number)

        try:
            # Click on invoice number to open detail
            invoice_link = await self.page.query_selector(
                f"a[data-invoice='{invoice_number}'], td:has-text('{invoice_number}')"
            )
            if invoice_link:
                await invoice_link.click()
                await self._wait_for_loading()

            # Parse detail page
            invoice = await self._parse_detail_page()
            return invoice

        except Exception as e:
            self.log.error("get_detail_error", error=str(e))
            return None

    async def _parse_detail_page(self) -> Optional[TaxInvoice]:
        """Parse tax invoice detail page."""
        try:
            invoice_number = await self._get_element_value(SELECTORS["invoice_number"])
            issue_date_str = await self._get_element_value(SELECTORS["issue_date"])
            supplier_brn = await self._get_element_value(SELECTORS["supplier_brn"])
            supplier_name = await self._get_element_value(SELECTORS["supplier_name"])
            buyer_brn = await self._get_element_value(SELECTORS["buyer_brn"])
            buyer_name = await self._get_element_value(SELECTORS["buyer_name"])
            supply_amount_str = await self._get_element_value(SELECTORS["supply_amount"])
            tax_amount_str = await self._get_element_value(SELECTORS["tax_amount"])
            nts_confirm = await self._get_element_value(SELECTORS["nts_confirm"])

            issue_date = datetime.strptime(issue_date_str, "%Y-%m-%d")
            supply_amount = Decimal(supply_amount_str.replace(",", ""))
            tax_amount = Decimal(tax_amount_str.replace(",", ""))

            return TaxInvoice(
                invoice_number=invoice_number,
                issue_date=issue_date,
                invoice_type=InvoiceType.SALES,
                status="confirmed",
                supplier_business_number=supplier_brn.replace("-", ""),
                supplier_name=supplier_name,
                buyer_business_number=buyer_brn.replace("-", ""),
                buyer_name=buyer_name,
                supply_amount=supply_amount,
                tax_amount=tax_amount,
                total_amount=supply_amount + tax_amount,
                nts_confirm_number=nts_confirm if nts_confirm else None,
            )

        except Exception as e:
            self.log.error("parse_detail_error", error=str(e))
            return None

    async def _get_element_value(self, selector: str) -> str:
        """Get value from form element or text from span."""
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
