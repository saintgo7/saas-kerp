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

// Validate validates the entry data
func (e *VoucherEntry) Validate() error {
	// Check that exactly one of debit or credit is set
	if e.DebitAmount > 0 && e.CreditAmount > 0 {
		return ErrEntryInvalidAmount
	}
	if e.DebitAmount == 0 && e.CreditAmount == 0 {
		return ErrEntryZeroAmount
	}
	if e.DebitAmount < 0 || e.CreditAmount < 0 {
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
