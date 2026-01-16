package repository

import (
	"context"

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
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByEmailAndCompany(ctx context.Context, companyID uuid.UUID, email string) (*domain.User, error)
	FindAll(ctx context.Context, filter UserFilter) ([]domain.User, int64, error)

	// Validation helpers
	ExistsByEmail(ctx context.Context, companyID uuid.UUID, email string, excludeID *uuid.UUID) (bool, error)

	// Login helpers
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error
}

// RefreshTokenRepository defines the interface for refresh token data access
type RefreshTokenRepository interface {
	// Create stores a new refresh token
	Create(ctx context.Context, token *domain.RefreshToken) error

	// FindByToken retrieves a refresh token by its token string
	FindByToken(ctx context.Context, token string) (*domain.RefreshToken, error)

	// RevokeByUserID revokes all refresh tokens for a user
	RevokeByUserID(ctx context.Context, userID uuid.UUID) error

	// RevokeByToken revokes a specific refresh token
	RevokeByToken(ctx context.Context, token string) error

	// DeleteExpired removes all expired tokens
	DeleteExpired(ctx context.Context) error
}
