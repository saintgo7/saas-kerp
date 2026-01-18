package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/saintgo7/saas-kerp/internal/domain"
)

// ProjectResponse represents a project in API responses
type ProjectResponse struct {
	ID              string   `json:"id"`
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	ManagerID       string   `json:"manager_id,omitempty"`
	Status          string   `json:"status"`
	StartDate       *string  `json:"start_date,omitempty"`
	EndDate         *string  `json:"end_date,omitempty"`
	Budget          float64  `json:"budget"`
	ActualCost      float64  `json:"actual_cost"`
	BudgetRemaining float64  `json:"budget_remaining"`
	BudgetUsedPct   float64  `json:"budget_used_percent"`
	IsActive        bool     `json:"is_active"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

// FromProject converts domain.Project to ProjectResponse
func FromProject(project *domain.Project) ProjectResponse {
	resp := ProjectResponse{
		ID:              project.ID.String(),
		Code:            project.Code,
		Name:            project.Name,
		Description:     project.Description,
		Status:          string(project.Status),
		Budget:          project.Budget,
		ActualCost:      project.ActualCost,
		BudgetRemaining: project.BudgetRemaining(),
		BudgetUsedPct:   project.BudgetUsedPercent(),
		IsActive:        project.IsActive,
		CreatedAt:       project.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       project.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if project.ManagerID != nil {
		resp.ManagerID = project.ManagerID.String()
	}
	if project.StartDate != nil {
		formatted := project.StartDate.Format("2006-01-02")
		resp.StartDate = &formatted
	}
	if project.EndDate != nil {
		formatted := project.EndDate.Format("2006-01-02")
		resp.EndDate = &formatted
	}

	return resp
}

// FromProjects converts []domain.Project to []ProjectResponse
func FromProjects(projects []domain.Project) []ProjectResponse {
	responses := make([]ProjectResponse, len(projects))
	for i := range projects {
		responses[i] = FromProject(&projects[i])
	}
	return responses
}

// CreateProjectRequest represents the request to create a project
type CreateProjectRequest struct {
	Code        string  `json:"code" binding:"required,max=50"`
	Name        string  `json:"name" binding:"required,max=200"`
	Description string  `json:"description,omitempty" binding:"max=1000"`
	ManagerID   string  `json:"manager_id,omitempty" binding:"omitempty,uuid"`
	Status      string  `json:"status,omitempty" binding:"omitempty,oneof=planning active on_hold completed cancelled"`
	StartDate   string  `json:"start_date,omitempty"` // Format: 2006-01-02
	EndDate     string  `json:"end_date,omitempty"`   // Format: 2006-01-02
	Budget      float64 `json:"budget,omitempty" binding:"min=0"`
}

// ToProject converts CreateProjectRequest to domain.Project
func (r *CreateProjectRequest) ToProject(companyID uuid.UUID) (*domain.Project, error) {
	project, err := domain.NewProject(companyID, r.Code, r.Name)
	if err != nil {
		return nil, err
	}

	project.Description = r.Description
	project.Budget = r.Budget

	if r.ManagerID != "" {
		managerID, err := uuid.Parse(r.ManagerID)
		if err == nil {
			project.ManagerID = &managerID
		}
	}

	if r.Status != "" {
		status := domain.ProjectStatus(r.Status)
		if status.IsValid() {
			project.Status = status
		}
	}

	if r.StartDate != "" {
		if t, err := time.Parse("2006-01-02", r.StartDate); err == nil {
			project.StartDate = &t
		}
	}

	if r.EndDate != "" {
		if t, err := time.Parse("2006-01-02", r.EndDate); err == nil {
			project.EndDate = &t
		}
	}

	return project, nil
}

// UpdateProjectRequest represents the request to update a project
type UpdateProjectRequest struct {
	Code        string   `json:"code" binding:"required,max=50"`
	Name        string   `json:"name" binding:"required,max=200"`
	Description string   `json:"description,omitempty" binding:"max=1000"`
	ManagerID   string   `json:"manager_id,omitempty" binding:"omitempty,uuid"`
	Status      string   `json:"status,omitempty" binding:"omitempty,oneof=planning active on_hold completed cancelled"`
	StartDate   string   `json:"start_date,omitempty"`
	EndDate     string   `json:"end_date,omitempty"`
	Budget      *float64 `json:"budget,omitempty" binding:"omitempty,min=0"`
	IsActive    *bool    `json:"is_active,omitempty"`
}

// ApplyTo applies the update to an existing project
func (r *UpdateProjectRequest) ApplyTo(project *domain.Project) {
	project.Code = r.Code
	project.Name = r.Name
	project.Description = r.Description

	if r.ManagerID != "" {
		managerID, err := uuid.Parse(r.ManagerID)
		if err == nil {
			project.ManagerID = &managerID
		}
	} else {
		project.ManagerID = nil
	}

	if r.Status != "" {
		status := domain.ProjectStatus(r.Status)
		if status.IsValid() {
			project.Status = status
		}
	}

	if r.StartDate != "" {
		if t, err := time.Parse("2006-01-02", r.StartDate); err == nil {
			project.StartDate = &t
		}
	} else {
		project.StartDate = nil
	}

	if r.EndDate != "" {
		if t, err := time.Parse("2006-01-02", r.EndDate); err == nil {
			project.EndDate = &t
		}
	} else {
		project.EndDate = nil
	}

	if r.Budget != nil {
		project.Budget = *r.Budget
	}

	if r.IsActive != nil {
		project.IsActive = *r.IsActive
	}
}

// ProjectStatsResponse represents project statistics
type ProjectStatsResponse struct {
	TotalCount     int64   `json:"total_count"`
	ActiveCount    int64   `json:"active_count"`
	CompletedCount int64   `json:"completed_count"`
	OnHoldCount    int64   `json:"on_hold_count"`
	TotalBudget    float64 `json:"total_budget"`
	TotalActualCost float64 `json:"total_actual_cost"`
}
