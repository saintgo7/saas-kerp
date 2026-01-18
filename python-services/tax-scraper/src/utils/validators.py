"""
Input validation utilities for tax scraper service.

Provides validation for Korean business numbers, dates, and other inputs.
"""

import re
from datetime import date, datetime
from typing import Optional, Tuple

import structlog

logger = structlog.get_logger()


def validate_business_number(brn: str) -> Tuple[bool, Optional[str]]:
    """
    Validate Korean business registration number (사업자등록번호).

    Korean business numbers are 10 digits with a check digit algorithm.

    Format: XXX-XX-XXXXX (10 digits)

    Args:
        brn: Business registration number (with or without hyphens)

    Returns:
        Tuple of (is_valid, error_message)
    """
    # Remove hyphens and whitespace
    cleaned = re.sub(r"[-\s]", "", brn)

    # Check length
    if len(cleaned) != 10:
        return False, "Business number must be 10 digits"

    # Check if all digits
    if not cleaned.isdigit():
        return False, "Business number must contain only digits"

    # Validate check digit (modulus 10 algorithm)
    weights = [1, 3, 7, 1, 3, 7, 1, 3, 5]
    checksum = 0

    for i in range(9):
        checksum += int(cleaned[i]) * weights[i]

    # Add special handling for the 9th position
    checksum += (int(cleaned[8]) * 5) // 10

    # Calculate check digit
    remainder = checksum % 10
    expected_check = (10 - remainder) % 10

    if int(cleaned[9]) != expected_check:
        return False, "Invalid business number (checksum failed)"

    return True, None


def validate_date_range(
    start_date: str,
    end_date: str,
    max_days: int = 366,
) -> Tuple[bool, Optional[str], Optional[date], Optional[date]]:
    """
    Validate date range for search queries.

    Args:
        start_date: Start date string (YYYY-MM-DD or YYYYMMDD)
        end_date: End date string (YYYY-MM-DD or YYYYMMDD)
        max_days: Maximum allowed range in days

    Returns:
        Tuple of (is_valid, error_message, parsed_start, parsed_end)
    """
    try:
        # Parse start date
        start = parse_date(start_date)
        if not start:
            return False, "Invalid start date format", None, None

        # Parse end date
        end = parse_date(end_date)
        if not end:
            return False, "Invalid end date format", None, None

        # Check order
        if start > end:
            return False, "Start date must be before or equal to end date", None, None

        # Check range
        delta = (end - start).days
        if delta > max_days:
            return False, f"Date range exceeds maximum of {max_days} days", None, None

        # Check not in future
        today = date.today()
        if end > today:
            return False, "End date cannot be in the future", None, None

        return True, None, start, end

    except Exception as e:
        logger.error("date_validation_error", error=str(e))
        return False, f"Date validation error: {str(e)}", None, None


def parse_date(date_str: str) -> Optional[date]:
    """
    Parse date string in various formats.

    Supported formats:
    - YYYY-MM-DD
    - YYYYMMDD
    - YYYY/MM/DD

    Args:
        date_str: Date string to parse

    Returns:
        Parsed date or None if invalid
    """
    if not date_str:
        return None

    # Remove whitespace
    date_str = date_str.strip()

    # Try different formats
    formats = [
        "%Y-%m-%d",
        "%Y%m%d",
        "%Y/%m/%d",
    ]

    for fmt in formats:
        try:
            return datetime.strptime(date_str, fmt).date()
        except ValueError:
            continue

    return None


def format_business_number(brn: str) -> str:
    """
    Format business number with hyphens.

    Args:
        brn: Business registration number (10 digits)

    Returns:
        Formatted string (XXX-XX-XXXXX)
    """
    cleaned = re.sub(r"[-\s]", "", brn)
    if len(cleaned) != 10:
        return brn

    return f"{cleaned[:3]}-{cleaned[3:5]}-{cleaned[5:]}"


def mask_business_number(brn: str) -> str:
    """
    Mask business number for logging (show first 6 digits only).

    Args:
        brn: Business registration number

    Returns:
        Masked string (XXX-XX-****)
    """
    cleaned = re.sub(r"[-\s]", "", brn)
    if len(cleaned) < 6:
        return "****"

    return f"{cleaned[:3]}-{cleaned[3:5]}-****"


def validate_invoice_number(invoice_number: str) -> Tuple[bool, Optional[str]]:
    """
    Validate tax invoice number format.

    Korean electronic tax invoice numbers follow the format:
    YYYYMMDD-XXXXXXXX (date-serial)

    Args:
        invoice_number: Invoice number to validate

    Returns:
        Tuple of (is_valid, error_message)
    """
    if not invoice_number:
        return False, "Invoice number is required"

    # Check length (typically 17 characters with hyphen)
    if len(invoice_number) < 10:
        return False, "Invoice number too short"

    # Check format pattern
    pattern = r"^\d{8}-\d{8}$|^\d{16}$"
    if not re.match(pattern, invoice_number):
        # Allow more flexible formats
        if not re.match(r"^[\d\-]+$", invoice_number):
            return False, "Invalid invoice number format"

    return True, None


def validate_amount(amount: int) -> Tuple[bool, Optional[str]]:
    """
    Validate monetary amount.

    Args:
        amount: Amount in KRW (integer)

    Returns:
        Tuple of (is_valid, error_message)
    """
    if amount < 0:
        return False, "Amount cannot be negative"

    # Maximum single invoice amount (10 billion KRW)
    if amount > 10_000_000_000:
        return False, "Amount exceeds maximum allowed"

    return True, None


def calculate_vat(supply_amount: int, tax_rate: float = 0.10) -> int:
    """
    Calculate VAT amount from supply amount.

    Args:
        supply_amount: Supply amount before tax
        tax_rate: Tax rate (default 10%)

    Returns:
        VAT amount (rounded down)
    """
    return int(supply_amount * tax_rate)
