"""
Insurance gRPC Service Implementation

Implements the InsuranceService defined in insurance.proto.
Handles 4대보험 EDI operations through gRPC.
"""
import uuid
from datetime import datetime
from typing import Dict, List, Optional

import grpc
import structlog

# These imports will work after proto generation
# from generated import insurance_pb2
# from generated import insurance_pb2_grpc

from config import settings
from edi.client import create_nps_client, create_nhis_client, create_ei_client, EDIClient
from edi.message import (
    EDIMessage,
    InsuranceType as EDIInsuranceType,
    DocumentType as EDIDocumentType,
)
from providers import NPSProvider, NHISProvider, EIProvider, BaseProvider


logger = structlog.get_logger(__name__)


class InsuranceServicer:
    """
    gRPC Insurance Service implementation.

    Provides methods for:
    - Document submission (취득/상실/변경)
    - Status queries
    - Result downloads
    - Batch operations
    """

    def __init__(self):
        """Initialize service with providers."""
        self._providers: Dict[int, BaseProvider] = {}
        self._clients: Dict[int, EDIClient] = {}
        self._init_providers()

    def _init_providers(self) -> None:
        """Initialize insurance providers."""
        # Note: In production, encryption keys should come from config/secrets
        encryption_key = settings.crypto.aria_key
        if encryption_key:
            key_bytes = bytes.fromhex(encryption_key)
        else:
            # Use placeholder for development
            key_bytes = bytes(16)
            logger.warning("Using placeholder encryption key - configure ARIA_ENCRYPTION_KEY for production")

        # Initialize providers
        self._providers = {
            1: NPSProvider(),   # NPS - 국민연금
            2: NHISProvider(),  # NHIS - 건강보험
            3: EIProvider(),    # EI - 고용보험
            4: EIProvider(),    # WCI - 산재보험 (same provider)
        }

        logger.info("Insurance providers initialized", count=len(self._providers))

    def _get_provider(self, insurance_type: int) -> Optional[BaseProvider]:
        """Get provider for insurance type."""
        return self._providers.get(insurance_type)

    async def SubmitAcquisition(self, request, context):
        """
        Handle acquisition submission (취득신고).

        Args:
            request: AcquisitionRequest
            context: gRPC context

        Returns:
            SubmissionResponse
        """
        # Import here to avoid circular imports before proto generation
        from generated import insurance_pb2

        request_id = request.request_id or str(uuid.uuid4())

        logger.info(
            "Processing acquisition submission",
            request_id=request_id,
            company_id=request.company.company_id,
            employee_name=request.employee.name,
            insurance_types=[t for t in request.insurance_types],
        )

        results = []
        all_success = True

        for ins_type in request.insurance_types:
            provider = self._get_provider(ins_type)
            if not provider:
                results.append(insurance_pb2.InsuranceSubmissionResult(
                    insurance_type=ins_type,
                    success=False,
                    error_code="PROVIDER_NOT_FOUND",
                    error_message=f"Provider not configured for type {ins_type}",
                ))
                all_success = False
                continue

            try:
                # Build submission data
                submission_data = {
                    "company": {
                        "business_no": request.company.business_no,
                        "workplace_no": request.company.workplace_no,
                        "name": request.company.company_name,
                    },
                    "employee": {
                        "name": request.employee.name,
                        "resident_no": request.employee.resident_no,
                        "nationality": request.employee.nationality,
                    },
                    "acquisition": {
                        "date": request.data.acquisition_date,
                        "job_type": request.data.job_type,
                        "monthly_income": request.data.monthly_income,
                        "work_hours": request.data.work_hours_weekly,
                        "contract_type": request.data.contract_type,
                    },
                }

                # Submit through provider
                result = await provider.submit_acquisition(submission_data)

                results.append(insurance_pb2.InsuranceSubmissionResult(
                    insurance_type=ins_type,
                    success=result.get("success", False),
                    reference_id=result.get("reference_id", ""),
                    error_code=result.get("error_code", ""),
                    error_message=result.get("error_message", ""),
                ))

                if not result.get("success"):
                    all_success = False

            except Exception as e:
                logger.exception("Submission failed", insurance_type=ins_type, error=str(e))
                results.append(insurance_pb2.InsuranceSubmissionResult(
                    insurance_type=ins_type,
                    success=False,
                    error_code="SUBMISSION_ERROR",
                    error_message=str(e),
                ))
                all_success = False

        submission_id = f"ACQ-{request_id[:8]}-{datetime.now().strftime('%Y%m%d%H%M%S')}"

        return insurance_pb2.SubmissionResponse(
            success=all_success,
            message="Acquisition submitted successfully" if all_success else "Some submissions failed",
            submission_id=submission_id,
            results=results,
        )

    async def SubmitLoss(self, request, context):
        """Handle loss submission (상실신고)."""
        from generated import insurance_pb2

        request_id = request.request_id or str(uuid.uuid4())

        logger.info(
            "Processing loss submission",
            request_id=request_id,
            company_id=request.company.company_id,
            employee_name=request.employee.name,
        )

        results = []
        all_success = True

        for ins_type in request.insurance_types:
            provider = self._get_provider(ins_type)
            if not provider:
                results.append(insurance_pb2.InsuranceSubmissionResult(
                    insurance_type=ins_type,
                    success=False,
                    error_code="PROVIDER_NOT_FOUND",
                    error_message=f"Provider not configured for type {ins_type}",
                ))
                all_success = False
                continue

            try:
                submission_data = {
                    "company": {
                        "business_no": request.company.business_no,
                        "workplace_no": request.company.workplace_no,
                        "name": request.company.company_name,
                    },
                    "employee": {
                        "name": request.employee.name,
                        "resident_no": request.employee.resident_no,
                    },
                    "loss": {
                        "date": request.data.loss_date,
                        "reason_code": request.data.loss_reason_code,
                        "reason_detail": request.data.loss_reason_detail,
                        "final_income": request.data.final_monthly_income,
                        "is_voluntary": request.data.is_voluntary,
                    },
                }

                result = await provider.submit_loss(submission_data)

                results.append(insurance_pb2.InsuranceSubmissionResult(
                    insurance_type=ins_type,
                    success=result.get("success", False),
                    reference_id=result.get("reference_id", ""),
                    error_code=result.get("error_code", ""),
                    error_message=result.get("error_message", ""),
                ))

                if not result.get("success"):
                    all_success = False

            except Exception as e:
                logger.exception("Loss submission failed", insurance_type=ins_type, error=str(e))
                results.append(insurance_pb2.InsuranceSubmissionResult(
                    insurance_type=ins_type,
                    success=False,
                    error_code="SUBMISSION_ERROR",
                    error_message=str(e),
                ))
                all_success = False

        submission_id = f"LOSS-{request_id[:8]}-{datetime.now().strftime('%Y%m%d%H%M%S')}"

        return insurance_pb2.SubmissionResponse(
            success=all_success,
            message="Loss submitted successfully" if all_success else "Some submissions failed",
            submission_id=submission_id,
            results=results,
        )

    async def SubmitChange(self, request, context):
        """Handle change submission (변경신고)."""
        from generated import insurance_pb2

        request_id = request.request_id or str(uuid.uuid4())

        logger.info(
            "Processing change submission",
            request_id=request_id,
            company_id=request.company.company_id,
        )

        results = []
        all_success = True

        for ins_type in request.insurance_types:
            provider = self._get_provider(ins_type)
            if not provider:
                results.append(insurance_pb2.InsuranceSubmissionResult(
                    insurance_type=ins_type,
                    success=False,
                    error_code="PROVIDER_NOT_FOUND",
                    error_message=f"Provider not configured for type {ins_type}",
                ))
                all_success = False
                continue

            try:
                submission_data = {
                    "company": {
                        "business_no": request.company.business_no,
                        "workplace_no": request.company.workplace_no,
                    },
                    "employee": {
                        "name": request.employee.name,
                        "resident_no": request.employee.resident_no,
                    },
                    "change": {
                        "date": request.data.change_date,
                        "type": request.data.change_type,
                        "before": request.data.before_value,
                        "after": request.data.after_value,
                        "reason": request.data.reason,
                    },
                }

                result = await provider.submit_change(submission_data)

                results.append(insurance_pb2.InsuranceSubmissionResult(
                    insurance_type=ins_type,
                    success=result.get("success", False),
                    reference_id=result.get("reference_id", ""),
                    error_code=result.get("error_code", ""),
                    error_message=result.get("error_message", ""),
                ))

                if not result.get("success"):
                    all_success = False

            except Exception as e:
                logger.exception("Change submission failed", error=str(e))
                results.append(insurance_pb2.InsuranceSubmissionResult(
                    insurance_type=ins_type,
                    success=False,
                    error_code="SUBMISSION_ERROR",
                    error_message=str(e),
                ))
                all_success = False

        submission_id = f"CHG-{request_id[:8]}-{datetime.now().strftime('%Y%m%d%H%M%S')}"

        return insurance_pb2.SubmissionResponse(
            success=all_success,
            message="Change submitted successfully" if all_success else "Some submissions failed",
            submission_id=submission_id,
            results=results,
        )

    async def QueryStatus(self, request, context):
        """Query submission status."""
        from generated import insurance_pb2

        logger.info(
            "Querying status",
            submission_id=request.submission_id,
            insurance_type=request.insurance_type,
        )

        provider = self._get_provider(request.insurance_type)
        if not provider:
            return insurance_pb2.StatusResponse(
                submission_id=request.submission_id,
                insurance_type=request.insurance_type,
                status=insurance_pb2.SUBMISSION_STATUS_ERROR,
                status_message="Provider not configured",
            )

        try:
            result = await provider.query_status(request.submission_id)

            status_map = {
                "pending": insurance_pb2.SUBMISSION_STATUS_PENDING,
                "processing": insurance_pb2.SUBMISSION_STATUS_PROCESSING,
                "completed": insurance_pb2.SUBMISSION_STATUS_COMPLETED,
                "rejected": insurance_pb2.SUBMISSION_STATUS_REJECTED,
                "error": insurance_pb2.SUBMISSION_STATUS_ERROR,
            }

            errors = [
                insurance_pb2.ValidationError(
                    field=e.get("field", ""),
                    code=e.get("code", ""),
                    message=e.get("message", ""),
                )
                for e in result.get("errors", [])
            ]

            return insurance_pb2.StatusResponse(
                submission_id=request.submission_id,
                insurance_type=request.insurance_type,
                status=status_map.get(result.get("status"), insurance_pb2.SUBMISSION_STATUS_UNSPECIFIED),
                status_message=result.get("message", ""),
                processed_at=result.get("processed_at", ""),
                errors=errors,
            )

        except Exception as e:
            logger.exception("Status query failed", error=str(e))
            return insurance_pb2.StatusResponse(
                submission_id=request.submission_id,
                insurance_type=request.insurance_type,
                status=insurance_pb2.SUBMISSION_STATUS_ERROR,
                status_message=str(e),
            )

    async def DownloadResult(self, request, context):
        """Download result document."""
        from generated import insurance_pb2

        logger.info(
            "Downloading result",
            submission_id=request.submission_id,
            insurance_type=request.insurance_type,
        )

        provider = self._get_provider(request.insurance_type)
        if not provider:
            return insurance_pb2.DownloadResponse(
                success=False,
            )

        try:
            result = await provider.download_result(
                request.submission_id,
                request.document_type,
            )

            return insurance_pb2.DownloadResponse(
                success=result.get("success", False),
                filename=result.get("filename", ""),
                content=result.get("content", b""),
                content_type=result.get("content_type", "application/octet-stream"),
            )

        except Exception as e:
            logger.exception("Download failed", error=str(e))
            return insurance_pb2.DownloadResponse(success=False)

    async def BatchSubmit(self, request, context):
        """Handle batch submission."""
        from generated import insurance_pb2

        logger.info(
            "Processing batch submission",
            batch_id=request.batch_id,
            item_count=len(request.items),
        )

        results = []
        success_count = 0
        failed_count = 0

        for item in request.items:
            try:
                # Determine document type and create appropriate request
                if item.HasField("acquisition"):
                    # Create acquisition request
                    acq_request = insurance_pb2.AcquisitionRequest(
                        request_id=item.item_id,
                        company=request.company,
                        employee=item.employee,
                        data=item.acquisition,
                        insurance_types=list(item.insurance_types),
                    )
                    response = await self.SubmitAcquisition(acq_request, context)
                elif item.HasField("loss"):
                    loss_request = insurance_pb2.LossRequest(
                        request_id=item.item_id,
                        company=request.company,
                        employee=item.employee,
                        data=item.loss,
                        insurance_types=list(item.insurance_types),
                    )
                    response = await self.SubmitLoss(loss_request, context)
                elif item.HasField("change"):
                    change_request = insurance_pb2.ChangeRequest(
                        request_id=item.item_id,
                        company=request.company,
                        employee=item.employee,
                        data=item.change,
                        insurance_types=list(item.insurance_types),
                    )
                    response = await self.SubmitChange(change_request, context)
                else:
                    results.append(insurance_pb2.BatchItemResult(
                        item_id=item.item_id,
                        success=False,
                        error_message="Unknown document type",
                    ))
                    failed_count += 1
                    continue

                if response.success:
                    success_count += 1
                else:
                    failed_count += 1

                results.append(insurance_pb2.BatchItemResult(
                    item_id=item.item_id,
                    success=response.success,
                    submission_id=response.submission_id,
                    error_message=response.message if not response.success else "",
                ))

            except Exception as e:
                logger.exception("Batch item failed", item_id=item.item_id, error=str(e))
                results.append(insurance_pb2.BatchItemResult(
                    item_id=item.item_id,
                    success=False,
                    error_message=str(e),
                ))
                failed_count += 1

        return insurance_pb2.BatchSubmitResponse(
            success=failed_count == 0,
            batch_id=request.batch_id,
            total_count=len(request.items),
            success_count=success_count,
            failed_count=failed_count,
            results=results,
        )

    async def HealthCheck(self, request, context):
        """Check service health and provider connectivity."""
        from generated import insurance_pb2

        providers_status = {}

        for ins_type, provider in self._providers.items():
            try:
                is_healthy = await provider.health_check()
                providers_status[str(ins_type)] = is_healthy
            except Exception:
                providers_status[str(ins_type)] = False

        all_healthy = all(providers_status.values())

        return insurance_pb2.HealthCheckResponse(
            healthy=all_healthy,
            version=settings.service_version,
            providers=providers_status,
        )
