package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Company errors
var (
	ErrCompanyNotFound   = errors.New("company not found")
	ErrCompanyCodeExists = errors.New("company code already exists")
	ErrCompanyNameEmpty  = errors.New("company name is required")
)

// CompanyStatus represents the status of a company
type CompanyStatus string

const (
	CompanyStatusActive    CompanyStatus = "active"
	CompanyStatusSuspended CompanyStatus = "suspended"
	CompanyStatusTrial     CompanyStatus = "trial"
)

// IsValid checks if the company status is valid
func (s CompanyStatus) IsValid() bool {
	switch s {
	case CompanyStatusActive, CompanyStatusSuspended, CompanyStatusTrial:
		return true
	}
	return false
}

// CompanySettings represents company-specific settings
type CompanySettings struct {
	FiscalYearStart    int    `json:"fiscal_year_start"`      // Month (1-12)
	DefaultCurrency    string `json:"default_currency"`       // KRW, USD, etc.
	DecimalPlaces      int    `json:"decimal_places"`         // Number of decimal places for amounts
	TaxRate            float64 `json:"tax_rate"`              // Default VAT rate (e.g., 10.0)
	VoucherAutoNumber  bool   `json:"voucher_auto_number"`    // Auto-generate voucher numbers
	VoucherNumberFormat string `json:"voucher_number_format"` // Format: YYYYMM-NNNN
	InvoicePrefix      string `json:"invoice_prefix"`         // Prefix for invoice numbers
	Timezone           string `json:"timezone"`               // Timezone: Asia/Seoul
	DateFormat         string `json:"date_format"`            // Date format: YYYY-MM-DD
	Language           string `json:"language"`               // Default language: ko, en
}

// DefaultCompanySettings returns default settings for a new company
func DefaultCompanySettings() CompanySettings {
	return CompanySettings{
		FiscalYearStart:    1,
		DefaultCurrency:    "KRW",
		DecimalPlaces:      0,
		TaxRate:            10.0,
		VoucherAutoNumber:  true,
		VoucherNumberFormat: "YYYYMM-NNNN",
		InvoicePrefix:      "INV",
		Timezone:           "Asia/Seoul",
		DateFormat:         "YYYY-MM-DD",
		Language:           "ko",
	}
}

// Company represents a company (tenant) in the system
type Company struct {
	BaseModel
	Code           string          `gorm:"type:varchar(50);not null;uniqueIndex" json:"code"`
	Name           string          `gorm:"type:varchar(200);not null" json:"name"`
	NameEn         string          `gorm:"type:varchar(200)" json:"name_en,omitempty"`
	BusinessNumber string          `gorm:"type:varchar(12)" json:"business_number,omitempty"`
	Representative string          `gorm:"type:varchar(100)" json:"representative,omitempty"`
	Phone          string          `gorm:"type:varchar(20)" json:"phone,omitempty"`
	Fax            string          `gorm:"type:varchar(20)" json:"fax,omitempty"`
	Email          string          `gorm:"type:varchar(255)" json:"email,omitempty"`
	Website        string          `gorm:"type:varchar(200)" json:"website,omitempty"`
	ZipCode        string          `gorm:"type:varchar(10)" json:"zip_code,omitempty"`
	Address        string          `gorm:"type:varchar(300)" json:"address,omitempty"`
	AddressDetail  string          `gorm:"type:varchar(200)" json:"address_detail,omitempty"`
	Status         CompanyStatus   `gorm:"type:varchar(20);default:'active'" json:"status"`
	Settings       CompanySettings `gorm:"type:jsonb;serializer:json" json:"settings"`
	TrialEndsAt    *time.Time      `json:"trial_ends_at,omitempty"`
	Logo           string          `gorm:"type:varchar(500)" json:"logo,omitempty"`
}

// TableName returns the table name for Company
func (Company) TableName() string {
	return "kerp.companies"
}

// NewCompany creates a new company
func NewCompany(code, name string) (*Company, error) {
	if name == "" {
		return nil, ErrCompanyNameEmpty
	}
	if code == "" {
		code = uuid.New().String()[:8]
	}

	return &Company{
		Code:     code,
		Name:     name,
		Status:   CompanyStatusActive,
		Settings: DefaultCompanySettings(),
	}, nil
}

// IsActive returns true if the company is active
func (c *Company) IsActive() bool {
	return c.Status == CompanyStatusActive || c.Status == CompanyStatusTrial
}

// IsTrial returns true if the company is in trial period
func (c *Company) IsTrial() bool {
	return c.Status == CompanyStatusTrial
}

// IsTrialExpired returns true if the trial period has expired
func (c *Company) IsTrialExpired() bool {
	if !c.IsTrial() || c.TrialEndsAt == nil {
		return false
	}
	return time.Now().After(*c.TrialEndsAt)
}
