package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// VoucherType represents the type of voucher
type VoucherType string

const (
	VoucherTypeGeneral    VoucherType = "general"
	VoucherTypeSales      VoucherType = "sales"
	VoucherTypePurchase   VoucherType = "purchase"
	VoucherTypePayment    VoucherType = "payment"
	VoucherTypeReceipt    VoucherType = "receipt"
	VoucherTypeAdjustment VoucherType = "adjustment"
	VoucherTypeClosing    VoucherType = "closing"
)

// IsValid checks if the voucher type is valid
func (t VoucherType) IsValid() bool {
	switch t {
	case VoucherTypeGeneral, VoucherTypeSales, VoucherTypePurchase,
		VoucherTypePayment, VoucherTypeReceipt, VoucherTypeAdjustment, VoucherTypeClosing:
		return true
	}
	return false
}

// GetPrefix returns the prefix for voucher number generation
func (t VoucherType) GetPrefix() string {
	switch t {
	case VoucherTypeGeneral:
		return "GJ"
	case VoucherTypeSales:
		return "SJ"
	case VoucherTypePurchase:
		return "PJ"
	case VoucherTypePayment:
		return "PM"
	case VoucherTypeReceipt:
		return "RC"
	case VoucherTypeAdjustment:
		return "AJ"
	case VoucherTypeClosing:
		return "CL"
	default:
		return "XX"
	}
}

// VoucherStatus represents the status of a voucher
type VoucherStatus string

const (
	VoucherStatusDraft     VoucherStatus = "draft"
	VoucherStatusPending   VoucherStatus = "pending"
	VoucherStatusApproved  VoucherStatus = "approved"
	VoucherStatusPosted    VoucherStatus = "posted"
	VoucherStatusRejected  VoucherStatus = "rejected"
	VoucherStatusCancelled VoucherStatus = "cancelled"
)

// IsValid checks if the voucher status is valid
func (s VoucherStatus) IsValid() bool {
	switch s {
	case VoucherStatusDraft, VoucherStatusPending, VoucherStatusApproved,
		VoucherStatusPosted, VoucherStatusRejected, VoucherStatusCancelled:
		return true
	}
	return false
}

// CanEdit returns true if voucher can be edited in this status
func (s VoucherStatus) CanEdit() bool {
	return s == VoucherStatusDraft || s == VoucherStatusRejected
}

// CanSubmit returns true if voucher can be submitted for approval
func (s VoucherStatus) CanSubmit() bool {
	return s == VoucherStatusDraft || s == VoucherStatusRejected
}

// CanApprove returns true if voucher can be approved
func (s VoucherStatus) CanApprove() bool {
	return s == VoucherStatusPending
}

// CanPost returns true if voucher can be posted
func (s VoucherStatus) CanPost() bool {
	return s == VoucherStatusApproved
}

// CanReverse returns true if voucher can be reversed
func (s VoucherStatus) CanReverse() bool {
	return s == VoucherStatusPosted
}

// Voucher errors
var (
	ErrVoucherNotFound       = errors.New("voucher not found")
	ErrVoucherUnbalanced     = errors.New("voucher debit and credit must be equal")
	ErrVoucherNoEntries      = errors.New("voucher must have at least one entry")
	ErrVoucherInvalidStatus  = errors.New("invalid voucher status")
	ErrVoucherCannotEdit     = errors.New("voucher cannot be edited in current status")
	ErrVoucherCannotSubmit   = errors.New("voucher cannot be submitted in current status")
	ErrVoucherCannotApprove  = errors.New("voucher cannot be approved in current status")
	ErrVoucherCannotReject   = errors.New("voucher cannot be rejected in current status")
	ErrVoucherCannotPost     = errors.New("voucher cannot be posted in current status")
	ErrVoucherCannotReverse  = errors.New("voucher cannot be reversed in current status")
	ErrVoucherCannotCancel   = errors.New("voucher cannot be cancelled in current status")
	ErrVoucherAlreadyReversed = errors.New("voucher has already been reversed")
	ErrInvalidVoucherType    = errors.New("invalid voucher type")
	ErrInvalidVoucherDate    = errors.New("invalid voucher date")
	ErrPeriodClosed          = errors.New("fiscal period is closed")
)

// Voucher represents a journal voucher (double-entry bookkeeping)
type Voucher struct {
	TenantModel

	// Voucher info
	VoucherNo   string        `gorm:"type:varchar(20);not null" json:"voucher_no"`
	VoucherDate time.Time     `gorm:"type:date;not null" json:"voucher_date"`
	VoucherType VoucherType   `gorm:"type:varchar(20);not null" json:"voucher_type"`
	Status      VoucherStatus `gorm:"type:varchar(20);not null;default:draft" json:"status"`

	// Amounts (for validation)
	TotalDebit  float64 `gorm:"type:decimal(18,2);not null;default:0" json:"total_debit"`
	TotalCredit float64 `gorm:"type:decimal(18,2);not null;default:0" json:"total_credit"`

	// Description
	Description   string     `gorm:"type:varchar(500)" json:"description,omitempty"`
	ReferenceType string     `gorm:"type:varchar(50)" json:"reference_type,omitempty"`
	ReferenceID   *uuid.UUID `gorm:"type:uuid" json:"reference_id,omitempty"`

	// Attachments
	AttachmentCount int `gorm:"default:0" json:"attachment_count"`

	// Approval workflow
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	SubmittedBy *uuid.UUID `gorm:"type:uuid" json:"submitted_by,omitempty"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	ApprovedBy  *uuid.UUID `gorm:"type:uuid" json:"approved_by,omitempty"`
	RejectedAt  *time.Time `json:"rejected_at,omitempty"`
	RejectedBy  *uuid.UUID `gorm:"type:uuid" json:"rejected_by,omitempty"`
	RejectionReason string `gorm:"type:varchar(500)" json:"rejection_reason,omitempty"`

	// Posting
	PostedAt *time.Time `json:"posted_at,omitempty"`
	PostedBy *uuid.UUID `gorm:"type:uuid" json:"posted_by,omitempty"`

	// Reversal
	IsReversal    bool       `gorm:"default:false" json:"is_reversal"`
	ReversalOfID  *uuid.UUID `gorm:"type:uuid" json:"reversal_of_id,omitempty"`
	ReversedByID  *uuid.UUID `gorm:"type:uuid" json:"reversed_by_id,omitempty"`

	// Audit
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy *uuid.UUID `gorm:"type:uuid" json:"updated_by,omitempty"`

	// Relations
	Entries      []VoucherEntry `gorm:"foreignKey:VoucherID" json:"entries,omitempty"`
	ReversalOf   *Voucher       `gorm:"foreignKey:ReversalOfID" json:"reversal_of,omitempty"`
	ReversedBy   *Voucher       `gorm:"foreignKey:ReversedByID" json:"reversed_by,omitempty"`
}

// TableName specifies the table name for GORM
func (Voucher) TableName() string {
	return "vouchers"
}

// Validate validates the voucher data
func (v *Voucher) Validate() error {
	if !v.VoucherType.IsValid() {
		return ErrInvalidVoucherType
	}
	if v.VoucherDate.IsZero() {
		return ErrInvalidVoucherDate
	}
	return nil
}

// ValidateBalance validates that debit equals credit
func (v *Voucher) ValidateBalance() error {
	if v.TotalDebit != v.TotalCredit {
		return ErrVoucherUnbalanced
	}
	return nil
}

// CalculateTotals calculates total debit and credit from entries
func (v *Voucher) CalculateTotals() {
	v.TotalDebit = 0
	v.TotalCredit = 0
	for _, entry := range v.Entries {
		v.TotalDebit += entry.DebitAmount
		v.TotalCredit += entry.CreditAmount
	}
}

// IsBalanced returns true if debit equals credit
func (v *Voucher) IsBalanced() bool {
	return v.TotalDebit == v.TotalCredit
}

// CanEdit returns true if voucher can be edited
func (v *Voucher) CanEdit() bool {
	return v.Status.CanEdit()
}

// Submit submits voucher for approval
func (v *Voucher) Submit(userID uuid.UUID) error {
	if !v.Status.CanSubmit() {
		return ErrVoucherCannotSubmit
	}
	if err := v.ValidateBalance(); err != nil {
		return err
	}
	if len(v.Entries) == 0 {
		return ErrVoucherNoEntries
	}

	now := time.Now()
	v.Status = VoucherStatusPending
	v.SubmittedAt = &now
	v.SubmittedBy = &userID
	return nil
}

// Approve approves the voucher
func (v *Voucher) Approve(userID uuid.UUID) error {
	if !v.Status.CanApprove() {
		return ErrVoucherCannotApprove
	}

	now := time.Now()
	v.Status = VoucherStatusApproved
	v.ApprovedAt = &now
	v.ApprovedBy = &userID
	return nil
}

// Reject rejects the voucher
func (v *Voucher) Reject(userID uuid.UUID, reason string) error {
	if !v.Status.CanApprove() {
		return ErrVoucherCannotReject
	}

	now := time.Now()
	v.Status = VoucherStatusRejected
	v.RejectedAt = &now
	v.RejectedBy = &userID
	v.RejectionReason = reason
	return nil
}

// Post posts the voucher to the ledger
func (v *Voucher) Post(userID uuid.UUID) error {
	if !v.Status.CanPost() {
		return ErrVoucherCannotPost
	}

	now := time.Now()
	v.Status = VoucherStatusPosted
	v.PostedAt = &now
	v.PostedBy = &userID
	return nil
}

// Cancel cancels the voucher
func (v *Voucher) Cancel() error {
	if v.Status == VoucherStatusPosted {
		return ErrVoucherCannotCancel
	}
	v.Status = VoucherStatusCancelled
	return nil
}

// GetTypeLabel returns Korean label for voucher type
func (v *Voucher) GetTypeLabel() string {
	switch v.VoucherType {
	case VoucherTypeGeneral:
		return "일반전표"
	case VoucherTypeSales:
		return "매출전표"
	case VoucherTypePurchase:
		return "매입전표"
	case VoucherTypePayment:
		return "지급전표"
	case VoucherTypeReceipt:
		return "입금전표"
	case VoucherTypeAdjustment:
		return "수정전표"
	case VoucherTypeClosing:
		return "결산전표"
	default:
		return ""
	}
}

// GetStatusLabel returns Korean label for voucher status
func (v *Voucher) GetStatusLabel() string {
	switch v.Status {
	case VoucherStatusDraft:
		return "작성중"
	case VoucherStatusPending:
		return "승인대기"
	case VoucherStatusApproved:
		return "승인완료"
	case VoucherStatusPosted:
		return "전기완료"
	case VoucherStatusRejected:
		return "반려"
	case VoucherStatusCancelled:
		return "취소"
	default:
		return ""
	}
}
