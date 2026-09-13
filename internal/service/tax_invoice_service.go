// Package service provides business logic for tax invoice operations.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/grpcclient"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// TaxInvoiceFilter is re-exported from repository for convenience
type TaxInvoiceFilter = repository.TaxInvoiceFilter

// TaxInvoiceService provides business logic for tax invoice operations.
type TaxInvoiceService struct {
	repo       repository.TaxInvoiceRepository
	grpcClient *grpcclient.TaxInvoiceClient
}

// NewTaxInvoiceService creates a new tax invoice service.
func NewTaxInvoiceService(repo repository.TaxInvoiceRepository, grpcClient *grpcclient.TaxInvoiceClient) *TaxInvoiceService {
	return &TaxInvoiceService{
		repo:       repo,
		grpcClient: grpcClient,
	}
}

// CreateInput represents input for creating a tax invoice.
type CreateInput struct {
	InvoiceNumber          string
	InvoiceType            domain.TaxInvoiceType
	IssueDate              time.Time
	SupplierBusinessNumber string
	SupplierName           string
	SupplierCEOName        string
	SupplierAddress        string
	BuyerBusinessNumber    string
	BuyerName              string
	BuyerCEOName           string
	BuyerAddress           string
	SupplyAmount           int64
	TaxAmount              int64
	Items                  []CreateItemInput
	Remarks                string
}

// CreateItemInput represents input for creating a tax invoice item.
type CreateItemInput struct {
	SupplyDate    *time.Time
	Description   string
	Specification string
	Quantity      float64
	UnitPrice     float64
	Amount        int64
	TaxAmount     int64
	Remarks       string
}

// Create creates a new tax invoice.
func (s *TaxInvoiceService) Create(ctx context.Context, companyID uuid.UUID, input *CreateInput, userID *uuid.UUID) (*domain.TaxInvoice, error) {
	invoice := &domain.TaxInvoice{
		ID:                     uuid.New(),
		CompanyID:              companyID,
		InvoiceNumber:          input.InvoiceNumber,
		InvoiceType:            input.InvoiceType,
		IssueDate:              input.IssueDate,
		Status:                 domain.TaxInvoiceStatusDraft,
		SupplierBusinessNumber: input.SupplierBusinessNumber,
		SupplierName:           input.SupplierName,
		SupplierCEOName:        input.SupplierCEOName,
		SupplierAddress:        input.SupplierAddress,
		BuyerBusinessNumber:    input.BuyerBusinessNumber,
		BuyerName:              input.BuyerName,
		BuyerCEOName:           input.BuyerCEOName,
		BuyerAddress:           input.BuyerAddress,
		SupplyAmount:           input.SupplyAmount,
		TaxAmount:              input.TaxAmount,
		TotalAmount:            input.SupplyAmount + input.TaxAmount,
		Remarks:                input.Remarks,
		CreatedBy:              userID,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	// Build the items before anything is written so the header/item
	// reconciliation happens on the complete document.
	now := time.Now()
	items := make([]domain.TaxInvoiceItem, 0, len(input.Items))
	for i, itemInput := range input.Items {
		items = append(items, domain.TaxInvoiceItem{
			ID:             uuid.New(),
			TaxInvoiceID:   invoice.ID,
			CompanyID:      companyID,
			SequenceNumber: i + 1,
			SupplyDate:     itemInput.SupplyDate,
			Description:    itemInput.Description,
			Specification:  itemInput.Specification,
			Quantity:       itemInput.Quantity,
			UnitPrice:      itemInput.UnitPrice,
			Amount:         itemInput.Amount,
			TaxAmount:      itemInput.TaxAmount,
			Remarks:        itemInput.Remarks,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}
	invoice.Items = items

	// Header validation (amounts, business numbers, VAT rate) plus the
	// header-to-items reconciliation. Amounts used to be stored exactly as the
	// client sent them, with only total = supply + tax checked, so an invoice
	// declaring 10,000,000 of supply and 0 tax was accepted and filed.
	if err := invoice.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	if len(items) > 0 {
		if err := invoice.ValidateItems(); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	history := &domain.TaxInvoiceHistory{
		ID:           uuid.New(),
		TaxInvoiceID: invoice.ID,
		CompanyID:    companyID,
		NewStatus:    domain.TaxInvoiceStatusDraft,
		ChangedBy:    userID,
		ChangeReason: "Invoice created",
		CreatedAt:    now,
	}

	// One transaction for the header, the items and the history row. They used
	// to be three independent commits, so a failure partway through left a
	// header with no items - an invoice whose amounts can no longer be
	// reconciled against anything.
	err := s.repo.WithTransaction(ctx, func(repo repository.TaxInvoiceRepository) error {
		if err := repo.Create(ctx, invoice); err != nil {
			return fmt.Errorf("failed to create invoice: %w", err)
		}
		for i := range items {
			if err := repo.CreateItem(ctx, &items[i]); err != nil {
				return fmt.Errorf("failed to create item: %w", err)
			}
		}
		// The history row is the audit trail for a document that can be
		// disputed with the tax authority; its failure must not be swallowed.
		if err := repo.CreateHistory(ctx, history); err != nil {
			return fmt.Errorf("failed to record history: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return invoice, nil
}

// GetByID retrieves a tax invoice by ID.
func (s *TaxInvoiceService) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.TaxInvoice, error) {
	invoice, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	// Load items
	items, err := s.repo.ListItems(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}

	for _, item := range items {
		invoice.Items = append(invoice.Items, *item)
	}

	return invoice, nil
}

// List retrieves tax invoices with filtering.
func (s *TaxInvoiceService) List(ctx context.Context, filter *TaxInvoiceFilter) ([]*domain.TaxInvoice, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	return s.repo.List(ctx, filter)
}

// Issue issues a draft tax invoice.
func (s *TaxInvoiceService) Issue(ctx context.Context, companyID, id uuid.UUID, userID *uuid.UUID) (*domain.TaxInvoice, error) {
	invoice, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	if !invoice.CanBeModified() {
		return nil, fmt.Errorf("invoice cannot be modified in status: %s", invoice.Status)
	}

	// Re-validate against what is actually stored. Issuing is the point of no
	// return - after this the document is transmitted to the NTS - so the
	// amounts are reconciled against the stored items here rather than trusted
	// from creation time.
	storedItems, err := s.repo.ListItems(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}
	invoice.Items = invoice.Items[:0]
	for _, item := range storedItems {
		invoice.Items = append(invoice.Items, *item)
	}
	if err := invoice.ValidateForIssue(); err != nil {
		return nil, fmt.Errorf("invoice cannot be issued: %w", err)
	}

	oldStatus := invoice.Status
	invoice.Status = domain.TaxInvoiceStatusIssued
	invoice.UpdatedBy = userID
	invoice.UpdatedAt = time.Now()

	history := &domain.TaxInvoiceHistory{
		ID:             uuid.New(),
		TaxInvoiceID:   invoice.ID,
		CompanyID:      companyID,
		PreviousStatus: oldStatus,
		NewStatus:      domain.TaxInvoiceStatusIssued,
		ChangedBy:      userID,
		ChangeReason:   "Invoice issued",
		CreatedAt:      time.Now(),
	}

	if err := s.repo.WithTransaction(ctx, func(repo repository.TaxInvoiceRepository) error {
		if err := repo.Update(ctx, invoice); err != nil {
			return fmt.Errorf("failed to update invoice: %w", err)
		}
		if err := repo.CreateHistory(ctx, history); err != nil {
			return fmt.Errorf("failed to record history: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return invoice, nil
}

// TransmitToNTS transmits the invoice to National Tax Service via gRPC.
func (s *TaxInvoiceService) TransmitToNTS(ctx context.Context, companyID, id uuid.UUID, sessionID string, userID *uuid.UUID) (*domain.TaxInvoice, error) {
	invoice, err := s.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}

	if invoice.Status != domain.TaxInvoiceStatusIssued {
		return nil, fmt.Errorf("invoice must be issued before transmission")
	}

	// Idempotency guard. If a confirmation number is already recorded the NTS
	// has accepted this invoice, and sending it again produces a duplicate
	// filing that can only be undone with a 수정세금계산서.
	if invoice.NTSConfirmNumber != "" {
		return nil, fmt.Errorf("invoice has already been transmitted to NTS (승인번호 %s)", invoice.NTSConfirmNumber)
	}

	// Fail closed. Without a gRPC client the invoice used to be marked
	// transmitted without anything having been sent, so the operator believed
	// the filing was done.
	if s.grpcClient == nil {
		return nil, fmt.Errorf("NTS gRPC client not configured; refusing to mark invoice as transmitted")
	}

	// Last validation before the document leaves the system.
	if err := invoice.ValidateForIssue(); err != nil {
		return nil, fmt.Errorf("invoice cannot be transmitted: %w", err)
	}

	resp, err := s.grpcClient.IssueTaxInvoice(ctx, &grpcclient.IssueTaxInvoiceRequest{
		SessionID: sessionID,
		// The tax scraper cannot verify that the caller owns the session it
		// named without this, and rejects the call with UNAUTHENTICATED.
		// Omitting it is how one company could act on another company's
		// 세금계산서 through a leaked session id.
		CompanyID: companyID.String(),
		// The invoice id is stable across retries of the same issuance, which
		// is exactly what the scraper needs to keep a retry from producing a
		// second 세금계산서 at the NTS.
		IdempotencyKey: invoice.ID.String(),
		Invoice: grpcclient.TaxInvoice{
			InvoiceNumber:          invoice.InvoiceNumber,
			IssueDate:              invoice.IssueDate,
			InvoiceType:            string(invoice.InvoiceType),
			SupplierBusinessNumber: invoice.SupplierBusinessNumber,
			SupplierName:           invoice.SupplierName,
			BuyerBusinessNumber:    invoice.BuyerBusinessNumber,
			BuyerName:              invoice.BuyerName,
			SupplyAmount:           invoice.SupplyAmount,
			TaxAmount:              invoice.TaxAmount,
			TotalAmount:            invoice.TotalAmount,
		},
		TransmitImmediately: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to transmit to NTS: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("NTS transmission failed: %s", resp.ErrorMessage)
	}

	// A success with no confirmation number is not a success. Accepting it
	// would mark the invoice transmitted with nts_confirm_number empty, and
	// the idempotency guard above keys on exactly that field - so the next
	// retry would sail past it and file the invoice a second time.
	if resp.NTSConfirmNumber == "" {
		return nil, fmt.Errorf("NTS reported success without a confirmation number; refusing to mark invoice as transmitted")
	}

	transmittedAt := time.Now()
	invoice.NTSConfirmNumber = resp.NTSConfirmNumber
	invoice.NTSTransmittedAt = &transmittedAt

	oldStatus := invoice.Status
	invoice.Status = domain.TaxInvoiceStatusTransmitted
	invoice.UpdatedBy = userID
	invoice.UpdatedAt = transmittedAt

	history := &domain.TaxInvoiceHistory{
		ID:             uuid.New(),
		TaxInvoiceID:   invoice.ID,
		CompanyID:      companyID,
		PreviousStatus: oldStatus,
		NewStatus:      domain.TaxInvoiceStatusTransmitted,
		ChangedBy:      userID,
		ChangeReason:   "Invoice transmitted to NTS",
		CreatedAt:      transmittedAt,
	}

	// The confirmation number, the transmission timestamp and the status are
	// written together. If this fails the NTS has the invoice but we have no
	// record of it, so the error names the confirmation number: it is the only
	// place it still exists, and re-transmitting without it duplicates the
	// filing.
	if err := s.repo.WithTransaction(ctx, func(repo repository.TaxInvoiceRepository) error {
		if err := repo.Update(ctx, invoice); err != nil {
			return err
		}
		return repo.CreateHistory(ctx, history)
	}); err != nil {
		return nil, fmt.Errorf(
			"NTS accepted the invoice (승인번호 %s, 전송시각 %s) but persisting it failed - "+
				"record the confirmation number manually and do NOT re-transmit: %w",
			resp.NTSConfirmNumber, transmittedAt.Format(time.RFC3339), err)
	}

	return invoice, nil
}

// Cancel cancels an issued or transmitted invoice.
func (s *TaxInvoiceService) Cancel(ctx context.Context, companyID, id uuid.UUID, reason string, userID *uuid.UUID) (*domain.TaxInvoice, error) {
	invoice, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	if !invoice.CanBeCancelled() {
		return nil, fmt.Errorf("invoice cannot be cancelled in status: %s", invoice.Status)
	}

	oldStatus := invoice.Status
	invoice.Status = domain.TaxInvoiceStatusCancelled
	invoice.UpdatedBy = userID
	invoice.UpdatedAt = time.Now()

	history := &domain.TaxInvoiceHistory{
		ID:             uuid.New(),
		TaxInvoiceID:   invoice.ID,
		CompanyID:      companyID,
		PreviousStatus: oldStatus,
		NewStatus:      domain.TaxInvoiceStatusCancelled,
		ChangedBy:      userID,
		ChangeReason:   reason,
		CreatedAt:      time.Now(),
	}

	// The cancellation reason is the audit trail for a tax document; losing it
	// while the status change succeeds leaves a cancelled invoice nobody can
	// account for.
	if err := s.repo.WithTransaction(ctx, func(repo repository.TaxInvoiceRepository) error {
		if err := repo.Update(ctx, invoice); err != nil {
			return fmt.Errorf("failed to update invoice: %w", err)
		}
		if err := repo.CreateHistory(ctx, history); err != nil {
			return fmt.Errorf("failed to record history: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return invoice, nil
}

// Delete deletes a draft tax invoice.
func (s *TaxInvoiceService) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	invoice, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return fmt.Errorf("failed to get invoice: %w", err)
	}

	if !invoice.CanBeModified() {
		return fmt.Errorf("only draft invoices can be deleted")
	}

	// One transaction: deleting the items and then failing to delete the
	// header left an invoice whose amounts can no longer be reconciled.
	return s.repo.WithTransaction(ctx, func(repo repository.TaxInvoiceRepository) error {
		if err := repo.DeleteItems(ctx, companyID, id); err != nil {
			return fmt.Errorf("failed to delete items: %w", err)
		}
		if err := repo.Delete(ctx, companyID, id); err != nil {
			return fmt.Errorf("failed to delete invoice: %w", err)
		}
		return nil
	})
}

// GetSummary retrieves aggregated tax invoice data.
func (s *TaxInvoiceService) GetSummary(ctx context.Context, companyID uuid.UUID, startDate, endDate time.Time) (*domain.TaxInvoiceSummary, error) {
	return s.repo.GetSummary(ctx, companyID, startDate, endDate)
}

// SyncFromHometax syncs tax invoices from Hometax via gRPC.
func (s *TaxInvoiceService) SyncFromHometax(ctx context.Context, companyID uuid.UUID, sessionID string, startDate, endDate string, userID *uuid.UUID) (int, error) {
	if s.grpcClient == nil {
		return 0, fmt.Errorf("gRPC client not configured")
	}

	const pageSize = 500

	synced := 0
	var failures []string

	// Walk every page. The previous version requested page 1 with a page size
	// of 1000 and stopped, so a period with more than 1000 invoices was
	// silently truncated and the operator was told the sync had completed.
	for page := int32(1); ; page++ {
		resp, err := s.grpcClient.GetTaxInvoices(ctx, &grpcclient.GetTaxInvoicesRequest{
			SessionID: sessionID,
			CompanyID: companyID.String(),
			StartDate: startDate,
			EndDate:   endDate,
			Page:      page,
			PageSize:  pageSize,
		})
		if err != nil {
			return synced, fmt.Errorf("failed to get invoices from Hometax (page %d): %w", page, err)
		}
		if !resp.Success {
			return synced, fmt.Errorf("Hometax sync failed (page %d): %s", page, resp.ErrorMessage)
		}
		if len(resp.Invoices) == 0 {
			break
		}

		for _, inv := range resp.Invoices {
			// Distinguish "not present yet" from a database failure. Treating
			// every error as not-found meant a dropped connection turned into
			// a duplicate insert attempt for every invoice in the page.
			_, err := s.repo.GetByNumber(ctx, companyID, inv.InvoiceNumber, domain.TaxInvoiceType(inv.InvoiceType))
			switch {
			case err == nil:
				continue // already stored
			case errors.Is(err, domain.ErrTaxInvoiceNotFound):
				// fall through and create it
			default:
				return synced, fmt.Errorf("failed to look up invoice %s: %w", inv.InvoiceNumber, err)
			}

			input := &CreateInput{
				InvoiceNumber:          inv.InvoiceNumber,
				InvoiceType:            domain.TaxInvoiceType(inv.InvoiceType),
				IssueDate:              inv.IssueDate,
				SupplierBusinessNumber: inv.SupplierBusinessNumber,
				SupplierName:           inv.SupplierName,
				SupplierCEOName:        inv.SupplierCEOName,
				BuyerBusinessNumber:    inv.BuyerBusinessNumber,
				BuyerName:              inv.BuyerName,
				BuyerCEOName:           inv.BuyerCEOName,
				SupplyAmount:           inv.SupplyAmount,
				TaxAmount:              inv.TaxAmount,
				Remarks:                inv.Remarks,
			}

			if _, err := s.Create(ctx, companyID, input, userID); err != nil {
				// Per-invoice failures are collected rather than swallowed:
				// the caller was previously told how many synced but never
				// which ones had not.
				failures = append(failures, fmt.Sprintf("%s: %v", inv.InvoiceNumber, err))
				continue
			}
			synced++
		}

		if len(resp.Invoices) < pageSize {
			break
		}
	}

	if len(failures) > 0 {
		return synced, fmt.Errorf("%d개 세금계산서를 저장하지 못했습니다: %s",
			len(failures), strings.Join(failures, "; "))
	}

	return synced, nil
}
