package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// Department-related errors
var (
	ErrDepartmentNotFound        = errors.New("department not found")
	ErrDepartmentCodeExists      = errors.New("department code already exists")
	ErrDepartmentHasChildren     = errors.New("department has child departments")
	ErrDepartmentHasTransactions = errors.New("department has transactions")
	ErrDepartmentCircularRef     = errors.New("circular reference detected")
)

// DepartmentFilter is re-exported from repository
type DepartmentFilter = repository.DepartmentFilter

// DepartmentService defines the interface for department business logic
type DepartmentService interface {
	// CRUD operations
	Create(ctx context.Context, dept *domain.Department) error
	Update(ctx context.Context, dept *domain.Department) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Query operations
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Department, error)
	GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Department, error)
	List(ctx context.Context, filter *DepartmentFilter) ([]domain.Department, int64, error)

	// Hierarchy operations
	GetTree(ctx context.Context, companyID uuid.UUID) ([]domain.Department, error)
	GetChildren(ctx context.Context, companyID, parentID uuid.UUID) ([]domain.Department, error)
	Move(ctx context.Context, companyID, id uuid.UUID, newParentID *uuid.UUID) error

	// Validation
	CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error)
}

// departmentService implements DepartmentService
type departmentService struct {
	repo repository.DepartmentRepository
}

// NewDepartmentService creates a new DepartmentService
func NewDepartmentService(repo repository.DepartmentRepository) DepartmentService {
	return &departmentService{repo: repo}
}

// Create creates a new department
func (s *departmentService) Create(ctx context.Context, dept *domain.Department) error {
	// Check for duplicate code
	exists, err := s.repo.ExistsByCode(ctx, dept.CompanyID, dept.Code, nil)
	if err != nil {
		return err
	}
	if exists {
		return ErrDepartmentCodeExists
	}

	// Set level based on parent
	if dept.ParentID != nil {
		parent, err := s.repo.GetByID(ctx, dept.CompanyID, *dept.ParentID)
		if err != nil {
			return err
		}
		dept.Level = parent.Level + 1
	} else {
		dept.Level = 1
	}

	return s.repo.Create(ctx, dept)
}

// Update updates a department
func (s *departmentService) Update(ctx context.Context, dept *domain.Department) error {
	// Check existing
	existing, err := s.repo.GetByID(ctx, dept.CompanyID, dept.ID)
	if err != nil {
		return ErrDepartmentNotFound
	}

	// Check for duplicate code
	exists, err := s.repo.ExistsByCode(ctx, dept.CompanyID, dept.Code, &dept.ID)
	if err != nil {
		return err
	}
	if exists {
		return ErrDepartmentCodeExists
	}

	// If parent changed, check for circular reference
	if (dept.ParentID == nil && existing.ParentID != nil) ||
		(dept.ParentID != nil && existing.ParentID == nil) ||
		(dept.ParentID != nil && existing.ParentID != nil && *dept.ParentID != *existing.ParentID) {

		if dept.ParentID != nil {
			// A department cannot be its own parent. GetDescendants anchors on
			// `WHERE parent_id = ?` and therefore never contains the node
			// itself, so the descendant scan below cannot catch this case: for
			// a leaf department it returns an empty set and the self-reference
			// was accepted. Once stored, every recursive walk over that
			// company's departments looped forever.
			if *dept.ParentID == dept.ID {
				return ErrDepartmentCircularRef
			}

			// Check if new parent is a descendant
			descendants, err := s.repo.GetDescendants(ctx, dept.CompanyID, dept.ID)
			if err != nil {
				return err
			}
			for _, d := range descendants {
				if d.ID == *dept.ParentID {
					return ErrDepartmentCircularRef
				}
			}

			// Update level
			parent, err := s.repo.GetByID(ctx, dept.CompanyID, *dept.ParentID)
			if err != nil {
				return err
			}
			dept.Level = parent.Level + 1
		} else {
			dept.Level = 1
		}
	}

	return s.repo.Update(ctx, dept)
}

// Delete deletes a department.
//
// The refusal is reported through the declared sentinel errors rather than as
// errors.New(reason): the handler maps sentinels to 409 Conflict, and an
// anonymous error string fell through to a generic 500.
func (s *departmentService) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	canDelete, reason, err := s.CanDelete(ctx, companyID, id)
	if err != nil {
		return err
	}
	if !canDelete {
		switch reason {
		case reasonDepartmentHasChildren:
			return ErrDepartmentHasChildren
		case reasonDepartmentHasEntries:
			return ErrDepartmentHasTransactions
		default:
			return errors.New(reason)
		}
	}

	return s.repo.Delete(ctx, companyID, id)
}

// Reasons reported by CanDelete. They are constants so Delete can map them
// back onto the sentinel errors without matching free-form strings.
const (
	reasonDepartmentHasChildren = "department has child departments"
	reasonDepartmentHasEntries  = "department has voucher entries"
)

// GetByID retrieves a department by ID
func (s *departmentService) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Department, error) {
	return s.repo.GetByID(ctx, companyID, id)
}

// GetByCode retrieves a department by code
func (s *departmentService) GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Department, error) {
	return s.repo.GetByCode(ctx, companyID, code)
}

// List retrieves departments with filtering
func (s *departmentService) List(ctx context.Context, filter *DepartmentFilter) ([]domain.Department, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	return s.repo.List(ctx, filter)
}

// GetTree retrieves departments as a tree structure
func (s *departmentService) GetTree(ctx context.Context, companyID uuid.UUID) ([]domain.Department, error) {
	return s.repo.GetTree(ctx, companyID)
}

// GetChildren retrieves child departments
func (s *departmentService) GetChildren(ctx context.Context, companyID, parentID uuid.UUID) ([]domain.Department, error) {
	return s.repo.GetChildren(ctx, companyID, parentID)
}

// Move moves a department to a new parent
func (s *departmentService) Move(ctx context.Context, companyID, id uuid.UUID, newParentID *uuid.UUID) error {
	dept, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return ErrDepartmentNotFound
	}

	// Check for circular reference if moving to a new parent
	if newParentID != nil {
		// See Update: GetDescendants excludes the node itself, so the
		// self-parent case has to be rejected explicitly.
		if *newParentID == id {
			return ErrDepartmentCircularRef
		}

		descendants, err := s.repo.GetDescendants(ctx, companyID, id)
		if err != nil {
			return err
		}
		for _, d := range descendants {
			if d.ID == *newParentID {
				return ErrDepartmentCircularRef
			}
		}

		// Update level
		parent, err := s.repo.GetByID(ctx, companyID, *newParentID)
		if err != nil {
			return err
		}
		dept.Level = parent.Level + 1
	} else {
		dept.Level = 1
	}

	dept.ParentID = newParentID
	return s.repo.Update(ctx, dept)
}

// CanDelete checks if a department can be deleted
func (s *departmentService) CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error) {
	// Check if department exists. Only a genuine miss is reported as
	// not-found; a database failure must surface as itself, otherwise a broken
	// connection is reported to the client as a 404.
	_, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		if errors.Is(err, domain.ErrDepartmentNotFound) || errors.Is(err, ErrDepartmentNotFound) {
			return false, "department not found", ErrDepartmentNotFound
		}
		return false, "", err
	}

	// Check for child departments
	hasChildren, err := s.repo.HasChildren(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}
	if hasChildren {
		return false, reasonDepartmentHasChildren, nil
	}

	// Check for voucher entries
	hasEntries, err := s.repo.HasVoucherEntries(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}
	if hasEntries {
		return false, reasonDepartmentHasEntries, nil
	}

	return true, "", nil
}
