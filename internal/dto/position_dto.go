package dto

import (
	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// PositionResponse represents a position in API responses.
type PositionResponse struct {
	ID        string   `json:"id"`
	Code      string   `json:"code"`
	Name      string   `json:"name"`
	NameEn    string   `json:"name_en,omitempty"`
	RankLevel int      `json:"rank_level"`
	MinSalary *float64 `json:"min_salary,omitempty"`
	MaxSalary *float64 `json:"max_salary,omitempty"`
	IsActive  bool     `json:"is_active"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// FromPosition converts domain.Position to PositionResponse.
func FromPosition(position *domain.Position) PositionResponse {
	return PositionResponse{
		ID:        position.ID.String(),
		Code:      position.Code,
		Name:      position.Name,
		NameEn:    position.NameEn,
		RankLevel: position.RankLevel,
		MinSalary: position.MinSalary,
		MaxSalary: position.MaxSalary,
		IsActive:  position.IsActive,
		CreatedAt: position.CreatedAt.Format(hrTimeFormat),
		UpdatedAt: position.UpdatedAt.Format(hrTimeFormat),
	}
}

// FromPositions converts []domain.Position to []PositionResponse.
func FromPositions(positions []domain.Position) []PositionResponse {
	responses := make([]PositionResponse, len(positions))
	for i := range positions {
		responses[i] = FromPosition(&positions[i])
	}
	return responses
}

// CreatePositionRequest represents the request to create a position.
type CreatePositionRequest struct {
	Code      string   `json:"code" binding:"required,max=20"`
	Name      string   `json:"name" binding:"required,max=100"`
	NameEn    string   `json:"name_en,omitempty" binding:"max=100"`
	RankLevel int      `json:"rank_level,omitempty" binding:"omitempty,min=0"`
	MinSalary *float64 `json:"min_salary,omitempty" binding:"omitempty,min=0"`
	MaxSalary *float64 `json:"max_salary,omitempty" binding:"omitempty,min=0"`
	IsActive  *bool    `json:"is_active,omitempty"`
}

// ToPosition converts the request to a domain.Position.
func (r *CreatePositionRequest) ToPosition(companyID uuid.UUID) *domain.Position {
	position := &domain.Position{
		TenantModel: domain.TenantModel{CompanyID: companyID},
		Code:        r.Code,
		Name:        r.Name,
		NameEn:      r.NameEn,
		RankLevel:   r.RankLevel,
		MinSalary:   r.MinSalary,
		MaxSalary:   r.MaxSalary,
		IsActive:    true,
	}
	if r.IsActive != nil {
		position.IsActive = *r.IsActive
	}
	return position
}

// UpdatePositionRequest represents the request to update a position.
type UpdatePositionRequest struct {
	Code      string   `json:"code" binding:"required,max=20"`
	Name      string   `json:"name" binding:"required,max=100"`
	NameEn    string   `json:"name_en,omitempty" binding:"max=100"`
	RankLevel int      `json:"rank_level,omitempty" binding:"omitempty,min=0"`
	MinSalary *float64 `json:"min_salary,omitempty" binding:"omitempty,min=0"`
	MaxSalary *float64 `json:"max_salary,omitempty" binding:"omitempty,min=0"`
	IsActive  *bool    `json:"is_active,omitempty"`
}

// ApplyTo applies the update to an existing position.
func (r *UpdatePositionRequest) ApplyTo(position *domain.Position) {
	position.Code = r.Code
	position.Name = r.Name
	position.NameEn = r.NameEn
	position.RankLevel = r.RankLevel
	position.MinSalary = r.MinSalary
	position.MaxSalary = r.MaxSalary
	if r.IsActive != nil {
		position.IsActive = *r.IsActive
	}
}
