package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// projectRepositoryGorm implements ProjectRepository using GORM
type projectRepositoryGorm struct {
	db *gorm.DB
}

// NewProjectRepository creates a new GORM-based project repository
func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepositoryGorm{db: db}
}

func (r *projectRepositoryGorm) Create(ctx context.Context, project *domain.Project) error {
	return r.db.WithContext(ctx).Create(project).Error
}

func (r *projectRepositoryGorm) Update(ctx context.Context, project *domain.Project) error {
	return r.db.WithContext(ctx).Save(project).Error
}

func (r *projectRepositoryGorm) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("company_id = ? AND id = ?", companyID, id).
		Delete(&domain.Project{}).Error
}

func (r *projectRepositoryGorm) FindByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Project, error) {
	var project domain.Project
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND id = ?", companyID, id).
		First(&project).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrProjectNotFound
		}
		return nil, err
	}
	return &project, nil
}

func (r *projectRepositoryGorm) FindByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Project, error) {
	var project domain.Project
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND code = ?", companyID, code).
		First(&project).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrProjectNotFound
		}
		return nil, err
	}
	return &project, nil
}

func (r *projectRepositoryGorm) FindAll(ctx context.Context, filter ProjectFilter) ([]domain.Project, int64, error) {
	var projects []domain.Project
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Project{}).
		Where("company_id = ?", filter.CompanyID)

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.ManagerID != nil {
		query = query.Where("manager_id = ?", *filter.ManagerID)
	}
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

	if err := query.Find(&projects).Error; err != nil {
		return nil, 0, err
	}

	return projects, total, nil
}

func (r *projectRepositoryGorm) ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&domain.Project{}).
		Where("company_id = ? AND code = ?", companyID, code)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *projectRepositoryGorm) IsInUse(ctx context.Context, companyID, projectID uuid.UUID) (bool, error) {
	// Check if project is referenced in vouchers or other tables
	// For now, return false as we don't have project references in vouchers yet
	return false, nil
}

func (r *projectRepositoryGorm) GetStats(ctx context.Context, companyID uuid.UUID) (*ProjectStats, error) {
	stats := &ProjectStats{}

	// Total count
	var total int64
	if err := r.db.WithContext(ctx).Model(&domain.Project{}).
		Where("company_id = ?", companyID).
		Count(&total).Error; err != nil {
		return nil, err
	}
	stats.TotalCount = total

	// Active count
	var active int64
	if err := r.db.WithContext(ctx).Model(&domain.Project{}).
		Where("company_id = ? AND status = ?", companyID, domain.ProjectStatusActive).
		Count(&active).Error; err != nil {
		return nil, err
	}
	stats.ActiveCount = active

	// Completed count
	var completed int64
	if err := r.db.WithContext(ctx).Model(&domain.Project{}).
		Where("company_id = ? AND status = ?", companyID, domain.ProjectStatusCompleted).
		Count(&completed).Error; err != nil {
		return nil, err
	}
	stats.CompletedCount = completed

	// On hold count
	var onHold int64
	if err := r.db.WithContext(ctx).Model(&domain.Project{}).
		Where("company_id = ? AND status = ?", companyID, domain.ProjectStatusOnHold).
		Count(&onHold).Error; err != nil {
		return nil, err
	}
	stats.OnHoldCount = onHold

	// Budget totals
	type BudgetResult struct {
		TotalBudget     float64
		TotalActualCost float64
	}
	var budgetResult BudgetResult
	if err := r.db.WithContext(ctx).Model(&domain.Project{}).
		Where("company_id = ?", companyID).
		Select("COALESCE(SUM(budget), 0) as total_budget, COALESCE(SUM(actual_cost), 0) as total_actual_cost").
		Scan(&budgetResult).Error; err != nil {
		return nil, err
	}
	stats.TotalBudget = budgetResult.TotalBudget
	stats.TotalActualCost = budgetResult.TotalActualCost

	return stats, nil
}
