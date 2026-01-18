package domain

import (
	"errors"

	"github.com/google/uuid"
)

// Department errors
var (
	ErrDepartmentNotFound   = errors.New("department not found")
	ErrDepartmentCodeExists = errors.New("department code already exists")
	ErrDepartmentHasChildren = errors.New("department has children and cannot be deleted")
)

// Department represents an organizational department
type Department struct {
	TenantModel

	// Basic info
	Code   string `gorm:"type:varchar(20);not null" json:"code"`
	Name   string `gorm:"type:varchar(100);not null" json:"name"`
	NameEn string `gorm:"type:varchar(100)" json:"name_en,omitempty"`

	// Hierarchy
	ParentID *uuid.UUID   `gorm:"type:uuid" json:"parent_id,omitempty"`
	Parent   *Department  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Department `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Level    int          `gorm:"not null;default:1" json:"level"`
	Path     string       `gorm:"type:ltree" json:"path,omitempty"`

	// Manager
	ManagerID *uuid.UUID `gorm:"type:uuid" json:"manager_id,omitempty"`

	// Status
	IsActive bool `gorm:"default:true" json:"is_active"`
}

// TableName specifies the table name for GORM
func (Department) TableName() string {
	return "departments"
}
