package dto

import (
	"github.com/saintgo7/saas-kerp/internal/domain"
)

// PartnerResponse represents a partner in API responses
type PartnerResponse struct {
	ID               string  `json:"id"`
	Code             string  `json:"code"`
	Name             string  `json:"name"`
	NameEn           string  `json:"name_en,omitempty"`
	BusinessNumber   string  `json:"business_number,omitempty"`
	PartnerType      string  `json:"partner_type"`
	Representative   string  `json:"representative,omitempty"`
	Phone            string  `json:"phone,omitempty"`
	Fax              string  `json:"fax,omitempty"`
	Email            string  `json:"email,omitempty"`
	Website          string  `json:"website,omitempty"`
	ZipCode          string  `json:"zip_code,omitempty"`
	Address          string  `json:"address,omitempty"`
	AddressDetail    string  `json:"address_detail,omitempty"`
	PaymentTermDays  int     `json:"payment_term_days"`
	CreditLimit      float64 `json:"credit_limit"`
	ARAccountID      string  `json:"ar_account_id,omitempty"`
	APAccountID      string  `json:"ap_account_id,omitempty"`
	IsActive         bool    `json:"is_active"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// FromPartner converts domain.Partner to PartnerResponse
func FromPartner(partner *domain.Partner) PartnerResponse {
	resp := PartnerResponse{
		ID:              partner.ID.String(),
		Code:            partner.Code,
		Name:            partner.Name,
		NameEn:          partner.NameEn,
		BusinessNumber:  partner.BusinessNumber,
		PartnerType:     partner.PartnerType,
		Representative:  partner.Representative,
		Phone:           partner.Phone,
		Fax:             partner.Fax,
		Email:           partner.Email,
		Website:         partner.Website,
		ZipCode:         partner.ZipCode,
		Address:         partner.Address,
		AddressDetail:   partner.AddressDetail,
		PaymentTermDays: partner.PaymentTermDays,
		CreditLimit:     partner.CreditLimit,
		IsActive:        partner.IsActive,
		CreatedAt:       partner.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       partner.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if partner.ARAccountID != nil {
		resp.ARAccountID = partner.ARAccountID.String()
	}
	if partner.APAccountID != nil {
		resp.APAccountID = partner.APAccountID.String()
	}

	return resp
}

// FromPartners converts []domain.Partner to []PartnerResponse
func FromPartners(partners []domain.Partner) []PartnerResponse {
	responses := make([]PartnerResponse, len(partners))
	for i := range partners {
		responses[i] = FromPartner(&partners[i])
	}
	return responses
}

// CreatePartnerRequest represents the request to create a partner
type CreatePartnerRequest struct {
	Code            string  `json:"code" binding:"required,max=20"`
	Name            string  `json:"name" binding:"required,max=100"`
	NameEn          string  `json:"name_en,omitempty" binding:"max=100"`
	BusinessNumber  string  `json:"business_number,omitempty" binding:"max=12"`
	PartnerType     string  `json:"partner_type" binding:"required,oneof=customer vendor both"`
	Representative  string  `json:"representative,omitempty" binding:"max=50"`
	Phone           string  `json:"phone,omitempty" binding:"max=20"`
	Fax             string  `json:"fax,omitempty" binding:"max=20"`
	Email           string  `json:"email,omitempty" binding:"max=100,omitempty,email"`
	Website         string  `json:"website,omitempty" binding:"max=200"`
	ZipCode         string  `json:"zip_code,omitempty" binding:"max=10"`
	Address         string  `json:"address,omitempty" binding:"max=200"`
	AddressDetail   string  `json:"address_detail,omitempty" binding:"max=100"`
	PaymentTermDays int     `json:"payment_term_days,omitempty"`
	CreditLimit     float64 `json:"credit_limit,omitempty"`
	ARAccountID     string  `json:"ar_account_id,omitempty" binding:"omitempty,uuid"`
	APAccountID     string  `json:"ap_account_id,omitempty" binding:"omitempty,uuid"`
	IsActive        *bool   `json:"is_active,omitempty"`
}

// UpdatePartnerRequest represents the request to update a partner
type UpdatePartnerRequest struct {
	Code            string  `json:"code" binding:"required,max=20"`
	Name            string  `json:"name" binding:"required,max=100"`
	NameEn          string  `json:"name_en,omitempty" binding:"max=100"`
	BusinessNumber  string  `json:"business_number,omitempty" binding:"max=12"`
	PartnerType     string  `json:"partner_type" binding:"required,oneof=customer vendor both"`
	Representative  string  `json:"representative,omitempty" binding:"max=50"`
	Phone           string  `json:"phone,omitempty" binding:"max=20"`
	Fax             string  `json:"fax,omitempty" binding:"max=20"`
	Email           string  `json:"email,omitempty" binding:"max=100"`
	Website         string  `json:"website,omitempty" binding:"max=200"`
	ZipCode         string  `json:"zip_code,omitempty" binding:"max=10"`
	Address         string  `json:"address,omitempty" binding:"max=200"`
	AddressDetail   string  `json:"address_detail,omitempty" binding:"max=100"`
	PaymentTermDays int     `json:"payment_term_days,omitempty"`
	CreditLimit     float64 `json:"credit_limit,omitempty"`
	ARAccountID     string  `json:"ar_account_id,omitempty" binding:"omitempty,uuid"`
	APAccountID     string  `json:"ap_account_id,omitempty" binding:"omitempty,uuid"`
	IsActive        *bool   `json:"is_active,omitempty"`
}

// BulkStatusRequest represents the request to change status for multiple partners
type BulkStatusRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// PartnerStatsResponse represents partner statistics
type PartnerStatsResponse struct {
	TotalCount    int64 `json:"total_count"`
	CustomerCount int64 `json:"customer_count"`
	VendorCount   int64 `json:"vendor_count"`
	ActiveCount   int64 `json:"active_count"`
	InactiveCount int64 `json:"inactive_count"`
}
