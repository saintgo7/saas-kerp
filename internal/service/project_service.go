package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// ProjectService defines the interface for project business logic
type ProjectService interface {
	// CRUD operations
	Create(ctx context.Context, project *domain.Project) error
	Update(ctx context.Context, project *domain.Project) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Query operations
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Project, error)
	GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Project, error)
	List(ctx context.Context, filter repository.ProjectFilter) ([]domain.Project, int64, error)

	// Validation
	CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error)

	// Statistics
	GetStats(ctx context.Context, companyID uuid.UUID) (*repository.ProjectStats, error)
}

// projectServiceImpl implements ProjectService
type projectServiceImpl struct {
	repo repository.ProjectRepository
}

// NewProjectService creates a new project service
func NewProjectService(repo repository.ProjectRepository) ProjectService {
	return &projectServiceImpl{repo: repo}
}

func (s *projectServiceImpl) Create(ctx context.Context, project *domain.Project) error {
	// Check if code already exists
	exists, err := s.repo.ExistsByCode(ctx, project.CompanyID, project.Code, nil)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrProjectCodeExists
	}

	return s.repo.Create(ctx, project)
}

func (s *projectServiceImpl) Update(ctx context.Context, project *domain.Project) error {
	// Check if code already exists (excluding current project)
	exists, err := s.repo.ExistsByCode(ctx, project.CompanyID, project.Code, &project.ID)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrProjectCodeExists
	}

	return s.repo.Update(ctx, project)
}

func (s *projectServiceImpl) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	// Check if project is in use
	inUse, err := s.repo.IsInUse(ctx, companyID, id)
	if err != nil {
		return err
	}
	if inUse {
		return domain.ErrProjectInUse
	}

	return s.repo.Delete(ctx, companyID, id)
}

func (s *projectServiceImpl) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Project, error) {
	return s.repo.FindByID(ctx, companyID, id)
}

func (s *projectServiceImpl) GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Project, error) {
	return s.repo.FindByCode(ctx, companyID, code)
}

func (s *projectServiceImpl) List(ctx context.Context, filter repository.ProjectFilter) ([]domain.Project, int64, error) {
	return s.repo.FindAll(ctx, filter)
}

func (s *projectServiceImpl) CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error) {
	// Check if project exists
	_, err := s.repo.FindByID(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}

	// Check if project is in use
	inUse, err := s.repo.IsInUse(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}
	if inUse {
		return false, "Project is referenced in transactions", nil
	}

	return true, "", nil
}

func (s *projectServiceImpl) GetStats(ctx context.Context, companyID uuid.UUID) (*repository.ProjectStats, error) {
	return s.repo.GetStats(ctx, companyID)
}
