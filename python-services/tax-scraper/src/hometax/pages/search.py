"""
Tax invoice search page object.
"""
import asyncio
import re
from datetime import date, datetime
from typing import Optional

import structlog
from playwright.async_api import Page, TimeoutError as PlaywrightTimeout

from ..amounts import parse_amount
from ..constants import (
    MENU_TAX_INVOICE_SEARCH,
    SELECTORS,
    STATUS_MAP,
    TIMEOUTS,
)
from ..errors import HometaxScrapeError
from ..models import InvoiceType, TaxInvoice

logger = structlog.get_logger()

# Korean electronic tax invoice numbers are digits and hyphens only.
_INVOICE_NUMBER_RE = re.compile(r"\A[0-9\-]{8,32}\Z")


def _css_quote(value: str) -> str:
    """
    Quote a value for safe use inside a CSS attribute selector.

    Args:
        value: Attribute value

    Returns:
        A single-quoted, escaped CSS string
    """
    escaped = value.replace("\\", "\\\\").replace("'", "\\'")
    return f"'{escaped}'"


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

        except HometaxScrapeError:
            raise
        except PlaywrightTimeout as e:
            # A timeout is a failed query, not an empty period. Returning []
            # here made a session expiry or a Hometax UI change look like
            # "0 세금계산서 in this period" -- a number that goes straight into
            # 부가가치세 신고 as missing 매출/매입.
            self.log.error("search_timeout", error=str(e))
            raise HometaxScrapeError(
                f"Hometax invoice search timed out for {start_date}~{end_date}; "
                f"the result set is unknown, not empty"
            ) from e
        except Exception as e:
            self.log.error("search_error", error=type(e).__name__)
            raise HometaxScrapeError(
                f"Hometax invoice search failed for {start_date}~{end_date}: "
                f"{type(e).__name__}"
            ) from e

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

    # Element that Hometax renders when a search genuinely matched nothing.
    # Its presence is what distinguishes "no 세금계산서" from "the table never
    # appeared".
    EMPTY_RESULT_SELECTOR = ".no_data, .empty_data, #noResultMsg"

    async def _parse_search_results(self) -> list[TaxInvoice]:
        """
        Parse the search result table.

        Distinguishes three outcomes that were previously collapsed into an
        empty list:

        - The table appeared with rows -> parse them, and fail if any row does
          not parse. A partial parse silently under-reports 매출/매입.
        - The table did not appear but an explicit "no results" marker did ->
          a genuine empty period.
        - Neither appeared -> the query failed and the result is unknown.

        Returns:
            Parsed invoices

        Raises:
            HometaxScrapeError: If the outcome cannot be established, or if any
                row failed to parse
        """
        invoices: list[TaxInvoice] = []

        try:
            await self.page.wait_for_selector(
                SELECTORS["result_table"],
                timeout=TIMEOUTS["element_wait"],
            )
        except PlaywrightTimeout as exc:
            empty_marker = await self.page.query_selector(self.EMPTY_RESULT_SELECTOR)
            if empty_marker and await empty_marker.is_visible():
                self.log.info("search_returned_no_rows")
                return []

            raise HometaxScrapeError(
                "Neither the result table nor an empty-result marker appeared. "
                "The query outcome is unknown -- do not treat it as zero invoices."
            ) from exc

        rows = await self.page.query_selector_all(SELECTORS["result_rows"])

        failures: list[str] = []
        for index, row in enumerate(rows):
            try:
                invoice = await self._parse_row(row)
            except Exception as exc:
                failures.append(f"row {index}: {type(exc).__name__}")
                continue

            if invoice is None:
                failures.append(f"row {index}: unrecognised row shape")
                continue

            invoices.append(invoice)

        if failures:
            # Reporting "3 invoices retrieved" when 10 rows were present and 7
            # failed to parse is worse than reporting nothing: the shortfall is
            # invisible downstream.
            self.log.error(
                "parse_row_failures",
                failed=len(failures),
                total=len(rows),
                detail=failures[:5],
            )
            raise HometaxScrapeError(
                f"{len(failures)} of {len(rows)} result rows could not be parsed; "
                f"refusing to return a partial invoice set"
            )

        return invoices

    async def _parse_row(self, row) -> Optional[TaxInvoice]:
        """
        Parse a single result row into a TaxInvoice.

        Args:
            row: Playwright element handle for the table row

        Returns:
            Parsed invoice, or None if the row is not a data row

        Raises:
            Exception: Propagated to the caller, which counts failures. Swallowing
                them here dropped invoices silently.
        """
        cells = await row.query_selector_all("td")
        if len(cells) < 8:
            # Header/spacer rows have no data cells; not an error.
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
        status_code = await self._get_cell_text(cells[8]) if len(cells) > 8 else ""
        nts_confirm = await self._get_cell_text(cells[9]) if len(cells) > 9 else ""

        issue_date = datetime.strptime(issue_date_str, "%Y-%m-%d")

        supply_amount = parse_amount(supply_amount_str)
        tax_amount = parse_amount(tax_amount_str)

        # An unknown status code used to map to "confirmed" (국세청 확인 완료) --
        # the most favourable reading of a value we do not understand.
        status = STATUS_MAP.get(status_code)
        if status is None:
            raise ValueError(f"Unknown Hometax status code {status_code!r}")

        return TaxInvoice(
            invoice_number=invoice_number,
            issue_date=issue_date,
            invoice_type=InvoiceType.SALES,
            status=status,
            supplier_business_number=supplier_brn.replace("-", ""),
            supplier_name=supplier_name,
            buyer_business_number=buyer_brn.replace("-", ""),
            buyer_name=buyer_name,
            supply_amount=supply_amount,
            tax_amount=tax_amount,
            total_amount=supply_amount + tax_amount,
            nts_confirm_number=nts_confirm if nts_confirm else None,
        )

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

        # Reject anything that is not an invoice-number shape before it reaches
        # a selector. Interpolating raw input into a CSS selector let a value
        # containing a quote break out of the attribute and match a different
        # element -- i.e. open somebody else's invoice.
        if not _INVOICE_NUMBER_RE.match(invoice_number):
            raise HometaxScrapeError(
                "Invoice number contains characters that are not permitted"
            )

        try:
            # Locator API with a bound value: Playwright escapes the argument,
            # so the value can never alter the selector's structure.
            link = self.page.locator(
                f"a[data-invoice={_css_quote(invoice_number)}]"
            ).first
            if await link.count() > 0:
                await link.click()
            else:
                # get_by_text takes the string as data, not as selector syntax.
                await self.page.get_by_text(invoice_number, exact=True).first.click()

            await self._wait_for_loading()

            return await self._parse_detail_page()

        except HometaxScrapeError:
            raise
        except Exception as e:
            self.log.error("get_detail_error", error=type(e).__name__)
            raise HometaxScrapeError(
                f"Could not open the detail page for the requested invoice: "
                f"{type(e).__name__}"
            ) from e

    async def _parse_detail_page(self) -> TaxInvoice:
        """
        Parse the tax invoice detail page.

        Returns:
            The parsed invoice

        Raises:
            HometaxScrapeError: If the page could not be parsed. Returning None
                on failure made an unreadable page indistinguishable from an
                invoice that does not exist.
        """
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
            supply_amount = parse_amount(supply_amount_str)
            tax_amount = parse_amount(tax_amount_str)

            return TaxInvoice(
                invoice_number=invoice_number,
                issue_date=issue_date,
                invoice_type=InvoiceType.SALES,
                # The detail page does not expose a status field; report what we
                # know rather than asserting 국세청 confirmation.
                status="issued",
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
            self.log.error("parse_detail_error", error=type(e).__name__)
            raise HometaxScrapeError(
                f"Could not parse the tax invoice detail page: {type(e).__name__}"
            ) from e

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
