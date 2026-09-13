package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// departmentRepositoryGorm implements DepartmentRepository using GORM
type departmentRepositoryGorm struct {
	db *gorm.DB
}

// maxHierarchyDepth bounds every recursive department walk. Organisation
// charts are never this deep; the cap exists so a corrupt parent_id cannot
// turn a lookup into an unbounded query.
const maxHierarchyDepth = 64

// NewDepartmentRepositoryGorm creates a new DepartmentRepository with GORM
func NewDepartmentRepositoryGorm(db *gorm.DB) DepartmentRepository {
	return &departmentRepositoryGorm{db: db}
}

// Create creates a new department
func (r *departmentRepositoryGorm) Create(ctx context.Context, dept *domain.Department) error {
	return r.db.WithContext(ctx).Create(dept).Error
}

// GetByID retrieves a department by ID
func (r *departmentRepositoryGorm) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Department, error) {
	var dept domain.Department
	err := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		First(&dept).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("department not found")
		}
		return nil, err
	}
	return &dept, nil
}

// GetByCode retrieves a department by code
func (r *departmentRepositoryGorm) GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Department, error) {
	var dept domain.Department
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND code = ?", companyID, code).
		First(&dept).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("department not found")
		}
		return nil, err
	}
	return &dept, nil
}

// List retrieves departments with filtering
func (r *departmentRepositoryGorm) List(ctx context.Context, filter *DepartmentFilter) ([]domain.Department, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Department{}).
		Where("company_id = ?", filter.CompanyID)

	// Apply filters
	if filter.ParentID != nil {
		query = query.Where("parent_id = ?", *filter.ParentID)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.SearchTerm != "" {
		searchPattern := "%" + filter.SearchTerm + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", searchPattern, searchPattern)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	query = query.Order("level ASC, code ASC")

	var depts []domain.Department
	if err := query.Find(&depts).Error; err != nil {
		return nil, 0, err
	}

	return depts, total, nil
}

// Update updates a department
func (r *departmentRepositoryGorm) Update(ctx context.Context, dept *domain.Department) error {
	return r.db.WithContext(ctx).Save(dept).Error
}

// Delete deletes a department
func (r *departmentRepositoryGorm) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		Delete(&domain.Department{}).Error
}

// GetTree retrieves all departments as a flat list (for tree construction)
func (r *departmentRepositoryGorm) GetTree(ctx context.Context, companyID uuid.UUID) ([]domain.Department, error) {
	var depts []domain.Department
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND is_active = true", companyID).
		Order("level ASC, code ASC").
		Find(&depts).Error
	if err != nil {
		return nil, err
	}
	return depts, nil
}

// GetChildren retrieves child departments
func (r *departmentRepositoryGorm) GetChildren(ctx context.Context, companyID, parentID uuid.UUID) ([]domain.Department, error) {
	var depts []domain.Department
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND parent_id = ?", companyID, parentID).
		Order("code ASC").
		Find(&depts).Error
	if err != nil {
		return nil, err
	}
	return depts, nil
}

// GetAncestors retrieves all ancestors of a department
func (r *departmentRepositoryGorm) GetAncestors(ctx context.Context, companyID, id uuid.UUID) ([]domain.Department, error) {
	var depts []domain.Department
	// The depth counter is a hard stop, not an optimisation. A department whose
	// parent_id points at itself (or at any node in its own ancestor chain)
	// makes this UNION ALL run forever, holding a connection and growing memory
	// until the pool is exhausted. The recursive term also repeats the
	// company_id predicate so a parent_id that crosses tenants cannot pull
	// another company's rows into the result.
	query := `
		WITH RECURSIVE ancestors AS (
			SELECT d.*, 1 AS depth FROM departments d WHERE d.id = ? AND d.company_id = ?
			UNION ALL
			SELECT d.*, a.depth + 1 FROM departments d
			JOIN ancestors a ON d.id = a.parent_id
			WHERE d.company_id = ? AND a.depth < ?
		)
		SELECT * FROM ancestors WHERE id != ? ORDER BY level ASC
	`
	err := r.db.WithContext(ctx).Raw(query, id, companyID, companyID, maxHierarchyDepth, id).Scan(&depts).Error
	if err != nil {
		return nil, err
	}
	return depts, nil
}

// GetDescendants retrieves all descendants of a department
func (r *departmentRepositoryGorm) GetDescendants(ctx context.Context, companyID, id uuid.UUID) ([]domain.Department, error) {
	var depts []domain.Department
	// See GetAncestors: depth cap prevents a cyclic parent_id from hanging the
	// connection, and the company_id predicate is repeated in the recursive
	// term so the walk cannot leave the tenant.
	query := `
		WITH RECURSIVE descendants AS (
			SELECT d.*, 1 AS depth FROM departments d WHERE d.parent_id = ? AND d.company_id = ?
			UNION ALL
			SELECT d.*, c.depth + 1 FROM departments d
			JOIN descendants c ON d.parent_id = c.id
			WHERE d.company_id = ? AND c.depth < ?
		)
		SELECT * FROM descendants ORDER BY level ASC, code ASC
	`
	err := r.db.WithContext(ctx).Raw(query, id, companyID, companyID, maxHierarchyDepth).Scan(&depts).Error
	if err != nil {
		return nil, err
	}
	return depts, nil
}

// ExistsByCode checks if a department with the given code exists
func (r *departmentRepositoryGorm) ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&domain.Department{}).
		Where("company_id = ? AND code = ?", companyID, code)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasChildren checks if a department has children
func (r *departmentRepositoryGorm) HasChildren(ctx context.Context, companyID, id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Department{}).
		Where("company_id = ? AND parent_id = ?", companyID, id).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasVoucherEntries checks if the department has any voucher entries
func (r *departmentRepositoryGorm) HasVoucherEntries(ctx context.Context, companyID, departmentID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.VoucherEntry{}).
		Where("company_id = ? AND department_id = ?", companyID, departmentID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateActiveStatus updates the active status for multiple departments
func (r *departmentRepositoryGorm) UpdateActiveStatus(ctx context.Context, companyID uuid.UUID, ids []uuid.UUID, isActive bool) error {
	return r.db.WithContext(ctx).Model(&domain.Department{}).
		Where("company_id = ? AND id IN ?", companyID, ids).
		Update("is_active", isActive).Error
}
