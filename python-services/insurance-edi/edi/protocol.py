"""
EDI Protocol Handler

Handles the protocol-level operations for EDI communication including:
- Message framing
- Encryption/decryption
- Digital signing
- Protocol state management
"""
import struct
import asyncio
from typing import Optional, Tuple
from dataclasses import dataclass
from enum import Enum

import structlog

from .message import EDIMessage, EDIHeader, EDIBody, MessageType

# Import from shared crypto module
import sys
sys.path.insert(0, str(__file__).rsplit("/", 3)[0])
from shared.crypto import ARIAModeCBC, PKCS7Padding, PKCS7Signature, generate_iv


logger = structlog.get_logger(__name__)


class ProtocolState(Enum):
    """Protocol connection states."""

    DISCONNECTED = "disconnected"
    CONNECTING = "connecting"
    CONNECTED = "connected"
    AUTHENTICATING = "authenticating"
    AUTHENTICATED = "authenticated"
    TRANSMITTING = "transmitting"
    ERROR = "error"


@dataclass
class ProtocolConfig:
    """EDI protocol configuration."""

    # Encryption settings
    encryption_enabled: bool = True
    encryption_key: Optional[bytes] = None
    encryption_iv: Optional[bytes] = None

    # Signing settings
    signing_enabled: bool = True
    private_key_path: Optional[str] = None
    certificate_path: Optional[str] = None

    # Protocol settings
    timeout: int = 30
    max_retries: int = 3
    encoding: str = "euc-kr"

    # Frame settings
    header_size: int = 100
    length_prefix_size: int = 4
    max_body_size: int = 10 * 1024 * 1024  # 10MB


class EDIProtocol:
    """
    EDI Protocol handler for 4대보험 communication.

    Handles:
    - Message framing and parsing
    - ARIA encryption/decryption
    - PKCS#7 digital signatures
    - Protocol state management
    """

    def __init__(self, config: Optional[ProtocolConfig] = None):
        """
        Initialize EDI protocol handler.

        Args:
            config: Protocol configuration
        """
        self.config = config or ProtocolConfig()
        self.state = ProtocolState.DISCONNECTED
        self._cipher: Optional[ARIAModeCBC] = None
        self._signer: Optional[PKCS7Signature] = None
        self._padding = PKCS7Padding(block_size=16)

        self._setup_crypto()

    def _setup_crypto(self) -> None:
        """Initialize cryptographic components."""
        # Setup ARIA cipher
        if self.config.encryption_enabled and self.config.encryption_key:
            iv = self.config.encryption_iv or generate_iv(16)
            self._cipher = ARIAModeCBC(self.config.encryption_key, iv)
            logger.info("ARIA cipher initialized")

        # Setup PKCS#7 signer
        if self.config.signing_enabled:
            self._signer = PKCS7Signature(
                private_key_path=self.config.private_key_path,
                certificate_path=self.config.certificate_path,
            )
            logger.info("PKCS7 signer initialized")

    def frame_message(self, message: EDIMessage) -> bytes:
        """
        Frame an EDI message for transmission.

        Steps:
        1. Serialize message to bytes
        2. Sign if enabled
        3. Encrypt if enabled
        4. Add length prefix

        Args:
            message: EDI message to frame

        Returns:
            Framed message bytes ready for transmission
        """
        # Serialize body
        body_bytes = message.body.to_bytes(self.config.encoding)
        logger.debug("Body serialized", size=len(body_bytes))

        # Sign the body
        signature = b""
        if self.config.signing_enabled and self._signer:
            try:
                signature = self._signer.sign_raw(body_bytes)
                logger.debug("Body signed", signature_size=len(signature))
            except Exception as e:
                logger.warning("Signing failed", error=str(e))

        # Combine body + signature
        if signature:
            # Format: body_length (4B) + body + signature_length (4B) + signature
            payload = (
                struct.pack(">I", len(body_bytes)) +
                body_bytes +
                struct.pack(">I", len(signature)) +
                signature
            )
        else:
            payload = body_bytes

        # Encrypt payload
        if self.config.encryption_enabled and self._cipher:
            padded = self._padding.pad(payload)
            encrypted = self._cipher.encrypt(padded)
            logger.debug(
                "Payload encrypted",
                original_size=len(payload),
                encrypted_size=len(encrypted),
            )
            payload = encrypted

        # Update header flags
        message.header.encrypted = bool(self._cipher)
        message.header.signed = bool(signature)

        # Serialize header
        header_bytes = message.header.to_bytes()

        # Combine: header (100B) + payload_length (4B) + payload
        framed = (
            header_bytes +
            struct.pack(">I", len(payload)) +
            payload
        )

        logger.info(
            "Message framed",
            total_size=len(framed),
            encrypted=message.header.encrypted,
            signed=message.header.signed,
        )

        return framed

    def parse_message(self, data: bytes) -> Tuple[EDIMessage, bool]:
        """
        Parse a received EDI message.

        Steps:
        1. Extract header
        2. Extract payload
        3. Decrypt if encrypted
        4. Verify signature if signed
        5. Parse body

        Args:
            data: Received message bytes

        Returns:
            Tuple of (parsed message, signature_valid)
        """
        if len(data) < self.config.header_size + self.config.length_prefix_size:
            raise ValueError("Message too short")

        # Parse header
        header_data = data[:self.config.header_size]
        header = EDIHeader.from_bytes(header_data)
        logger.debug("Header parsed", message_id=header.message_id)

        # Extract payload length
        length_start = self.config.header_size
        length_end = length_start + self.config.length_prefix_size
        payload_length = struct.unpack(">I", data[length_start:length_end])[0]

        if payload_length > self.config.max_body_size:
            raise ValueError(f"Payload too large: {payload_length}")

        # Extract payload
        payload = data[length_end:length_end + payload_length]

        # Decrypt if needed
        if header.encrypted and self._cipher:
            decrypted = self._cipher.decrypt(payload)
            payload = self._padding.unpad(decrypted)
            logger.debug("Payload decrypted", size=len(payload))

        # Extract body and signature
        signature_valid = True
        if header.signed:
            # Format: body_length (4B) + body + signature_length (4B) + signature
            body_length = struct.unpack(">I", payload[:4])[0]
            body_data = payload[4:4 + body_length]

            sig_length_start = 4 + body_length
            sig_length = struct.unpack(">I", payload[sig_length_start:sig_length_start + 4])[0]
            signature = payload[sig_length_start + 4:sig_length_start + 4 + sig_length]

            # Verify signature
            if self._signer:
                signature_valid = self._signer.verify_raw(body_data, signature)
                logger.debug("Signature verified", valid=signature_valid)
        else:
            body_data = payload

        # Parse body
        body = EDIBody.from_bytes(body_data, self.config.encoding)

        message = EDIMessage(header=header, body=body)

        return message, signature_valid

    async def read_message(
        self,
        reader: asyncio.StreamReader,
    ) -> Tuple[EDIMessage, bool]:
        """
        Read and parse a message from stream.

        Args:
            reader: Async stream reader

        Returns:
            Tuple of (parsed message, signature_valid)
        """
        # Read header
        header_data = await asyncio.wait_for(
            reader.readexactly(self.config.header_size),
            timeout=self.config.timeout,
        )

        # Read length prefix
        length_data = await asyncio.wait_for(
            reader.readexactly(self.config.length_prefix_size),
            timeout=self.config.timeout,
        )
        payload_length = struct.unpack(">I", length_data)[0]

        if payload_length > self.config.max_body_size:
            raise ValueError(f"Payload too large: {payload_length}")

        # Read payload
        payload = await asyncio.wait_for(
            reader.readexactly(payload_length),
            timeout=self.config.timeout,
        )

        # Combine and parse
        full_data = header_data + length_data + payload
        return self.parse_message(full_data)

    async def write_message(
        self,
        writer: asyncio.StreamWriter,
        message: EDIMessage,
    ) -> None:
        """
        Frame and write a message to stream.

        Args:
            writer: Async stream writer
            message: Message to send
        """
        framed = self.frame_message(message)
        writer.write(framed)
        await writer.drain()
        logger.debug("Message written", size=len(framed))


class EDIProtocolFactory:
    """Factory for creating protocol handlers with specific configurations."""

    @staticmethod
    def create_nps_protocol(
        encryption_key: bytes,
        private_key_path: Optional[str] = None,
        certificate_path: Optional[str] = None,
    ) -> EDIProtocol:
        """
        Create protocol handler for NPS (국민연금) communication.

        Args:
            encryption_key: ARIA encryption key
            private_key_path: Path to private key for signing
            certificate_path: Path to certificate

        Returns:
            Configured EDI protocol handler
        """
        config = ProtocolConfig(
            encryption_enabled=True,
            encryption_key=encryption_key,
            signing_enabled=bool(private_key_path),
            private_key_path=private_key_path,
            certificate_path=certificate_path,
            timeout=30,
        )
        return EDIProtocol(config)

    @staticmethod
    def create_nhis_protocol(
        encryption_key: bytes,
        private_key_path: Optional[str] = None,
        certificate_path: Optional[str] = None,
    ) -> EDIProtocol:
        """
        Create protocol handler for NHIS (건강보험) communication.
        """
        config = ProtocolConfig(
            encryption_enabled=True,
            encryption_key=encryption_key,
            signing_enabled=bool(private_key_path),
            private_key_path=private_key_path,
            certificate_path=certificate_path,
            timeout=30,
        )
        return EDIProtocol(config)

    @staticmethod
    def create_ei_protocol(
        encryption_key: bytes,
        private_key_path: Optional[str] = None,
        certificate_path: Optional[str] = None,
    ) -> EDIProtocol:
        """
        Create protocol handler for EI/WCI (고용산재보험) communication.
        """
        config = ProtocolConfig(
            encryption_enabled=True,
            encryption_key=encryption_key,
            signing_enabled=bool(private_key_path),
            private_key_path=private_key_path,
            certificate_path=certificate_path,
            timeout=30,
        )
        return EDIProtocol(config)
