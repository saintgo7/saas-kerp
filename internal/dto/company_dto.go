package dto

import (
	"github.com/saintgo7/saas-kerp/internal/domain"
)

// CompanySettingsResponse represents company settings in API responses
type CompanySettingsResponse struct {
	FiscalYearStart     int     `json:"fiscal_year_start"`
	DefaultCurrency     string  `json:"default_currency"`
	DecimalPlaces       int     `json:"decimal_places"`
	TaxRate             float64 `json:"tax_rate"`
	VoucherAutoNumber   bool    `json:"voucher_auto_number"`
	VoucherNumberFormat string  `json:"voucher_number_format"`
	InvoicePrefix       string  `json:"invoice_prefix"`
	Timezone            string  `json:"timezone"`
	DateFormat          string  `json:"date_format"`
	Language            string  `json:"language"`
}

// CompanyResponse represents a company in API responses
type CompanyResponse struct {
	ID             string                  `json:"id"`
	Code           string                  `json:"code"`
	Name           string                  `json:"name"`
	NameEn         string                  `json:"name_en,omitempty"`
	BusinessNumber string                  `json:"business_number,omitempty"`
	Representative string                  `json:"representative,omitempty"`
	Phone          string                  `json:"phone,omitempty"`
	Fax            string                  `json:"fax,omitempty"`
	Email          string                  `json:"email,omitempty"`
	Website        string                  `json:"website,omitempty"`
	ZipCode        string                  `json:"zip_code,omitempty"`
	Address        string                  `json:"address,omitempty"`
	AddressDetail  string                  `json:"address_detail,omitempty"`
	Status         string                  `json:"status"`
	Settings       CompanySettingsResponse `json:"settings"`
	TrialEndsAt    *string                 `json:"trial_ends_at,omitempty"`
	Logo           string                  `json:"logo,omitempty"`
	CreatedAt      string                  `json:"created_at"`
	UpdatedAt      string                  `json:"updated_at"`
}

// FromCompany converts domain.Company to CompanyResponse
func FromCompany(company *domain.Company) CompanyResponse {
	resp := CompanyResponse{
		ID:             company.ID.String(),
		Code:           company.Code,
		Name:           company.Name,
		NameEn:         company.NameEn,
		BusinessNumber: company.BusinessNumber,
		Representative: company.Representative,
		Phone:          company.Phone,
		Fax:            company.Fax,
		Email:          company.Email,
		Website:        company.Website,
		ZipCode:        company.ZipCode,
		Address:        company.Address,
		AddressDetail:  company.AddressDetail,
		Status:         string(company.Status),
		Settings: CompanySettingsResponse{
			FiscalYearStart:     company.Settings.FiscalYearStart,
			DefaultCurrency:     company.Settings.DefaultCurrency,
			DecimalPlaces:       company.Settings.DecimalPlaces,
			TaxRate:             company.Settings.TaxRate,
			VoucherAutoNumber:   company.Settings.VoucherAutoNumber,
			VoucherNumberFormat: company.Settings.VoucherNumberFormat,
			InvoicePrefix:       company.Settings.InvoicePrefix,
			Timezone:            company.Settings.Timezone,
			DateFormat:          company.Settings.DateFormat,
			Language:            company.Settings.Language,
		},
		Logo:      company.Logo,
		CreatedAt: company.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: company.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if company.TrialEndsAt != nil {
		formatted := company.TrialEndsAt.Format("2006-01-02T15:04:05Z07:00")
		resp.TrialEndsAt = &formatted
	}

	return resp
}

// UpdateCompanyRequest represents the request to update company info
type UpdateCompanyRequest struct {
	Name           string `json:"name" binding:"required,max=200"`
	NameEn         string `json:"name_en,omitempty" binding:"max=200"`
	BusinessNumber string `json:"business_number,omitempty" binding:"max=12"`
	Representative string `json:"representative,omitempty" binding:"max=100"`
	Phone          string `json:"phone,omitempty" binding:"max=20"`
	Fax            string `json:"fax,omitempty" binding:"max=20"`
	Email          string `json:"email,omitempty" binding:"max=255,omitempty,email"`
	Website        string `json:"website,omitempty" binding:"max=200"`
	ZipCode        string `json:"zip_code,omitempty" binding:"max=10"`
	Address        string `json:"address,omitempty" binding:"max=300"`
	AddressDetail  string `json:"address_detail,omitempty" binding:"max=200"`
	Logo           string `json:"logo,omitempty" binding:"max=500"`
}

// ApplyTo applies the update to an existing company
func (r *UpdateCompanyRequest) ApplyTo(company *domain.Company) {
	company.Name = r.Name
	company.NameEn = r.NameEn
	company.BusinessNumber = r.BusinessNumber
	company.Representative = r.Representative
	company.Phone = r.Phone
	company.Fax = r.Fax
	company.Email = r.Email
	company.Website = r.Website
	company.ZipCode = r.ZipCode
	company.Address = r.Address
	company.AddressDetail = r.AddressDetail
	company.Logo = r.Logo
}

// UpdateCompanySettingsRequest represents the request to update company settings
type UpdateCompanySettingsRequest struct {
	FiscalYearStart     *int     `json:"fiscal_year_start,omitempty" binding:"omitempty,min=1,max=12"`
	DefaultCurrency     string   `json:"default_currency,omitempty" binding:"max=10"`
	DecimalPlaces       *int     `json:"decimal_places,omitempty" binding:"omitempty,min=0,max=4"`
	TaxRate             *float64 `json:"tax_rate,omitempty" binding:"omitempty,min=0,max=100"`
	VoucherAutoNumber   *bool    `json:"voucher_auto_number,omitempty"`
	VoucherNumberFormat string   `json:"voucher_number_format,omitempty" binding:"max=50"`
	InvoicePrefix       string   `json:"invoice_prefix,omitempty" binding:"max=20"`
	Timezone            string   `json:"timezone,omitempty" binding:"max=50"`
	DateFormat          string   `json:"date_format,omitempty" binding:"max=20"`
	Language            string   `json:"language,omitempty" binding:"max=10"`
}

// ApplyTo applies the settings update to an existing company
func (r *UpdateCompanySettingsRequest) ApplyTo(company *domain.Company) {
	if r.FiscalYearStart != nil {
		company.Settings.FiscalYearStart = *r.FiscalYearStart
	}
	if r.DefaultCurrency != "" {
		company.Settings.DefaultCurrency = r.DefaultCurrency
	}
	if r.DecimalPlaces != nil {
		company.Settings.DecimalPlaces = *r.DecimalPlaces
	}
	if r.TaxRate != nil {
		company.Settings.TaxRate = *r.TaxRate
	}
	if r.VoucherAutoNumber != nil {
		company.Settings.VoucherAutoNumber = *r.VoucherAutoNumber
	}
	if r.VoucherNumberFormat != "" {
		company.Settings.VoucherNumberFormat = r.VoucherNumberFormat
	}
	if r.InvoicePrefix != "" {
		company.Settings.InvoicePrefix = r.InvoicePrefix
	}
	if r.Timezone != "" {
		company.Settings.Timezone = r.Timezone
	}
	if r.DateFormat != "" {
		company.Settings.DateFormat = r.DateFormat
	}
	if r.Language != "" {
		company.Settings.Language = r.Language
	}
}
