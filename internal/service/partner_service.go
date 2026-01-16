package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// Partner-related errors
var (
	ErrPartnerNotFound        = errors.New("partner not found")
	ErrPartnerCodeExists      = errors.New("partner code already exists")
	ErrPartnerBizNoExists     = errors.New("business number already exists")
	ErrPartnerHasTransactions = errors.New("partner has transactions and cannot be deleted")
	ErrPartnerInvalidType     = errors.New("invalid partner type")
)

// PartnerFilter is re-exported from repository
type PartnerFilter = repository.PartnerFilter

// PartnerService defines the interface for partner business logic
type PartnerService interface {
	// CRUD operations
	Create(ctx context.Context, partner *domain.Partner) error
	Update(ctx context.Context, partner *domain.Partner) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Query operations
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Partner, error)
	GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Partner, error)
	GetByBusinessNumber(ctx context.Context, companyID uuid.UUID, businessNumber string) (*domain.Partner, error)
	List(ctx context.Context, filter *PartnerFilter) ([]domain.Partner, int64, error)

	// Batch operations
	CreateBatch(ctx context.Context, partners []domain.Partner) error
	Activate(ctx context.Context, companyID uuid.UUID, ids []uuid.UUID) error
	Deactivate(ctx context.Context, companyID uuid.UUID, ids []uuid.UUID) error

	// Validation
	CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error)

	// Statistics
	GetStats(ctx context.Context, companyID uuid.UUID) (*PartnerStats, error)
}

// PartnerStats holds partner statistics
type PartnerStats struct {
	TotalCount    int64 `json:"total_count"`
	CustomerCount int64 `json:"customer_count"`
	VendorCount   int64 `json:"vendor_count"`
	ActiveCount   int64 `json:"active_count"`
	InactiveCount int64 `json:"inactive_count"`
}

// partnerService implements PartnerService
type partnerService struct {
	repo repository.PartnerRepository
}

// NewPartnerService creates a new PartnerService
func NewPartnerService(repo repository.PartnerRepository) PartnerService {
	return &partnerService{repo: repo}
}

// Create creates a new partner
func (s *partnerService) Create(ctx context.Context, partner *domain.Partner) error {
	// Validate partner type
	if partner.PartnerType != "customer" && partner.PartnerType != "vendor" && partner.PartnerType != "both" {
		return ErrPartnerInvalidType
	}

	// Check for duplicate code
	exists, err := s.repo.ExistsByCode(ctx, partner.CompanyID, partner.Code, nil)
	if err != nil {
		return err
	}
	if exists {
		return ErrPartnerCodeExists
	}

	// Check for duplicate business number if provided
	if partner.BusinessNumber != "" {
		exists, err := s.repo.ExistsByBusinessNumber(ctx, partner.CompanyID, partner.BusinessNumber, nil)
		if err != nil {
			return err
		}
		if exists {
			return ErrPartnerBizNoExists
		}
	}

	return s.repo.Create(ctx, partner)
}

// Update updates a partner
func (s *partnerService) Update(ctx context.Context, partner *domain.Partner) error {
	// Validate partner type
	if partner.PartnerType != "customer" && partner.PartnerType != "vendor" && partner.PartnerType != "both" {
		return ErrPartnerInvalidType
	}

	// Check existing
	_, err := s.repo.GetByID(ctx, partner.CompanyID, partner.ID)
	if err != nil {
		return ErrPartnerNotFound
	}

	// Check for duplicate code
	exists, err := s.repo.ExistsByCode(ctx, partner.CompanyID, partner.Code, &partner.ID)
	if err != nil {
		return err
	}
	if exists {
		return ErrPartnerCodeExists
	}

	// Check for duplicate business number if provided
	if partner.BusinessNumber != "" {
		exists, err := s.repo.ExistsByBusinessNumber(ctx, partner.CompanyID, partner.BusinessNumber, &partner.ID)
		if err != nil {
			return err
		}
		if exists {
			return ErrPartnerBizNoExists
		}
	}

	return s.repo.Update(ctx, partner)
}

// Delete deletes a partner
func (s *partnerService) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	canDelete, reason, err := s.CanDelete(ctx, companyID, id)
	if err != nil {
		return err
	}
	if !canDelete {
		return errors.New(reason)
	}

	return s.repo.Delete(ctx, companyID, id)
}

// GetByID retrieves a partner by ID
func (s *partnerService) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Partner, error) {
	return s.repo.GetByID(ctx, companyID, id)
}

// GetByCode retrieves a partner by code
func (s *partnerService) GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Partner, error) {
	return s.repo.GetByCode(ctx, companyID, code)
}

// GetByBusinessNumber retrieves a partner by business number
func (s *partnerService) GetByBusinessNumber(ctx context.Context, companyID uuid.UUID, businessNumber string) (*domain.Partner, error) {
	return s.repo.GetByBusinessNumber(ctx, companyID, businessNumber)
}

// List retrieves partners with filtering
func (s *partnerService) List(ctx context.Context, filter *PartnerFilter) ([]domain.Partner, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	return s.repo.List(ctx, filter)
}

// CreateBatch creates multiple partners
func (s *partnerService) CreateBatch(ctx context.Context, partners []domain.Partner) error {
	for _, p := range partners {
		// Validate each partner
		if p.PartnerType != "customer" && p.PartnerType != "vendor" && p.PartnerType != "both" {
			return ErrPartnerInvalidType
		}
	}
	return s.repo.CreateBatch(ctx, partners)
}

// Activate activates partners
func (s *partnerService) Activate(ctx context.Context, companyID uuid.UUID, ids []uuid.UUID) error {
	return s.repo.UpdateActiveStatus(ctx, companyID, ids, true)
}

// Deactivate deactivates partners
func (s *partnerService) Deactivate(ctx context.Context, companyID uuid.UUID, ids []uuid.UUID) error {
	return s.repo.UpdateActiveStatus(ctx, companyID, ids, false)
}

// CanDelete checks if a partner can be deleted
func (s *partnerService) CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error) {
	// Check if partner exists
	_, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return false, "partner not found", ErrPartnerNotFound
	}

	// Check for voucher entries
	hasEntries, err := s.repo.HasVoucherEntries(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}
	if hasEntries {
		return false, "partner has voucher entries", nil
	}

	// Check for tax invoices
	hasInvoices, err := s.repo.HasTaxInvoices(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}
	if hasInvoices {
		return false, "partner has tax invoices", nil
	}

	return true, "", nil
}

// GetStats retrieves partner statistics
func (s *partnerService) GetStats(ctx context.Context, companyID uuid.UUID) (*PartnerStats, error) {
	customerCount, err := s.repo.GetCustomerCount(ctx, companyID)
	if err != nil {
		return nil, err
	}

	vendorCount, err := s.repo.GetVendorCount(ctx, companyID)
	if err != nil {
		return nil, err
	}

	// Get total and active counts
	allActive := true
	activeFilter := &PartnerFilter{CompanyID: companyID, IsActive: &allActive}
	_, activeCount, err := s.repo.List(ctx, activeFilter)
	if err != nil {
		return nil, err
	}

	allFilter := &PartnerFilter{CompanyID: companyID}
	_, totalCount, err := s.repo.List(ctx, allFilter)
	if err != nil {
		return nil, err
	}

	return &PartnerStats{
		TotalCount:    totalCount,
		CustomerCount: customerCount,
		VendorCount:   vendorCount,
		ActiveCount:   activeCount,
		InactiveCount: totalCount - activeCount,
	}, nil
}
