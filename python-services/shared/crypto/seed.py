"""
AES-CBC helper, and an explicit placeholder for the (unimplemented) SEED cipher.

SEED is a block cipher developed by the Korea Internet & Security Agency (KISA)
and is required by some Korean government systems, including parts of the
National Tax Service (Hometax) integration.

This module does NOT implement SEED. It never did: the previous version of this
file was named `SEEDCipher` but ran AES underneath. That naming is dangerous --
callers reasonably believe they are producing SEED ciphertext, so any peer that
actually expects SEED silently fails to decrypt, and any compliance claim about
"SEED encryption" made on the basis of this module is false.

What this module provides instead:

- `AESCBCCipher`: an honestly named AES-128/192/256-CBC helper with a random,
  prepended IV. Use it wherever AES is acceptable.
- `SEEDCipher`: raises `SEEDNotImplementedError` on construction. It exists so
  that code (and tests) referring to SEED fail loudly instead of quietly
  encrypting with the wrong algorithm.

To implement SEED for real, follow TTAS.KO-12.0004/R1 (KS X 1213) or link the
official KISA reference implementation: https://seed.kisa.or.kr/
"""
import base64
import secrets
from typing import Optional


def _load_aes():
    """
    Import pycryptodome's AES lazily.

    Kept out of module scope so that importing `shared.crypto` -- which the
    ARIA/EDI path does on every start -- does not hard-require pycryptodome.
    Only callers that actually use AES pay for it.

    Returns:
        Tuple of (AES module, pad, unpad)

    Raises:
        RuntimeError: If pycryptodome is unavailable
    """
    try:
        from Crypto.Cipher import AES
        from Crypto.Util.Padding import pad, unpad
    except Exception as exc:  # ImportError, or a broken native build
        raise RuntimeError(
            f"pycryptodome is required for AESCBCCipher but could not be loaded: {exc}"
        ) from exc
    return AES, pad, unpad


class SEEDNotImplementedError(NotImplementedError):
    """Raised when SEED encryption is requested but no SEED implementation exists."""


class AESCBCCipher:
    """
    AES-CBC cipher with PKCS#7 padding and a per-message random IV.

    This is AES, not SEED. Do not use it where a counterpart requires SEED.
    """

    BLOCK_SIZE = 16  # 128 bits
    KEY_SIZES = (16, 24, 32)

    def __init__(self, key: bytes) -> None:
        """
        Initialize the cipher with a key.

        Args:
            key: 16, 24, or 32 byte key

        Raises:
            ValueError: If the key length is unsupported or the key is trivially weak
        """
        if len(key) not in self.KEY_SIZES:
            raise ValueError(
                f"Key must be one of {self.KEY_SIZES} bytes, got {len(key)}"
            )
        if len(set(key)) == 1:
            raise ValueError(
                "Key is a single repeated byte and provides no confidentiality"
            )
        self._key = key

    @classmethod
    def from_base64_key(cls, key_b64: str) -> "AESCBCCipher":
        """
        Create a cipher from a base64-encoded key.

        Args:
            key_b64: Base64-encoded key string

        Returns:
            AESCBCCipher instance
        """
        return cls(base64.b64decode(key_b64))

    @classmethod
    def from_hex_key(cls, key_hex: str) -> "AESCBCCipher":
        """
        Create a cipher from a hex-encoded key.

        Args:
            key_hex: Hex-encoded key string

        Returns:
            AESCBCCipher instance
        """
        return cls(bytes.fromhex(key_hex))

    def encrypt(self, plaintext: bytes, iv: Optional[bytes] = None) -> tuple[bytes, bytes]:
        """
        Encrypt data using AES-CBC.

        Args:
            plaintext: Data to encrypt
            iv: Initialization vector (16 bytes). A fresh random IV is generated
                when omitted -- pass one only to reproduce a known ciphertext.

        Returns:
            Tuple of (ciphertext, iv)
        """
        if iv is None:
            iv = secrets.token_bytes(self.BLOCK_SIZE)
        elif len(iv) != self.BLOCK_SIZE:
            raise ValueError(f"IV must be {self.BLOCK_SIZE} bytes")

        AES, pad, _ = _load_aes()
        cipher = AES.new(self._key, AES.MODE_CBC, iv)
        ciphertext = cipher.encrypt(pad(plaintext, self.BLOCK_SIZE))
        return ciphertext, iv

    def decrypt(self, ciphertext: bytes, iv: bytes) -> bytes:
        """
        Decrypt data using AES-CBC.

        Args:
            ciphertext: Encrypted data
            iv: Initialization vector used for encryption

        Returns:
            Decrypted plaintext
        """
        if len(iv) != self.BLOCK_SIZE:
            raise ValueError(f"IV must be {self.BLOCK_SIZE} bytes")

        AES, _, unpad = _load_aes()
        cipher = AES.new(self._key, AES.MODE_CBC, iv)
        return unpad(cipher.decrypt(ciphertext), self.BLOCK_SIZE)

    def seal(self, plaintext: bytes) -> bytes:
        """
        Encrypt with a fresh IV, prepending the IV to the ciphertext.

        Args:
            plaintext: Data to encrypt

        Returns:
            iv (16 bytes) + ciphertext
        """
        ciphertext, iv = self.encrypt(plaintext)
        return iv + ciphertext

    def unseal(self, data: bytes) -> bytes:
        """
        Decrypt a message produced by `seal()`.

        Args:
            data: iv (16 bytes) + ciphertext

        Returns:
            Decrypted plaintext

        Raises:
            ValueError: If the message is too short to carry an IV and one block
        """
        if len(data) < self.BLOCK_SIZE * 2:
            raise ValueError("Ciphertext too short")
        return self.decrypt(data[self.BLOCK_SIZE:], data[:self.BLOCK_SIZE])

    def encrypt_base64(self, plaintext: str, iv: Optional[bytes] = None) -> tuple[str, str]:
        """
        Encrypt a string and return base64-encoded ciphertext and IV.

        Args:
            plaintext: String to encrypt (UTF-8)
            iv: Optional initialization vector

        Returns:
            Tuple of (base64_ciphertext, base64_iv)
        """
        ciphertext, used_iv = self.encrypt(plaintext.encode("utf-8"), iv)
        return (
            base64.b64encode(ciphertext).decode("ascii"),
            base64.b64encode(used_iv).decode("ascii"),
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
        plaintext = self.decrypt(
            base64.b64decode(ciphertext_b64),
            base64.b64decode(iv_b64),
        )
        return plaintext.decode("utf-8")


class SEEDCipher:
    """
    Placeholder for the SEED block cipher (KS X 1213 / TTAS.KO-12.0004/R1).

    Not implemented. Constructing this class raises, by design: an earlier
    version silently used AES here, which would produce ciphertext no SEED peer
    can read while appearing to work.

    Use `AESCBCCipher` when AES is acceptable, or link the KISA reference
    implementation when SEED is genuinely required.
    """

    BLOCK_SIZE = 16
    KEY_SIZE = 16

    def __init__(self, key: bytes) -> None:
        """Always raises; SEED is not implemented in this codebase."""
        raise SEEDNotImplementedError(
            "SEED (KS X 1213 / TTAS.KO-12.0004/R1) is not implemented. "
            "AES is not a substitute -- a SEED peer cannot decrypt AES ciphertext. "
            "Use AESCBCCipher where AES is acceptable, or link the KISA SEED "
            "reference implementation (https://seed.kisa.or.kr/)."
        )

    @classmethod
    def from_base64_key(cls, key_b64: str) -> "SEEDCipher":
        """Always raises; SEED is not implemented."""
        return cls(base64.b64decode(key_b64))

    @classmethod
    def from_hex_key(cls, key_hex: str) -> "SEEDCipher":
        """Always raises; SEED is not implemented."""
        return cls(bytes.fromhex(key_hex))


def generate_aes_key(size: int = 16) -> bytes:
    """
    Generate a random AES key.

    Args:
        size: Key size in bytes (16, 24, or 32)

    Returns:
        Random key bytes
    """
    if size not in AESCBCCipher.KEY_SIZES:
        raise ValueError(f"Key size must be one of {AESCBCCipher.KEY_SIZES}")
    return secrets.token_bytes(size)
