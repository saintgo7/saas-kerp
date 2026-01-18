package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Project errors
var (
	ErrProjectNotFound   = errors.New("project not found")
	ErrProjectCodeExists = errors.New("project code already exists")
	ErrProjectCodeEmpty  = errors.New("project code is required")
	ErrProjectNameEmpty  = errors.New("project name is required")
	ErrProjectInUse      = errors.New("project is in use and cannot be deleted")
)

// ProjectStatus represents the status of a project
type ProjectStatus string

const (
	ProjectStatusPlanning  ProjectStatus = "planning"
	ProjectStatusActive    ProjectStatus = "active"
	ProjectStatusOnHold    ProjectStatus = "on_hold"
	ProjectStatusCompleted ProjectStatus = "completed"
	ProjectStatusCancelled ProjectStatus = "cancelled"
)

// IsValid checks if the project status is valid
func (s ProjectStatus) IsValid() bool {
	switch s {
	case ProjectStatusPlanning, ProjectStatusActive, ProjectStatusOnHold, ProjectStatusCompleted, ProjectStatusCancelled:
		return true
	}
	return false
}

// Project represents a project for cost tracking
type Project struct {
	TenantModel

	// Basic info
	Code        string `gorm:"type:varchar(50);not null;uniqueIndex:idx_projects_company_code" json:"code"`
	Name        string `gorm:"type:varchar(200);not null" json:"name"`
	Description string `gorm:"type:varchar(1000)" json:"description,omitempty"`

	// Manager
	ManagerID *uuid.UUID `gorm:"type:uuid" json:"manager_id,omitempty"`

	// Status
	Status    ProjectStatus `gorm:"type:varchar(20);default:'planning'" json:"status"`
	StartDate *time.Time    `json:"start_date,omitempty"`
	EndDate   *time.Time    `json:"end_date,omitempty"`

	// Budget
	Budget     float64 `gorm:"type:decimal(18,2);default:0" json:"budget"`
	ActualCost float64 `gorm:"type:decimal(18,2);default:0" json:"actual_cost"`

	// Status
	IsActive bool `gorm:"default:true" json:"is_active"`
}

// TableName specifies the table name for GORM
func (Project) TableName() string {
	return "projects"
}

// NewProject creates a new project
func NewProject(companyID uuid.UUID, code, name string) (*Project, error) {
	if code == "" {
		return nil, ErrProjectCodeEmpty
	}
	if name == "" {
		return nil, ErrProjectNameEmpty
	}

	return &Project{
		TenantModel: TenantModel{
			CompanyID: companyID,
		},
		Code:     code,
		Name:     name,
		Status:   ProjectStatusPlanning,
		IsActive: true,
	}, nil
}

// Activate activates the project
func (p *Project) Activate() {
	p.Status = ProjectStatusActive
	now := time.Now()
	if p.StartDate == nil {
		p.StartDate = &now
	}
}

// Complete marks the project as completed
func (p *Project) Complete() {
	p.Status = ProjectStatusCompleted
	now := time.Now()
	p.EndDate = &now
}

// Cancel cancels the project
func (p *Project) Cancel() {
	p.Status = ProjectStatusCancelled
	now := time.Now()
	p.EndDate = &now
}

// PutOnHold puts the project on hold
func (p *Project) PutOnHold() {
	p.Status = ProjectStatusOnHold
}

// IsCompleted returns true if the project is completed
func (p *Project) IsCompleted() bool {
	return p.Status == ProjectStatusCompleted
}

// IsCancelled returns true if the project is cancelled
func (p *Project) IsCancelled() bool {
	return p.Status == ProjectStatusCancelled
}

// BudgetRemaining returns the remaining budget
func (p *Project) BudgetRemaining() float64 {
	return p.Budget - p.ActualCost
}

// BudgetUsedPercent returns the percentage of budget used
func (p *Project) BudgetUsedPercent() float64 {
	if p.Budget == 0 {
		return 0
	}
	return (p.ActualCost / p.Budget) * 100
}

// CostCenter represents a cost center for expense tracking
type CostCenter struct {
	TenantModel

	// Basic info
	Code        string `gorm:"type:varchar(20);not null" json:"code"`
	Name        string `gorm:"type:varchar(100);not null" json:"name"`
	Description string `gorm:"type:varchar(500)" json:"description,omitempty"`

	// Hierarchy
	ParentID *uuid.UUID   `gorm:"type:uuid" json:"parent_id,omitempty"`
	Parent   *CostCenter  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []CostCenter `gorm:"foreignKey:ParentID" json:"children,omitempty"`

	// Status
	IsActive bool `gorm:"default:true" json:"is_active"`
}

// TableName specifies the table name for GORM
func (CostCenter) TableName() string {
	return "cost_centers"
}
