package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// userRepositoryGorm implements UserRepository using GORM
type userRepositoryGorm struct {
	db *gorm.DB
}

// NewUserRepository creates a new GORM-based user repository
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepositoryGorm{db: db}
}

func (r *userRepositoryGorm) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepositoryGorm) Update(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepositoryGorm) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("company_id = ? AND id = ?", companyID, id).
		Delete(&domain.User{}).Error
}

func (r *userRepositoryGorm) FindByID(ctx context.Context, companyID, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND id = ?", companyID, id).
		First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepositoryGorm) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepositoryGorm) FindByEmailAndCompany(ctx context.Context, companyID uuid.UUID, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND email = ?", companyID, email).
		First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepositoryGorm) FindAll(ctx context.Context, filter UserFilter) ([]domain.User, int64, error) {
	var users []domain.User
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.User{}).
		Where("company_id = ?", filter.CompanyID)

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Role != nil {
		query = query.Where("role = ?", *filter.Role)
	}
	if filter.SearchTerm != "" {
		searchPattern := "%" + filter.SearchTerm + "%"
		query = query.Where("name ILIKE ? OR email ILIKE ?", searchPattern, searchPattern)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	if filter.SortDesc {
		sortBy += " DESC"
	}
	query = query.Order(sortBy)

	// Apply pagination
	if filter.PageSize > 0 {
		query = query.Limit(filter.PageSize)
		if filter.Page > 0 {
			query = query.Offset((filter.Page - 1) * filter.PageSize)
		}
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *userRepositoryGorm) ExistsByEmail(ctx context.Context, companyID uuid.UUID, email string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&domain.User{}).
		Where("company_id = ? AND email = ?", companyID, email)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *userRepositoryGorm) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		Update("last_login_at", time.Now()).Error
}

// refreshTokenRepositoryGorm implements RefreshTokenRepository using GORM
type refreshTokenRepositoryGorm struct {
	db *gorm.DB
}

// NewRefreshTokenRepository creates a new GORM-based refresh token repository
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepositoryGorm{db: db}
}

func (r *refreshTokenRepositoryGorm) Create(ctx context.Context, token *domain.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *refreshTokenRepositoryGorm) FindByToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	var rt domain.RefreshToken
	err := r.db.WithContext(ctx).
		Where("token = ? AND revoked = false", token).
		First(&rt).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrRefreshTokenNotFound
		}
		return nil, err
	}
	return &rt, nil
}

func (r *refreshTokenRepositoryGorm) RevokeByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&domain.RefreshToken{}).
		Where("user_id = ?", userID).
		Update("revoked", true).Error
}

func (r *refreshTokenRepositoryGorm) RevokeByToken(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).
		Model(&domain.RefreshToken{}).
		Where("token = ?", token).
		Update("revoked", true).Error
}

func (r *refreshTokenRepositoryGorm) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ? OR revoked = true", time.Now()).
		Delete(&domain.RefreshToken{}).Error
}
