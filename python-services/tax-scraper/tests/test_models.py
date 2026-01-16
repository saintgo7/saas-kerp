"""
Tests for tax invoice data models.
"""
from datetime import datetime
from decimal import Decimal

import pytest

from src.hometax.models import (
    AuthType,
    HometaxSession,
    InvoiceStatus,
    InvoiceType,
    IssuedInvoiceResult,
    TaxInvoice,
    TaxInvoiceItem,
)


class TestHometaxSession:
    """Tests for HometaxSession model."""

    def test_create_session(self):
        """Test creating a valid session."""
        session = HometaxSession(
            session_id="test-session-123",
            business_number="1234567890",
            company_name="Test Company",
            expires_at=datetime(2024, 1, 15, 12, 0, 0),
            auth_type=AuthType.CERTIFICATE,
        )

        assert session.session_id == "test-session-123"
        assert session.business_number == "1234567890"
        assert session.company_name == "Test Company"
        assert session.auth_type == AuthType.CERTIFICATE

    def test_session_created_at_default(self):
        """Test that created_at defaults to current time."""
        session = HometaxSession(
            session_id="test",
            business_number="1234567890",
            company_name="Test",
            expires_at=datetime.now(),
            auth_type=AuthType.ID_PASSWORD,
        )

        assert session.created_at is not None
        assert isinstance(session.created_at, datetime)


class TestTaxInvoice:
    """Tests for TaxInvoice model."""

    def test_create_sales_invoice(self):
        """Test creating a sales invoice."""
        invoice = TaxInvoice(
            invoice_number="20240115-001",
            issue_date=datetime(2024, 1, 15),
            invoice_type=InvoiceType.SALES,
            supplier_business_number="1234567890",
            supplier_name="Supplier Co.",
            buyer_business_number="0987654321",
            buyer_name="Buyer Co.",
            supply_amount=Decimal("100000"),
            tax_amount=Decimal("10000"),
            total_amount=Decimal("110000"),
        )

        assert invoice.invoice_number == "20240115-001"
        assert invoice.invoice_type == InvoiceType.SALES
        assert invoice.status == InvoiceStatus.DRAFT
        assert invoice.total_amount == Decimal("110000")

    def test_invoice_with_items(self):
        """Test invoice with line items."""
        items = [
            TaxInvoiceItem(
                sequence=1,
                description="Product A",
                quantity=Decimal("10"),
                unit_price=Decimal("5000"),
                amount=Decimal("50000"),
                tax_amount=Decimal("5000"),
            ),
            TaxInvoiceItem(
                sequence=2,
                description="Product B",
                quantity=Decimal("5"),
                unit_price=Decimal("10000"),
                amount=Decimal("50000"),
                tax_amount=Decimal("5000"),
            ),
        ]

        invoice = TaxInvoice(
            invoice_number="20240115-002",
            issue_date=datetime(2024, 1, 15),
            supplier_business_number="1234567890",
            supplier_name="Supplier",
            buyer_business_number="0987654321",
            buyer_name="Buyer",
            supply_amount=Decimal("100000"),
            tax_amount=Decimal("10000"),
            total_amount=Decimal("110000"),
            items=items,
        )

        assert len(invoice.items) == 2
        assert invoice.items[0].description == "Product A"
        assert invoice.items[1].description == "Product B"

    def test_invalid_business_number_length(self):
        """Test that invalid business number length raises error."""
        with pytest.raises(ValueError):
            TaxInvoice(
                invoice_number="test",
                issue_date=datetime.now(),
                supplier_business_number="123",  # Too short
                supplier_name="Test",
                buyer_business_number="0987654321",
                buyer_name="Test",
                supply_amount=Decimal("100"),
                tax_amount=Decimal("10"),
                total_amount=Decimal("110"),
            )


class TestIssuedInvoiceResult:
    """Tests for IssuedInvoiceResult model."""

    def test_successful_result(self):
        """Test successful invoice issuance result."""
        result = IssuedInvoiceResult(
            success=True,
            invoice_number="20240115-001",
            issue_date=datetime(2024, 1, 15),
            nts_confirm_number="NTS-12345",
        )

        assert result.success is True
        assert result.nts_confirm_number == "NTS-12345"
        assert result.error_message is None

    def test_failed_result(self):
        """Test failed invoice issuance result."""
        result = IssuedInvoiceResult(
            success=False,
            invoice_number="",
            issue_date=datetime.now(),
            error_message="Authentication failed",
        )

        assert result.success is False
        assert result.error_message == "Authentication failed"
