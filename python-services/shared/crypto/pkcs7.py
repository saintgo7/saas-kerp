"""
PKCS#7 Padding and Signature Implementation

PKCS#7 is used for:
1. Block cipher padding (PKCS#7 padding scheme)
2. Cryptographic Message Syntax (CMS/PKCS#7) for digital signatures

Required for 4대보험 EDI communication and electronic document signing.
"""
from typing import Optional
from datetime import datetime, timezone

from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import padding
from cryptography.hazmat.primitives.serialization import pkcs7
from cryptography.x509.oid import NameOID


class VerificationUnavailableError(RuntimeError):
    """
    Raised when a signature cannot be checked at all (no peer certificate).

    Distinct from a verification *failure* so callers cannot conflate "we could
    not check" with "it checked out".
    """


class PaddingError(ValueError):
    """
    Raised for any PKCS#7 unpadding failure.

    Deliberately carries a single, constant message: distinguishing "bad padding
    length" from "bad padding bytes" hands an attacker a padding oracle.
    """

    def __init__(self, message: str = "Invalid padding") -> None:
        super().__init__(message)


class PKCS7Padding:
    """
    PKCS#7 padding implementation for block ciphers.

    PKCS#7 padding adds N bytes of value N to make the data
    a multiple of the block size.
    """

    def __init__(self, block_size: int = 16):
        """
        Initialize PKCS7 padding.

        Args:
            block_size: Block size in bytes (default: 16 for 128-bit)
        """
        if not 1 <= block_size <= 255:
            raise ValueError("Block size must be between 1 and 255")
        self._block_size = block_size

    @property
    def block_size(self) -> int:
        """Return the block size."""
        return self._block_size

    def pad(self, data: bytes) -> bytes:
        """
        Add PKCS#7 padding to data.

        Args:
            data: Data to pad

        Returns:
            Padded data (multiple of block_size)
        """
        padding_len = self._block_size - (len(data) % self._block_size)
        padding_bytes = bytes([padding_len] * padding_len)
        return data + padding_bytes

    def unpad(self, data: bytes) -> bytes:
        """
        Remove PKCS#7 padding from data.

        Args:
            data: Padded data

        Returns:
            Original data without padding

        Raises:
            ValueError: If padding is invalid
        """
        if not data:
            raise PaddingError()

        if len(data) % self._block_size != 0:
            raise PaddingError()

        padding_len = data[-1]

        # Check length and every padding byte without early exit and without
        # distinguishing the failure mode. A caller-visible difference between
        # "bad length" and "bad bytes", or an early return on the first
        # mismatched byte, is a padding oracle: an attacker who can submit
        # tampered ciphertext (there is no MAC on the wire) recovers plaintext
        # byte by byte from the difference.
        valid = 1 if 1 <= padding_len <= self._block_size else 0
        # Compare a fixed number of trailing bytes regardless of padding_len.
        for i in range(1, self._block_size + 1):
            in_padding = 1 if i <= padding_len else 0
            byte_matches = 1 if (i <= len(data) and data[-i] == padding_len) else 0
            # Only bytes claimed to be padding must match.
            valid &= byte_matches | (1 - in_padding)

        if not valid:
            raise PaddingError()

        return data[:-padding_len]

    def is_valid_padding(self, data: bytes) -> bool:
        """
        Check if data has valid PKCS#7 padding.

        Args:
            data: Data to check

        Returns:
            True if padding is valid, False otherwise
        """
        try:
            self.unpad(data)
            return True
        except ValueError:
            return False


class PKCS7Signature:
    """
    PKCS#7/CMS digital signature implementation.

    Used for signing electronic documents for Korean government systems.
    Supports:
    - RSA signatures with SHA-256
    - Certificate-based signing
    - Signature verification
    """

    def __init__(
        self,
        private_key_path: Optional[str] = None,
        certificate_path: Optional[str] = None,
        private_key_password: Optional[bytes] = None,
    ):
        """
        Initialize PKCS7 signature handler.

        Args:
            private_key_path: Path to private key file (PEM format)
            certificate_path: Path to certificate file (PEM format)
            private_key_password: Password for encrypted private key
        """
        self._private_key = None
        self._certificate = None
        # Certificate of the counterparty (공단). Separate from `_certificate`,
        # which is ours: our own certificate must never be used to verify a
        # signature that is supposed to have come from someone else.
        self._peer_certificate = None

        if private_key_path:
            self._load_private_key(private_key_path, private_key_password)

        if certificate_path:
            self._load_certificate(certificate_path)

    def _load_private_key(self, path: str, password: Optional[bytes] = None) -> None:
        """Load private key from PEM file."""
        with open(path, "rb") as f:
            self._private_key = serialization.load_pem_private_key(
                f.read(),
                password=password,
            )

    def _load_certificate(self, path: str) -> None:
        """Load certificate from PEM file."""
        with open(path, "rb") as f:
            self._certificate = x509.load_pem_x509_certificate(f.read())

    def load_private_key_bytes(
        self,
        key_data: bytes,
        password: Optional[bytes] = None,
    ) -> None:
        """
        Load private key from bytes.

        Args:
            key_data: PEM-encoded private key
            password: Password if key is encrypted
        """
        self._private_key = serialization.load_pem_private_key(
            key_data,
            password=password,
        )

    def load_certificate_bytes(self, cert_data: bytes) -> None:
        """
        Load certificate from bytes.

        Args:
            cert_data: PEM or DER-encoded certificate
        """
        try:
            self._certificate = x509.load_pem_x509_certificate(cert_data)
        except ValueError:
            self._certificate = x509.load_der_x509_certificate(cert_data)

    def sign(self, data: bytes) -> bytes:
        """
        Create PKCS#7 signed data.

        Args:
            data: Data to sign

        Returns:
            PKCS#7 signed message (DER-encoded)

        Raises:
            ValueError: If private key or certificate not loaded
        """
        if not self._private_key:
            raise ValueError("Private key not loaded")
        if not self._certificate:
            raise ValueError("Certificate not loaded")

        # Build PKCS7 signed data
        builder = (
            pkcs7.PKCS7SignatureBuilder()
            .set_data(data)
            .add_signer(
                self._certificate,
                self._private_key,
                hashes.SHA256(),
            )
        )

        return builder.sign(
            serialization.Encoding.DER,
            [pkcs7.PKCS7Options.DetachedSignature],
        )

    def sign_with_content(self, data: bytes) -> bytes:
        """
        Create PKCS#7 signed data with embedded content.

        Args:
            data: Data to sign

        Returns:
            PKCS#7 signed message with content (DER-encoded)
        """
        if not self._private_key:
            raise ValueError("Private key not loaded")
        if not self._certificate:
            raise ValueError("Certificate not loaded")

        builder = (
            pkcs7.PKCS7SignatureBuilder()
            .set_data(data)
            .add_signer(
                self._certificate,
                self._private_key,
                hashes.SHA256(),
            )
        )

        return builder.sign(serialization.Encoding.DER, [])

    def sign_raw(self, data: bytes) -> bytes:
        """
        Create raw RSA signature (not PKCS#7 wrapped).

        Args:
            data: Data to sign

        Returns:
            Raw signature bytes
        """
        if not self._private_key:
            raise ValueError("Private key not loaded")

        return self._private_key.sign(
            data,
            padding.PKCS1v15(),
            hashes.SHA256(),
        )

    def load_peer_certificate(self, path: str) -> None:
        """
        Load the counterparty's signing certificate into the trust store.

        Responses must be verified against the *peer's* certificate, not ours.
        Verifying with `self._certificate` (our own signing certificate) only
        proves that we could have produced the signature, which is no proof at
        all about the peer.

        Args:
            path: Path to the peer certificate (PEM or DER)

        Raises:
            ValueError: If the certificate cannot be parsed
        """
        with open(path, "rb") as f:
            cert_data = f.read()
        try:
            self._peer_certificate = x509.load_pem_x509_certificate(cert_data)
        except ValueError:
            self._peer_certificate = x509.load_der_x509_certificate(cert_data)

    def load_peer_certificate_bytes(self, cert_data: bytes) -> None:
        """
        Load the counterparty's signing certificate from bytes.

        Args:
            cert_data: PEM or DER-encoded certificate
        """
        try:
            self._peer_certificate = x509.load_pem_x509_certificate(cert_data)
        except ValueError:
            self._peer_certificate = x509.load_der_x509_certificate(cert_data)

    @property
    def can_verify(self) -> bool:
        """Whether a peer certificate is available to verify incoming signatures."""
        return self._peer_certificate is not None

    @property
    def can_sign(self) -> bool:
        """Whether a private key is available to produce signatures."""
        return self._private_key is not None

    def verify_raw(self, data: bytes, signature: bytes, public_key=None) -> bool:
        """
        Verify a raw RSA signature against the peer's certificate.

        Args:
            data: Original data
            signature: Signature to verify
            public_key: Explicit public key; otherwise the loaded peer certificate

        Returns:
            True if the signature is valid, False otherwise

        Raises:
            VerificationUnavailableError: If no peer certificate/public key is
                configured. This is deliberately not `return False` and not
                `return True` -- the caller must be able to tell "the peer's
                signature is wrong" apart from "we are unable to check", and
                must refuse the message in both cases.
        """
        if public_key is None:
            if self._peer_certificate is None:
                raise VerificationUnavailableError(
                    "No peer certificate loaded; cannot verify the counterparty's "
                    "signature. Configure the issuing agency's certificate via "
                    "load_peer_certificate(). Verifying against our own signing "
                    "certificate would prove nothing."
                )
            if not self._is_certificate_currently_valid(self._peer_certificate):
                raise VerificationUnavailableError(
                    "Peer certificate is outside its validity period; refusing to "
                    "verify against it."
                )
            public_key = self._peer_certificate.public_key()

        try:
            public_key.verify(
                signature,
                data,
                padding.PKCS1v15(),
                hashes.SHA256(),
            )
            return True
        except Exception:
            return False

    @staticmethod
    def _is_certificate_currently_valid(certificate) -> bool:
        """Check a certificate's validity window against the current UTC time."""
        now = datetime.now(timezone.utc)
        return (
            certificate.not_valid_before_utc <= now <= certificate.not_valid_after_utc
        )

    @property
    def certificate_info(self) -> dict:
        """
        Get certificate information.

        Returns:
            Dictionary with certificate details
        """
        if not self._certificate:
            return {}

        subject = self._certificate.subject
        issuer = self._certificate.issuer

        return {
            "subject": {
                "common_name": self._get_name_attribute(subject, NameOID.COMMON_NAME),
                "organization": self._get_name_attribute(subject, NameOID.ORGANIZATION_NAME),
                "country": self._get_name_attribute(subject, NameOID.COUNTRY_NAME),
            },
            "issuer": {
                "common_name": self._get_name_attribute(issuer, NameOID.COMMON_NAME),
                "organization": self._get_name_attribute(issuer, NameOID.ORGANIZATION_NAME),
            },
            "serial_number": str(self._certificate.serial_number),
            "not_valid_before": self._certificate.not_valid_before_utc.isoformat(),
            "not_valid_after": self._certificate.not_valid_after_utc.isoformat(),
        }

    def _get_name_attribute(self, name, oid) -> Optional[str]:
        """Extract attribute from X.509 name."""
        try:
            return name.get_attributes_for_oid(oid)[0].value
        except (IndexError, AttributeError):
            return None

    def is_certificate_valid(self) -> bool:
        """
        Check if certificate is currently valid.

        Returns:
            True if certificate is within validity period
        """
        if not self._certificate:
            return False

        now = datetime.now(timezone.utc)
        return (
            self._certificate.not_valid_before_utc <= now
            <= self._certificate.not_valid_after_utc
        )


def generate_test_keypair() -> tuple[bytes, bytes]:
    """
    Generate a test RSA key pair and self-signed certificate.

    Returns:
        Tuple of (private_key_pem, certificate_pem)

    Note: For testing only. Do not use in production.
    """
    from cryptography.x509.oid import NameOID
    from cryptography.hazmat.primitives.asymmetric import rsa
    from datetime import timedelta

    # Generate private key
    private_key = rsa.generate_private_key(
        public_exponent=65537,
        key_size=2048,
    )

    # Create self-signed certificate
    subject = issuer = x509.Name([
        x509.NameAttribute(NameOID.COUNTRY_NAME, "KR"),
        x509.NameAttribute(NameOID.ORGANIZATION_NAME, "Test Organization"),
        x509.NameAttribute(NameOID.COMMON_NAME, "Test Certificate"),
    ])

    certificate = (
        x509.CertificateBuilder()
        .subject_name(subject)
        .issuer_name(issuer)
        .public_key(private_key.public_key())
        .serial_number(x509.random_serial_number())
        .not_valid_before(datetime.now(timezone.utc))
        .not_valid_after(datetime.now(timezone.utc) + timedelta(days=365))
        .sign(private_key, hashes.SHA256())
    )

    private_key_pem = private_key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.PKCS8,
        encryption_algorithm=serialization.NoEncryption(),
    )

    certificate_pem = certificate.public_bytes(serialization.Encoding.PEM)

    return private_key_pem, certificate_pem
