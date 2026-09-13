package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

// SoftDeleteModel adds soft delete capability.
//
// The field must be gorm.DeletedAt, not *time.Time: only that type makes GORM
// turn Delete into an UPDATE and append `deleted_at IS NULL` to every query.
// With a plain *time.Time the column exists, nothing ever writes it, and
// Delete issues a hard DELETE - which is what made deleting a user that had
// ever created a voucher fail with a foreign key violation.
//
// It is tagged json:"-" because gorm.DeletedAt marshals as
// {"Time":...,"Valid":...}, which has no business in an API response.
type SoftDeleteModel struct {
	TenantModel
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
