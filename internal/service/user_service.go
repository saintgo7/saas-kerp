package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// UserService errors
var (
	ErrUserEmailExists       = errors.New("email already exists")
	ErrUserCannotDeleteSelf  = errors.New("cannot delete your own account")
	ErrUserCannotDeactivateSelf = errors.New("cannot deactivate your own account")
	ErrInvalidCurrentPassword = errors.New("invalid current password")
)

// UserService defines the interface for user business logic
type UserService interface {
	// CRUD operations
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Query operations
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.User, error)
	List(ctx context.Context, filter repository.UserFilter) ([]domain.User, int64, error)

	// Password management
	ChangePassword(ctx context.Context, companyID, userID uuid.UUID, currentPassword, newPassword string) error
	ResetPassword(ctx context.Context, companyID, userID uuid.UUID, newPassword string) error

	// Status management
	Activate(ctx context.Context, companyID, id uuid.UUID) error
	Deactivate(ctx context.Context, companyID, id uuid.UUID) error

	// Statistics
	GetStats(ctx context.Context, companyID uuid.UUID) (*UserStats, error)
}

// UserStats represents user statistics
type UserStats struct {
	TotalCount    int64
	ActiveCount   int64
	InactiveCount int64
	LockedCount   int64
	AdminCount    int64
	UserCount     int64
	ViewerCount   int64
}

// userServiceImpl implements UserService
type userServiceImpl struct {
	repo repository.UserRepository
}

// NewUserService creates a new user service
func NewUserService(repo repository.UserRepository) UserService {
	return &userServiceImpl{repo: repo}
}

func (s *userServiceImpl) Create(ctx context.Context, user *domain.User) error {
	// Check if email already exists
	exists, err := s.repo.ExistsByEmail(ctx, user.CompanyID, user.Email, nil)
	if err != nil {
		return err
	}
	if exists {
		return ErrUserEmailExists
	}

	return s.repo.Create(ctx, user)
}

func (s *userServiceImpl) Update(ctx context.Context, user *domain.User) error {
	// Check if email already exists (excluding current user)
	exists, err := s.repo.ExistsByEmail(ctx, user.CompanyID, user.Email, &user.ID)
	if err != nil {
		return err
	}
	if exists {
		return ErrUserEmailExists
	}

	return s.repo.Update(ctx, user)
}

func (s *userServiceImpl) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	// Note: Caller should check if user is trying to delete themselves
	return s.repo.Delete(ctx, companyID, id)
}

func (s *userServiceImpl) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.User, error) {
	return s.repo.FindByID(ctx, companyID, id)
}

func (s *userServiceImpl) List(ctx context.Context, filter repository.UserFilter) ([]domain.User, int64, error) {
	return s.repo.FindAll(ctx, filter)
}

func (s *userServiceImpl) ChangePassword(ctx context.Context, companyID, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.repo.FindByID(ctx, companyID, userID)
	if err != nil {
		return err
	}

	// Verify current password
	if !user.CheckPassword(currentPassword) {
		return ErrInvalidCurrentPassword
	}

	// Set new password
	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	return s.repo.Update(ctx, user)
}

func (s *userServiceImpl) ResetPassword(ctx context.Context, companyID, userID uuid.UUID, newPassword string) error {
	user, err := s.repo.FindByID(ctx, companyID, userID)
	if err != nil {
		return err
	}

	// Set new password without verifying current
	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	return s.repo.Update(ctx, user)
}

func (s *userServiceImpl) Activate(ctx context.Context, companyID, id uuid.UUID) error {
	user, err := s.repo.FindByID(ctx, companyID, id)
	if err != nil {
		return err
	}

	user.Status = domain.UserStatusActive
	return s.repo.Update(ctx, user)
}

func (s *userServiceImpl) Deactivate(ctx context.Context, companyID, id uuid.UUID) error {
	user, err := s.repo.FindByID(ctx, companyID, id)
	if err != nil {
		return err
	}

	user.Status = domain.UserStatusInactive
	return s.repo.Update(ctx, user)
}

func (s *userServiceImpl) GetStats(ctx context.Context, companyID uuid.UUID) (*UserStats, error) {
	stats := &UserStats{}

	// Get all users for counting (could be optimized with dedicated queries)
	users, total, err := s.repo.FindAll(ctx, repository.UserFilter{
		CompanyID: companyID,
		PageSize:  0, // No limit for stats
	})
	if err != nil {
		return nil, err
	}

	stats.TotalCount = total

	for _, user := range users {
		// Count by status
		switch user.Status {
		case domain.UserStatusActive:
			stats.ActiveCount++
		case domain.UserStatusInactive:
			stats.InactiveCount++
		case domain.UserStatusLocked:
			stats.LockedCount++
		}

		// Count by role
		switch user.Role {
		case domain.UserRoleAdmin:
			stats.AdminCount++
		case domain.UserRoleUser:
			stats.UserCount++
		case domain.UserRoleViewer:
			stats.ViewerCount++
		}
	}

	return stats, nil
}
