package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MockTaxInvoiceProvider is a mock implementation for testing
type MockTaxInvoiceProvider struct {
	priority    int
	available   bool
	quota       *ProviderQuota
	invoices    map[string]*TaxInvoiceData
	shouldFail  bool
	failMessage string
}

// NewMockTaxInvoiceProvider creates a new mock provider
func NewMockTaxInvoiceProvider() *MockTaxInvoiceProvider {
	return &MockTaxInvoiceProvider{
		priority:  100, // Low priority (for fallback)
		available: true,
		quota: &ProviderQuota{
			MonthlyLimit: 1000,
			MonthlyUsed:  0,
			DailyLimit:   100,
			DailyUsed:    0,
			ResetAt:      time.Now().AddDate(0, 1, 0),
		},
		invoices: make(map[string]*TaxInvoiceData),
	}
}

// Type returns the provider type
func (p *MockTaxInvoiceProvider) Type() ProviderType {
	return ProviderTypeMock
}

// Name returns the provider name
func (p *MockTaxInvoiceProvider) Name() string {
	return "Mock Provider"
}

// IsAvailable returns availability status
func (p *MockTaxInvoiceProvider) IsAvailable(ctx context.Context) bool {
	return p.available
}

// Health returns the health status
func (p *MockTaxInvoiceProvider) Health(ctx context.Context) *ProviderHealth {
	status := ProviderStatusActive
	if !p.available {
		status = ProviderStatusInactive
	}
	return &ProviderHealth{
		Type:        ProviderTypeMock,
		Status:      status,
		LastChecked: time.Now(),
		Latency:     time.Millisecond * 10,
		Quota:       p.quota,
	}
}

// Priority returns the priority
func (p *MockTaxInvoiceProvider) Priority() int {
	return p.priority
}

// Close closes the provider
func (p *MockTaxInvoiceProvider) Close() error {
	return nil
}

// SetAvailable sets the availability for testing
func (p *MockTaxInvoiceProvider) SetAvailable(available bool) {
	p.available = available
}

// SetShouldFail sets whether operations should fail
func (p *MockTaxInvoiceProvider) SetShouldFail(shouldFail bool, message string) {
	p.shouldFail = shouldFail
	p.failMessage = message
}

// SetPriority sets the priority for testing
func (p *MockTaxInvoiceProvider) SetPriority(priority int) {
	p.priority = priority
}

// Issue issues a tax invoice
func (p *MockTaxInvoiceProvider) Issue(ctx context.Context, companyID uuid.UUID, invoice *TaxInvoiceData) (*TaxInvoiceIssueResult, error) {
	if p.shouldFail {
		return &TaxInvoiceIssueResult{
			Success:      false,
			ErrorCode:    "MOCK_ERROR",
			ErrorMessage: p.failMessage,
		}, fmt.Errorf(p.failMessage)
	}

	// Generate mock NTS confirm number
	confirmNumber := fmt.Sprintf("MOCK-%s-%d", time.Now().Format("20060102"), time.Now().UnixNano()%1000000)
	now := time.Now()

	// Store invoice
	invoice.ID = uuid.New()
	p.invoices[confirmNumber] = invoice

	// Update quota
	p.quota.MonthlyUsed++
	p.quota.DailyUsed++

	return &TaxInvoiceIssueResult{
		Success:          true,
		NTSConfirmNumber: confirmNumber,
		ASPInvoiceID:     fmt.Sprintf("MOCK-ASP-%d", time.Now().UnixNano()%1000000),
		TransmittedAt:    &now,
	}, nil
}

// Cancel cancels a tax invoice
func (p *MockTaxInvoiceProvider) Cancel(ctx context.Context, companyID uuid.UUID, ntsConfirmNumber string, reason string) (*TaxInvoiceIssueResult, error) {
	if p.shouldFail {
		return &TaxInvoiceIssueResult{
			Success:      false,
			ErrorCode:    "MOCK_ERROR",
			ErrorMessage: p.failMessage,
		}, fmt.Errorf(p.failMessage)
	}

	// Check if invoice exists
	if _, exists := p.invoices[ntsConfirmNumber]; !exists {
		return &TaxInvoiceIssueResult{
			Success:      false,
			ErrorCode:    "NOT_FOUND",
			ErrorMessage: "Invoice not found",
		}, fmt.Errorf("invoice not found")
	}

	// Remove from store
	delete(p.invoices, ntsConfirmNumber)

	return &TaxInvoiceIssueResult{
		Success:          true,
		NTSConfirmNumber: ntsConfirmNumber,
	}, nil
}

// Search searches for tax invoices
func (p *MockTaxInvoiceProvider) Search(ctx context.Context, companyID uuid.UUID, filter *TaxInvoiceSearchFilter) (*TaxInvoiceSearchResult, error) {
	if p.shouldFail {
		return nil, fmt.Errorf(p.failMessage)
	}

	// Return all stored invoices (simplified mock)
	result := &TaxInvoiceSearchResult{
		Invoices:   make([]TaxInvoiceData, 0, len(p.invoices)),
		TotalCount: len(p.invoices),
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		HasMore:    false,
	}

	for _, inv := range p.invoices {
		result.Invoices = append(result.Invoices, *inv)
	}

	return result, nil
}

// GetByConfirmNumber retrieves a tax invoice by confirm number
func (p *MockTaxInvoiceProvider) GetByConfirmNumber(ctx context.Context, companyID uuid.UUID, confirmNumber string) (*TaxInvoiceData, error) {
	if p.shouldFail {
		return nil, fmt.Errorf(p.failMessage)
	}

	inv, exists := p.invoices[confirmNumber]
	if !exists {
		return nil, fmt.Errorf("invoice not found")
	}

	return inv, nil
}

// GetQuota returns the current quota
func (p *MockTaxInvoiceProvider) GetQuota(ctx context.Context, companyID uuid.UUID) (*ProviderQuota, error) {
	return p.quota, nil
}

// GetStoredInvoices returns all stored invoices (for testing)
func (p *MockTaxInvoiceProvider) GetStoredInvoices() map[string]*TaxInvoiceData {
	return p.invoices
}
