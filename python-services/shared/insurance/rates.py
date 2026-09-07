"""
4대보험 요율표와 보험료 계산.

설계 원칙
--------
1. **요율을 코드에 흩어 놓지 않습니다.** 모든 값은 아래 `RATE_TABLE` /
   `NPS_INCOME_BOUNDS` 한 곳에만 있고, 각 항목에 근거(고시·시행령)를 주석으로
   답니다. 요율이 바뀌면 표에 연도를 추가할 뿐 계산 코드는 건드리지 않습니다.

2. **모르는 연도는 예외입니다.** 표에 없는 연도를 조회하면 `RateTableError` 를
   던집니다. 가장 가까운 연도로 대체하거나 마지막 값을 이어 쓰지 않습니다 --
   보험료는 법정 금액이고, 틀린 요율로 계산한 값은 과소·과다 납부와
   연말정산 정정으로 이어집니다. 조용히 틀리느니 멈추는 편이 낫습니다.

3. **산재보험료율은 표에 없습니다.** 업종별로 다르고(사업장마다 다름) 매년
   고용노동부 고시로 정해지므로, 호출자가 사업장 요율을 명시적으로 넘겨야
   합니다. 평균값을 기본값으로 두면 사업장마다 틀린 금액이 나옵니다.

4. **금액은 `Decimal`, 원 단위 절사** -- 4대보험 보험료는 원 단위 미만을
   버립니다(국민연금·건강보험 모두 10원 미만 절사 규정이 별도로 있어
   `rounding_unit` 로 조정합니다).

주의: 이 표는 2024·2025년만 담고 있습니다. **2026년 이후 요율은 확인되지
않았으므로 넣지 않았습니다.** 2026년 값을 쓰려면 아래 출처에서 확인한 뒤
표에 추가하십시오. 확인 없이 추정치를 넣는 것이 이 모듈이 막으려는 바로
그 실패입니다.
"""
from dataclasses import dataclass
from datetime import date
from decimal import Decimal, ROUND_DOWN
from typing import Optional


class RateTableError(LookupError):
    """요율표에 해당 연도·기간의 값이 없을 때 발생합니다."""


@dataclass(frozen=True)
class InsuranceRates:
    """한 연도의 4대보험 요율.

    모든 요율은 비율(예: 0.09 = 9%)이며, 보수월액/기준소득월액에 곱합니다.
    """

    year: int

    # 국민연금 (National Pension Service)
    # 근거: 국민연금법 제88조. 1998년 이후 9.0% 고정, 사용자·근로자 각 4.5%.
    nps_employee: Decimal
    nps_employer: Decimal

    # 건강보험 (National Health Insurance Service)
    # 근거: 국민건강보험법 시행령 제44조 및 매년 건강보험정책심의위원회 의결.
    # 직장가입자 보험료율의 절반씩 근로자·사용자가 부담합니다.
    nhis_employee: Decimal
    nhis_employer: Decimal

    # 장기요양보험 (Long-term care)
    # 근거: 노인장기요양보험법 시행령 제4조.
    # 건강보험료에 대한 비율입니다(보수월액이 아니라 **건강보험료** 기준).
    ltc_rate_of_nhis: Decimal

    # 고용보험 - 실업급여 (Employment Insurance, unemployment benefit)
    # 근거: 고용보험법 시행령 제12조. 근로자·사업주 각 0.9% (2022.07.01~).
    ei_employee: Decimal
    ei_employer: Decimal

    # 고용보험 - 고용안정·직업능력개발사업 (사업주 전액 부담)
    # 근거: 고용보험법 시행령 제12조. 사업장 규모별로 다릅니다.
    ei_stability_under_150: Decimal
    ei_stability_150_priority: Decimal
    ei_stability_150_to_999: Decimal
    ei_stability_1000_plus: Decimal

    # 산재보험 (Workers' Compensation)
    # 업종별 요율이므로 표에 값을 두지 않습니다. 출퇴근재해 요율만 전 업종 공통.
    # 근거: 고용노동부 「사업종류별 산재보험료율」 고시.
    wci_commute_accident: Decimal

    # 출처 표기
    source: str

    @property
    def nhis_total(self) -> Decimal:
        """건강보험 총 요율 (근로자 + 사용자)."""
        return self.nhis_employee + self.nhis_employer

    @property
    def nps_total(self) -> Decimal:
        """국민연금 총 요율 (근로자 + 사용자)."""
        return self.nps_employee + self.nps_employer


# ---------------------------------------------------------------------------
# 연도별 요율표
#
# 값을 추가할 때는 반드시 `source` 에 확인한 고시/시행령과 확인 일자를 적으십시오.
# ---------------------------------------------------------------------------
RATE_TABLE: dict[int, InsuranceRates] = {
    2024: InsuranceRates(
        year=2024,
        # 국민연금 9.0% (근로자 4.5% + 사용자 4.5%)
        nps_employee=Decimal("0.045"),
        nps_employer=Decimal("0.045"),
        # 건강보험 7.09% (근로자 3.545% + 사용자 3.545%)
        nhis_employee=Decimal("0.03545"),
        nhis_employer=Decimal("0.03545"),
        # 장기요양보험료율: 건강보험료의 12.95%
        ltc_rate_of_nhis=Decimal("0.1295"),
        # 실업급여 1.8% (근로자 0.9% + 사업주 0.9%)
        ei_employee=Decimal("0.009"),
        ei_employer=Decimal("0.009"),
        # 고용안정·직업능력개발사업 (사업주 부담)
        ei_stability_under_150=Decimal("0.0025"),
        ei_stability_150_priority=Decimal("0.0045"),
        ei_stability_150_to_999=Decimal("0.0065"),
        ei_stability_1000_plus=Decimal("0.0085"),
        # 출퇴근재해 요율 0.06%
        wci_commute_accident=Decimal("0.0006"),
        source=(
            "국민연금법 제88조; 국민건강보험법 시행령 제44조(2024년 보험료율 7.09%); "
            "노인장기요양보험법 시행령 제4조(2024년 12.95%); "
            "고용보험법 시행령 제12조(2022.07.01 개정)"
        ),
    ),
    2025: InsuranceRates(
        year=2025,
        nps_employee=Decimal("0.045"),
        nps_employer=Decimal("0.045"),
        # 2025년 건강보험료율은 2024년과 동일하게 동결(7.09%)
        nhis_employee=Decimal("0.03545"),
        nhis_employer=Decimal("0.03545"),
        # 2025년 장기요양보험료율도 12.95% 동결
        ltc_rate_of_nhis=Decimal("0.1295"),
        ei_employee=Decimal("0.009"),
        ei_employer=Decimal("0.009"),
        ei_stability_under_150=Decimal("0.0025"),
        ei_stability_150_priority=Decimal("0.0045"),
        ei_stability_150_to_999=Decimal("0.0065"),
        ei_stability_1000_plus=Decimal("0.0085"),
        wci_commute_accident=Decimal("0.0006"),
        source=(
            "국민연금법 제88조; 2025년 건강보험료율 동결(7.09%); "
            "2025년 장기요양보험료율 동결(건강보험료의 12.95%); "
            "고용보험법 시행령 제12조"
        ),
    ),
    # 2026년 이후: 확인되지 않았습니다. 고시를 확인한 뒤 추가하십시오.
    # 추정치를 넣지 마십시오 -- 조회는 RateTableError 로 실패하는 편이 낫습니다.
}


# ---------------------------------------------------------------------------
# 국민연금 기준소득월액 상·하한
#
# 매년 7월 1일에 개정되어 다음 해 6월 30일까지 적용됩니다(국민연금법 시행령
# 제5조). 따라서 연도가 아니라 **적용 기간**으로 관리합니다.
# ---------------------------------------------------------------------------
@dataclass(frozen=True)
class IncomeBounds:
    """국민연금 기준소득월액 상·하한과 그 적용 기간."""

    effective_from: date
    effective_to: date
    lower: Decimal
    upper: Decimal
    source: str


NPS_INCOME_BOUNDS: tuple[IncomeBounds, ...] = (
    IncomeBounds(
        effective_from=date(2023, 7, 1),
        effective_to=date(2024, 6, 30),
        lower=Decimal("370000"),
        upper=Decimal("5900000"),
        source="보건복지부 고시 - 2023.7.1~2024.6.30 국민연금 기준소득월액 상·하한액",
    ),
    IncomeBounds(
        effective_from=date(2024, 7, 1),
        effective_to=date(2025, 6, 30),
        lower=Decimal("390000"),
        upper=Decimal("6170000"),
        source="보건복지부 고시 - 2024.7.1~2025.6.30 국민연금 기준소득월액 상·하한액",
    ),
    IncomeBounds(
        effective_from=date(2025, 7, 1),
        effective_to=date(2026, 6, 30),
        lower=Decimal("400000"),
        upper=Decimal("6370000"),
        source="보건복지부 고시 - 2025.7.1~2026.6.30 국민연금 기준소득월액 상·하한액",
    ),
    # 2026.7.1 이후 구간은 고시 확인 후 추가하십시오.
)


class WorkplaceSize:
    """고용안정·직업능력개발사업 요율 구간."""

    UNDER_150 = "under_150"
    PRIORITY_150_PLUS = "priority_150_plus"  # 150인 이상 우선지원대상기업
    FROM_150_TO_999 = "from_150_to_999"
    OVER_1000 = "over_1000"


@dataclass(frozen=True)
class ContributionBreakdown:
    """한 근로자의 월 4대보험료 산출 결과."""

    nps_employee: Decimal
    nps_employer: Decimal
    nhis_employee: Decimal
    nhis_employer: Decimal
    ltc_employee: Decimal
    ltc_employer: Decimal
    ei_employee: Decimal
    ei_employer: Decimal
    wci_employer: Decimal

    # 국민연금 계산에 실제로 사용된 기준소득월액 (상·하한 적용 후)
    nps_taxable_income: Decimal

    @property
    def employee_total(self) -> Decimal:
        """근로자 부담 합계."""
        return (
            self.nps_employee
            + self.nhis_employee
            + self.ltc_employee
            + self.ei_employee
        )

    @property
    def employer_total(self) -> Decimal:
        """사업주 부담 합계."""
        return (
            self.nps_employer
            + self.nhis_employer
            + self.ltc_employer
            + self.ei_employer
            + self.wci_employer
        )

    @property
    def total(self) -> Decimal:
        """총 보험료."""
        return self.employee_total + self.employer_total


def get_rates(year: int) -> InsuranceRates:
    """해당 연도의 4대보험 요율을 반환합니다.

    Args:
        year: 적용 연도

    Returns:
        해당 연도의 요율

    Raises:
        RateTableError: 표에 없는 연도인 경우. 가까운 연도로 대체하지 않습니다.
    """
    rates = RATE_TABLE.get(year)
    if rates is None:
        available = ", ".join(str(y) for y in sorted(RATE_TABLE))
        raise RateTableError(
            f"{year}년 4대보험 요율이 요율표에 없습니다. 현재 등록된 연도: "
            f"{available}. 보건복지부·고용노동부 고시로 확인한 값을 "
            f"shared/insurance/rates.py 의 RATE_TABLE 에 출처와 함께 추가하십시오. "
            f"추정치나 직전 연도 값으로 대체하지 마십시오."
        )
    return rates


def get_nps_income_bounds(on: date) -> IncomeBounds:
    """해당 시점에 적용되는 국민연금 기준소득월액 상·하한을 반환합니다.

    Args:
        on: 기준일

    Returns:
        적용 구간의 상·하한

    Raises:
        RateTableError: 해당 시점을 포함하는 구간이 없는 경우
    """
    for bounds in NPS_INCOME_BOUNDS:
        if bounds.effective_from <= on <= bounds.effective_to:
            return bounds

    raise RateTableError(
        f"{on.isoformat()} 에 적용되는 국민연금 기준소득월액 상·하한이 "
        f"요율표에 없습니다. 보건복지부 고시를 확인해 "
        f"shared/insurance/rates.py 의 NPS_INCOME_BOUNDS 에 추가하십시오."
    )


def apply_nps_income_bounds(monthly_income: Decimal, on: date) -> Decimal:
    """기준소득월액에 국민연금 상·하한을 적용합니다.

    국민연금은 기준소득월액이 상한을 넘으면 상한액으로, 하한에 못 미치면
    하한액으로 보험료를 산정합니다.

    Args:
        monthly_income: 신고된 기준소득월액
        on: 적용 기준일

    Returns:
        상·하한을 적용한 기준소득월액
    """
    bounds = get_nps_income_bounds(on)
    return min(max(monthly_income, bounds.lower), bounds.upper)


def _truncate(amount: Decimal, unit: int) -> Decimal:
    """보험료를 지정한 단위 미만에서 절사합니다.

    Args:
        amount: 절사 전 금액
        unit: 절사 단위(원). 예: 10 이면 10원 미만 절사

    Returns:
        절사된 금액
    """
    if unit <= 1:
        return amount.quantize(Decimal("1"), rounding=ROUND_DOWN)
    unit_dec = Decimal(unit)
    return (amount / unit_dec).quantize(Decimal("1"), rounding=ROUND_DOWN) * unit_dec


def calculate_contributions(
    monthly_income: Decimal,
    *,
    year: int,
    on: Optional[date] = None,
    wci_rate: Optional[Decimal] = None,
    workplace_size: str = WorkplaceSize.UNDER_150,
    include_commute_accident: bool = True,
    rounding_unit: int = 10,
) -> ContributionBreakdown:
    """한 근로자의 월 4대보험료를 산출합니다.

    Args:
        monthly_income: 보수월액 / 기준소득월액 (원)
        year: 요율을 적용할 연도
        on: 국민연금 상·하한 적용 기준일. 생략하면 `year` 의 7월 1일을 씁니다
            (상·하한은 7월에 개정되므로 연 중반 기준이 안전합니다).
        wci_rate: 사업장의 산재보험료율. **업종별로 다르므로 기본값이 없습니다.**
            생략하면 산재보험료는 0으로 두고, 호출자가 따로 계산해야 합니다.
        workplace_size: 고용안정·직업능력개발사업 요율 구간
        include_commute_accident: 출퇴근재해 요율 포함 여부
        rounding_unit: 절사 단위(원)

    Returns:
        항목별 부담액

    Raises:
        RateTableError: 요율표나 상·하한표에 해당 연도/기간이 없는 경우
        ValueError: 보수월액이 음수인 경우
    """
    if monthly_income < 0:
        raise ValueError("보수월액은 음수일 수 없습니다")

    rates = get_rates(year)
    reference_date = on or date(year, 7, 1)

    # 국민연금만 상·하한이 있습니다.
    nps_base = apply_nps_income_bounds(monthly_income, reference_date)

    nps_employee = _truncate(nps_base * rates.nps_employee, rounding_unit)
    nps_employer = _truncate(nps_base * rates.nps_employer, rounding_unit)

    nhis_employee = _truncate(monthly_income * rates.nhis_employee, rounding_unit)
    nhis_employer = _truncate(monthly_income * rates.nhis_employer, rounding_unit)

    # 장기요양보험료는 보수월액이 아니라 건강보험료에 요율을 곱합니다.
    ltc_employee = _truncate(nhis_employee * rates.ltc_rate_of_nhis, rounding_unit)
    ltc_employer = _truncate(nhis_employer * rates.ltc_rate_of_nhis, rounding_unit)

    ei_employee = _truncate(monthly_income * rates.ei_employee, rounding_unit)

    stability_rate = {
        WorkplaceSize.UNDER_150: rates.ei_stability_under_150,
        WorkplaceSize.PRIORITY_150_PLUS: rates.ei_stability_150_priority,
        WorkplaceSize.FROM_150_TO_999: rates.ei_stability_150_to_999,
        WorkplaceSize.OVER_1000: rates.ei_stability_1000_plus,
    }.get(workplace_size)

    if stability_rate is None:
        raise ValueError(f"알 수 없는 사업장 규모 구분: {workplace_size!r}")

    ei_employer = _truncate(
        monthly_income * (rates.ei_employer + stability_rate), rounding_unit
    )

    # 산재보험은 전액 사업주 부담이며 업종별 요율이 필요합니다.
    if wci_rate is None:
        wci_employer = Decimal("0")
    else:
        effective_wci = wci_rate
        if include_commute_accident:
            effective_wci += rates.wci_commute_accident
        wci_employer = _truncate(monthly_income * effective_wci, rounding_unit)

    return ContributionBreakdown(
        nps_employee=nps_employee,
        nps_employer=nps_employer,
        nhis_employee=nhis_employee,
        nhis_employer=nhis_employer,
        ltc_employee=ltc_employee,
        ltc_employer=ltc_employer,
        ei_employee=ei_employee,
        ei_employer=ei_employer,
        wci_employer=wci_employer,
        nps_taxable_income=nps_base,
    )
