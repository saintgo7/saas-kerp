"""
EI/WCI Provider - 고용보험/산재보험 (근로복지공단)

Handles EDI communication with Korea Workers' Compensation & Welfare Service for:
- 고용보험 취득/상실신고
- 산재보험 취득/상실신고
- 이직확인서
- 월별고용정보현황
"""
from typing import Dict, Any
from datetime import datetime

import structlog

from .base import BaseProvider, ProviderStatus, SubmissionResult, StatusResult
from edi.client import create_ei_client
from edi.message import (
    EDIMessage,
    InsuranceType,
    DocumentType,
    MessageType,
)
from config import settings


logger = structlog.get_logger(__name__)


class EIProvider(BaseProvider):
    """
    Employment Insurance / Workers' Compensation Insurance Provider.
    근로복지공단 (Korea Workers' Compensation & Welfare Service)

    Document codes:
    - 3001: 고용보험 취득신고
    - 3002: 고용보험 상실신고
    - 4001: 산재보험 취득신고
    - 4002: 산재보험 상실신고
    """

    def __init__(self):
        """Initialize EI/WCI provider."""
        super().__init__(name="근로복지공단", code="COMWEL")
        self._host = settings.edi.ei_host
        self._port = settings.edi.ei_port
        self._timeout = settings.edi.ei_timeout

    async def connect(self) -> bool:
        """Establish connection to EI/WCI EDI server."""
        try:
            encryption_key = self._get_encryption_key()
            self._client = create_ei_client(
                encryption_key=encryption_key,
                host=self._host,
                port=self._port,
                timeout=self._timeout,
            )
            await self._client.connect()
            self._status = ProviderStatus.AVAILABLE
            logger.info("Connected to EI/WCI EDI server")
            return True
        except Exception as e:
            logger.error("Failed to connect to EI/WCI", error=str(e))
            self._status = ProviderStatus.UNAVAILABLE
            return False

    async def disconnect(self) -> None:
        """Close connection to EI/WCI EDI server."""
        if self._client:
            await self._client.disconnect()
            self._client = None
        self._status = ProviderStatus.UNKNOWN

    async def health_check(self) -> bool:
        """Check EI/WCI provider availability."""
        return self._status == ProviderStatus.AVAILABLE

    def _get_encryption_key(self) -> bytes:
        """Get ARIA encryption key for EI/WCI."""
        key_hex = settings.crypto.aria_key
        if key_hex:
            return bytes.fromhex(key_hex)
        return bytes(16)

    async def submit_acquisition(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Submit acquisition report to EI/WCI (고용/산재보험 취득신고).

        Required data:
        - company.business_no: 사업자등록번호
        - company.workplace_no: 사업장관리번호
        - employee.name: 성명
        - employee.resident_no: 주민등록번호
        - acquisition.date: 취득일
        - acquisition.monthly_income: 월보수액
        - acquisition.work_hours: 주당 근로시간
        - acquisition.contract_type: 계약형태
        """
        logger.info("Submitting EI acquisition", data=data)

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

        # EI specific: Employment type affects eligibility
        work_hours = acq.get("work_hours", 40)
        employment_type = self._determine_employment_type(work_hours, acq)

        records = [{
            "record_type": "D",
            "resident_no": employee.get("resident_no", "").replace("-", ""),
            "name": employee.get("name", ""),
            "acquisition_date": self._format_date(acq.get("date", "")),
            "monthly_income": self._format_amount(acq.get("monthly_income", 0)),
            "work_hours_weekly": str(work_hours).zfill(2),
            "employment_type": employment_type,
            "contract_period": acq.get("contract_period", ""),  # 계약기간 (계약직의 경우)
            "job_code": acq.get("job_type", "000"),  # 직종코드
            "is_foreign_worker": "Y" if acq.get("is_foreign_worker") else "N",
            "visa_type": acq.get("visa_type", "") if acq.get("is_foreign_worker") else "",
        }]

        # Determine document type based on insurance type
        # Default to employment insurance
        doc_type = DocumentType.EI_ACQUISITION

        message = EDIMessage.create_submit_message(
            sender_id=company.get("workplace_no", ""),
            insurance_type=InsuranceType.EI,
            document_type=doc_type,
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
            logger.exception("EI acquisition submission failed", error=str(e))
            return SubmissionResult(
                success=False,
                error_code="SUBMISSION_ERROR",
                error_message=str(e),
            ).to_dict()

    def _determine_employment_type(self, work_hours: int, acq: Dict) -> str:
        """
        Determine employment type code for EI.

        Employment types:
        - 1: 상용직 (Regular, >= 15 hours/week)
        - 2: 일용직 (Daily worker)
        - 3: 자영업자 (Self-employed)
        - 4: 예술인 (Artist)
        - 5: 특수형태근로종사자 (Gig worker)
        """
        contract_type = acq.get("contract_type", "")

        if work_hours < 15:
            return "2"  # Below threshold, likely daily worker

        if contract_type == "self_employed":
            return "3"
        elif contract_type == "artist":
            return "4"
        elif contract_type == "gig":
            return "5"

        return "1"  # Default: regular employee

    async def submit_loss(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Submit loss report to EI/WCI (고용/산재보험 상실신고).

        EI loss is particularly important for unemployment benefits.
        Loss reasons determine benefit eligibility.
        """
        logger.info("Submitting EI loss", data=data)

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

        # Map loss reason for unemployment benefit eligibility
        loss_reason = self._map_loss_reason(
            loss.get("reason_code", ""),
            loss.get("is_voluntary", False),
        )

        records = [{
            "record_type": "D",
            "resident_no": employee.get("resident_no", "").replace("-", ""),
            "name": employee.get("name", ""),
            "loss_date": self._format_date(loss.get("date", "")),
            "loss_reason_code": loss_reason["code"],
            "loss_reason_detail": loss_reason["detail"],
            "final_income": self._format_amount(loss.get("final_income", 0)),
            "total_work_days": str(loss.get("total_work_days", 0)).zfill(4),
            "is_voluntary": "Y" if loss.get("is_voluntary") else "N",
            "benefit_eligible": "Y" if loss_reason["eligible"] else "N",
        }]

        message = EDIMessage.create_submit_message(
            sender_id=company.get("workplace_no", ""),
            insurance_type=InsuranceType.EI,
            document_type=DocumentType.EI_LOSS,
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
            logger.exception("EI loss submission failed", error=str(e))
            return SubmissionResult(
                success=False,
                error_code="SUBMISSION_ERROR",
                error_message=str(e),
            ).to_dict()

    def _map_loss_reason(self, reason_code: str, is_voluntary: bool) -> Dict[str, Any]:
        """
        Map loss reason to EI codes and determine benefit eligibility.

        Involuntary termination (비자발적 이직) is eligible for benefits.
        Voluntary resignation (자발적 이직) may not be eligible.
        """
        # EI loss reason codes
        reason_map = {
            # Involuntary - Eligible
            "11": {"code": "11", "detail": "회사사정 해고", "eligible": True},
            "12": {"code": "12", "detail": "계약기간 만료", "eligible": True},
            "13": {"code": "13", "detail": "정년퇴직", "eligible": True},
            "14": {"code": "14", "detail": "구조조정", "eligible": True},
            "15": {"code": "15", "detail": "사업장 이전", "eligible": True},
            "16": {"code": "16", "detail": "권고사직", "eligible": True},
            # Voluntary - Generally not eligible
            "21": {"code": "21", "detail": "자진퇴사", "eligible": False},
            "22": {"code": "22", "detail": "전직", "eligible": False},
            "23": {"code": "23", "detail": "개인사정", "eligible": False},
            # Special cases - May be eligible
            "31": {"code": "31", "detail": "임금체불로 퇴직", "eligible": True},
            "32": {"code": "32", "detail": "직장 내 괴롭힘", "eligible": True},
            "33": {"code": "33", "detail": "가족 간병", "eligible": True},
        }

        if reason_code in reason_map:
            return reason_map[reason_code]

        # Default based on voluntary flag
        if is_voluntary:
            return {"code": "21", "detail": "자진퇴사", "eligible": False}
        else:
            return {"code": "11", "detail": "회사사정 해고", "eligible": True}

    async def submit_change(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """Submit change report to EI/WCI."""
        logger.info("Submitting EI change", data=data)

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
            insurance_type=InsuranceType.EI,
            document_type=DocumentType.EI_ACQUISITION,  # Use as placeholder
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
            logger.exception("EI change submission failed", error=str(e))
            return SubmissionResult(
                success=False,
                error_code="SUBMISSION_ERROR",
                error_message=str(e),
            ).to_dict()

    async def query_status(self, submission_id: str) -> Dict[str, Any]:
        """Query submission status from EI/WCI."""
        logger.info("Querying EI status", submission_id=submission_id)

        try:
            if not self._client:
                await self.connect()

            message = EDIMessage.create_query_message(
                sender_id="",
                insurance_type=InsuranceType.EI,
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
            logger.exception("EI status query failed", error=str(e))
            return StatusResult(
                status="error",
                message=str(e),
            ).to_dict()

    async def download_result(
        self,
        submission_id: str,
        document_type: str,
    ) -> Dict[str, Any]:
        """Download result document from EI/WCI."""
        logger.info(
            "Downloading EI result",
            submission_id=submission_id,
            document_type=document_type,
        )

        try:
            if not self._client:
                await self.connect()

            message = EDIMessage(
                header=EDIMessage.create_query_message(
                    sender_id="",
                    insurance_type=InsuranceType.EI,
                    reference_id=submission_id,
                ).header,
            )
            message.header.message_type = MessageType.REQUEST_DOWNLOAD

            response, _ = await self._client.send_with_retry(message)

            if response.response_data:
                return {
                    "success": True,
                    "filename": f"ei_result_{submission_id}.pdf",
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
            logger.exception("EI download failed", error=str(e))
            return {
                "success": False,
                "filename": "",
                "content": b"",
                "content_type": "",
            }
