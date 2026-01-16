"""
Tax Scraper Service Entry Point

gRPC server for Hometax tax invoice operations.
"""
import asyncio
import logging
import signal
import sys
from concurrent import futures

import grpc
import structlog
from grpc_health.v1 import health, health_pb2, health_pb2_grpc
from grpc_reflection.v1alpha import reflection

# Add parent directory to path for imports
sys.path.insert(0, str(__file__).rsplit("/", 2)[0])

from config import get_settings
from src.server import TaxInvoiceServicer

# Generated proto imports (will be available after proto generation)
try:
    from src.grpc_gen import tax_pb2, tax_pb2_grpc
except ImportError:
    tax_pb2 = None
    tax_pb2_grpc = None

logger = structlog.get_logger()


def configure_logging(settings) -> None:
    """Configure structured logging."""
    structlog.configure(
        processors=[
            structlog.contextvars.merge_contextvars,
            structlog.processors.add_log_level,
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.StackInfoRenderer(),
            structlog.processors.format_exc_info,
            (
                structlog.processors.JSONRenderer()
                if settings.log_format == "json"
                else structlog.dev.ConsoleRenderer()
            ),
        ],
        wrapper_class=structlog.make_filtering_bound_logger(
            getattr(logging, settings.log_level.upper(), logging.INFO)
        ),
        context_class=dict,
        logger_factory=structlog.PrintLoggerFactory(),
        cache_logger_on_first_use=True,
    )


async def serve() -> None:
    """Start the gRPC server."""
    settings = get_settings()
    configure_logging(settings)

    log = logger.bind(service=settings.service_name, version=settings.service_version)

    # Create gRPC server
    server = grpc.aio.server(
        futures.ThreadPoolExecutor(max_workers=settings.grpc_max_workers),
        options=[
            ("grpc.max_send_message_length", 50 * 1024 * 1024),
            ("grpc.max_receive_message_length", 50 * 1024 * 1024),
        ],
    )

    # Register tax invoice service
    if tax_pb2_grpc:
        tax_invoice_servicer = TaxInvoiceServicer()
        tax_pb2_grpc.add_TaxInvoiceServiceServicer_to_server(tax_invoice_servicer, server)
        log.info("tax_invoice_service_registered")

    # Register health check service
    health_servicer = health.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)
    if tax_pb2:
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
        if tax_pb2:
            service_names.append(tax_pb2.DESCRIPTOR.services_by_name["TaxInvoiceService"].full_name)
        reflection.enable_server_reflection(service_names, server)
        log.info("grpc_reflection_enabled")

    # Start server
    listen_addr = settings.grpc_address
    server.add_insecure_port(listen_addr)

    log.info("starting_grpc_server", address=listen_addr, environment=settings.environment)
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
    await server.stop(grace=5)
    log.info("grpc_server_stopped")


def main() -> None:
    """Main entry point."""
    asyncio.run(serve())


if __name__ == "__main__":
    main()
