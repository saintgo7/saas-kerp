package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// MockAccountRepository is a mock implementation of AccountRepository
type MockAccountRepository struct {
	mock.Mock
}

// Create mocks the Create method
func (m *MockAccountRepository) Create(ctx context.Context, account *domain.Account) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

// Update mocks the Update method
func (m *MockAccountRepository) Update(ctx context.Context, account *domain.Account) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *MockAccountRepository) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	args := m.Called(ctx, companyID, id)
	return args.Error(0)
}

// FindByID mocks the FindByID method
func (m *MockAccountRepository) FindByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Account, error) {
	args := m.Called(ctx, companyID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}

// FindByCode mocks the FindByCode method
func (m *MockAccountRepository) FindByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Account, error) {
	args := m.Called(ctx, companyID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}

// FindAll mocks the FindAll method
func (m *MockAccountRepository) FindAll(ctx context.Context, filter repository.AccountFilter) ([]domain.Account, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]domain.Account), args.Get(1).(int64), args.Error(2)
}

// FindChildren mocks the FindChildren method
func (m *MockAccountRepository) FindChildren(ctx context.Context, companyID, parentID uuid.UUID) ([]domain.Account, error) {
	args := m.Called(ctx, companyID, parentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Account), args.Error(1)
}

// FindByType mocks the FindByType method
func (m *MockAccountRepository) FindByType(ctx context.Context, companyID uuid.UUID, accountType domain.AccountType) ([]domain.Account, error) {
	args := m.Called(ctx, companyID, accountType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Account), args.Error(1)
}

// GetTree mocks the GetTree method
func (m *MockAccountRepository) GetTree(ctx context.Context, companyID uuid.UUID) ([]domain.Account, error) {
	args := m.Called(ctx, companyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Account), args.Error(1)
}

// GetAncestors mocks the GetAncestors method
func (m *MockAccountRepository) GetAncestors(ctx context.Context, companyID, id uuid.UUID) ([]domain.Account, error) {
	args := m.Called(ctx, companyID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Account), args.Error(1)
}

// GetDescendants mocks the GetDescendants method
func (m *MockAccountRepository) GetDescendants(ctx context.Context, companyID, id uuid.UUID) ([]domain.Account, error) {
	args := m.Called(ctx, companyID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Account), args.Error(1)
}

// UpdatePath mocks the UpdatePath method
func (m *MockAccountRepository) UpdatePath(ctx context.Context, account *domain.Account) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

// ExistsByCode mocks the ExistsByCode method
func (m *MockAccountRepository) ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	args := m.Called(ctx, companyID, code, excludeID)
	return args.Bool(0), args.Error(1)
}

// HasChildren mocks the HasChildren method
func (m *MockAccountRepository) HasChildren(ctx context.Context, companyID, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, companyID, id)
	return args.Bool(0), args.Error(1)
}

// HasVoucherEntries mocks the HasVoucherEntries method
func (m *MockAccountRepository) HasVoucherEntries(ctx context.Context, companyID, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, companyID, id)
	return args.Bool(0), args.Error(1)
}

// CreateBatch mocks the CreateBatch method
func (m *MockAccountRepository) CreateBatch(ctx context.Context, accounts []domain.Account) error {
	args := m.Called(ctx, accounts)
	return args.Error(0)
}

// UpdateSortOrder mocks the UpdateSortOrder method
func (m *MockAccountRepository) UpdateSortOrder(ctx context.Context, companyID uuid.UUID, orders map[uuid.UUID]int) error {
	args := m.Called(ctx, companyID, orders)
	return args.Error(0)
}

// Ensure MockAccountRepository implements AccountRepository
var _ repository.AccountRepository = (*MockAccountRepository)(nil)
