package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/saintgo7/saas-kerp/internal/domain"
)

// UserFilter defines filter options for user queries
type UserFilter struct {
	CompanyID  uuid.UUID
	Status     *domain.UserStatus
	Role       *domain.UserRole
	SearchTerm string
	Page       int
	PageSize   int
	SortBy     string
	SortDesc   bool
}

// UserRepository defines the interface for user data access
type UserRepository interface {
	// CRUD operations
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Query operations
	FindByID(ctx context.Context, companyID, id uuid.UUID) (*domain.User, error)
	FindByEmailAndCompany(ctx context.Context, companyID uuid.UUID, email string) (*domain.User, error)
	FindAll(ctx context.Context, filter UserFilter) ([]domain.User, int64, error)

	// FindAuthCandidatesByEmail returns EVERY tenant's user carrying this
	// address, for the pre-tenant authentication path only.
	//
	// It replaces the former FindByEmail, which was
	// `WHERE email = ? ... First()` - a global lookup with an implicit
	// `ORDER BY id LIMIT 1`. Users are unique per (company_id, email), not
	// globally, so that query silently picked one arbitrary tenant's row: a
	// second company holding the same address either locked the real owner out
	// forever or, when both happened to share a password, handed out a session
	// scoped to the WRONG company_id.
	//
	// Under RLS this cannot be an ordinary SELECT either - the tenant is not
	// known yet, so app.current_tenant is unset and every policy matches zero
	// rows. The GORM implementation therefore calls the SECURITY DEFINER
	// function auth_find_users_by_email() from
	// db/migrations/000021_auth_bootstrap.up.sql.
	//
	// Callers MUST treat the result as a candidate set: verify the credential
	// against each row and decide what to do when more than one verifies. An
	// empty slice (not an error) means "no such address".
	FindAuthCandidatesByEmail(ctx context.Context, email string) ([]domain.User, error)

	// Validation helpers
	ExistsByEmail(ctx context.Context, companyID uuid.UUID, email string, excludeID *uuid.UUID) (bool, error)

	// Login helpers.
	//
	// companyID is required even though RLS already constrains the UPDATE to
	// the current tenant: the predicate must not depend on the session variable
	// alone, so that a deployment running with tenant_guc disabled - or with a
	// superuser DSN, which bypasses RLS - still cannot touch another tenant's
	// row.
	UpdateLastLogin(ctx context.Context, companyID, userID uuid.UUID) error
}

// AuthRefreshToken is the pre-tenant view of a refresh token.
//
// It is not domain.RefreshToken because it carries the owning CompanyID, which
// the refresh_tokens table does not store: the caller needs it to set
// app.current_tenant before doing anything else, and it is resolved by joining
// users inside auth_find_refresh_token().
type AuthRefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CompanyID uuid.UUID
	ExpiresAt time.Time
	Revoked   bool
}

// IsExpired reports whether the token is past its expiry.
func (t *AuthRefreshToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// RefreshTokenRepository defines the interface for refresh token data access
type RefreshTokenRepository interface {
	// Create stores a new refresh token
	Create(ctx context.Context, token *domain.RefreshToken) error

	// FindByTokenForAuth retrieves a refresh token by its value, together with
	// the company_id of the user that owns it, WITHOUT requiring
	// app.current_tenant to be set - /auth/refresh has no tenant context until
	// this lookup answers.
	//
	// It returns revoked tokens as well: reuse of a revoked token is a signal
	// the caller may want to act on, and hiding it inside the query turned every
	// such attempt into an indistinguishable "not found".
	FindByTokenForAuth(ctx context.Context, token string) (*AuthRefreshToken, error)

	// RevokeByUserID revokes all refresh tokens for a user
	RevokeByUserID(ctx context.Context, userID uuid.UUID) error

	// RevokeByToken revokes a specific refresh token
	RevokeByToken(ctx context.Context, token string) error

	// DeleteExpired removes all expired tokens
	DeleteExpired(ctx context.Context) error
}
