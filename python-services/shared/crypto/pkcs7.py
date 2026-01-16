"""
PKCS#7 Padding and Signature Implementation

PKCS#7 is used for:
1. Block cipher padding (PKCS#7 padding scheme)
2. Cryptographic Message Syntax (CMS/PKCS#7) for digital signatures

Required for 4대보험 EDI communication and electronic document signing.
"""
import hashlib
from typing import Union, Optional
from datetime import datetime

from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa, padding
from cryptography.hazmat.primitives.serialization import pkcs7
from cryptography.x509.oid import NameOID


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
            raise ValueError("Data is empty")

        if len(data) % self._block_size != 0:
            raise ValueError("Data length is not a multiple of block size")

        padding_len = data[-1]

        if padding_len == 0 or padding_len > self._block_size:
            raise ValueError(f"Invalid padding length: {padding_len}")

        # Verify all padding bytes
        for i in range(1, padding_len + 1):
            if data[-i] != padding_len:
                raise ValueError("Invalid padding bytes")

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

    def verify_raw(self, data: bytes, signature: bytes, public_key=None) -> bool:
        """
        Verify raw RSA signature.

        Args:
            data: Original data
            signature: Signature to verify
            public_key: Public key (uses certificate's key if not provided)

        Returns:
            True if signature is valid
        """
        if public_key is None:
            if not self._certificate:
                raise ValueError("Certificate not loaded")
            public_key = self._certificate.public_key()

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

        now = datetime.utcnow()
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
        .not_valid_before(datetime.utcnow())
        .not_valid_after(datetime.utcnow() + timedelta(days=365))
        .sign(private_key, hashes.SHA256())
    )

    private_key_pem = private_key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.PKCS8,
        encryption_algorithm=serialization.NoEncryption(),
    )

    certificate_pem = certificate.public_bytes(serialization.Encoding.PEM)

    return private_key_pem, certificate_pem
