package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// PositionFilter defines filter criteria for listing positions.
//
// CompanyID is not optional: every query this repository issues is scoped to
// one tenant, and a zero CompanyID matches nothing rather than everything.
type PositionFilter struct {
	CompanyID  uuid.UUID
	IsActive   *bool
	SearchTerm string // matched against code, name and name_en
	Page       int
	PageSize   int
}

// PositionRepository defines the interface for position data access.
type PositionRepository interface {
	// CRUD operations
	Create(ctx context.Context, position *domain.Position) error
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Position, error)
	GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Position, error)
	List(ctx context.Context, filter *PositionFilter) ([]domain.Position, int64, error)
	Update(ctx context.Context, position *domain.Position) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Validation
	ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)

	// CountEmployees returns how many live (not soft-deleted) employees hold
	// this position. positions has no deleted_at, so its Delete is a real
	// DELETE and employees.position_id would refuse it.
	CountEmployees(ctx context.Context, companyID, positionID uuid.UUID) (int64, error)
}
