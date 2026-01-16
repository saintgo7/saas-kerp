"""
Insurance Provider Implementations

각 보험 기관별 EDI 연동 프로바이더
- NPS: 국민연금공단 (National Pension Service)
- NHIS: 건강보험공단 (National Health Insurance Service)
- EI: 고용산재보험 (Employment Insurance / Industrial Accident Compensation Insurance)
"""
from .base import BaseProvider
from .nps import NPSProvider
from .nhis import NHISProvider
from .ei import EIProvider

__all__ = [
    "BaseProvider",
    "NPSProvider",
    "NHISProvider",
    "EIProvider",
]
