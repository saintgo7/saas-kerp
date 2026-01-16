"""
Insurance EDI gRPC Server Entry Point
"""
import asyncio
import signal
import sys
from concurrent import futures

import grpc
import structlog

from config import settings

# Configure structured logging
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
        self._shutdown_event = asyncio.Event()

    async def start(self):
        """Start the gRPC server."""
        self.server = grpc.aio.server(
            futures.ThreadPoolExecutor(max_workers=settings.grpc_max_workers),
            options=[
                ("grpc.max_send_message_length", 50 * 1024 * 1024),
                ("grpc.max_receive_message_length", 50 * 1024 * 1024),
            ],
        )

        # Register services (will be added after proto generation)
        # from generated import insurance_pb2_grpc
        # from services.insurance_service import InsuranceServicer
        # insurance_pb2_grpc.add_InsuranceServiceServicer_to_server(
        #     InsuranceServicer(), self.server
        # )

        listen_addr = f"{settings.grpc_host}:{settings.grpc_port}"
        self.server.add_insecure_port(listen_addr)

        await self.server.start()
        logger.info(
            "gRPC server started",
            address=listen_addr,
            service=settings.service_name,
            version=settings.service_version,
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
