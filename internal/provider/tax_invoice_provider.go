package provider

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TaxInvoiceData represents tax invoice data for provider operations
type TaxInvoiceData struct {
	ID                     uuid.UUID
	InvoiceNumber          string
	InvoiceType            string // "sales" or "purchase"
	IssueDate              time.Time
	SupplierBusinessNumber string
	SupplierName           string
	SupplierCEOName        string
	SupplierAddress        string
	SupplierBusinessType   string
	SupplierBusinessItem   string
	SupplierEmail          string
	BuyerBusinessNumber    string
	BuyerName              string
	BuyerCEOName           string
	BuyerAddress           string
	BuyerBusinessType      string
	BuyerBusinessItem      string
	BuyerEmail             string
	SupplyAmount           int64
	TaxAmount              int64
	TotalAmount            int64
	Items                  []TaxInvoiceItemData
	Remarks                string
}

// TaxInvoiceItemData represents a line item in tax invoice
type TaxInvoiceItemData struct {
	SequenceNumber int
	SupplyDate     time.Time
	Description    string
	Specification  string
	Quantity       float64
	UnitPrice      float64
	Amount         int64
	TaxAmount      int64
	Remarks        string
}

// TaxInvoiceIssueResult represents the result of issuing a tax invoice
type TaxInvoiceIssueResult struct {
	Success          bool
	NTSConfirmNumber string
	ASPInvoiceID     string
	TransmittedAt    *time.Time
	ErrorCode        string
	ErrorMessage     string
	RawResponse      []byte
}

// TaxInvoiceSearchResult represents search results from provider
type TaxInvoiceSearchResult struct {
	Invoices   []TaxInvoiceData
	TotalCount int
	Page       int
	PageSize   int
	HasMore    bool
}

// TaxInvoiceProvider interface for tax invoice operations
type TaxInvoiceProvider interface {
	Provider

	// Issue issues a new tax invoice
	Issue(ctx context.Context, companyID uuid.UUID, invoice *TaxInvoiceData) (*TaxInvoiceIssueResult, error)

	// Cancel cancels an issued tax invoice
	Cancel(ctx context.Context, companyID uuid.UUID, ntsConfirmNumber string, reason string) (*TaxInvoiceIssueResult, error)

	// Search searches for tax invoices
	Search(ctx context.Context, companyID uuid.UUID, filter *TaxInvoiceSearchFilter) (*TaxInvoiceSearchResult, error)

	// GetByConfirmNumber retrieves a tax invoice by NTS confirm number
	GetByConfirmNumber(ctx context.Context, companyID uuid.UUID, confirmNumber string) (*TaxInvoiceData, error)

	// GetQuota returns the current quota status
	GetQuota(ctx context.Context, companyID uuid.UUID) (*ProviderQuota, error)
}

// TaxInvoiceSearchFilter defines search filter for tax invoices
type TaxInvoiceSearchFilter struct {
	StartDate      time.Time
	EndDate        time.Time
	InvoiceType    string // "sales", "purchase", or empty for both
	BusinessNumber string // Counterparty business number
	Page           int
	PageSize       int
}

// TaxInvoiceProviderChain manages tax invoice providers
type TaxInvoiceProviderChain struct {
	*ProviderChain
}

// NewTaxInvoiceProviderChain creates a new tax invoice provider chain
func NewTaxInvoiceProviderChain(config *ChainConfig) *TaxInvoiceProviderChain {
	return &TaxInvoiceProviderChain{
		ProviderChain: NewProviderChain(config),
	}
}

// Issue issues a tax invoice using the provider chain
func (c *TaxInvoiceProviderChain) Issue(ctx context.Context, companyID uuid.UUID, invoice *TaxInvoiceData) (*ProviderResult, error) {
	available := c.GetAvailableProviders(ctx)
	if len(available) == 0 {
		return nil, ErrProviderUnavailable
	}

	var lastErr error
	for _, p := range available {
		taxProvider, ok := p.(TaxInvoiceProvider)
		if !ok {
			continue
		}

		start := time.Now()
		result, err := taxProvider.Issue(ctx, companyID, invoice)
		duration := time.Since(start)

		if err == nil && result.Success {
			return &ProviderResult{
				Success:  true,
				Provider: p.Type(),
				Data:     result,
				Duration: duration,
			}, nil
		}

		lastErr = err
		if !c.config.EnableFallback {
			break
		}
	}

	if lastErr != nil {
		return &ProviderResult{
			Success: false,
			Error:   lastErr,
		}, lastErr
	}

	return nil, ErrAllProvidersFailed
}

// Cancel cancels a tax invoice using the provider chain
func (c *TaxInvoiceProviderChain) Cancel(ctx context.Context, companyID uuid.UUID, ntsConfirmNumber string, reason string) (*ProviderResult, error) {
	available := c.GetAvailableProviders(ctx)
	if len(available) == 0 {
		return nil, ErrProviderUnavailable
	}

	var lastErr error
	for _, p := range available {
		taxProvider, ok := p.(TaxInvoiceProvider)
		if !ok {
			continue
		}

		start := time.Now()
		result, err := taxProvider.Cancel(ctx, companyID, ntsConfirmNumber, reason)
		duration := time.Since(start)

		if err == nil && result.Success {
			return &ProviderResult{
				Success:  true,
				Provider: p.Type(),
				Data:     result,
				Duration: duration,
			}, nil
		}

		lastErr = err
		if !c.config.EnableFallback {
			break
		}
	}

	if lastErr != nil {
		return &ProviderResult{
			Success: false,
			Error:   lastErr,
		}, lastErr
	}

	return nil, ErrAllProvidersFailed
}

// Search searches for tax invoices using the primary provider
func (c *TaxInvoiceProviderChain) Search(ctx context.Context, companyID uuid.UUID, filter *TaxInvoiceSearchFilter) (*TaxInvoiceSearchResult, error) {
	primary, err := c.GetPrimaryProvider(ctx)
	if err != nil {
		return nil, err
	}

	taxProvider, ok := primary.(TaxInvoiceProvider)
	if !ok {
		return nil, ErrProviderNotFound
	}

	return taxProvider.Search(ctx, companyID, filter)
}
