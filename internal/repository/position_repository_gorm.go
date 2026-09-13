package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// positionRepositoryGorm implements PositionRepository using GORM.
type positionRepositoryGorm struct {
	db *gorm.DB
}

// NewPositionRepositoryGorm creates a new PositionRepository with GORM.
func NewPositionRepositoryGorm(db *gorm.DB) PositionRepository {
	return &positionRepositoryGorm{db: db}
}

// Create inserts a position.
func (r *positionRepositoryGorm) Create(ctx context.Context, position *domain.Position) error {
	return r.db.WithContext(ctx).Create(position).Error
}

// GetByID retrieves one position of one company.
func (r *positionRepositoryGorm) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Position, error) {
	var position domain.Position
	err := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		First(&position).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrPositionNotFound
		}
		return nil, err
	}
	return &position, nil
}

// GetByCode retrieves one position of one company by its code.
func (r *positionRepositoryGorm) GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Position, error) {
	var position domain.Position
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND code = ?", companyID, code).
		First(&position).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrPositionNotFound
		}
		return nil, err
	}
	return &position, nil
}

// List retrieves positions with filtering and pagination.
func (r *positionRepositoryGorm) List(ctx context.Context, filter *PositionFilter) ([]domain.Position, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Position{}).
		Where("company_id = ?", filter.CompanyID)

	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.SearchTerm != "" {
		pattern := "%" + filter.SearchTerm + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ? OR name_en ILIKE ?", pattern, pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	var positions []domain.Position
	if err := query.Order("rank_level DESC, code ASC").Find(&positions).Error; err != nil {
		return nil, 0, err
	}
	return positions, total, nil
}

// Update writes a position back. The company_id predicate keeps a caller from
// updating another tenant's row by supplying its id.
func (r *positionRepositoryGorm) Update(ctx context.Context, position *domain.Position) error {
	result := r.db.WithContext(ctx).Model(&domain.Position{}).
		Where("id = ? AND company_id = ?", position.ID, position.CompanyID).
		Select("code", "name", "name_en", "rank_level", "min_salary", "max_salary", "is_active", "updated_at").
		Updates(position)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrPositionNotFound
	}
	return nil
}

// Delete removes a position. positions carries no deleted_at, so this is a
// real DELETE; the service refuses it while employees still reference the row.
func (r *positionRepositoryGorm) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		Delete(&domain.Position{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrPositionNotFound
	}
	return nil
}

// ExistsByCode reports whether the company already uses this position code.
func (r *positionRepositoryGorm) ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).Model(&domain.Position{}).
		Where("company_id = ? AND code = ?", companyID, code)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountEmployees counts the live employees holding this position.
func (r *positionRepositoryGorm) CountEmployees(ctx context.Context, companyID, positionID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Employee{}).
		Where("company_id = ? AND position_id = ?", companyID, positionID).
		Count(&count).Error
	return count, err
}
