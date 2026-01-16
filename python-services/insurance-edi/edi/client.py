"""
EDI Client

Handles TCP/IP communication with insurance provider EDI servers.
Provides async connection management, message transmission, and
automatic retry logic.
"""
import asyncio
from typing import Optional, Tuple, Callable, Awaitable
from dataclasses import dataclass
from contextlib import asynccontextmanager

import structlog

from .message import EDIMessage, MessageType, InsuranceType
from .protocol import EDIProtocol, ProtocolConfig, ProtocolState


logger = structlog.get_logger(__name__)


@dataclass
class ConnectionConfig:
    """EDI server connection configuration."""

    host: str
    port: int
    timeout: int = 30
    max_retries: int = 3
    retry_delay: float = 1.0
    keepalive: bool = True
    ssl_enabled: bool = False


class EDIClient:
    """
    Async EDI client for communication with insurance providers.

    Provides:
    - Automatic connection management
    - Message transmission with retry
    - Connection pooling
    - Event callbacks
    """

    def __init__(
        self,
        connection_config: ConnectionConfig,
        protocol: EDIProtocol,
        on_connect: Optional[Callable[[], Awaitable[None]]] = None,
        on_disconnect: Optional[Callable[[], Awaitable[None]]] = None,
        on_error: Optional[Callable[[Exception], Awaitable[None]]] = None,
    ):
        """
        Initialize EDI client.

        Args:
            connection_config: Server connection settings
            protocol: Protocol handler for message framing
            on_connect: Callback when connected
            on_disconnect: Callback when disconnected
            on_error: Callback on error
        """
        self.config = connection_config
        self.protocol = protocol

        self._reader: Optional[asyncio.StreamReader] = None
        self._writer: Optional[asyncio.StreamWriter] = None
        self._connected = False
        self._lock = asyncio.Lock()

        # Callbacks
        self._on_connect = on_connect
        self._on_disconnect = on_disconnect
        self._on_error = on_error

    @property
    def is_connected(self) -> bool:
        """Check if client is connected."""
        return self._connected and self._writer is not None

    async def connect(self) -> None:
        """
        Establish connection to EDI server.

        Raises:
            ConnectionError: If connection fails after retries
        """
        async with self._lock:
            if self._connected:
                return

            last_error = None

            for attempt in range(self.config.max_retries):
                try:
                    logger.info(
                        "Connecting to EDI server",
                        host=self.config.host,
                        port=self.config.port,
                        attempt=attempt + 1,
                    )

                    self._reader, self._writer = await asyncio.wait_for(
                        asyncio.open_connection(
                            self.config.host,
                            self.config.port,
                            ssl=self.config.ssl_enabled,
                        ),
                        timeout=self.config.timeout,
                    )

                    self._connected = True
                    self.protocol.state = ProtocolState.CONNECTED

                    logger.info(
                        "Connected to EDI server",
                        host=self.config.host,
                        port=self.config.port,
                    )

                    if self._on_connect:
                        await self._on_connect()

                    return

                except Exception as e:
                    last_error = e
                    logger.warning(
                        "Connection attempt failed",
                        attempt=attempt + 1,
                        error=str(e),
                    )

                    if attempt < self.config.max_retries - 1:
                        await asyncio.sleep(self.config.retry_delay * (attempt + 1))

            error = ConnectionError(
                f"Failed to connect after {self.config.max_retries} attempts: {last_error}"
            )

            if self._on_error:
                await self._on_error(error)

            raise error

    async def disconnect(self) -> None:
        """Close connection to EDI server."""
        async with self._lock:
            if not self._connected:
                return

            try:
                if self._writer:
                    self._writer.close()
                    await self._writer.wait_closed()

                logger.info("Disconnected from EDI server")

                if self._on_disconnect:
                    await self._on_disconnect()

            except Exception as e:
                logger.warning("Error during disconnect", error=str(e))

            finally:
                self._reader = None
                self._writer = None
                self._connected = False
                self.protocol.state = ProtocolState.DISCONNECTED

    async def send_message(self, message: EDIMessage) -> Tuple[EDIMessage, bool]:
        """
        Send message and receive response.

        Args:
            message: EDI message to send

        Returns:
            Tuple of (response message, signature_valid)

        Raises:
            ConnectionError: If not connected
            TimeoutError: If response timeout
        """
        if not self.is_connected:
            raise ConnectionError("Not connected to EDI server")

        try:
            self.protocol.state = ProtocolState.TRANSMITTING

            # Send message
            await self.protocol.write_message(self._writer, message)

            logger.info(
                "Message sent",
                message_id=message.header.message_id,
                type=message.header.message_type.value,
            )

            # Receive response
            response, sig_valid = await self.protocol.read_message(self._reader)

            logger.info(
                "Response received",
                message_id=response.header.message_id,
                type=response.header.message_type.value,
                signature_valid=sig_valid,
            )

            self.protocol.state = ProtocolState.AUTHENTICATED

            return response, sig_valid

        except asyncio.TimeoutError:
            self.protocol.state = ProtocolState.ERROR
            error = TimeoutError("Response timeout")

            if self._on_error:
                await self._on_error(error)

            raise error

        except Exception as e:
            self.protocol.state = ProtocolState.ERROR
            logger.error("Send/receive error", error=str(e))

            if self._on_error:
                await self._on_error(e)

            raise

    async def send_with_retry(
        self,
        message: EDIMessage,
        max_retries: Optional[int] = None,
    ) -> Tuple[EDIMessage, bool]:
        """
        Send message with automatic retry on failure.

        Args:
            message: EDI message to send
            max_retries: Override default max retries

        Returns:
            Tuple of (response message, signature_valid)
        """
        retries = max_retries or self.config.max_retries
        last_error = None

        for attempt in range(retries):
            try:
                # Ensure connected
                if not self.is_connected:
                    await self.connect()

                return await self.send_message(message)

            except Exception as e:
                last_error = e
                logger.warning(
                    "Send attempt failed",
                    attempt=attempt + 1,
                    error=str(e),
                )

                # Reconnect on connection errors
                await self.disconnect()

                if attempt < retries - 1:
                    await asyncio.sleep(self.config.retry_delay * (attempt + 1))

        raise last_error

    @asynccontextmanager
    async def session(self):
        """
        Context manager for EDI session.

        Example:
            async with client.session():
                response = await client.send_message(message)
        """
        try:
            await self.connect()
            yield self
        finally:
            await self.disconnect()


class EDIClientPool:
    """
    Connection pool for EDI clients.

    Manages multiple connections for high-throughput scenarios.
    """

    def __init__(
        self,
        connection_config: ConnectionConfig,
        protocol_factory: Callable[[], EDIProtocol],
        pool_size: int = 5,
    ):
        """
        Initialize connection pool.

        Args:
            connection_config: Server connection settings
            protocol_factory: Factory function for creating protocols
            pool_size: Maximum connections in pool
        """
        self.config = connection_config
        self._protocol_factory = protocol_factory
        self._pool_size = pool_size
        self._pool: asyncio.Queue[EDIClient] = asyncio.Queue(maxsize=pool_size)
        self._created = 0
        self._lock = asyncio.Lock()

    async def _create_client(self) -> EDIClient:
        """Create a new client instance."""
        protocol = self._protocol_factory()
        client = EDIClient(self.config, protocol)
        await client.connect()
        return client

    async def acquire(self) -> EDIClient:
        """
        Acquire a client from the pool.

        Returns:
            Connected EDI client
        """
        # Try to get from pool
        try:
            return self._pool.get_nowait()
        except asyncio.QueueEmpty:
            pass

        # Create new if under limit
        async with self._lock:
            if self._created < self._pool_size:
                client = await self._create_client()
                self._created += 1
                return client

        # Wait for available client
        return await self._pool.get()

    async def release(self, client: EDIClient) -> None:
        """
        Return a client to the pool.

        Args:
            client: Client to return
        """
        if client.is_connected:
            try:
                self._pool.put_nowait(client)
            except asyncio.QueueFull:
                await client.disconnect()
        else:
            async with self._lock:
                self._created -= 1

    @asynccontextmanager
    async def client(self):
        """
        Context manager for pool client access.

        Example:
            async with pool.client() as client:
                response = await client.send_message(message)
        """
        client = await self.acquire()
        try:
            yield client
        finally:
            await self.release(client)

    async def close(self) -> None:
        """Close all connections in the pool."""
        while not self._pool.empty():
            try:
                client = self._pool.get_nowait()
                await client.disconnect()
            except asyncio.QueueEmpty:
                break

        self._created = 0


# Factory functions for specific insurance providers
def create_nps_client(
    encryption_key: bytes,
    host: str = "edi.nps.or.kr",
    port: int = 9100,
    **kwargs,
) -> EDIClient:
    """
    Create EDI client for NPS (국민연금공단).

    Args:
        encryption_key: ARIA encryption key
        host: NPS EDI server host
        port: NPS EDI server port

    Returns:
        Configured EDI client
    """
    from .protocol import EDIProtocolFactory

    config = ConnectionConfig(host=host, port=port, **kwargs)
    protocol = EDIProtocolFactory.create_nps_protocol(encryption_key)
    return EDIClient(config, protocol)


def create_nhis_client(
    encryption_key: bytes,
    host: str = "edi.nhis.or.kr",
    port: int = 9100,
    **kwargs,
) -> EDIClient:
    """
    Create EDI client for NHIS (건강보험공단).
    """
    from .protocol import EDIProtocolFactory

    config = ConnectionConfig(host=host, port=port, **kwargs)
    protocol = EDIProtocolFactory.create_nhis_protocol(encryption_key)
    return EDIClient(config, protocol)


def create_ei_client(
    encryption_key: bytes,
    host: str = "edi.comwel.or.kr",
    port: int = 9100,
    **kwargs,
) -> EDIClient:
    """
    Create EDI client for EI/WCI (고용산재보험).
    """
    from .protocol import EDIProtocolFactory

    config = ConnectionConfig(host=host, port=port, **kwargs)
    protocol = EDIProtocolFactory.create_ei_protocol(encryption_key)
    return EDIClient(config, protocol)
