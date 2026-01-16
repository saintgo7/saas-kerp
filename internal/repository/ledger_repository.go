package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// LedgerRepository defines the interface for ledger data access
type LedgerRepository interface {
	// Ledger balance operations
	GetBalance(ctx context.Context, companyID, accountID uuid.UUID, year, month int) (*domain.LedgerBalance, error)
	GetBalances(ctx context.Context, companyID uuid.UUID, year, month int) ([]domain.LedgerBalance, error)
	GetBalancesByType(ctx context.Context, companyID uuid.UUID, year, month int, accountType domain.AccountType) ([]domain.LedgerBalance, error)
	UpsertBalance(ctx context.Context, balance *domain.LedgerBalance) error
	UpsertBalances(ctx context.Context, balances []domain.LedgerBalance) error

	// Ledger calculation from vouchers
	CalculatePeriodBalances(ctx context.Context, companyID uuid.UUID, year, month int) ([]domain.LedgerBalance, error)
	RecalculateBalances(ctx context.Context, companyID uuid.UUID, fromYear, fromMonth int) error

	// Account ledger (detailed transactions)
	GetAccountLedger(ctx context.Context, companyID, accountID uuid.UUID, from, to time.Time) ([]domain.AccountLedgerEntry, error)
	GetAccountLedgerByPeriod(ctx context.Context, companyID, accountID uuid.UUID, year, month int) ([]domain.AccountLedgerEntry, error)

	// Trial balance
	GetTrialBalance(ctx context.Context, companyID uuid.UUID, year, month int) (*domain.TrialBalance, error)
	GetTrialBalanceRange(ctx context.Context, companyID uuid.UUID, fromYear, fromMonth, toYear, toMonth int) (*domain.TrialBalance, error)

	// Fiscal period operations
	GetFiscalPeriod(ctx context.Context, companyID uuid.UUID, year, month int) (*domain.FiscalPeriod, error)
	GetFiscalPeriods(ctx context.Context, companyID uuid.UUID, year int) ([]domain.FiscalPeriod, error)
	CreateFiscalPeriod(ctx context.Context, period *domain.FiscalPeriod) error
	UpdateFiscalPeriod(ctx context.Context, period *domain.FiscalPeriod) error
	GetOpenPeriods(ctx context.Context, companyID uuid.UUID) ([]domain.FiscalPeriod, error)

	// Carry forward
	CarryForwardBalances(ctx context.Context, companyID uuid.UUID, fromYear, fromMonth, toYear, toMonth int) error
}
