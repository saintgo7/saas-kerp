package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/saintgo7/saas-kerp/internal/domain"
)

// CompanyRepository defines the interface for company data access
type CompanyRepository interface {
	// CRUD operations
	Create(ctx context.Context, company *domain.Company) error
	Update(ctx context.Context, company *domain.Company) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Query operations
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Company, error)
	FindByCode(ctx context.Context, code string) (*domain.Company, error)
	FindAll(ctx context.Context) ([]domain.Company, error)

	// Validation helpers
	ExistsByCode(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error)
}
