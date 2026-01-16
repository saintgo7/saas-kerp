// Package service provides business logic for tax invoice operations.
package service

import (
	"context"
	"fmt"
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

	if err := invoice.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := s.repo.Create(ctx, invoice); err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// Create items
	for i, itemInput := range input.Items {
		item := &domain.TaxInvoiceItem{
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
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if err := s.repo.CreateItem(ctx, item); err != nil {
			return nil, fmt.Errorf("failed to create item: %w", err)
		}
		invoice.Items = append(invoice.Items, *item)
	}

	// Create history
	history := &domain.TaxInvoiceHistory{
		ID:           uuid.New(),
		TaxInvoiceID: invoice.ID,
		CompanyID:    companyID,
		NewStatus:    domain.TaxInvoiceStatusDraft,
		ChangedBy:    userID,
		ChangeReason: "Invoice created",
		CreatedAt:    time.Now(),
	}
	_ = s.repo.CreateHistory(ctx, history)

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

	oldStatus := invoice.Status
	invoice.Status = domain.TaxInvoiceStatusIssued
	invoice.UpdatedBy = userID
	invoice.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, invoice); err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	// Create history
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
	_ = s.repo.CreateHistory(ctx, history)

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

	// Call gRPC service to transmit
	if s.grpcClient != nil {
		resp, err := s.grpcClient.IssueTaxInvoice(ctx, &grpcclient.IssueTaxInvoiceRequest{
			SessionID: sessionID,
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

		invoice.NTSConfirmNumber = resp.NTSConfirmNumber
		now := time.Now()
		invoice.NTSTransmittedAt = &now
	}

	oldStatus := invoice.Status
	invoice.Status = domain.TaxInvoiceStatusTransmitted
	invoice.UpdatedBy = userID
	invoice.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, invoice); err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	// Create history
	history := &domain.TaxInvoiceHistory{
		ID:             uuid.New(),
		TaxInvoiceID:   invoice.ID,
		CompanyID:      companyID,
		PreviousStatus: oldStatus,
		NewStatus:      domain.TaxInvoiceStatusTransmitted,
		ChangedBy:      userID,
		ChangeReason:   "Invoice transmitted to NTS",
		CreatedAt:      time.Now(),
	}
	_ = s.repo.CreateHistory(ctx, history)

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

	if err := s.repo.Update(ctx, invoice); err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	// Create history
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
	_ = s.repo.CreateHistory(ctx, history)

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

	// Delete items first
	if err := s.repo.DeleteItems(ctx, companyID, id); err != nil {
		return fmt.Errorf("failed to delete items: %w", err)
	}

	if err := s.repo.Delete(ctx, companyID, id); err != nil {
		return fmt.Errorf("failed to delete invoice: %w", err)
	}

	return nil
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

	resp, err := s.grpcClient.GetTaxInvoices(ctx, &grpcclient.GetTaxInvoicesRequest{
		SessionID: sessionID,
		StartDate: startDate,
		EndDate:   endDate,
		Page:      1,
		PageSize:  1000,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get invoices from Hometax: %w", err)
	}

	if !resp.Success {
		return 0, fmt.Errorf("Hometax sync failed: %s", resp.ErrorMessage)
	}

	synced := 0
	for _, inv := range resp.Invoices {
		// Check if invoice already exists
		_, err := s.repo.GetByNumber(ctx, companyID, inv.InvoiceNumber, domain.TaxInvoiceType(inv.InvoiceType))
		if err == nil {
			// Invoice exists, skip
			continue
		}

		// Create new invoice
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

		_, err = s.Create(ctx, companyID, input, userID)
		if err != nil {
			continue // Log and continue
		}
		synced++
	}

	return synced, nil
}
