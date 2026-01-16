package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// VoucherFilter defines filter options for voucher queries
type VoucherFilter struct {
	CompanyID     uuid.UUID
	VoucherType   *domain.VoucherType
	Status        *domain.VoucherStatus
	DateFrom      *time.Time
	DateTo        *time.Time
	AccountID     *uuid.UUID
	PartnerID     *uuid.UUID
	DepartmentID  *uuid.UUID
	SearchTerm    string
	IncludeEntries bool
	Page          int
	PageSize      int
	SortBy        string
	SortDesc      bool
}

// VoucherRepository defines the interface for voucher data access
type VoucherRepository interface {
	// CRUD operations
	Create(ctx context.Context, voucher *domain.Voucher) error
	Update(ctx context.Context, voucher *domain.Voucher) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Query operations
	FindByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Voucher, error)
	FindByNo(ctx context.Context, companyID uuid.UUID, voucherNo string) (*domain.Voucher, error)
	FindAll(ctx context.Context, filter VoucherFilter) ([]domain.Voucher, int64, error)
	FindByDateRange(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]domain.Voucher, error)
	FindByStatus(ctx context.Context, companyID uuid.UUID, status domain.VoucherStatus) ([]domain.Voucher, error)

	// Entry operations
	CreateEntry(ctx context.Context, entry *domain.VoucherEntry) error
	UpdateEntry(ctx context.Context, entry *domain.VoucherEntry) error
	DeleteEntry(ctx context.Context, id uuid.UUID) error
	DeleteEntriesByVoucher(ctx context.Context, voucherID uuid.UUID) error
	FindEntriesByVoucher(ctx context.Context, voucherID uuid.UUID) ([]domain.VoucherEntry, error)
	FindEntriesByAccount(ctx context.Context, companyID, accountID uuid.UUID, from, to time.Time) ([]domain.VoucherEntry, error)

	// Workflow operations
	UpdateStatus(ctx context.Context, voucher *domain.Voucher) error

	// Number generation
	GenerateVoucherNo(ctx context.Context, companyID uuid.UUID, voucherType domain.VoucherType, voucherDate time.Time) (string, error)

	// Transaction support
	WithTransaction(ctx context.Context, fn func(repo VoucherRepository) error) error
}
