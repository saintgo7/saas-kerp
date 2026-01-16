package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// MockVoucherRepository is a mock implementation of VoucherRepository
type MockVoucherRepository struct {
	mock.Mock
}

// Create mocks the Create method
func (m *MockVoucherRepository) Create(ctx context.Context, voucher *domain.Voucher) error {
	args := m.Called(ctx, voucher)
	return args.Error(0)
}

// Update mocks the Update method
func (m *MockVoucherRepository) Update(ctx context.Context, voucher *domain.Voucher) error {
	args := m.Called(ctx, voucher)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *MockVoucherRepository) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	args := m.Called(ctx, companyID, id)
	return args.Error(0)
}

// FindByID mocks the FindByID method
func (m *MockVoucherRepository) FindByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Voucher, error) {
	args := m.Called(ctx, companyID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Voucher), args.Error(1)
}

// FindByNo mocks the FindByNo method
func (m *MockVoucherRepository) FindByNo(ctx context.Context, companyID uuid.UUID, voucherNo string) (*domain.Voucher, error) {
	args := m.Called(ctx, companyID, voucherNo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Voucher), args.Error(1)
}

// FindAll mocks the FindAll method
func (m *MockVoucherRepository) FindAll(ctx context.Context, filter repository.VoucherFilter) ([]domain.Voucher, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]domain.Voucher), args.Get(1).(int64), args.Error(2)
}

// FindByDateRange mocks the FindByDateRange method
func (m *MockVoucherRepository) FindByDateRange(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]domain.Voucher, error) {
	args := m.Called(ctx, companyID, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Voucher), args.Error(1)
}

// FindByStatus mocks the FindByStatus method
func (m *MockVoucherRepository) FindByStatus(ctx context.Context, companyID uuid.UUID, status domain.VoucherStatus) ([]domain.Voucher, error) {
	args := m.Called(ctx, companyID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Voucher), args.Error(1)
}

// CreateEntry mocks the CreateEntry method
func (m *MockVoucherRepository) CreateEntry(ctx context.Context, entry *domain.VoucherEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

// UpdateEntry mocks the UpdateEntry method
func (m *MockVoucherRepository) UpdateEntry(ctx context.Context, entry *domain.VoucherEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

// DeleteEntry mocks the DeleteEntry method
func (m *MockVoucherRepository) DeleteEntry(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// DeleteEntriesByVoucher mocks the DeleteEntriesByVoucher method
func (m *MockVoucherRepository) DeleteEntriesByVoucher(ctx context.Context, voucherID uuid.UUID) error {
	args := m.Called(ctx, voucherID)
	return args.Error(0)
}

// FindEntriesByVoucher mocks the FindEntriesByVoucher method
func (m *MockVoucherRepository) FindEntriesByVoucher(ctx context.Context, voucherID uuid.UUID) ([]domain.VoucherEntry, error) {
	args := m.Called(ctx, voucherID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.VoucherEntry), args.Error(1)
}

// FindEntriesByAccount mocks the FindEntriesByAccount method
func (m *MockVoucherRepository) FindEntriesByAccount(ctx context.Context, companyID, accountID uuid.UUID, from, to time.Time) ([]domain.VoucherEntry, error) {
	args := m.Called(ctx, companyID, accountID, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.VoucherEntry), args.Error(1)
}

// UpdateStatus mocks the UpdateStatus method
func (m *MockVoucherRepository) UpdateStatus(ctx context.Context, voucher *domain.Voucher) error {
	args := m.Called(ctx, voucher)
	return args.Error(0)
}

// GenerateVoucherNo mocks the GenerateVoucherNo method
func (m *MockVoucherRepository) GenerateVoucherNo(ctx context.Context, companyID uuid.UUID, voucherType domain.VoucherType, voucherDate time.Time) (string, error) {
	args := m.Called(ctx, companyID, voucherType, voucherDate)
	return args.String(0), args.Error(1)
}

// WithTransaction mocks the WithTransaction method
func (m *MockVoucherRepository) WithTransaction(ctx context.Context, fn func(repo repository.VoucherRepository) error) error {
	args := m.Called(ctx, fn)
	// Execute the function with the mock itself
	if err := fn(m); err != nil {
		return err
	}
	return args.Error(0)
}

// Ensure MockVoucherRepository implements VoucherRepository
var _ repository.VoucherRepository = (*MockVoucherRepository)(nil)
