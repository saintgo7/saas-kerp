package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// CompanyService defines the interface for company business logic
type CompanyService interface {
	// Query operations
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Company, error)

	// Update operations
	Update(ctx context.Context, company *domain.Company) error
	UpdateSettings(ctx context.Context, company *domain.Company) error
}

// companyServiceImpl implements CompanyService
type companyServiceImpl struct {
	repo repository.CompanyRepository
}

// NewCompanyService creates a new company service
func NewCompanyService(repo repository.CompanyRepository) CompanyService {
	return &companyServiceImpl{repo: repo}
}

func (s *companyServiceImpl) GetByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *companyServiceImpl) Update(ctx context.Context, company *domain.Company) error {
	return s.repo.Update(ctx, company)
}

func (s *companyServiceImpl) UpdateSettings(ctx context.Context, company *domain.Company) error {
	return s.repo.Update(ctx, company)
}
