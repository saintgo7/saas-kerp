package domain

import (
	"time"

	"github.com/google/uuid"
)

// BaseModel contains common fields for all domain entities
type BaseModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v7()" json:"id"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TenantModel extends BaseModel with company_id for multi-tenancy
type TenantModel struct {
	BaseModel
	CompanyID uuid.UUID `gorm:"type:uuid;not null;index" json:"company_id"`
}

// SoftDeleteModel adds soft delete capability
type SoftDeleteModel struct {
	TenantModel
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}
