"""
Monetary amount parsing for scraped Hometax values.

Amounts are `Decimal`, never `float`: 공급가액/세액/합계 feed 부가가치세 신고 and
double-entry postings, where binary floating point rounding is a defect.

Parsing failures raise. The previous helper returned 0.0 on any ValueError, so a
changed thousands separator, a spacing change, or a 음수 표기 recorded the amount
as zero and left no signal anywhere.
"""
import re
from decimal import Decimal, InvalidOperation

# Characters Hometax uses that carry no numeric meaning.
_STRIP_PATTERN = re.compile(r"[,\s원₩]")

# Korean accounting notation for a negative value.
_NEGATIVE_MARKERS = ("△", "▲", "-", "(")


class AmountParseError(ValueError):
    """Raised when a scraped amount cannot be interpreted."""


def parse_amount(text: str) -> Decimal:
    """
    Parse a scraped monetary string into a Decimal.

    Handles thousands separators, a trailing 원/₩, and the 음수 markers used in
    Korean accounting tables (△, ▲, parentheses).

    Args:
        text: Raw cell text, e.g. "1,000,000", "1,000,000 원", "△500,000"

    Returns:
        The amount as a Decimal

    Raises:
        AmountParseError: If the text does not represent a number. An amount we
            cannot read must stop the record, not become zero.
    """
    if text is None:
        raise AmountParseError("Amount is missing")

    raw = str(text).strip()
    if not raw:
        raise AmountParseError("Amount is empty")

    negative = raw[0] in _NEGATIVE_MARKERS or (raw.startswith("(") and raw.endswith(")"))

    cleaned = _STRIP_PATTERN.sub("", raw)
    cleaned = cleaned.lstrip("△▲()-+").rstrip(")")

    if not cleaned:
        raise AmountParseError(f"Amount {raw!r} contains no digits")

    try:
        value = Decimal(cleaned)
    except InvalidOperation as exc:
        raise AmountParseError(f"Amount {raw!r} is not a number") from exc

    return -value if negative else value
