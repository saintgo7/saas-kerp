package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/saintgo7/saas-kerp/internal/domain"
)

// ProjectFilter defines filter options for project queries
type ProjectFilter struct {
	CompanyID  uuid.UUID
	Status     *domain.ProjectStatus
	ManagerID  *uuid.UUID
	SearchTerm string
	IsActive   *bool
	Page       int
	PageSize   int
}

// ProjectRepository defines the interface for project data access
type ProjectRepository interface {
	// CRUD operations
	Create(ctx context.Context, project *domain.Project) error
	Update(ctx context.Context, project *domain.Project) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Query operations
	FindByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Project, error)
	FindByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Project, error)
	FindAll(ctx context.Context, filter ProjectFilter) ([]domain.Project, int64, error)

	// Validation helpers
	ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)

	// Usage check
	IsInUse(ctx context.Context, companyID, projectID uuid.UUID) (bool, error)

	// Statistics
	GetStats(ctx context.Context, companyID uuid.UUID) (*ProjectStats, error)
}

// ProjectStats represents project statistics
type ProjectStats struct {
	TotalCount      int64
	ActiveCount     int64
	CompletedCount  int64
	OnHoldCount     int64
	TotalBudget     float64
	TotalActualCost float64
}
