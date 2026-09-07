"""
Base Insurance Provider

Abstract base class for all insurance provider implementations.
Defines the common interface for EDI operations.
"""
import sys
from abc import ABC, abstractmethod
from pathlib import Path
from typing import Dict, Any, Optional, List
from dataclasses import dataclass
from enum import Enum

import structlog

# Make `python-services/` importable so the shared validators and crypto
# helpers resolve. Appending avoids shadowing same-named installed packages.
_PYTHON_SERVICES_ROOT = str(Path(__file__).resolve().parents[2])
if _PYTHON_SERVICES_ROOT not in sys.path:
    sys.path.append(_PYTHON_SERVICES_ROOT)

from shared.utils.validators import (  # noqa: E402
    mask_business_number,
    mask_name,
    mask_resident_number,
    validate_business_number,
    validate_resident_number,
)

# Re-exported for the concrete providers, which import them from here so the
# sys.path bootstrap above runs first. Named in __all__ so a lint autofix cannot
# strip them as "unused".
__all__ = [
    "BaseProvider",
    "ProviderStatus",
    "StatusResult",
    "SubmissionResult",
    "UntrustedResponseError",
    "mask_business_number",
    "mask_name",
    "mask_resident_number",
    "validate_business_number",
    "validate_resident_number",
]


logger = structlog.get_logger(__name__)


class UntrustedResponseError(Exception):
    """Raised when a counterparty response fails signature verification."""


class ProviderStatus(Enum):
    """Provider connection status."""

    UNKNOWN = "unknown"
    AVAILABLE = "available"
    UNAVAILABLE = "unavailable"
    MAINTENANCE = "maintenance"


@dataclass
class SubmissionResult:
    """Result of a submission operation."""

    success: bool
    reference_id: str = ""
    error_code: str = ""
    error_message: str = ""
    raw_response: Optional[bytes] = None

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary."""
        return {
            "success": self.success,
            "reference_id": self.reference_id,
            "error_code": self.error_code,
            "error_message": self.error_message,
        }


@dataclass
class StatusResult:
    """Result of a status query."""

    status: str
    message: str = ""
    processed_at: str = ""
    errors: List[Dict[str, str]] = None

    def __post_init__(self):
        if self.errors is None:
            self.errors = []

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary."""
        return {
            "status": self.status,
            "message": self.message,
            "processed_at": self.processed_at,
            "errors": self.errors,
        }


class BaseProvider(ABC):
    """
    Abstract base class for insurance providers.

    All provider implementations must inherit from this class
    and implement the required abstract methods.
    """

    def __init__(self, name: str, code: str):
        """
        Initialize provider.

        Args:
            name: Provider display name
            code: Provider code for EDI messages
        """
        self.name = name
        self.code = code
        self._status = ProviderStatus.UNKNOWN
        self._client = None

    @property
    def status(self) -> ProviderStatus:
        """Get current provider status."""
        return self._status

    @abstractmethod
    async def connect(self) -> bool:
        """
        Establish connection to provider EDI server.

        Returns:
            True if connection successful
        """
        pass

    @abstractmethod
    async def disconnect(self) -> None:
        """Close connection to provider EDI server."""
        pass

    @abstractmethod
    async def health_check(self) -> bool:
        """
        Check if provider is available.

        Returns:
            True if provider is healthy
        """
        pass

    @abstractmethod
    async def submit_acquisition(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Submit acquisition report (취득신고).

        Args:
            data: Acquisition data including company, employee, and details

        Returns:
            Submission result dictionary
        """
        pass

    @abstractmethod
    async def submit_loss(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Submit loss report (상실신고).

        Args:
            data: Loss data including company, employee, and details

        Returns:
            Submission result dictionary
        """
        pass

    @abstractmethod
    async def submit_change(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Submit change report (변경신고).

        Args:
            data: Change data including company, employee, and details

        Returns:
            Submission result dictionary
        """
        pass

    @abstractmethod
    async def query_status(self, submission_id: str) -> Dict[str, Any]:
        """
        Query submission status.

        Args:
            submission_id: ID of previous submission

        Returns:
            Status result dictionary
        """
        pass

    @abstractmethod
    async def download_result(
        self,
        submission_id: str,
        document_type: str,
    ) -> Dict[str, Any]:
        """
        Download result document.

        Args:
            submission_id: ID of submission
            document_type: Type of document to download

        Returns:
            Download result with content
        """
        pass

    @staticmethod
    def _client_safe_error(exc: Exception) -> str:
        """
        Reduce an internal exception to a message safe to return to a caller.

        Raw `str(e)` has carried file paths, internal hostnames and upstream
        response bodies out to gRPC clients. The detail belongs in the log.

        Args:
            exc: The exception that occurred

        Returns:
            A stable, non-revealing message
        """
        if isinstance(exc, UntrustedResponseError):
            return "공단 응답의 서명을 검증하지 못했습니다"
        if isinstance(exc, (ConnectionError, TimeoutError)):
            return "공단 EDI 서버와 통신하지 못했습니다"
        return "처리 중 오류가 발생했습니다"

    def _is_client_live(self) -> bool:
        """
        Report whether this provider currently holds a usable connection.

        Used by health checks. Reporting health from a cached status enum alone
        made a provider look healthy after its socket had gone away, so an
        orchestrator never restarted it.

        Returns:
            True if a connected client exists
        """
        client = self._client
        return bool(client is not None and getattr(client, "is_connected", False))

    def _require_valid_signature(self, response: Any, signature_valid: bool) -> None:
        """
        Refuse to act on a response whose signature did not verify.

        The protocol layer reports whether the counterparty's signature checked
        out; every call site used to discard that flag, so a forged or tampered
        response was processed exactly like a genuine one -- including one
        claiming a filing succeeded.

        Args:
            response: Parsed EDI response
            signature_valid: Verification result from the protocol layer

        Raises:
            UntrustedResponseError: If the response cannot be trusted
        """
        if signature_valid:
            return

        message_id = getattr(getattr(response, "header", None), "message_id", "")
        logger.error(
            "response_signature_invalid",
            provider=self.code,
            message_id=message_id,
        )
        raise UntrustedResponseError(
            f"{self.code} response signature did not verify; refusing to trust it"
        )

    def _validate_company_data(self, data: Dict[str, Any]) -> List[str]:
        """
        Validate company data.

        Args:
            data: Company data dictionary

        Returns:
            List of validation error messages
        """
        errors = []
        company = data.get("company", {})

        business_no = company.get("business_no", "")
        if not business_no:
            errors.append("사업자등록번호가 누락되었습니다")
        elif not validate_business_number(business_no):
            # Full check-digit validation, not just a length check: a typo that
            # passes a length check is transmitted to the 공단 and rejected there,
            # after the filing deadline has moved.
            errors.append("사업자등록번호가 올바르지 않습니다 (검증번호 불일치)")

        if not company.get("workplace_no"):
            errors.append("사업장관리번호가 누락되었습니다")

        return errors

    def _validate_employee_data(self, data: Dict[str, Any]) -> List[str]:
        """
        Validate employee data.

        Args:
            data: Employee data dictionary

        Returns:
            List of validation error messages
        """
        errors = []
        employee = data.get("employee", {})

        if not employee.get("name"):
            errors.append("직원 이름이 누락되었습니다")

        resident_no = employee.get("resident_no", "")
        if not resident_no:
            errors.append("주민등록번호가 누락되었습니다")
        elif len(str(resident_no).replace("-", "")) != 13:
            errors.append("주민등록번호는 13자리여야 합니다")
        elif not validate_resident_number(resident_no):
            # Check-digit validation catches transposed digits before they reach
            # the 공단. The error message deliberately carries no digits.
            errors.append("주민등록번호가 올바르지 않습니다 (검증번호 불일치)")

        return errors

    def _format_date(self, date_str: str) -> str:
        """
        Format date string to YYYYMMDD.

        Args:
            date_str: Input date string

        Returns:
            Formatted date string
        """
        # Remove any separators
        clean = date_str.replace("-", "").replace("/", "").replace(".", "")
        return clean[:8]

    def _format_amount(self, amount: int) -> str:
        """
        Format amount for EDI message.

        Args:
            amount: Amount in KRW

        Returns:
            Formatted amount string (15 digits, right-aligned)
        """
        return str(amount).zfill(15)

    def _parse_response_code(self, code: str) -> tuple[bool, str]:
        """
        Parse response code from EDI response.

        Args:
            code: Response code

        Returns:
            Tuple of (success, message)
        """
        success_codes = {"0000", "00", "0"}

        if code in success_codes:
            return True, "Success"

        # Common error codes
        error_messages = {
            "1001": "잘못된 요청 형식",
            "1002": "인증 실패",
            "2001": "중복 신고",
            "2002": "해당 자료 없음",
            "3001": "시스템 오류",
            "9999": "알 수 없는 오류",
        }

        return False, error_messages.get(code, f"오류 코드: {code}")
