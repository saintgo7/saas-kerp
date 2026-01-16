// Package provider implements the provider chain pattern for external service integrations.
// It supports multiple providers (Popbill, Hometax scraper) with fallback mechanisms.
package provider

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Common errors
var (
	ErrProviderNotFound     = errors.New("provider not found")
	ErrProviderUnavailable  = errors.New("provider unavailable")
	ErrProviderTimeout      = errors.New("provider timeout")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrQuotaExceeded        = errors.New("quota exceeded")
	ErrAllProvidersFailed   = errors.New("all providers failed")
)

// ProviderType represents the type of provider
type ProviderType string

const (
	ProviderTypePopbill  ProviderType = "popbill"
	ProviderTypeHometax  ProviderType = "hometax"
	ProviderTypeMock     ProviderType = "mock"
)

// ProviderStatus represents the status of a provider
type ProviderStatus string

const (
	ProviderStatusActive   ProviderStatus = "active"
	ProviderStatusInactive ProviderStatus = "inactive"
	ProviderStatusError    ProviderStatus = "error"
)

// ProviderConfig holds configuration for a provider
type ProviderConfig struct {
	Type       ProviderType
	Priority   int           // Lower is higher priority
	Enabled    bool
	Timeout    time.Duration
	RetryCount int
	Config     map[string]interface{}
}

// ProviderHealth represents the health status of a provider
type ProviderHealth struct {
	Type        ProviderType
	Status      ProviderStatus
	LastChecked time.Time
	LastError   error
	Latency     time.Duration
	Quota       *ProviderQuota
}

// ProviderQuota represents usage quota for a provider
type ProviderQuota struct {
	MonthlyLimit int
	MonthlyUsed  int
	DailyLimit   int
	DailyUsed    int
	ResetAt      time.Time
}

// ProviderResult represents the result of a provider operation
type ProviderResult struct {
	Success    bool
	Provider   ProviderType
	Data       interface{}
	Error      error
	Duration   time.Duration
	RetryCount int
}

// Provider is the base interface for all providers
type Provider interface {
	// Type returns the provider type
	Type() ProviderType

	// Name returns a human-readable name
	Name() string

	// IsAvailable checks if the provider is available
	IsAvailable(ctx context.Context) bool

	// Health returns the health status
	Health(ctx context.Context) *ProviderHealth

	// Priority returns the priority (lower is higher)
	Priority() int

	// Close closes the provider connection
	Close() error
}

// ProviderChain manages multiple providers with fallback support
type ProviderChain struct {
	providers []Provider
	config    *ChainConfig
}

// ChainConfig holds configuration for the provider chain
type ChainConfig struct {
	EnableFallback bool
	MaxRetries     int
	RetryDelay     time.Duration
	Timeout        time.Duration
}

// NewProviderChain creates a new provider chain
func NewProviderChain(config *ChainConfig) *ProviderChain {
	if config == nil {
		config = &ChainConfig{
			EnableFallback: true,
			MaxRetries:     3,
			RetryDelay:     time.Second,
			Timeout:        30 * time.Second,
		}
	}
	return &ProviderChain{
		providers: make([]Provider, 0),
		config:    config,
	}
}

// Register adds a provider to the chain
func (c *ProviderChain) Register(provider Provider) {
	c.providers = append(c.providers, provider)
	// Sort by priority
	c.sortByPriority()
}

// sortByPriority sorts providers by their priority (lower first)
func (c *ProviderChain) sortByPriority() {
	for i := 0; i < len(c.providers)-1; i++ {
		for j := i + 1; j < len(c.providers); j++ {
			if c.providers[j].Priority() < c.providers[i].Priority() {
				c.providers[i], c.providers[j] = c.providers[j], c.providers[i]
			}
		}
	}
}

// GetProvider returns a specific provider by type
func (c *ProviderChain) GetProvider(providerType ProviderType) (Provider, error) {
	for _, p := range c.providers {
		if p.Type() == providerType {
			return p, nil
		}
	}
	return nil, ErrProviderNotFound
}

// GetAvailableProviders returns all available providers
func (c *ProviderChain) GetAvailableProviders(ctx context.Context) []Provider {
	available := make([]Provider, 0)
	for _, p := range c.providers {
		if p.IsAvailable(ctx) {
			available = append(available, p)
		}
	}
	return available
}

// GetPrimaryProvider returns the highest priority available provider
func (c *ProviderChain) GetPrimaryProvider(ctx context.Context) (Provider, error) {
	available := c.GetAvailableProviders(ctx)
	if len(available) == 0 {
		return nil, ErrProviderUnavailable
	}
	return available[0], nil
}

// HealthCheck performs health checks on all providers
func (c *ProviderChain) HealthCheck(ctx context.Context) map[ProviderType]*ProviderHealth {
	results := make(map[ProviderType]*ProviderHealth)
	for _, p := range c.providers {
		results[p.Type()] = p.Health(ctx)
	}
	return results
}

// Close closes all providers
func (c *ProviderChain) Close() error {
	var lastErr error
	for _, p := range c.providers {
		if err := p.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// CompanyProviderConfig stores provider configuration per company
type CompanyProviderConfig struct {
	ID               uuid.UUID
	CompanyID        uuid.UUID
	ProviderType     ProviderType
	Priority         int
	Enabled          bool
	CredentialsJSON  []byte // Encrypted
	QuotaMonthly     int
	QuotaUsed        int
	QuotaResetAt     time.Time
	LastSuccessAt    *time.Time
	LastErrorAt      *time.Time
	LastErrorMessage string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ProviderRegistry manages company-specific provider configurations
type ProviderRegistry struct {
	configs map[uuid.UUID][]CompanyProviderConfig
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		configs: make(map[uuid.UUID][]CompanyProviderConfig),
	}
}

// GetCompanyConfig returns provider configs for a company
func (r *ProviderRegistry) GetCompanyConfig(companyID uuid.UUID) []CompanyProviderConfig {
	return r.configs[companyID]
}

// SetCompanyConfig sets provider configs for a company
func (r *ProviderRegistry) SetCompanyConfig(companyID uuid.UUID, configs []CompanyProviderConfig) {
	r.configs[companyID] = configs
}
