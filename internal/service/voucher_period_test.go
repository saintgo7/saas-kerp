package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/mocks"
	"github.com/saintgo7/saas-kerp/internal/service"
)

func newVoucherServiceWithLedger() (*mocks.MockVoucherRepository, *mocks.MockAccountRepository, *mocks.MockLedgerRepository, service.VoucherService) {
	voucherRepo := new(mocks.MockVoucherRepository)
	accountRepo := new(mocks.MockAccountRepository)
	ledgerRepo := new(mocks.MockLedgerRepository)
	svc := service.NewVoucherService(voucherRepo, accountRepo, ledgerRepo)
	return voucherRepo, accountRepo, ledgerRepo, svc
}

// ============================================================================
// Posting into a closed fiscal period
// ============================================================================

func TestVoucherService_Post_FiscalPeriodGate(t *testing.T) {
	companyID := newTestCompanyID()
	userID := newTestUserID()

	approvedVoucherOn := func(date time.Time) *domain.Voucher {
		v := newTestVoucher(companyID)
		v.Status = domain.VoucherStatusApproved
		v.VoucherDate = date
		return v
	}

	t.Run("마감된 기간에는 전기할 수 없다", func(t *testing.T) {
		voucherRepo, _, ledgerRepo, svc := newVoucherServiceWithLedger()
		ctx := context.Background()

		date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
		voucher := approvedVoucherOn(date)

		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(voucher, nil).Once()
		ledgerRepo.On("GetFiscalPeriod", ctx, companyID, 2026, 1).
			Return(&domain.FiscalPeriod{Status: domain.FiscalPeriodClosed}, nil).Once()

		err := svc.Post(ctx, companyID, voucher.ID, userID)

		// FiscalPeriod.CanPost and domain.ErrPeriodClosed were both dead code
		// before this: a voucher dated inside an already closed and carried
		// forward month could be posted and silently rewrite a published trial
		// balance.
		assert.ErrorIs(t, err, domain.ErrPeriodClosed)
		assert.Equal(t, domain.VoucherStatusApproved, voucher.Status, "상태가 바뀌면 안 된다")

		voucherRepo.AssertExpectations(t)
		ledgerRepo.AssertExpectations(t)
	})

	t.Run("잠긴 기간에도 전기할 수 없다", func(t *testing.T) {
		voucherRepo, _, ledgerRepo, svc := newVoucherServiceWithLedger()
		ctx := context.Background()

		date := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
		voucher := approvedVoucherOn(date)

		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(voucher, nil).Once()
		ledgerRepo.On("GetFiscalPeriod", ctx, companyID, 2026, 2).
			Return(&domain.FiscalPeriod{Status: domain.FiscalPeriodLocked}, nil).Once()

		assert.ErrorIs(t, svc.Post(ctx, companyID, voucher.ID, userID), domain.ErrPeriodClosed)
	})

	t.Run("열린 기간에는 전기된다", func(t *testing.T) {
		voucherRepo, _, ledgerRepo, svc := newVoucherServiceWithLedger()
		ctx := context.Background()

		date := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		voucher := approvedVoucherOn(date)

		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(voucher, nil).Once()
		ledgerRepo.On("GetFiscalPeriod", ctx, companyID, 2026, 3).
			Return(&domain.FiscalPeriod{Status: domain.FiscalPeriodOpen}, nil).Once()
		voucherRepo.On("UpdateStatus", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		require.NoError(t, svc.Post(ctx, companyID, voucher.ID, userID))
		assert.Equal(t, domain.VoucherStatusPosted, voucher.Status)
	})

	t.Run("회계기간 행이 없으면 전기를 막지 않는다", func(t *testing.T) {
		// Many tenants never call CreateFiscalPeriods; refusing every posting
		// for them would be a far bigger regression than the case this guards.
		voucherRepo, _, ledgerRepo, svc := newVoucherServiceWithLedger()
		ctx := context.Background()

		date := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
		voucher := approvedVoucherOn(date)

		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(voucher, nil).Once()
		ledgerRepo.On("GetFiscalPeriod", ctx, companyID, 2026, 4).
			Return(nil, domain.ErrFiscalPeriodNotFound).Once()
		voucherRepo.On("UpdateStatus", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		require.NoError(t, svc.Post(ctx, companyID, voucher.ID, userID))
		assert.Equal(t, domain.VoucherStatusPosted, voucher.Status)
	})

	t.Run("회계기간 조회 실패는 전파한다", func(t *testing.T) {
		voucherRepo, _, ledgerRepo, svc := newVoucherServiceWithLedger()
		ctx := context.Background()

		date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
		voucher := approvedVoucherOn(date)

		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(voucher, nil).Once()
		ledgerRepo.On("GetFiscalPeriod", ctx, companyID, 2026, 5).
			Return(nil, assert.AnError).Once()

		assert.ErrorIs(t, svc.Post(ctx, companyID, voucher.ID, userID), assert.AnError)
	})
}

// ============================================================================
// Posting a voucher whose header no longer matches its lines
// ============================================================================

func TestVoucherService_Post_RevalidatesStoredEntries(t *testing.T) {
	companyID := newTestCompanyID()
	userID := newTestUserID()

	t.Run("헤더 합계가 분개행과 다르면 전기를 거부한다", func(t *testing.T) {
		voucherRepo, _, _, svc := newVoucherServiceWithLedger()
		ctx := context.Background()

		// This is the state AddEntry used to leave behind: the line was
		// committed but the header update was rolled back.
		voucher := newTestVoucher(companyID)
		voucher.Status = domain.VoucherStatusApproved
		voucher.Entries = append(voucher.Entries, domain.VoucherEntry{
			CompanyID:   companyID,
			AccountID:   uuid.New(),
			DebitAmount: 50,
		})

		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(voucher, nil).Once()

		assert.ErrorIs(t, svc.Post(ctx, companyID, voucher.ID, userID), domain.ErrVoucherTotalsMismatch)
	})

	t.Run("분개행이 없으면 전기를 거부한다", func(t *testing.T) {
		voucherRepo, _, _, svc := newVoucherServiceWithLedger()
		ctx := context.Background()

		voucher := newTestVoucher(companyID)
		voucher.Status = domain.VoucherStatusApproved
		voucher.Entries = nil
		voucher.TotalDebit = 0
		voucher.TotalCredit = 0

		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(voucher, nil).Once()

		assert.ErrorIs(t, svc.Post(ctx, companyID, voucher.ID, userID), domain.ErrVoucherNoEntries)
	})
}

// ============================================================================
// AddEntry: one transaction, balance enforced before the write
// ============================================================================

func TestVoucherService_AddEntry(t *testing.T) {
	companyID := newTestCompanyID()

	t.Run("전표를 불균형하게 만드는 행은 저장되지 않는다", func(t *testing.T) {
		voucherRepo, accountRepo, _, svc := newVoucherServiceWithLedger()
		ctx := context.Background()

		voucher := newTestVoucher(companyID) // 1000 / 1000, balanced
		accountID := uuid.New()

		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(voucher, nil).Once()
		accountRepo.On("FindByID", ctx, companyID, accountID).
			Return(newTestAccount(companyID, accountID), nil).Once()
		voucherRepo.On("WithTransaction", ctx, mock.AnythingOfType("func(repository.VoucherRepository) error")).
			Return(nil).Once()
		voucherRepo.On("CreateEntry", ctx, mock.AnythingOfType("*domain.VoucherEntry")).Return(nil).Once()

		entry := &domain.VoucherEntry{
			CompanyID:   companyID,
			AccountID:   accountID,
			DebitAmount: 50,
		}

		err := svc.AddEntry(ctx, voucher.ID, entry)

		// The line insert and the header update share one transaction, so the
		// rejected header rolls the line back with it. Previously the line was
		// already committed and stayed behind forever.
		assert.ErrorIs(t, err, domain.ErrVoucherUnbalanced)
		voucherRepo.AssertNotCalled(t, "Update", ctx, mock.Anything)
	})

	t.Run("균형을 유지하는 행 한 쌍은 저장된다", func(t *testing.T) {
		voucherRepo, accountRepo, _, svc := newVoucherServiceWithLedger()
		ctx := context.Background()

		// A draft with a single credit line that the new debit line balances.
		voucher := newTestVoucher(companyID)
		voucher.Entries = voucher.Entries[1:] // credit 1000 only
		voucher.CalculateTotals()

		accountID := uuid.New()
		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(voucher, nil).Once()
		accountRepo.On("FindByID", ctx, companyID, accountID).
			Return(newTestAccount(companyID, accountID), nil).Once()
		voucherRepo.On("WithTransaction", ctx, mock.AnythingOfType("func(repository.VoucherRepository) error")).
			Return(nil).Once()
		voucherRepo.On("CreateEntry", ctx, mock.AnythingOfType("*domain.VoucherEntry")).Return(nil).Once()
		voucherRepo.On("Update", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		entry := &domain.VoucherEntry{
			CompanyID:   companyID,
			AccountID:   accountID,
			DebitAmount: 1000,
		}

		require.NoError(t, svc.AddEntry(ctx, voucher.ID, entry))
		assert.Equal(t, 1000.0, voucher.TotalDebit)
		assert.Equal(t, 1000.0, voucher.TotalCredit)

		voucherRepo.AssertExpectations(t)
	})

	t.Run("전기된 전표에는 행을 추가할 수 없다", func(t *testing.T) {
		voucherRepo, _, _, svc := newVoucherServiceWithLedger()
		ctx := context.Background()

		voucher := newTestVoucher(companyID)
		voucher.Status = domain.VoucherStatusPosted

		voucherRepo.On("FindByID", ctx, companyID, voucher.ID).Return(voucher, nil).Once()

		entry := &domain.VoucherEntry{CompanyID: companyID, AccountID: uuid.New(), DebitAmount: 100}
		assert.ErrorIs(t, svc.AddEntry(ctx, voucher.ID, entry), domain.ErrVoucherCannotEdit)
	})
}

// ============================================================================
// Duplicate reversal guard
// ============================================================================

func TestVoucherService_Reverse_MarksOriginal(t *testing.T) {
	companyID := newTestCompanyID()
	userID := newTestUserID()

	t.Run("리포지토리가 중복 역분개를 막으면 그 에러가 전달된다", func(t *testing.T) {
		voucherRepo, accountRepo, _, svc := newVoucherServiceWithLedger()
		ctx := context.Background()

		original := newTestVoucher(companyID)
		original.Status = domain.VoucherStatusPosted

		voucherRepo.On("FindByID", ctx, companyID, original.ID).Return(original, nil).Once()
		for _, entry := range original.Entries {
			accountRepo.On("FindByID", ctx, companyID, entry.AccountID).
				Return(newTestAccount(companyID, entry.AccountID), nil).Once()
		}
		voucherRepo.On("GenerateVoucherNo", ctx, companyID, original.VoucherType, mock.AnythingOfType("time.Time")).
			Return("GEN-2026-0002", nil).Once()
		voucherRepo.On("WithTransaction", ctx, mock.AnythingOfType("func(repository.VoucherRepository) error")).
			Return(nil).Once()
		voucherRepo.On("Create", ctx, mock.AnythingOfType("*domain.Voucher")).Return(nil).Once()

		// MarkReversed updates conditionally on reversed_by_id IS NULL, so a
		// second reversal of the same voucher loses the race and is rejected.
		voucherRepo.On("MarkReversed", ctx, companyID, original.ID, mock.AnythingOfType("uuid.UUID")).
			Return(domain.ErrVoucherAlreadyReversed).Once()

		_, err := svc.Reverse(ctx, companyID, original.ID, userID, time.Now(), "중복 역분개")
		assert.ErrorIs(t, err, domain.ErrVoucherAlreadyReversed)

		voucherRepo.AssertExpectations(t)
	})
}
