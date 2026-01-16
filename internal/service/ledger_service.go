package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// LedgerService defines the interface for ledger business logic
type LedgerService interface {
	// Balance operations
	GetAccountBalance(ctx context.Context, companyID, accountID uuid.UUID, year, month int) (*domain.LedgerBalance, error)
	GetPeriodBalances(ctx context.Context, companyID uuid.UUID, year, month int) ([]domain.LedgerBalance, error)
	RecalculateBalances(ctx context.Context, companyID uuid.UUID, year, month int) error

	// Account ledger (detailed transactions)
	GetAccountLedger(ctx context.Context, companyID, accountID uuid.UUID, from, to time.Time) ([]domain.AccountLedgerEntry, float64, error)

	// Trial balance
	GetTrialBalance(ctx context.Context, companyID uuid.UUID, year, month int) (*domain.TrialBalance, error)
	GetTrialBalanceRange(ctx context.Context, companyID uuid.UUID, fromYear, fromMonth, toYear, toMonth int) (*domain.TrialBalance, error)

	// Fiscal period management
	GetFiscalPeriod(ctx context.Context, companyID uuid.UUID, year, month int) (*domain.FiscalPeriod, error)
	GetFiscalPeriods(ctx context.Context, companyID uuid.UUID, year int) ([]domain.FiscalPeriod, error)
	CreateFiscalPeriods(ctx context.Context, companyID uuid.UUID, year int) ([]domain.FiscalPeriod, error)
	ClosePeriod(ctx context.Context, companyID uuid.UUID, year, month int, userID uuid.UUID) error
	ReopenPeriod(ctx context.Context, companyID uuid.UUID, year, month int) error

	// Year-end closing
	PerformYearEndClose(ctx context.Context, companyID uuid.UUID, year int, retainedEarningsAccountID uuid.UUID, userID uuid.UUID) error
}

// ledgerService implements LedgerService
type ledgerService struct {
	ledgerRepo  repository.LedgerRepository
	accountRepo repository.AccountRepository
}

// NewLedgerService creates a new LedgerService
func NewLedgerService(ledgerRepo repository.LedgerRepository, accountRepo repository.AccountRepository) LedgerService {
	return &ledgerService{
		ledgerRepo:  ledgerRepo,
		accountRepo: accountRepo,
	}
}

// GetAccountBalance retrieves a single account balance
func (s *ledgerService) GetAccountBalance(ctx context.Context, companyID, accountID uuid.UUID, year, month int) (*domain.LedgerBalance, error) {
	return s.ledgerRepo.GetBalance(ctx, companyID, accountID, year, month)
}

// GetPeriodBalances retrieves all balances for a period
func (s *ledgerService) GetPeriodBalances(ctx context.Context, companyID uuid.UUID, year, month int) ([]domain.LedgerBalance, error) {
	return s.ledgerRepo.GetBalances(ctx, companyID, year, month)
}

// RecalculateBalances recalculates balances from posted vouchers
func (s *ledgerService) RecalculateBalances(ctx context.Context, companyID uuid.UUID, year, month int) error {
	// Calculate balances from vouchers
	balances, err := s.ledgerRepo.CalculatePeriodBalances(ctx, companyID, year, month)
	if err != nil {
		return err
	}

	// Save balances
	return s.ledgerRepo.UpsertBalances(ctx, balances)
}

// GetAccountLedger retrieves detailed ledger entries with opening balance
func (s *ledgerService) GetAccountLedger(ctx context.Context, companyID, accountID uuid.UUID, from, to time.Time) ([]domain.AccountLedgerEntry, float64, error) {
	// Get opening balance
	openingBalance := 0.0

	// Find the period before the from date
	prevMonth := from.AddDate(0, 0, -1)
	prevYear := prevMonth.Year()
	prevMonthNum := int(prevMonth.Month())

	balance, err := s.ledgerRepo.GetBalance(ctx, companyID, accountID, prevYear, prevMonthNum)
	if err == nil {
		openingBalance = balance.GetClosingBalance()
	}

	// Get entries
	entries, err := s.ledgerRepo.GetAccountLedger(ctx, companyID, accountID, from, to)
	if err != nil {
		return nil, 0, err
	}

	// Adjust running balance to include opening
	for i := range entries {
		entries[i].Balance += openingBalance
	}

	return entries, openingBalance, nil
}

// GetTrialBalance generates a trial balance report
func (s *ledgerService) GetTrialBalance(ctx context.Context, companyID uuid.UUID, year, month int) (*domain.TrialBalance, error) {
	return s.ledgerRepo.GetTrialBalance(ctx, companyID, year, month)
}

// GetTrialBalanceRange generates a trial balance for a date range
func (s *ledgerService) GetTrialBalanceRange(ctx context.Context, companyID uuid.UUID, fromYear, fromMonth, toYear, toMonth int) (*domain.TrialBalance, error) {
	return s.ledgerRepo.GetTrialBalanceRange(ctx, companyID, fromYear, fromMonth, toYear, toMonth)
}

// GetFiscalPeriod retrieves a fiscal period
func (s *ledgerService) GetFiscalPeriod(ctx context.Context, companyID uuid.UUID, year, month int) (*domain.FiscalPeriod, error) {
	return s.ledgerRepo.GetFiscalPeriod(ctx, companyID, year, month)
}

// GetFiscalPeriods retrieves all fiscal periods for a year
func (s *ledgerService) GetFiscalPeriods(ctx context.Context, companyID uuid.UUID, year int) ([]domain.FiscalPeriod, error) {
	return s.ledgerRepo.GetFiscalPeriods(ctx, companyID, year)
}

// CreateFiscalPeriods creates all 12 fiscal periods for a year
func (s *ledgerService) CreateFiscalPeriods(ctx context.Context, companyID uuid.UUID, year int) ([]domain.FiscalPeriod, error) {
	var periods []domain.FiscalPeriod

	for month := 1; month <= 12; month++ {
		startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(0, 1, -1)

		period := domain.FiscalPeriod{
			CompanyID:   companyID,
			FiscalYear:  year,
			FiscalMonth: month,
			PeriodName:  startDate.Format("2006-01"),
			StartDate:   startDate,
			EndDate:     endDate,
			Status:      domain.FiscalPeriodOpen,
		}

		if err := s.ledgerRepo.CreateFiscalPeriod(ctx, &period); err != nil {
			return nil, err
		}

		periods = append(periods, period)
	}

	return periods, nil
}

// ClosePeriod closes a fiscal period
func (s *ledgerService) ClosePeriod(ctx context.Context, companyID uuid.UUID, year, month int, userID uuid.UUID) error {
	// Get period
	period, err := s.ledgerRepo.GetFiscalPeriod(ctx, companyID, year, month)
	if err != nil {
		return err
	}

	// Recalculate balances before closing
	if err := s.RecalculateBalances(ctx, companyID, year, month); err != nil {
		return err
	}

	// Carry forward to next period
	nextYear, nextMonth := year, month+1
	if nextMonth > 12 {
		nextYear++
		nextMonth = 1
	}
	if err := s.ledgerRepo.CarryForwardBalances(ctx, companyID, year, month, nextYear, nextMonth); err != nil {
		return err
	}

	// Close period
	if err := period.Close(userID); err != nil {
		return err
	}

	return s.ledgerRepo.UpdateFiscalPeriod(ctx, period)
}

// ReopenPeriod reopens a closed fiscal period
func (s *ledgerService) ReopenPeriod(ctx context.Context, companyID uuid.UUID, year, month int) error {
	period, err := s.ledgerRepo.GetFiscalPeriod(ctx, companyID, year, month)
	if err != nil {
		return err
	}

	if period.Status == domain.FiscalPeriodLocked {
		return domain.ErrFiscalPeriodClosed
	}

	period.Status = domain.FiscalPeriodOpen
	period.ClosedAt = nil
	period.ClosedBy = nil

	return s.ledgerRepo.UpdateFiscalPeriod(ctx, period)
}

// PerformYearEndClose performs year-end closing
func (s *ledgerService) PerformYearEndClose(ctx context.Context, companyID uuid.UUID, year int, retainedEarningsAccountID uuid.UUID, userID uuid.UUID) error {
	// This would:
	// 1. Close all periods for the year
	// 2. Calculate net income (Revenue - Expense)
	// 3. Create closing entries to zero out income/expense accounts
	// 4. Transfer net income to retained earnings
	// 5. Carry forward balance sheet accounts to next year

	// Get December balances
	balances, err := s.ledgerRepo.GetBalances(ctx, companyID, year, 12)
	if err != nil {
		return err
	}

	// Calculate net income from revenue and expense accounts
	var totalRevenue, totalExpense float64
	for _, balance := range balances {
		if balance.Account != nil {
			switch balance.Account.AccountType {
			case domain.AccountTypeRevenue:
				totalRevenue += balance.GetClosingBalance()
			case domain.AccountTypeExpense:
				totalExpense += balance.GetClosingBalance()
			}
		}
	}

	netIncome := totalRevenue - totalExpense

	// Create closing entry balances for next year
	nextYear := year + 1
	var nextYearBalances []domain.LedgerBalance

	for _, balance := range balances {
		if balance.Account == nil {
			continue
		}

		// For balance sheet accounts (Asset, Liability, Equity), carry forward
		if balance.Account.AccountType == domain.AccountTypeAsset ||
			balance.Account.AccountType == domain.AccountTypeLiability ||
			balance.Account.AccountType == domain.AccountTypeEquity {

			nextBalance := domain.LedgerBalance{
				CompanyID:     companyID,
				AccountID:     balance.AccountID,
				FiscalYear:    nextYear,
				FiscalMonth:   1,
				OpeningDebit:  balance.ClosingDebit,
				OpeningCredit: balance.ClosingCredit,
				ClosingDebit:  balance.ClosingDebit,
				ClosingCredit: balance.ClosingCredit,
			}

			// Add net income to retained earnings
			if balance.AccountID == retainedEarningsAccountID {
				if netIncome > 0 {
					nextBalance.OpeningCredit += netIncome
					nextBalance.ClosingCredit += netIncome
				} else {
					nextBalance.OpeningDebit += -netIncome
					nextBalance.ClosingDebit += -netIncome
				}
			}

			nextYearBalances = append(nextYearBalances, nextBalance)
		}
		// Revenue and Expense accounts start fresh (zero balance)
	}

	return s.ledgerRepo.UpsertBalances(ctx, nextYearBalances)
}
