package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/mocks"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// ============================================================================
// Test Helpers
// ============================================================================

func newTestVoucherService() (*mocks.MockVoucherRepository, *mocks.MockAccountRepository, service.VoucherService) {
	voucherRepo := new(mocks.MockVoucherRepository)
	accountRepo := new(mocks.MockAccountRepository)
	svc := service.NewVoucherService(voucherRepo, accountRepo)
	return voucherRepo, accountRepo, svc
}

func newTestCompanyID() uuid.UUID {
	return uuid.MustParse("00000000-0000-0000-0000-000000000001")
}

func newTestUserID() uuid.UUID {
	return uuid.MustParse("00000000-0000-0000-0000-000000000002")
}

func newTestAccount(companyID, accountID uuid.UUID) *domain.Account {
	return &domain.Account{
		TenantModel: domain.TenantModel{
			BaseModel: domain.BaseModel{
				ID: accountID,
			},
			CompanyID: companyID,
		},
		Code:               "101",
		Name:               "Cash",
		AccountType:        domain.AccountTypeAsset,
		AccountNature:      domain.AccountNatureDebit,
		IsActive:           true,
		AllowDirectPosting: true,
		IsControlAccount:   false,
	}
}

func newTestVoucher(companyID uuid.UUID) *domain.Voucher {
	accountID1 := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	accountID2 := uuid.MustParse("00000000-0000-0000-0000-000000000011")

	return &domain.Voucher{
		TenantModel: domain.TenantModel{
			BaseModel: domain.BaseModel{
				ID: uuid.New(),
			},
			CompanyID: companyID,
		},
		VoucherDate: time.Now(),
		VoucherType: domain.VoucherTypeGeneral,
		Description: "Test voucher",
		Status:      domain.VoucherStatusDraft,
		Entries: []domain.VoucherEntry{
			{
				CompanyID:    companyID,
				AccountID:    accountID1,
				DebitAmount:  1000,
				CreditAmount: 0,
				Description:  "Debit entry",
			},
			{
				CompanyID:    companyID,
				AccountID:    accountID2,
				DebitAmount:  0,
				CreditAmount: 1000,
				Description:  "Credit entry",
			},
		},
	}
}

// ============================================================================
// Create Tests
// ============================================================================

func TestVoucherService_Create(t *testing.T) {
	t.Run("successfully creates voucher", func(t *testing.T) {
		voucherRepo, accountRepo, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		voucher := newTestVoucher(companyID)

		// Mock account validation
		for _, entry := range voucher.Entries {
			account := newTestAccount(companyID, entry.AccountID)
			accountRepo.On("FindByID", ctx, companyID, entry.AccountID).Return(account, nil).Once()
		}

		// Mock voucher number generation
		voucherRepo.On("GenerateVoucherNo", ctx, companyID, voucher.VoucherType, mock.AnythingOfType("time.Time")).
			Return("GEN-2024-0001", nil).Once()

		// Mock create
		voucherRepo.On("Create", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		err := svc.Create(ctx, voucher)

		require.NoError(t, err)
		assert.Equal(t, "GEN-2024-0001", voucher.VoucherNo)
		assert.Equal(t, domain.VoucherStatusDraft, voucher.Status)
		assert.Equal(t, 1, voucher.Entries[0].LineNo)
		assert.Equal(t, 2, voucher.Entries[1].LineNo)
		voucherRepo.AssertExpectations(t)
		accountRepo.AssertExpectations(t)
	})

	t.Run("fails with no entries", func(t *testing.T) {
		_, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()

		voucher := &domain.Voucher{
			TenantModel: domain.TenantModel{
				CompanyID: companyID,
			},
			VoucherDate: time.Now(),
			VoucherType: domain.VoucherTypeGeneral,
			Description: "Test voucher",
			Entries:     []domain.VoucherEntry{},
		}

		err := svc.Create(ctx, voucher)

		assert.Equal(t, domain.ErrVoucherNoEntries, err)
	})

	t.Run("fails with unbalanced entries", func(t *testing.T) {
		voucherRepo, accountRepo, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		accountID1 := uuid.New()
		accountID2 := uuid.New()

		voucher := &domain.Voucher{
			TenantModel: domain.TenantModel{
				CompanyID: companyID,
			},
			VoucherDate: time.Now(),
			VoucherType: domain.VoucherTypeGeneral,
			Description: "Unbalanced voucher",
			Entries: []domain.VoucherEntry{
				{
					CompanyID:    companyID,
					AccountID:    accountID1,
					DebitAmount:  1000,
					CreditAmount: 0,
				},
				{
					CompanyID:    companyID,
					AccountID:    accountID2,
					DebitAmount:  0,
					CreditAmount: 500, // Unbalanced!
				},
			},
		}

		// Mock account validation
		account1 := newTestAccount(companyID, accountID1)
		account2 := newTestAccount(companyID, accountID2)
		accountRepo.On("FindByID", ctx, companyID, accountID1).Return(account1, nil).Once()
		accountRepo.On("FindByID", ctx, companyID, accountID2).Return(account2, nil).Once()

		// ValidateEntries will fail due to unbalanced
		err := svc.Create(ctx, voucher)

		assert.Equal(t, domain.ErrVoucherUnbalanced, err)
		voucherRepo.AssertNotCalled(t, "Create")
	})

	t.Run("fails when account cannot post", func(t *testing.T) {
		_, accountRepo, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		voucher := newTestVoucher(companyID)

		// First account is a control account (cannot post)
		controlAccount := newTestAccount(companyID, voucher.Entries[0].AccountID)
		controlAccount.IsControlAccount = true // Cannot post
		accountRepo.On("FindByID", ctx, companyID, voucher.Entries[0].AccountID).Return(controlAccount, nil).Once()

		err := svc.Create(ctx, voucher)

		assert.Equal(t, domain.ErrControlAccountPosting, err)
	})

	t.Run("fails when voucher number generation fails", func(t *testing.T) {
		voucherRepo, accountRepo, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		voucher := newTestVoucher(companyID)

		// Mock account validation
		for _, entry := range voucher.Entries {
			account := newTestAccount(companyID, entry.AccountID)
			accountRepo.On("FindByID", ctx, companyID, entry.AccountID).Return(account, nil).Once()
		}

		// Mock voucher number generation failure
		genErr := errors.New("sequence error")
		voucherRepo.On("GenerateVoucherNo", ctx, companyID, voucher.VoucherType, mock.AnythingOfType("time.Time")).
			Return("", genErr).Once()

		err := svc.Create(ctx, voucher)

		assert.Equal(t, genErr, err)
	})
}

// ============================================================================
// Update Tests
// ============================================================================

func TestVoucherService_Update(t *testing.T) {
	t.Run("successfully updates draft voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		voucher := newTestVoucher(companyID)
		voucher.Status = domain.VoucherStatusDraft

		// Mock find existing
		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(voucher, nil).Once()
		voucherRepo.On("Update", ctx, voucher).Return(nil).Once()

		err := svc.Update(ctx, voucher)

		require.NoError(t, err)
		voucherRepo.AssertExpectations(t)
	})

	t.Run("fails to update posted voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		voucher := newTestVoucher(companyID)

		existingVoucher := newTestVoucher(companyID)
		existingVoucher.ID = voucher.ID
		existingVoucher.Status = domain.VoucherStatusPosted // Cannot edit

		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(existingVoucher, nil).Once()

		err := svc.Update(ctx, voucher)

		assert.Equal(t, domain.ErrVoucherCannotEdit, err)
		voucherRepo.AssertNotCalled(t, "Update")
	})

	t.Run("fails when voucher not found", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		voucher := newTestVoucher(companyID)

		notFoundErr := errors.New("voucher not found")
		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(nil, notFoundErr).Once()

		err := svc.Update(ctx, voucher)

		assert.Equal(t, notFoundErr, err)
	})
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestVoucherService_Delete(t *testing.T) {
	t.Run("successfully deletes draft voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		voucherID := uuid.New()

		existingVoucher := newTestVoucher(companyID)
		existingVoucher.ID = voucherID
		existingVoucher.Status = domain.VoucherStatusDraft

		voucherRepo.On("FindByID", ctx, companyID, voucherID).Return(existingVoucher, nil).Once()
		voucherRepo.On("Delete", ctx, companyID, voucherID).Return(nil).Once()

		err := svc.Delete(ctx, companyID, voucherID)

		require.NoError(t, err)
		voucherRepo.AssertExpectations(t)
	})

	t.Run("fails to delete posted voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		voucherID := uuid.New()

		existingVoucher := newTestVoucher(companyID)
		existingVoucher.ID = voucherID
		existingVoucher.Status = domain.VoucherStatusPosted

		voucherRepo.On("FindByID", ctx, companyID, voucherID).Return(existingVoucher, nil).Once()

		err := svc.Delete(ctx, companyID, voucherID)

		assert.Equal(t, domain.ErrVoucherCannotEdit, err)
		voucherRepo.AssertNotCalled(t, "Delete")
	})
}

// ============================================================================
// Workflow Tests
// ============================================================================

func TestVoucherService_Submit(t *testing.T) {
	t.Run("successfully submits draft voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		userID := newTestUserID()
		voucherID := uuid.New()

		existingVoucher := newTestVoucher(companyID)
		existingVoucher.ID = voucherID
		existingVoucher.Status = domain.VoucherStatusDraft

		voucherRepo.On("FindByID", ctx, companyID, voucherID).Return(existingVoucher, nil).Once()
		voucherRepo.On("UpdateStatus", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		err := svc.Submit(ctx, companyID, voucherID, userID)

		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusPending, existingVoucher.Status)
		voucherRepo.AssertExpectations(t)
	})

	t.Run("fails to submit already pending voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		userID := newTestUserID()
		voucherID := uuid.New()

		existingVoucher := newTestVoucher(companyID)
		existingVoucher.ID = voucherID
		existingVoucher.Status = domain.VoucherStatusPending

		voucherRepo.On("FindByID", ctx, companyID, voucherID).Return(existingVoucher, nil).Once()

		err := svc.Submit(ctx, companyID, voucherID, userID)

		assert.Equal(t, domain.ErrVoucherCannotSubmit, err)
	})
}

func TestVoucherService_Approve(t *testing.T) {
	t.Run("successfully approves pending voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		userID := newTestUserID()
		voucherID := uuid.New()

		existingVoucher := newTestVoucher(companyID)
		existingVoucher.ID = voucherID
		existingVoucher.Status = domain.VoucherStatusPending

		voucherRepo.On("FindByID", ctx, companyID, voucherID).Return(existingVoucher, nil).Once()
		voucherRepo.On("UpdateStatus", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		err := svc.Approve(ctx, companyID, voucherID, userID)

		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusApproved, existingVoucher.Status)
		voucherRepo.AssertExpectations(t)
	})

	t.Run("fails to approve draft voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		userID := newTestUserID()
		voucherID := uuid.New()

		existingVoucher := newTestVoucher(companyID)
		existingVoucher.ID = voucherID
		existingVoucher.Status = domain.VoucherStatusDraft

		voucherRepo.On("FindByID", ctx, companyID, voucherID).Return(existingVoucher, nil).Once()

		err := svc.Approve(ctx, companyID, voucherID, userID)

		assert.Equal(t, domain.ErrVoucherCannotApprove, err)
	})
}

func TestVoucherService_Reject(t *testing.T) {
	t.Run("successfully rejects pending voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		userID := newTestUserID()
		voucherID := uuid.New()
		reason := "Invalid documentation"

		existingVoucher := newTestVoucher(companyID)
		existingVoucher.ID = voucherID
		existingVoucher.Status = domain.VoucherStatusPending

		voucherRepo.On("FindByID", ctx, companyID, voucherID).Return(existingVoucher, nil).Once()
		voucherRepo.On("UpdateStatus", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		err := svc.Reject(ctx, companyID, voucherID, userID, reason)

		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusRejected, existingVoucher.Status)
		assert.Equal(t, reason, existingVoucher.RejectionReason)
		voucherRepo.AssertExpectations(t)
	})
}

func TestVoucherService_Post(t *testing.T) {
	t.Run("successfully posts approved voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		userID := newTestUserID()
		voucherID := uuid.New()

		existingVoucher := newTestVoucher(companyID)
		existingVoucher.ID = voucherID
		existingVoucher.Status = domain.VoucherStatusApproved

		voucherRepo.On("FindByID", ctx, companyID, voucherID).Return(existingVoucher, nil).Once()
		voucherRepo.On("UpdateStatus", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		err := svc.Post(ctx, companyID, voucherID, userID)

		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusPosted, existingVoucher.Status)
		voucherRepo.AssertExpectations(t)
	})

	t.Run("fails to post draft voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		userID := newTestUserID()
		voucherID := uuid.New()

		existingVoucher := newTestVoucher(companyID)
		existingVoucher.ID = voucherID
		existingVoucher.Status = domain.VoucherStatusDraft

		voucherRepo.On("FindByID", ctx, companyID, voucherID).Return(existingVoucher, nil).Once()

		err := svc.Post(ctx, companyID, voucherID, userID)

		assert.Equal(t, domain.ErrVoucherCannotPost, err)
	})
}

func TestVoucherService_Cancel(t *testing.T) {
	t.Run("successfully cancels pending voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		voucherID := uuid.New()

		existingVoucher := newTestVoucher(companyID)
		existingVoucher.ID = voucherID
		existingVoucher.Status = domain.VoucherStatusPending

		voucherRepo.On("FindByID", ctx, companyID, voucherID).Return(existingVoucher, nil).Once()
		voucherRepo.On("UpdateStatus", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		err := svc.Cancel(ctx, companyID, voucherID)

		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusCancelled, existingVoucher.Status)
		voucherRepo.AssertExpectations(t)
	})

	t.Run("fails to cancel posted voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		voucherID := uuid.New()

		existingVoucher := newTestVoucher(companyID)
		existingVoucher.ID = voucherID
		existingVoucher.Status = domain.VoucherStatusPosted

		voucherRepo.On("FindByID", ctx, companyID, voucherID).Return(existingVoucher, nil).Once()

		err := svc.Cancel(ctx, companyID, voucherID)

		assert.Equal(t, domain.ErrVoucherCannotCancel, err)
	})
}

// ============================================================================
// Workflow Integration Tests
// ============================================================================

func TestVoucherService_FullWorkflow(t *testing.T) {
	t.Run("complete workflow: Draft -> Pending -> Approved -> Posted", func(t *testing.T) {
		voucherRepo, accountRepo, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		userID := newTestUserID()

		// Create voucher
		voucher := newTestVoucher(companyID)

		// Mock account validation
		for _, entry := range voucher.Entries {
			account := newTestAccount(companyID, entry.AccountID)
			accountRepo.On("FindByID", ctx, companyID, entry.AccountID).Return(account, nil).Once()
		}

		voucherRepo.On("GenerateVoucherNo", ctx, companyID, voucher.VoucherType, mock.AnythingOfType("time.Time")).
			Return("GEN-2024-0001", nil).Once()
		voucherRepo.On("Create", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		err := svc.Create(ctx, voucher)
		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusDraft, voucher.Status)

		// Submit (Draft -> Pending)
		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(voucher, nil).Once()
		voucherRepo.On("UpdateStatus", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		err = svc.Submit(ctx, companyID, voucher.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusPending, voucher.Status)

		// Approve (Pending -> Approved)
		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(voucher, nil).Once()
		voucherRepo.On("UpdateStatus", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		err = svc.Approve(ctx, companyID, voucher.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusApproved, voucher.Status)

		// Post (Approved -> Posted)
		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(voucher, nil).Once()
		voucherRepo.On("UpdateStatus", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		err = svc.Post(ctx, companyID, voucher.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, domain.VoucherStatusPosted, voucher.Status)

		voucherRepo.AssertExpectations(t)
		accountRepo.AssertExpectations(t)
	})
}

// ============================================================================
// Reversal Tests
// ============================================================================

func TestVoucherService_Reverse(t *testing.T) {
	t.Run("successfully reverses posted voucher", func(t *testing.T) {
		voucherRepo, accountRepo, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		userID := newTestUserID()
		reversalDate := time.Now()
		description := "Reversal of GEN-2024-0001"

		originalVoucher := newTestVoucher(companyID)
		originalVoucher.Status = domain.VoucherStatusPosted
		originalVoucher.VoucherNo = "GEN-2024-0001"

		// Find original
		voucherRepo.On("FindByID", ctx, companyID, originalVoucher.ID).Return(originalVoucher, nil).Twice()

		// Validate accounts for reversal entries
		for _, entry := range originalVoucher.Entries {
			account := newTestAccount(companyID, entry.AccountID)
			accountRepo.On("FindByID", ctx, companyID, entry.AccountID).Return(account, nil).Once()
		}

		// Generate number for reversal
		voucherRepo.On("GenerateVoucherNo", ctx, companyID, originalVoucher.VoucherType, mock.AnythingOfType("time.Time")).
			Return("GEN-2024-0002", nil).Once()

		// Create reversal
		voucherRepo.On("Create", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		// Update original to reference reversal
		voucherRepo.On("Update", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		reversal, err := svc.Reverse(ctx, companyID, originalVoucher.ID, userID, reversalDate, description)

		require.NoError(t, err)
		require.NotNil(t, reversal)
		assert.True(t, reversal.IsReversal)
		assert.Equal(t, &originalVoucher.ID, reversal.ReversalOfID)
		assert.Equal(t, description, reversal.Description)

		// Check entries are swapped
		assert.Equal(t, originalVoucher.Entries[0].CreditAmount, reversal.Entries[0].DebitAmount)
		assert.Equal(t, originalVoucher.Entries[0].DebitAmount, reversal.Entries[0].CreditAmount)

		voucherRepo.AssertExpectations(t)
		accountRepo.AssertExpectations(t)
	})

	t.Run("fails to reverse draft voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		userID := newTestUserID()

		originalVoucher := newTestVoucher(companyID)
		originalVoucher.Status = domain.VoucherStatusDraft

		voucherRepo.On("FindByID", ctx, companyID, originalVoucher.ID).Return(originalVoucher, nil).Once()

		_, err := svc.Reverse(ctx, companyID, originalVoucher.ID, userID, time.Now(), "Reversal")

		assert.Equal(t, domain.ErrVoucherCannotReverse, err)
	})

	t.Run("fails to reverse already reversed voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		userID := newTestUserID()
		reversalID := uuid.New()

		originalVoucher := newTestVoucher(companyID)
		originalVoucher.Status = domain.VoucherStatusPosted
		originalVoucher.ReversedByID = &reversalID // Already reversed

		voucherRepo.On("FindByID", ctx, companyID, originalVoucher.ID).Return(originalVoucher, nil).Once()

		_, err := svc.Reverse(ctx, companyID, originalVoucher.ID, userID, time.Now(), "Reversal")

		assert.Equal(t, domain.ErrVoucherAlreadyReversed, err)
	})
}

// ============================================================================
// Query Tests
// ============================================================================

func TestVoucherService_GetByID(t *testing.T) {
	t.Run("successfully retrieves voucher", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		voucher := newTestVoucher(companyID)

		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(voucher, nil).Once()

		result, err := svc.GetByID(ctx, companyID, voucher.ID)

		require.NoError(t, err)
		assert.Equal(t, voucher, result)
		voucherRepo.AssertExpectations(t)
	})
}

func TestVoucherService_List(t *testing.T) {
	t.Run("successfully lists vouchers", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()

		filter := repository.VoucherFilter{
			CompanyID: companyID,
			Page:      1,
			PageSize:  10,
		}

		vouchers := []domain.Voucher{*newTestVoucher(companyID), *newTestVoucher(companyID)}
		total := int64(2)

		voucherRepo.On("FindAll", ctx, filter).Return(vouchers, total, nil).Once()

		result, count, err := svc.List(ctx, filter)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, total, count)
		voucherRepo.AssertExpectations(t)
	})
}

func TestVoucherService_GetPending(t *testing.T) {
	t.Run("successfully retrieves pending vouchers", func(t *testing.T) {
		voucherRepo, _, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()

		pendingVoucher := newTestVoucher(companyID)
		pendingVoucher.Status = domain.VoucherStatusPending
		vouchers := []domain.Voucher{*pendingVoucher}

		voucherRepo.On("FindByStatus", ctx, companyID, domain.VoucherStatusPending).Return(vouchers, nil).Once()

		result, err := svc.GetPending(ctx, companyID)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, domain.VoucherStatusPending, result[0].Status)
		voucherRepo.AssertExpectations(t)
	})
}

// ============================================================================
// Entry Operations Tests
// ============================================================================

func TestVoucherService_ValidateEntries(t *testing.T) {
	t.Run("validates balanced entries", func(t *testing.T) {
		_, accountRepo, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		accountID1 := uuid.New()
		accountID2 := uuid.New()

		entries := []domain.VoucherEntry{
			{
				CompanyID:    companyID,
				AccountID:    accountID1,
				DebitAmount:  1000,
				CreditAmount: 0,
			},
			{
				CompanyID:    companyID,
				AccountID:    accountID2,
				DebitAmount:  0,
				CreditAmount: 1000,
			},
		}

		accountRepo.On("FindByID", ctx, companyID, accountID1).Return(newTestAccount(companyID, accountID1), nil).Once()
		accountRepo.On("FindByID", ctx, companyID, accountID2).Return(newTestAccount(companyID, accountID2), nil).Once()

		err := svc.ValidateEntries(ctx, companyID, entries)

		require.NoError(t, err)
		accountRepo.AssertExpectations(t)
	})

	t.Run("rejects unbalanced entries", func(t *testing.T) {
		_, accountRepo, svc := newTestVoucherService()
		ctx := context.Background()
		companyID := newTestCompanyID()
		accountID1 := uuid.New()
		accountID2 := uuid.New()

		entries := []domain.VoucherEntry{
			{
				CompanyID:    companyID,
				AccountID:    accountID1,
				DebitAmount:  1000,
				CreditAmount: 0,
			},
			{
				CompanyID:    companyID,
				AccountID:    accountID2,
				DebitAmount:  0,
				CreditAmount: 500, // Unbalanced
			},
		}

		accountRepo.On("FindByID", ctx, companyID, accountID1).Return(newTestAccount(companyID, accountID1), nil).Once()
		accountRepo.On("FindByID", ctx, companyID, accountID2).Return(newTestAccount(companyID, accountID2), nil).Once()

		err := svc.ValidateEntries(ctx, companyID, entries)

		assert.Equal(t, domain.ErrVoucherUnbalanced, err)
	})
}
