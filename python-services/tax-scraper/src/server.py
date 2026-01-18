"""
gRPC Server Implementation for Tax Invoice Service

Provides the gRPC servicer that handles incoming requests and delegates
to the TaxInvoiceService for business logic.
"""

import asyncio
import signal
import time
from concurrent import futures
from typing import Any, AsyncIterator, Optional

import grpc
import structlog
from grpc_health.v1 import health, health_pb2, health_pb2_grpc
from grpc_reflection.v1alpha import reflection

from config import get_settings
from src.services.tax_service import TaxInvoiceService

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
            context.set_code(grpc.StatusCode.UNAUTHENTICATED)
            context.set_details(result.get("error_message", "Login failed"))

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

        result = await self.service.logout(session_id=request.session_id)

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
        )

        if not result["success"]:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(result.get("error_message", "Query failed"))

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
        )

        if not result["success"]:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(result.get("error_message", "Issue failed"))

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
        )

        if not result["success"]:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(result.get("error_message", "Cancel failed"))

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
        )

        if not result["success"]:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(result.get("error_message", "Status query failed"))

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
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(result.get("error_message", "Sync failed"))

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

        # This is a placeholder for streaming notifications
        # In production, this would connect to a message queue or webhook system
        while not context.cancelled():
            await asyncio.sleep(30)  # Poll interval
            # Check for new notifications and yield them
            # yield tax_pb2.InvoiceNotification(...)

    async def HealthCheck(
        self,
        request: Any,
        context: grpc.aio.ServicerContext,
    ) -> Any:
        """Handle HealthCheck RPC."""
        uptime = time.time() - self._start_time
        settings = get_settings()

        return tax_pb2.HealthCheckResponse(
            healthy=True,
            version=settings.service_version,
            uptime=f"{uptime:.2f}s",
            services={
                "hometax_scraper": True,
                "popbill_client": True,
            },
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

    # Create gRPC server
    server = grpc.aio.server(
        futures.ThreadPoolExecutor(max_workers=settings.grpc_max_workers),
        options=[
            ("grpc.max_send_message_length", 50 * 1024 * 1024),
            ("grpc.max_receive_message_length", 50 * 1024 * 1024),
            ("grpc.keepalive_time_ms", 30000),
            ("grpc.keepalive_timeout_ms", 5000),
        ],
    )

    # Create servicer
    tax_servicer = TaxInvoiceServicer()

    # Register services
    if PROTO_AVAILABLE:
        tax_pb2_grpc.add_TaxInvoiceServiceServicer_to_server(tax_servicer, server)
        log.info("tax_invoice_service_registered")

        # Optionally register PopbillService
        # popbill_servicer = PopbillServicer()
        # tax_pb2_grpc.add_PopbillServiceServicer_to_server(popbill_servicer, server)
    else:
        log.warning("proto_not_available", message="Run scripts/generate_grpc.sh first")

    # Register health check service
    health_servicer = health.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)

    if PROTO_AVAILABLE:
        health_servicer.set(
            tax_pb2.DESCRIPTOR.services_by_name["TaxInvoiceService"].full_name,
            health_pb2.HealthCheckResponse.SERVING,
        )

    log.info("health_service_registered")

    # Enable reflection for development
    if settings.grpc_reflection_enabled:
        service_names = [
            reflection.SERVICE_NAME,
            health_pb2.DESCRIPTOR.services_by_name["Health"].full_name,
        ]
        if PROTO_AVAILABLE:
            service_names.append(
                tax_pb2.DESCRIPTOR.services_by_name["TaxInvoiceService"].full_name
            )
        reflection.enable_server_reflection(service_names, server)
        log.info("grpc_reflection_enabled")

    # Start server
    listen_addr = settings.grpc_address
    server.add_insecure_port(listen_addr)

    log.info(
        "starting_grpc_server",
        address=listen_addr,
        environment=settings.environment,
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
