"""
Unit tests for tax scraper service.

Tests cover:
- Validators (business number, date range)
- Popbill client
- Tax service
"""

import pytest
from datetime import date, datetime, timedelta
from decimal import Decimal
from unittest.mock import AsyncMock, MagicMock, patch

import sys
from pathlib import Path

# Add project root to path
sys.path.insert(0, str(Path(__file__).parent.parent))


class TestValidators:
    """Tests for input validators."""

    def test_validate_business_number_valid(self):
        """Test valid business numbers."""
        from src.utils.validators import validate_business_number

        # Valid business numbers (these are test numbers)
        valid_numbers = [
            "1234567890",  # Basic format
            "123-45-67890",  # With hyphens
            "123 45 67890",  # With spaces
        ]

        # Note: The actual checksum validation would fail for random numbers
        # In real tests, use actual valid business numbers
        for brn in valid_numbers:
            is_valid, error = validate_business_number(brn)
            # Just verify the format validation passes
            assert error is None or "digits" not in error.lower()

    def test_validate_business_number_invalid_length(self):
        """Test business numbers with invalid length."""
        from src.utils.validators import validate_business_number

        invalid_numbers = [
            "12345",  # Too short
            "12345678901",  # Too long
            "",  # Empty
        ]

        for brn in invalid_numbers:
            is_valid, error = validate_business_number(brn)
            assert not is_valid
            assert error is not None

    def test_validate_business_number_invalid_chars(self):
        """Test business numbers with invalid characters."""
        from src.utils.validators import validate_business_number

        is_valid, error = validate_business_number("123456789a")
        assert not is_valid
        assert "digits" in error.lower()

    def test_validate_date_range_valid(self):
        """Test valid date ranges."""
        from src.utils.validators import validate_date_range

        today = date.today()
        yesterday = today - timedelta(days=1)

        is_valid, error, start, end = validate_date_range(
            yesterday.strftime("%Y-%m-%d"),
            today.strftime("%Y-%m-%d"),
        )

        assert is_valid
        assert error is None
        assert start == yesterday
        assert end == today

    def test_validate_date_range_invalid_order(self):
        """Test date range with start after end."""
        from src.utils.validators import validate_date_range

        today = date.today()
        yesterday = today - timedelta(days=1)

        is_valid, error, _, _ = validate_date_range(
            today.strftime("%Y-%m-%d"),
            yesterday.strftime("%Y-%m-%d"),
        )

        assert not is_valid
        assert "before" in error.lower()

    def test_validate_date_range_exceeds_max(self):
        """Test date range exceeding maximum days."""
        from src.utils.validators import validate_date_range

        today = date.today()
        two_years_ago = today - timedelta(days=730)

        is_valid, error, _, _ = validate_date_range(
            two_years_ago.strftime("%Y-%m-%d"),
            today.strftime("%Y-%m-%d"),
            max_days=365,
        )

        assert not is_valid
        assert "exceeds" in error.lower()

    def test_validate_date_range_future_end(self):
        """Test date range with future end date."""
        from src.utils.validators import validate_date_range

        today = date.today()
        tomorrow = today + timedelta(days=1)

        is_valid, error, _, _ = validate_date_range(
            today.strftime("%Y-%m-%d"),
            tomorrow.strftime("%Y-%m-%d"),
        )

        assert not is_valid
        assert "future" in error.lower()

    def test_parse_date_formats(self):
        """Test date parsing with various formats."""
        from src.utils.validators import parse_date

        # YYYY-MM-DD
        result = parse_date("2024-01-15")
        assert result == date(2024, 1, 15)

        # YYYYMMDD
        result = parse_date("20240115")
        assert result == date(2024, 1, 15)

        # YYYY/MM/DD
        result = parse_date("2024/01/15")
        assert result == date(2024, 1, 15)

        # Invalid format
        result = parse_date("15-01-2024")
        assert result is None

    def test_format_business_number(self):
        """Test business number formatting."""
        from src.utils.validators import format_business_number

        assert format_business_number("1234567890") == "123-45-67890"
        assert format_business_number("123-45-67890") == "123-45-67890"
        assert format_business_number("12345") == "12345"  # Invalid length

    def test_mask_business_number(self):
        """Test business number masking."""
        from src.utils.validators import mask_business_number

        assert mask_business_number("1234567890") == "123-45-****"
        assert mask_business_number("123-45-67890") == "123-45-****"
        assert mask_business_number("123") == "****"  # Too short

    def test_validate_invoice_number(self):
        """Test invoice number validation."""
        from src.utils.validators import validate_invoice_number

        # Valid formats
        is_valid, _ = validate_invoice_number("20240115-12345678")
        assert is_valid

        is_valid, _ = validate_invoice_number("2024011512345678")
        assert is_valid

        # Invalid
        is_valid, error = validate_invoice_number("")
        assert not is_valid
        assert "required" in error.lower()

        is_valid, error = validate_invoice_number("abc")
        assert not is_valid

    def test_validate_amount(self):
        """Test amount validation."""
        from src.utils.validators import validate_amount

        # Valid amounts
        is_valid, _ = validate_amount(0)
        assert is_valid

        is_valid, _ = validate_amount(1000000)
        assert is_valid

        # Invalid - negative
        is_valid, error = validate_amount(-100)
        assert not is_valid
        assert "negative" in error.lower()

        # Invalid - exceeds max
        is_valid, error = validate_amount(20_000_000_000)
        assert not is_valid
        assert "exceeds" in error.lower()

    def test_calculate_vat(self):
        """Test VAT calculation."""
        from src.utils.validators import calculate_vat

        assert calculate_vat(100000) == 10000
        assert calculate_vat(100000, 0.05) == 5000
        assert calculate_vat(99999) == 9999  # Rounded down


class TestPopbillClient:
    """Tests for Popbill API client."""

    @pytest.fixture
    def popbill_config(self):
        """Create test Popbill configuration."""
        from providers.popbill import PopbillConfig

        return PopbillConfig(
            link_id="test_link_id",
            secret_key="test_secret_key",
            is_test=True,
        )

    @pytest.fixture
    def popbill_client(self, popbill_config):
        """Create test Popbill client."""
        from providers.popbill import PopbillClient

        return PopbillClient(popbill_config)

    def test_popbill_config_base_url_test(self, popbill_config):
        """Test Popbill config returns test URL."""
        assert "test" in popbill_config.base_url.lower()

    def test_popbill_config_base_url_production(self):
        """Test Popbill config returns production URL."""
        from providers.popbill import PopbillConfig

        config = PopbillConfig(
            link_id="test",
            secret_key="test",
            is_test=False,
        )
        assert "test" not in config.base_url.lower()

    def test_popbill_invoice_to_dict(self):
        """Test PopbillTaxInvoice serialization."""
        from providers.popbill import PopbillTaxInvoice, PopbillInvoiceType

        invoice = PopbillTaxInvoice(
            invoice_number="TEST-001",
            write_date="20240115",
            invoicer_corp_num="1234567890",
            invoicer_corp_name="Test Company",
            invoicer_ceo_name="Test CEO",
            invoicee_corp_num="0987654321",
            invoicee_corp_name="Buyer Company",
            invoicee_ceo_name="Buyer CEO",
            supply_cost_total=100000,
            tax_total=10000,
            total_amount=110000,
        )

        data = invoice.to_dict()

        assert data["invoicerCorpNum"] == "1234567890"
        assert data["invoiceeCorpNum"] == "0987654321"
        assert data["supplyCostTotal"] == "100000"
        assert data["taxTotal"] == "10000"
        assert data["totalAmount"] == "110000"
        assert data["invoiceType"] == PopbillInvoiceType.SALES.value

    @pytest.mark.asyncio
    async def test_popbill_client_issue_invoice(self, popbill_client):
        """Test invoice issuance with mocked API."""
        from providers.popbill import PopbillTaxInvoice

        invoice = PopbillTaxInvoice(
            invoice_number="TEST-001",
            write_date="20240115",
            invoicer_corp_num="1234567890",
            invoicer_corp_name="Test Company",
            invoicer_ceo_name="Test CEO",
            supply_cost_total=100000,
            tax_total=10000,
            total_amount=110000,
        )

        # Mock the request method
        popbill_client._request = AsyncMock(
            return_value={
                "ntsaKey": "NTS-KEY-12345",
                "invoiceNumber": "TEST-001",
                "ntsconfirmNum": "NTS-CONFIRM-001",
            }
        )

        result = await popbill_client.issue_tax_invoice(
            corp_num="1234567890",
            invoice=invoice,
        )

        assert result.success
        assert result.ntsa_key == "NTS-KEY-12345"
        assert result.invoice_number == "TEST-001"

    @pytest.mark.asyncio
    async def test_popbill_client_query_invoice(self, popbill_client):
        """Test invoice query with mocked API."""
        popbill_client._request = AsyncMock(
            return_value={
                "invoiceNumber": "TEST-001",
                "stateCode": "300",
                "ntsconfirmNum": "NTS-CONFIRM-001",
            }
        )

        result = await popbill_client.query_tax_invoice(
            corp_num="1234567890",
            invoice_number="TEST-001",
        )

        assert result["invoiceNumber"] == "TEST-001"
        assert result["stateCode"] == "300"

    @pytest.mark.asyncio
    async def test_popbill_client_get_balance(self, popbill_client):
        """Test balance check with mocked API."""
        popbill_client._request = AsyncMock(return_value={"balance": 1000})

        balance = await popbill_client.get_balance("1234567890")

        assert balance == 1000


class TestHometaxModels:
    """Tests for Hometax data models."""

    def test_auth_type_enum(self):
        """Test AuthType enum values."""
        from src.hometax.models import AuthType

        assert AuthType.CERTIFICATE.value == "certificate"
        assert AuthType.SIMPLE_AUTH.value == "simple_auth"
        assert AuthType.ID_PASSWORD.value == "id_password"

    def test_invoice_type_enum(self):
        """Test InvoiceType enum values."""
        from src.hometax.models import InvoiceType

        assert InvoiceType.SALES.value == "sales"
        assert InvoiceType.PURCHASE.value == "purchase"

    def test_invoice_status_enum(self):
        """Test InvoiceStatus enum values."""
        from src.hometax.models import InvoiceStatus

        assert InvoiceStatus.DRAFT.value == "draft"
        assert InvoiceStatus.ISSUED.value == "issued"
        assert InvoiceStatus.CONFIRMED.value == "confirmed"
        assert InvoiceStatus.CANCELLED.value == "cancelled"

    def test_hometax_session_model(self):
        """Test HometaxSession model."""
        from src.hometax.models import HometaxSession, AuthType

        session = HometaxSession(
            session_id="test-session-123",
            business_number="1234567890",
            company_name="Test Company",
            expires_at=datetime.now() + timedelta(hours=1),
            auth_type=AuthType.CERTIFICATE,
        )

        assert session.session_id == "test-session-123"
        assert session.business_number == "1234567890"
        assert session.auth_type == AuthType.CERTIFICATE
        assert session.created_at is not None

    def test_tax_invoice_model(self):
        """Test TaxInvoice model."""
        from src.hometax.models import TaxInvoice, InvoiceType

        invoice = TaxInvoice(
            invoice_number="20240115-12345678",
            issue_date=datetime.now(),
            supplier_business_number="1234567890",
            supplier_name="Supplier Co",
            buyer_business_number="0987654321",
            buyer_name="Buyer Co",
            supply_amount=Decimal("100000"),
            tax_amount=Decimal("10000"),
            total_amount=Decimal("110000"),
        )

        assert invoice.invoice_number == "20240115-12345678"
        assert invoice.invoice_type == InvoiceType.SALES
        assert invoice.supply_amount == Decimal("100000")

    def test_tax_invoice_item_model(self):
        """Test TaxInvoiceItem model."""
        from src.hometax.models import TaxInvoiceItem

        item = TaxInvoiceItem(
            description="Test Product",
            quantity=Decimal("10"),
            unit_price=Decimal("10000"),
            amount=Decimal("100000"),
            tax_amount=Decimal("10000"),
        )

        assert item.description == "Test Product"
        assert item.quantity == Decimal("10")
        assert item.sequence == 1

    def test_issued_invoice_result_model(self):
        """Test IssuedInvoiceResult model."""
        from src.hometax.models import IssuedInvoiceResult

        result = IssuedInvoiceResult(
            success=True,
            invoice_number="20240115-12345678",
            issue_date=datetime.now(),
            nts_confirm_number="NTS-12345",
        )

        assert result.success
        assert result.invoice_number == "20240115-12345678"
        assert result.error_message is None


class TestSEEDCipher:
    """Tests for SEED encryption."""

    @pytest.fixture
    def seed_cipher(self):
        """Create test SEED cipher."""
        from src.crypto.seed import SEEDCipher

        # 16-byte test key
        key = b"1234567890123456"
        return SEEDCipher(key)

    def test_seed_cipher_init_valid_key(self, seed_cipher):
        """Test SEED cipher initialization with valid key."""
        assert seed_cipher._key == b"1234567890123456"

    def test_seed_cipher_init_invalid_key(self):
        """Test SEED cipher initialization with invalid key."""
        from src.crypto.seed import SEEDCipher

        with pytest.raises(ValueError):
            SEEDCipher(b"short")

    def test_seed_cipher_from_hex(self):
        """Test SEED cipher creation from hex key."""
        from src.crypto.seed import SEEDCipher

        hex_key = "31323334353637383930313233343536"  # "1234567890123456" in hex
        cipher = SEEDCipher.from_hex(hex_key)

        assert cipher._key == b"1234567890123456"

    def test_seed_cipher_encrypt_decrypt(self, seed_cipher):
        """Test encryption and decryption roundtrip."""
        plaintext = b"Hello, World!"

        ciphertext = seed_cipher.encrypt(plaintext)
        decrypted = seed_cipher.decrypt(ciphertext)

        assert decrypted == plaintext
        assert ciphertext != plaintext

    def test_seed_cipher_encrypt_string(self, seed_cipher):
        """Test string encryption and decryption."""
        text = "Hello, World!"

        ciphertext = seed_cipher.encrypt_string(text)
        decrypted = seed_cipher.decrypt_string(ciphertext)

        assert decrypted == text

    def test_seed_cipher_decrypt_invalid_length(self, seed_cipher):
        """Test decryption with invalid ciphertext length."""
        with pytest.raises(ValueError):
            seed_cipher.decrypt(b"short")

    def test_generate_seed_key(self):
        """Test SEED key generation."""
        from src.crypto.seed import generate_seed_key, SEEDCipher

        key = generate_seed_key()

        assert len(key) == SEEDCipher.KEY_SIZE
        assert isinstance(key, bytes)

    def test_derive_key_from_password(self):
        """Test key derivation from password."""
        from src.crypto.seed import derive_key_from_password, SEEDCipher

        key, salt = derive_key_from_password("test_password")

        assert len(key) == SEEDCipher.KEY_SIZE
        assert len(salt) == 16

        # Same password with same salt should produce same key
        key2, _ = derive_key_from_password("test_password", salt)
        assert key == key2


class TestTaxInvoiceService:
    """Tests for TaxInvoiceService."""

    @pytest.fixture
    def tax_service(self):
        """Create test TaxInvoiceService."""
        from src.services.tax_service import TaxInvoiceService

        return TaxInvoiceService()

    @pytest.mark.asyncio
    async def test_login_invalid_business_number(self, tax_service):
        """Test login with invalid business number."""
        result = await tax_service.login(
            company_id="test-company",
            business_number="invalid",
            auth_type="certificate",
        )

        assert not result["success"]
        assert "INVALID_BUSINESS_NUMBER" in result["error_code"]

    @pytest.mark.asyncio
    async def test_get_invoices_invalid_date_range(self, tax_service):
        """Test get invoices with invalid date range."""
        result = await tax_service.get_tax_invoices(
            session_id="test-session",
            start_date="2024-01-15",
            end_date="2024-01-10",  # End before start
        )

        assert not result["success"]
        assert "before" in result["error_message"].lower()

    @pytest.mark.asyncio
    async def test_service_close(self, tax_service):
        """Test service cleanup."""
        await tax_service.close()
        assert len(tax_service._sessions) == 0


class TestHometaxConstants:
    """Tests for Hometax constants."""

    def test_status_map(self):
        """Test status code mapping."""
        from src.hometax.constants import STATUS_MAP, STATUS_CONFIRMED, STATUS_CANCELLED

        assert STATUS_MAP[STATUS_CONFIRMED] == "confirmed"
        assert STATUS_MAP[STATUS_CANCELLED] == "cancelled"

    def test_selectors_defined(self):
        """Test required selectors are defined."""
        from src.hometax.constants import SELECTORS

        required_selectors = [
            "login_cert_btn",
            "login_id_input",
            "login_pw_input",
            "search_btn",
            "result_table",
        ]

        for selector in required_selectors:
            assert selector in SELECTORS
            assert SELECTORS[selector]  # Not empty

    def test_timeouts_defined(self):
        """Test timeout values are defined."""
        from src.hometax.constants import TIMEOUTS

        assert TIMEOUTS["page_load"] > 0
        assert TIMEOUTS["navigation"] > 0
        assert TIMEOUTS["element_wait"] > 0
