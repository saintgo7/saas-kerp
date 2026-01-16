"""SEED encryption implementation for Korean government systems.

SEED is a symmetric block cipher developed by KISA (Korea Internet & Security Agency).
It uses 128-bit keys and 128-bit blocks.
"""

from typing import ClassVar

from Crypto.Cipher import AES
from Crypto.Util.Padding import pad, unpad
from loguru import logger


class SEEDCipher:
    """SEED cipher implementation.

    Note: Python's pycryptodome doesn't have native SEED support.
    This implementation uses AES as a placeholder.
    For production, use pyseed or implement native SEED.
    """

    BLOCK_SIZE: ClassVar[int] = 16  # 128 bits
    KEY_SIZE: ClassVar[int] = 16  # 128 bits

    def __init__(self, key: bytes) -> None:
        """Initialize SEED cipher with key.

        Args:
            key: 16-byte (128-bit) encryption key
        """
        if len(key) != self.KEY_SIZE:
            raise ValueError(f"Key must be {self.KEY_SIZE} bytes")
        self._key = key

    @classmethod
    def from_hex(cls, hex_key: str) -> "SEEDCipher":
        """Create cipher from hex-encoded key."""
        return cls(bytes.fromhex(hex_key))

    def encrypt(self, plaintext: bytes) -> bytes:
        """Encrypt plaintext using SEED-CBC.

        Args:
            plaintext: Data to encrypt

        Returns:
            IV + ciphertext (IV is prepended)
        """
        from Crypto.Random import get_random_bytes

        iv = get_random_bytes(self.BLOCK_SIZE)

        # Using AES as placeholder for SEED
        # In production, replace with actual SEED implementation
        cipher = AES.new(self._key, AES.MODE_CBC, iv)
        padded = pad(plaintext, self.BLOCK_SIZE)
        ciphertext = cipher.encrypt(padded)

        logger.debug(f"Encrypted {len(plaintext)} bytes")
        return iv + ciphertext

    def decrypt(self, ciphertext: bytes) -> bytes:
        """Decrypt ciphertext using SEED-CBC.

        Args:
            ciphertext: IV + encrypted data

        Returns:
            Decrypted plaintext
        """
        if len(ciphertext) < self.BLOCK_SIZE * 2:
            raise ValueError("Ciphertext too short")

        iv = ciphertext[: self.BLOCK_SIZE]
        encrypted = ciphertext[self.BLOCK_SIZE :]

        # Using AES as placeholder for SEED
        cipher = AES.new(self._key, AES.MODE_CBC, iv)
        padded = cipher.decrypt(encrypted)
        plaintext = unpad(padded, self.BLOCK_SIZE)

        logger.debug(f"Decrypted {len(plaintext)} bytes")
        return plaintext

    def encrypt_string(self, text: str, encoding: str = "utf-8") -> bytes:
        """Encrypt string."""
        return self.encrypt(text.encode(encoding))

    def decrypt_string(self, ciphertext: bytes, encoding: str = "utf-8") -> str:
        """Decrypt to string."""
        return self.decrypt(ciphertext).decode(encoding)


# Korean government systems often use SEED-CBC with PKCS7 padding
# The actual SEED algorithm is more complex than AES
# For production use, consider:
# 1. pyseed library (if available)
# 2. Calling external SEED implementation via FFI
# 3. Implementing SEED algorithm from KISA specification


def generate_seed_key() -> bytes:
    """Generate a random SEED key."""
    from Crypto.Random import get_random_bytes

    return get_random_bytes(SEEDCipher.KEY_SIZE)


def derive_key_from_password(password: str, salt: bytes | None = None) -> tuple[bytes, bytes]:
    """Derive SEED key from password using PBKDF2.

    Args:
        password: Password string
        salt: Optional salt (generated if not provided)

    Returns:
        Tuple of (key, salt)
    """
    from Crypto.Protocol.KDF import PBKDF2
    from Crypto.Random import get_random_bytes

    if salt is None:
        salt = get_random_bytes(16)

    key = PBKDF2(password, salt, dkLen=SEEDCipher.KEY_SIZE, count=100000)
    return key, salt
