package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// partnerRepositoryGorm implements PartnerRepository using GORM
type partnerRepositoryGorm struct {
	db *gorm.DB
}

// NewPartnerRepositoryGorm creates a new PartnerRepository with GORM
func NewPartnerRepositoryGorm(db *gorm.DB) PartnerRepository {
	return &partnerRepositoryGorm{db: db}
}

// Create creates a new partner
func (r *partnerRepositoryGorm) Create(ctx context.Context, partner *domain.Partner) error {
	return r.db.WithContext(ctx).Create(partner).Error
}

// GetByID retrieves a partner by ID
func (r *partnerRepositoryGorm) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Partner, error) {
	var partner domain.Partner
	err := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		First(&partner).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("partner not found")
		}
		return nil, err
	}
	return &partner, nil
}

// GetByCode retrieves a partner by code
func (r *partnerRepositoryGorm) GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Partner, error) {
	var partner domain.Partner
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND code = ?", companyID, code).
		First(&partner).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("partner not found")
		}
		return nil, err
	}
	return &partner, nil
}

// GetByBusinessNumber retrieves a partner by business number
func (r *partnerRepositoryGorm) GetByBusinessNumber(ctx context.Context, companyID uuid.UUID, businessNumber string) (*domain.Partner, error) {
	var partner domain.Partner
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND business_number = ?", companyID, businessNumber).
		First(&partner).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("partner not found")
		}
		return nil, err
	}
	return &partner, nil
}

// List retrieves partners with filtering
func (r *partnerRepositoryGorm) List(ctx context.Context, filter *PartnerFilter) ([]domain.Partner, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Partner{}).
		Where("company_id = ?", filter.CompanyID)

	// Apply filters
	if filter.PartnerType != "" {
		if filter.PartnerType == "customer" || filter.PartnerType == "vendor" {
			query = query.Where("partner_type = ? OR partner_type = 'both'", filter.PartnerType)
		} else {
			query = query.Where("partner_type = ?", filter.PartnerType)
		}
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.SearchTerm != "" {
		searchPattern := "%" + filter.SearchTerm + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ? OR business_number ILIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	query = query.Order("code ASC")

	var partners []domain.Partner
	if err := query.Find(&partners).Error; err != nil {
		return nil, 0, err
	}

	return partners, total, nil
}

// Update updates a partner
func (r *partnerRepositoryGorm) Update(ctx context.Context, partner *domain.Partner) error {
	return r.db.WithContext(ctx).Save(partner).Error
}

// Delete deletes a partner
func (r *partnerRepositoryGorm) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		Delete(&domain.Partner{}).Error
}

// ExistsByCode checks if a partner with the given code exists
func (r *partnerRepositoryGorm) ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&domain.Partner{}).
		Where("company_id = ? AND code = ?", companyID, code)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByBusinessNumber checks if a partner with the given business number exists
func (r *partnerRepositoryGorm) ExistsByBusinessNumber(ctx context.Context, companyID uuid.UUID, businessNumber string, excludeID *uuid.UUID) (bool, error) {
	if businessNumber == "" {
		return false, nil
	}

	var count int64
	query := r.db.WithContext(ctx).Model(&domain.Partner{}).
		Where("company_id = ? AND business_number = ?", companyID, businessNumber)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasVoucherEntries checks if the partner has any voucher entries
func (r *partnerRepositoryGorm) HasVoucherEntries(ctx context.Context, companyID, partnerID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.VoucherEntry{}).
		Where("company_id = ? AND partner_id = ?", companyID, partnerID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasTaxInvoices checks if the partner has any tax invoices
func (r *partnerRepositoryGorm) HasTaxInvoices(ctx context.Context, companyID, partnerID uuid.UUID) (bool, error) {
	// Check in both supplier and buyer fields
	var count int64
	err := r.db.WithContext(ctx).
		Table("tax_invoices").
		Where("company_id = ? AND (supplier_business_number = (SELECT business_number FROM partners WHERE id = ?) OR buyer_business_number = (SELECT business_number FROM partners WHERE id = ?))",
			companyID, partnerID, partnerID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateBatch creates multiple partners
func (r *partnerRepositoryGorm) CreateBatch(ctx context.Context, partners []domain.Partner) error {
	return r.db.WithContext(ctx).Create(&partners).Error
}

// UpdateActiveStatus updates the active status for multiple partners
func (r *partnerRepositoryGorm) UpdateActiveStatus(ctx context.Context, companyID uuid.UUID, ids []uuid.UUID, isActive bool) error {
	return r.db.WithContext(ctx).Model(&domain.Partner{}).
		Where("company_id = ? AND id IN ?", companyID, ids).
		Update("is_active", isActive).Error
}

// GetCustomerCount returns the count of customers
func (r *partnerRepositoryGorm) GetCustomerCount(ctx context.Context, companyID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Partner{}).
		Where("company_id = ? AND (partner_type = 'customer' OR partner_type = 'both') AND is_active = true", companyID).
		Count(&count).Error
	return count, err
}

// GetVendorCount returns the count of vendors
func (r *partnerRepositoryGorm) GetVendorCount(ctx context.Context, companyID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Partner{}).
		Where("company_id = ? AND (partner_type = 'vendor' OR partner_type = 'both') AND is_active = true", companyID).
		Count(&count).Error
	return count, err
}
