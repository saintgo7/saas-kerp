package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// taxInvoiceRepositoryGorm implements TaxInvoiceRepository using GORM
type taxInvoiceRepositoryGorm struct {
	db *gorm.DB
}

// NewTaxInvoiceRepositoryGorm creates a new TaxInvoiceRepository with GORM
func NewTaxInvoiceRepositoryGorm(db *gorm.DB) TaxInvoiceRepository {
	return &taxInvoiceRepositoryGorm{db: db}
}

// Create creates a new tax invoice
func (r *taxInvoiceRepositoryGorm) Create(ctx context.Context, invoice *domain.TaxInvoice) error {
	return r.db.WithContext(ctx).Create(invoice).Error
}

// GetByID retrieves a tax invoice by ID
func (r *taxInvoiceRepositoryGorm) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.TaxInvoice, error) {
	var invoice domain.TaxInvoice
	err := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		First(&invoice).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tax invoice not found")
		}
		return nil, err
	}
	return &invoice, nil
}

// GetByNumber retrieves a tax invoice by invoice number and type
func (r *taxInvoiceRepositoryGorm) GetByNumber(ctx context.Context, companyID uuid.UUID, number string, invoiceType domain.TaxInvoiceType) (*domain.TaxInvoice, error) {
	var invoice domain.TaxInvoice
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND invoice_number = ? AND invoice_type = ?", companyID, number, invoiceType).
		First(&invoice).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tax invoice not found")
		}
		return nil, err
	}
	return &invoice, nil
}

// List retrieves tax invoices with filtering
func (r *taxInvoiceRepositoryGorm) List(ctx context.Context, filter *TaxInvoiceFilter) ([]*domain.TaxInvoice, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.TaxInvoice{}).
		Where("company_id = ?", filter.CompanyID)

	// Apply filters
	if filter.StartDate != nil {
		query = query.Where("issue_date >= ?", filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("issue_date <= ?", filter.EndDate)
	}
	if filter.InvoiceType != nil {
		query = query.Where("invoice_type = ?", *filter.InvoiceType)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.BusinessNumber != nil {
		query = query.Where("supplier_business_number = ? OR buyer_business_number = ?",
			*filter.BusinessNumber, *filter.BusinessNumber)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize).Order("issue_date DESC, created_at DESC")

	var invoices []*domain.TaxInvoice
	if err := query.Find(&invoices).Error; err != nil {
		return nil, 0, err
	}

	return invoices, total, nil
}

// Update updates a tax invoice
func (r *taxInvoiceRepositoryGorm) Update(ctx context.Context, invoice *domain.TaxInvoice) error {
	return r.db.WithContext(ctx).Save(invoice).Error
}

// UpdateStatus updates the status of a tax invoice
func (r *taxInvoiceRepositoryGorm) UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status domain.TaxInvoiceStatus, userID *uuid.UUID) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if userID != nil {
		updates["updated_by"] = userID
	}

	return r.db.WithContext(ctx).Model(&domain.TaxInvoice{}).
		Where("id = ? AND company_id = ?", id, companyID).
		Updates(updates).Error
}

// Delete deletes a tax invoice
func (r *taxInvoiceRepositoryGorm) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", id, companyID).
		Delete(&domain.TaxInvoice{}).Error
}

// CreateItem creates a tax invoice item
func (r *taxInvoiceRepositoryGorm) CreateItem(ctx context.Context, item *domain.TaxInvoiceItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// ListItems lists all items for a tax invoice
func (r *taxInvoiceRepositoryGorm) ListItems(ctx context.Context, companyID, invoiceID uuid.UUID) ([]*domain.TaxInvoiceItem, error) {
	var items []*domain.TaxInvoiceItem
	err := r.db.WithContext(ctx).
		Where("tax_invoice_id = ? AND company_id = ?", invoiceID, companyID).
		Order("sequence_number ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// DeleteItems deletes all items for a tax invoice
func (r *taxInvoiceRepositoryGorm) DeleteItems(ctx context.Context, companyID, invoiceID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("tax_invoice_id = ? AND company_id = ?", invoiceID, companyID).
		Delete(&domain.TaxInvoiceItem{}).Error
}

// CreateHistory creates a tax invoice history entry
func (r *taxInvoiceRepositoryGorm) CreateHistory(ctx context.Context, history *domain.TaxInvoiceHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

// ListHistory lists all history entries for a tax invoice
func (r *taxInvoiceRepositoryGorm) ListHistory(ctx context.Context, companyID, invoiceID uuid.UUID) ([]*domain.TaxInvoiceHistory, error) {
	var history []*domain.TaxInvoiceHistory
	err := r.db.WithContext(ctx).
		Where("tax_invoice_id = ? AND company_id = ?", invoiceID, companyID).
		Order("created_at DESC").
		Find(&history).Error
	if err != nil {
		return nil, err
	}
	return history, nil
}

// GetSummary retrieves aggregated tax invoice data for a date range
func (r *taxInvoiceRepositoryGorm) GetSummary(ctx context.Context, companyID uuid.UUID, startDate, endDate time.Time) (*domain.TaxInvoiceSummary, error) {
	var summary domain.TaxInvoiceSummary

	// Sales summary
	var salesResult struct {
		Count       int64 `gorm:"column:count"`
		SupplyTotal int64 `gorm:"column:supply_total"`
		TaxTotal    int64 `gorm:"column:tax_total"`
	}
	err := r.db.WithContext(ctx).
		Model(&domain.TaxInvoice{}).
		Select("COUNT(*) as count, COALESCE(SUM(supply_amount), 0) as supply_total, COALESCE(SUM(tax_amount), 0) as tax_total").
		Where("company_id = ? AND invoice_type = ? AND issue_date >= ? AND issue_date <= ? AND status NOT IN (?, ?)",
			companyID, domain.TaxInvoiceTypeSales, startDate, endDate,
			domain.TaxInvoiceStatusCancelled, domain.TaxInvoiceStatusRejected).
		Scan(&salesResult).Error
	if err != nil {
		return nil, err
	}

	summary.SalesCount = salesResult.Count
	summary.SalesSupplyTotal = salesResult.SupplyTotal
	summary.SalesTaxTotal = salesResult.TaxTotal

	// Purchase summary
	var purchaseResult struct {
		Count       int64 `gorm:"column:count"`
		SupplyTotal int64 `gorm:"column:supply_total"`
		TaxTotal    int64 `gorm:"column:tax_total"`
	}
	err = r.db.WithContext(ctx).
		Model(&domain.TaxInvoice{}).
		Select("COUNT(*) as count, COALESCE(SUM(supply_amount), 0) as supply_total, COALESCE(SUM(tax_amount), 0) as tax_total").
		Where("company_id = ? AND invoice_type = ? AND issue_date >= ? AND issue_date <= ? AND status NOT IN (?, ?)",
			companyID, domain.TaxInvoiceTypePurchase, startDate, endDate,
			domain.TaxInvoiceStatusCancelled, domain.TaxInvoiceStatusRejected).
		Scan(&purchaseResult).Error
	if err != nil {
		return nil, err
	}

	summary.PurchaseCount = purchaseResult.Count
	summary.PurchaseSupplyTotal = purchaseResult.SupplyTotal
	summary.PurchaseTaxTotal = purchaseResult.TaxTotal

	return &summary, nil
}
