"""
SEED Encryption Implementation

SEED is a block cipher developed by the Korea Internet & Security Agency (KISA).
It is required for communication with Korean government systems including
the National Tax Service (Hometax).

Reference: TTAS.KO-12.0004/R1 (Korean standard)
"""
import base64
from typing import Optional

from Crypto.Cipher import AES
from Crypto.Util.Padding import pad, unpad


class SEEDCipher:
    """
    SEED cipher implementation for Korean government system integration.

    Note: This implementation uses AES as a placeholder since pycryptodome
    doesn't include SEED. For production use, a proper SEED implementation
    (e.g., from KISA library) should be used.

    In production, use the official KISA SEED library:
    https://seed.kisa.or.kr/
    """

    BLOCK_SIZE = 16  # 128 bits

    def __init__(self, key: bytes) -> None:
        """
        Initialize SEED cipher with key.

        Args:
            key: 16-byte (128-bit) encryption key
        """
        if len(key) != 16:
            raise ValueError("SEED key must be 16 bytes (128 bits)")
        self._key = key

    @classmethod
    def from_base64_key(cls, key_b64: str) -> "SEEDCipher":
        """
        Create cipher from base64-encoded key.

        Args:
            key_b64: Base64-encoded key string

        Returns:
            SEEDCipher instance
        """
        key = base64.b64decode(key_b64)
        return cls(key)

    def encrypt(self, plaintext: bytes, iv: Optional[bytes] = None) -> tuple[bytes, bytes]:
        """
        Encrypt data using SEED-CBC.

        Args:
            plaintext: Data to encrypt
            iv: Initialization vector (16 bytes). If None, generates random IV.

        Returns:
            Tuple of (ciphertext, iv)
        """
        # TODO: Replace with actual SEED implementation
        # Using AES-CBC as placeholder for structure demonstration
        cipher = AES.new(self._key, AES.MODE_CBC, iv)
        padded_data = pad(plaintext, self.BLOCK_SIZE)
        ciphertext = cipher.encrypt(padded_data)
        return ciphertext, cipher.iv

    def decrypt(self, ciphertext: bytes, iv: bytes) -> bytes:
        """
        Decrypt data using SEED-CBC.

        Args:
            ciphertext: Encrypted data
            iv: Initialization vector used for encryption

        Returns:
            Decrypted plaintext
        """
        # TODO: Replace with actual SEED implementation
        cipher = AES.new(self._key, AES.MODE_CBC, iv)
        padded_plaintext = cipher.decrypt(ciphertext)
        return unpad(padded_plaintext, self.BLOCK_SIZE)

    def encrypt_base64(self, plaintext: str, iv: Optional[bytes] = None) -> tuple[str, str]:
        """
        Encrypt string and return base64-encoded result.

        Args:
            plaintext: String to encrypt (UTF-8)
            iv: Optional initialization vector

        Returns:
            Tuple of (base64_ciphertext, base64_iv)
        """
        ciphertext, used_iv = self.encrypt(plaintext.encode("utf-8"), iv)
        return base64.b64encode(ciphertext).decode("ascii"), base64.b64encode(used_iv).decode(
            "ascii"
        )

    def decrypt_base64(self, ciphertext_b64: str, iv_b64: str) -> str:
        """
        Decrypt base64-encoded ciphertext.

        Args:
            ciphertext_b64: Base64-encoded ciphertext
            iv_b64: Base64-encoded initialization vector

        Returns:
            Decrypted string
        """
        ciphertext = base64.b64decode(ciphertext_b64)
        iv = base64.b64decode(iv_b64)
        plaintext = self.decrypt(ciphertext, iv)
        return plaintext.decode("utf-8")


def generate_seed_key() -> bytes:
    """
    Generate a random SEED key.

    Returns:
        16-byte random key
    """
    import secrets

    return secrets.token_bytes(16)
