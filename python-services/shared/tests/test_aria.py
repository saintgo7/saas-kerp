"""
ARIA Block Cipher Tests

Tests for Korean national standard block cipher (KS X 1213:2004).
Validates:
- Key sizes: 128, 192, 256 bits
- Block encryption/decryption
- CBC mode operation
- Round-trip encryption
"""
import pytest
from python_services.shared.crypto.aria import ARIACipher, ARIAModeCBC


class TestARIACipher:
    """Test ARIA block cipher core functionality."""

    # ========================================================================
    # Initialization Tests
    # ========================================================================

    def test_init_with_128bit_key(self):
        """Test initialization with 128-bit (16-byte) key."""
        key = b"0123456789abcdef"  # 16 bytes
        cipher = ARIACipher(key)
        assert cipher._key_size == 128
        assert cipher._rounds == 12

    def test_init_with_192bit_key(self):
        """Test initialization with 192-bit (24-byte) key."""
        key = b"0123456789abcdef01234567"  # 24 bytes
        cipher = ARIACipher(key)
        assert cipher._key_size == 192
        assert cipher._rounds == 14

    def test_init_with_256bit_key(self):
        """Test initialization with 256-bit (32-byte) key."""
        key = b"0123456789abcdef0123456789abcdef"  # 32 bytes
        cipher = ARIACipher(key)
        assert cipher._key_size == 256
        assert cipher._rounds == 16

    def test_init_with_string_key(self):
        """Test initialization with string key (auto-converted to bytes)."""
        key = "0123456789abcdef"
        cipher = ARIACipher(key)
        assert cipher._key_size == 128

    def test_init_invalid_key_size_raises_error(self):
        """Test that invalid key sizes raise ValueError."""
        with pytest.raises(ValueError, match="Invalid key size"):
            ARIACipher(b"tooshort")  # 8 bytes

        with pytest.raises(ValueError, match="Invalid key size"):
            ARIACipher(b"0123456789")  # 10 bytes

        with pytest.raises(ValueError, match="Invalid key size"):
            ARIACipher(b"0" * 33)  # 33 bytes

    # ========================================================================
    # Block Encryption Tests
    # ========================================================================

    def test_encrypt_block_produces_16_bytes(self):
        """Test that encrypting a block produces 16 bytes."""
        key = b"0123456789abcdef"
        cipher = ARIACipher(key)
        plaintext = b"sixteen byte txt"

        ciphertext = cipher.encrypt_block(plaintext)

        assert len(ciphertext) == 16
        assert ciphertext != plaintext

    def test_encrypt_block_invalid_size_raises_error(self):
        """Test that non-16-byte blocks raise ValueError."""
        key = b"0123456789abcdef"
        cipher = ARIACipher(key)

        with pytest.raises(ValueError, match="Block size must be 16 bytes"):
            cipher.encrypt_block(b"short")

        with pytest.raises(ValueError, match="Block size must be 16 bytes"):
            cipher.encrypt_block(b"this is too long for a block")

    def test_encrypt_block_deterministic(self):
        """Test that encryption is deterministic (same input = same output)."""
        key = b"0123456789abcdef"
        cipher = ARIACipher(key)
        plaintext = b"sixteen byte txt"

        result1 = cipher.encrypt_block(plaintext)
        result2 = cipher.encrypt_block(plaintext)

        assert result1 == result2

    def test_different_keys_produce_different_ciphertext(self):
        """Test that different keys produce different ciphertexts."""
        key1 = b"0123456789abcdef"
        key2 = b"fedcba9876543210"
        plaintext = b"sixteen byte txt"

        cipher1 = ARIACipher(key1)
        cipher2 = ARIACipher(key2)

        ciphertext1 = cipher1.encrypt_block(plaintext)
        ciphertext2 = cipher2.encrypt_block(plaintext)

        assert ciphertext1 != ciphertext2

    # ========================================================================
    # Block Decryption Tests
    # ========================================================================

    def test_decrypt_block_produces_16_bytes(self):
        """Test that decrypting a block produces 16 bytes."""
        key = b"0123456789abcdef"
        cipher = ARIACipher(key)
        plaintext = b"sixteen byte txt"
        ciphertext = cipher.encrypt_block(plaintext)

        decrypted = cipher.decrypt_block(ciphertext)

        assert len(decrypted) == 16

    def test_decrypt_block_invalid_size_raises_error(self):
        """Test that non-16-byte blocks raise ValueError."""
        key = b"0123456789abcdef"
        cipher = ARIACipher(key)

        with pytest.raises(ValueError, match="Block size must be 16 bytes"):
            cipher.decrypt_block(b"short")

    # ========================================================================
    # Round-trip Tests
    # ========================================================================

    def test_roundtrip_128bit_key(self):
        """Test encrypt then decrypt with 128-bit key."""
        key = b"0123456789abcdef"
        cipher = ARIACipher(key)
        plaintext = b"Hello, ARIA 128!"

        ciphertext = cipher.encrypt_block(plaintext)
        decrypted = cipher.decrypt_block(ciphertext)

        assert decrypted == plaintext

    def test_roundtrip_192bit_key(self):
        """Test encrypt then decrypt with 192-bit key."""
        key = b"0123456789abcdef01234567"
        cipher = ARIACipher(key)
        plaintext = b"Hello, ARIA 192!"

        ciphertext = cipher.encrypt_block(plaintext)
        decrypted = cipher.decrypt_block(ciphertext)

        assert decrypted == plaintext

    def test_roundtrip_256bit_key(self):
        """Test encrypt then decrypt with 256-bit key."""
        key = b"0123456789abcdef0123456789abcdef"
        cipher = ARIACipher(key)
        plaintext = b"Hello, ARIA 256!"

        ciphertext = cipher.encrypt_block(plaintext)
        decrypted = cipher.decrypt_block(ciphertext)

        assert decrypted == plaintext

    def test_roundtrip_binary_data(self):
        """Test encrypt/decrypt with binary data including null bytes."""
        key = b"0123456789abcdef"
        cipher = ARIACipher(key)
        plaintext = bytes(range(16))  # 0x00 through 0x0F

        ciphertext = cipher.encrypt_block(plaintext)
        decrypted = cipher.decrypt_block(ciphertext)

        assert decrypted == plaintext

    def test_roundtrip_all_zeros(self):
        """Test encrypt/decrypt block of all zeros."""
        key = b"0123456789abcdef"
        cipher = ARIACipher(key)
        plaintext = b"\x00" * 16

        ciphertext = cipher.encrypt_block(plaintext)
        decrypted = cipher.decrypt_block(ciphertext)

        assert decrypted == plaintext
        assert ciphertext != plaintext  # Should not be all zeros

    def test_roundtrip_all_ones(self):
        """Test encrypt/decrypt block of all 0xFF bytes."""
        key = b"0123456789abcdef"
        cipher = ARIACipher(key)
        plaintext = b"\xff" * 16

        ciphertext = cipher.encrypt_block(plaintext)
        decrypted = cipher.decrypt_block(ciphertext)

        assert decrypted == plaintext


class TestARIAModeCBC:
    """Test ARIA cipher in CBC mode."""

    # ========================================================================
    # Initialization Tests
    # ========================================================================

    def test_init_with_key_only(self):
        """Test initialization with key only (default IV)."""
        key = b"0123456789abcdef"
        cipher = ARIAModeCBC(key)

        assert cipher.iv == bytes(16)  # Default IV is all zeros

    def test_init_with_key_and_iv(self):
        """Test initialization with key and custom IV."""
        key = b"0123456789abcdef"
        iv = b"fedcba9876543210"
        cipher = ARIAModeCBC(key, iv)

        assert cipher.iv == iv

    def test_init_invalid_iv_size_raises_error(self):
        """Test that invalid IV size raises ValueError."""
        key = b"0123456789abcdef"

        with pytest.raises(ValueError, match="IV must be 16 bytes"):
            ARIAModeCBC(key, b"tooshort")

        with pytest.raises(ValueError, match="IV must be 16 bytes"):
            ARIAModeCBC(key, b"0" * 17)

    # ========================================================================
    # CBC Encryption Tests
    # ========================================================================

    def test_encrypt_single_block(self):
        """Test CBC encryption of single 16-byte block."""
        key = b"0123456789abcdef"
        iv = b"fedcba9876543210"
        cipher = ARIAModeCBC(key, iv)
        plaintext = b"sixteen byte txt"

        ciphertext = cipher.encrypt(plaintext)

        assert len(ciphertext) == 16
        assert ciphertext != plaintext

    def test_encrypt_multiple_blocks(self):
        """Test CBC encryption of multiple blocks."""
        key = b"0123456789abcdef"
        iv = b"fedcba9876543210"
        cipher = ARIAModeCBC(key, iv)
        plaintext = b"A" * 32 + b"B" * 16  # 3 blocks (48 bytes)

        ciphertext = cipher.encrypt(plaintext)

        assert len(ciphertext) == 48

    def test_encrypt_invalid_length_raises_error(self):
        """Test that plaintext not multiple of 16 raises ValueError."""
        key = b"0123456789abcdef"
        cipher = ARIAModeCBC(key)

        with pytest.raises(ValueError, match="multiple of 16"):
            cipher.encrypt(b"not 16 bytes")

    def test_encrypt_same_blocks_produce_different_ciphertext(self):
        """Test CBC chaining - identical plaintext blocks produce different ciphertext."""
        key = b"0123456789abcdef"
        iv = b"fedcba9876543210"
        cipher = ARIAModeCBC(key, iv)
        plaintext = b"A" * 16 + b"A" * 16  # Two identical blocks

        ciphertext = cipher.encrypt(plaintext)

        block1 = ciphertext[:16]
        block2 = ciphertext[16:]

        # In CBC mode, identical plaintext blocks produce different ciphertext
        assert block1 != block2

    def test_encrypt_different_iv_produces_different_ciphertext(self):
        """Test that different IVs produce different ciphertexts."""
        key = b"0123456789abcdef"
        iv1 = b"1111111111111111"
        iv2 = b"2222222222222222"
        plaintext = b"sixteen byte txt"

        cipher1 = ARIAModeCBC(key, iv1)
        cipher2 = ARIAModeCBC(key, iv2)

        ciphertext1 = cipher1.encrypt(plaintext)
        ciphertext2 = cipher2.encrypt(plaintext)

        assert ciphertext1 != ciphertext2

    # ========================================================================
    # CBC Decryption Tests
    # ========================================================================

    def test_decrypt_single_block(self):
        """Test CBC decryption of single block."""
        key = b"0123456789abcdef"
        iv = b"fedcba9876543210"
        cipher = ARIAModeCBC(key, iv)
        plaintext = b"sixteen byte txt"
        ciphertext = cipher.encrypt(plaintext)

        decrypted = cipher.decrypt(ciphertext)

        assert decrypted == plaintext

    def test_decrypt_multiple_blocks(self):
        """Test CBC decryption of multiple blocks."""
        key = b"0123456789abcdef"
        iv = b"fedcba9876543210"
        cipher = ARIAModeCBC(key, iv)
        plaintext = b"This is a test message that is exactly 48 bytes!"

        ciphertext = cipher.encrypt(plaintext)
        decrypted = cipher.decrypt(ciphertext)

        assert decrypted == plaintext

    def test_decrypt_invalid_length_raises_error(self):
        """Test that ciphertext not multiple of 16 raises ValueError."""
        key = b"0123456789abcdef"
        cipher = ARIAModeCBC(key)

        with pytest.raises(ValueError, match="multiple of 16"):
            cipher.decrypt(b"not 16 bytes")

    # ========================================================================
    # CBC Round-trip Tests
    # ========================================================================

    def test_roundtrip_with_all_key_sizes(self):
        """Test CBC round-trip with all supported key sizes."""
        keys = [
            b"0123456789abcdef",  # 128-bit
            b"0123456789abcdef01234567",  # 192-bit
            b"0123456789abcdef0123456789abcdef",  # 256-bit
        ]
        iv = b"fedcba9876543210"
        plaintext = b"Test message 123"

        for key in keys:
            cipher = ARIAModeCBC(key, iv)
            ciphertext = cipher.encrypt(plaintext)
            decrypted = cipher.decrypt(ciphertext)
            assert decrypted == plaintext, f"Round-trip failed for {len(key)*8}-bit key"

    def test_roundtrip_large_data(self):
        """Test CBC round-trip with larger data (10 blocks)."""
        key = b"0123456789abcdef"
        iv = b"fedcba9876543210"
        cipher = ARIAModeCBC(key, iv)
        plaintext = b"X" * 160  # 10 blocks

        ciphertext = cipher.encrypt(plaintext)
        decrypted = cipher.decrypt(ciphertext)

        assert decrypted == plaintext

    def test_roundtrip_korean_text(self):
        """Test CBC round-trip with Korean text (UTF-8 encoded)."""
        key = b"0123456789abcdef"
        iv = b"fedcba9876543210"
        cipher = ARIAModeCBC(key, iv)

        # Korean text padded to 32 bytes (multiple of 16)
        korean_text = "안녕하세요".encode("utf-8")
        # Pad to multiple of 16
        padding_len = 16 - (len(korean_text) % 16)
        plaintext = korean_text + bytes([padding_len] * padding_len)

        ciphertext = cipher.encrypt(plaintext)
        decrypted = cipher.decrypt(ciphertext)

        assert decrypted == plaintext


class TestARIAKnownVectors:
    """Test ARIA with known test vectors (if available)."""

    def test_zero_key_zero_plaintext(self):
        """Test with all-zero key and plaintext (basic sanity check)."""
        key = b"\x00" * 16
        plaintext = b"\x00" * 16
        cipher = ARIACipher(key)

        ciphertext = cipher.encrypt_block(plaintext)
        decrypted = cipher.decrypt_block(ciphertext)

        # Ciphertext should not be all zeros
        assert ciphertext != plaintext
        # Decryption should recover original
        assert decrypted == plaintext
