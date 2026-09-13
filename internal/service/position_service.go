package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// Position-related errors.
//
// They are the domain errors, not copies: the repository returns
// domain.ErrPositionNotFound and the handler compares against
// service.ErrPositionNotFound, so aliasing keeps the two ends in agreement
// instead of letting a second, look-alike error value drift away from it.
var (
	ErrPositionNotFound    = domain.ErrPositionNotFound
	ErrPositionCodeExists  = domain.ErrPositionCodeExists
	ErrPositionInUse       = domain.ErrPositionInUse
	ErrPositionSalaryRange = domain.ErrPositionSalaryRange
)

// PositionFilter is re-exported from repository
type PositionFilter = repository.PositionFilter

// Page-size bounds for the HR services. They mirror
// internal/handler/response.DefaultPerPage / MaxPerPage, which the service
// layer cannot import without depending on the transport layer.
const (
	hrDefaultPageSize = 20
	hrMaxPageSize     = 100
)

// clampHRListPage forces a page and page size into the supported range.
//
// The handlers already clamp what arrives over HTTP, but a service is also
// called from tests, jobs and other services: a page size of 0 must never
// reach a LIMIT or a total-pages division, and an unbounded one must never
// turn a list call into a full-table scan.
func clampHRListPage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = hrDefaultPageSize
	}
	if pageSize > hrMaxPageSize {
		pageSize = hrMaxPageSize
	}
	return page, pageSize
}

// PositionService defines the interface for position business logic.
type PositionService interface {
	Create(ctx context.Context, position *domain.Position) error
	Update(ctx context.Context, position *domain.Position) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Position, error)
	GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Position, error)
	List(ctx context.Context, filter *PositionFilter) ([]domain.Position, int64, error)

	CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error)
}

// positionService implements PositionService.
type positionService struct {
	repo repository.PositionRepository
}

// NewPositionService creates a new PositionService.
func NewPositionService(repo repository.PositionRepository) PositionService {
	return &positionService{repo: repo}
}

// Create validates and stores a new position.
func (s *positionService) Create(ctx context.Context, position *domain.Position) error {
	position.Code = strings.TrimSpace(position.Code)
	position.Name = strings.TrimSpace(position.Name)

	if err := position.Validate(); err != nil {
		return err
	}

	exists, err := s.repo.ExistsByCode(ctx, position.CompanyID, position.Code, nil)
	if err != nil {
		return err
	}
	if exists {
		return ErrPositionCodeExists
	}

	return s.repo.Create(ctx, position)
}

// Update validates and stores changes to a position.
func (s *positionService) Update(ctx context.Context, position *domain.Position) error {
	position.Code = strings.TrimSpace(position.Code)
	position.Name = strings.TrimSpace(position.Name)

	if err := position.Validate(); err != nil {
		return err
	}

	if _, err := s.repo.GetByID(ctx, position.CompanyID, position.ID); err != nil {
		return err
	}

	exists, err := s.repo.ExistsByCode(ctx, position.CompanyID, position.Code, &position.ID)
	if err != nil {
		return err
	}
	if exists {
		return ErrPositionCodeExists
	}

	return s.repo.Update(ctx, position)
}

// Delete removes a position once nothing references it.
func (s *positionService) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	canDelete, _, err := s.CanDelete(ctx, companyID, id)
	if err != nil {
		return err
	}
	if !canDelete {
		return ErrPositionInUse
	}
	return s.repo.Delete(ctx, companyID, id)
}

// GetByID retrieves one position.
func (s *positionService) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Position, error) {
	return s.repo.GetByID(ctx, companyID, id)
}

// GetByCode retrieves one position by code.
func (s *positionService) GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Position, error) {
	return s.repo.GetByCode(ctx, companyID, code)
}

// List retrieves positions with filtering.
func (s *positionService) List(ctx context.Context, filter *PositionFilter) ([]domain.Position, int64, error) {
	filter.Page, filter.PageSize = clampHRListPage(filter.Page, filter.PageSize)
	return s.repo.List(ctx, filter)
}

// CanDelete reports whether the position may be deleted, and why not.
//
// positions has no deleted_at, so Delete is a real DELETE that
// employees.position_id would refuse. Answering here turns a driver-level
// foreign key violation into a message a user can act on.
func (s *positionService) CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error) {
	if _, err := s.repo.GetByID(ctx, companyID, id); err != nil {
		return false, "", err
	}

	count, err := s.repo.CountEmployees(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}
	if count > 0 {
		return false, "position is assigned to employees", nil
	}

	return true, "", nil
}
