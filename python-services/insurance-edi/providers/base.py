"""
Base Insurance Provider

Abstract base class for all insurance provider implementations.
Defines the common interface for EDI operations.
"""
from abc import ABC, abstractmethod
from typing import Dict, Any, Optional, List
from dataclasses import dataclass
from enum import Enum

import structlog


logger = structlog.get_logger(__name__)


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

        if not company.get("business_no"):
            errors.append("사업자등록번호가 누락되었습니다")
        elif len(company.get("business_no", "")) != 10:
            errors.append("사업자등록번호는 10자리여야 합니다")

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

        if not employee.get("resident_no"):
            errors.append("주민등록번호가 누락되었습니다")
        elif len(employee.get("resident_no", "").replace("-", "")) != 13:
            errors.append("주민등록번호는 13자리여야 합니다")

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
