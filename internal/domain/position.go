package domain

import "errors"

// Position errors
var (
	ErrPositionNotFound       = errors.New("position not found")
	ErrPositionCodeExists     = errors.New("position code already exists")
	ErrPositionCodeEmpty      = errors.New("position code is required")
	ErrPositionNameEmpty      = errors.New("position name is required")
	ErrPositionInUse          = errors.New("position is assigned to employees and cannot be deleted")
	ErrPositionSalaryRange    = errors.New("minimum salary cannot exceed maximum salary")
	ErrPositionSalaryNegative = errors.New("salary bounds cannot be negative")
	ErrPositionRankNegative   = errors.New("rank level cannot be negative")
)

// Position represents a job title / rank inside a company.
//
// Table: positions (db/migrations/000006_hr_tables.up.sql). The table has no
// deleted_at column, so this model has no gorm.DeletedAt: deleting a position
// is a real DELETE and employees.position_id references it, which is why the
// service refuses the delete while any employee still points at the row.
type Position struct {
	TenantModel

	// Basic info
	Code   string `gorm:"type:varchar(20);not null" json:"code"`
	Name   string `gorm:"type:varchar(100);not null" json:"name"`
	NameEn string `gorm:"type:varchar(100)" json:"name_en,omitempty"`

	// RankLevel orders positions from junior to senior. It is not a hierarchy
	// link: two positions may legitimately share a level.
	RankLevel int `gorm:"default:0" json:"rank_level"`

	// Pay band. Both bounds are nullable in the schema, so a position may
	// declare only a floor, only a ceiling, or neither.
	MinSalary *float64 `gorm:"type:decimal(18,0)" json:"min_salary,omitempty"`
	MaxSalary *float64 `gorm:"type:decimal(18,0)" json:"max_salary,omitempty"`

	// Status
	IsActive bool `gorm:"default:true" json:"is_active"`
}

// TableName specifies the table name for GORM
func (Position) TableName() string {
	return "positions"
}

// Validate checks the invariants that the database cannot express.
func (p *Position) Validate() error {
	if p.Code == "" {
		return ErrPositionCodeEmpty
	}
	if p.Name == "" {
		return ErrPositionNameEmpty
	}
	if p.RankLevel < 0 {
		return ErrPositionRankNegative
	}
	if p.MinSalary != nil && *p.MinSalary < 0 {
		return ErrPositionSalaryNegative
	}
	if p.MaxSalary != nil && *p.MaxSalary < 0 {
		return ErrPositionSalaryNegative
	}
	if p.MinSalary != nil && p.MaxSalary != nil && *p.MinSalary > *p.MaxSalary {
		return ErrPositionSalaryRange
	}
	return nil
}
