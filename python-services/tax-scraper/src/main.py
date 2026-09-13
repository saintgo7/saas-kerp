"""
Tax Scraper Service Entry Point

gRPC server for Hometax tax invoice operations.
"""
import asyncio
import logging
import signal
import sys
from concurrent import futures
from pathlib import Path

import grpc
import structlog
from grpc_health.v1 import health, health_pb2, health_pb2_grpc
from grpc_reflection.v1alpha import reflection

# Make the service root and `python-services/` importable. `str(__file__)` was
# already a string and the "/" split broke off POSIX; pathlib is portable, and
# appending avoids shadowing installed packages of the same name.
_SERVICE_ROOT = str(Path(__file__).resolve().parents[1])
_PYTHON_SERVICES_ROOT = str(Path(__file__).resolve().parents[2])
for _path in (_SERVICE_ROOT, _PYTHON_SERVICES_ROOT):
    if _path not in sys.path:
        sys.path.append(_path)

from config import get_settings  # noqa: E402
from src.server import TaxInvoiceServicer  # noqa: E402
from shared.grpc_security import (  # noqa: E402
    ServerSecurity,
    bind_port,
    build_interceptors,
    reflection_enabled,
)

# Generated proto imports (available after proto generation)
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

    # Fail closed rather than serving a health check over an empty service.
    if not tax_pb2_grpc or not tax_pb2:
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
            ("grpc.max_send_message_length", 8 * 1024 * 1024),
            ("grpc.max_receive_message_length", 8 * 1024 * 1024),
            ("grpc.max_concurrent_streams", 64),
        ],
    )

    # Register tax invoice service
    tax_invoice_servicer = TaxInvoiceServicer()
    tax_pb2_grpc.add_TaxInvoiceServiceServicer_to_server(tax_invoice_servicer, server)
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

    # Reflection exposes the whole schema; production requires an explicit opt-in.
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

    # Start server -- TLS when configured, never unauthenticated plaintext in
    # production without an explicit opt-in.
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

    # Graceful shutdown. The servicer owns the Playwright browser and the
    # Popbill HTTP client; without closing it the node driver process survives.
    log.info("initiating_graceful_shutdown")
    await tax_invoice_servicer.close()
    await server.stop(grace=5)
    log.info("grpc_server_stopped")


def main() -> None:
    """Main entry point."""
    asyncio.run(serve())


if __name__ == "__main__":
    main()
