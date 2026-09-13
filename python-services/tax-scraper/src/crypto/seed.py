"""AES-CBC helper, and an explicit placeholder for the (unimplemented) SEED cipher.

SEED is a symmetric block cipher developed by KISA (Korea Internet & Security
Agency) with 128-bit keys and 128-bit blocks. Some Korean government interfaces
require it.

This module does NOT implement SEED. The previous version of this file exported
a class named `SEEDCipher` that ran AES underneath, which is worse than having
nothing: callers believe they hold SEED ciphertext, the counterpart cannot
decrypt it, and any "SEED encryption" compliance claim resting on this module is
false.

- `AESCBCCipher` is the honestly named AES-CBC helper (random IV, prepended).
- `SEEDCipher` raises on construction so SEED requirements fail loudly.

Implementing SEED for real means following TTAS.KO-12.0004/R1 (KS X 1213) or
linking the KISA reference implementation: https://seed.kisa.or.kr/
"""

import secrets
from typing import ClassVar

from loguru import logger


def _load_pycryptodome():
    """Import pycryptodome lazily.

    Kept out of module scope so that importing this module -- and in particular
    reaching the SEEDCipher refusal below -- does not require pycryptodome. Only
    callers that actually run AES pay for it.

    Returns:
        Tuple of (AES, pad, unpad, PBKDF2, get_random_bytes)

    Raises:
        RuntimeError: If pycryptodome is unavailable or its native build is
            unusable on this platform
    """
    try:
        from Crypto.Cipher import AES
        from Crypto.Protocol.KDF import PBKDF2
        from Crypto.Random import get_random_bytes
        from Crypto.Util.Padding import pad, unpad
    except Exception as exc:  # ImportError, or a broken native build
        raise RuntimeError(
            f"pycryptodome is required for AES operations but could not be loaded: {exc}"
        ) from exc
    return AES, pad, unpad, PBKDF2, get_random_bytes


class SEEDNotImplementedError(NotImplementedError):
    """Raised when SEED encryption is requested but no SEED implementation exists."""


class AESCBCCipher:
    """AES-CBC cipher with PKCS#7 padding and a per-message random IV.

    This is AES, not SEED. Do not use it where the counterpart requires SEED.
    """

    BLOCK_SIZE: ClassVar[int] = 16  # 128 bits
    KEY_SIZE: ClassVar[int] = 16  # 128 bits
    KEY_SIZES: ClassVar[tuple[int, ...]] = (16, 24, 32)

    def __init__(self, key: bytes) -> None:
        """Initialize the cipher with a key.

        Args:
            key: 16, 24, or 32 byte encryption key

        Raises:
            ValueError: If the key length is unsupported or the key is trivially weak
        """
        if len(key) not in self.KEY_SIZES:
            raise ValueError(f"Key must be one of {self.KEY_SIZES} bytes, got {len(key)}")
        if len(set(key)) == 1:
            raise ValueError("Key is a single repeated byte and provides no confidentiality")
        self._key = key

    @classmethod
    def from_hex(cls, hex_key: str) -> "AESCBCCipher":
        """Create cipher from hex-encoded key."""
        return cls(bytes.fromhex(hex_key))

    def encrypt(self, plaintext: bytes) -> bytes:
        """Encrypt plaintext using AES-CBC with a fresh random IV.

        Args:
            plaintext: Data to encrypt

        Returns:
            IV + ciphertext (IV is prepended)
        """
        AES, pad, _, _, get_random_bytes = _load_pycryptodome()

        iv = get_random_bytes(self.BLOCK_SIZE)
        cipher = AES.new(self._key, AES.MODE_CBC, iv)
        ciphertext = cipher.encrypt(pad(plaintext, self.BLOCK_SIZE))

        logger.debug(f"Encrypted {len(plaintext)} bytes")
        return iv + ciphertext

    def decrypt(self, ciphertext: bytes) -> bytes:
        """Decrypt ciphertext using AES-CBC.

        Args:
            ciphertext: IV + encrypted data

        Returns:
            Decrypted plaintext
        """
        if len(ciphertext) < self.BLOCK_SIZE * 2:
            raise ValueError("Ciphertext too short")

        AES, _, unpad, _, _ = _load_pycryptodome()

        iv = ciphertext[: self.BLOCK_SIZE]
        encrypted = ciphertext[self.BLOCK_SIZE :]

        cipher = AES.new(self._key, AES.MODE_CBC, iv)
        plaintext = unpad(cipher.decrypt(encrypted), self.BLOCK_SIZE)

        logger.debug(f"Decrypted {len(plaintext)} bytes")
        return plaintext

    def encrypt_string(self, text: str, encoding: str = "utf-8") -> bytes:
        """Encrypt string."""
        return self.encrypt(text.encode(encoding))

    def decrypt_string(self, ciphertext: bytes, encoding: str = "utf-8") -> str:
        """Decrypt to string."""
        return self.decrypt(ciphertext).decode(encoding)


class SEEDCipher:
    """Placeholder for the SEED block cipher (KS X 1213 / TTAS.KO-12.0004/R1).

    Not implemented. Constructing this class raises, by design.
    """

    BLOCK_SIZE: ClassVar[int] = 16
    KEY_SIZE: ClassVar[int] = 16

    def __init__(self, key: bytes) -> None:
        """Always raises; SEED is not implemented in this codebase."""
        raise SEEDNotImplementedError(
            "SEED (KS X 1213 / TTAS.KO-12.0004/R1) is not implemented. "
            "AES is not a substitute -- a SEED peer cannot decrypt AES ciphertext. "
            "Use AESCBCCipher where AES is acceptable, or link the KISA SEED "
            "reference implementation (https://seed.kisa.or.kr/)."
        )

    @classmethod
    def from_hex(cls, hex_key: str) -> "SEEDCipher":
        """Always raises; SEED is not implemented."""
        return cls(bytes.fromhex(hex_key))


def generate_aes_key() -> bytes:
    """Generate a random AES key."""
    return secrets.token_bytes(AESCBCCipher.KEY_SIZE)


def derive_key_from_password(password: str, salt: bytes | None = None) -> tuple[bytes, bytes]:
    """Derive an AES key from a password using PBKDF2-HMAC-SHA256.

    Args:
        password: Password string
        salt: Optional salt (generated if not provided)

    Returns:
        Tuple of (key, salt)
    """
    _, _, _, PBKDF2, _ = _load_pycryptodome()

    if salt is None:
        salt = secrets.token_bytes(16)

    key = PBKDF2(password, salt, dkLen=AESCBCCipher.KEY_SIZE, count=100000)
    return key, salt
