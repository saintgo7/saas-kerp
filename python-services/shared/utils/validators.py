"""
Korean business document validators and PII masking helpers.
"""
import re
from typing import Any


def mask_resident_number(number: str) -> str:
    """
    Mask a Korean resident registration number for logging.

    주민등록번호 is 고유식별정보 under the 개인정보 보호법: it must not appear in
    plaintext in application logs, which routinely ship to collectors outside
    the system's access controls.

    Keeps the leading 6 digits (birth date) and masks the trailing 7.

    Args:
        number: Resident registration number, with or without a dash

    Returns:
        Masked form such as "880101-1******", or "***" if the input does not
        look like a resident number at all
    """
    if not number:
        return ""

    cleaned = re.sub(r"[-\s]", "", str(number))

    if len(cleaned) != 13 or not cleaned.isdigit():
        # Never echo an unrecognised value: it may still be sensitive.
        return "***"

    return f"{cleaned[:6]}-{cleaned[6]}{'*' * 6}"


def mask_business_number(number: str) -> str:
    """
    Mask a business registration number for logging.

    Args:
        number: Business registration number, with or without dashes

    Returns:
        Masked form such as "123456****"
    """
    if not number:
        return ""

    cleaned = re.sub(r"[-\s]", "", str(number))
    if len(cleaned) < 6:
        return "***"

    return f"{cleaned[:6]}{'*' * (len(cleaned) - 6)}"


def mask_name(name: str) -> str:
    """
    Mask a person's name for logging, keeping only the first character.

    Args:
        name: Person name

    Returns:
        Masked form such as "홍**"
    """
    if not name:
        return ""

    text = str(name)
    if len(text) == 1:
        return "*"

    return f"{text[0]}{'*' * (len(text) - 1)}"


# Keys whose values must never be written to logs in plaintext.
SENSITIVE_LOG_KEYS = frozenset(
    {
        "resident_no",
        "resident_number",
        "rrn",
        "주민등록번호",
        "cert_password",
        "password",
        "user_password",
        "secret_key",
        "access_token",
        "private_key",
        "aria_key",
        "seed_key",
        "encryption_key",
    }
)


def scrub_sensitive(value: Any, _depth: int = 0) -> Any:
    """
    Recursively redact sensitive keys from a structure destined for logs.

    Intended both as an explicit call site helper and as the core of a structlog
    processor, so that a future `logger.info(..., data=payload)` cannot leak a
    주민등록번호 the way the provider submit paths previously did.

    Args:
        value: Any JSON-like structure
        _depth: Internal recursion guard

    Returns:
        A copy with sensitive values replaced by masked forms
    """
    if _depth > 8:
        return "***"

    if isinstance(value, dict):
        scrubbed = {}
        for key, item in value.items():
            lowered = str(key).lower()
            if lowered in SENSITIVE_LOG_KEYS:
                if lowered in ("resident_no", "resident_number", "rrn", "주민등록번호"):
                    scrubbed[key] = mask_resident_number(item)
                else:
                    scrubbed[key] = "***"
            elif lowered == "name":
                scrubbed[key] = mask_name(item) if isinstance(item, str) else item
            else:
                scrubbed[key] = scrub_sensitive(item, _depth + 1)
        return scrubbed

    if isinstance(value, (list, tuple)):
        return [scrub_sensitive(item, _depth + 1) for item in value]

    return value


def scrub_log_processor(_logger: Any, _method_name: str, event_dict: dict) -> dict:
    """
    structlog processor that scrubs sensitive keys from every log event.

    Install it ahead of the renderer so a stray `data=` keyword cannot put a
    주민등록번호 into the log stream.

    Args:
        _logger: Unused (structlog processor signature)
        _method_name: Unused (structlog processor signature)
        event_dict: Structured log event

    Returns:
        The event dict with sensitive values masked
    """
    return scrub_sensitive(event_dict)


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
