package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserStatus represents the status of a user account
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusLocked   UserStatus = "locked"
)

// IsValid checks if the user status is valid
func (s UserStatus) IsValid() bool {
	switch s {
	case UserStatusActive, UserStatusInactive, UserStatusLocked:
		return true
	}
	return false
}

// UserRole represents the role of a user
type UserRole string

const (
	UserRoleAdmin  UserRole = "admin"
	UserRoleUser   UserRole = "user"
	UserRoleViewer UserRole = "viewer"
)

// IsValid checks if the user role is valid
func (r UserRole) IsValid() bool {
	switch r {
	case UserRoleAdmin, UserRoleUser, UserRoleViewer:
		return true
	}
	return false
}

// User errors
var (
	ErrUserNotFound          = errors.New("user not found")
	ErrUserEmailExists       = errors.New("email already exists")
	ErrInvalidCredentials    = errors.New("invalid email or password")
	ErrUserInactive          = errors.New("user account is inactive")
	ErrUserLocked            = errors.New("user account is locked")
	ErrInvalidUserStatus     = errors.New("invalid user status")
	ErrInvalidUserRole       = errors.New("invalid user role")
	ErrEmailRequired         = errors.New("email is required")
	ErrPasswordRequired      = errors.New("password is required")
	ErrNameRequired          = errors.New("name is required")
	ErrPasswordTooShort      = errors.New("password must be at least 8 characters")
	ErrRefreshTokenNotFound  = errors.New("refresh token not found")
	ErrRefreshTokenExpired   = errors.New("refresh token expired")
)

// User represents a user in the system
type User struct {
	TenantModel
	Email        string     `gorm:"type:varchar(255);not null;uniqueIndex:idx_users_company_email" json:"email"`
	PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"`
	Name         string     `gorm:"type:varchar(100);not null" json:"name"`
	Role         UserRole   `gorm:"type:varchar(50);default:'user'" json:"role"`
	Status       UserStatus `gorm:"type:varchar(20);default:'active'" json:"status"`
	LastLoginAt  *time.Time `gorm:"" json:"last_login_at,omitempty"`
}

// TableName returns the table name for User
func (User) TableName() string {
	return "kerp.users"
}

// NewUser creates a new user with the given details
func NewUser(companyID uuid.UUID, email, password, name string, role UserRole) (*User, error) {
	if email == "" {
		return nil, ErrEmailRequired
	}
	if password == "" {
		return nil, ErrPasswordRequired
	}
	if name == "" {
		return nil, ErrNameRequired
	}
	if len(password) < 8 {
		return nil, ErrPasswordTooShort
	}
	if !role.IsValid() {
		role = UserRoleUser
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return &User{
		TenantModel: TenantModel{
			CompanyID: companyID,
		},
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		Role:         role,
		Status:       UserStatusActive,
	}, nil
}

// CheckPassword verifies the password against the stored hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// SetPassword sets a new password for the user
func (u *User) SetPassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// IsActive returns true if the user account is active
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// UpdateLastLogin updates the last login timestamp
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
}

// GetRoles returns the user's roles as a string slice
func (u *User) GetRoles() []string {
	return []string{string(u.Role)}
}

// RefreshToken represents a refresh token for JWT authentication
type RefreshToken struct {
	BaseModel
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Token     string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	Revoked   bool      `gorm:"default:false" json:"revoked"`
}

// TableName returns the table name for RefreshToken
func (RefreshToken) TableName() string {
	return "kerp.refresh_tokens"
}

// IsExpired checks if the refresh token is expired
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// IsValid checks if the refresh token is valid (not expired and not revoked)
func (rt *RefreshToken) IsValid() bool {
	return !rt.IsExpired() && !rt.Revoked
}
