package domain

import (
	"errors"
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

	// VATCategory 과세구분. Transient: there is no column for it yet, so it is
	// marked gorm:"-" and is not persisted. When it is left empty the category
	// is inferred from the amounts (see DeclaredVATCategory); when a caller
	// sets it, the 10% rate is enforced instead of inferred, which is what
	// catches a taxable supply declared with zero tax.
	//
	// HANDOFF: persisting this needs a tax_invoices.vat_category column (DB)
	// and a field on the create/update DTO (handler) - neither is in this
	// package.
	VATCategory VATCategory `gorm:"-" json:"vat_category,omitempty"`

	// NTS information
	NTSConfirmNumber string     `json:"nts_confirm_number,omitempty"`
	NTSTransmittedAt *time.Time `json:"nts_transmitted_at,omitempty"`
	NTSConfirmedAt   *time.Time `json:"nts_confirmed_at,omitempty"`

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
	SalesCount          int64 `json:"sales_count"`
	PurchaseCount       int64 `json:"purchase_count"`
	SalesSupplyTotal    int64 `json:"sales_supply_total"`
	SalesTaxTotal       int64 `json:"sales_tax_total"`
	PurchaseSupplyTotal int64 `json:"purchase_supply_total"`
	PurchaseTaxTotal    int64 `json:"purchase_tax_total"`
}

// Tax invoice validation errors.
var (
	// ErrTaxInvoiceNotFound is the sentinel the repository returns for a
	// genuine miss. It exists so callers can tell "no such invoice" apart from
	// "the database is unreachable"; the message is unchanged from the ad hoc
	// error it replaces so any string matching still behaves the same.
	ErrTaxInvoiceNotFound = errors.New("tax invoice not found")

	ErrTaxInvoiceNumberRequired  = errors.New("invoice_number is required")
	ErrInvalidBusinessNumber     = errors.New("사업자등록번호가 올바르지 않습니다")
	ErrTaxInvoiceNameRequired    = errors.New("supplier_name and buyer_name are required")
	ErrTaxInvoiceNegativeAmount  = errors.New("세금계산서 금액은 음수일 수 없습니다")
	ErrTaxInvoiceZeroAmount      = errors.New("세금계산서 공급가액은 0보다 커야 합니다")
	ErrTaxInvoiceTotalMismatch   = errors.New("total_amount must equal supply_amount + tax_amount")
	ErrTaxInvoiceItemSumMismatch = errors.New("품목 합계가 세금계산서 헤더 금액과 다릅니다")
	ErrTaxInvoiceNoItems         = errors.New("세금계산서에 품목이 없습니다")
	ErrTaxInvoiceItemInvalid     = errors.New("품목 금액이 올바르지 않습니다")
)

// businessNumberWeights are the multipliers used by the 사업자등록번호
// check digit algorithm.
var businessNumberWeights = [9]int{1, 3, 7, 1, 3, 7, 1, 3, 5}

// ValidateBusinessNumber verifies a 10 digit 사업자등록번호, check digit
// included.
//
// The previous check was len(...) != 10, which let "abcdefghij" through and
// transmitted it to the NTS. The check digit algorithm is:
//
//	sum = Σ digit[i] x weight[i]  (i = 0..8)
//	sum += (digit[8] x 5) / 10    (integer division)
//	check = (10 - sum % 10) % 10  must equal digit[9]
//
// CONFIRM / 확인 필요: 위 알고리즘은 널리 쓰이는 구현이나 국세청이 공표한
// 사양과 대조하지는 않았다. 발급 전 차단에 쓰이므로 오탐이 없는지 실제
// 사업자등록번호 표본으로 확인할 것.
func ValidateBusinessNumber(businessNumber string) error {
	if len(businessNumber) != 10 {
		return ErrInvalidBusinessNumber
	}

	var digits [10]int
	for i := 0; i < 10; i++ {
		c := businessNumber[i]
		if c < '0' || c > '9' {
			return ErrInvalidBusinessNumber
		}
		digits[i] = int(c - '0')
	}

	sum := 0
	for i := 0; i < 9; i++ {
		sum += digits[i] * businessNumberWeights[i]
	}
	sum += digits[8] * 5 / 10

	if (10-sum%10)%10 != digits[9] {
		return ErrInvalidBusinessNumber
	}
	return nil
}

// DeclaredVATCategory returns the VAT treatment the stored amounts imply.
//
// The table has no column for the category, so it is inferred: a non-zero tax
// amount means the supply was treated as 과세, and a zero tax amount means it
// was treated as 영세율 or 면세 (which are indistinguishable from the amounts
// alone and are therefore both reported as VATCategoryZeroRated). Set
// VATCategory explicitly to have the rate enforced rather than inferred.
func (t *TaxInvoice) DeclaredVATCategory() VATCategory {
	if t.VATCategory.IsValid() {
		return t.VATCategory
	}
	if t.TaxAmount > 0 {
		return VATCategoryTaxable
	}
	return VATCategoryZeroRated
}

// Validate validates the tax invoice header.
func (t *TaxInvoice) Validate() error {
	if t.InvoiceNumber == "" {
		return ErrTaxInvoiceNumberRequired
	}
	if err := ValidateBusinessNumber(t.SupplierBusinessNumber); err != nil {
		return fmt.Errorf("supplier_business_number: %w", err)
	}
	if err := ValidateBusinessNumber(t.BuyerBusinessNumber); err != nil {
		return fmt.Errorf("buyer_business_number: %w", err)
	}
	if t.SupplierName == "" || t.BuyerName == "" {
		return ErrTaxInvoiceNameRequired
	}

	// Negative amounts are rejected here rather than left to the database
	// CHECK constraint added in migration 000019, so the caller gets a
	// validation error instead of a raw 500.
	if t.SupplyAmount < 0 || t.TaxAmount < 0 || t.TotalAmount < 0 {
		return ErrTaxInvoiceNegativeAmount
	}
	if t.SupplyAmount == 0 {
		return ErrTaxInvoiceZeroAmount
	}
	if t.TotalAmount != t.SupplyAmount+t.TaxAmount {
		return ErrTaxInvoiceTotalMismatch
	}

	// The tax must be what the applicable rate produces. Without this the
	// header amounts were stored and transmitted exactly as the client sent
	// them: supply_amount 10,000,000 with tax_amount 0 was accepted and filed
	// as zero output VAT.
	if err := ValidateVATAmount(t.SupplyAmount, t.TaxAmount, t.DeclaredVATCategory()); err != nil {
		return err
	}

	return nil
}

// ValidateItems reconciles the line items against the header.
//
// SUM(items.amount) must equal supply_amount and SUM(items.tax_amount) must
// equal tax_amount. Neither was checked before, so an invoice could carry
// items totalling 10,000,000 with 1,000,000 of tax while the header declared
// 10,000,000 supply and 0 tax; GetSummary aggregates the header, so the VAT
// return came out wrong while the printed invoice looked right.
func (t *TaxInvoice) ValidateItems() error {
	if len(t.Items) == 0 {
		return ErrTaxInvoiceNoItems
	}

	var supplySum, taxSum int64
	for i := range t.Items {
		item := &t.Items[i]
		if item.Amount < 0 || item.TaxAmount < 0 {
			return ErrTaxInvoiceNegativeAmount
		}
		if item.Amount == 0 && item.TaxAmount == 0 {
			return ErrTaxInvoiceItemInvalid
		}
		supplySum += item.Amount
		taxSum += item.TaxAmount
	}

	if supplySum != t.SupplyAmount || taxSum != t.TaxAmount {
		return fmt.Errorf("%w: 품목 공급가액 %d / 세액 %d, 헤더 공급가액 %d / 세액 %d",
			ErrTaxInvoiceItemSumMismatch, supplySum, taxSum, t.SupplyAmount, t.TaxAmount)
	}

	return nil
}

// ValidateForIssue runs every check that must pass before an invoice may be
// issued or transmitted to the NTS.
func (t *TaxInvoice) ValidateForIssue() error {
	if err := t.Validate(); err != nil {
		return err
	}
	return t.ValidateItems()
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
