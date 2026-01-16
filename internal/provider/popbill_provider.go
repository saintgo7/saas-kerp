package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PopbillConfig holds Popbill API configuration
type PopbillConfig struct {
	LinkID    string
	SecretKey string
	IsSandbox bool
	Timeout   time.Duration
}

// PopbillTaxInvoiceProvider implements TaxInvoiceProvider for Popbill API
type PopbillTaxInvoiceProvider struct {
	config   *PopbillConfig
	priority int
	// client will be implemented in Phase 5
}

// NewPopbillTaxInvoiceProvider creates a new Popbill provider
func NewPopbillTaxInvoiceProvider(config *PopbillConfig) *PopbillTaxInvoiceProvider {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &PopbillTaxInvoiceProvider{
		config:   config,
		priority: 1, // High priority (primary provider)
	}
}

// Type returns the provider type
func (p *PopbillTaxInvoiceProvider) Type() ProviderType {
	return ProviderTypePopbill
}

// Name returns the provider name
func (p *PopbillTaxInvoiceProvider) Name() string {
	if p.config.IsSandbox {
		return "Popbill (Sandbox)"
	}
	return "Popbill"
}

// IsAvailable checks if the provider is available
func (p *PopbillTaxInvoiceProvider) IsAvailable(ctx context.Context) bool {
	// TODO: Implement actual availability check via API in Phase 5
	return p.config.LinkID != "" && p.config.SecretKey != ""
}

// Health returns the health status
func (p *PopbillTaxInvoiceProvider) Health(ctx context.Context) *ProviderHealth {
	health := &ProviderHealth{
		Type:        ProviderTypePopbill,
		Status:      ProviderStatusActive,
		LastChecked: time.Now(),
	}

	if !p.IsAvailable(ctx) {
		health.Status = ProviderStatusInactive
	}

	// TODO: Implement actual health check via API in Phase 5
	return health
}

// Priority returns the priority
func (p *PopbillTaxInvoiceProvider) Priority() int {
	return p.priority
}

// Close closes the provider
func (p *PopbillTaxInvoiceProvider) Close() error {
	return nil
}

// Issue issues a tax invoice via Popbill API
func (p *PopbillTaxInvoiceProvider) Issue(ctx context.Context, companyID uuid.UUID, invoice *TaxInvoiceData) (*TaxInvoiceIssueResult, error) {
	// TODO: Implement actual API call in Phase 5
	// This is a stub implementation
	return nil, fmt.Errorf("popbill provider not implemented - will be completed in Phase 5")
}

// Cancel cancels a tax invoice via Popbill API
func (p *PopbillTaxInvoiceProvider) Cancel(ctx context.Context, companyID uuid.UUID, ntsConfirmNumber string, reason string) (*TaxInvoiceIssueResult, error) {
	// TODO: Implement actual API call in Phase 5
	return nil, fmt.Errorf("popbill provider not implemented - will be completed in Phase 5")
}

// Search searches for tax invoices via Popbill API
func (p *PopbillTaxInvoiceProvider) Search(ctx context.Context, companyID uuid.UUID, filter *TaxInvoiceSearchFilter) (*TaxInvoiceSearchResult, error) {
	// TODO: Implement actual API call in Phase 5
	return nil, fmt.Errorf("popbill provider not implemented - will be completed in Phase 5")
}

// GetByConfirmNumber retrieves a tax invoice by NTS confirm number
func (p *PopbillTaxInvoiceProvider) GetByConfirmNumber(ctx context.Context, companyID uuid.UUID, confirmNumber string) (*TaxInvoiceData, error) {
	// TODO: Implement actual API call in Phase 5
	return nil, fmt.Errorf("popbill provider not implemented - will be completed in Phase 5")
}

// GetQuota returns the current quota status
func (p *PopbillTaxInvoiceProvider) GetQuota(ctx context.Context, companyID uuid.UUID) (*ProviderQuota, error) {
	// TODO: Implement actual API call in Phase 5
	return nil, fmt.Errorf("popbill provider not implemented - will be completed in Phase 5")
}
