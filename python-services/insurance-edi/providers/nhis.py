"""
NHIS (National Health Insurance Service) Provider - 건강보험공단

Handles EDI communication with the National Health Insurance Service for:
- 취득신고 (Acquisition)
- 상실신고 (Loss)
- 보수월액변경 (Salary change)
- 피부양자신고 (Dependent report)
"""
from typing import Dict, Any
from datetime import datetime

import structlog

from .base import BaseProvider, ProviderStatus, SubmissionResult, StatusResult
from edi.client import create_nhis_client
from edi.message import (
    EDIMessage,
    InsuranceType,
    DocumentType,
    MessageType,
)
from config import settings


logger = structlog.get_logger(__name__)


class NHISProvider(BaseProvider):
    """
    National Health Insurance Service (건강보험공단) EDI provider.

    Document codes:
    - 2001: 취득신고
    - 2002: 상실신고
    - 2003: 보수월액변경
    - 2004: 피부양자신고
    """

    def __init__(self):
        """Initialize NHIS provider."""
        super().__init__(name="건강보험공단", code="NHIS")
        self._host = settings.edi.nhis_host
        self._port = settings.edi.nhis_port
        self._timeout = settings.edi.nhis_timeout

    async def connect(self) -> bool:
        """Establish connection to NHIS EDI server."""
        try:
            encryption_key = self._get_encryption_key()
            self._client = create_nhis_client(
                encryption_key=encryption_key,
                host=self._host,
                port=self._port,
                timeout=self._timeout,
            )
            await self._client.connect()
            self._status = ProviderStatus.AVAILABLE
            logger.info("Connected to NHIS EDI server")
            return True
        except Exception as e:
            logger.error("Failed to connect to NHIS", error=str(e))
            self._status = ProviderStatus.UNAVAILABLE
            return False

    async def disconnect(self) -> None:
        """Close connection to NHIS EDI server."""
        if self._client:
            await self._client.disconnect()
            self._client = None
        self._status = ProviderStatus.UNKNOWN

    async def health_check(self) -> bool:
        """Check NHIS provider availability."""
        return self._status == ProviderStatus.AVAILABLE

    def _get_encryption_key(self) -> bytes:
        """Get ARIA encryption key for NHIS."""
        key_hex = settings.crypto.aria_key
        if key_hex:
            return bytes.fromhex(key_hex)
        return bytes(16)

    async def submit_acquisition(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Submit acquisition report to NHIS (건강보험 취득신고).

        Required data:
        - company.business_no: 사업자등록번호
        - company.workplace_no: 사업장관리번호
        - employee.name: 성명
        - employee.resident_no: 주민등록번호
        - acquisition.date: 취득일
        - acquisition.monthly_income: 보수월액
        """
        logger.info("Submitting NHIS acquisition", data=data)

        errors = self._validate_company_data(data) + self._validate_employee_data(data)
        if errors:
            return SubmissionResult(
                success=False,
                error_code="VALIDATION_ERROR",
                error_message="; ".join(errors),
            ).to_dict()

        company = data.get("company", {})
        employee = data.get("employee", {})
        acq = data.get("acquisition", {})

        # NHIS specific: Include work hours for part-time workers
        work_hours = acq.get("work_hours", 40)
        is_part_time = work_hours < 40

        records = [{
            "record_type": "D",
            "resident_no": employee.get("resident_no", "").replace("-", ""),
            "name": employee.get("name", ""),
            "acquisition_date": self._format_date(acq.get("date", "")),
            "monthly_salary": self._format_amount(acq.get("monthly_income", 0)),
            "work_hours_weekly": str(work_hours).zfill(2),
            "is_part_time": "Y" if is_part_time else "N",
            "contract_type": acq.get("contract_type", "1"),  # 1: 정규직
        }]

        message = EDIMessage.create_submit_message(
            sender_id=company.get("workplace_no", ""),
            insurance_type=InsuranceType.NHIS,
            document_type=DocumentType.NHIS_ACQUISITION,
            records=records,
            company_id=company.get("workplace_no", ""),
            business_no=company.get("business_no", ""),
        )

        try:
            if not self._client:
                await self.connect()

            response, _ = await self._client.send_with_retry(message)
            success, msg = self._parse_response_code(response.response_code)

            return SubmissionResult(
                success=success,
                reference_id=response.header.message_id,
                error_code="" if success else response.response_code,
                error_message="" if success else msg,
            ).to_dict()

        except Exception as e:
            logger.exception("NHIS acquisition submission failed", error=str(e))
            return SubmissionResult(
                success=False,
                error_code="SUBMISSION_ERROR",
                error_message=str(e),
            ).to_dict()

    async def submit_loss(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Submit loss report to NHIS (건강보험 상실신고).

        NHIS loss reasons:
        - 11: 퇴직
        - 12: 사망
        - 21: 지역가입자 전환
        - 31: 타사업장 취득
        """
        logger.info("Submitting NHIS loss", data=data)

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
            "loss_reason": loss.get("reason_code", "11"),
            "final_salary": self._format_amount(loss.get("final_income", 0)),
        }]

        message = EDIMessage.create_submit_message(
            sender_id=company.get("workplace_no", ""),
            insurance_type=InsuranceType.NHIS,
            document_type=DocumentType.NHIS_LOSS,
            records=records,
            company_id=company.get("workplace_no", ""),
            business_no=company.get("business_no", ""),
        )

        try:
            if not self._client:
                await self.connect()

            response, _ = await self._client.send_with_retry(message)
            success, msg = self._parse_response_code(response.response_code)

            return SubmissionResult(
                success=success,
                reference_id=response.header.message_id,
                error_code="" if success else response.response_code,
                error_message="" if success else msg,
            ).to_dict()

        except Exception as e:
            logger.exception("NHIS loss submission failed", error=str(e))
            return SubmissionResult(
                success=False,
                error_code="SUBMISSION_ERROR",
                error_message=str(e),
            ).to_dict()

    async def submit_change(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Submit change report to NHIS (건강보험 보수월액변경).

        Primary use: Monthly salary (보수월액) changes
        """
        logger.info("Submitting NHIS change", data=data)

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
            "before_salary": change.get("before", "0").zfill(15),
            "after_salary": change.get("after", "0").zfill(15),
            "reason": change.get("reason", ""),
        }]

        message = EDIMessage.create_submit_message(
            sender_id=company.get("workplace_no", ""),
            insurance_type=InsuranceType.NHIS,
            document_type=DocumentType.NHIS_CHANGE,
            records=records,
            company_id=company.get("workplace_no", ""),
            business_no=company.get("business_no", ""),
        )

        try:
            if not self._client:
                await self.connect()

            response, _ = await self._client.send_with_retry(message)
            success, msg = self._parse_response_code(response.response_code)

            return SubmissionResult(
                success=success,
                reference_id=response.header.message_id,
                error_code="" if success else response.response_code,
                error_message="" if success else msg,
            ).to_dict()

        except Exception as e:
            logger.exception("NHIS change submission failed", error=str(e))
            return SubmissionResult(
                success=False,
                error_code="SUBMISSION_ERROR",
                error_message=str(e),
            ).to_dict()

    async def query_status(self, submission_id: str) -> Dict[str, Any]:
        """Query submission status from NHIS."""
        logger.info("Querying NHIS status", submission_id=submission_id)

        try:
            if not self._client:
                await self.connect()

            message = EDIMessage.create_query_message(
                sender_id="",
                insurance_type=InsuranceType.NHIS,
                reference_id=submission_id,
            )

            response, _ = await self._client.send_with_retry(message)

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
            logger.exception("NHIS status query failed", error=str(e))
            return StatusResult(
                status="error",
                message=str(e),
            ).to_dict()

    async def download_result(
        self,
        submission_id: str,
        document_type: str,
    ) -> Dict[str, Any]:
        """Download result document from NHIS."""
        logger.info(
            "Downloading NHIS result",
            submission_id=submission_id,
            document_type=document_type,
        )

        try:
            if not self._client:
                await self.connect()

            message = EDIMessage(
                header=EDIMessage.create_query_message(
                    sender_id="",
                    insurance_type=InsuranceType.NHIS,
                    reference_id=submission_id,
                ).header,
            )
            message.header.message_type = MessageType.REQUEST_DOWNLOAD

            response, _ = await self._client.send_with_retry(message)

            if response.response_data:
                return {
                    "success": True,
                    "filename": f"nhis_result_{submission_id}.pdf",
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
            logger.exception("NHIS download failed", error=str(e))
            return {
                "success": False,
                "filename": "",
                "content": b"",
                "content_type": "",
            }
