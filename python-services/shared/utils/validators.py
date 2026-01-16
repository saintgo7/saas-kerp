"""
Korean business document validators.
"""
import re


def validate_business_number(number: str) -> bool:
    """
    Validate Korean business registration number (사업자등록번호).

    Korean business numbers are 10 digits with a check digit algorithm.
    Format: XXX-XX-XXXXX

    Args:
        number: Business registration number (with or without dashes)

    Returns:
        True if valid, False otherwise
    """
    # Remove dashes and whitespace
    cleaned = re.sub(r"[-\s]", "", number)

    # Must be exactly 10 digits
    if not cleaned.isdigit() or len(cleaned) != 10:
        return False

    # Check digit validation algorithm
    weights = [1, 3, 7, 1, 3, 7, 1, 3, 5]
    digits = [int(d) for d in cleaned]

    checksum = sum(w * d for w, d in zip(weights, digits[:9]))
    checksum += (weights[8] * digits[8]) // 10
    remainder = checksum % 10
    check_digit = (10 - remainder) % 10

    return check_digit == digits[9]


def format_business_number(number: str) -> str:
    """
    Format business number with dashes.

    Args:
        number: Business registration number (10 digits)

    Returns:
        Formatted number (XXX-XX-XXXXX)

    Raises:
        ValueError: If number is not valid
    """
    cleaned = re.sub(r"[-\s]", "", number)

    if not validate_business_number(cleaned):
        raise ValueError(f"Invalid business number: {number}")

    return f"{cleaned[:3]}-{cleaned[3:5]}-{cleaned[5:]}"


def validate_resident_number(number: str) -> bool:
    """
    Validate Korean resident registration number (주민등록번호).

    WARNING: Only use for validation. Never store full resident numbers.

    Args:
        number: Resident registration number (with or without dash)

    Returns:
        True if valid, False otherwise
    """
    # Remove dash and whitespace
    cleaned = re.sub(r"[-\s]", "", number)

    # Must be exactly 13 digits
    if not cleaned.isdigit() or len(cleaned) != 13:
        return False

    # Check digit validation
    weights = [2, 3, 4, 5, 6, 7, 8, 9, 2, 3, 4, 5]
    digits = [int(d) for d in cleaned]

    checksum = sum(w * d for w, d in zip(weights, digits[:12]))
    check_digit = (11 - (checksum % 11)) % 10

    return check_digit == digits[12]


def validate_corporate_number(number: str) -> bool:
    """
    Validate Korean corporate registration number (법인등록번호).

    Args:
        number: Corporate registration number (13 digits with dash)

    Returns:
        True if valid, False otherwise
    """
    # Remove dash and whitespace
    cleaned = re.sub(r"[-\s]", "", number)

    # Must be exactly 13 digits
    if not cleaned.isdigit() or len(cleaned) != 13:
        return False

    # Check digit validation
    weights = [1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2]
    digits = [int(d) for d in cleaned]

    checksum = 0
    for w, d in zip(weights, digits[:12]):
        product = w * d
        checksum += product // 10 + product % 10

    check_digit = (10 - (checksum % 10)) % 10

    return check_digit == digits[12]
