"""
Insurance EDI gRPC Server Entry Point
"""
import asyncio
import signal
import sys
from concurrent import futures
from pathlib import Path

import grpc
import structlog
from grpc_health.v1 import health, health_pb2, health_pb2_grpc

from config import settings

# Make `python-services/` importable for the shared modules.
_PYTHON_SERVICES_ROOT = str(Path(__file__).resolve().parents[1])
if _PYTHON_SERVICES_ROOT not in sys.path:
    sys.path.append(_PYTHON_SERVICES_ROOT)

from shared.grpc_security import (  # noqa: E402
    ServerSecurity,
    bind_port,
    build_interceptors,
    reflection_enabled,
)
from shared.utils.validators import scrub_log_processor  # noqa: E402

# Configure structured logging.
# `scrub_log_processor` runs before the renderer so a stray `data=` keyword
# cannot put a 주민등록번호 into the log stream, whatever a call site does.
structlog.configure(
    processors=[
        structlog.stdlib.filter_by_level,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
        structlog.processors.UnicodeDecoder(),
        scrub_log_processor,
        structlog.processors.JSONRenderer()
        if settings.log_format == "json"
        else structlog.dev.ConsoleRenderer(),
    ],
    wrapper_class=structlog.stdlib.BoundLogger,
    context_class=dict,
    logger_factory=structlog.stdlib.LoggerFactory(),
    cache_logger_on_first_use=True,
)

logger = structlog.get_logger(__name__)


class InsuranceEDIServer:
    """Insurance EDI gRPC Server."""

    def __init__(self):
        self.server = None
        self._servicer = None
        self._shutdown_event = asyncio.Event()

    async def start(self):
        """
        Start the gRPC server.

        Raises:
            RuntimeError: If the InsuranceService cannot be registered, or the
                transport configuration is unsafe for this environment
        """
        security = ServerSecurity.from_env(environment=settings.environment)

        self.server = grpc.aio.server(
            futures.ThreadPoolExecutor(max_workers=settings.grpc_max_workers),
            interceptors=build_interceptors(security),
            options=[
                # 4대보험 전문 are small; a 50MB ceiling is a free denial-of-service
                # amplifier. Batch submissions stay comfortably under 4MB.
                ("grpc.max_send_message_length", 4 * 1024 * 1024),
                ("grpc.max_receive_message_length", 4 * 1024 * 1024),
                ("grpc.max_concurrent_streams", 64),
                ("grpc.keepalive_time_ms", 30000),
                ("grpc.keepalive_timeout_ms", 5000),
            ],
        )

        # Register the actual service. Previously this block was commented out:
        # the process opened :50052, logged "gRPC server started", passed its
        # Docker health check, and answered every RPC with UNIMPLEMENTED.
        try:
            from generated import insurance_pb2, insurance_pb2_grpc
            from services.insurance_service import InsuranceServicer
        except ImportError as exc:
            # Fail closed: a server that cannot serve must not report healthy.
            raise RuntimeError(
                "InsuranceService could not be registered -- generated protobuf "
                "modules are missing. Run the proto generation step before "
                f"starting the service. ({exc})"
            ) from exc

        self._servicer = InsuranceServicer()
        insurance_pb2_grpc.add_InsuranceServiceServicer_to_server(
            self._servicer, self.server
        )
        logger.info("insurance_service_registered")

        # Health service reflects real registration, so an orchestrator probing
        # grpc.health.v1 learns whether the service is actually serving.
        health_servicer = health.HealthServicer()
        health_pb2_grpc.add_HealthServicer_to_server(health_servicer, self.server)
        health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)
        health_servicer.set(
            insurance_pb2.DESCRIPTOR.services_by_name["InsuranceService"].full_name,
            health_pb2.HealthCheckResponse.SERVING,
        )

        if reflection_enabled(settings.environment, configured=True):
            from grpc_reflection.v1alpha import reflection

            reflection.enable_server_reflection(
                [
                    reflection.SERVICE_NAME,
                    health_pb2.DESCRIPTOR.services_by_name["Health"].full_name,
                    insurance_pb2.DESCRIPTOR.services_by_name["InsuranceService"].full_name,
                ],
                self.server,
            )
            logger.info("grpc_reflection_enabled")

        listen_addr = f"{settings.grpc_host}:{settings.grpc_port}"
        bind_port(self.server, listen_addr, security)

        await self.server.start()
        logger.info(
            "gRPC server started",
            address=listen_addr,
            service=settings.service_name,
            version=settings.service_version,
            tls=security.tls_enabled,
            auth=bool(security.auth_token),
        )

    async def stop(self):
        """Stop the gRPC server gracefully."""
        if self.server:
            logger.info("Shutting down gRPC server...")
            await self.server.stop(grace=5)
            logger.info("gRPC server stopped")

    async def wait_for_termination(self):
        """Wait for server termination signal."""
        await self._shutdown_event.wait()

    def request_shutdown(self):
        """Request server shutdown."""
        self._shutdown_event.set()


async def serve():
    """Main server function."""
    server = InsuranceEDIServer()

    # Setup signal handlers
    loop = asyncio.get_event_loop()

    def signal_handler():
        logger.info("Received shutdown signal")
        server.request_shutdown()

    for sig in (signal.SIGTERM, signal.SIGINT):
        loop.add_signal_handler(sig, signal_handler)

    try:
        await server.start()
        await server.wait_for_termination()
    finally:
        await server.stop()


def main():
    """Entry point."""
    logger.info(
        "Starting Insurance EDI Service",
        service=settings.service_name,
        version=settings.service_version,
        environment=settings.environment,
    )

    try:
        asyncio.run(serve())
    except KeyboardInterrupt:
        logger.info("Service interrupted by user")
    except Exception as e:
        logger.exception("Service failed", error=str(e))
        sys.exit(1)


if __name__ == "__main__":
    main()
