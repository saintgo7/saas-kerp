package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TaxInvoiceType represents the type of tax invoice.
type TaxInvoiceType string

const (
	TaxInvoiceTypeSales    TaxInvoiceType = "sales"
	TaxInvoiceTypePurchase TaxInvoiceType = "purchase"
)

// TaxInvoiceStatus represents the status of a tax invoice.
type TaxInvoiceStatus string

const (
	TaxInvoiceStatusDraft       TaxInvoiceStatus = "draft"
	TaxInvoiceStatusIssued      TaxInvoiceStatus = "issued"
	TaxInvoiceStatusTransmitted TaxInvoiceStatus = "transmitted"
	TaxInvoiceStatusConfirmed   TaxInvoiceStatus = "confirmed"
	TaxInvoiceStatusCancelled   TaxInvoiceStatus = "cancelled"
	TaxInvoiceStatusRejected    TaxInvoiceStatus = "rejected"
)

// TaxInvoice represents a tax invoice (세금계산서).
type TaxInvoice struct {
	ID        uuid.UUID `json:"id"`
	CompanyID uuid.UUID `json:"company_id"`

	// Invoice identification
	InvoiceNumber string           `json:"invoice_number"`
	InvoiceType   TaxInvoiceType   `json:"invoice_type"`
	IssueDate     time.Time        `json:"issue_date"`
	Status        TaxInvoiceStatus `json:"status"`

	// Supplier (seller) information
	SupplierBusinessNumber string `json:"supplier_business_number"`
	SupplierName           string `json:"supplier_name"`
	SupplierCEOName        string `json:"supplier_ceo_name,omitempty"`
	SupplierAddress        string `json:"supplier_address,omitempty"`
	SupplierBusinessType   string `json:"supplier_business_type,omitempty"`
	SupplierBusinessItem   string `json:"supplier_business_item,omitempty"`
	SupplierEmail          string `json:"supplier_email,omitempty"`

	// Buyer (purchaser) information
	BuyerBusinessNumber string `json:"buyer_business_number"`
	BuyerName           string `json:"buyer_name"`
	BuyerCEOName        string `json:"buyer_ceo_name,omitempty"`
	BuyerAddress        string `json:"buyer_address,omitempty"`
	BuyerBusinessType   string `json:"buyer_business_type,omitempty"`
	BuyerBusinessItem   string `json:"buyer_business_item,omitempty"`
	BuyerEmail          string `json:"buyer_email,omitempty"`

	// Amount information
	SupplyAmount int64 `json:"supply_amount"`
	TaxAmount    int64 `json:"tax_amount"`
	TotalAmount  int64 `json:"total_amount"`

	// NTS information
	NTSConfirmNumber  string     `json:"nts_confirm_number,omitempty"`
	NTSTransmittedAt  *time.Time `json:"nts_transmitted_at,omitempty"`
	NTSConfirmedAt    *time.Time `json:"nts_confirmed_at,omitempty"`

	// ASP information
	ASPProvider  string `json:"asp_provider,omitempty"`
	ASPInvoiceID string `json:"asp_invoice_id,omitempty"`

	// Linked voucher
	VoucherID *uuid.UUID `json:"voucher_id,omitempty"`

	// Items
	Items []TaxInvoiceItem `json:"items,omitempty"`

	// Metadata
	Remarks   string     `json:"remarks,omitempty"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	UpdatedBy *uuid.UUID `json:"updated_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// TaxInvoiceItem represents a line item in a tax invoice.
type TaxInvoiceItem struct {
	ID             uuid.UUID  `json:"id"`
	TaxInvoiceID   uuid.UUID  `json:"tax_invoice_id"`
	CompanyID      uuid.UUID  `json:"company_id"`
	SequenceNumber int        `json:"sequence_number"`
	SupplyDate     *time.Time `json:"supply_date,omitempty"`
	Description    string     `json:"description"`
	Specification  string     `json:"specification,omitempty"`
	Quantity       float64    `json:"quantity"`
	UnitPrice      float64    `json:"unit_price"`
	Amount         int64      `json:"amount"`
	TaxAmount      int64      `json:"tax_amount"`
	Remarks        string     `json:"remarks,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// TaxInvoiceHistory represents a status change record.
type TaxInvoiceHistory struct {
	ID             uuid.UUID        `json:"id"`
	TaxInvoiceID   uuid.UUID        `json:"tax_invoice_id"`
	CompanyID      uuid.UUID        `json:"company_id"`
	PreviousStatus TaxInvoiceStatus `json:"previous_status,omitempty"`
	NewStatus      TaxInvoiceStatus `json:"new_status"`
	ChangedBy      *uuid.UUID       `json:"changed_by,omitempty"`
	ChangeReason   string           `json:"change_reason,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
}

// HometaxSession represents a Hometax scraper session.
type HometaxSession struct {
	ID             uuid.UUID  `json:"id"`
	CompanyID      uuid.UUID  `json:"company_id"`
	SessionID      string     `json:"session_id"`
	BusinessNumber string     `json:"business_number"`
	AuthType       string     `json:"auth_type"`
	ExpiresAt      time.Time  `json:"expires_at"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	IsActive       bool       `json:"is_active"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// PopbillConfig represents Popbill API configuration for a company.
type PopbillConfig struct {
	ID                 uuid.UUID  `json:"id"`
	CompanyID          uuid.UUID  `json:"company_id"`
	LinkID             string     `json:"link_id"`
	SecretKeyEncrypted []byte     `json:"-"`
	IsSandbox          bool       `json:"is_sandbox"`
	IsActive           bool       `json:"is_active"`
	MonthlyQuota       int        `json:"monthly_quota"`
	MonthlyUsed        int        `json:"monthly_used"`
	QuotaResetAt       *time.Time `json:"quota_reset_at,omitempty"`
	CreatedBy          *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// TaxInvoiceSummary represents aggregated tax invoice data.
type TaxInvoiceSummary struct {
	SalesCount         int64 `json:"sales_count"`
	PurchaseCount      int64 `json:"purchase_count"`
	SalesSupplyTotal   int64 `json:"sales_supply_total"`
	SalesTaxTotal      int64 `json:"sales_tax_total"`
	PurchaseSupplyTotal int64 `json:"purchase_supply_total"`
	PurchaseTaxTotal   int64 `json:"purchase_tax_total"`
}

// Validate validates the tax invoice.
func (t *TaxInvoice) Validate() error {
	if t.InvoiceNumber == "" {
		return fmt.Errorf("invoice_number is required")
	}
	if len(t.SupplierBusinessNumber) != 10 {
		return fmt.Errorf("supplier_business_number must be 10 digits")
	}
	if len(t.BuyerBusinessNumber) != 10 {
		return fmt.Errorf("buyer_business_number must be 10 digits")
	}
	if t.SupplierName == "" {
		return fmt.Errorf("supplier_name is required")
	}
	if t.BuyerName == "" {
		return fmt.Errorf("buyer_name is required")
	}
	if t.TotalAmount != t.SupplyAmount+t.TaxAmount {
		return fmt.Errorf("total_amount must equal supply_amount + tax_amount")
	}
	return nil
}

// CanBeModified checks if the invoice can be modified.
func (t *TaxInvoice) CanBeModified() bool {
	return t.Status == TaxInvoiceStatusDraft
}

// CanBeCancelled checks if the invoice can be cancelled.
func (t *TaxInvoice) CanBeCancelled() bool {
	return t.Status == TaxInvoiceStatusIssued || t.Status == TaxInvoiceStatusTransmitted
}

// IsTransmitted checks if the invoice has been transmitted to NTS.
func (t *TaxInvoice) IsTransmitted() bool {
	return t.Status == TaxInvoiceStatusTransmitted || t.Status == TaxInvoiceStatusConfirmed
}
