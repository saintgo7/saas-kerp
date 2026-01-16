package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// DepartmentFilter defines filter criteria for listing departments
type DepartmentFilter struct {
	CompanyID  uuid.UUID
	ParentID   *uuid.UUID
	IsActive   *bool
	SearchTerm string
	Page       int
	PageSize   int
}

// DepartmentRepository defines the interface for department data access
type DepartmentRepository interface {
	// CRUD operations
	Create(ctx context.Context, dept *domain.Department) error
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Department, error)
	GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Department, error)
	List(ctx context.Context, filter *DepartmentFilter) ([]domain.Department, int64, error)
	Update(ctx context.Context, dept *domain.Department) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Hierarchy operations
	GetTree(ctx context.Context, companyID uuid.UUID) ([]domain.Department, error)
	GetChildren(ctx context.Context, companyID, parentID uuid.UUID) ([]domain.Department, error)
	GetAncestors(ctx context.Context, companyID, id uuid.UUID) ([]domain.Department, error)
	GetDescendants(ctx context.Context, companyID, id uuid.UUID) ([]domain.Department, error)

	// Validation
	ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)
	HasChildren(ctx context.Context, companyID, id uuid.UUID) (bool, error)
	HasVoucherEntries(ctx context.Context, companyID, departmentID uuid.UUID) (bool, error)

	// Batch operations
	UpdateActiveStatus(ctx context.Context, companyID uuid.UUID, ids []uuid.UUID, isActive bool) error
}
