package dto

import (
	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// CreateAccountRequest represents the request to create an account
type CreateAccountRequest struct {
	Code               string `json:"code" binding:"required,max=10"`
	Name               string `json:"name" binding:"required,max=100"`
	NameEn             string `json:"name_en,omitempty" binding:"max=100"`
	ParentID           string `json:"parent_id,omitempty" binding:"omitempty,uuid"`
	AccountType        string `json:"account_type" binding:"required,oneof=asset liability equity revenue expense"`
	AccountNature      string `json:"account_nature,omitempty" binding:"omitempty,oneof=debit credit"`
	AccountCategory    string `json:"account_category,omitempty" binding:"max=50"`
	IsActive           *bool  `json:"is_active,omitempty"`
	IsControlAccount   *bool  `json:"is_control_account,omitempty"`
	AllowDirectPosting *bool  `json:"allow_direct_posting,omitempty"`
	SortOrder          int    `json:"sort_order,omitempty"`
}

// ToAccount converts CreateAccountRequest to domain.Account
func (r *CreateAccountRequest) ToAccount(companyID uuid.UUID) (*domain.Account, error) {
	account := &domain.Account{
		TenantModel: domain.TenantModel{
			CompanyID: companyID,
		},
		Code:            r.Code,
		Name:            r.Name,
		NameEn:          r.NameEn,
		AccountType:     domain.AccountType(r.AccountType),
		AccountCategory: r.AccountCategory,
		SortOrder:       r.SortOrder,
	}

	if r.ParentID != "" {
		parentID, err := uuid.Parse(r.ParentID)
		if err != nil {
			return nil, err
		}
		account.ParentID = &parentID
	}

	if r.AccountNature != "" {
		account.AccountNature = domain.AccountNature(r.AccountNature)
	}

	if r.IsActive != nil {
		account.IsActive = *r.IsActive
	} else {
		account.IsActive = true
	}

	if r.IsControlAccount != nil {
		account.IsControlAccount = *r.IsControlAccount
	}

	if r.AllowDirectPosting != nil {
		account.AllowDirectPosting = *r.AllowDirectPosting
	} else {
		account.AllowDirectPosting = true
	}

	return account, nil
}

// UpdateAccountRequest represents the request to update an account
type UpdateAccountRequest struct {
	Code               string `json:"code" binding:"required,max=10"`
	Name               string `json:"name" binding:"required,max=100"`
	NameEn             string `json:"name_en,omitempty" binding:"max=100"`
	ParentID           string `json:"parent_id,omitempty" binding:"omitempty,uuid"`
	AccountType        string `json:"account_type" binding:"required,oneof=asset liability equity revenue expense"`
	AccountNature      string `json:"account_nature" binding:"required,oneof=debit credit"`
	AccountCategory    string `json:"account_category,omitempty" binding:"max=50"`
	IsActive           *bool  `json:"is_active"`
	IsControlAccount   *bool  `json:"is_control_account"`
	AllowDirectPosting *bool  `json:"allow_direct_posting"`
	SortOrder          int    `json:"sort_order,omitempty"`
}

// ApplyTo applies the update request to an existing account
func (r *UpdateAccountRequest) ApplyTo(account *domain.Account) error {
	account.Code = r.Code
	account.Name = r.Name
	account.NameEn = r.NameEn
	account.AccountType = domain.AccountType(r.AccountType)
	account.AccountNature = domain.AccountNature(r.AccountNature)
	account.AccountCategory = r.AccountCategory
	account.SortOrder = r.SortOrder

	if r.ParentID != "" {
		parentID, err := uuid.Parse(r.ParentID)
		if err != nil {
			return err
		}
		account.ParentID = &parentID
	} else {
		account.ParentID = nil
	}

	if r.IsActive != nil {
		account.IsActive = *r.IsActive
	}
	if r.IsControlAccount != nil {
		account.IsControlAccount = *r.IsControlAccount
	}
	if r.AllowDirectPosting != nil {
		account.AllowDirectPosting = *r.AllowDirectPosting
	}

	return nil
}

// AccountResponse represents the response for an account
type AccountResponse struct {
	ID                 string             `json:"id"`
	Code               string             `json:"code"`
	Name               string             `json:"name"`
	NameEn             string             `json:"name_en,omitempty"`
	ParentID           string             `json:"parent_id,omitempty"`
	Level              int                `json:"level"`
	Path               string             `json:"path,omitempty"`
	AccountType        string             `json:"account_type"`
	AccountTypeLabel   string             `json:"account_type_label"`
	AccountNature      string             `json:"account_nature"`
	AccountNatureLabel string             `json:"account_nature_label"`
	AccountCategory    string             `json:"account_category,omitempty"`
	IsActive           bool               `json:"is_active"`
	IsControlAccount   bool               `json:"is_control_account"`
	AllowDirectPosting bool               `json:"allow_direct_posting"`
	SortOrder          int                `json:"sort_order"`
	Children           []AccountResponse  `json:"children,omitempty"`
	CreatedAt          string             `json:"created_at"`
	UpdatedAt          string             `json:"updated_at"`
}

// FromAccount converts domain.Account to AccountResponse
func FromAccount(account *domain.Account) AccountResponse {
	resp := AccountResponse{
		ID:                 account.ID.String(),
		Code:               account.Code,
		Name:               account.Name,
		NameEn:             account.NameEn,
		Level:              account.Level,
		Path:               account.Path,
		AccountType:        string(account.AccountType),
		AccountTypeLabel:   account.GetTypeLabel(),
		AccountNature:      string(account.AccountNature),
		AccountNatureLabel: account.GetNatureLabel(),
		AccountCategory:    account.AccountCategory,
		IsActive:           account.IsActive,
		IsControlAccount:   account.IsControlAccount,
		AllowDirectPosting: account.AllowDirectPosting,
		SortOrder:          account.SortOrder,
		CreatedAt:          account.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          account.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if account.ParentID != nil {
		resp.ParentID = account.ParentID.String()
	}

	// Convert children recursively
	if len(account.Children) > 0 {
		resp.Children = make([]AccountResponse, len(account.Children))
		for i, child := range account.Children {
			resp.Children[i] = FromAccount(&child)
		}
	}

	return resp
}

// FromAccounts converts a slice of domain.Account to []AccountResponse
func FromAccounts(accounts []domain.Account) []AccountResponse {
	responses := make([]AccountResponse, len(accounts))
	for i, account := range accounts {
		responses[i] = FromAccount(&account)
	}
	return responses
}

// AccountListRequest represents query parameters for listing accounts
type AccountListRequest struct {
	ParentID    string `form:"parent_id" binding:"omitempty,uuid"`
	AccountType string `form:"account_type" binding:"omitempty,oneof=asset liability equity revenue expense"`
	IsActive    *bool  `form:"is_active"`
	Search      string `form:"search" binding:"max=100"`
	Page        int    `form:"page" binding:"min=1"`
	PageSize    int    `form:"page_size" binding:"min=1,max=100"`
	SortBy      string `form:"sort_by" binding:"omitempty,oneof=code name sort_order created_at"`
	SortDesc    bool   `form:"sort_desc"`
}

// AccountListResponse represents the response for listing accounts
type AccountListResponse struct {
	Data       []AccountResponse `json:"data"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

// MoveAccountRequest represents the request to move an account to a new parent
type MoveAccountRequest struct {
	ParentID string `json:"parent_id" binding:"omitempty,uuid"`
}

// UpdateSortOrderRequest represents the request to update sort orders
type UpdateSortOrderRequest struct {
	Orders []SortOrderItem `json:"orders" binding:"required,dive"`
}

// SortOrderItem represents a single sort order update
type SortOrderItem struct {
	ID        string `json:"id" binding:"required,uuid"`
	SortOrder int    `json:"sort_order" binding:"min=0"`
}
