package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/saintgo7/saas-kerp/internal/domain"
)

// AccountFilter defines filter options for account queries
type AccountFilter struct {
	CompanyID    uuid.UUID
	ParentID     *uuid.UUID
	AccountType  *domain.AccountType
	IsActive     *bool
	SearchTerm   string
	IncludeTree  bool
	Page         int
	PageSize     int
	SortBy       string
	SortDesc     bool
}

// AccountRepository defines the interface for account data access
type AccountRepository interface {
	// CRUD operations
	Create(ctx context.Context, account *domain.Account) error
	Update(ctx context.Context, account *domain.Account) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Query operations
	FindByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Account, error)
	FindByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Account, error)
	FindAll(ctx context.Context, filter AccountFilter) ([]domain.Account, int64, error)
	FindChildren(ctx context.Context, companyID, parentID uuid.UUID) ([]domain.Account, error)
	FindByType(ctx context.Context, companyID uuid.UUID, accountType domain.AccountType) ([]domain.Account, error)

	// Hierarchy operations
	GetTree(ctx context.Context, companyID uuid.UUID) ([]domain.Account, error)
	GetAncestors(ctx context.Context, companyID, id uuid.UUID) ([]domain.Account, error)
	GetDescendants(ctx context.Context, companyID, id uuid.UUID) ([]domain.Account, error)
	UpdatePath(ctx context.Context, account *domain.Account) error

	// Validation helpers
	ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)
	HasChildren(ctx context.Context, companyID, id uuid.UUID) (bool, error)
	HasVoucherEntries(ctx context.Context, companyID, id uuid.UUID) (bool, error)

	// Batch operations
	CreateBatch(ctx context.Context, accounts []domain.Account) error
	UpdateSortOrder(ctx context.Context, companyID uuid.UUID, orders map[uuid.UUID]int) error
}
