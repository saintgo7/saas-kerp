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
import sys
from pathlib import Path
from typing import Optional, Tuple
from dataclasses import dataclass
from enum import Enum

import structlog

from .message import EDIMessage, EDIHeader, EDIBody

# Import from the shared crypto module. `python-services/` must be importable;
# appending (rather than inserting) avoids shadowing same-named stdlib or
# site-packages modules, and pathlib keeps this working off POSIX separators.
_PYTHON_SERVICES_ROOT = str(Path(__file__).resolve().parents[2])
if _PYTHON_SERVICES_ROOT not in sys.path:
    sys.path.append(_PYTHON_SERVICES_ROOT)

from shared.crypto import (  # noqa: E402
    ARIAModeCBC,
    MessageAuthentication,
    PKCS7Padding,
    PKCS7Signature,
    KeyDerivation,
    assert_strong_key,
    generate_iv,
)
from shared.crypto.pkcs7 import VerificationUnavailableError  # noqa: E402


logger = structlog.get_logger(__name__)


class ProtocolError(Exception):
    """Raised for malformed or untrustworthy EDI frames."""


class SignatureError(ProtocolError):
    """Raised when a message signature is missing, invalid, or unverifiable."""


class IntegrityError(ProtocolError):
    """Raised when the authentication tag on a received frame does not match."""


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
    # Certificate of the counterparty (공단), used to verify *their* signatures.
    # Without it, incoming signed messages are rejected rather than trusted.
    peer_certificate_path: Optional[str] = None

    # Protocol settings
    timeout: int = 30
    max_retries: int = 3
    encoding: str = "euc-kr"

    # Frame settings
    header_size: int = 100
    length_prefix_size: int = 4
    max_body_size: int = 10 * 1024 * 1024  # 10MB

    # Wire format
    # CBC ciphertext is malleable and this framing carries no AEAD, so an
    # encrypt-then-MAC tag is appended. Disable only when speaking to a
    # counterpart whose published spec forbids it -- and record why.
    mac_enabled: bool = True


class EDIProtocol:
    """
    EDI Protocol handler for 4대보험 communication.

    Handles:
    - Message framing and parsing
    - ARIA encryption/decryption (fresh IV per message, IV carried on the wire)
    - Encrypt-then-MAC integrity protection
    - PKCS#7 digital signatures (fail-closed verification)
    - Protocol state management

    WIRE FORMAT WARNING
    -------------------
    The framing below is this codebase's own invention, not a published 공단
    specification. It is *not* interoperable with a real EDI counterpart and
    must be replaced with the official 전자문서 교환 규격 before production use.
    It is kept only so the surrounding code has a coherent, safe shape.

    Frame layout:
        header (100B) | payload_length (4B) | payload
    where payload, when encryption is on:
        iv (16B) | ciphertext | mac (32B, when mac_enabled)
    and the plaintext under encryption, when signing is on:
        body_length (4B) | body | sig_length (4B) | signature
    """

    MAC_SIZE = 32  # HMAC-SHA256

    def __init__(self, config: Optional[ProtocolConfig] = None):
        """
        Initialize EDI protocol handler.

        Args:
            config: Protocol configuration

        Raises:
            ValueError: If encryption is enabled without a usable key
        """
        self.config = config or ProtocolConfig()
        self.state = ProtocolState.DISCONNECTED
        self._cipher: Optional[ARIAModeCBC] = None
        self._signer: Optional[PKCS7Signature] = None
        self._mac_key: Optional[bytes] = None
        self._padding = PKCS7Padding(block_size=16)

        self._setup_crypto()

    def _setup_crypto(self) -> None:
        """
        Initialize cryptographic components, failing closed on misconfiguration.

        Raises:
            ValueError: If encryption is requested but no usable key is supplied
        """
        if self.config.encryption_enabled:
            key = self.config.encryption_key
            if not key:
                raise ValueError(
                    "encryption_enabled is set but no encryption_key was supplied. "
                    "Refusing to transmit 주민등록번호-bearing EDI bodies in the clear."
                )
            # Rejects the all-zero "development placeholder" key outright.
            assert_strong_key(key, name="ARIA encryption key")

            # The instance IV is a placeholder only; every message gets a fresh
            # random IV via seal(). config.encryption_iv is honoured solely for
            # deterministic test vectors.
            iv = self.config.encryption_iv or generate_iv(16)
            self._cipher = ARIAModeCBC(key, iv)

            if self.config.mac_enabled:
                # Separate MAC key derived from the same secret, so the MAC key
                # is never the encryption key (RFC 5869 domain separation).
                self._mac_key = KeyDerivation.from_shared_secret(
                    key, info=b"kerp-edi-frame-mac-v1", length=32
                )

            logger.info(
                "ARIA cipher initialized",
                mac_enabled=self.config.mac_enabled,
            )

        # Setup PKCS#7 signer
        if self.config.signing_enabled:
            if not self.config.private_key_path:
                raise ValueError(
                    "signing_enabled is set but private_key_path is empty. A signer "
                    "without a key silently emits unsigned messages; disable signing "
                    "explicitly instead."
                )
            self._signer = PKCS7Signature(
                private_key_path=self.config.private_key_path,
                certificate_path=self.config.certificate_path,
            )
            if self.config.peer_certificate_path:
                self._signer.load_peer_certificate(self.config.peer_certificate_path)
            else:
                logger.warning(
                    "no_peer_certificate_configured",
                    message=(
                        "Incoming signed messages will be rejected: without the "
                        "counterparty certificate their signatures cannot be verified."
                    ),
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

        # Sign the body. A signing failure aborts transmission: emitting an
        # unsigned 취득/상실 신고 because the private key was unreadable removes
        # the tamper protection without anyone noticing.
        signature = b""
        if self.config.signing_enabled:
            if not self._signer:
                raise SignatureError(
                    "signing_enabled but no signer is configured; refusing to send "
                    "an unsigned message"
                )
            try:
                signature = self._signer.sign_raw(body_bytes)
            except Exception as exc:
                raise SignatureError(
                    f"Signing failed; refusing to transmit unsigned: {exc}"
                ) from exc
            if not signature:
                raise SignatureError("Signer produced an empty signature")
            logger.debug("Body signed", signature_size=len(signature))

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

        # Encrypt payload with a FRESH IV per message, transmitted inline.
        # Reusing one IV for the life of the connection would let an observer
        # see that two 신고 bodies share a prefix -- and these bodies begin with
        # a fixed format (문서코드|건수|사업장관리번호|사업자번호).
        if self.config.encryption_enabled:
            if not self._cipher:
                raise ProtocolError(
                    "encryption_enabled but no cipher is configured; refusing to "
                    "send in the clear"
                )
            padded = self._padding.pad(payload)
            sealed = self._cipher.seal(padded)  # iv || ciphertext

            if self._mac_key is not None:
                # Encrypt-then-MAC over iv || ciphertext.
                tag = MessageAuthentication.hmac_sha256(self._mac_key, sealed)
                sealed = sealed + tag

            logger.debug(
                "Payload encrypted",
                original_size=len(payload),
                encrypted_size=len(sealed),
            )
            payload = sealed

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
            raise ProtocolError("Message too short")

        # Parse header
        header_data = data[:self.config.header_size]
        header = EDIHeader.from_bytes(header_data)
        logger.debug("Header parsed", message_id=header.message_id)

        # Extract payload length
        length_start = self.config.header_size
        length_end = length_start + self.config.length_prefix_size
        payload_length = struct.unpack(">I", data[length_start:length_end])[0]

        if payload_length > self.config.max_body_size:
            raise ProtocolError(f"Payload too large: {payload_length}")

        # Extract payload -- and verify we actually received all of it. Slicing
        # past the end silently truncates, feeding a short body to signature
        # verification and body parsing.
        payload = data[length_end:length_end + payload_length]
        if len(payload) != payload_length:
            raise ProtocolError(
                f"Truncated frame: declared {payload_length} payload bytes, "
                f"got {len(payload)}"
            )

        # Decrypt if needed
        if header.encrypted:
            if not self._cipher:
                raise ProtocolError(
                    "Received an encrypted frame but no cipher is configured"
                )

            if self._mac_key is not None:
                if len(payload) < self.MAC_SIZE:
                    raise IntegrityError("Frame too short to carry an authentication tag")
                sealed, tag = payload[:-self.MAC_SIZE], payload[-self.MAC_SIZE:]
                # Verify before decrypting: never run attacker-chosen bytes
                # through the cipher and the unpadder.
                if not MessageAuthentication.verify_hmac(self._mac_key, sealed, tag):
                    raise IntegrityError("Frame authentication tag mismatch")
            else:
                sealed = payload

            decrypted = self._cipher.unseal(sealed)
            payload = self._padding.unpad(decrypted)
            logger.debug("Payload decrypted", size=len(payload))

        # Extract body and signature.
        # Default is False: an unverified message must never be reported as
        # verified. Callers treat False as "do not trust this response".
        signature_valid = False
        if header.signed:
            body_data, signature = self._split_signed_payload(payload)

            if not self._signer:
                raise SignatureError(
                    "Received a signed frame but no signer/verifier is configured"
                )
            try:
                signature_valid = self._signer.verify_raw(body_data, signature)
            except VerificationUnavailableError as exc:
                # No peer certificate: we cannot check, so we must not accept.
                raise SignatureError(
                    f"Cannot verify counterparty signature: {exc}"
                ) from exc

            logger.debug("Signature verified", valid=signature_valid)
            if not signature_valid:
                raise SignatureError("Counterparty signature verification failed")
        else:
            body_data = payload
            # An unsigned frame is "valid" only in the sense that there was
            # nothing to check; callers that require signing must inspect
            # header.signed themselves.
            signature_valid = not self.config.signing_enabled

        # Parse body
        body = EDIBody.from_bytes(body_data, self.config.encoding)

        message = EDIMessage(header=header, body=body)
        self._populate_response_fields(message)

        return message, signature_valid

    def _split_signed_payload(self, payload: bytes) -> Tuple[bytes, bytes]:
        """
        Split `body_length | body | sig_length | signature`, checking every bound.

        Args:
            payload: Decrypted payload bytes

        Returns:
            Tuple of (body bytes, signature bytes)

        Raises:
            ProtocolError: If any declared length runs past the buffer
        """
        if len(payload) < 4:
            raise ProtocolError("Signed payload too short for a body length prefix")

        body_length = struct.unpack(">I", payload[:4])[0]
        body_end = 4 + body_length
        if body_length > self.config.max_body_size or body_end > len(payload):
            raise ProtocolError(
                f"Declared body length {body_length} exceeds payload "
                f"({len(payload)} bytes)"
            )
        body_data = payload[4:body_end]

        if body_end + 4 > len(payload):
            raise ProtocolError("Signed payload too short for a signature length prefix")

        sig_length = struct.unpack(">I", payload[body_end:body_end + 4])[0]
        sig_end = body_end + 4 + sig_length
        if sig_end > len(payload):
            raise ProtocolError(
                f"Declared signature length {sig_length} exceeds payload "
                f"({len(payload)} bytes)"
            )
        signature = payload[body_end + 4:sig_end]

        if not signature:
            raise SignatureError("Frame is marked signed but carries no signature")

        return body_data, signature

    @staticmethod
    def _populate_response_fields(message: EDIMessage) -> None:
        """
        Fill `response_code` / `response_message` from the parsed body.

        Previously these were left at their empty-string defaults, so every
        submission was scored "not success" by `_parse_response_code` regardless
        of what the counterparty actually said.

        Args:
            message: Message whose response fields should be populated
        """
        code, text = message.body.response_status()
        message.response_code = code
        message.response_message = text

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
            raise ProtocolError(f"Payload too large: {payload_length}")

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
    def _build(
        encryption_key: bytes,
        private_key_path: Optional[str],
        certificate_path: Optional[str],
        peer_certificate_path: Optional[str],
    ) -> EDIProtocol:
        """
        Build a protocol handler with the common 4대보험 settings.

        Signing is enabled only when a private key path is supplied. That is a
        deployment choice, not a silent downgrade: `EDIProtocol` refuses to run
        with signing on and no key, and refuses to accept a signed response it
        cannot verify.

        Args:
            encryption_key: ARIA encryption key (validated by EDIProtocol)
            private_key_path: Path to our private key for signing
            certificate_path: Path to our certificate
            peer_certificate_path: Path to the agency certificate used to verify
                their signatures

        Returns:
            Configured EDI protocol handler
        """
        config = ProtocolConfig(
            encryption_enabled=True,
            encryption_key=encryption_key,
            signing_enabled=bool(private_key_path),
            private_key_path=private_key_path,
            certificate_path=certificate_path,
            peer_certificate_path=peer_certificate_path,
            timeout=30,
        )
        return EDIProtocol(config)

    @staticmethod
    def create_nps_protocol(
        encryption_key: bytes,
        private_key_path: Optional[str] = None,
        certificate_path: Optional[str] = None,
        peer_certificate_path: Optional[str] = None,
    ) -> EDIProtocol:
        """
        Create protocol handler for NPS (국민연금) communication.

        Args:
            encryption_key: ARIA encryption key
            private_key_path: Path to private key for signing
            certificate_path: Path to certificate
            peer_certificate_path: Path to the NPS certificate for verification

        Returns:
            Configured EDI protocol handler
        """
        return EDIProtocolFactory._build(
            encryption_key, private_key_path, certificate_path, peer_certificate_path
        )

    @staticmethod
    def create_nhis_protocol(
        encryption_key: bytes,
        private_key_path: Optional[str] = None,
        certificate_path: Optional[str] = None,
        peer_certificate_path: Optional[str] = None,
    ) -> EDIProtocol:
        """
        Create protocol handler for NHIS (건강보험) communication.
        """
        return EDIProtocolFactory._build(
            encryption_key, private_key_path, certificate_path, peer_certificate_path
        )

    @staticmethod
    def create_ei_protocol(
        encryption_key: bytes,
        private_key_path: Optional[str] = None,
        certificate_path: Optional[str] = None,
        peer_certificate_path: Optional[str] = None,
    ) -> EDIProtocol:
        """
        Create protocol handler for EI/WCI (고용산재보험) communication.
        """
        return EDIProtocolFactory._build(
            encryption_key, private_key_path, certificate_path, peer_certificate_path
        )
