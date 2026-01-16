package domain

import (
	"errors"

	"github.com/google/uuid"
)

// AccountType represents the five major account classifications in K-IFRS
type AccountType string

const (
	AccountTypeAsset     AccountType = "asset"
	AccountTypeLiability AccountType = "liability"
	AccountTypeEquity    AccountType = "equity"
	AccountTypeRevenue   AccountType = "revenue"
	AccountTypeExpense   AccountType = "expense"
)

// IsValid checks if the account type is valid
func (t AccountType) IsValid() bool {
	switch t {
	case AccountTypeAsset, AccountTypeLiability, AccountTypeEquity, AccountTypeRevenue, AccountTypeExpense:
		return true
	}
	return false
}

// AccountNature represents the normal balance side of an account
type AccountNature string

const (
	AccountNatureDebit  AccountNature = "debit"
	AccountNatureCredit AccountNature = "credit"
)

// IsValid checks if the account nature is valid
func (n AccountNature) IsValid() bool {
	return n == AccountNatureDebit || n == AccountNatureCredit
}

// Account errors
var (
	ErrAccountNotFound       = errors.New("account not found")
	ErrAccountCodeExists     = errors.New("account code already exists")
	ErrAccountHasChildren    = errors.New("cannot delete account with children")
	ErrAccountHasEntries     = errors.New("cannot delete account with voucher entries")
	ErrInvalidAccountType    = errors.New("invalid account type")
	ErrInvalidAccountNature  = errors.New("invalid account nature")
	ErrAccountCodeRequired   = errors.New("account code is required")
	ErrAccountNameRequired   = errors.New("account name is required")
	ErrParentNotFound        = errors.New("parent account not found")
	ErrCircularReference     = errors.New("circular reference detected")
	ErrControlAccountPosting = errors.New("cannot post directly to control account")
)

// Account represents a chart of accounts entry following K-IFRS
type Account struct {
	TenantModel

	// Account identification
	Code   string `gorm:"type:varchar(10);not null" json:"code"`
	Name   string `gorm:"type:varchar(100);not null" json:"name"`
	NameEn string `gorm:"type:varchar(100)" json:"name_en,omitempty"`

	// Hierarchy
	ParentID *uuid.UUID `gorm:"type:uuid" json:"parent_id,omitempty"`
	Parent   *Account   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Account  `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Level    int        `gorm:"not null;default:1" json:"level"`
	Path     string     `gorm:"type:ltree" json:"path,omitempty"`

	// Classification (K-IFRS)
	AccountType   AccountType   `gorm:"type:varchar(20);not null" json:"account_type"`
	AccountNature AccountNature `gorm:"type:varchar(10);not null" json:"account_nature"`

	// Sub-classification
	AccountCategory string `gorm:"type:varchar(50)" json:"account_category,omitempty"`

	// Settings
	IsActive           bool `gorm:"default:true" json:"is_active"`
	IsControlAccount   bool `gorm:"default:false" json:"is_control_account"`
	AllowDirectPosting bool `gorm:"default:true" json:"allow_direct_posting"`

	// Display order
	SortOrder int `gorm:"default:0" json:"sort_order"`
}

// TableName specifies the table name for GORM
func (Account) TableName() string {
	return "accounts"
}

// Validate validates the account data
func (a *Account) Validate() error {
	if a.Code == "" {
		return ErrAccountCodeRequired
	}
	if a.Name == "" {
		return ErrAccountNameRequired
	}
	if !a.AccountType.IsValid() {
		return ErrInvalidAccountType
	}
	if !a.AccountNature.IsValid() {
		return ErrInvalidAccountNature
	}
	return nil
}

// SetDefaults sets default values based on account type
func (a *Account) SetDefaults() {
	// Set account nature based on account type if not specified
	if a.AccountNature == "" {
		switch a.AccountType {
		case AccountTypeAsset, AccountTypeExpense:
			a.AccountNature = AccountNatureDebit
		case AccountTypeLiability, AccountTypeEquity, AccountTypeRevenue:
			a.AccountNature = AccountNatureCredit
		}
	}

	// Set default level if not specified
	if a.Level == 0 {
		a.Level = 1
	}
}

// CanPost checks if direct posting is allowed on this account
func (a *Account) CanPost() bool {
	return a.IsActive && a.AllowDirectPosting && !a.IsControlAccount
}

// IsDebitNature returns true if the account has debit nature
func (a *Account) IsDebitNature() bool {
	return a.AccountNature == AccountNatureDebit
}

// IsCreditNature returns true if the account has credit nature
func (a *Account) IsCreditNature() bool {
	return a.AccountNature == AccountNatureCredit
}

// GetTypeLabel returns Korean label for account type
func (a *Account) GetTypeLabel() string {
	switch a.AccountType {
	case AccountTypeAsset:
		return "자산"
	case AccountTypeLiability:
		return "부채"
	case AccountTypeEquity:
		return "자본"
	case AccountTypeRevenue:
		return "수익"
	case AccountTypeExpense:
		return "비용"
	default:
		return ""
	}
}

// GetNatureLabel returns Korean label for account nature
func (a *Account) GetNatureLabel() string {
	switch a.AccountNature {
	case AccountNatureDebit:
		return "차변"
	case AccountNatureCredit:
		return "대변"
	default:
		return ""
	}
}
