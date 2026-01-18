package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// CreateVoucherRequest represents the request to create a voucher
type CreateVoucherRequest struct {
	VoucherDate   string                      `json:"voucher_date" binding:"required"`
	VoucherType   string                      `json:"voucher_type" binding:"required,oneof=general sales purchase payment receipt adjustment closing"`
	Description   string                      `json:"description,omitempty" binding:"max=500"`
	ReferenceType string                      `json:"reference_type,omitempty" binding:"max=50"`
	ReferenceID   string                      `json:"reference_id,omitempty" binding:"omitempty,uuid"`
	Entries       []CreateVoucherEntryRequest `json:"entries" binding:"required,min=1,dive"`
}

// CreateVoucherEntryRequest represents a single entry in the voucher
type CreateVoucherEntryRequest struct {
	AccountID    string  `json:"account_id" binding:"required,uuid"`
	DebitAmount  float64 `json:"debit_amount" binding:"min=0"`
	CreditAmount float64 `json:"credit_amount" binding:"min=0"`
	Description  string  `json:"description,omitempty" binding:"max=200"`
	PartnerID    string  `json:"partner_id,omitempty" binding:"omitempty,uuid"`
	DepartmentID string  `json:"department_id,omitempty" binding:"omitempty,uuid"`
	ProjectID    string  `json:"project_id,omitempty" binding:"omitempty,uuid"`
	CostCenterID string  `json:"cost_center_id,omitempty" binding:"omitempty,uuid"`
}

// ToVoucher converts CreateVoucherRequest to domain.Voucher
func (r *CreateVoucherRequest) ToVoucher(companyID, userID uuid.UUID) (*domain.Voucher, error) {
	voucherDate, err := time.Parse("2006-01-02", r.VoucherDate)
	if err != nil {
		return nil, domain.ErrInvalidVoucherDate
	}

	voucher := &domain.Voucher{
		TenantModel: domain.TenantModel{
			CompanyID: companyID,
		},
		VoucherDate:   voucherDate,
		VoucherType:   domain.VoucherType(r.VoucherType),
		Description:   r.Description,
		ReferenceType: r.ReferenceType,
		CreatedBy:     &userID,
	}

	if r.ReferenceID != "" {
		refID, err := uuid.Parse(r.ReferenceID)
		if err != nil {
			return nil, err
		}
		voucher.ReferenceID = &refID
	}

	// Convert entries
	for _, entryReq := range r.Entries {
		entry, err := entryReq.ToEntry(companyID)
		if err != nil {
			return nil, err
		}
		voucher.Entries = append(voucher.Entries, *entry)
	}

	return voucher, nil
}

// ToEntry converts CreateVoucherEntryRequest to domain.VoucherEntry
func (r *CreateVoucherEntryRequest) ToEntry(companyID uuid.UUID) (*domain.VoucherEntry, error) {
	accountID, err := uuid.Parse(r.AccountID)
	if err != nil {
		return nil, err
	}

	entry := &domain.VoucherEntry{
		CompanyID:    companyID,
		AccountID:    accountID,
		DebitAmount:  r.DebitAmount,
		CreditAmount: r.CreditAmount,
		Description:  r.Description,
	}

	if r.PartnerID != "" {
		partnerID, err := uuid.Parse(r.PartnerID)
		if err != nil {
			return nil, err
		}
		entry.PartnerID = &partnerID
	}

	if r.DepartmentID != "" {
		deptID, err := uuid.Parse(r.DepartmentID)
		if err != nil {
			return nil, err
		}
		entry.DepartmentID = &deptID
	}

	if r.ProjectID != "" {
		projectID, err := uuid.Parse(r.ProjectID)
		if err != nil {
			return nil, err
		}
		entry.ProjectID = &projectID
	}

	if r.CostCenterID != "" {
		ccID, err := uuid.Parse(r.CostCenterID)
		if err != nil {
			return nil, err
		}
		entry.CostCenterID = &ccID
	}

	return entry, nil
}

// UpdateVoucherRequest represents the request to update a voucher
type UpdateVoucherRequest struct {
	VoucherDate   string                      `json:"voucher_date" binding:"required"`
	Description   string                      `json:"description,omitempty" binding:"max=500"`
	ReferenceType string                      `json:"reference_type,omitempty" binding:"max=50"`
	ReferenceID   string                      `json:"reference_id,omitempty" binding:"omitempty,uuid"`
	Entries       []CreateVoucherEntryRequest `json:"entries" binding:"required,min=1,dive"`
}

// VoucherResponse represents the response for a voucher
type VoucherResponse struct {
	ID              string                 `json:"id"`
	VoucherNo       string                 `json:"voucher_no"`
	VoucherDate     string                 `json:"voucher_date"`
	VoucherType     string                 `json:"voucher_type"`
	VoucherTypeLabel string                `json:"voucher_type_label"`
	Status          string                 `json:"status"`
	StatusLabel     string                 `json:"status_label"`
	TotalDebit      float64                `json:"total_debit"`
	TotalCredit     float64                `json:"total_credit"`
	Description     string                 `json:"description,omitempty"`
	ReferenceType   string                 `json:"reference_type,omitempty"`
	ReferenceID     string                 `json:"reference_id,omitempty"`
	AttachmentCount int                    `json:"attachment_count"`
	IsReversal      bool                   `json:"is_reversal"`
	ReversalOfID    string                 `json:"reversal_of_id,omitempty"`
	ReversedByID    string                 `json:"reversed_by_id,omitempty"`
	SubmittedAt     string                 `json:"submitted_at,omitempty"`
	ApprovedAt      string                 `json:"approved_at,omitempty"`
	PostedAt        string                 `json:"posted_at,omitempty"`
	Entries         []VoucherEntryResponse `json:"entries,omitempty"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
}

// VoucherEntryResponse represents the response for a voucher entry
type VoucherEntryResponse struct {
	ID           string           `json:"id"`
	LineNo       int              `json:"line_no"`
	AccountID    string           `json:"account_id"`
	AccountCode  string           `json:"account_code,omitempty"`
	AccountName  string           `json:"account_name,omitempty"`
	DebitAmount  float64          `json:"debit_amount"`
	CreditAmount float64          `json:"credit_amount"`
	Description  string           `json:"description,omitempty"`
	PartnerID    string           `json:"partner_id,omitempty"`
	PartnerName  string           `json:"partner_name,omitempty"`
	DepartmentID string           `json:"department_id,omitempty"`
	DepartmentName string         `json:"department_name,omitempty"`
	ProjectID    string           `json:"project_id,omitempty"`
	CostCenterID string           `json:"cost_center_id,omitempty"`
}

// FromVoucher converts domain.Voucher to VoucherResponse
func FromVoucher(voucher *domain.Voucher) VoucherResponse {
	resp := VoucherResponse{
		ID:              voucher.ID.String(),
		VoucherNo:       voucher.VoucherNo,
		VoucherDate:     voucher.VoucherDate.Format("2006-01-02"),
		VoucherType:     string(voucher.VoucherType),
		VoucherTypeLabel: voucher.GetTypeLabel(),
		Status:          string(voucher.Status),
		StatusLabel:     voucher.GetStatusLabel(),
		TotalDebit:      voucher.TotalDebit,
		TotalCredit:     voucher.TotalCredit,
		Description:     voucher.Description,
		ReferenceType:   voucher.ReferenceType,
		AttachmentCount: voucher.AttachmentCount,
		IsReversal:      voucher.IsReversal,
		CreatedAt:       voucher.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       voucher.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if voucher.ReferenceID != nil {
		resp.ReferenceID = voucher.ReferenceID.String()
	}
	if voucher.ReversalOfID != nil {
		resp.ReversalOfID = voucher.ReversalOfID.String()
	}
	if voucher.ReversedByID != nil {
		resp.ReversedByID = voucher.ReversedByID.String()
	}
	if voucher.SubmittedAt != nil {
		resp.SubmittedAt = voucher.SubmittedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if voucher.ApprovedAt != nil {
		resp.ApprovedAt = voucher.ApprovedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if voucher.PostedAt != nil {
		resp.PostedAt = voucher.PostedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	// Convert entries
	for _, entry := range voucher.Entries {
		resp.Entries = append(resp.Entries, FromVoucherEntry(&entry))
	}

	return resp
}

// FromVoucherEntry converts domain.VoucherEntry to VoucherEntryResponse
func FromVoucherEntry(entry *domain.VoucherEntry) VoucherEntryResponse {
	resp := VoucherEntryResponse{
		ID:           entry.ID.String(),
		LineNo:       entry.LineNo,
		AccountID:    entry.AccountID.String(),
		DebitAmount:  entry.DebitAmount,
		CreditAmount: entry.CreditAmount,
		Description:  entry.Description,
	}

	if entry.Account != nil {
		resp.AccountCode = entry.Account.Code
		resp.AccountName = entry.Account.Name
	}
	if entry.PartnerID != nil {
		resp.PartnerID = entry.PartnerID.String()
		if entry.Partner != nil {
			resp.PartnerName = entry.Partner.Name
		}
	}
	if entry.DepartmentID != nil {
		resp.DepartmentID = entry.DepartmentID.String()
		if entry.Department != nil {
			resp.DepartmentName = entry.Department.Name
		}
	}
	if entry.ProjectID != nil {
		resp.ProjectID = entry.ProjectID.String()
	}
	if entry.CostCenterID != nil {
		resp.CostCenterID = entry.CostCenterID.String()
	}

	return resp
}

// FromVouchers converts a slice of domain.Voucher to []VoucherResponse
func FromVouchers(vouchers []domain.Voucher) []VoucherResponse {
	responses := make([]VoucherResponse, len(vouchers))
	for i, voucher := range vouchers {
		responses[i] = FromVoucher(&voucher)
	}
	return responses
}

// VoucherListRequest represents query parameters for listing vouchers
type VoucherListRequest struct {
	VoucherType  string `form:"voucher_type" binding:"omitempty,oneof=general sales purchase payment receipt adjustment closing"`
	Status       string `form:"status" binding:"omitempty,oneof=draft pending approved posted rejected cancelled"`
	DateFrom     string `form:"date_from" binding:"omitempty"`
	DateTo       string `form:"date_to" binding:"omitempty"`
	AccountID    string `form:"account_id" binding:"omitempty,uuid"`
	PartnerID    string `form:"partner_id" binding:"omitempty,uuid"`
	DepartmentID string `form:"department_id" binding:"omitempty,uuid"`
	Search       string `form:"search" binding:"max=100"`
	IncludeEntries bool `form:"include_entries"`
	Page         int    `form:"page" binding:"omitempty,min=1"`
	PageSize     int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	SortBy       string `form:"sort_by"`
	SortDesc     bool   `form:"sort_desc"`
}

// WorkflowActionRequest represents a workflow action request
type WorkflowActionRequest struct {
	Reason string `json:"reason,omitempty" binding:"max=500"`
}

// ReverseVoucherRequest represents the request to reverse a voucher
type ReverseVoucherRequest struct {
	ReversalDate string `json:"reversal_date" binding:"required"`
	Description  string `json:"description,omitempty" binding:"max=500"`
}
