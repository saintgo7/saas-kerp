package dto

import (
	"github.com/google/uuid"
	"github.com/saintgo7/saas-kerp/internal/domain"
)

// UserResponse represents a user in API responses
type UserResponse struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	Name        string  `json:"name"`
	Role        string  `json:"role"`
	Status      string  `json:"status"`
	LastLoginAt *string `json:"last_login_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// FromUser converts domain.User to UserResponse
func FromUser(user *domain.User) UserResponse {
	resp := UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		Role:      string(user.Role),
		Status:    string(user.Status),
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if user.LastLoginAt != nil {
		formatted := user.LastLoginAt.Format("2006-01-02T15:04:05Z07:00")
		resp.LastLoginAt = &formatted
	}

	return resp
}

// FromUsers converts []domain.User to []UserResponse
func FromUsers(users []domain.User) []UserResponse {
	responses := make([]UserResponse, len(users))
	for i := range users {
		responses[i] = FromUser(&users[i])
	}
	return responses
}

// CreateUserRequest represents the request to create a user
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=100"`
	Name     string `json:"name" binding:"required,max=100"`
	Role     string `json:"role" binding:"omitempty,oneof=admin user viewer"`
}

// ToUser converts CreateUserRequest to domain.User
func (r *CreateUserRequest) ToUser(companyID uuid.UUID) (*domain.User, error) {
	role := domain.UserRole(r.Role)
	if !role.IsValid() {
		role = domain.UserRoleUser
	}
	return domain.NewUser(companyID, r.Email, r.Password, r.Name, role)
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Email string `json:"email" binding:"required,email,max=255"`
	Name  string `json:"name" binding:"required,max=100"`
	Role  string `json:"role" binding:"omitempty,oneof=admin user viewer"`
}

// ApplyTo applies the update to an existing user
func (r *UpdateUserRequest) ApplyTo(user *domain.User) {
	user.Email = r.Email
	user.Name = r.Name
	if r.Role != "" {
		role := domain.UserRole(r.Role)
		if role.IsValid() {
			user.Role = role
		}
	}
}

// ChangePasswordRequest represents the request to change a user's password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=100"`
}

// UserStatsResponse represents user statistics
type UserStatsResponse struct {
	TotalCount    int64 `json:"total_count"`
	ActiveCount   int64 `json:"active_count"`
	InactiveCount int64 `json:"inactive_count"`
	LockedCount   int64 `json:"locked_count"`
	AdminCount    int64 `json:"admin_count"`
	UserCount     int64 `json:"user_count"`
	ViewerCount   int64 `json:"viewer_count"`
}
