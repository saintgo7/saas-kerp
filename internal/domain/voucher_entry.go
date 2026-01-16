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

// Department represents an organizational department
type Department struct {
	TenantModel

	// Basic info
	Code   string `gorm:"type:varchar(20);not null" json:"code"`
	Name   string `gorm:"type:varchar(100);not null" json:"name"`
	NameEn string `gorm:"type:varchar(100)" json:"name_en,omitempty"`

	// Hierarchy
	ParentID *uuid.UUID    `gorm:"type:uuid" json:"parent_id,omitempty"`
	Parent   *Department   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Department  `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Level    int           `gorm:"not null;default:1" json:"level"`
	Path     string        `gorm:"type:ltree" json:"path,omitempty"`

	// Manager
	ManagerID *uuid.UUID `gorm:"type:uuid" json:"manager_id,omitempty"`

	// Status
	IsActive bool `gorm:"default:true" json:"is_active"`
}

// TableName specifies the table name for GORM
func (Department) TableName() string {
	return "departments"
}

// Partner represents a business partner (customer/vendor)
type Partner struct {
	TenantModel

	// Basic info
	Code           string `gorm:"type:varchar(20);not null" json:"code"`
	Name           string `gorm:"type:varchar(100);not null" json:"name"`
	NameEn         string `gorm:"type:varchar(100)" json:"name_en,omitempty"`
	BusinessNumber string `gorm:"type:varchar(12)" json:"business_number,omitempty"`

	// Type
	PartnerType   string `gorm:"type:varchar(20);not null" json:"partner_type"` // customer, vendor, both
	Representative string `gorm:"type:varchar(50)" json:"representative,omitempty"`

	// Contact
	Phone   string `gorm:"type:varchar(20)" json:"phone,omitempty"`
	Fax     string `gorm:"type:varchar(20)" json:"fax,omitempty"`
	Email   string `gorm:"type:varchar(100)" json:"email,omitempty"`
	Website string `gorm:"type:varchar(200)" json:"website,omitempty"`

	// Address
	ZipCode       string `gorm:"type:varchar(10)" json:"zip_code,omitempty"`
	Address       string `gorm:"type:varchar(200)" json:"address,omitempty"`
	AddressDetail string `gorm:"type:varchar(100)" json:"address_detail,omitempty"`

	// Accounting
	PaymentTermDays int        `gorm:"default:30" json:"payment_term_days"`
	CreditLimit     float64    `gorm:"type:decimal(18,2);default:0" json:"credit_limit"`
	ARAccountID     *uuid.UUID `gorm:"type:uuid" json:"ar_account_id,omitempty"` // Accounts Receivable
	APAccountID     *uuid.UUID `gorm:"type:uuid" json:"ap_account_id,omitempty"` // Accounts Payable

	// Status
	IsActive bool `gorm:"default:true" json:"is_active"`
}

// TableName specifies the table name for GORM
func (Partner) TableName() string {
	return "partners"
}

// Project represents a project for cost tracking
type Project struct {
	TenantModel

	// Basic info
	Code        string `gorm:"type:varchar(20);not null" json:"code"`
	Name        string `gorm:"type:varchar(100);not null" json:"name"`
	Description string `gorm:"type:varchar(500)" json:"description,omitempty"`

	// Manager
	ManagerID *uuid.UUID `gorm:"type:uuid" json:"manager_id,omitempty"`

	// Budget
	Budget float64 `gorm:"type:decimal(18,2);default:0" json:"budget"`

	// Status
	Status string `gorm:"type:varchar(20);default:active" json:"status"` // active, completed, cancelled
}

// TableName specifies the table name for GORM
func (Project) TableName() string {
	return "projects"
}

// CostCenter represents a cost center for expense tracking
type CostCenter struct {
	TenantModel

	// Basic info
	Code        string `gorm:"type:varchar(20);not null" json:"code"`
	Name        string `gorm:"type:varchar(100);not null" json:"name"`
	Description string `gorm:"type:varchar(500)" json:"description,omitempty"`

	// Hierarchy
	ParentID *uuid.UUID    `gorm:"type:uuid" json:"parent_id,omitempty"`
	Parent   *CostCenter   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []CostCenter  `gorm:"foreignKey:ParentID" json:"children,omitempty"`

	// Status
	IsActive bool `gorm:"default:true" json:"is_active"`
}

// TableName specifies the table name for GORM
func (CostCenter) TableName() string {
	return "cost_centers"
}
