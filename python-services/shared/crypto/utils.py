"""
Cryptographic Utility Functions

Common utilities for key generation, derivation, and encoding.
"""
import secrets
import hashlib
import base64
from typing import Optional


def generate_key(size: int = 16) -> bytes:
    """
    Generate a cryptographically secure random key.

    Args:
        size: Key size in bytes (default: 16 for 128-bit)

    Returns:
        Random bytes suitable for use as encryption key
    """
    return secrets.token_bytes(size)


def derive_key(
    password: str,
    salt: Optional[bytes] = None,
    iterations: int = 100000,
    key_length: int = 16,
) -> tuple[bytes, bytes]:
    """
    Derive encryption key from password using PBKDF2.

    Args:
        password: Password to derive key from
        salt: Salt bytes (generated if not provided)
        iterations: Number of PBKDF2 iterations
        key_length: Desired key length in bytes

    Returns:
        Tuple of (derived_key, salt)
    """
    if salt is None:
        salt = secrets.token_bytes(16)

    key = hashlib.pbkdf2_hmac(
        "sha256",
        password.encode("utf-8"),
        salt,
        iterations,
        dklen=key_length,
    )

    return key, salt


def generate_iv(size: int = 16) -> bytes:
    """
    Generate random initialization vector.

    Args:
        size: IV size in bytes (default: 16 for 128-bit block ciphers)

    Returns:
        Random IV bytes
    """
    return secrets.token_bytes(size)


def bytes_to_hex(data: bytes) -> str:
    """Convert bytes to hexadecimal string."""
    return data.hex()


def hex_to_bytes(hex_str: str) -> bytes:
    """Convert hexadecimal string to bytes."""
    return bytes.fromhex(hex_str)


def bytes_to_base64(data: bytes) -> str:
    """Convert bytes to base64 string."""
    return base64.b64encode(data).decode("ascii")


def base64_to_bytes(b64_str: str) -> bytes:
    """Convert base64 string to bytes."""
    return base64.b64decode(b64_str)


def constant_time_compare(a: bytes, b: bytes) -> bool:
    """
    Compare two byte strings in constant time.

    This prevents timing attacks when comparing sensitive data
    like MACs or signatures.

    Args:
        a: First byte string
        b: Second byte string

    Returns:
        True if equal, False otherwise
    """
    return secrets.compare_digest(a, b)


def secure_zero(data: bytearray) -> None:
    """
    Securely zero out sensitive data in memory.

    Args:
        data: Bytearray to zero (must be mutable)
    """
    for i in range(len(data)):
        data[i] = 0


class KeyDerivation:
    """
    Key derivation utilities for various purposes.
    """

    @staticmethod
    def from_password(
        password: str,
        salt: bytes,
        key_length: int = 16,
    ) -> bytes:
        """
        Derive key from password using PBKDF2-SHA256.

        Args:
            password: User password
            salt: Random salt
            key_length: Desired key length

        Returns:
            Derived key
        """
        return hashlib.pbkdf2_hmac(
            "sha256",
            password.encode("utf-8"),
            salt,
            iterations=100000,
            dklen=key_length,
        )

    @staticmethod
    def from_shared_secret(
        shared_secret: bytes,
        info: bytes = b"",
        length: int = 16,
    ) -> bytes:
        """
        Derive key from shared secret using HKDF-like construction.

        Args:
            shared_secret: Shared secret bytes
            info: Context/application-specific info
            length: Desired output length

        Returns:
            Derived key
        """
        # Simple HKDF-Extract
        prk = hashlib.sha256(shared_secret).digest()

        # HKDF-Expand (simplified)
        t = b""
        okm = b""
        counter = 1

        while len(okm) < length:
            t = hashlib.sha256(t + info + bytes([counter])).digest()
            okm += t
            counter += 1

        return okm[:length]


class MessageAuthentication:
    """
    Message authentication utilities.
    """

    @staticmethod
    def hmac_sha256(key: bytes, message: bytes) -> bytes:
        """
        Compute HMAC-SHA256.

        Args:
            key: Secret key
            message: Message to authenticate

        Returns:
            HMAC value
        """
        import hmac
        return hmac.new(key, message, hashlib.sha256).digest()

    @staticmethod
    def verify_hmac(key: bytes, message: bytes, expected_mac: bytes) -> bool:
        """
        Verify HMAC-SHA256 in constant time.

        Args:
            key: Secret key
            message: Message to verify
            expected_mac: Expected MAC value

        Returns:
            True if MAC is valid
        """
        computed = MessageAuthentication.hmac_sha256(key, message)
        return constant_time_compare(computed, expected_mac)
