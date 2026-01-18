package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// companyRepositoryGorm implements CompanyRepository using GORM
type companyRepositoryGorm struct {
	db *gorm.DB
}

// NewCompanyRepository creates a new GORM-based company repository
func NewCompanyRepository(db *gorm.DB) CompanyRepository {
	return &companyRepositoryGorm{db: db}
}

func (r *companyRepositoryGorm) Create(ctx context.Context, company *domain.Company) error {
	return r.db.WithContext(ctx).Create(company).Error
}

func (r *companyRepositoryGorm) Update(ctx context.Context, company *domain.Company) error {
	return r.db.WithContext(ctx).Save(company).Error
}

func (r *companyRepositoryGorm) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&domain.Company{}).Error
}

func (r *companyRepositoryGorm) FindByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	var company domain.Company
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&company).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrCompanyNotFound
		}
		return nil, err
	}
	return &company, nil
}

func (r *companyRepositoryGorm) FindByCode(ctx context.Context, code string) (*domain.Company, error) {
	var company domain.Company
	err := r.db.WithContext(ctx).
		Where("code = ?", code).
		First(&company).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrCompanyNotFound
		}
		return nil, err
	}
	return &company, nil
}

func (r *companyRepositoryGorm) FindAll(ctx context.Context) ([]domain.Company, error) {
	var companies []domain.Company
	err := r.db.WithContext(ctx).
		Order("name ASC").
		Find(&companies).Error
	if err != nil {
		return nil, err
	}
	return companies, nil
}

func (r *companyRepositoryGorm) ExistsByCode(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&domain.Company{}).
		Where("code = ?", code)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
