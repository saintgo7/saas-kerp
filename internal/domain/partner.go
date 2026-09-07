package domain

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Partner errors
var (
	ErrPartnerNotFound   = errors.New("partner not found")
	ErrPartnerCodeExists = errors.New("partner code already exists")
)

// Partner represents a business partner (customer/vendor)
type Partner struct {
	TenantModel

	// DeletedAt enables GORM soft delete. The table carries deleted_at and
	// db/migrations/000020_index_hygiene makes the uniqueness partial on
	// `deleted_at IS NULL`, so a hard delete here would both break audit
	// foreign keys and waste that work. Tagged json:"-" - gorm.DeletedAt
	// marshals as {"Time":...,"Valid":...}.
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Basic info
	Code           string `gorm:"type:varchar(20);not null" json:"code"`
	Name           string `gorm:"type:varchar(100);not null" json:"name"`
	NameEn         string `gorm:"type:varchar(100)" json:"name_en,omitempty"`
	BusinessNumber string `gorm:"type:varchar(12)" json:"business_number,omitempty"`

	// Type
	PartnerType    string `gorm:"type:varchar(20);not null" json:"partner_type"` // customer, vendor, both
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
