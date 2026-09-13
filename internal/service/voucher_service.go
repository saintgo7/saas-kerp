package service

import (
	"context"
	"errors"
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

	// Entry operations.
	//
	// UpdateEntry and RemoveEntry take the owning company and voucher
	// explicitly. Without them the repository filtered on the entry UUID
	// alone, so a caller holding a bare entry id could mutate a posted
	// voucher - or another tenant's voucher - and the header totals were
	// never recomputed.
	AddEntry(ctx context.Context, voucherID uuid.UUID, entry *domain.VoucherEntry) error
	UpdateEntry(ctx context.Context, companyID, voucherID uuid.UUID, entry *domain.VoucherEntry) error
	RemoveEntry(ctx context.Context, companyID, voucherID, entryID uuid.UUID) error
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
	ledgerRepo  repository.LedgerRepository
}

// NewVoucherService creates a new VoucherService.
//
// ledgerRepo is variadic only so that this signature stays source compatible
// with existing call sites; it is not optional in any meaningful sense. Post
// needs it to refuse postings into a closed fiscal period, and when it is not
// supplied that check cannot run. Wire it:
//
//	service.NewVoucherService(voucherRepo, accountRepo, ledgerRepo)
func NewVoucherService(
	voucherRepo repository.VoucherRepository,
	accountRepo repository.AccountRepository,
	ledgerRepo ...repository.LedgerRepository,
) VoucherService {
	svc := &voucherService{
		voucherRepo: voucherRepo,
		accountRepo: accountRepo,
	}
	if len(ledgerRepo) > 0 {
		svc.ledgerRepo = ledgerRepo[0]
	}
	return svc
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

// AddEntry adds an entry to a voucher.
//
// The line insert and the header total update run in one transaction. They
// used to be two independent commits: the line was committed first, and the
// following header UPDATE could then be rejected by the chk_voucher_balance
// constraint (total_debit = total_credit). The caller saw the error, but the
// line stayed in the database forever - SUM(voucher_entries) no longer matched
// the header, the account ledger was permanently off by that amount, and
// Submit's balance check only inspects the header so it never noticed.
func (s *voucherService) AddEntry(ctx context.Context, voucherID uuid.UUID, entry *domain.VoucherEntry) error {
	entry.Normalize()

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

	return s.voucherRepo.WithTransaction(ctx, func(repo repository.VoucherRepository) error {
		if err := repo.CreateEntry(ctx, entry); err != nil {
			return err
		}

		// Recalculate voucher totals from every line, including the new one.
		voucher.Entries = append(voucher.Entries, *entry)
		voucher.CalculateTotals()

		// Refuse the write before the database does. A draft may legitimately
		// be unbalanced while it is being assembled, but chk_voucher_balance
		// does not allow an unbalanced header to be stored, so the whole unit
		// of work has to be rejected here rather than half-applied.
		if err := voucher.ValidateBalance(); err != nil {
			return err
		}

		return repo.Update(ctx, voucher)
	})
}

// UpdateEntry updates an existing entry and recomputes the header totals in
// the same transaction.
func (s *voucherService) UpdateEntry(ctx context.Context, companyID, voucherID uuid.UUID, entry *domain.VoucherEntry) error {
	entry.Normalize()

	if err := entry.Validate(); err != nil {
		return err
	}

	voucher, err := s.voucherRepo.FindByID(ctx, companyID, voucherID)
	if err != nil {
		return err
	}
	if !voucher.CanEdit() {
		return domain.ErrVoucherCannotEdit
	}
	if err := s.validateAccountForPosting(ctx, companyID, entry.AccountID); err != nil {
		return err
	}

	// Make sure the entry really belongs to this voucher before writing it.
	found := false
	for i := range voucher.Entries {
		if voucher.Entries[i].ID == entry.ID {
			entry.VoucherID = voucherID
			entry.CompanyID = companyID
			if entry.LineNo == 0 {
				entry.LineNo = voucher.Entries[i].LineNo
			}
			voucher.Entries[i] = *entry
			found = true
			break
		}
	}
	if !found {
		return domain.ErrEntryNotFound
	}

	return s.voucherRepo.WithTransaction(ctx, func(repo repository.VoucherRepository) error {
		if err := repo.UpdateEntry(ctx, entry); err != nil {
			return err
		}
		voucher.CalculateTotals()
		if err := voucher.ValidateBalance(); err != nil {
			return err
		}
		return repo.Update(ctx, voucher)
	})
}

// RemoveEntry removes an entry from a voucher and recomputes the header
// totals in the same transaction.
func (s *voucherService) RemoveEntry(ctx context.Context, companyID, voucherID, entryID uuid.UUID) error {
	voucher, err := s.voucherRepo.FindByID(ctx, companyID, voucherID)
	if err != nil {
		return err
	}
	if !voucher.CanEdit() {
		return domain.ErrVoucherCannotEdit
	}

	remaining := make([]domain.VoucherEntry, 0, len(voucher.Entries))
	found := false
	for _, e := range voucher.Entries {
		if e.ID == entryID {
			found = true
			continue
		}
		remaining = append(remaining, e)
	}
	if !found {
		return domain.ErrEntryNotFound
	}
	if len(remaining) == 0 {
		return domain.ErrVoucherNoEntries
	}

	return s.voucherRepo.WithTransaction(ctx, func(repo repository.VoucherRepository) error {
		if err := repo.DeleteEntry(ctx, companyID, entryID); err != nil {
			return err
		}
		voucher.Entries = remaining
		voucher.CalculateTotals()
		if err := voucher.ValidateBalance(); err != nil {
			return err
		}
		return repo.Update(ctx, voucher)
	})
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
		if err := repo.DeleteEntriesByVoucher(ctx, voucher.CompanyID, voucherID); err != nil {
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

// Post posts a voucher to the ledger.
//
// Three checks run before the status changes, none of which existed before:
//
//  1. the voucher still has lines,
//  2. the stored lines still balance and still agree with the stored header
//     (a voucher whose header and lines drifted apart must not reach the
//     ledger), and
//  3. the fiscal period covering voucher_date is still open.
//
// Without (3), FiscalPeriod.CanPost and domain.ErrPeriodClosed were dead code:
// a voucher dated in an already closed and carried-forward month could be
// posted, and the next recalculation would silently rewrite a trial balance
// that had already been published.
func (s *voucherService) Post(ctx context.Context, companyID, voucherID, userID uuid.UUID) error {
	voucher, err := s.voucherRepo.FindByID(ctx, companyID, voucherID)
	if err != nil {
		return err
	}

	// Status first, so a voucher in the wrong state reports that rather than a
	// downstream data error.
	if !voucher.Status.CanPost() {
		return domain.ErrVoucherCannotPost
	}

	if len(voucher.Entries) == 0 {
		return domain.ErrVoucherNoEntries
	}

	// Re-derive the totals from the stored lines and compare them with the
	// stored header before trusting either.
	storedDebit, storedCredit := voucher.TotalDebit, voucher.TotalCredit
	voucher.CalculateTotals()
	if !domain.AmountsEqual(storedDebit, voucher.TotalDebit) ||
		!domain.AmountsEqual(storedCredit, voucher.TotalCredit) {
		return domain.ErrVoucherTotalsMismatch
	}
	if err := voucher.ValidateBalance(); err != nil {
		return err
	}

	if err := s.checkPeriodOpen(ctx, companyID, voucher.VoucherDate); err != nil {
		return err
	}

	if err := voucher.Post(userID); err != nil {
		return err
	}

	return s.voucherRepo.UpdateStatus(ctx, voucher)
}

// checkPeriodOpen refuses postings into a closed or locked fiscal period.
//
// A period that has no fiscal_periods row at all is treated as open: many
// tenants never call CreateFiscalPeriods, and refusing every posting for them
// would be a far bigger regression than the case this guards against.
func (s *voucherService) checkPeriodOpen(ctx context.Context, companyID uuid.UUID, date time.Time) error {
	if s.ledgerRepo == nil {
		// Not wired: see NewVoucherService. The check cannot run.
		return nil
	}

	period, err := s.ledgerRepo.GetFiscalPeriod(ctx, companyID, date.Year(), int(date.Month()))
	if err != nil {
		if errors.Is(err, domain.ErrFiscalPeriodNotFound) {
			return nil
		}
		return err
	}
	if !period.CanPost() {
		return domain.ErrPeriodClosed
	}
	return nil
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
		VoucherDate:  reversalDate,
		VoucherType:  original.VoucherType,
		Status:       domain.VoucherStatusDraft,
		Description:  description,
		IsReversal:   true,
		ReversalOfID: &original.ID,
		CreatedBy:    &userID,
	}

	// Create reversed entries (swap debit and credit)
	for _, entry := range original.Entries {
		reversalEntry := domain.VoucherEntry{
			CompanyID:    companyID,
			AccountID:    entry.AccountID,
			DebitAmount:  entry.CreditAmount, // Swap
			CreditAmount: entry.DebitAmount,  // Swap
			Description:  entry.Description,
			PartnerID:    entry.PartnerID,
			DepartmentID: entry.DepartmentID,
			ProjectID:    entry.ProjectID,
			CostCenterID: entry.CostCenterID,
		}
		reversal.Entries = append(reversal.Entries, reversalEntry)
	}

	// Creating the reversal and stamping the original run in one transaction.
	// Previously they were two commits, so a failure in between left an orphan
	// reversal voucher behind and a retry produced a second one.
	//
	// MarkReversed is used instead of Update because Update's column whitelist
	// silently dropped reversed_by_id: the UPDATE reported success, the column
	// stayed NULL, and the "already reversed" guard above could never fire. It
	// also updates conditionally on reversed_by_id IS NULL, so two concurrent
	// reversals cannot both succeed.
	err = s.voucherRepo.WithTransaction(ctx, func(repo repository.VoucherRepository) error {
		txService := &voucherService{
			voucherRepo: repo,
			accountRepo: s.accountRepo,
			ledgerRepo:  s.ledgerRepo,
		}
		if err := txService.Create(ctx, reversal); err != nil {
			return err
		}
		return repo.MarkReversed(ctx, companyID, original.ID, reversal.ID)
	})
	if err != nil {
		return nil, err
	}

	original.ReversedByID = &reversal.ID
	return reversal, nil
}

// ValidateEntries validates all entries for a voucher.
//
// Amounts are quantized to the stored scale first and summed from the
// quantized values, then compared with BalanceEpsilon. Summing the raw
// float64 inputs and comparing with == accepted lines such as
// 33.333/33.333/33.334 against 100.00 as balanced, while PostgreSQL stored
// 33.33 three times and the ledger ended up 0.01 short on every such voucher.
func (s *voucherService) ValidateEntries(ctx context.Context, companyID uuid.UUID, entries []domain.VoucherEntry) error {
	var totalDebit, totalCredit float64

	for i := range entries {
		entries[i].Normalize()

		// Validate entry
		if err := entries[i].Validate(); err != nil {
			return err
		}

		// Validate account can accept postings
		if err := s.validateAccountForPosting(ctx, companyID, entries[i].AccountID); err != nil {
			return err
		}

		totalDebit += entries[i].DebitAmount
		totalCredit += entries[i].CreditAmount
	}

	// Check balance
	if !domain.AmountsEqual(domain.RoundAmount(totalDebit), domain.RoundAmount(totalCredit)) {
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
