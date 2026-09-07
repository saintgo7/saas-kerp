package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// MockLedgerRepository is a mock implementation of repository.LedgerRepository
type MockLedgerRepository struct {
	mock.Mock
}

// GetBalance mocks the GetBalance method
func (m *MockLedgerRepository) GetBalance(ctx context.Context, companyID, accountID uuid.UUID, year, month int) (*domain.LedgerBalance, error) {
	args := m.Called(ctx, companyID, accountID, year, month)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LedgerBalance), args.Error(1)
}

// GetBalances mocks the GetBalances method
func (m *MockLedgerRepository) GetBalances(ctx context.Context, companyID uuid.UUID, year, month int) ([]domain.LedgerBalance, error) {
	args := m.Called(ctx, companyID, year, month)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.LedgerBalance), args.Error(1)
}

// GetBalancesByType mocks the GetBalancesByType method
func (m *MockLedgerRepository) GetBalancesByType(ctx context.Context, companyID uuid.UUID, year, month int, accountType domain.AccountType) ([]domain.LedgerBalance, error) {
	args := m.Called(ctx, companyID, year, month, accountType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.LedgerBalance), args.Error(1)
}

// UpsertBalance mocks the UpsertBalance method
func (m *MockLedgerRepository) UpsertBalance(ctx context.Context, balance *domain.LedgerBalance) error {
	args := m.Called(ctx, balance)
	return args.Error(0)
}

// UpsertBalances mocks the UpsertBalances method
func (m *MockLedgerRepository) UpsertBalances(ctx context.Context, balances []domain.LedgerBalance) error {
	args := m.Called(ctx, balances)
	return args.Error(0)
}

// CalculatePeriodBalances mocks the CalculatePeriodBalances method
func (m *MockLedgerRepository) CalculatePeriodBalances(ctx context.Context, companyID uuid.UUID, year, month int) ([]domain.LedgerBalance, error) {
	args := m.Called(ctx, companyID, year, month)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.LedgerBalance), args.Error(1)
}

// RecalculateBalances mocks the RecalculateBalances method
func (m *MockLedgerRepository) RecalculateBalances(ctx context.Context, companyID uuid.UUID, fromYear, fromMonth int) error {
	args := m.Called(ctx, companyID, fromYear, fromMonth)
	return args.Error(0)
}

// GetAccountLedger mocks the GetAccountLedger method
func (m *MockLedgerRepository) GetAccountLedger(ctx context.Context, companyID, accountID uuid.UUID, from, to time.Time) ([]domain.AccountLedgerEntry, error) {
	args := m.Called(ctx, companyID, accountID, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.AccountLedgerEntry), args.Error(1)
}

// GetAccountLedgerByPeriod mocks the GetAccountLedgerByPeriod method
func (m *MockLedgerRepository) GetAccountLedgerByPeriod(ctx context.Context, companyID, accountID uuid.UUID, year, month int) ([]domain.AccountLedgerEntry, error) {
	args := m.Called(ctx, companyID, accountID, year, month)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.AccountLedgerEntry), args.Error(1)
}

// GetTrialBalance mocks the GetTrialBalance method
func (m *MockLedgerRepository) GetTrialBalance(ctx context.Context, companyID uuid.UUID, year, month int) (*domain.TrialBalance, error) {
	args := m.Called(ctx, companyID, year, month)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TrialBalance), args.Error(1)
}

// GetTrialBalanceRange mocks the GetTrialBalanceRange method
func (m *MockLedgerRepository) GetTrialBalanceRange(ctx context.Context, companyID uuid.UUID, fromYear, fromMonth, toYear, toMonth int) (*domain.TrialBalance, error) {
	args := m.Called(ctx, companyID, fromYear, fromMonth, toYear, toMonth)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TrialBalance), args.Error(1)
}

// GetFiscalPeriod mocks the GetFiscalPeriod method
func (m *MockLedgerRepository) GetFiscalPeriod(ctx context.Context, companyID uuid.UUID, year, month int) (*domain.FiscalPeriod, error) {
	args := m.Called(ctx, companyID, year, month)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.FiscalPeriod), args.Error(1)
}

// GetFiscalPeriods mocks the GetFiscalPeriods method
func (m *MockLedgerRepository) GetFiscalPeriods(ctx context.Context, companyID uuid.UUID, year int) ([]domain.FiscalPeriod, error) {
	args := m.Called(ctx, companyID, year)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.FiscalPeriod), args.Error(1)
}

// CreateFiscalPeriod mocks the CreateFiscalPeriod method
func (m *MockLedgerRepository) CreateFiscalPeriod(ctx context.Context, period *domain.FiscalPeriod) error {
	args := m.Called(ctx, period)
	return args.Error(0)
}

// UpdateFiscalPeriod mocks the UpdateFiscalPeriod method
func (m *MockLedgerRepository) UpdateFiscalPeriod(ctx context.Context, period *domain.FiscalPeriod) error {
	args := m.Called(ctx, period)
	return args.Error(0)
}

// GetOpenPeriods mocks the GetOpenPeriods method
func (m *MockLedgerRepository) GetOpenPeriods(ctx context.Context, companyID uuid.UUID) ([]domain.FiscalPeriod, error) {
	args := m.Called(ctx, companyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.FiscalPeriod), args.Error(1)
}

// CarryForwardBalances mocks the CarryForwardBalances method
func (m *MockLedgerRepository) CarryForwardBalances(ctx context.Context, companyID uuid.UUID, fromYear, fromMonth, toYear, toMonth int) error {
	args := m.Called(ctx, companyID, fromYear, fromMonth, toYear, toMonth)
	return args.Error(0)
}

// Ensure MockLedgerRepository implements LedgerRepository
var _ repository.LedgerRepository = (*MockLedgerRepository)(nil)
