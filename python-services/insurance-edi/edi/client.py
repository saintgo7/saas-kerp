"""
EDI Client

Handles TCP/IP communication with insurance provider EDI servers.
Provides async connection management, message transmission, and
automatic retry logic.
"""
import asyncio
import ssl as ssl_module
from typing import Optional, Tuple, Callable, Awaitable
from dataclasses import dataclass
from contextlib import asynccontextmanager

import structlog

from .message import EDIMessage
from .protocol import (
    EDIProtocol,
    ProtocolError,
    ProtocolState,
    SignatureError,
)


logger = structlog.get_logger(__name__)


class PermanentEDIError(Exception):
    """
    An error that retrying cannot fix.

    Authentication failures, validation rejections and signature failures are
    permanent: retrying them burns attempts, and in the case of authentication
    can lock the account at the 공단.
    """


@dataclass
class ConnectionConfig:
    """EDI server connection configuration."""

    host: str
    port: int
    timeout: int = 30
    max_retries: int = 3
    retry_delay: float = 1.0
    keepalive: bool = True
    # TLS on by default. 4대보험 전문 carry 주민등록번호; the application-layer
    # ARIA encryption is not a substitute for transport security, and it was
    # previously possible for that ARIA key to be all zeros.
    ssl_enabled: bool = True
    # Set only for a counterpart using a private CA. `None` uses the system
    # trust store with hostname verification.
    ssl_ca_file: Optional[str] = None
    # Escape hatch for a lab/staging endpoint with a self-signed certificate.
    # Never set this in production: it disables certificate and hostname checks.
    ssl_insecure_skip_verify: bool = False

    def build_ssl_context(self) -> Optional[ssl_module.SSLContext]:
        """
        Build the TLS context for this connection.

        Returns:
            Configured SSLContext, or None when TLS is disabled

        Raises:
            ValueError: If verification is disabled without TLS being enabled
        """
        if not self.ssl_enabled:
            logger.warning(
                "edi_tls_disabled",
                host=self.host,
                port=self.port,
                message=(
                    "EDI traffic will be sent in plaintext. 주민등록번호 will be "
                    "visible to anyone on the path."
                ),
            )
            return None

        context = ssl_module.create_default_context(cafile=self.ssl_ca_file)

        if self.ssl_insecure_skip_verify:
            context.check_hostname = False
            context.verify_mode = ssl_module.CERT_NONE
            logger.warning(
                "edi_tls_verification_disabled",
                host=self.host,
                message="Certificate and hostname verification are OFF.",
            )

        return context


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
                            ssl=self.config.build_ssl_context(),
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

            raise error from last_error

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

        except asyncio.TimeoutError as exc:
            self.protocol.state = ProtocolState.ERROR
            error = TimeoutError("Response timeout")

            if self._on_error:
                await self._on_error(error)

            raise error from exc

        except Exception as e:
            self.protocol.state = ProtocolState.ERROR
            logger.error("Send/receive error", error=str(e))

            if self._on_error:
                await self._on_error(e)

            raise

    # Exceptions that a retry might plausibly get past. Anything else -- a
    # rejected signature, a validation failure, a malformed frame -- is
    # permanent, and retrying only wastes attempts.
    _RETRYABLE = (ConnectionError, ConnectionResetError, asyncio.IncompleteReadError)

    @classmethod
    def _is_retryable(cls, error: BaseException) -> bool:
        """Decide whether an error is worth another attempt."""
        if isinstance(error, (PermanentEDIError, SignatureError)):
            return False
        if isinstance(error, ProtocolError):
            # A malformed frame will be malformed again.
            return False
        return isinstance(error, cls._RETRYABLE)

    async def send_with_retry(
        self,
        message: EDIMessage,
        max_retries: Optional[int] = None,
        allow_resend_after_write: bool = False,
    ) -> Tuple[EDIMessage, bool]:
        """
        Send message, retrying only errors that a retry can actually fix.

        Retry safety
        ------------
        `send_message` writes the request and then waits for the response. If
        the write succeeded and the *response* timed out, the 공단 may already
        have accepted the filing. Resending the same 취득신고 in that state
        produces a duplicate (their error code 2001, 중복 신고).

        So a failure after the request was written is not retried unless the
        caller passes `allow_resend_after_write=True`, which it should only do
        when the message carries a stable idempotency key (see
        `EDIMessage.create_submit_message(idempotency_key=...)`) or the
        operation is read-only.

        Args:
            message: EDI message to send
            max_retries: Override default max retries
            allow_resend_after_write: Permit retrying once the request has
                already reached the wire

        Returns:
            Tuple of (response message, signature_valid)

        Raises:
            Exception: The last error encountered
        """
        retries = max_retries or self.config.max_retries
        last_error: Optional[BaseException] = None

        for attempt in range(retries):
            request_written = False
            try:
                # Ensure connected
                if not self.is_connected:
                    await self.connect()

                await self.protocol.write_message(self._writer, message)
                request_written = True

                logger.info(
                    "Message sent",
                    message_id=message.header.message_id,
                    type=message.header.message_type.value,
                    attempt=attempt + 1,
                )

                response, sig_valid = await self.protocol.read_message(self._reader)

                logger.info(
                    "Response received",
                    message_id=response.header.message_id,
                    signature_valid=sig_valid,
                )

                self.protocol.state = ProtocolState.AUTHENTICATED
                return response, sig_valid

            except Exception as e:
                last_error = e
                self.protocol.state = ProtocolState.ERROR

                logger.warning(
                    "Send attempt failed",
                    attempt=attempt + 1,
                    error=str(e),
                    request_written=request_written,
                )

                if self._on_error:
                    await self._on_error(e)

                await self.disconnect()

                if not self._is_retryable(e):
                    logger.info("not_retrying_permanent_error", error_type=type(e).__name__)
                    raise

                if request_written and not allow_resend_after_write:
                    logger.error(
                        "not_retrying_after_write",
                        message_id=message.header.message_id,
                        message=(
                            "Request reached the server but the response did not "
                            "arrive. The filing may already be accepted; resending "
                            "could duplicate it. Query status before retrying."
                        ),
                    )
                    raise

                if attempt < retries - 1:
                    await asyncio.sleep(self.config.retry_delay * (attempt + 1))

        assert last_error is not None
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

    async def acquire(self, timeout: Optional[float] = None) -> EDIClient:
        """
        Acquire a client from the pool.

        Args:
            timeout: Seconds to wait for a free client; defaults to the
                connection timeout. Waiting forever is not an option: if every
                client is dropped while a coroutine waits on the queue, nothing
                ever wakes it.

        Returns:
            Connected EDI client

        Raises:
            TimeoutError: If no client became available in time
        """
        # Try to get from pool
        try:
            return self._pool.get_nowait()
        except asyncio.QueueEmpty:
            pass

        # Reserve a slot under the lock, but create the connection outside it:
        # `connect()` retries with backoff and would otherwise serialise the
        # whole pool behind one slow handshake.
        reserved = False
        async with self._lock:
            if self._created < self._pool_size:
                self._created += 1
                reserved = True

        if reserved:
            try:
                return await self._create_client()
            except Exception:
                # Give the slot back, or the pool permanently shrinks.
                async with self._lock:
                    self._created -= 1
                raise

        wait_for = timeout if timeout is not None else float(self.config.timeout)
        try:
            return await asyncio.wait_for(self._pool.get(), timeout=wait_for)
        except asyncio.TimeoutError as exc:
            raise TimeoutError(
                f"No EDI client available within {wait_for}s (pool size {self._pool_size})"
            ) from exc

    async def release(self, client: EDIClient) -> None:
        """
        Return a client to the pool.

        Args:
            client: Client to return
        """
        if client.is_connected:
            try:
                self._pool.put_nowait(client)
                return
            except asyncio.QueueFull:
                await client.disconnect()

        async with self._lock:
            self._created = max(0, self._created - 1)

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
