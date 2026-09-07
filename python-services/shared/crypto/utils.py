"""
Cryptographic Utility Functions

Common utilities for key generation, derivation, and encoding.
"""
import secrets
import hashlib
import hmac
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


class WeakKeyError(ValueError):
    """Raised when a configured key is missing, malformed, or trivially weak."""


# Placeholder values that have appeared in example env files / docs and must
# never be accepted as a real key.
_PLACEHOLDER_KEYS = frozenset(
    {
        "changeme",
        "change-me",
        "placeholder",
        "your-key-here",
        "test",
        "testkey",
        "secret",
        "example",
    }
)


def load_symmetric_key(
    key_hex: Optional[str],
    *,
    name: str,
    allowed_sizes: tuple[int, ...] = (16, 24, 32),
) -> bytes:
    """
    Parse and validate a hex-encoded symmetric key, failing closed.

    This is the counterpart of the Go-side JWT secret hardening: a missing or
    obviously weak key must stop the process rather than silently degrade to a
    known constant (e.g. an all-zero key).

    Args:
        key_hex: Hex-encoded key material from configuration
        name: Setting name, used in the error message (e.g. "ARIA_ENCRYPTION_KEY")
        allowed_sizes: Acceptable key lengths in bytes

    Returns:
        Decoded key bytes

    Raises:
        WeakKeyError: If the key is absent, malformed, the wrong size, or weak
    """
    if key_hex is None or not key_hex.strip():
        raise WeakKeyError(
            f"{name} is not set. Refusing to start: a missing key would "
            f"otherwise fall back to a publicly known constant. "
            f"Generate one with: python -c "
            f"\"import secrets; print(secrets.token_bytes(16).hex())\""
        )

    candidate = key_hex.strip()

    if candidate.lower() in _PLACEHOLDER_KEYS:
        raise WeakKeyError(f"{name} is set to a placeholder value. Set a real key.")

    try:
        key = bytes.fromhex(candidate)
    except ValueError as exc:
        raise WeakKeyError(f"{name} must be hex-encoded: {exc}") from None

    if len(key) not in allowed_sizes:
        sizes = "/".join(str(s * 8) for s in allowed_sizes)
        raise WeakKeyError(
            f"{name} must decode to {sizes} bits, got {len(key) * 8} bits"
        )

    assert_strong_key(key, name=name)
    return key


def assert_strong_key(key: bytes, *, name: str) -> None:
    """
    Reject trivially weak key material.

    Catches the specific failure modes seen in this codebase: all-zero keys used
    as a "development placeholder", and single-byte repeats.

    Args:
        key: Key bytes to check
        name: Setting name, used in the error message

    Raises:
        WeakKeyError: If the key is all-zero or a single repeated byte
    """
    if not key:
        raise WeakKeyError(f"{name} is empty")

    if len(set(key)) == 1:
        raise WeakKeyError(
            f"{name} is a single repeated byte (0x{key[0]:02x}) and provides no "
            f"confidentiality. Use a random key."
        )


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
        salt: Optional[bytes] = None,
    ) -> bytes:
        """
        Derive key from shared secret using HKDF-SHA256 (RFC 5869).

        Args:
            shared_secret: Shared secret bytes (IKM)
            info: Context/application-specific info
            length: Desired output length in bytes
            salt: Optional salt; RFC 5869 uses HashLen zero bytes when absent

        Returns:
            Derived key of `length` bytes

        Raises:
            ValueError: If inputs are empty or `length` is out of range
        """
        if not shared_secret:
            raise ValueError("shared_secret must not be empty")

        hash_len = hashlib.sha256().digest_size

        if length <= 0:
            raise ValueError("length must be positive")
        # RFC 5869 section 2.3: L <= 255 * HashLen
        if length > 255 * hash_len:
            raise ValueError(f"length must not exceed {255 * hash_len} bytes")

        # HKDF-Extract: PRK = HMAC-Hash(salt, IKM)
        if salt is None:
            salt = bytes(hash_len)
        prk = hmac.new(salt, shared_secret, hashlib.sha256).digest()

        # HKDF-Expand: T(n) = HMAC-Hash(PRK, T(n-1) | info | n)
        t = b""
        okm = bytearray()
        counter = 1

        while len(okm) < length:
            t = hmac.new(prk, t + info + bytes([counter]), hashlib.sha256).digest()
            okm.extend(t)
            counter += 1

        return bytes(okm[:length])


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
