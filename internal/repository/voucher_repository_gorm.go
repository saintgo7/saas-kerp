package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// voucherRepositoryGorm implements VoucherRepository using GORM
type voucherRepositoryGorm struct {
	db *gorm.DB
}

// NewVoucherRepository creates a new VoucherRepository with GORM
func NewVoucherRepository(db *gorm.DB) VoucherRepository {
	return &voucherRepositoryGorm{db: db}
}

// Create inserts a new voucher with entries
func (r *voucherRepositoryGorm) Create(ctx context.Context, voucher *domain.Voucher) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create voucher
		if err := tx.Create(voucher).Error; err != nil {
			return err
		}

		// Create entries
		for i := range voucher.Entries {
			voucher.Entries[i].VoucherID = voucher.ID
			voucher.Entries[i].CompanyID = voucher.CompanyID
			if err := tx.Create(&voucher.Entries[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// Update modifies an existing voucher
func (r *voucherRepositoryGorm) Update(ctx context.Context, voucher *domain.Voucher) error {
	return r.db.WithContext(ctx).
		Model(voucher).
		Select("voucher_date", "voucher_type", "description", "reference_type", "reference_id",
			"total_debit", "total_credit", "updated_by").
		Updates(voucher).Error
}

// Delete removes a voucher by ID (soft delete by setting status to cancelled)
func (r *voucherRepositoryGorm) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete entries first
		if err := tx.Where("voucher_id = ?", id).Delete(&domain.VoucherEntry{}).Error; err != nil {
			return err
		}

		// Delete voucher
		return tx.Where("company_id = ? AND id = ?", companyID, id).Delete(&domain.Voucher{}).Error
	})
}

// FindByID retrieves a voucher by ID with entries
func (r *voucherRepositoryGorm) FindByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Voucher, error) {
	var voucher domain.Voucher
	err := r.db.WithContext(ctx).
		Preload("Entries", func(db *gorm.DB) *gorm.DB {
			return db.Order("line_no ASC")
		}).
		Preload("Entries.Account").
		Where("company_id = ? AND id = ?", companyID, id).
		First(&voucher).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrVoucherNotFound
		}
		return nil, err
	}
	return &voucher, nil
}

// FindByNo retrieves a voucher by voucher number
func (r *voucherRepositoryGorm) FindByNo(ctx context.Context, companyID uuid.UUID, voucherNo string) (*domain.Voucher, error) {
	var voucher domain.Voucher
	err := r.db.WithContext(ctx).
		Preload("Entries", func(db *gorm.DB) *gorm.DB {
			return db.Order("line_no ASC")
		}).
		Where("company_id = ? AND voucher_no = ?", companyID, voucherNo).
		First(&voucher).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrVoucherNotFound
		}
		return nil, err
	}
	return &voucher, nil
}

// FindAll retrieves vouchers with filtering and pagination
func (r *voucherRepositoryGorm) FindAll(ctx context.Context, filter VoucherFilter) ([]domain.Voucher, int64, error) {
	var vouchers []domain.Voucher
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Voucher{}).
		Where("company_id = ?", filter.CompanyID)

	// Apply filters
	if filter.VoucherType != nil {
		query = query.Where("voucher_type = ?", *filter.VoucherType)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.DateFrom != nil {
		query = query.Where("voucher_date >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("voucher_date <= ?", *filter.DateTo)
	}
	if filter.SearchTerm != "" {
		searchTerm := "%" + strings.ToLower(filter.SearchTerm) + "%"
		query = query.Where("LOWER(voucher_no) LIKE ? OR LOWER(description) LIKE ?",
			searchTerm, searchTerm)
	}

	// Filter by account/partner/department through entries
	if filter.AccountID != nil || filter.PartnerID != nil || filter.DepartmentID != nil {
		subQuery := r.db.Model(&domain.VoucherEntry{}).
			Select("DISTINCT voucher_id").
			Where("company_id = ?", filter.CompanyID)

		if filter.AccountID != nil {
			subQuery = subQuery.Where("account_id = ?", *filter.AccountID)
		}
		if filter.PartnerID != nil {
			subQuery = subQuery.Where("partner_id = ?", *filter.PartnerID)
		}
		if filter.DepartmentID != nil {
			subQuery = subQuery.Where("department_id = ?", *filter.DepartmentID)
		}

		query = query.Where("id IN (?)", subQuery)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortBy := "voucher_date DESC, voucher_no DESC"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
		if filter.SortDesc {
			sortBy = sortBy + " DESC"
		}
	}
	query = query.Order(sortBy)

	// Apply pagination
	if filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		if offset < 0 {
			offset = 0
		}
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Include entries if requested
	if filter.IncludeEntries {
		query = query.Preload("Entries", func(db *gorm.DB) *gorm.DB {
			return db.Order("line_no ASC")
		}).Preload("Entries.Account")
	}

	if err := query.Find(&vouchers).Error; err != nil {
		return nil, 0, err
	}

	return vouchers, total, nil
}

// FindByDateRange retrieves vouchers within a date range
func (r *voucherRepositoryGorm) FindByDateRange(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]domain.Voucher, error) {
	var vouchers []domain.Voucher
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND voucher_date >= ? AND voucher_date <= ?", companyID, from, to).
		Order("voucher_date, voucher_no").
		Find(&vouchers).Error
	return vouchers, err
}

// FindByStatus retrieves vouchers by status
func (r *voucherRepositoryGorm) FindByStatus(ctx context.Context, companyID uuid.UUID, status domain.VoucherStatus) ([]domain.Voucher, error) {
	var vouchers []domain.Voucher
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND status = ?", companyID, status).
		Order("voucher_date DESC, voucher_no DESC").
		Find(&vouchers).Error
	return vouchers, err
}

// CreateEntry inserts a new voucher entry
func (r *voucherRepositoryGorm) CreateEntry(ctx context.Context, entry *domain.VoucherEntry) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

// UpdateEntry modifies an existing entry
func (r *voucherRepositoryGorm) UpdateEntry(ctx context.Context, entry *domain.VoucherEntry) error {
	return r.db.WithContext(ctx).
		Model(entry).
		Select("line_no", "account_id", "debit_amount", "credit_amount", "description",
			"partner_id", "department_id", "project_id", "cost_center_id", "tags").
		Updates(entry).Error
}

// DeleteEntry removes an entry by ID
func (r *voucherRepositoryGorm) DeleteEntry(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.VoucherEntry{}).Error
}

// DeleteEntriesByVoucher removes all entries for a voucher
func (r *voucherRepositoryGorm) DeleteEntriesByVoucher(ctx context.Context, voucherID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("voucher_id = ?", voucherID).Delete(&domain.VoucherEntry{}).Error
}

// FindEntriesByVoucher retrieves all entries for a voucher
func (r *voucherRepositoryGorm) FindEntriesByVoucher(ctx context.Context, voucherID uuid.UUID) ([]domain.VoucherEntry, error) {
	var entries []domain.VoucherEntry
	err := r.db.WithContext(ctx).
		Preload("Account").
		Preload("Partner").
		Preload("Department").
		Where("voucher_id = ?", voucherID).
		Order("line_no").
		Find(&entries).Error
	return entries, err
}

// FindEntriesByAccount retrieves entries for an account within a date range
func (r *voucherRepositoryGorm) FindEntriesByAccount(ctx context.Context, companyID, accountID uuid.UUID, from, to time.Time) ([]domain.VoucherEntry, error) {
	var entries []domain.VoucherEntry
	err := r.db.WithContext(ctx).
		Joins("JOIN vouchers v ON voucher_entries.voucher_id = v.id").
		Where("voucher_entries.company_id = ? AND voucher_entries.account_id = ?", companyID, accountID).
		Where("v.voucher_date >= ? AND v.voucher_date <= ?", from, to).
		Where("v.status = ?", domain.VoucherStatusPosted).
		Order("v.voucher_date, v.voucher_no, voucher_entries.line_no").
		Find(&entries).Error
	return entries, err
}

// UpdateStatus updates the voucher status and related fields
func (r *voucherRepositoryGorm) UpdateStatus(ctx context.Context, voucher *domain.Voucher) error {
	updates := map[string]interface{}{
		"status":     voucher.Status,
		"updated_at": time.Now(),
	}

	switch voucher.Status {
	case domain.VoucherStatusPending:
		updates["submitted_at"] = voucher.SubmittedAt
		updates["submitted_by"] = voucher.SubmittedBy
	case domain.VoucherStatusApproved:
		updates["approved_at"] = voucher.ApprovedAt
		updates["approved_by"] = voucher.ApprovedBy
	case domain.VoucherStatusRejected:
		updates["rejected_at"] = voucher.RejectedAt
		updates["rejected_by"] = voucher.RejectedBy
		updates["rejection_reason"] = voucher.RejectionReason
	case domain.VoucherStatusPosted:
		updates["posted_at"] = voucher.PostedAt
		updates["posted_by"] = voucher.PostedBy
	}

	return r.db.WithContext(ctx).
		Model(&domain.Voucher{}).
		Where("id = ?", voucher.ID).
		Updates(updates).Error
}

// GenerateVoucherNo generates a unique voucher number
func (r *voucherRepositoryGorm) GenerateVoucherNo(ctx context.Context, companyID uuid.UUID, voucherType domain.VoucherType, voucherDate time.Time) (string, error) {
	var voucherNo string

	err := r.db.WithContext(ctx).Raw(
		"SELECT generate_voucher_number(?, ?, ?)",
		companyID, string(voucherType), voucherDate,
	).Scan(&voucherNo).Error

	if err != nil {
		// Fallback: generate manually if function doesn't exist
		year := voucherDate.Year()
		prefix := voucherType.GetPrefix()

		var lastNumber int
		r.db.WithContext(ctx).
			Table("voucher_sequences").
			Select("last_number").
			Where("company_id = ? AND fiscal_year = ? AND voucher_type = ?",
				companyID, year, string(voucherType)).
			Scan(&lastNumber)

		lastNumber++
		voucherNo = fmt.Sprintf("%s-%d-%06d", prefix, year, lastNumber)

		// Update or insert sequence
		r.db.WithContext(ctx).Exec(`
			INSERT INTO voucher_sequences (id, company_id, fiscal_year, voucher_type, prefix, last_number, updated_at)
			VALUES (uuid_generate_v7(), ?, ?, ?, ?, ?, NOW())
			ON CONFLICT (company_id, fiscal_year, voucher_type)
			DO UPDATE SET last_number = ?, updated_at = NOW()
		`, companyID, year, string(voucherType), prefix, lastNumber, lastNumber)
	}

	return voucherNo, nil
}

// WithTransaction executes a function within a transaction
func (r *voucherRepositoryGorm) WithTransaction(ctx context.Context, fn func(repo VoucherRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &voucherRepositoryGorm{db: tx}
		return fn(txRepo)
	})
}
