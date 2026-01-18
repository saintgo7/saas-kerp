package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// roleRepositoryGorm implements RoleRepository using GORM
type roleRepositoryGorm struct {
	db *gorm.DB
}

// NewRoleRepository creates a new GORM-based role repository
func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepositoryGorm{db: db}
}

func (r *roleRepositoryGorm) Create(ctx context.Context, role *domain.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *roleRepositoryGorm) Update(ctx context.Context, role *domain.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *roleRepositoryGorm) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("company_id = ? AND id = ?", companyID, id).
		Delete(&domain.Role{}).Error
}

func (r *roleRepositoryGorm) FindByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Role, error) {
	var role domain.Role
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND id = ?", companyID, id).
		First(&role).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrRoleNotFound
		}
		return nil, err
	}
	return &role, nil
}

func (r *roleRepositoryGorm) FindByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Role, error) {
	var role domain.Role
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND code = ?", companyID, code).
		First(&role).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrRoleNotFound
		}
		return nil, err
	}
	return &role, nil
}

func (r *roleRepositoryGorm) FindAll(ctx context.Context, filter RoleFilter) ([]domain.Role, int64, error) {
	var roles []domain.Role
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Role{}).
		Where("company_id = ?", filter.CompanyID)

	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.SearchTerm != "" {
		searchPattern := "%" + filter.SearchTerm + "%"
		query = query.Where("name ILIKE ? OR code ILIKE ?", searchPattern, searchPattern)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	query = query.Order("code ASC")

	// Apply pagination
	if filter.PageSize > 0 {
		query = query.Limit(filter.PageSize)
		if filter.Page > 0 {
			query = query.Offset((filter.Page - 1) * filter.PageSize)
		}
	}

	if err := query.Find(&roles).Error; err != nil {
		return nil, 0, err
	}

	return roles, total, nil
}

func (r *roleRepositoryGorm) ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&domain.Role{}).
		Where("company_id = ? AND code = ?", companyID, code)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *roleRepositoryGorm) IsInUse(ctx context.Context, companyID, roleID uuid.UUID) (bool, error) {
	// Check if any users have this role assigned
	// For now, return false as we don't have user-role association table
	// This would need to be implemented when role-based access is fully set up
	return false, nil
}
