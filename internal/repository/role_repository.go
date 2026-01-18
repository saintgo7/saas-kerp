package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/saintgo7/saas-kerp/internal/domain"
)

// RoleFilter defines filter options for role queries
type RoleFilter struct {
	CompanyID  uuid.UUID
	SearchTerm string
	IsActive   *bool
	Page       int
	PageSize   int
}

// RoleRepository defines the interface for role data access
type RoleRepository interface {
	// CRUD operations
	Create(ctx context.Context, role *domain.Role) error
	Update(ctx context.Context, role *domain.Role) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Query operations
	FindByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Role, error)
	FindByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Role, error)
	FindAll(ctx context.Context, filter RoleFilter) ([]domain.Role, int64, error)

	// Validation helpers
	ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)

	// Usage check
	IsInUse(ctx context.Context, companyID, roleID uuid.UUID) (bool, error)
}
