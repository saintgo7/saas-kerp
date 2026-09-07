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

func newTestLedgerService() (*mocks.MockLedgerRepository, *mocks.MockAccountRepository, service.LedgerService) {
	ledgerRepo := new(mocks.MockLedgerRepository)
	accountRepo := new(mocks.MockAccountRepository)
	return ledgerRepo, accountRepo, service.NewLedgerService(ledgerRepo, accountRepo)
}

// balanceOf builds a December closing balance for an account of the given
// type, stated on its normal balance side.
func balanceOf(companyID, accountID uuid.UUID, accountType domain.AccountType, amount float64) domain.LedgerBalance {
	nature := domain.AccountNatureDebit
	if accountType == domain.AccountTypeLiability ||
		accountType == domain.AccountTypeEquity ||
		accountType == domain.AccountTypeRevenue {
		nature = domain.AccountNatureCredit
	}

	balance := domain.LedgerBalance{
		CompanyID:   companyID,
		AccountID:   accountID,
		FiscalYear:  2026,
		FiscalMonth: 12,
		Account: &domain.Account{
			TenantModel:   domain.TenantModel{BaseModel: domain.BaseModel{ID: accountID}, CompanyID: companyID},
			AccountType:   accountType,
			AccountNature: nature,
		},
	}
	if nature == domain.AccountNatureDebit {
		balance.ClosingDebit = amount
	} else {
		balance.ClosingCredit = amount
	}
	return balance
}

func findBalance(balances []domain.LedgerBalance, accountID uuid.UUID) *domain.LedgerBalance {
	for i := range balances {
		if balances[i].AccountID == accountID {
			return &balances[i]
		}
	}
	return nil
}

// ============================================================================
// Year-end close: net income sign
// ============================================================================

func TestLedgerService_PerformYearEndClose_NetIncomeSign(t *testing.T) {
	companyID := newTestCompanyID()
	userID := newTestUserID()
	retainedEarningsID := uuid.New()

	t.Run("이익이면 이월이익잉여금에 대변으로 이월된다", func(t *testing.T) {
		ledgerRepo, accountRepo, svc := newTestLedgerService()
		ctx := context.Background()

		// 수익 1,000,000 / 비용 600,000 -> 당기순이익 +400,000.
		//
		// GetClosingBalance는 차변-대변이므로 수익은 -1,000,000으로 나온다.
		// 예전 코드는 그 값을 그대로 (수익 - 비용)에 넣어 -1,600,000, 즉
		// 손실로 계산하고 차변으로 이월했다.
		balances := []domain.LedgerBalance{
			balanceOf(companyID, uuid.New(), domain.AccountTypeRevenue, 1_000_000),
			balanceOf(companyID, uuid.New(), domain.AccountTypeExpense, 600_000),
			balanceOf(companyID, retainedEarningsID, domain.AccountTypeEquity, 0),
		}

		ledgerRepo.On("GetBalances", ctx, companyID, 2026, 12).Return(balances, nil).Once()

		var saved []domain.LedgerBalance
		ledgerRepo.On("UpsertBalances", ctx, mock.AnythingOfType("[]domain.LedgerBalance")).
			Run(func(args mock.Arguments) {
				saved = args.Get(1).([]domain.LedgerBalance)
			}).Return(nil).Once()

		err := svc.PerformYearEndClose(ctx, companyID, 2026, retainedEarningsID, userID)
		require.NoError(t, err)

		re := findBalance(saved, retainedEarningsID)
		require.NotNil(t, re, "이월이익잉여금 행이 만들어져야 한다")

		assert.Equal(t, 400_000.0, re.OpeningCredit, "당기순이익은 대변으로 이월된다")
		assert.Equal(t, 0.0, re.OpeningDebit)
		assert.Equal(t, 400_000.0, re.ClosingCredit)
		assert.Equal(t, 2027, re.FiscalYear)
		assert.Equal(t, 1, re.FiscalMonth)

		ledgerRepo.AssertExpectations(t)
		accountRepo.AssertExpectations(t)
	})

	t.Run("손실이면 차변으로 이월된다", func(t *testing.T) {
		ledgerRepo, _, svc := newTestLedgerService()
		ctx := context.Background()

		// 수익 600,000 / 비용 1,000,000 -> 당기순손실 -400,000.
		balances := []domain.LedgerBalance{
			balanceOf(companyID, uuid.New(), domain.AccountTypeRevenue, 600_000),
			balanceOf(companyID, uuid.New(), domain.AccountTypeExpense, 1_000_000),
			balanceOf(companyID, retainedEarningsID, domain.AccountTypeEquity, 0),
		}

		ledgerRepo.On("GetBalances", ctx, companyID, 2026, 12).Return(balances, nil).Once()

		var saved []domain.LedgerBalance
		ledgerRepo.On("UpsertBalances", ctx, mock.AnythingOfType("[]domain.LedgerBalance")).
			Run(func(args mock.Arguments) {
				saved = args.Get(1).([]domain.LedgerBalance)
			}).Return(nil).Once()

		require.NoError(t, svc.PerformYearEndClose(ctx, companyID, 2026, retainedEarningsID, userID))

		re := findBalance(saved, retainedEarningsID)
		require.NotNil(t, re)
		assert.Equal(t, 400_000.0, re.OpeningDebit, "당기순손실은 차변으로 이월된다")
		assert.Equal(t, 0.0, re.OpeningCredit)
	})

	t.Run("이미 잉여금이 있으면 당기순이익이 그 위에 더해진다", func(t *testing.T) {
		ledgerRepo, _, svc := newTestLedgerService()
		ctx := context.Background()

		balances := []domain.LedgerBalance{
			balanceOf(companyID, uuid.New(), domain.AccountTypeRevenue, 1_000_000),
			balanceOf(companyID, uuid.New(), domain.AccountTypeExpense, 600_000),
			balanceOf(companyID, retainedEarningsID, domain.AccountTypeEquity, 2_000_000),
		}

		ledgerRepo.On("GetBalances", ctx, companyID, 2026, 12).Return(balances, nil).Once()

		var saved []domain.LedgerBalance
		ledgerRepo.On("UpsertBalances", ctx, mock.AnythingOfType("[]domain.LedgerBalance")).
			Run(func(args mock.Arguments) {
				saved = args.Get(1).([]domain.LedgerBalance)
			}).Return(nil).Once()

		require.NoError(t, svc.PerformYearEndClose(ctx, companyID, 2026, retainedEarningsID, userID))

		re := findBalance(saved, retainedEarningsID)
		require.NotNil(t, re)
		assert.Equal(t, 2_400_000.0, re.OpeningCredit)
	})

	t.Run("손익계정은 다음 연도로 이월되지 않는다", func(t *testing.T) {
		ledgerRepo, _, svc := newTestLedgerService()
		ctx := context.Background()

		revenueID := uuid.New()
		expenseID := uuid.New()
		assetID := uuid.New()

		balances := []domain.LedgerBalance{
			balanceOf(companyID, revenueID, domain.AccountTypeRevenue, 1_000_000),
			balanceOf(companyID, expenseID, domain.AccountTypeExpense, 600_000),
			balanceOf(companyID, assetID, domain.AccountTypeAsset, 5_000_000),
			balanceOf(companyID, retainedEarningsID, domain.AccountTypeEquity, 0),
		}

		ledgerRepo.On("GetBalances", ctx, companyID, 2026, 12).Return(balances, nil).Once()

		var saved []domain.LedgerBalance
		ledgerRepo.On("UpsertBalances", ctx, mock.AnythingOfType("[]domain.LedgerBalance")).
			Run(func(args mock.Arguments) {
				saved = args.Get(1).([]domain.LedgerBalance)
			}).Return(nil).Once()

		require.NoError(t, svc.PerformYearEndClose(ctx, companyID, 2026, retainedEarningsID, userID))

		assert.Nil(t, findBalance(saved, revenueID), "수익 계정은 새 회계연도에 0에서 시작한다")
		assert.Nil(t, findBalance(saved, expenseID), "비용 계정은 새 회계연도에 0에서 시작한다")

		asset := findBalance(saved, assetID)
		require.NotNil(t, asset, "자산 계정은 이월된다")
		assert.Equal(t, 5_000_000.0, asset.OpeningDebit)
	})

	t.Run("이월이익잉여금 계정에 12월 잔액이 없어도 순이익이 사라지지 않는다", func(t *testing.T) {
		ledgerRepo, _, svc := newTestLedgerService()
		ctx := context.Background()

		// 첫 사업연도라 이월이익잉여금 행이 아직 없는 경우.
		balances := []domain.LedgerBalance{
			balanceOf(companyID, uuid.New(), domain.AccountTypeRevenue, 1_000_000),
			balanceOf(companyID, uuid.New(), domain.AccountTypeExpense, 600_000),
		}

		ledgerRepo.On("GetBalances", ctx, companyID, 2026, 12).Return(balances, nil).Once()

		var saved []domain.LedgerBalance
		ledgerRepo.On("UpsertBalances", ctx, mock.AnythingOfType("[]domain.LedgerBalance")).
			Run(func(args mock.Arguments) {
				saved = args.Get(1).([]domain.LedgerBalance)
			}).Return(nil).Once()

		require.NoError(t, svc.PerformYearEndClose(ctx, companyID, 2026, retainedEarningsID, userID))

		re := findBalance(saved, retainedEarningsID)
		require.NotNil(t, re, "잔액 행이 없어도 이익잉여금 행을 만들어야 한다")
		assert.Equal(t, 400_000.0, re.OpeningCredit)
	})

	t.Run("계정 정보가 없으면 추측하지 않고 실패한다", func(t *testing.T) {
		ledgerRepo, _, svc := newTestLedgerService()
		ctx := context.Background()

		orphan := balanceOf(companyID, uuid.New(), domain.AccountTypeRevenue, 1_000_000)
		orphan.Account = nil

		ledgerRepo.On("GetBalances", ctx, companyID, 2026, 12).
			Return([]domain.LedgerBalance{orphan}, nil).Once()

		err := svc.PerformYearEndClose(ctx, companyID, 2026, retainedEarningsID, userID)
		assert.Error(t, err, "분류할 수 없는 잔액을 조용히 건너뛰면 순이익이 틀어진다")
	})
}

// ============================================================================
// Account ledger opening balance: previous MONTH, not previous day
// ============================================================================

func TestLedgerService_GetAccountLedger_OpeningBalancePeriod(t *testing.T) {
	companyID := newTestCompanyID()
	accountID := uuid.New()

	t.Run("월 중간부터 조회해도 전월 잔액을 읽는다", func(t *testing.T) {
		ledgerRepo, _, svc := newTestLedgerService()
		ctx := context.Background()

		from := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

		// from.AddDate(0, 0, -1)은 2026-03-14, 즉 같은 3월이라 3월 자신의
		// 잔액을 기초잔액으로 읽어 3월 발생액이 두 번 반영됐다.
		ledgerRepo.On("GetBalance", ctx, companyID, accountID, 2026, 2).
			Return(&domain.LedgerBalance{ClosingDebit: 1_000_000}, nil).Once()
		ledgerRepo.On("GetAccountLedger", ctx, companyID, accountID, from, to).
			Return([]domain.AccountLedgerEntry{{DebitAmount: 500_000, Balance: 500_000}}, nil).Once()

		entries, opening, err := svc.GetAccountLedger(ctx, companyID, accountID, from, to)
		require.NoError(t, err)

		assert.Equal(t, 1_000_000.0, opening)
		require.Len(t, entries, 1)
		assert.Equal(t, 1_500_000.0, entries[0].Balance)

		ledgerRepo.AssertExpectations(t)
	})

	t.Run("1월 조회는 전년 12월을 읽는다", func(t *testing.T) {
		ledgerRepo, _, svc := newTestLedgerService()
		ctx := context.Background()

		from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

		ledgerRepo.On("GetBalance", ctx, companyID, accountID, 2025, 12).
			Return(&domain.LedgerBalance{ClosingDebit: 700_000}, nil).Once()
		ledgerRepo.On("GetAccountLedger", ctx, companyID, accountID, from, to).
			Return([]domain.AccountLedgerEntry{}, nil).Once()

		_, opening, err := svc.GetAccountLedger(ctx, companyID, accountID, from, to)
		require.NoError(t, err)
		assert.Equal(t, 700_000.0, opening)

		ledgerRepo.AssertExpectations(t)
	})

	t.Run("전월 잔액이 없으면 0에서 시작한다", func(t *testing.T) {
		ledgerRepo, _, svc := newTestLedgerService()
		ctx := context.Background()

		from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

		ledgerRepo.On("GetBalance", ctx, companyID, accountID, 2026, 2).
			Return(nil, domain.ErrLedgerBalanceNotFound).Once()
		ledgerRepo.On("GetAccountLedger", ctx, companyID, accountID, from, to).
			Return([]domain.AccountLedgerEntry{}, nil).Once()

		_, opening, err := svc.GetAccountLedger(ctx, companyID, accountID, from, to)
		require.NoError(t, err)
		assert.Equal(t, 0.0, opening)
	})

	t.Run("조회 실패는 0으로 넘기지 않고 전파한다", func(t *testing.T) {
		ledgerRepo, _, svc := newTestLedgerService()
		ctx := context.Background()

		from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

		ledgerRepo.On("GetBalance", ctx, companyID, accountID, 2026, 2).
			Return(nil, assert.AnError).Once()

		_, _, err := svc.GetAccountLedger(ctx, companyID, accountID, from, to)
		assert.ErrorIs(t, err, assert.AnError)
	})
}
