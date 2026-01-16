package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// VoucherService defines the interface for voucher business logic
type VoucherService interface {
	// CRUD operations
	Create(ctx context.Context, voucher *domain.Voucher) error
	Update(ctx context.Context, voucher *domain.Voucher) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// Query operations
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Voucher, error)
	GetByNo(ctx context.Context, companyID uuid.UUID, voucherNo string) (*domain.Voucher, error)
	List(ctx context.Context, filter repository.VoucherFilter) ([]domain.Voucher, int64, error)
	GetByDateRange(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]domain.Voucher, error)
	GetPending(ctx context.Context, companyID uuid.UUID) ([]domain.Voucher, error)

	// Entry operations
	AddEntry(ctx context.Context, voucherID uuid.UUID, entry *domain.VoucherEntry) error
	UpdateEntry(ctx context.Context, entry *domain.VoucherEntry) error
	RemoveEntry(ctx context.Context, entryID uuid.UUID) error
	ReplaceEntries(ctx context.Context, voucherID uuid.UUID, entries []domain.VoucherEntry) error

	// Workflow operations
	Submit(ctx context.Context, companyID, voucherID, userID uuid.UUID) error
	Approve(ctx context.Context, companyID, voucherID, userID uuid.UUID) error
	Reject(ctx context.Context, companyID, voucherID, userID uuid.UUID, reason string) error
	Post(ctx context.Context, companyID, voucherID, userID uuid.UUID) error
	Cancel(ctx context.Context, companyID, voucherID uuid.UUID) error

	// Reversal
	Reverse(ctx context.Context, companyID, voucherID, userID uuid.UUID, reversalDate time.Time, description string) (*domain.Voucher, error)

	// Validation
	ValidateEntries(ctx context.Context, companyID uuid.UUID, entries []domain.VoucherEntry) error
}

// voucherService implements VoucherService
type voucherService struct {
	voucherRepo repository.VoucherRepository
	accountRepo repository.AccountRepository
}

// NewVoucherService creates a new VoucherService
func NewVoucherService(voucherRepo repository.VoucherRepository, accountRepo repository.AccountRepository) VoucherService {
	return &voucherService{
		voucherRepo: voucherRepo,
		accountRepo: accountRepo,
	}
}

// Create creates a new voucher with entries
func (s *voucherService) Create(ctx context.Context, voucher *domain.Voucher) error {
	// Validate voucher
	if err := voucher.Validate(); err != nil {
		return err
	}

	// Validate entries
	if len(voucher.Entries) == 0 {
		return domain.ErrVoucherNoEntries
	}

	if err := s.ValidateEntries(ctx, voucher.CompanyID, voucher.Entries); err != nil {
		return err
	}

	// Calculate totals
	voucher.CalculateTotals()

	// Validate balance (debit must equal credit)
	if err := voucher.ValidateBalance(); err != nil {
		return err
	}

	// Generate voucher number
	voucherNo, err := s.voucherRepo.GenerateVoucherNo(ctx, voucher.CompanyID, voucher.VoucherType, voucher.VoucherDate)
	if err != nil {
		return err
	}
	voucher.VoucherNo = voucherNo

	// Set status to draft
	voucher.Status = domain.VoucherStatusDraft

	// Assign line numbers
	for i := range voucher.Entries {
		voucher.Entries[i].LineNo = i + 1
	}

	return s.voucherRepo.Create(ctx, voucher)
}

// Update updates an existing voucher
func (s *voucherService) Update(ctx context.Context, voucher *domain.Voucher) error {
	// Get existing voucher
	existing, err := s.voucherRepo.FindByID(ctx, voucher.CompanyID, voucher.ID)
	if err != nil {
		return err
	}

	// Check if can edit
	if !existing.CanEdit() {
		return domain.ErrVoucherCannotEdit
	}

	// Validate voucher
	if err := voucher.Validate(); err != nil {
		return err
	}

	return s.voucherRepo.Update(ctx, voucher)
}

// Delete removes a voucher
func (s *voucherService) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	// Get existing voucher
	existing, err := s.voucherRepo.FindByID(ctx, companyID, id)
	if err != nil {
		return err
	}

	// Can only delete draft or rejected vouchers
	if !existing.CanEdit() {
		return domain.ErrVoucherCannotEdit
	}

	return s.voucherRepo.Delete(ctx, companyID, id)
}

// GetByID retrieves a voucher by ID
func (s *voucherService) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Voucher, error) {
	return s.voucherRepo.FindByID(ctx, companyID, id)
}

// GetByNo retrieves a voucher by voucher number
func (s *voucherService) GetByNo(ctx context.Context, companyID uuid.UUID, voucherNo string) (*domain.Voucher, error) {
	return s.voucherRepo.FindByNo(ctx, companyID, voucherNo)
}

// List retrieves vouchers with filtering and pagination
func (s *voucherService) List(ctx context.Context, filter repository.VoucherFilter) ([]domain.Voucher, int64, error) {
	return s.voucherRepo.FindAll(ctx, filter)
}

// GetByDateRange retrieves vouchers within a date range
func (s *voucherService) GetByDateRange(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]domain.Voucher, error) {
	return s.voucherRepo.FindByDateRange(ctx, companyID, from, to)
}

// GetPending retrieves vouchers pending approval
func (s *voucherService) GetPending(ctx context.Context, companyID uuid.UUID) ([]domain.Voucher, error) {
	return s.voucherRepo.FindByStatus(ctx, companyID, domain.VoucherStatusPending)
}

// AddEntry adds an entry to a voucher
func (s *voucherService) AddEntry(ctx context.Context, voucherID uuid.UUID, entry *domain.VoucherEntry) error {
	// Validate entry
	if err := entry.Validate(); err != nil {
		return err
	}

	// Get voucher to check status
	voucher, err := s.voucherRepo.FindByID(ctx, entry.CompanyID, voucherID)
	if err != nil {
		return err
	}

	if !voucher.CanEdit() {
		return domain.ErrVoucherCannotEdit
	}

	// Validate account
	if err := s.validateAccountForPosting(ctx, entry.CompanyID, entry.AccountID); err != nil {
		return err
	}

	// Set line number
	entry.LineNo = len(voucher.Entries) + 1
	entry.VoucherID = voucherID

	if err := s.voucherRepo.CreateEntry(ctx, entry); err != nil {
		return err
	}

	// Recalculate voucher totals
	voucher.Entries = append(voucher.Entries, *entry)
	voucher.CalculateTotals()
	return s.voucherRepo.Update(ctx, voucher)
}

// UpdateEntry updates an existing entry
func (s *voucherService) UpdateEntry(ctx context.Context, entry *domain.VoucherEntry) error {
	if err := entry.Validate(); err != nil {
		return err
	}

	return s.voucherRepo.UpdateEntry(ctx, entry)
}

// RemoveEntry removes an entry from a voucher
func (s *voucherService) RemoveEntry(ctx context.Context, entryID uuid.UUID) error {
	return s.voucherRepo.DeleteEntry(ctx, entryID)
}

// ReplaceEntries replaces all entries of a voucher
func (s *voucherService) ReplaceEntries(ctx context.Context, voucherID uuid.UUID, entries []domain.VoucherEntry) error {
	if len(entries) == 0 {
		return domain.ErrVoucherNoEntries
	}

	// Get voucher
	voucher, err := s.voucherRepo.FindByID(ctx, entries[0].CompanyID, voucherID)
	if err != nil {
		return err
	}

	if !voucher.CanEdit() {
		return domain.ErrVoucherCannotEdit
	}

	// Validate all entries
	if err := s.ValidateEntries(ctx, voucher.CompanyID, entries); err != nil {
		return err
	}

	return s.voucherRepo.WithTransaction(ctx, func(repo repository.VoucherRepository) error {
		// Delete existing entries
		if err := repo.DeleteEntriesByVoucher(ctx, voucherID); err != nil {
			return err
		}

		// Create new entries
		for i := range entries {
			entries[i].VoucherID = voucherID
			entries[i].CompanyID = voucher.CompanyID
			entries[i].LineNo = i + 1
			if err := repo.CreateEntry(ctx, &entries[i]); err != nil {
				return err
			}
		}

		// Recalculate totals
		voucher.Entries = entries
		voucher.CalculateTotals()

		// Validate balance
		if err := voucher.ValidateBalance(); err != nil {
			return err
		}

		return repo.Update(ctx, voucher)
	})
}

// Submit submits a voucher for approval
func (s *voucherService) Submit(ctx context.Context, companyID, voucherID, userID uuid.UUID) error {
	voucher, err := s.voucherRepo.FindByID(ctx, companyID, voucherID)
	if err != nil {
		return err
	}

	if err := voucher.Submit(userID); err != nil {
		return err
	}

	return s.voucherRepo.UpdateStatus(ctx, voucher)
}

// Approve approves a voucher
func (s *voucherService) Approve(ctx context.Context, companyID, voucherID, userID uuid.UUID) error {
	voucher, err := s.voucherRepo.FindByID(ctx, companyID, voucherID)
	if err != nil {
		return err
	}

	if err := voucher.Approve(userID); err != nil {
		return err
	}

	return s.voucherRepo.UpdateStatus(ctx, voucher)
}

// Reject rejects a voucher
func (s *voucherService) Reject(ctx context.Context, companyID, voucherID, userID uuid.UUID, reason string) error {
	voucher, err := s.voucherRepo.FindByID(ctx, companyID, voucherID)
	if err != nil {
		return err
	}

	if err := voucher.Reject(userID, reason); err != nil {
		return err
	}

	return s.voucherRepo.UpdateStatus(ctx, voucher)
}

// Post posts a voucher to the ledger
func (s *voucherService) Post(ctx context.Context, companyID, voucherID, userID uuid.UUID) error {
	voucher, err := s.voucherRepo.FindByID(ctx, companyID, voucherID)
	if err != nil {
		return err
	}

	if err := voucher.Post(userID); err != nil {
		return err
	}

	return s.voucherRepo.UpdateStatus(ctx, voucher)
}

// Cancel cancels a voucher
func (s *voucherService) Cancel(ctx context.Context, companyID, voucherID uuid.UUID) error {
	voucher, err := s.voucherRepo.FindByID(ctx, companyID, voucherID)
	if err != nil {
		return err
	}

	if err := voucher.Cancel(); err != nil {
		return err
	}

	return s.voucherRepo.UpdateStatus(ctx, voucher)
}

// Reverse creates a reversal voucher
func (s *voucherService) Reverse(ctx context.Context, companyID, voucherID, userID uuid.UUID, reversalDate time.Time, description string) (*domain.Voucher, error) {
	// Get original voucher
	original, err := s.voucherRepo.FindByID(ctx, companyID, voucherID)
	if err != nil {
		return nil, err
	}

	// Check if can reverse
	if !original.Status.CanReverse() {
		return nil, domain.ErrVoucherCannotReverse
	}

	// Check if already reversed
	if original.ReversedByID != nil {
		return nil, domain.ErrVoucherAlreadyReversed
	}

	// Create reversal voucher
	reversal := &domain.Voucher{
		TenantModel: domain.TenantModel{
			CompanyID: companyID,
		},
		VoucherDate:   reversalDate,
		VoucherType:   original.VoucherType,
		Status:        domain.VoucherStatusDraft,
		Description:   description,
		IsReversal:    true,
		ReversalOfID:  &original.ID,
		CreatedBy:     &userID,
	}

	// Create reversed entries (swap debit and credit)
	for _, entry := range original.Entries {
		reversalEntry := domain.VoucherEntry{
			CompanyID:    companyID,
			AccountID:    entry.AccountID,
			DebitAmount:  entry.CreditAmount,  // Swap
			CreditAmount: entry.DebitAmount,   // Swap
			Description:  entry.Description,
			PartnerID:    entry.PartnerID,
			DepartmentID: entry.DepartmentID,
			ProjectID:    entry.ProjectID,
			CostCenterID: entry.CostCenterID,
		}
		reversal.Entries = append(reversal.Entries, reversalEntry)
	}

	// Create the reversal voucher
	if err := s.Create(ctx, reversal); err != nil {
		return nil, err
	}

	// Update original voucher to reference the reversal
	original.ReversedByID = &reversal.ID
	if err := s.voucherRepo.Update(ctx, original); err != nil {
		return nil, err
	}

	return reversal, nil
}

// ValidateEntries validates all entries for a voucher
func (s *voucherService) ValidateEntries(ctx context.Context, companyID uuid.UUID, entries []domain.VoucherEntry) error {
	var totalDebit, totalCredit float64

	for _, entry := range entries {
		// Validate entry
		if err := entry.Validate(); err != nil {
			return err
		}

		// Validate account can accept postings
		if err := s.validateAccountForPosting(ctx, companyID, entry.AccountID); err != nil {
			return err
		}

		totalDebit += entry.DebitAmount
		totalCredit += entry.CreditAmount
	}

	// Check balance
	if totalDebit != totalCredit {
		return domain.ErrVoucherUnbalanced
	}

	return nil
}

// validateAccountForPosting checks if an account can accept postings
func (s *voucherService) validateAccountForPosting(ctx context.Context, companyID, accountID uuid.UUID) error {
	account, err := s.accountRepo.FindByID(ctx, companyID, accountID)
	if err != nil {
		return err
	}

	if !account.CanPost() {
		return domain.ErrControlAccountPosting
	}

	return nil
}
