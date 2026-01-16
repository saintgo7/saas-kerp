"""
Korean date format utilities.
"""
from datetime import datetime, date
from typing import Union
import re


def parse_korean_date(date_str: str) -> date:
    """
    Parse various Korean date formats.

    Supported formats:
    - YYYY-MM-DD
    - YYYY/MM/DD
    - YYYY.MM.DD
    - YYYYMMDD
    - YYYY년 MM월 DD일

    Args:
        date_str: Date string in Korean format

    Returns:
        Parsed date object

    Raises:
        ValueError: If date format is not recognized
    """
    # Remove extra whitespace
    date_str = date_str.strip()

    # Try standard formats
    patterns = [
        (r"^(\d{4})-(\d{1,2})-(\d{1,2})$", "%Y-%m-%d"),
        (r"^(\d{4})/(\d{1,2})/(\d{1,2})$", "%Y/%m/%d"),
        (r"^(\d{4})\.(\d{1,2})\.(\d{1,2})$", "%Y.%m.%d"),
        (r"^(\d{8})$", "%Y%m%d"),
    ]

    for pattern, fmt in patterns:
        if re.match(pattern, date_str):
            try:
                return datetime.strptime(date_str, fmt).date()
            except ValueError:
                continue

    # Try Korean format (YYYY년 MM월 DD일)
    korean_pattern = r"^(\d{4})년\s*(\d{1,2})월\s*(\d{1,2})일$"
    match = re.match(korean_pattern, date_str)
    if match:
        year, month, day = map(int, match.groups())
        return date(year, month, day)

    raise ValueError(f"Unrecognized date format: {date_str}")


def format_korean_date(
    d: Union[date, datetime],
    fmt: str = "standard",
) -> str:
    """
    Format date in various Korean styles.

    Args:
        d: Date or datetime object
        fmt: Format style
            - "standard": YYYY-MM-DD
            - "compact": YYYYMMDD
            - "korean": YYYY년 MM월 DD일
            - "slash": YYYY/MM/DD
            - "dot": YYYY.MM.DD

    Returns:
        Formatted date string
    """
    if isinstance(d, datetime):
        d = d.date()

    formats = {
        "standard": d.strftime("%Y-%m-%d"),
        "compact": d.strftime("%Y%m%d"),
        "korean": f"{d.year}년 {d.month:02d}월 {d.day:02d}일",
        "slash": d.strftime("%Y/%m/%d"),
        "dot": d.strftime("%Y.%m.%d"),
    }

    if fmt not in formats:
        raise ValueError(f"Unknown format: {fmt}. Use one of: {list(formats.keys())}")

    return formats[fmt]


def get_korean_fiscal_year(d: Union[date, datetime]) -> int:
    """
    Get Korean fiscal year for a date.

    Korean fiscal year typically aligns with calendar year.

    Args:
        d: Date or datetime

    Returns:
        Fiscal year (YYYY)
    """
    if isinstance(d, datetime):
        d = d.date()
    return d.year


def get_korean_fiscal_quarter(d: Union[date, datetime]) -> int:
    """
    Get Korean fiscal quarter for a date.

    Args:
        d: Date or datetime

    Returns:
        Quarter number (1-4)
    """
    if isinstance(d, datetime):
        d = d.date()
    return (d.month - 1) // 3 + 1


def get_vat_period(d: Union[date, datetime]) -> tuple[int, int]:
    """
    Get Korean VAT reporting period for a date.

    VAT is reported semi-annually:
    - Period 1: January - June
    - Period 2: July - December

    Args:
        d: Date or datetime

    Returns:
        Tuple of (year, period) where period is 1 or 2
    """
    if isinstance(d, datetime):
        d = d.date()

    period = 1 if d.month <= 6 else 2
    return d.year, period
