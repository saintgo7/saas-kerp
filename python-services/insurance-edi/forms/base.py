"""
Base Form Definitions

Provides base classes for 4대보험 form definitions and field specifications.
Forms are used to define document structure for EDI submissions.
"""
from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional, Callable
from enum import Enum
from datetime import date


class FieldType(Enum):
    """Field data types."""

    STRING = "string"
    INTEGER = "integer"
    DECIMAL = "decimal"
    DATE = "date"
    BOOLEAN = "boolean"
    CHOICE = "choice"


class FieldAlignment(Enum):
    """Field alignment in fixed-width format."""

    LEFT = "left"
    RIGHT = "right"


@dataclass
class FormField:
    """
    Form field definition.

    Defines a single field in an insurance form including:
    - Name and data type
    - Validation rules
    - Formatting specifications for EDI
    """

    name: str
    label: str
    field_type: FieldType
    required: bool = False
    length: int = 0  # For fixed-width formatting
    alignment: FieldAlignment = FieldAlignment.LEFT
    pad_char: str = " "
    default: Any = None
    choices: List[tuple] = field(default_factory=list)  # (value, label) pairs
    validators: List[Callable[[Any], Optional[str]]] = field(default_factory=list)
    description: str = ""

    def validate(self, value: Any) -> List[str]:
        """
        Validate field value.

        Args:
            value: Value to validate

        Returns:
            List of error messages (empty if valid)
        """
        errors = []

        # Required check
        if self.required and (value is None or value == ""):
            errors.append(f"{self.label}은(는) 필수 항목입니다")
            return errors

        # Type-specific validation
        if value is not None and value != "":
            if self.field_type == FieldType.INTEGER:
                try:
                    int(value)
                except (ValueError, TypeError):
                    errors.append(f"{self.label}은(는) 숫자여야 합니다")

            elif self.field_type == FieldType.DECIMAL:
                try:
                    float(value)
                except (ValueError, TypeError):
                    errors.append(f"{self.label}은(는) 숫자여야 합니다")

            elif self.field_type == FieldType.DATE:
                if isinstance(value, str):
                    clean = value.replace("-", "").replace("/", "")
                    if len(clean) != 8 or not clean.isdigit():
                        errors.append(f"{self.label}은(는) 올바른 날짜 형식이어야 합니다 (YYYYMMDD)")

            elif self.field_type == FieldType.CHOICE:
                valid_values = [c[0] for c in self.choices]
                if value not in valid_values:
                    errors.append(f"{self.label}은(는) 유효한 선택지가 아닙니다")

        # Custom validators
        for validator in self.validators:
            error = validator(value)
            if error:
                errors.append(error)

        return errors

    def format_value(self, value: Any) -> str:
        """
        Format value for EDI output.

        Args:
            value: Value to format

        Returns:
            Formatted string value
        """
        if value is None:
            value = self.default if self.default is not None else ""

        # Type-specific formatting
        if self.field_type == FieldType.INTEGER:
            str_val = str(int(value)) if value else "0"
        elif self.field_type == FieldType.DECIMAL:
            str_val = str(float(value)) if value else "0"
        elif self.field_type == FieldType.DATE:
            if isinstance(value, date):
                str_val = value.strftime("%Y%m%d")
            else:
                str_val = str(value).replace("-", "").replace("/", "")[:8]
        elif self.field_type == FieldType.BOOLEAN:
            str_val = "Y" if value else "N"
        else:
            str_val = str(value)

        # Apply length formatting
        if self.length > 0:
            if self.alignment == FieldAlignment.LEFT:
                str_val = str_val.ljust(self.length, self.pad_char)[:self.length]
            else:
                if self.field_type in (FieldType.INTEGER, FieldType.DECIMAL):
                    str_val = str_val.zfill(self.length)[:self.length]
                else:
                    str_val = str_val.rjust(self.length, self.pad_char)[:self.length]

        return str_val


@dataclass
class FormSection:
    """
    Form section containing multiple fields.

    Represents a logical grouping of fields within a form.
    """

    name: str
    label: str
    fields: List[FormField] = field(default_factory=list)
    repeating: bool = False  # True for detail lines
    min_items: int = 0
    max_items: int = 9999

    def add_field(self, field_def: FormField) -> None:
        """Add a field to this section."""
        self.fields.append(field_def)

    def get_field(self, name: str) -> Optional[FormField]:
        """Get field by name."""
        for f in self.fields:
            if f.name == name:
                return f
        return None

    def validate(self, data: Dict[str, Any]) -> List[str]:
        """
        Validate section data.

        Args:
            data: Dictionary of field values

        Returns:
            List of error messages
        """
        errors = []

        for field_def in self.fields:
            value = data.get(field_def.name)
            field_errors = field_def.validate(value)
            errors.extend(field_errors)

        return errors

    def format_line(self, data: Dict[str, Any]) -> str:
        """
        Format section data as EDI line.

        Args:
            data: Dictionary of field values

        Returns:
            Formatted line string
        """
        parts = []
        for field_def in self.fields:
            value = data.get(field_def.name)
            formatted = field_def.format_value(value)
            parts.append(formatted)

        return "".join(parts)


class BaseForm(ABC):
    """
    Abstract base class for insurance forms.

    Defines the structure and behavior of an insurance document form.
    """

    def __init__(self):
        """Initialize form with sections."""
        self.sections: Dict[str, FormSection] = {}
        self._setup_form()

    @property
    @abstractmethod
    def form_code(self) -> str:
        """Form code for EDI."""
        pass

    @property
    @abstractmethod
    def form_name(self) -> str:
        """Form display name."""
        pass

    @property
    @abstractmethod
    def insurance_type(self) -> str:
        """Insurance type code."""
        pass

    @abstractmethod
    def _setup_form(self) -> None:
        """Set up form sections and fields. Override in subclass."""
        pass

    def add_section(self, section: FormSection) -> None:
        """Add a section to the form."""
        self.sections[section.name] = section

    def get_section(self, name: str) -> Optional[FormSection]:
        """Get section by name."""
        return self.sections.get(name)

    def validate(self, data: Dict[str, Any]) -> List[str]:
        """
        Validate complete form data.

        Args:
            data: Form data with section keys

        Returns:
            List of all validation errors
        """
        errors = []

        for section_name, section in self.sections.items():
            section_data = data.get(section_name, {})

            if section.repeating:
                # Handle repeating sections (detail lines)
                if not isinstance(section_data, list):
                    section_data = [section_data] if section_data else []

                if len(section_data) < section.min_items:
                    errors.append(f"{section.label}: 최소 {section.min_items}개 항목이 필요합니다")
                elif len(section_data) > section.max_items:
                    errors.append(f"{section.label}: 최대 {section.max_items}개 항목까지 가능합니다")

                for i, item in enumerate(section_data):
                    item_errors = section.validate(item)
                    for err in item_errors:
                        errors.append(f"{section.label} [{i+1}]: {err}")
            else:
                section_errors = section.validate(section_data)
                for err in section_errors:
                    errors.append(f"{section.label}: {err}")

        return errors

    def to_edi(self, data: Dict[str, Any], encoding: str = "euc-kr") -> bytes:
        """
        Convert form data to EDI format.

        Args:
            data: Form data
            encoding: Character encoding

        Returns:
            EDI-formatted bytes
        """
        lines = []

        for section_name, section in self.sections.items():
            section_data = data.get(section_name, {})

            if section.repeating:
                if not isinstance(section_data, list):
                    section_data = [section_data] if section_data else []

                for item in section_data:
                    line = section.format_line(item)
                    lines.append(line)
            else:
                line = section.format_line(section_data)
                lines.append(line)

        content = "\n".join(lines)
        return content.encode(encoding)

    @classmethod
    def from_edi(cls, data: bytes, encoding: str = "euc-kr") -> Dict[str, Any]:
        """
        Parse EDI data into form structure.

        Args:
            data: EDI-formatted bytes
            encoding: Character encoding

        Returns:
            Parsed form data
        """
        # Implementation depends on specific form format
        raise NotImplementedError("Subclass must implement from_edi")


# Common validators
def validate_business_no(value: str) -> Optional[str]:
    """Validate Korean business registration number."""
    if not value:
        return None

    clean = value.replace("-", "")
    if len(clean) != 10 or not clean.isdigit():
        return "사업자등록번호는 10자리 숫자여야 합니다"

    # Checksum validation
    weights = [1, 3, 7, 1, 3, 7, 1, 3, 5]
    total = sum(int(clean[i]) * weights[i] for i in range(9))
    total += int(clean[8]) * 5 // 10
    check = (10 - (total % 10)) % 10

    if check != int(clean[9]):
        return "사업자등록번호가 유효하지 않습니다"

    return None


def validate_resident_no(value: str) -> Optional[str]:
    """Validate Korean resident registration number."""
    if not value:
        return None

    clean = value.replace("-", "")
    if len(clean) != 13 or not clean.isdigit():
        return "주민등록번호는 13자리 숫자여야 합니다"

    # Basic validation (birth date check)
    birth_century = {"1": "19", "2": "19", "3": "20", "4": "20", "5": "19", "6": "19", "7": "20", "8": "20"}
    gender_digit = clean[6]

    if gender_digit not in birth_century:
        return "주민등록번호가 유효하지 않습니다"

    year = int(birth_century[gender_digit] + clean[0:2])
    month = int(clean[2:4])
    day = int(clean[4:6])

    if month < 1 or month > 12 or day < 1 or day > 31:
        return "주민등록번호의 생년월일이 유효하지 않습니다"

    return None


def validate_phone(value: str) -> Optional[str]:
    """Validate phone number."""
    if not value:
        return None

    clean = value.replace("-", "").replace(" ", "")
    if not clean.isdigit() or len(clean) < 9 or len(clean) > 11:
        return "전화번호 형식이 올바르지 않습니다"

    return None
