package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// RoleService defines the interface for role business logic
type RoleService interface {
	// CRUD operations
	Create(ctx context.Context, role *domain.Role) error
	Update(ctx context.Context, role *domain.Role) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Query operations
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Role, error)
	GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Role, error)
	List(ctx context.Context, filter repository.RoleFilter) ([]domain.Role, int64, error)

	// Permission management
	SetPermissions(ctx context.Context, companyID, roleID uuid.UUID, permissions []domain.Permission) error

	// Validation
	CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error)
}

// roleServiceImpl implements RoleService
type roleServiceImpl struct {
	repo repository.RoleRepository
}

// NewRoleService creates a new role service
func NewRoleService(repo repository.RoleRepository) RoleService {
	return &roleServiceImpl{repo: repo}
}

func (s *roleServiceImpl) Create(ctx context.Context, role *domain.Role) error {
	// Check if code already exists
	exists, err := s.repo.ExistsByCode(ctx, role.CompanyID, role.Code, nil)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrRoleCodeExists
	}

	return s.repo.Create(ctx, role)
}

func (s *roleServiceImpl) Update(ctx context.Context, role *domain.Role) error {
	// Check if code already exists (excluding current role)
	exists, err := s.repo.ExistsByCode(ctx, role.CompanyID, role.Code, &role.ID)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrRoleCodeExists
	}

	// Prevent modifying system roles
	existing, err := s.repo.FindByID(ctx, role.CompanyID, role.ID)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		// Only allow permission updates for system roles
		existing.Permissions = role.Permissions
		return s.repo.Update(ctx, existing)
	}

	return s.repo.Update(ctx, role)
}

func (s *roleServiceImpl) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	// Check if role exists
	role, err := s.repo.FindByID(ctx, companyID, id)
	if err != nil {
		return err
	}

	// Prevent deleting system roles
	if role.IsSystem {
		return domain.ErrRoleInUse
	}

	// Check if role is in use
	inUse, err := s.repo.IsInUse(ctx, companyID, id)
	if err != nil {
		return err
	}
	if inUse {
		return domain.ErrRoleInUse
	}

	return s.repo.Delete(ctx, companyID, id)
}

func (s *roleServiceImpl) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Role, error) {
	return s.repo.FindByID(ctx, companyID, id)
}

func (s *roleServiceImpl) GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Role, error) {
	return s.repo.FindByCode(ctx, companyID, code)
}

func (s *roleServiceImpl) List(ctx context.Context, filter repository.RoleFilter) ([]domain.Role, int64, error) {
	return s.repo.FindAll(ctx, filter)
}

func (s *roleServiceImpl) SetPermissions(ctx context.Context, companyID, roleID uuid.UUID, permissions []domain.Permission) error {
	role, err := s.repo.FindByID(ctx, companyID, roleID)
	if err != nil {
		return err
	}

	role.SetPermissions(permissions)
	return s.repo.Update(ctx, role)
}

func (s *roleServiceImpl) CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error) {
	role, err := s.repo.FindByID(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}

	if role.IsSystem {
		return false, "System roles cannot be deleted", nil
	}

	inUse, err := s.repo.IsInUse(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}
	if inUse {
		return false, "Role is assigned to users", nil
	}

	return true, "", nil
}
