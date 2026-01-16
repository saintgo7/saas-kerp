package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// MockVoucherService is a mock implementation of service.VoucherService
type MockVoucherService struct {
	mock.Mock
}

// Create mocks the Create method
func (m *MockVoucherService) Create(ctx context.Context, voucher *domain.Voucher) error {
	args := m.Called(ctx, voucher)
	return args.Error(0)
}

// Update mocks the Update method
func (m *MockVoucherService) Update(ctx context.Context, voucher *domain.Voucher) error {
	args := m.Called(ctx, voucher)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *MockVoucherService) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	args := m.Called(ctx, companyID, id)
	return args.Error(0)
}

// GetByID mocks the GetByID method
func (m *MockVoucherService) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Voucher, error) {
	args := m.Called(ctx, companyID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Voucher), args.Error(1)
}

// GetByNo mocks the GetByNo method
func (m *MockVoucherService) GetByNo(ctx context.Context, companyID uuid.UUID, voucherNo string) (*domain.Voucher, error) {
	args := m.Called(ctx, companyID, voucherNo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Voucher), args.Error(1)
}

// List mocks the List method
func (m *MockVoucherService) List(ctx context.Context, filter repository.VoucherFilter) ([]domain.Voucher, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]domain.Voucher), args.Get(1).(int64), args.Error(2)
}

// GetByDateRange mocks the GetByDateRange method
func (m *MockVoucherService) GetByDateRange(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]domain.Voucher, error) {
	args := m.Called(ctx, companyID, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Voucher), args.Error(1)
}

// GetPending mocks the GetPending method
func (m *MockVoucherService) GetPending(ctx context.Context, companyID uuid.UUID) ([]domain.Voucher, error) {
	args := m.Called(ctx, companyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Voucher), args.Error(1)
}

// AddEntry mocks the AddEntry method
func (m *MockVoucherService) AddEntry(ctx context.Context, voucherID uuid.UUID, entry *domain.VoucherEntry) error {
	args := m.Called(ctx, voucherID, entry)
	return args.Error(0)
}

// UpdateEntry mocks the UpdateEntry method
func (m *MockVoucherService) UpdateEntry(ctx context.Context, entry *domain.VoucherEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

// RemoveEntry mocks the RemoveEntry method
func (m *MockVoucherService) RemoveEntry(ctx context.Context, entryID uuid.UUID) error {
	args := m.Called(ctx, entryID)
	return args.Error(0)
}

// ReplaceEntries mocks the ReplaceEntries method
func (m *MockVoucherService) ReplaceEntries(ctx context.Context, voucherID uuid.UUID, entries []domain.VoucherEntry) error {
	args := m.Called(ctx, voucherID, entries)
	return args.Error(0)
}

// Submit mocks the Submit method
func (m *MockVoucherService) Submit(ctx context.Context, companyID, voucherID, userID uuid.UUID) error {
	args := m.Called(ctx, companyID, voucherID, userID)
	return args.Error(0)
}

// Approve mocks the Approve method
func (m *MockVoucherService) Approve(ctx context.Context, companyID, voucherID, userID uuid.UUID) error {
	args := m.Called(ctx, companyID, voucherID, userID)
	return args.Error(0)
}

// Reject mocks the Reject method
func (m *MockVoucherService) Reject(ctx context.Context, companyID, voucherID, userID uuid.UUID, reason string) error {
	args := m.Called(ctx, companyID, voucherID, userID, reason)
	return args.Error(0)
}

// Post mocks the Post method
func (m *MockVoucherService) Post(ctx context.Context, companyID, voucherID, userID uuid.UUID) error {
	args := m.Called(ctx, companyID, voucherID, userID)
	return args.Error(0)
}

// Cancel mocks the Cancel method
func (m *MockVoucherService) Cancel(ctx context.Context, companyID, voucherID uuid.UUID) error {
	args := m.Called(ctx, companyID, voucherID)
	return args.Error(0)
}

// Reverse mocks the Reverse method
func (m *MockVoucherService) Reverse(ctx context.Context, companyID, voucherID, userID uuid.UUID, reversalDate time.Time, description string) (*domain.Voucher, error) {
	args := m.Called(ctx, companyID, voucherID, userID, reversalDate, description)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Voucher), args.Error(1)
}

// ValidateEntries mocks the ValidateEntries method
func (m *MockVoucherService) ValidateEntries(ctx context.Context, companyID uuid.UUID, entries []domain.VoucherEntry) error {
	args := m.Called(ctx, companyID, entries)
	return args.Error(0)
}

// Ensure MockVoucherService implements service.VoucherService
var _ service.VoucherService = (*MockVoucherService)(nil)
