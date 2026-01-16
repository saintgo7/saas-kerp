"""
NPS (National Pension Service) Form Definitions - 국민연금 신고서 양식

Forms:
- NPSAcquisitionForm: 취득신고서
- NPSLossForm: 상실신고서
- NPSChangeForm: 내용변경신고서
"""
from typing import Dict, Any

from .base import (
    BaseForm,
    FormSection,
    FormField,
    FieldType,
    FieldAlignment,
    validate_business_no,
    validate_resident_no,
)


class NPSAcquisitionForm(BaseForm):
    """
    NPS Acquisition Form (국민연금 취득신고서).

    Used for reporting new employee enrollment in national pension.
    """

    @property
    def form_code(self) -> str:
        return "NPS-1001"

    @property
    def form_name(self) -> str:
        return "국민연금 취득신고서"

    @property
    def insurance_type(self) -> str:
        return "NPS"

    def _setup_form(self) -> None:
        """Set up acquisition form structure."""
        # Header section
        header = FormSection(name="header", label="신고서 헤더")
        header.add_field(FormField(
            name="form_code",
            label="서식코드",
            field_type=FieldType.STRING,
            length=10,
            default="NPS-1001",
        ))
        header.add_field(FormField(
            name="submit_date",
            label="제출일자",
            field_type=FieldType.DATE,
            required=True,
            length=8,
        ))
        self.add_section(header)

        # Company section
        company = FormSection(name="company", label="사업장 정보")
        company.add_field(FormField(
            name="business_no",
            label="사업자등록번호",
            field_type=FieldType.STRING,
            required=True,
            length=10,
            validators=[validate_business_no],
        ))
        company.add_field(FormField(
            name="workplace_no",
            label="사업장관리번호",
            field_type=FieldType.STRING,
            required=True,
            length=13,
        ))
        company.add_field(FormField(
            name="company_name",
            label="사업장명",
            field_type=FieldType.STRING,
            required=True,
            length=40,
        ))
        company.add_field(FormField(
            name="representative",
            label="대표자명",
            field_type=FieldType.STRING,
            length=20,
        ))
        self.add_section(company)

        # Detail section (employees - repeating)
        detail = FormSection(
            name="employees",
            label="취득자 명세",
            repeating=True,
            min_items=1,
        )
        detail.add_field(FormField(
            name="resident_no",
            label="주민등록번호",
            field_type=FieldType.STRING,
            required=True,
            length=13,
            validators=[validate_resident_no],
        ))
        detail.add_field(FormField(
            name="name",
            label="성명",
            field_type=FieldType.STRING,
            required=True,
            length=20,
        ))
        detail.add_field(FormField(
            name="acquisition_date",
            label="취득일",
            field_type=FieldType.DATE,
            required=True,
            length=8,
        ))
        detail.add_field(FormField(
            name="monthly_income",
            label="기준소득월액",
            field_type=FieldType.INTEGER,
            required=True,
            length=15,
            alignment=FieldAlignment.RIGHT,
            pad_char="0",
        ))
        detail.add_field(FormField(
            name="acquisition_type",
            label="취득유형",
            field_type=FieldType.CHOICE,
            required=True,
            length=2,
            choices=[
                ("01", "신규취득"),
                ("02", "재취득"),
                ("03", "전입취득"),
            ],
            default="01",
        ))
        detail.add_field(FormField(
            name="job_type",
            label="직종",
            field_type=FieldType.CHOICE,
            length=2,
            choices=[
                ("01", "사무직"),
                ("02", "생산직"),
                ("03", "판매직"),
                ("04", "서비스직"),
                ("99", "기타"),
            ],
            default="01",
        ))
        detail.add_field(FormField(
            name="nationality",
            label="국적코드",
            field_type=FieldType.STRING,
            length=3,
            default="KOR",
        ))
        self.add_section(detail)


class NPSLossForm(BaseForm):
    """
    NPS Loss Form (국민연금 상실신고서).

    Used for reporting employee departure from national pension.
    """

    @property
    def form_code(self) -> str:
        return "NPS-1002"

    @property
    def form_name(self) -> str:
        return "국민연금 상실신고서"

    @property
    def insurance_type(self) -> str:
        return "NPS"

    def _setup_form(self) -> None:
        """Set up loss form structure."""
        # Header section
        header = FormSection(name="header", label="신고서 헤더")
        header.add_field(FormField(
            name="form_code",
            label="서식코드",
            field_type=FieldType.STRING,
            length=10,
            default="NPS-1002",
        ))
        header.add_field(FormField(
            name="submit_date",
            label="제출일자",
            field_type=FieldType.DATE,
            required=True,
            length=8,
        ))
        self.add_section(header)

        # Company section
        company = FormSection(name="company", label="사업장 정보")
        company.add_field(FormField(
            name="business_no",
            label="사업자등록번호",
            field_type=FieldType.STRING,
            required=True,
            length=10,
            validators=[validate_business_no],
        ))
        company.add_field(FormField(
            name="workplace_no",
            label="사업장관리번호",
            field_type=FieldType.STRING,
            required=True,
            length=13,
        ))
        company.add_field(FormField(
            name="company_name",
            label="사업장명",
            field_type=FieldType.STRING,
            required=True,
            length=40,
        ))
        self.add_section(company)

        # Detail section
        detail = FormSection(
            name="employees",
            label="상실자 명세",
            repeating=True,
            min_items=1,
        )
        detail.add_field(FormField(
            name="resident_no",
            label="주민등록번호",
            field_type=FieldType.STRING,
            required=True,
            length=13,
            validators=[validate_resident_no],
        ))
        detail.add_field(FormField(
            name="name",
            label="성명",
            field_type=FieldType.STRING,
            required=True,
            length=20,
        ))
        detail.add_field(FormField(
            name="loss_date",
            label="상실일",
            field_type=FieldType.DATE,
            required=True,
            length=8,
        ))
        detail.add_field(FormField(
            name="loss_reason",
            label="상실사유",
            field_type=FieldType.CHOICE,
            required=True,
            length=2,
            choices=[
                ("11", "퇴직"),
                ("12", "사망"),
                ("13", "60세 도달"),
                ("14", "국적상실/국외이주"),
                ("21", "타사업장 취득"),
                ("22", "공무원/교직원 임용"),
                ("31", "기타"),
            ],
        ))
        detail.add_field(FormField(
            name="final_income",
            label="최종기준소득월액",
            field_type=FieldType.INTEGER,
            required=True,
            length=15,
            alignment=FieldAlignment.RIGHT,
            pad_char="0",
        ))
        self.add_section(detail)


class NPSChangeForm(BaseForm):
    """
    NPS Change Form (국민연금 내용변경신고서).

    Used for reporting changes in employee pension information.
    """

    @property
    def form_code(self) -> str:
        return "NPS-1003"

    @property
    def form_name(self) -> str:
        return "국민연금 내용변경신고서"

    @property
    def insurance_type(self) -> str:
        return "NPS"

    def _setup_form(self) -> None:
        """Set up change form structure."""
        header = FormSection(name="header", label="신고서 헤더")
        header.add_field(FormField(
            name="form_code",
            label="서식코드",
            field_type=FieldType.STRING,
            length=10,
            default="NPS-1003",
        ))
        header.add_field(FormField(
            name="submit_date",
            label="제출일자",
            field_type=FieldType.DATE,
            required=True,
            length=8,
        ))
        self.add_section(header)

        company = FormSection(name="company", label="사업장 정보")
        company.add_field(FormField(
            name="business_no",
            label="사업자등록번호",
            field_type=FieldType.STRING,
            required=True,
            length=10,
            validators=[validate_business_no],
        ))
        company.add_field(FormField(
            name="workplace_no",
            label="사업장관리번호",
            field_type=FieldType.STRING,
            required=True,
            length=13,
        ))
        self.add_section(company)

        detail = FormSection(
            name="changes",
            label="변경내역",
            repeating=True,
            min_items=1,
        )
        detail.add_field(FormField(
            name="resident_no",
            label="주민등록번호",
            field_type=FieldType.STRING,
            required=True,
            length=13,
            validators=[validate_resident_no],
        ))
        detail.add_field(FormField(
            name="name",
            label="성명",
            field_type=FieldType.STRING,
            required=True,
            length=20,
        ))
        detail.add_field(FormField(
            name="change_type",
            label="변경유형",
            field_type=FieldType.CHOICE,
            required=True,
            length=2,
            choices=[
                ("01", "기준소득월액 변경"),
                ("02", "성명 변경"),
                ("03", "주민등록번호 정정"),
                ("04", "취득일 정정"),
                ("05", "상실일 정정"),
            ],
        ))
        detail.add_field(FormField(
            name="change_date",
            label="변경일",
            field_type=FieldType.DATE,
            required=True,
            length=8,
        ))
        detail.add_field(FormField(
            name="before_value",
            label="변경전",
            field_type=FieldType.STRING,
            length=50,
        ))
        detail.add_field(FormField(
            name="after_value",
            label="변경후",
            field_type=FieldType.STRING,
            required=True,
            length=50,
        ))
        detail.add_field(FormField(
            name="reason",
            label="변경사유",
            field_type=FieldType.STRING,
            length=100,
        ))
        self.add_section(detail)
