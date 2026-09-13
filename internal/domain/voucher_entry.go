package domain

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

// VoucherEntry errors
var (
	ErrEntryNotFound       = errors.New("voucher entry not found")
	ErrEntryInvalidAmount  = errors.New("entry must have either debit or credit amount, not both")
	ErrEntryZeroAmount     = errors.New("entry amount must be greater than zero")
	ErrEntryAccountInvalid = errors.New("invalid account for entry")
)

// VoucherEntry represents a single debit/credit entry within a voucher
type VoucherEntry struct {
	BaseModel
	VoucherID uuid.UUID `gorm:"type:uuid;not null;index" json:"voucher_id"`
	CompanyID uuid.UUID `gorm:"type:uuid;not null;index" json:"company_id"`

	// Entry info
	LineNo    int       `gorm:"not null" json:"line_no"`
	AccountID uuid.UUID `gorm:"type:uuid;not null" json:"account_id"`

	// Amounts (one must be zero)
	DebitAmount  float64 `gorm:"type:decimal(18,2);not null;default:0" json:"debit_amount"`
	CreditAmount float64 `gorm:"type:decimal(18,2);not null;default:0" json:"credit_amount"`

	// Description
	Description string `gorm:"type:varchar(200)" json:"description,omitempty"`

	// Dimensions
	PartnerID    *uuid.UUID `gorm:"type:uuid" json:"partner_id,omitempty"`
	DepartmentID *uuid.UUID `gorm:"type:uuid" json:"department_id,omitempty"`
	ProjectID    *uuid.UUID `gorm:"type:uuid" json:"project_id,omitempty"`
	CostCenterID *uuid.UUID `gorm:"type:uuid" json:"cost_center_id,omitempty"`

	// Tags for analysis
	Tags json.RawMessage `gorm:"type:jsonb;default:'[]'" json:"tags,omitempty"`

	// Relations
	Account    *Account    `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	Partner    *Partner    `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	Department *Department `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
}

// TableName specifies the table name for GORM
func (VoucherEntry) TableName() string {
	return "voucher_entries"
}

// Normalize quantizes the entry amounts to the scale the database stores
// (DECIMAL(18,2)). It must be called before the entry is validated, summed or
// persisted: PostgreSQL rounds every row independently, so an unrounded Go
// value silently becomes a different stored value and the voucher header stops
// matching SUM(voucher_entries).
func (e *VoucherEntry) Normalize() {
	e.DebitAmount = RoundAmount(e.DebitAmount)
	e.CreditAmount = RoundAmount(e.CreditAmount)
}

// Validate validates the entry data.
//
// Amounts are judged at the stored scale. An amount such as 0.004 is non-zero
// in Go but is stored as 0.00, which violates the chk_entry_amount database
// constraint; rejecting it here turns a raw 500 into a validation error.
func (e *VoucherEntry) Validate() error {
	if e.DebitAmount < 0 || e.CreditAmount < 0 {
		return ErrEntryZeroAmount
	}

	debit := RoundAmount(e.DebitAmount)
	credit := RoundAmount(e.CreditAmount)

	// Check that exactly one of debit or credit is set
	if debit > 0 && credit > 0 {
		return ErrEntryInvalidAmount
	}
	if IsZeroAmount(debit) && IsZeroAmount(credit) {
		return ErrEntryZeroAmount
	}
	return nil
}

// IsDebit returns true if this is a debit entry
func (e *VoucherEntry) IsDebit() bool {
	return e.DebitAmount > 0
}

// IsCredit returns true if this is a credit entry
func (e *VoucherEntry) IsCredit() bool {
	return e.CreditAmount > 0
}

// GetAmount returns the non-zero amount
func (e *VoucherEntry) GetAmount() float64 {
	if e.DebitAmount > 0 {
		return e.DebitAmount
	}
	return e.CreditAmount
}

// SetDebit sets the entry as a debit entry
func (e *VoucherEntry) SetDebit(amount float64) {
	e.DebitAmount = amount
	e.CreditAmount = 0
}

// SetCredit sets the entry as a credit entry
func (e *VoucherEntry) SetCredit(amount float64) {
	e.DebitAmount = 0
	e.CreditAmount = amount
}
