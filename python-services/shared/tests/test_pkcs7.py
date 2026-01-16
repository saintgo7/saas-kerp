"""
PKCS#7 Padding and Signature Tests

Tests for:
1. PKCS#7 padding scheme (block cipher padding)
2. PKCS#7/CMS digital signatures (RSA with SHA-256)

Used for 4대보험 EDI communication.
"""
import pytest
from shared.crypto.pkcs7 import (
    PKCS7Padding,
    PKCS7Signature,
    generate_test_keypair,
)


class TestPKCS7Padding:
    """Test PKCS#7 padding for block ciphers."""

    # ========================================================================
    # Initialization Tests
    # ========================================================================

    def test_init_default_block_size(self):
        """Test default block size is 16 bytes (128 bits)."""
        padder = PKCS7Padding()
        assert padder.block_size == 16

    def test_init_custom_block_size(self):
        """Test custom block size."""
        padder = PKCS7Padding(block_size=8)
        assert padder.block_size == 8

    def test_init_invalid_block_size_raises_error(self):
        """Test that invalid block sizes raise ValueError."""
        with pytest.raises(ValueError, match="between 1 and 255"):
            PKCS7Padding(block_size=0)

        with pytest.raises(ValueError, match="between 1 and 255"):
            PKCS7Padding(block_size=256)

        with pytest.raises(ValueError, match="between 1 and 255"):
            PKCS7Padding(block_size=-1)

    # ========================================================================
    # Padding Tests
    # ========================================================================

    def test_pad_empty_data(self):
        """Test padding empty data adds full block of padding."""
        padder = PKCS7Padding(block_size=16)

        result = padder.pad(b"")

        assert len(result) == 16
        assert result == bytes([16] * 16)

    def test_pad_data_smaller_than_block(self):
        """Test padding data smaller than block size."""
        padder = PKCS7Padding(block_size=16)
        data = b"Hello"  # 5 bytes

        result = padder.pad(data)

        assert len(result) == 16
        # Last 11 bytes should all be 0x0B (11)
        assert result[-11:] == bytes([11] * 11)
        assert result[:5] == data

    def test_pad_data_exactly_one_block(self):
        """Test padding data that is exactly one block adds another full block."""
        padder = PKCS7Padding(block_size=16)
        data = b"sixteen byte txt"  # 16 bytes

        result = padder.pad(data)

        # Should add full block of padding (16 bytes of value 16)
        assert len(result) == 32
        assert result[16:] == bytes([16] * 16)

    def test_pad_data_one_byte_short(self):
        """Test padding data that is one byte short of block size."""
        padder = PKCS7Padding(block_size=16)
        data = b"fifteen bytes!!"  # 15 bytes

        result = padder.pad(data)

        assert len(result) == 16
        assert result[-1] == 1  # Padding of 0x01

    def test_pad_data_multiple_blocks(self):
        """Test padding data that spans multiple blocks."""
        padder = PKCS7Padding(block_size=16)
        data = b"A" * 30  # 30 bytes

        result = padder.pad(data)

        assert len(result) == 32  # 2 blocks
        assert result[-2:] == bytes([2, 2])  # Last 2 bytes are 0x02

    def test_pad_with_8byte_block_size(self):
        """Test padding with 8-byte block size."""
        padder = PKCS7Padding(block_size=8)
        data = b"Hi"  # 2 bytes

        result = padder.pad(data)

        assert len(result) == 8
        assert result[-6:] == bytes([6] * 6)

    # ========================================================================
    # Unpadding Tests
    # ========================================================================

    def test_unpad_valid_padding(self):
        """Test removing valid padding."""
        padder = PKCS7Padding(block_size=16)
        original = b"Hello"
        padded = padder.pad(original)

        result = padder.unpad(padded)

        assert result == original

    def test_unpad_full_block_padding(self):
        """Test removing full block of padding."""
        padder = PKCS7Padding(block_size=16)
        original = b"sixteen byte txt"
        padded = padder.pad(original)

        result = padder.unpad(padded)

        assert result == original

    def test_unpad_single_byte_padding(self):
        """Test removing single byte padding."""
        padder = PKCS7Padding(block_size=16)
        original = b"fifteen bytes!!"
        padded = padder.pad(original)

        result = padder.unpad(padded)

        assert result == original

    def test_unpad_empty_data_raises_error(self):
        """Test that unpadding empty data raises ValueError."""
        padder = PKCS7Padding(block_size=16)

        with pytest.raises(ValueError, match="Data is empty"):
            padder.unpad(b"")

    def test_unpad_invalid_length_raises_error(self):
        """Test that data not multiple of block size raises ValueError."""
        padder = PKCS7Padding(block_size=16)

        with pytest.raises(ValueError, match="not a multiple of block size"):
            padder.unpad(b"not 16 bytes")

    def test_unpad_zero_padding_value_raises_error(self):
        """Test that zero padding value raises ValueError."""
        padder = PKCS7Padding(block_size=16)
        # Manually construct invalid data with padding value 0
        invalid_data = b"A" * 15 + b"\x00"

        with pytest.raises(ValueError, match="Invalid padding length"):
            padder.unpad(invalid_data)

    def test_unpad_padding_larger_than_block_raises_error(self):
        """Test that padding value larger than block size raises ValueError."""
        padder = PKCS7Padding(block_size=16)
        # Manually construct invalid data with padding value 17
        invalid_data = b"A" * 15 + bytes([17])

        with pytest.raises(ValueError, match="Invalid padding length"):
            padder.unpad(invalid_data)

    def test_unpad_inconsistent_padding_raises_error(self):
        """Test that inconsistent padding bytes raise ValueError."""
        padder = PKCS7Padding(block_size=16)
        # Padding claims to be 3 bytes but values don't match
        invalid_data = b"A" * 13 + bytes([3, 2, 3])

        with pytest.raises(ValueError, match="Invalid padding bytes"):
            padder.unpad(invalid_data)

    # ========================================================================
    # is_valid_padding Tests
    # ========================================================================

    def test_is_valid_padding_returns_true_for_valid(self):
        """Test is_valid_padding returns True for valid padding."""
        padder = PKCS7Padding(block_size=16)
        valid_data = padder.pad(b"Hello")

        assert padder.is_valid_padding(valid_data) is True

    def test_is_valid_padding_returns_false_for_invalid(self):
        """Test is_valid_padding returns False for invalid padding."""
        padder = PKCS7Padding(block_size=16)
        invalid_data = b"A" * 15 + b"\x00"

        assert padder.is_valid_padding(invalid_data) is False

    def test_is_valid_padding_returns_false_for_empty(self):
        """Test is_valid_padding returns False for empty data."""
        padder = PKCS7Padding(block_size=16)

        assert padder.is_valid_padding(b"") is False

    # ========================================================================
    # Round-trip Tests
    # ========================================================================

    def test_roundtrip_various_lengths(self):
        """Test pad then unpad for various data lengths."""
        padder = PKCS7Padding(block_size=16)

        for length in [0, 1, 5, 15, 16, 17, 31, 32, 100]:
            original = b"X" * length
            padded = padder.pad(original)
            result = padder.unpad(padded)
            assert result == original, f"Round-trip failed for length {length}"

    def test_roundtrip_binary_data(self):
        """Test pad/unpad with binary data including null bytes."""
        padder = PKCS7Padding(block_size=16)
        original = bytes(range(256))  # All possible byte values

        padded = padder.pad(original)
        result = padder.unpad(padded)

        assert result == original


class TestPKCS7Signature:
    """Test PKCS#7/CMS digital signatures."""

    @pytest.fixture
    def test_keypair(self):
        """Generate test key pair for signature tests."""
        private_key_pem, certificate_pem = generate_test_keypair()
        return private_key_pem, certificate_pem

    @pytest.fixture
    def signature_handler(self, test_keypair):
        """Create signature handler with test keypair."""
        private_key_pem, certificate_pem = test_keypair
        handler = PKCS7Signature()
        handler.load_private_key_bytes(private_key_pem)
        handler.load_certificate_bytes(certificate_pem)
        return handler

    # ========================================================================
    # Initialization Tests
    # ========================================================================

    def test_init_without_keys(self):
        """Test initialization without keys."""
        handler = PKCS7Signature()
        assert handler._private_key is None
        assert handler._certificate is None

    def test_load_private_key_bytes(self, test_keypair):
        """Test loading private key from bytes."""
        private_key_pem, _ = test_keypair
        handler = PKCS7Signature()

        handler.load_private_key_bytes(private_key_pem)

        assert handler._private_key is not None

    def test_load_certificate_bytes(self, test_keypair):
        """Test loading certificate from bytes."""
        _, certificate_pem = test_keypair
        handler = PKCS7Signature()

        handler.load_certificate_bytes(certificate_pem)

        assert handler._certificate is not None

    # ========================================================================
    # Signing Tests
    # ========================================================================

    def test_sign_requires_private_key(self):
        """Test that signing without private key raises ValueError."""
        handler = PKCS7Signature()

        with pytest.raises(ValueError, match="Private key not loaded"):
            handler.sign(b"data")

    def test_sign_requires_certificate(self, test_keypair):
        """Test that signing without certificate raises ValueError."""
        private_key_pem, _ = test_keypair
        handler = PKCS7Signature()
        handler.load_private_key_bytes(private_key_pem)

        with pytest.raises(ValueError, match="Certificate not loaded"):
            handler.sign(b"data")

    def test_sign_produces_bytes(self, signature_handler):
        """Test that signing produces bytes."""
        data = b"Test data to sign"

        signature = signature_handler.sign(data)

        assert isinstance(signature, bytes)
        assert len(signature) > 0

    def test_sign_with_content_produces_bytes(self, signature_handler):
        """Test that signing with embedded content produces bytes."""
        data = b"Test data to sign"

        signature = signature_handler.sign_with_content(data)

        assert isinstance(signature, bytes)
        assert len(signature) > 0

    def test_sign_different_data_produces_different_signatures(self, signature_handler):
        """Test that different data produces different signatures."""
        data1 = b"First message"
        data2 = b"Second message"

        sig1 = signature_handler.sign(data1)
        sig2 = signature_handler.sign(data2)

        assert sig1 != sig2

    # ========================================================================
    # Raw Signature Tests
    # ========================================================================

    def test_sign_raw_produces_bytes(self, signature_handler):
        """Test that raw signing produces bytes."""
        data = b"Test data"

        signature = signature_handler.sign_raw(data)

        assert isinstance(signature, bytes)
        # RSA 2048-bit signature should be 256 bytes
        assert len(signature) == 256

    def test_sign_raw_requires_private_key(self):
        """Test that raw signing without private key raises ValueError."""
        handler = PKCS7Signature()

        with pytest.raises(ValueError, match="Private key not loaded"):
            handler.sign_raw(b"data")

    def test_verify_raw_valid_signature(self, signature_handler):
        """Test verifying valid raw signature."""
        data = b"Test data to verify"
        signature = signature_handler.sign_raw(data)

        result = signature_handler.verify_raw(data, signature)

        assert result is True

    def test_verify_raw_invalid_signature(self, signature_handler):
        """Test verifying invalid raw signature."""
        data = b"Test data"
        invalid_signature = b"\x00" * 256

        result = signature_handler.verify_raw(data, invalid_signature)

        assert result is False

    def test_verify_raw_modified_data(self, signature_handler):
        """Test that modified data fails verification."""
        original_data = b"Original data"
        signature = signature_handler.sign_raw(original_data)
        modified_data = b"Modified data"

        result = signature_handler.verify_raw(modified_data, signature)

        assert result is False

    def test_verify_raw_requires_certificate(self):
        """Test that verify_raw without certificate raises ValueError."""
        handler = PKCS7Signature()

        with pytest.raises(ValueError, match="Certificate not loaded"):
            handler.verify_raw(b"data", b"sig")

    # ========================================================================
    # Certificate Info Tests
    # ========================================================================

    def test_certificate_info_empty_without_cert(self):
        """Test certificate_info returns empty dict without certificate."""
        handler = PKCS7Signature()

        info = handler.certificate_info

        assert info == {}

    def test_certificate_info_returns_details(self, signature_handler):
        """Test certificate_info returns certificate details."""
        info = signature_handler.certificate_info

        assert "subject" in info
        assert "issuer" in info
        assert "serial_number" in info
        assert "not_valid_before" in info
        assert "not_valid_after" in info

        assert info["subject"]["country"] == "KR"
        assert info["subject"]["organization"] == "Test Organization"
        assert info["subject"]["common_name"] == "Test Certificate"

    # ========================================================================
    # Certificate Validity Tests
    # ========================================================================

    def test_is_certificate_valid_returns_false_without_cert(self):
        """Test is_certificate_valid returns False without certificate."""
        handler = PKCS7Signature()

        assert handler.is_certificate_valid() is False

    def test_is_certificate_valid_returns_true_for_valid_cert(self, signature_handler):
        """Test is_certificate_valid returns True for valid certificate."""
        # Test certificate is valid for 365 days from creation
        assert signature_handler.is_certificate_valid() is True


class TestGenerateTestKeypair:
    """Test the test keypair generation utility."""

    def test_generates_private_key_and_certificate(self):
        """Test that function generates both private key and certificate."""
        private_key_pem, certificate_pem = generate_test_keypair()

        assert isinstance(private_key_pem, bytes)
        assert isinstance(certificate_pem, bytes)

        assert b"-----BEGIN PRIVATE KEY-----" in private_key_pem
        assert b"-----END PRIVATE KEY-----" in private_key_pem

        assert b"-----BEGIN CERTIFICATE-----" in certificate_pem
        assert b"-----END CERTIFICATE-----" in certificate_pem

    def test_generated_keypair_is_usable(self):
        """Test that generated keypair can be used for signing/verification."""
        private_key_pem, certificate_pem = generate_test_keypair()

        handler = PKCS7Signature()
        handler.load_private_key_bytes(private_key_pem)
        handler.load_certificate_bytes(certificate_pem)

        data = b"Test message"
        signature = handler.sign_raw(data)
        result = handler.verify_raw(data, signature)

        assert result is True

    def test_each_call_generates_unique_keypair(self):
        """Test that each call generates a unique keypair."""
        key1, cert1 = generate_test_keypair()
        key2, cert2 = generate_test_keypair()

        assert key1 != key2
        assert cert1 != cert2
