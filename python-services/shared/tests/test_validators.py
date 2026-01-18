"""
Korean Business Document Validators Tests

Tests for:
- Business registration number (사업자등록번호)
- Resident registration number (주민등록번호)
- Corporate registration number (법인등록번호)
"""
import pytest
from shared.utils.validators import (
    validate_business_number,
    format_business_number,
    validate_resident_number,
    validate_corporate_number,
)


class TestValidateBusinessNumber:
    """Test Korean business registration number validation."""

    # ========================================================================
    # Valid Business Number Tests
    # ========================================================================

    def test_valid_business_number_no_dash(self):
        """Test valid business number without dashes."""
        # 1234567890 with correct check digit
        assert validate_business_number("1234567891") is True

    def test_valid_business_number_with_dashes(self):
        """Test valid business number with dashes."""
        assert validate_business_number("123-45-67891") is True

    def test_valid_business_number_with_spaces(self):
        """Test valid business number with spaces."""
        assert validate_business_number("123 45 67891") is True

    def test_known_valid_business_numbers(self):
        """Test known valid business numbers."""
        # Note: Using computed test patterns that pass the checksum algorithm
        # The check digit is calculated per the official algorithm
        valid_numbers = [
            "1234567891",  # Test number with valid check digit
        ]
        for num in valid_numbers:
            assert validate_business_number(num) is True, f"{num} should be valid"

    # ========================================================================
    # Invalid Business Number Tests
    # ========================================================================

    def test_invalid_length_too_short(self):
        """Test business number with less than 10 digits."""
        assert validate_business_number("123456789") is False

    def test_invalid_length_too_long(self):
        """Test business number with more than 10 digits."""
        assert validate_business_number("12345678901") is False

    def test_invalid_non_numeric(self):
        """Test business number with non-numeric characters."""
        assert validate_business_number("12345678ab") is False
        assert validate_business_number("abcdefghij") is False

    def test_invalid_check_digit(self):
        """Test business number with wrong check digit."""
        # 1234567890 - last digit should not be 0 for this prefix
        assert validate_business_number("1234567890") is False

    def test_invalid_empty_string(self):
        """Test empty string."""
        assert validate_business_number("") is False

    def test_invalid_whitespace_only(self):
        """Test whitespace only."""
        assert validate_business_number("   ") is False


class TestFormatBusinessNumber:
    """Test business number formatting."""

    def test_format_valid_number(self):
        """Test formatting valid business number."""
        result = format_business_number("1234567891")
        assert result == "123-45-67891"

    def test_format_already_formatted(self):
        """Test formatting already formatted number."""
        result = format_business_number("123-45-67891")
        assert result == "123-45-67891"

    def test_format_with_spaces(self):
        """Test formatting number with spaces."""
        result = format_business_number("123 45 67891")
        assert result == "123-45-67891"

    def test_format_invalid_number_raises_error(self):
        """Test that formatting invalid number raises ValueError."""
        with pytest.raises(ValueError, match="Invalid business number"):
            format_business_number("1234567890")  # Invalid check digit

    def test_format_too_short_raises_error(self):
        """Test that formatting short number raises ValueError."""
        with pytest.raises(ValueError, match="Invalid business number"):
            format_business_number("12345")


class TestValidateResidentNumber:
    """Test Korean resident registration number validation."""

    # ========================================================================
    # Valid Resident Number Tests
    # ========================================================================

    def test_valid_resident_number_no_dash(self):
        """Test valid resident number without dash."""
        # Note: Using test pattern, not real resident numbers
        # Check digit calculated: (11 - (122 % 11)) % 10 = 0
        assert validate_resident_number("8001011234560") is True

    def test_valid_resident_number_with_dash(self):
        """Test valid resident number with dash."""
        assert validate_resident_number("800101-1234560") is True

    def test_valid_resident_number_with_spaces(self):
        """Test valid resident number with spaces."""
        assert validate_resident_number("800101 1234560") is True

    # ========================================================================
    # Invalid Resident Number Tests
    # ========================================================================

    def test_invalid_length_too_short(self):
        """Test resident number with less than 13 digits."""
        assert validate_resident_number("800101123456") is False

    def test_invalid_length_too_long(self):
        """Test resident number with more than 13 digits."""
        assert validate_resident_number("80010112345678") is False

    def test_invalid_non_numeric(self):
        """Test resident number with non-numeric characters."""
        assert validate_resident_number("800101-123456a") is False

    def test_invalid_check_digit(self):
        """Test resident number with wrong check digit."""
        # 8001011234567 has incorrect check digit (correct is 0)
        assert validate_resident_number("8001011234567") is False

    def test_invalid_empty_string(self):
        """Test empty string."""
        assert validate_resident_number("") is False


class TestValidateCorporateNumber:
    """Test Korean corporate registration number validation."""

    # ========================================================================
    # Valid Corporate Number Tests
    # ========================================================================

    def test_valid_corporate_number_no_dash(self):
        """Test valid corporate number without dash."""
        # Check digit calculated: (10 - (32 % 10)) % 10 = 8
        assert validate_corporate_number("1101111234568") is True

    def test_valid_corporate_number_with_dash(self):
        """Test valid corporate number with dash."""
        assert validate_corporate_number("110111-1234568") is True

    # ========================================================================
    # Invalid Corporate Number Tests
    # ========================================================================

    def test_invalid_length_too_short(self):
        """Test corporate number with less than 13 digits."""
        assert validate_corporate_number("110111123456") is False

    def test_invalid_length_too_long(self):
        """Test corporate number with more than 13 digits."""
        assert validate_corporate_number("11011112345678") is False

    def test_invalid_non_numeric(self):
        """Test corporate number with non-numeric characters."""
        assert validate_corporate_number("110111-123456a") is False

    def test_invalid_check_digit(self):
        """Test corporate number with wrong check digit."""
        assert validate_corporate_number("1101111234561") is False

    def test_invalid_empty_string(self):
        """Test empty string."""
        assert validate_corporate_number("") is False


class TestCheckDigitAlgorithms:
    """Test the check digit calculation algorithms."""

    def test_business_number_check_digit_calculation(self):
        """Verify business number check digit algorithm."""
        # Test case: 123-45-67891
        # Weights: [1, 3, 7, 1, 3, 7, 1, 3, 5]
        # Digits:  [1, 2, 3, 4, 5, 6, 7, 8, 9, ?]
        #
        # Checksum:
        # 1*1 + 2*3 + 3*7 + 4*1 + 5*3 + 6*7 + 7*1 + 8*3 + 9*5
        # = 1 + 6 + 21 + 4 + 15 + 42 + 7 + 24 + 45 = 165
        # + floor(5*9/10) = 4
        # Total = 169
        # Remainder = 169 % 10 = 9
        # Check digit = (10 - 9) % 10 = 1
        assert validate_business_number("1234567891") is True

    def test_resident_number_gender_digits(self):
        """Test resident numbers with different gender/century digits."""
        # Gender digit meanings:
        # 1, 2: Korean born 1900-1999 (male, female)
        # 3, 4: Korean born 2000-2099 (male, female)
        # 5, 6: Foreigner born 1900-1999 (male, female)
        # 7, 8: Foreigner born 2000-2099 (male, female)

        # Test pattern with valid check digit (calculated)
        # 8001011234560 has check digit = 0
        assert validate_resident_number("8001011234560") is True
