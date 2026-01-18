package dto

import (
	"github.com/google/uuid"
	"github.com/saintgo7/saas-kerp/internal/domain"
)

// PermissionResponse represents a permission in API responses
type PermissionResponse struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Module      string `json:"module"`
}

// RoleResponse represents a role in API responses
type RoleResponse struct {
	ID          string               `json:"id"`
	Code        string               `json:"code"`
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Permissions []PermissionResponse `json:"permissions"`
	IsSystem    bool                 `json:"is_system"`
	IsActive    bool                 `json:"is_active"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
}

// FromRole converts domain.Role to RoleResponse
func FromRole(role *domain.Role) RoleResponse {
	perms := make([]PermissionResponse, len(role.Permissions))
	for i, p := range role.Permissions {
		perms[i] = PermissionResponse{
			Code:        p.Code,
			Name:        p.Name,
			Description: p.Description,
			Module:      p.Module,
		}
	}

	return RoleResponse{
		ID:          role.ID.String(),
		Code:        role.Code,
		Name:        role.Name,
		Description: role.Description,
		Permissions: perms,
		IsSystem:    role.IsSystem,
		IsActive:    role.IsActive,
		CreatedAt:   role.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   role.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// FromRoles converts []domain.Role to []RoleResponse
func FromRoles(roles []domain.Role) []RoleResponse {
	responses := make([]RoleResponse, len(roles))
	for i := range roles {
		responses[i] = FromRole(&roles[i])
	}
	return responses
}

// CreateRoleRequest represents the request to create a role
type CreateRoleRequest struct {
	Code        string `json:"code" binding:"required,max=50"`
	Name        string `json:"name" binding:"required,max=100"`
	Description string `json:"description,omitempty" binding:"max=500"`
}

// ToRole converts CreateRoleRequest to domain.Role
func (r *CreateRoleRequest) ToRole(companyID uuid.UUID) (*domain.Role, error) {
	return domain.NewRole(companyID, r.Code, r.Name, r.Description)
}

// UpdateRoleRequest represents the request to update a role
type UpdateRoleRequest struct {
	Code        string `json:"code" binding:"required,max=50"`
	Name        string `json:"name" binding:"required,max=100"`
	Description string `json:"description,omitempty" binding:"max=500"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

// ApplyTo applies the update to an existing role
func (r *UpdateRoleRequest) ApplyTo(role *domain.Role) {
	role.Code = r.Code
	role.Name = r.Name
	role.Description = r.Description
	if r.IsActive != nil {
		role.IsActive = *r.IsActive
	}
}

// SetPermissionsRequest represents the request to set role permissions
type SetPermissionsRequest struct {
	Permissions []PermissionRequest `json:"permissions" binding:"required"`
}

// PermissionRequest represents a permission in request
type PermissionRequest struct {
	Code        string `json:"code" binding:"required,max=100"`
	Name        string `json:"name" binding:"required,max=100"`
	Description string `json:"description,omitempty" binding:"max=500"`
	Module      string `json:"module" binding:"required,max=50"`
}

// ToPermissions converts []PermissionRequest to []domain.Permission
func (r *SetPermissionsRequest) ToPermissions() []domain.Permission {
	perms := make([]domain.Permission, len(r.Permissions))
	for i, p := range r.Permissions {
		perms[i] = domain.Permission{
			Code:        p.Code,
			Name:        p.Name,
			Description: p.Description,
			Module:      p.Module,
		}
	}
	return perms
}
