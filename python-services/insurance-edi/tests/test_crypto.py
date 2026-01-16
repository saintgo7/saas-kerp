"""
Tests for cryptographic modules.
"""
import pytest
import sys
from pathlib import Path

# Add shared module to path
sys.path.insert(0, str(Path(__file__).parent.parent.parent / "shared"))

from crypto import (
    ARIACipher,
    ARIAModeCBC,
    PKCS7Padding,
    generate_key,
    derive_key,
    generate_iv,
)


class TestARIACipher:
    """Test ARIA block cipher."""

    def test_encrypt_decrypt_block(self):
        """Test basic block encryption/decryption."""
        key = bytes(16)  # 128-bit zero key
        cipher = ARIACipher(key)

        plaintext = b"0123456789ABCDEF"  # 16 bytes
        ciphertext = cipher.encrypt_block(plaintext)

        assert ciphertext != plaintext
        assert len(ciphertext) == 16

        decrypted = cipher.decrypt_block(ciphertext)
        assert decrypted == plaintext

    def test_different_key_sizes(self):
        """Test 128, 192, and 256-bit keys."""
        plaintext = b"TestBlock1234567"

        for key_size in [16, 24, 32]:
            key = bytes(key_size)
            cipher = ARIACipher(key)

            ciphertext = cipher.encrypt_block(plaintext)
            decrypted = cipher.decrypt_block(ciphertext)

            assert decrypted == plaintext

    def test_invalid_key_size(self):
        """Test rejection of invalid key sizes."""
        with pytest.raises(ValueError):
            ARIACipher(bytes(10))

        with pytest.raises(ValueError):
            ARIACipher(bytes(20))

    def test_invalid_block_size(self):
        """Test rejection of invalid block sizes."""
        cipher = ARIACipher(bytes(16))

        with pytest.raises(ValueError):
            cipher.encrypt_block(b"short")

        with pytest.raises(ValueError):
            cipher.decrypt_block(b"too long block data")


class TestARIAModeCBC:
    """Test ARIA CBC mode."""

    def test_cbc_encrypt_decrypt(self):
        """Test CBC mode encryption/decryption."""
        key = generate_key(16)
        iv = generate_iv(16)

        cipher = ARIAModeCBC(key, iv)

        # Plaintext must be multiple of 16 bytes
        plaintext = b"Hello World!!!!!" * 4  # 64 bytes

        ciphertext = cipher.encrypt(plaintext)
        assert len(ciphertext) == len(plaintext)
        assert ciphertext != plaintext

        decrypted = cipher.decrypt(ciphertext)
        assert decrypted == plaintext

    def test_cbc_with_padding(self):
        """Test CBC mode with PKCS7 padding."""
        key = generate_key(16)
        iv = generate_iv(16)

        cipher = ARIAModeCBC(key, iv)
        padding = PKCS7Padding(16)

        plaintext = b"Hello World!"  # 12 bytes
        padded = padding.pad(plaintext)

        assert len(padded) == 16

        ciphertext = cipher.encrypt(padded)
        decrypted = cipher.decrypt(ciphertext)
        unpadded = padding.unpad(decrypted)

        assert unpadded == plaintext


class TestPKCS7Padding:
    """Test PKCS7 padding."""

    def test_pad_unpad(self):
        """Test basic pad/unpad."""
        padding = PKCS7Padding(16)

        # Test various lengths
        for length in range(1, 32):
            data = bytes(length)
            padded = padding.pad(data)

            assert len(padded) % 16 == 0
            assert len(padded) >= len(data)

            unpadded = padding.unpad(padded)
            assert unpadded == data

    def test_full_block_padding(self):
        """Test padding when data is already block-aligned."""
        padding = PKCS7Padding(16)

        data = bytes(16)  # Already 16 bytes
        padded = padding.pad(data)

        # Should add full block of padding
        assert len(padded) == 32
        assert padded[-16:] == bytes([16] * 16)

    def test_invalid_padding_detection(self):
        """Test detection of invalid padding."""
        padding = PKCS7Padding(16)

        # Invalid padding byte
        invalid1 = bytes(15) + bytes([17])  # Padding value > block size
        assert not padding.is_valid_padding(invalid1)

        # Inconsistent padding
        invalid2 = bytes(14) + bytes([2, 1])  # Should be [2, 2]
        assert not padding.is_valid_padding(invalid2)


class TestKeyUtilities:
    """Test key generation and derivation."""

    def test_generate_key(self):
        """Test random key generation."""
        key1 = generate_key(16)
        key2 = generate_key(16)

        assert len(key1) == 16
        assert len(key2) == 16
        assert key1 != key2  # Should be random

    def test_derive_key(self):
        """Test key derivation from password."""
        password = "test_password"

        key1, salt1 = derive_key(password)
        key2, salt2 = derive_key(password)

        # Different salts should produce different keys
        assert key1 != key2
        assert salt1 != salt2

        # Same salt should produce same key
        key3, _ = derive_key(password, salt1)
        assert key3 == key1

    def test_generate_iv(self):
        """Test IV generation."""
        iv1 = generate_iv(16)
        iv2 = generate_iv(16)

        assert len(iv1) == 16
        assert len(iv2) == 16
        assert iv1 != iv2


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
