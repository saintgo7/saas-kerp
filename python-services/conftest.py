"""Shared pytest fixtures for all Python services."""

import os
import secrets
import sys
from pathlib import Path

import pytest

# Add shared module to path
sys.path.insert(0, str(Path(__file__).parent / "shared"))


# Crypto fixtures
#
# These are real random keys, not `bytes(n)`. Every one of them used to be
# all-zero -- exactly the value the production code fell back to when
# ARIA_ENCRYPTION_KEY was unset -- so the test suite validated the insecure
# fallback path and reported it working.
@pytest.fixture
def aria_key_128() -> bytes:
    """Return a random 128-bit ARIA encryption key."""
    return secrets.token_bytes(16)


@pytest.fixture
def aria_key_192() -> bytes:
    """Return a random 192-bit ARIA encryption key."""
    return secrets.token_bytes(24)


@pytest.fixture
def aria_key_256() -> bytes:
    """Return a random 256-bit ARIA encryption key."""
    return secrets.token_bytes(32)


@pytest.fixture
def seed_key() -> bytes:
    """Return a random 128-bit symmetric key."""
    return secrets.token_bytes(16)


@pytest.fixture
def sample_iv() -> bytes:
    """Return a random initialization vector."""
    return secrets.token_bytes(16)


@pytest.fixture
def all_zero_key_128() -> bytes:
    """The all-zero key, for tests that assert it is REJECTED."""
    return bytes(16)


# Korean business data fixtures
@pytest.fixture
def valid_business_number() -> str:
    """Return a business registration number that passes the check digit.

    "1234567890" does NOT pass `validate_business_number`; it only passed the
    old length-only check. Callers that now run real validation need a number
    whose check digit is correct.
    """
    return "1010138752"


@pytest.fixture
def invalid_business_number() -> str:
    """Ten digits with a wrong check digit."""
    return "1234567890"


@pytest.fixture
def valid_corporate_number() -> str:
    """Return a Korean corporate registration number (13 digits)."""
    return "1234567890123"


@pytest.fixture
def valid_resident_number() -> str:
    """Return a resident registration number with a correct check digit.

    Synthetic, for validation tests only. Never log or store a real one.
    """
    return "9001011234568"


@pytest.fixture
def invalid_resident_number() -> str:
    """Thirteen digits with a wrong check digit."""
    return "9001011234569"


# Company fixtures
@pytest.fixture
def sample_company() -> dict:
    """Return sample company data."""
    return {
        "company_id": "11111111-1111-1111-1111-111111111111",
        "business_no": "1234567890",
        "company_name": "Test Company",
        "representative": "Hong Gildong",
        "address": "Seoul, Korea",
        "phone": "02-1234-5678",
        "email": "test@example.com",
    }


# Employee fixtures
@pytest.fixture
def sample_employee() -> dict:
    """Return sample employee data."""
    return {
        "employee_id": "emp-001",
        "name": "Kim Cheolsu",
        "resident_no": "900101-1234568",
        "nationality": "KR",
        "hire_date": "2026-01-15",
        "department": "Engineering",
        "position": "Developer",
        "monthly_salary": 3000000,
    }


# Tax invoice fixtures
@pytest.fixture
def sample_tax_invoice() -> dict:
    """Return sample tax invoice data."""
    return {
        "invoice_id": "inv-001",
        "invoice_date": "2026-01-15",
        "invoice_type": "sales",
        "supplier": {
            "business_no": "1234567890",
            "company_name": "Supplier Co",
            "representative": "Hong Gildong",
        },
        "buyer": {
            "business_no": "9876543210",
            "company_name": "Buyer Co",
            "representative": "Kim Cheolsu",
        },
        "items": [
            {
                "description": "Product A",
                "quantity": 10,
                "unit_price": 10000,
                "supply_value": 100000,
                "tax_amount": 10000,
            }
        ],
        "total_supply_value": 100000,
        "total_tax_amount": 10000,
        "total_amount": 110000,
    }


# gRPC fixtures
@pytest.fixture
def grpc_channel():
    """Create a mock gRPC channel for testing."""
    import grpc
    from unittest.mock import MagicMock

    channel = MagicMock(spec=grpc.Channel)
    return channel


# Environment fixtures
@pytest.fixture(autouse=True)
def reset_env():
    """Reset environment variables after each test."""
    original_env = dict(os.environ)
    yield
    os.environ.clear()
    os.environ.update(original_env)


@pytest.fixture
def test_env():
    """Set up test environment variables."""
    os.environ.update({
        "ENV": "test",
        "DEBUG": "true",
        "LOG_LEVEL": "DEBUG",
    })
    return os.environ
