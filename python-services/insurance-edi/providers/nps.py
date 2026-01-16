"""
NPS (National Pension Service) Provider - 국민연금공단

Handles EDI communication with the National Pension Service for:
- 취득신고 (Acquisition)
- 상실신고 (Loss)
- 내용변경 (Change)
- 월별납부내역 (Monthly report)
"""
import uuid
from typing import Dict, Any, Optional
from datetime import datetime

import structlog

from .base import BaseProvider, ProviderStatus, SubmissionResult, StatusResult
from edi.client import create_nps_client, EDIClient, ConnectionConfig
from edi.message import (
    EDIMessage,
    InsuranceType,
    DocumentType,
    MessageType,
)
from edi.protocol import EDIProtocolFactory
from config import settings


logger = structlog.get_logger(__name__)


class NPSProvider(BaseProvider):
    """
    National Pension Service (국민연금공단) EDI provider.

    Document codes:
    - 1001: 취득신고
    - 1002: 상실신고
    - 1003: 내용변경
    - 1004: 월별납부내역
    """

    def __init__(self):
        """Initialize NPS provider."""
        super().__init__(name="국민연금공단", code="NPS")
        self._host = settings.edi.nps_host
        self._port = settings.edi.nps_port
        self._timeout = settings.edi.nps_timeout

    async def connect(self) -> bool:
        """Establish connection to NPS EDI server."""
        try:
            encryption_key = self._get_encryption_key()
            self._client = create_nps_client(
                encryption_key=encryption_key,
                host=self._host,
                port=self._port,
                timeout=self._timeout,
            )
            await self._client.connect()
            self._status = ProviderStatus.AVAILABLE
            logger.info("Connected to NPS EDI server")
            return True
        except Exception as e:
            logger.error("Failed to connect to NPS", error=str(e))
            self._status = ProviderStatus.UNAVAILABLE
            return False

    async def disconnect(self) -> None:
        """Close connection to NPS EDI server."""
        if self._client:
            await self._client.disconnect()
            self._client = None
        self._status = ProviderStatus.UNKNOWN

    async def health_check(self) -> bool:
        """Check NPS provider availability."""
        try:
            # In production, this would send a test message
            # For now, just check if we can create a client
            return self._status == ProviderStatus.AVAILABLE
        except Exception:
            return False

    def _get_encryption_key(self) -> bytes:
        """Get ARIA encryption key for NPS."""
        key_hex = settings.crypto.aria_key
        if key_hex:
            return bytes.fromhex(key_hex)
        # Return placeholder for development
        return bytes(16)

    async def submit_acquisition(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Submit acquisition report to NPS (국민연금 취득신고).

        Required data:
        - company.business_no: 사업자등록번호
        - company.workplace_no: 사업장관리번호
        - employee.name: 성명
        - employee.resident_no: 주민등록번호
        - acquisition.date: 취득일
        - acquisition.monthly_income: 기준소득월액
        """
        logger.info("Submitting NPS acquisition", data=data)

        # Validate data
        errors = self._validate_company_data(data) + self._validate_employee_data(data)
        if errors:
            return SubmissionResult(
                success=False,
                error_code="VALIDATION_ERROR",
                error_message="; ".join(errors),
            ).to_dict()

        # Build EDI message
        company = data.get("company", {})
        employee = data.get("employee", {})
        acq = data.get("acquisition", {})

        # Format records for NPS acquisition
        records = [{
            "record_type": "D",  # Detail record
            "resident_no": employee.get("resident_no", "").replace("-", ""),
            "name": employee.get("name", ""),
            "acquisition_date": self._format_date(acq.get("date", "")),
            "monthly_income": self._format_amount(acq.get("monthly_income", 0)),
            "job_type": acq.get("job_type", "01"),  # Default: 일반
            "nationality": employee.get("nationality", "KR"),
        }]

        message = EDIMessage.create_submit_message(
            sender_id=company.get("workplace_no", ""),
            insurance_type=InsuranceType.NPS,
            document_type=DocumentType.NPS_ACQUISITION,
            records=records,
            company_id=company.get("workplace_no", ""),
            business_no=company.get("business_no", ""),
        )

        # Submit message
        try:
            if not self._client:
                await self.connect()

            response, sig_valid = await self._client.send_with_retry(message)

            # Parse response
            success, msg = self._parse_response_code(response.response_code)

            return SubmissionResult(
                success=success,
                reference_id=response.header.message_id,
                error_code="" if success else response.response_code,
                error_message="" if success else msg,
            ).to_dict()

        except Exception as e:
            logger.exception("NPS acquisition submission failed", error=str(e))
            return SubmissionResult(
                success=False,
                error_code="SUBMISSION_ERROR",
                error_message=str(e),
            ).to_dict()

    async def submit_loss(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Submit loss report to NPS (국민연금 상실신고).

        Required data:
        - company.business_no: 사업자등록번호
        - company.workplace_no: 사업장관리번호
        - employee.name: 성명
        - employee.resident_no: 주민등록번호
        - loss.date: 상실일
        - loss.reason_code: 상실사유코드
        """
        logger.info("Submitting NPS loss", data=data)

        errors = self._validate_company_data(data) + self._validate_employee_data(data)
        if errors:
            return SubmissionResult(
                success=False,
                error_code="VALIDATION_ERROR",
                error_message="; ".join(errors),
            ).to_dict()

        company = data.get("company", {})
        employee = data.get("employee", {})
        loss = data.get("loss", {})

        records = [{
            "record_type": "D",
            "resident_no": employee.get("resident_no", "").replace("-", ""),
            "name": employee.get("name", ""),
            "loss_date": self._format_date(loss.get("date", "")),
            "loss_reason": loss.get("reason_code", "11"),  # Default: 퇴직
            "final_income": self._format_amount(loss.get("final_income", 0)),
        }]

        message = EDIMessage.create_submit_message(
            sender_id=company.get("workplace_no", ""),
            insurance_type=InsuranceType.NPS,
            document_type=DocumentType.NPS_LOSS,
            records=records,
            company_id=company.get("workplace_no", ""),
            business_no=company.get("business_no", ""),
        )

        try:
            if not self._client:
                await self.connect()

            response, sig_valid = await self._client.send_with_retry(message)
            success, msg = self._parse_response_code(response.response_code)

            return SubmissionResult(
                success=success,
                reference_id=response.header.message_id,
                error_code="" if success else response.response_code,
                error_message="" if success else msg,
            ).to_dict()

        except Exception as e:
            logger.exception("NPS loss submission failed", error=str(e))
            return SubmissionResult(
                success=False,
                error_code="SUBMISSION_ERROR",
                error_message=str(e),
            ).to_dict()

    async def submit_change(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Submit change report to NPS (국민연금 내용변경).

        Supported change types:
        - 01: 기준소득월액 변경
        - 02: 성명 변경
        - 03: 주민등록번호 정정
        """
        logger.info("Submitting NPS change", data=data)

        errors = self._validate_company_data(data) + self._validate_employee_data(data)
        if errors:
            return SubmissionResult(
                success=False,
                error_code="VALIDATION_ERROR",
                error_message="; ".join(errors),
            ).to_dict()

        company = data.get("company", {})
        employee = data.get("employee", {})
        change = data.get("change", {})

        records = [{
            "record_type": "D",
            "resident_no": employee.get("resident_no", "").replace("-", ""),
            "name": employee.get("name", ""),
            "change_date": self._format_date(change.get("date", "")),
            "change_type": change.get("type", "01"),
            "before_value": change.get("before", ""),
            "after_value": change.get("after", ""),
            "reason": change.get("reason", ""),
        }]

        message = EDIMessage.create_submit_message(
            sender_id=company.get("workplace_no", ""),
            insurance_type=InsuranceType.NPS,
            document_type=DocumentType.NPS_CHANGE,
            records=records,
            company_id=company.get("workplace_no", ""),
            business_no=company.get("business_no", ""),
        )

        try:
            if not self._client:
                await self.connect()

            response, sig_valid = await self._client.send_with_retry(message)
            success, msg = self._parse_response_code(response.response_code)

            return SubmissionResult(
                success=success,
                reference_id=response.header.message_id,
                error_code="" if success else response.response_code,
                error_message="" if success else msg,
            ).to_dict()

        except Exception as e:
            logger.exception("NPS change submission failed", error=str(e))
            return SubmissionResult(
                success=False,
                error_code="SUBMISSION_ERROR",
                error_message=str(e),
            ).to_dict()

    async def query_status(self, submission_id: str) -> Dict[str, Any]:
        """Query submission status from NPS."""
        logger.info("Querying NPS status", submission_id=submission_id)

        try:
            if not self._client:
                await self.connect()

            message = EDIMessage.create_query_message(
                sender_id="",  # Will be set from context
                insurance_type=InsuranceType.NPS,
                reference_id=submission_id,
            )

            response, _ = await self._client.send_with_retry(message)

            # Parse status from response
            status_map = {
                "0": "completed",
                "1": "processing",
                "2": "pending",
                "9": "rejected",
            }

            return StatusResult(
                status=status_map.get(response.response_code[:1], "error"),
                message=response.response_message,
                processed_at=datetime.now().isoformat() if response.response_code == "0" else "",
            ).to_dict()

        except Exception as e:
            logger.exception("NPS status query failed", error=str(e))
            return StatusResult(
                status="error",
                message=str(e),
            ).to_dict()

    async def download_result(
        self,
        submission_id: str,
        document_type: str,
    ) -> Dict[str, Any]:
        """Download result document from NPS."""
        logger.info(
            "Downloading NPS result",
            submission_id=submission_id,
            document_type=document_type,
        )

        try:
            if not self._client:
                await self.connect()

            # Create download request message
            message = EDIMessage(
                header=EDIMessage.create_query_message(
                    sender_id="",
                    insurance_type=InsuranceType.NPS,
                    reference_id=submission_id,
                ).header,
            )
            message.header.message_type = MessageType.REQUEST_DOWNLOAD

            response, _ = await self._client.send_with_retry(message)

            if response.response_data:
                return {
                    "success": True,
                    "filename": f"nps_result_{submission_id}.pdf",
                    "content": response.response_data,
                    "content_type": "application/pdf",
                }

            return {
                "success": False,
                "filename": "",
                "content": b"",
                "content_type": "",
            }

        except Exception as e:
            logger.exception("NPS download failed", error=str(e))
            return {
                "success": False,
                "filename": "",
                "content": b"",
                "content_type": "",
            }
