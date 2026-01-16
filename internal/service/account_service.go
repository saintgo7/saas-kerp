package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// AccountService defines the interface for account business logic
type AccountService interface {
	// CRUD operations
	Create(ctx context.Context, account *domain.Account) error
	Update(ctx context.Context, account *domain.Account) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Query operations
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Account, error)
	GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Account, error)
	List(ctx context.Context, filter repository.AccountFilter) ([]domain.Account, int64, error)
	GetByType(ctx context.Context, companyID uuid.UUID, accountType domain.AccountType) ([]domain.Account, error)

	// Hierarchy operations
	GetTree(ctx context.Context, companyID uuid.UUID) ([]domain.Account, error)
	GetChildren(ctx context.Context, companyID, parentID uuid.UUID) ([]domain.Account, error)
	Move(ctx context.Context, companyID, id uuid.UUID, newParentID *uuid.UUID) error

	// Batch operations
	CreateBatch(ctx context.Context, accounts []domain.Account) error
	UpdateSortOrder(ctx context.Context, companyID uuid.UUID, orders map[uuid.UUID]int) error

	// Validation
	CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error)
	CanPost(ctx context.Context, companyID, id uuid.UUID) (bool, string, error)
}

// accountService implements AccountService
type accountService struct {
	repo repository.AccountRepository
}

// NewAccountService creates a new AccountService
func NewAccountService(repo repository.AccountRepository) AccountService {
	return &accountService{repo: repo}
}

// Create creates a new account
func (s *accountService) Create(ctx context.Context, account *domain.Account) error {
	// Set defaults
	account.SetDefaults()

	// Validate account data
	if err := account.Validate(); err != nil {
		return err
	}

	// Check for duplicate code
	exists, err := s.repo.ExistsByCode(ctx, account.CompanyID, account.Code, nil)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrAccountCodeExists
	}

	// Validate parent if specified
	if account.ParentID != nil {
		parent, err := s.repo.FindByID(ctx, account.CompanyID, *account.ParentID)
		if err != nil {
			if err == domain.ErrAccountNotFound {
				return domain.ErrParentNotFound
			}
			return err
		}
		account.Level = parent.Level + 1
	}

	// Create account
	if err := s.repo.Create(ctx, account); err != nil {
		return err
	}

	// Update path
	return s.repo.UpdatePath(ctx, account)
}

// Update updates an existing account
func (s *accountService) Update(ctx context.Context, account *domain.Account) error {
	// Validate account data
	if err := account.Validate(); err != nil {
		return err
	}

	// Check if account exists
	existing, err := s.repo.FindByID(ctx, account.CompanyID, account.ID)
	if err != nil {
		return err
	}

	// Check for duplicate code (excluding current account)
	exists, err := s.repo.ExistsByCode(ctx, account.CompanyID, account.Code, &account.ID)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrAccountCodeExists
	}

	// If parent changed, validate and update level
	if account.ParentID != existing.ParentID {
		if account.ParentID != nil {
			// Check for circular reference
			if *account.ParentID == account.ID {
				return domain.ErrCircularReference
			}

			// Check if new parent is a descendant
			descendants, err := s.repo.GetDescendants(ctx, account.CompanyID, account.ID)
			if err != nil {
				return err
			}
			for _, d := range descendants {
				if d.ID == *account.ParentID {
					return domain.ErrCircularReference
				}
			}

			// Get parent level
			parent, err := s.repo.FindByID(ctx, account.CompanyID, *account.ParentID)
			if err != nil {
				if err == domain.ErrAccountNotFound {
					return domain.ErrParentNotFound
				}
				return err
			}
			account.Level = parent.Level + 1
		} else {
			account.Level = 1
		}
	}

	// Update account
	if err := s.repo.Update(ctx, account); err != nil {
		return err
	}

	// Update path if code or parent changed
	if account.Code != existing.Code || account.ParentID != existing.ParentID {
		if err := s.repo.UpdatePath(ctx, account); err != nil {
			return err
		}
		// Update descendants' paths
		descendants, err := s.repo.GetDescendants(ctx, account.CompanyID, account.ID)
		if err != nil {
			return err
		}
		for _, d := range descendants {
			if err := s.repo.UpdatePath(ctx, &d); err != nil {
				return err
			}
		}
	}

	return nil
}

// Delete removes an account
func (s *accountService) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	// Check if can delete
	canDelete, reason, err := s.CanDelete(ctx, companyID, id)
	if err != nil {
		return err
	}
	if !canDelete {
		if reason == "children" {
			return domain.ErrAccountHasChildren
		}
		return domain.ErrAccountHasEntries
	}

	return s.repo.Delete(ctx, companyID, id)
}

// GetByID retrieves an account by ID
func (s *accountService) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Account, error) {
	return s.repo.FindByID(ctx, companyID, id)
}

// GetByCode retrieves an account by code
func (s *accountService) GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Account, error) {
	return s.repo.FindByCode(ctx, companyID, code)
}

// List retrieves accounts with filtering and pagination
func (s *accountService) List(ctx context.Context, filter repository.AccountFilter) ([]domain.Account, int64, error) {
	return s.repo.FindAll(ctx, filter)
}

// GetByType retrieves accounts by type
func (s *accountService) GetByType(ctx context.Context, companyID uuid.UUID, accountType domain.AccountType) ([]domain.Account, error) {
	return s.repo.FindByType(ctx, companyID, accountType)
}

// GetTree retrieves the full account tree
func (s *accountService) GetTree(ctx context.Context, companyID uuid.UUID) ([]domain.Account, error) {
	return s.repo.GetTree(ctx, companyID)
}

// GetChildren retrieves direct children of an account
func (s *accountService) GetChildren(ctx context.Context, companyID, parentID uuid.UUID) ([]domain.Account, error) {
	return s.repo.FindChildren(ctx, companyID, parentID)
}

// Move moves an account to a new parent
func (s *accountService) Move(ctx context.Context, companyID, id uuid.UUID, newParentID *uuid.UUID) error {
	account, err := s.repo.FindByID(ctx, companyID, id)
	if err != nil {
		return err
	}

	account.ParentID = newParentID
	return s.Update(ctx, account)
}

// CreateBatch creates multiple accounts
func (s *accountService) CreateBatch(ctx context.Context, accounts []domain.Account) error {
	// Validate all accounts
	for i := range accounts {
		accounts[i].SetDefaults()
		if err := accounts[i].Validate(); err != nil {
			return err
		}
	}

	// Create accounts
	if err := s.repo.CreateBatch(ctx, accounts); err != nil {
		return err
	}

	// Update paths
	for i := range accounts {
		if err := s.repo.UpdatePath(ctx, &accounts[i]); err != nil {
			return err
		}
	}

	return nil
}

// UpdateSortOrder updates sort orders for accounts
func (s *accountService) UpdateSortOrder(ctx context.Context, companyID uuid.UUID, orders map[uuid.UUID]int) error {
	return s.repo.UpdateSortOrder(ctx, companyID, orders)
}

// CanDelete checks if an account can be deleted
func (s *accountService) CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error) {
	// Check if account exists
	_, err := s.repo.FindByID(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}

	// Check for children
	hasChildren, err := s.repo.HasChildren(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}
	if hasChildren {
		return false, "children", nil
	}

	// Check for voucher entries
	hasEntries, err := s.repo.HasVoucherEntries(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}
	if hasEntries {
		return false, "entries", nil
	}

	return true, "", nil
}

// CanPost checks if direct posting is allowed on an account
func (s *accountService) CanPost(ctx context.Context, companyID, id uuid.UUID) (bool, string, error) {
	account, err := s.repo.FindByID(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}

	if !account.IsActive {
		return false, "inactive", nil
	}
	if account.IsControlAccount {
		return false, "control", nil
	}
	if !account.AllowDirectPosting {
		return false, "no_posting", nil
	}

	return true, "", nil
}
