package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// ============================================================================
// VoucherType Tests
// ============================================================================

func TestVoucherType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		vType    domain.VoucherType
		expected bool
	}{
		{"general is valid", domain.VoucherTypeGeneral, true},
		{"sales is valid", domain.VoucherTypeSales, true},
		{"purchase is valid", domain.VoucherTypePurchase, true},
		{"payment is valid", domain.VoucherTypePayment, true},
		{"receipt is valid", domain.VoucherTypeReceipt, true},
		{"adjustment is valid", domain.VoucherTypeAdjustment, true},
		{"closing is valid", domain.VoucherTypeClosing, true},
		{"invalid type", domain.VoucherType("invalid"), false},
		{"empty type", domain.VoucherType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.vType.IsValid())
		})
	}
}

func TestVoucherType_GetPrefix(t *testing.T) {
	tests := []struct {
		name     string
		vType    domain.VoucherType
		expected string
	}{
		{"general prefix", domain.VoucherTypeGeneral, "GJ"},
		{"sales prefix", domain.VoucherTypeSales, "SJ"},
		{"purchase prefix", domain.VoucherTypePurchase, "PJ"},
		{"payment prefix", domain.VoucherTypePayment, "PM"},
		{"receipt prefix", domain.VoucherTypeReceipt, "RC"},
		{"adjustment prefix", domain.VoucherTypeAdjustment, "AJ"},
		{"closing prefix", domain.VoucherTypeClosing, "CL"},
		{"unknown type prefix", domain.VoucherType("unknown"), "XX"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.vType.GetPrefix())
		})
	}
}

// ============================================================================
// VoucherStatus Tests
// ============================================================================

func TestVoucherStatus_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   domain.VoucherStatus
		expected bool
	}{
		{"draft is valid", domain.VoucherStatusDraft, true},
		{"pending is valid", domain.VoucherStatusPending, true},
		{"approved is valid", domain.VoucherStatusApproved, true},
		{"posted is valid", domain.VoucherStatusPosted, true},
		{"rejected is valid", domain.VoucherStatusRejected, true},
		{"cancelled is valid", domain.VoucherStatusCancelled, true},
		{"invalid status", domain.VoucherStatus("invalid"), false},
		{"empty status", domain.VoucherStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsValid())
		})
	}
}

func TestVoucherStatus_CanEdit(t *testing.T) {
	tests := []struct {
		name     string
		status   domain.VoucherStatus
		expected bool
	}{
		{"draft can edit", domain.VoucherStatusDraft, true},
		{"rejected can edit", domain.VoucherStatusRejected, true},
		{"pending cannot edit", domain.VoucherStatusPending, false},
		{"approved cannot edit", domain.VoucherStatusApproved, false},
		{"posted cannot edit", domain.VoucherStatusPosted, false},
		{"cancelled cannot edit", domain.VoucherStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.CanEdit())
		})
	}
}

func TestVoucherStatus_CanSubmit(t *testing.T) {
	tests := []struct {
		name     string
		status   domain.VoucherStatus
		expected bool
	}{
		{"draft can submit", domain.VoucherStatusDraft, true},
		{"rejected can submit", domain.VoucherStatusRejected, true},
		{"pending cannot submit", domain.VoucherStatusPending, false},
		{"approved cannot submit", domain.VoucherStatusApproved, false},
		{"posted cannot submit", domain.VoucherStatusPosted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.CanSubmit())
		})
	}
}

func TestVoucherStatus_CanApprove(t *testing.T) {
	tests := []struct {
		name     string
		status   domain.VoucherStatus
		expected bool
	}{
		{"pending can approve", domain.VoucherStatusPending, true},
		{"draft cannot approve", domain.VoucherStatusDraft, false},
		{"approved cannot approve", domain.VoucherStatusApproved, false},
		{"posted cannot approve", domain.VoucherStatusPosted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.CanApprove())
		})
	}
}

func TestVoucherStatus_CanPost(t *testing.T) {
	tests := []struct {
		name     string
		status   domain.VoucherStatus
		expected bool
	}{
		{"approved can post", domain.VoucherStatusApproved, true},
		{"draft cannot post", domain.VoucherStatusDraft, false},
		{"pending cannot post", domain.VoucherStatusPending, false},
		{"posted cannot post", domain.VoucherStatusPosted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.CanPost())
		})
	}
}

func TestVoucherStatus_CanReverse(t *testing.T) {
	tests := []struct {
		name     string
		status   domain.VoucherStatus
		expected bool
	}{
		{"posted can reverse", domain.VoucherStatusPosted, true},
		{"draft cannot reverse", domain.VoucherStatusDraft, false},
		{"approved cannot reverse", domain.VoucherStatusApproved, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.CanReverse())
		})
	}
}

// ============================================================================
// Voucher Validation Tests
// ============================================================================

func TestVoucher_Validate(t *testing.T) {
	tests := []struct {
		name    string
		voucher *domain.Voucher
		wantErr error
	}{
		{
			name: "valid voucher",
			voucher: &domain.Voucher{
				VoucherType: domain.VoucherTypeGeneral,
				VoucherDate: time.Now(),
			},
			wantErr: nil,
		},
		{
			name: "invalid voucher type",
			voucher: &domain.Voucher{
				VoucherType: domain.VoucherType("invalid"),
				VoucherDate: time.Now(),
			},
			wantErr: domain.ErrInvalidVoucherType,
		},
		{
			name: "zero voucher date",
			voucher: &domain.Voucher{
				VoucherType: domain.VoucherTypeGeneral,
				VoucherDate: time.Time{},
			},
			wantErr: domain.ErrInvalidVoucherDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.voucher.Validate()
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

// ============================================================================
// Voucher Balance Validation Tests (Critical)
// ============================================================================

func TestVoucher_ValidateBalance(t *testing.T) {
	tests := []struct {
		name        string
		totalDebit  float64
		totalCredit float64
		wantErr     error
	}{
		{
			name:        "balanced voucher - equal amounts",
			totalDebit:  100000,
			totalCredit: 100000,
			wantErr:     nil,
		},
		{
			name:        "balanced voucher - zero",
			totalDebit:  0,
			totalCredit: 0,
			wantErr:     nil,
		},
		{
			name:        "balanced voucher - large amounts",
			totalDebit:  999999999.99,
			totalCredit: 999999999.99,
			wantErr:     nil,
		},
		{
			name:        "unbalanced - debit higher",
			totalDebit:  100000,
			totalCredit: 90000,
			wantErr:     domain.ErrVoucherUnbalanced,
		},
		{
			name:        "unbalanced - credit higher",
			totalDebit:  90000,
			totalCredit: 100000,
			wantErr:     domain.ErrVoucherUnbalanced,
		},
		{
			name:        "unbalanced - small difference",
			totalDebit:  100000.01,
			totalCredit: 100000.00,
			wantErr:     domain.ErrVoucherUnbalanced,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &domain.Voucher{
				TotalDebit:  tt.totalDebit,
				TotalCredit: tt.totalCredit,
			}
			err := v.ValidateBalance()
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestVoucher_IsBalanced(t *testing.T) {
	t.Run("balanced voucher", func(t *testing.T) {
		v := &domain.Voucher{TotalDebit: 100000, TotalCredit: 100000}
		assert.True(t, v.IsBalanced())
	})

	t.Run("unbalanced voucher", func(t *testing.T) {
		v := &domain.Voucher{TotalDebit: 100000, TotalCredit: 90000}
		assert.False(t, v.IsBalanced())
	})
}

func TestVoucher_CalculateTotals(t *testing.T) {
	t.Run("calculate totals from entries", func(t *testing.T) {
		v := &domain.Voucher{
			Entries: []domain.VoucherEntry{
				{DebitAmount: 50000, CreditAmount: 0},
				{DebitAmount: 50000, CreditAmount: 0},
				{DebitAmount: 0, CreditAmount: 100000},
			},
		}

		v.CalculateTotals()

		assert.Equal(t, float64(100000), v.TotalDebit)
		assert.Equal(t, float64(100000), v.TotalCredit)
	})

	t.Run("empty entries", func(t *testing.T) {
		v := &domain.Voucher{Entries: []domain.VoucherEntry{}}

		v.CalculateTotals()

		assert.Equal(t, float64(0), v.TotalDebit)
		assert.Equal(t, float64(0), v.TotalCredit)
	})

	t.Run("reset previous totals", func(t *testing.T) {
		v := &domain.Voucher{
			TotalDebit:  999999,
			TotalCredit: 888888,
			Entries: []domain.VoucherEntry{
				{DebitAmount: 100, CreditAmount: 0},
				{DebitAmount: 0, CreditAmount: 100},
			},
		}

		v.CalculateTotals()

		assert.Equal(t, float64(100), v.TotalDebit)
		assert.Equal(t, float64(100), v.TotalCredit)
	})
}

// ============================================================================
// Voucher Workflow Tests
// ============================================================================

func TestVoucher_Submit(t *testing.T) {
	userID := uuid.New()

	t.Run("submit draft voucher with balanced entries", func(t *testing.T) {
		v := &domain.Voucher{
			Status:      domain.VoucherStatusDraft,
			TotalDebit:  100000,
			TotalCredit: 100000,
			Entries: []domain.VoucherEntry{
				{DebitAmount: 100000},
				{CreditAmount: 100000},
			},
		}

		err := v.Submit(userID)

		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusPending, v.Status)
		assert.NotNil(t, v.SubmittedAt)
		assert.Equal(t, &userID, v.SubmittedBy)
	})

	t.Run("submit rejected voucher", func(t *testing.T) {
		v := &domain.Voucher{
			Status:      domain.VoucherStatusRejected,
			TotalDebit:  100000,
			TotalCredit: 100000,
			Entries: []domain.VoucherEntry{
				{DebitAmount: 100000},
				{CreditAmount: 100000},
			},
		}

		err := v.Submit(userID)

		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusPending, v.Status)
	})

	t.Run("cannot submit unbalanced voucher", func(t *testing.T) {
		v := &domain.Voucher{
			Status:      domain.VoucherStatusDraft,
			TotalDebit:  100000,
			TotalCredit: 90000, // Unbalanced
			Entries: []domain.VoucherEntry{
				{DebitAmount: 100000},
			},
		}

		err := v.Submit(userID)

		assert.Equal(t, domain.ErrVoucherUnbalanced, err)
		assert.Equal(t, domain.VoucherStatusDraft, v.Status) // Status unchanged
	})

	t.Run("cannot submit voucher without entries", func(t *testing.T) {
		v := &domain.Voucher{
			Status:      domain.VoucherStatusDraft,
			TotalDebit:  0,
			TotalCredit: 0,
			Entries:     []domain.VoucherEntry{},
		}

		err := v.Submit(userID)

		assert.Equal(t, domain.ErrVoucherNoEntries, err)
	})

	t.Run("cannot submit pending voucher", func(t *testing.T) {
		v := &domain.Voucher{
			Status:      domain.VoucherStatusPending,
			TotalDebit:  100000,
			TotalCredit: 100000,
			Entries:     []domain.VoucherEntry{{DebitAmount: 100000}, {CreditAmount: 100000}},
		}

		err := v.Submit(userID)

		assert.Equal(t, domain.ErrVoucherCannotSubmit, err)
	})

	t.Run("cannot submit approved voucher", func(t *testing.T) {
		v := &domain.Voucher{
			Status:      domain.VoucherStatusApproved,
			TotalDebit:  100000,
			TotalCredit: 100000,
			Entries:     []domain.VoucherEntry{{DebitAmount: 100000}, {CreditAmount: 100000}},
		}

		err := v.Submit(userID)

		assert.Equal(t, domain.ErrVoucherCannotSubmit, err)
	})

	t.Run("cannot submit posted voucher", func(t *testing.T) {
		v := &domain.Voucher{
			Status:      domain.VoucherStatusPosted,
			TotalDebit:  100000,
			TotalCredit: 100000,
			Entries:     []domain.VoucherEntry{{DebitAmount: 100000}, {CreditAmount: 100000}},
		}

		err := v.Submit(userID)

		assert.Equal(t, domain.ErrVoucherCannotSubmit, err)
	})
}

func TestVoucher_Approve(t *testing.T) {
	userID := uuid.New()

	t.Run("approve pending voucher", func(t *testing.T) {
		v := &domain.Voucher{Status: domain.VoucherStatusPending}

		err := v.Approve(userID)

		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusApproved, v.Status)
		assert.NotNil(t, v.ApprovedAt)
		assert.Equal(t, &userID, v.ApprovedBy)
	})

	t.Run("cannot approve draft voucher", func(t *testing.T) {
		v := &domain.Voucher{Status: domain.VoucherStatusDraft}

		err := v.Approve(userID)

		assert.Equal(t, domain.ErrVoucherCannotApprove, err)
	})

	t.Run("cannot approve already approved voucher", func(t *testing.T) {
		v := &domain.Voucher{Status: domain.VoucherStatusApproved}

		err := v.Approve(userID)

		assert.Equal(t, domain.ErrVoucherCannotApprove, err)
	})
}

func TestVoucher_Reject(t *testing.T) {
	userID := uuid.New()

	t.Run("reject pending voucher", func(t *testing.T) {
		v := &domain.Voucher{Status: domain.VoucherStatusPending}
		reason := "Incomplete documentation"

		err := v.Reject(userID, reason)

		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusRejected, v.Status)
		assert.NotNil(t, v.RejectedAt)
		assert.Equal(t, &userID, v.RejectedBy)
		assert.Equal(t, reason, v.RejectionReason)
	})

	t.Run("cannot reject draft voucher", func(t *testing.T) {
		v := &domain.Voucher{Status: domain.VoucherStatusDraft}

		err := v.Reject(userID, "reason")

		assert.Equal(t, domain.ErrVoucherCannotReject, err)
	})
}

func TestVoucher_Post(t *testing.T) {
	userID := uuid.New()

	t.Run("post approved voucher", func(t *testing.T) {
		v := &domain.Voucher{Status: domain.VoucherStatusApproved}

		err := v.Post(userID)

		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusPosted, v.Status)
		assert.NotNil(t, v.PostedAt)
		assert.Equal(t, &userID, v.PostedBy)
	})

	t.Run("cannot post draft voucher", func(t *testing.T) {
		v := &domain.Voucher{Status: domain.VoucherStatusDraft}

		err := v.Post(userID)

		assert.Equal(t, domain.ErrVoucherCannotPost, err)
	})

	t.Run("cannot post pending voucher", func(t *testing.T) {
		v := &domain.Voucher{Status: domain.VoucherStatusPending}

		err := v.Post(userID)

		assert.Equal(t, domain.ErrVoucherCannotPost, err)
	})

	t.Run("cannot post already posted voucher", func(t *testing.T) {
		v := &domain.Voucher{Status: domain.VoucherStatusPosted}

		err := v.Post(userID)

		assert.Equal(t, domain.ErrVoucherCannotPost, err)
	})
}

func TestVoucher_Cancel(t *testing.T) {
	t.Run("cancel draft voucher", func(t *testing.T) {
		v := &domain.Voucher{Status: domain.VoucherStatusDraft}

		err := v.Cancel()

		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusCancelled, v.Status)
	})

	t.Run("cancel pending voucher", func(t *testing.T) {
		v := &domain.Voucher{Status: domain.VoucherStatusPending}

		err := v.Cancel()

		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusCancelled, v.Status)
	})

	t.Run("cannot cancel posted voucher", func(t *testing.T) {
		v := &domain.Voucher{Status: domain.VoucherStatusPosted}

		err := v.Cancel()

		assert.Equal(t, domain.ErrVoucherCannotCancel, err)
	})
}

// ============================================================================
// Voucher Workflow Integration Tests
// ============================================================================

func TestVoucher_FullWorkflow(t *testing.T) {
	userID := uuid.New()
	approverID := uuid.New()

	t.Run("complete workflow: draft -> pending -> approved -> posted", func(t *testing.T) {
		v := &domain.Voucher{
			Status:      domain.VoucherStatusDraft,
			VoucherType: domain.VoucherTypeGeneral,
			VoucherDate: time.Now(),
			TotalDebit:  100000,
			TotalCredit: 100000,
			Entries: []domain.VoucherEntry{
				{DebitAmount: 100000},
				{CreditAmount: 100000},
			},
		}

		// Step 1: Submit
		err := v.Submit(userID)
		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusPending, v.Status)

		// Step 2: Approve
		err = v.Approve(approverID)
		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusApproved, v.Status)

		// Step 3: Post
		err = v.Post(approverID)
		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusPosted, v.Status)
	})

	t.Run("workflow with rejection: draft -> pending -> rejected -> pending -> approved", func(t *testing.T) {
		v := &domain.Voucher{
			Status:      domain.VoucherStatusDraft,
			TotalDebit:  100000,
			TotalCredit: 100000,
			Entries: []domain.VoucherEntry{
				{DebitAmount: 100000},
				{CreditAmount: 100000},
			},
		}

		// Submit
		err := v.Submit(userID)
		require.NoError(t, err)

		// Reject
		err = v.Reject(approverID, "Need correction")
		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusRejected, v.Status)

		// Re-submit
		err = v.Submit(userID)
		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusPending, v.Status)

		// Approve
		err = v.Approve(approverID)
		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusApproved, v.Status)
	})
}

// ============================================================================
// Label Tests
// ============================================================================

func TestVoucher_GetTypeLabel(t *testing.T) {
	tests := []struct {
		vType    domain.VoucherType
		expected string
	}{
		{domain.VoucherTypeGeneral, "일반전표"},
		{domain.VoucherTypeSales, "매출전표"},
		{domain.VoucherTypePurchase, "매입전표"},
		{domain.VoucherTypePayment, "지급전표"},
		{domain.VoucherTypeReceipt, "입금전표"},
		{domain.VoucherTypeAdjustment, "수정전표"},
		{domain.VoucherTypeClosing, "결산전표"},
	}

	for _, tt := range tests {
		t.Run(string(tt.vType), func(t *testing.T) {
			v := &domain.Voucher{VoucherType: tt.vType}
			assert.Equal(t, tt.expected, v.GetTypeLabel())
		})
	}
}

func TestVoucher_GetStatusLabel(t *testing.T) {
	tests := []struct {
		status   domain.VoucherStatus
		expected string
	}{
		{domain.VoucherStatusDraft, "작성중"},
		{domain.VoucherStatusPending, "승인대기"},
		{domain.VoucherStatusApproved, "승인완료"},
		{domain.VoucherStatusPosted, "전기완료"},
		{domain.VoucherStatusRejected, "반려"},
		{domain.VoucherStatusCancelled, "취소"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			v := &domain.Voucher{Status: tt.status}
			assert.Equal(t, tt.expected, v.GetStatusLabel())
		})
	}
}
