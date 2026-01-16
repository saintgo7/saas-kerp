"""
Data models for Hometax tax invoice operations.
"""
from datetime import datetime
from decimal import Decimal
from enum import Enum
from typing import Optional

from pydantic import BaseModel, Field


class AuthType(str, Enum):
    """Authentication type for Hometax login."""

    CERTIFICATE = "certificate"  # Public key certificate (NPKI)
    SIMPLE_AUTH = "simple_auth"  # Simple authentication (ARS, etc.)
    ID_PASSWORD = "id_password"  # ID/Password login


class InvoiceType(str, Enum):
    """Tax invoice type."""

    SALES = "sales"  # Sales invoice (issued)
    PURCHASE = "purchase"  # Purchase invoice (received)


class InvoiceStatus(str, Enum):
    """Tax invoice status."""

    DRAFT = "draft"
    ISSUED = "issued"
    TRANSMITTED = "transmitted"  # Sent to NTS
    CONFIRMED = "confirmed"  # Confirmed by NTS
    CANCELLED = "cancelled"


class HometaxSession(BaseModel):
    """Hometax session information."""

    session_id: str
    business_number: str
    company_name: str
    expires_at: datetime
    auth_type: AuthType
    created_at: datetime = Field(default_factory=datetime.now)


class TaxInvoiceItem(BaseModel):
    """Individual item in a tax invoice."""

    sequence: int = 1
    supply_date: Optional[datetime] = None
    description: str
    specification: str = ""
    quantity: Decimal = Decimal("1")
    unit_price: Decimal
    amount: Decimal
    tax_amount: Decimal
    remarks: str = ""


class TaxInvoice(BaseModel):
    """Tax invoice (Sae-Geum-Gye-San-Seo) model."""

    # Invoice identification
    invoice_number: str
    issue_date: datetime
    invoice_type: InvoiceType = InvoiceType.SALES
    status: InvoiceStatus = InvoiceStatus.DRAFT

    # Supplier (seller) information
    supplier_business_number: str = Field(..., min_length=10, max_length=10)
    supplier_name: str
    supplier_ceo_name: str = ""
    supplier_address: str = ""
    supplier_business_type: str = ""
    supplier_business_item: str = ""
    supplier_email: str = ""

    # Buyer (purchaser) information
    buyer_business_number: str = Field(..., min_length=10, max_length=10)
    buyer_name: str
    buyer_ceo_name: str = ""
    buyer_address: str = ""
    buyer_business_type: str = ""
    buyer_business_item: str = ""
    buyer_email: str = ""

    # Amount information
    supply_amount: Decimal  # Total supply amount (before tax)
    tax_amount: Decimal  # VAT amount
    total_amount: Decimal  # Total amount (supply + tax)

    # Items
    items: list[TaxInvoiceItem] = []

    # NTS (National Tax Service) information
    nts_confirm_number: Optional[str] = None  # National Tax Service confirmation number
    nts_transmitted_at: Optional[datetime] = None

    # Metadata
    remarks: str = ""
    created_at: datetime = Field(default_factory=datetime.now)
    updated_at: datetime = Field(default_factory=datetime.now)


class IssuedInvoiceResult(BaseModel):
    """Result of issuing a tax invoice."""

    success: bool
    invoice_number: str
    issue_date: datetime
    nts_confirm_number: Optional[str] = None
    error_message: Optional[str] = None


class TaxInvoiceSearchCriteria(BaseModel):
    """Search criteria for tax invoices."""

    start_date: datetime
    end_date: datetime
    invoice_type: Optional[InvoiceType] = None
    business_number: Optional[str] = None
    min_amount: Optional[Decimal] = None
    max_amount: Optional[Decimal] = None
    status: Optional[InvoiceStatus] = None
