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

// authUserRow mirrors the RETURNS TABLE shape of auth_find_users_by_email().
// It is a separate type from domain.User because the function projects a fixed
// column list; scanning straight into domain.User would make the query depend
// on the ORM's model metadata for a result set the database, not GORM, defines.
type authUserRow struct {
	ID           uuid.UUID
	CompanyID    uuid.UUID
	Email        string
	PasswordHash string
	Name         string
	Role         string
	Status       string
}

// FindAuthCandidatesByEmail returns every tenant's user with this address.
//
// The lookup goes through auth_find_users_by_email(), a SECURITY DEFINER
// function owned by the NOLOGIN kerp_auth role (000021_auth_bootstrap). A plain
// SELECT cannot be used: /auth/login runs before the tenant is known, so
// app.current_tenant is unset, and once the application role loses BYPASSRLS
// (000016) the users policy matches zero rows and login breaks. The function is
// the whole privileged surface - "look up a user by e-mail" - and the tables
// keep RLS + FORCE.
//
// The function name is left unqualified on purpose: it is created in
// current_schema() by the migration, and both kerp_app and the database default
// put that schema first on search_path, exactly like the unqualified table names
// migrations 000001-000021 use.
func (r *userRepositoryGorm) FindAuthCandidatesByEmail(ctx context.Context, email string) ([]domain.User, error) {
	if email == "" {
		// An empty address can never match a NOT NULL, non-empty column; skip
		// the round trip rather than let a caller bug reach the database.
		return nil, nil
	}

	var rows []authUserRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT id, company_id, email, password_hash, name, role, status
		       FROM auth_find_users_by_email(?)`, email).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	users := make([]domain.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, domain.User{
			TenantModel: domain.TenantModel{
				BaseModel: domain.BaseModel{ID: row.ID},
				CompanyID: row.CompanyID,
			},
			Email:        row.Email,
			PasswordHash: row.PasswordHash,
			Name:         row.Name,
			Role:         domain.UserRole(row.Role),
			Status:       domain.UserStatus(row.Status),
		})
	}
	return users, nil
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

	// Apply sorting.
	//
	// GORM splices an Order() string into the SQL verbatim, so a caller-supplied
	// column name must never reach it. The sibling repositories (accounts,
	// vouchers) already whitelist; this one interpolated filter.SortBy
	// directly. UserFilter.SortBy happens not to be populated by any handler
	// today, which is the only reason it was not exploitable - wiring up a
	// sort_by query parameter would have made it so.
	sortBy := "created_at"
	if column, ok := userSortColumns[filter.SortBy]; ok {
		sortBy = column
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

// userSortColumns maps the sort keys the API accepts onto real column names.
// Anything not in this map falls back to the default ordering.
var userSortColumns = map[string]string{
	"created_at":    "created_at",
	"updated_at":    "updated_at",
	"email":         "email",
	"name":          "name",
	"role":          "role",
	"status":        "status",
	"last_login_at": "last_login_at",
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

func (r *userRepositoryGorm) UpdateLastLogin(ctx context.Context, companyID, userID uuid.UUID) error {
	// company_id is in the predicate as well as the RLS policy on purpose; see
	// the interface comment.
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("company_id = ? AND id = ?", companyID, userID).
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

// authRefreshTokenRow mirrors the RETURNS TABLE shape of
// auth_find_refresh_token().
type authRefreshTokenRow struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CompanyID uuid.UUID
	ExpiresAt time.Time
	Revoked   bool
}

// FindByTokenForAuth resolves a refresh token before the tenant is known.
//
// Same reasoning as FindAuthCandidatesByEmail: /auth/refresh has no tenant
// context until this lookup answers, so it goes through the SECURITY DEFINER
// auth_find_refresh_token() rather than a policy-filtered SELECT. The function
// joins users to return the owning company_id, which the caller must install as
// app.current_tenant before it touches anything else.
func (r *refreshTokenRepositoryGorm) FindByTokenForAuth(ctx context.Context, token string) (*AuthRefreshToken, error) {
	if token == "" {
		return nil, domain.ErrRefreshTokenNotFound
	}

	var rows []authRefreshTokenRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT id, user_id, company_id, expires_at, revoked
		       FROM auth_find_refresh_token(?)`, token).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, domain.ErrRefreshTokenNotFound
	}

	row := rows[0]
	return &AuthRefreshToken{
		ID:        row.ID,
		UserID:    row.UserID,
		CompanyID: row.CompanyID,
		ExpiresAt: row.ExpiresAt,
		Revoked:   row.Revoked,
	}, nil
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
