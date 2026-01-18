package domain

import (
	"errors"

	"github.com/google/uuid"
)

// Role errors
var (
	ErrRoleNotFound   = errors.New("role not found")
	ErrRoleCodeExists = errors.New("role code already exists")
	ErrRoleNameExists = errors.New("role name already exists")
	ErrRoleInUse      = errors.New("role is in use and cannot be deleted")
	ErrRoleCodeEmpty  = errors.New("role code is required")
	ErrRoleNameEmpty  = errors.New("role name is required")
)

// Permission represents a permission in the system
type Permission struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Module      string `json:"module"`
}

// Role represents a role in the system
type Role struct {
	TenantModel
	Code        string       `gorm:"type:varchar(50);not null;uniqueIndex:idx_roles_company_code" json:"code"`
	Name        string       `gorm:"type:varchar(100);not null" json:"name"`
	Description string       `gorm:"type:varchar(500)" json:"description,omitempty"`
	Permissions []Permission `gorm:"type:jsonb;serializer:json" json:"permissions"`
	IsSystem    bool         `gorm:"default:false" json:"is_system"`
	IsActive    bool         `gorm:"default:true" json:"is_active"`
}

// TableName returns the table name for Role
func (Role) TableName() string {
	return "kerp.roles"
}

// NewRole creates a new role
func NewRole(companyID uuid.UUID, code, name, description string) (*Role, error) {
	if code == "" {
		return nil, ErrRoleCodeEmpty
	}
	if name == "" {
		return nil, ErrRoleNameEmpty
	}

	return &Role{
		TenantModel: TenantModel{
			CompanyID: companyID,
		},
		Code:        code,
		Name:        name,
		Description: description,
		Permissions: []Permission{},
		IsSystem:    false,
		IsActive:    true,
	}, nil
}

// AddPermission adds a permission to the role
func (r *Role) AddPermission(perm Permission) {
	// Check if permission already exists
	for _, p := range r.Permissions {
		if p.Code == perm.Code {
			return
		}
	}
	r.Permissions = append(r.Permissions, perm)
}

// RemovePermission removes a permission from the role
func (r *Role) RemovePermission(code string) {
	for i, p := range r.Permissions {
		if p.Code == code {
			r.Permissions = append(r.Permissions[:i], r.Permissions[i+1:]...)
			return
		}
	}
}

// HasPermission checks if the role has a specific permission
func (r *Role) HasPermission(code string) bool {
	for _, p := range r.Permissions {
		if p.Code == code {
			return true
		}
	}
	return false
}

// SetPermissions replaces all permissions
func (r *Role) SetPermissions(perms []Permission) {
	r.Permissions = perms
}
