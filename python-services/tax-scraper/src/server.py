"""
gRPC Server Implementation for Tax Invoice Service

Provides the gRPC servicer that handles incoming requests and delegates
to the TaxInvoiceService for business logic.
"""

import asyncio
import signal
import sys
import time
from concurrent import futures
from pathlib import Path
from typing import Any, AsyncIterator

import grpc
import structlog
from grpc_health.v1 import health, health_pb2, health_pb2_grpc
from grpc_reflection.v1alpha import reflection

from config import get_settings
from src.services.tax_service import TaxInvoiceService

# Make `python-services/` importable for the shared security helpers.
_PYTHON_SERVICES_ROOT = str(Path(__file__).resolve().parents[2])
if _PYTHON_SERVICES_ROOT not in sys.path:
    sys.path.append(_PYTHON_SERVICES_ROOT)

from shared.grpc_security import (  # noqa: E402
    ServerSecurity,
    bind_port,
    build_interceptors,
    reflection_enabled,
)

# Maps service-layer error codes to gRPC status codes, so a caller can tell an
# unimplemented path from a rejected credential from a genuine upstream fault.
# Previously every failure came back as INTERNAL with the raw exception text.
ERROR_CODE_TO_STATUS = {
    "HOMETAX_NOT_ENABLED": grpc.StatusCode.UNIMPLEMENTED,
    "NOT_IMPLEMENTED": grpc.StatusCode.UNIMPLEMENTED,
    "LOGIN_FAILED": grpc.StatusCode.UNAUTHENTICATED,
    "SESSION_INVALID": grpc.StatusCode.UNAUTHENTICATED,
    "INVALID_BUSINESS_NUMBER": grpc.StatusCode.INVALID_ARGUMENT,
    "INVALID_INVOICE_NUMBER": grpc.StatusCode.INVALID_ARGUMENT,
    "SCRAPE_FAILED": grpc.StatusCode.UNAVAILABLE,
    "RESULT_UNDETERMINED": grpc.StatusCode.UNKNOWN,
    "UPSTREAM_UNAVAILABLE": grpc.StatusCode.UNAVAILABLE,
    "INTERNAL_ERROR": grpc.StatusCode.INTERNAL,
}

# Import generated proto code (will be available after proto generation)
try:
    from src.grpc_gen import tax_pb2, tax_pb2_grpc
    PROTO_AVAILABLE = True
except ImportError:
    tax_pb2 = None
    tax_pb2_grpc = None
    PROTO_AVAILABLE = False

logger = structlog.get_logger()


class TaxInvoiceServicer:
    """gRPC servicer for TaxInvoiceService."""

    def __init__(self) -> None:
        """Initialize the servicer."""
        self.service = TaxInvoiceService()
        self.log = logger.bind(component="TaxInvoiceServicer")
        self._start_time = time.time()

    @staticmethod
    def _apply_error_status(
        context: grpc.aio.ServicerContext,
        result: dict,
        default: grpc.StatusCode = grpc.StatusCode.INTERNAL,
    ) -> None:
        """
        Set the gRPC status from a service result's error code.

        The response carries the service's own stable message; internal detail
        stays in the server log.

        Args:
            context: gRPC servicer context
            result: Service-layer result dictionary
            default: Status to use when the code is unrecognised
        """
        code = result.get("error_code", "")
        context.set_code(ERROR_CODE_TO_STATUS.get(code, default))
        context.set_details(result.get("error_message") or code or "Request failed")

    async def Login(
        self,
        request: Any,
        context: grpc.aio.ServicerContext,
    ) -> Any:
        """Handle Login RPC."""
        self.log.info(
            "rpc_login",
            business_number=request.business_number[:6] + "****" if request.business_number else "",
            auth_type=request.auth_type,
        )

        result = await self.service.login(
            company_id=request.company_id,
            business_number=request.business_number,
            auth_type=self._map_auth_type(request.auth_type),
            cert_password=request.cert_password if request.HasField("cert_password") else None,
            cert_data=request.cert_data if request.HasField("cert_data") else None,
            user_id=request.user_id if request.HasField("user_id") else None,
            password=request.password if request.HasField("password") else None,
        )

        if not result["success"]:
            self._apply_error_status(context, result, grpc.StatusCode.UNAUTHENTICATED)

        return tax_pb2.LoginResponse(
            success=result["success"],
            session_id=result.get("session_id", ""),
            expires_at=result.get("expires_at", ""),
            company_name=result.get("company_name", ""),
            error_message=result.get("error_message", ""),
            error_code=result.get("error_code", ""),
        )

    async def Logout(
        self,
        request: Any,
        context: grpc.aio.ServicerContext,
    ) -> Any:
        """Handle Logout RPC."""
        self.log.info(
            "rpc_logout",
            session_id=request.session_id[:8] + "..." if request.session_id else "",
        )

        result = await self.service.logout(
            session_id=request.session_id,
            company_id=request.company_id,
        )

        return tax_pb2.LogoutResponse(
            success=result["success"],
            error_message=result.get("error_message", ""),
        )

    async def GetTaxInvoices(
        self,
        request: Any,
        context: grpc.aio.ServicerContext,
    ) -> Any:
        """Handle GetTaxInvoices RPC."""
        self.log.info(
            "rpc_get_tax_invoices",
            session_id=request.session_id[:8] + "..." if request.session_id else "",
            start_date=request.start_date,
            end_date=request.end_date,
        )

        invoice_type = None
        if request.HasField("invoice_type"):
            invoice_type = self._map_invoice_type(request.invoice_type)

        result = await self.service.get_tax_invoices(
            session_id=request.session_id,
            start_date=request.start_date,
            end_date=request.end_date,
            invoice_type=invoice_type,
            business_number=request.business_number if request.HasField("business_number") else None,
            page=request.page or 1,
            page_size=request.page_size or 50,
            company_id=request.company_id,
        )

        if not result["success"]:
            self._apply_error_status(context, result)

        # Convert invoices to proto messages
        proto_invoices = [
            self._dict_to_proto_invoice(inv)
            for inv in result.get("invoices", [])
        ]

        return tax_pb2.GetTaxInvoicesResponse(
            success=result["success"],
            invoices=proto_invoices,
            total_count=result.get("total_count", 0),
            page=result.get("page", 1),
            page_size=result.get("page_size", 50),
            error_message=result.get("error_message", ""),
        )

    async def IssueTaxInvoice(
        self,
        request: Any,
        context: grpc.aio.ServicerContext,
    ) -> Any:
        """Handle IssueTaxInvoice RPC."""
        self.log.info(
            "rpc_issue_tax_invoice",
            session_id=request.session_id[:8] + "..." if request.session_id else "",
            provider=request.provider,
        )

        invoice_data = self._proto_invoice_to_dict(request.invoice)
        provider = "popbill" if request.provider == tax_pb2.PROVIDER_TYPE_POPBILL else "hometax"

        result = await self.service.issue_tax_invoice(
            session_id=request.session_id,
            invoice_data=invoice_data,
            provider=provider,
            transmit_immediately=request.transmit_immediately,
            company_id=request.company_id,
            idempotency_key=request.idempotency_key,
        )

        if not result["success"]:
            self._apply_error_status(context, result)

        return tax_pb2.IssueTaxInvoiceResponse(
            success=result["success"],
            invoice_number=result.get("invoice_number", ""),
            issue_date=result.get("issue_date", ""),
            nts_confirm_number=result.get("nts_confirm_number", ""),
            error_message=result.get("error_message", ""),
            error_code=result.get("error_code", ""),
        )

    async def CancelTaxInvoice(
        self,
        request: Any,
        context: grpc.aio.ServicerContext,
    ) -> Any:
        """Handle CancelTaxInvoice RPC."""
        self.log.info(
            "rpc_cancel_tax_invoice",
            session_id=request.session_id[:8] + "..." if request.session_id else "",
            invoice_number=request.invoice_number,
        )

        result = await self.service.cancel_tax_invoice(
            session_id=request.session_id,
            invoice_number=request.invoice_number,
            cancel_reason=request.cancel_reason,
            company_id=request.company_id,
        )

        if not result["success"]:
            self._apply_error_status(context, result)

        return tax_pb2.CancelTaxInvoiceResponse(
            success=result["success"],
            cancelled_at=result.get("cancelled_at", ""),
            error_message=result.get("error_message", ""),
        )

    async def GetTaxInvoiceStatus(
        self,
        request: Any,
        context: grpc.aio.ServicerContext,
    ) -> Any:
        """Handle GetTaxInvoiceStatus RPC."""
        self.log.info(
            "rpc_get_status",
            session_id=request.session_id[:8] + "..." if request.session_id else "",
            invoice_number=request.invoice_number,
        )

        result = await self.service.get_invoice_status(
            session_id=request.session_id,
            invoice_number=request.invoice_number,
            company_id=request.company_id,
        )

        if not result["success"]:
            self._apply_error_status(context, result)

        return tax_pb2.GetTaxInvoiceStatusResponse(
            success=result["success"],
            invoice_number=result.get("invoice_number", ""),
            status=self._map_status_to_proto(result.get("status", "")),
            nts_confirm_number=result.get("nts_confirm_number", ""),
            last_updated=result.get("last_updated", ""),
            error_message=result.get("error_message", ""),
        )

    async def SyncFromHometax(
        self,
        request: Any,
        context: grpc.aio.ServicerContext,
    ) -> Any:
        """Handle SyncFromHometax RPC."""
        self.log.info(
            "rpc_sync_from_hometax",
            session_id=request.session_id[:8] + "..." if request.session_id else "",
            company_id=request.company_id,
        )

        invoice_type = None
        if request.invoice_type:
            invoice_type = self._map_invoice_type(request.invoice_type)

        result = await self.service.sync_from_hometax(
            session_id=request.session_id,
            company_id=request.company_id,
            start_date=request.start_date,
            end_date=request.end_date,
            invoice_type=invoice_type,
        )

        if not result["success"]:
            self._apply_error_status(context, result)

        return tax_pb2.SyncFromHometaxResponse(
            success=result["success"],
            synced_count=result.get("synced_count", 0),
            new_count=result.get("new_count", 0),
            updated_count=result.get("updated_count", 0),
            errors=result.get("errors", []),
            error_message=result.get("error_message", ""),
        )

    async def StreamInvoiceNotifications(
        self,
        request: Any,
        context: grpc.aio.ServicerContext,
    ) -> AsyncIterator[Any]:
        """Handle StreamInvoiceNotifications RPC (server streaming)."""
        self.log.info(
            "rpc_stream_notifications",
            session_id=request.session_id[:8] + "..." if request.session_id else "",
        )

        # Not implemented. The previous body looped on a 30-second sleep and
        # never yielded: every caller waited forever and each connection pinned
        # a coroutine, with no ceiling on how many.
        await context.abort(
            grpc.StatusCode.UNIMPLEMENTED,
            "StreamInvoiceNotifications is not implemented",
        )
        # `abort` raises; the yield below only marks this as an async generator.
        yield  # pragma: no cover

    async def HealthCheck(
        self,
        request: Any,
        context: grpc.aio.ServicerContext,
    ) -> Any:
        """
        Handle HealthCheck RPC.

        Reports what is actually true. Returning a hardcoded `healthy=True` with
        both dependencies pinned to True meant a dead browser or an expired
        Popbill token still looked healthy, so an orchestrator never restarted
        the pod.
        """
        uptime = time.time() - self._start_time
        settings = get_settings()

        services = await self.service.dependency_health()
        healthy = all(services.values())

        return tax_pb2.HealthCheckResponse(
            healthy=healthy,
            version=settings.service_version,
            uptime=f"{uptime:.2f}s",
            services=services,
        )

    def _map_auth_type(self, proto_auth_type: int) -> str:
        """Map proto AuthType to string."""
        if not tax_pb2:
            return "certificate"

        mapping = {
            tax_pb2.AUTH_TYPE_CERTIFICATE: "certificate",
            tax_pb2.AUTH_TYPE_SIMPLE_AUTH: "simple_auth",
            tax_pb2.AUTH_TYPE_ID_PASSWORD: "id_password",
        }
        return mapping.get(proto_auth_type, "certificate")

    def _map_invoice_type(self, proto_invoice_type: int) -> str:
        """Map proto InvoiceType to string."""
        if not tax_pb2:
            return "sales"

        mapping = {
            tax_pb2.INVOICE_TYPE_SALES: "sales",
            tax_pb2.INVOICE_TYPE_PURCHASE: "purchase",
        }
        return mapping.get(proto_invoice_type, "sales")

    def _map_status_to_proto(self, status: str) -> int:
        """Map status string to proto InvoiceStatus."""
        if not tax_pb2:
            return 0

        mapping = {
            "draft": tax_pb2.INVOICE_STATUS_DRAFT,
            "issued": tax_pb2.INVOICE_STATUS_ISSUED,
            "transmitted": tax_pb2.INVOICE_STATUS_TRANSMITTED,
            "confirmed": tax_pb2.INVOICE_STATUS_CONFIRMED,
            "cancelled": tax_pb2.INVOICE_STATUS_CANCELLED,
            "rejected": tax_pb2.INVOICE_STATUS_REJECTED,
        }
        return mapping.get(status, tax_pb2.INVOICE_STATUS_UNSPECIFIED)

    def _dict_to_proto_invoice(self, invoice_dict: dict) -> Any:
        """Convert invoice dictionary to proto message."""
        if not tax_pb2:
            return None

        return tax_pb2.TaxInvoice(
            invoice_number=invoice_dict.get("invoice_number", ""),
            issue_date=invoice_dict.get("issue_date", ""),
            invoice_type=self._map_invoice_type_to_proto(invoice_dict.get("invoice_type", "sales")),
            status=self._map_status_to_proto(invoice_dict.get("status", "")),
            supplier_business_number=invoice_dict.get("supplier_business_number", ""),
            supplier_name=invoice_dict.get("supplier_name", ""),
            supplier_ceo_name=invoice_dict.get("supplier_ceo_name", ""),
            supplier_address=invoice_dict.get("supplier_address", ""),
            buyer_business_number=invoice_dict.get("buyer_business_number", ""),
            buyer_name=invoice_dict.get("buyer_name", ""),
            buyer_ceo_name=invoice_dict.get("buyer_ceo_name", ""),
            buyer_address=invoice_dict.get("buyer_address", ""),
            supply_amount=invoice_dict.get("supply_amount", 0),
            tax_amount=invoice_dict.get("tax_amount", 0),
            total_amount=invoice_dict.get("total_amount", 0),
            nts_confirm_number=invoice_dict.get("nts_confirm_number", ""),
            remarks=invoice_dict.get("remarks", ""),
        )

    def _map_invoice_type_to_proto(self, invoice_type: str) -> int:
        """Map invoice type string to proto enum."""
        if not tax_pb2:
            return 0

        mapping = {
            "sales": tax_pb2.INVOICE_TYPE_SALES,
            "purchase": tax_pb2.INVOICE_TYPE_PURCHASE,
        }
        return mapping.get(invoice_type, tax_pb2.INVOICE_TYPE_UNSPECIFIED)

    def _proto_invoice_to_dict(self, proto_invoice: Any) -> dict:
        """Convert proto invoice to dictionary."""
        return {
            "invoice_number": proto_invoice.invoice_number,
            "issue_date": proto_invoice.issue_date,
            "supplier_business_number": proto_invoice.supplier_business_number,
            "supplier_name": proto_invoice.supplier_name,
            "supplier_ceo_name": proto_invoice.supplier_ceo_name,
            "supplier_address": proto_invoice.supplier_address,
            "supplier_email": proto_invoice.supplier_email,
            "buyer_business_number": proto_invoice.buyer_business_number,
            "buyer_name": proto_invoice.buyer_name,
            "buyer_ceo_name": proto_invoice.buyer_ceo_name,
            "buyer_address": proto_invoice.buyer_address,
            "buyer_email": proto_invoice.buyer_email,
            "supply_amount": proto_invoice.supply_amount,
            "tax_amount": proto_invoice.tax_amount,
            "total_amount": proto_invoice.total_amount,
            "remarks": proto_invoice.remarks,
            "items": [
                {
                    "sequence": item.sequence,
                    "supply_date": item.supply_date,
                    "description": item.description,
                    "quantity": item.quantity,
                    "unit_price": item.unit_price,
                    "amount": item.amount,
                    "tax_amount": item.tax_amount,
                }
                for item in proto_invoice.items
            ],
        }

    async def close(self) -> None:
        """Close the servicer and release resources."""
        await self.service.close()


class PopbillServicer:
    """gRPC servicer for PopbillService."""

    def __init__(self) -> None:
        """Initialize the servicer."""
        self.service = TaxInvoiceService()
        self.log = logger.bind(component="PopbillServicer")

    # Implement Popbill-specific RPCs here
    # These would delegate to the service layer's Popbill methods


async def serve() -> None:
    """Start the gRPC server."""
    settings = get_settings()

    log = logger.bind(
        service=settings.service_name,
        version=settings.service_version,
    )

    # Fail closed: without the generated stubs this process would bind the port,
    # serve only a health check, report healthy, and answer every real RPC with
    # UNIMPLEMENTED. Refuse to start instead.
    if not PROTO_AVAILABLE:
        raise RuntimeError(
            "Generated protobuf modules are missing (src/grpc_gen). Run "
            "scripts/generate_grpc.sh before starting the service."
        )

    security = ServerSecurity.from_env(environment=settings.environment)

    # Create gRPC server
    server = grpc.aio.server(
        futures.ThreadPoolExecutor(max_workers=settings.grpc_max_workers),
        interceptors=build_interceptors(security),
        options=[
            # Tax invoice payloads are small. A 50MB ceiling on both directions
            # is a free memory-exhaustion lever for any caller that reaches the
            # port.
            ("grpc.max_send_message_length", 8 * 1024 * 1024),
            ("grpc.max_receive_message_length", 8 * 1024 * 1024),
            ("grpc.max_concurrent_streams", 64),
            ("grpc.keepalive_time_ms", 30000),
            ("grpc.keepalive_timeout_ms", 5000),
        ],
    )

    # Create servicer
    tax_servicer = TaxInvoiceServicer()

    tax_pb2_grpc.add_TaxInvoiceServiceServicer_to_server(tax_servicer, server)
    log.info("tax_invoice_service_registered")

    # Register health check service
    health_servicer = health.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)

    health_servicer.set(
        tax_pb2.DESCRIPTOR.services_by_name["TaxInvoiceService"].full_name,
        health_pb2.HealthCheckResponse.SERVING,
    )

    log.info("health_service_registered")

    # Reflection hands out the full service schema; keep it out of production
    # unless explicitly re-enabled.
    if reflection_enabled(settings.environment, settings.grpc_reflection_enabled):
        reflection.enable_server_reflection(
            [
                reflection.SERVICE_NAME,
                health_pb2.DESCRIPTOR.services_by_name["Health"].full_name,
                tax_pb2.DESCRIPTOR.services_by_name["TaxInvoiceService"].full_name,
            ],
            server,
        )
        log.info("grpc_reflection_enabled")

    # Start server -- TLS when configured, and never unauthenticated plaintext
    # in production without an explicit opt-in.
    listen_addr = settings.grpc_address
    bind_port(server, listen_addr, security)

    log.info(
        "starting_grpc_server",
        address=listen_addr,
        environment=settings.environment,
        tls=security.tls_enabled,
        auth=bool(security.auth_token),
    )
    await server.start()

    # Setup graceful shutdown
    loop = asyncio.get_running_loop()
    shutdown_event = asyncio.Event()

    def signal_handler(sig: signal.Signals) -> None:
        log.info("received_shutdown_signal", signal=sig.name)
        shutdown_event.set()

    for sig in (signal.SIGTERM, signal.SIGINT):
        loop.add_signal_handler(sig, lambda s=sig: signal_handler(s))

    log.info("grpc_server_started", address=listen_addr)

    # Wait for shutdown signal
    await shutdown_event.wait()

    # Graceful shutdown
    log.info("initiating_graceful_shutdown")
    await tax_servicer.close()
    await server.stop(grace=5)
    log.info("grpc_server_stopped")
