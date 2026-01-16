package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// TaxInvoiceFilter defines filter criteria for listing tax invoices
type TaxInvoiceFilter struct {
	CompanyID      uuid.UUID
	StartDate      *time.Time
	EndDate        *time.Time
	InvoiceType    *domain.TaxInvoiceType
	Status         *domain.TaxInvoiceStatus
	BusinessNumber *string
	Page           int
	PageSize       int
}

// TaxInvoiceRepository defines the interface for tax invoice data access
type TaxInvoiceRepository interface {
	Create(ctx context.Context, invoice *domain.TaxInvoice) error
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.TaxInvoice, error)
	GetByNumber(ctx context.Context, companyID uuid.UUID, number string, invoiceType domain.TaxInvoiceType) (*domain.TaxInvoice, error)
	List(ctx context.Context, filter *TaxInvoiceFilter) ([]*domain.TaxInvoice, int64, error)
	Update(ctx context.Context, invoice *domain.TaxInvoice) error
	UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status domain.TaxInvoiceStatus, userID *uuid.UUID) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Items
	CreateItem(ctx context.Context, item *domain.TaxInvoiceItem) error
	ListItems(ctx context.Context, companyID, invoiceID uuid.UUID) ([]*domain.TaxInvoiceItem, error)
	DeleteItems(ctx context.Context, companyID, invoiceID uuid.UUID) error

	// History
	CreateHistory(ctx context.Context, history *domain.TaxInvoiceHistory) error
	ListHistory(ctx context.Context, companyID, invoiceID uuid.UUID) ([]*domain.TaxInvoiceHistory, error)

	// Summary
	GetSummary(ctx context.Context, companyID uuid.UUID, startDate, endDate time.Time) (*domain.TaxInvoiceSummary, error)
}
