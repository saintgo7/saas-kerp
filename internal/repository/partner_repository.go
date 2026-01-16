package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// PartnerFilter defines filter criteria for listing partners
type PartnerFilter struct {
	CompanyID   uuid.UUID
	PartnerType string // "customer", "vendor", "both", or empty for all
	IsActive    *bool
	SearchTerm  string // Search in code, name, business_number
	Page        int
	PageSize    int
}

// PartnerRepository defines the interface for partner data access
type PartnerRepository interface {
	// CRUD operations
	Create(ctx context.Context, partner *domain.Partner) error
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Partner, error)
	GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Partner, error)
	GetByBusinessNumber(ctx context.Context, companyID uuid.UUID, businessNumber string) (*domain.Partner, error)
	List(ctx context.Context, filter *PartnerFilter) ([]domain.Partner, int64, error)
	Update(ctx context.Context, partner *domain.Partner) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Validation
	ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)
	ExistsByBusinessNumber(ctx context.Context, companyID uuid.UUID, businessNumber string, excludeID *uuid.UUID) (bool, error)
	HasVoucherEntries(ctx context.Context, companyID, partnerID uuid.UUID) (bool, error)
	HasTaxInvoices(ctx context.Context, companyID, partnerID uuid.UUID) (bool, error)

	// Batch operations
	CreateBatch(ctx context.Context, partners []domain.Partner) error
	UpdateActiveStatus(ctx context.Context, companyID uuid.UUID, ids []uuid.UUID, isActive bool) error

	// Statistics
	GetCustomerCount(ctx context.Context, companyID uuid.UUID) (int64, error)
	GetVendorCount(ctx context.Context, companyID uuid.UUID) (int64, error)
}
