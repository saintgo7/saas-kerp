// Package grpcclient provides gRPC client connections to microservices.
package grpcclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"

	// Registers the client-side health checking producer that the
	// healthCheckConfig in the service config below asks for. Without this
	// blank import the setting is inert and a dead backend keeps receiving
	// requests.
	_ "google.golang.org/grpc/health"
)

// Default service addresses, matching deployments/docker/.env.example.
const (
	defaultTaxScraperAddr   = "localhost:50051"
	defaultInsuranceEDIAddr = "localhost:50052"
)

// ClientConfig holds configuration for gRPC client connections.
type ClientConfig struct {
	// Service addresses
	TaxScraperAddr   string
	InsuranceEDIAddr string

	// Connection settings
	DialTimeout      time.Duration
	KeepAliveTime    time.Duration
	KeepAliveTimeout time.Duration
	MaxRetryAttempts int

	// CallTimeout bounds a single RPC. It is applied on top of whatever
	// deadline the caller's context already carries, so the earlier of the two
	// wins: a request that is already about to be abandoned by the HTTP client
	// does not keep a scraper session open.
	CallTimeout time.Duration

	// Security decides how the connection is encrypted and how the caller
	// authenticates. A nil value is read from the environment, which fails
	// closed outside development.
	Security *ClientSecurity
}

// DefaultConfig returns default client configuration.
//
// Security is read from the environment, so a production process that has set
// neither GRPC_AUTH_TOKEN nor TLS material is refused at dial time rather than
// silently talking plaintext to the tax scraper.
func DefaultConfig() *ClientConfig {
	return &ClientConfig{
		TaxScraperAddr:   defaultTaxScraperAddr,
		InsuranceEDIAddr: defaultInsuranceEDIAddr,
		DialTimeout:      5 * time.Second,
		KeepAliveTime:    30 * time.Second,
		KeepAliveTimeout: 10 * time.Second,
		MaxRetryAttempts: 3,
		CallTimeout:      30 * time.Second,
		Security:         SecurityFromEnv(""),
	}
}

// ConfigFromEnv returns a configuration built from the process environment.
//
// environment is the application's own environment name (cfg.App.Env); it
// decides whether an unauthenticated plaintext connection is refused.
//
// Addresses come from TAX_SCRAPER_ADDR / INSURANCE_EDI_ADDR, or from the
// HOST + PORT pairs that deployments/docker/.env.example defines.
func ConfigFromEnv(environment string) *ClientConfig {
	cfg := DefaultConfig()
	cfg.Security = SecurityFromEnv(environment)

	if addr := serviceAddrFromEnv("TAX_SCRAPER_ADDR", "TAX_SCRAPER_HOST", "TAX_SCRAPER_PORT"); addr != "" {
		cfg.TaxScraperAddr = addr
	}
	if addr := serviceAddrFromEnv("INSURANCE_EDI_ADDR", "INSURANCE_EDI_HOST", "INSURANCE_EDI_PORT"); addr != "" {
		cfg.InsuranceEDIAddr = addr
	}

	return cfg
}

// serviceAddrFromEnv resolves a host:port address from either a single
// address variable or a host/port pair.
func serviceAddrFromEnv(addrVar, hostVar, portVar string) string {
	if addr := strings.TrimSpace(os.Getenv(addrVar)); addr != "" {
		return addr
	}

	host := strings.TrimSpace(os.Getenv(hostVar))
	port := strings.TrimSpace(os.Getenv(portVar))
	if host == "" || port == "" {
		return ""
	}
	return net.JoinHostPort(host, port)
}

// Manager manages gRPC client connections.
type Manager struct {
	config *ClientConfig
	mu     sync.RWMutex
	conns  map[string]*grpc.ClientConn
}

// NewManager creates a new gRPC client manager.
func NewManager(config *ClientConfig) *Manager {
	if config == nil {
		config = DefaultConfig()
	}
	if config.Security == nil {
		config.Security = SecurityFromEnv("")
	}
	if config.CallTimeout <= 0 {
		config.CallTimeout = 30 * time.Second
	}
	return &Manager{
		config: config,
		conns:  make(map[string]*grpc.ClientConn),
	}
}

// Config returns the manager's configuration.
func (m *Manager) Config() *ClientConfig {
	return m.config
}

// serviceConfigJSON builds the channel's default service config.
//
// Only the read-only methods carry a retry policy. IssueTaxInvoice,
// CancelTaxInvoice and Login are deliberately excluded: the transport cannot
// tell a request that never arrived from a response that was lost on the way
// back, and a silent second attempt at 세금계산서 발급 is a duplicate filing
// that can only be undone with a 수정세금계산서. Retrying those is the
// caller's decision, made with the same idempotency key.
func (m *Manager) serviceConfigJSON() string {
	attempts := m.config.MaxRetryAttempts
	if attempts < 2 {
		// A retry policy needs at least two attempts to mean anything.
		return `{"loadBalancingPolicy":"round_robin","healthCheckConfig":{"serviceName":""}}`
	}
	if attempts > 5 {
		attempts = 5
	}

	cfg := map[string]any{
		"loadBalancingPolicy": "round_robin",
		"healthCheckConfig":   map[string]any{"serviceName": ""},
		"methodConfig": []any{
			map[string]any{
				"name": []any{
					map[string]any{"service": "kerp.tax.v1.TaxInvoiceService", "method": "GetTaxInvoices"},
					map[string]any{"service": "kerp.tax.v1.TaxInvoiceService", "method": "GetTaxInvoiceStatus"},
					map[string]any{"service": "kerp.tax.v1.TaxInvoiceService", "method": "HealthCheck"},
					map[string]any{"service": "grpc.health.v1.Health", "method": "Check"},
				},
				"retryPolicy": map[string]any{
					"maxAttempts":          attempts,
					"initialBackoff":       "0.2s",
					"maxBackoff":           "2s",
					"backoffMultiplier":    2.0,
					"retryableStatusCodes": []any{"UNAVAILABLE"},
				},
			},
		},
	}

	encoded, err := json.Marshal(cfg)
	if err != nil {
		// The literal above is a constant shape; this cannot fail. Fall back
		// to the no-retry config rather than panic in a request path.
		return `{"loadBalancingPolicy":"round_robin","healthCheckConfig":{"serviceName":""}}`
	}
	return string(encoded)
}

// dial creates a new gRPC connection with standard options.
func (m *Manager) dial(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	if strings.TrimSpace(addr) == "" {
		return nil, fmt.Errorf("grpcclient: no address configured for the service")
	}

	secOpts, err := m.config.Security.dialOptions()
	if err != nil {
		return nil, err
	}

	opts := append(secOpts,
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                m.config.KeepAliveTime,
			Timeout:             m.config.KeepAliveTimeout,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultServiceConfig(m.serviceConfigJSON()),
		// The Python servers cap both directions at 8MB; asking for more only
		// turns a rejected message into a confusing error.
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(8*1024*1024),
			grpc.MaxCallRecvMsgSize(8*1024*1024),
		),
	)

	ctx, cancel := context.WithTimeout(ctx, m.config.DialTimeout)
	defer cancel()

	// DialContext is deprecated since grpc 1.63 but supported throughout 1.x.
	// NewClient switches the default resolver from passthrough to dns, so the
	// migration is a behaviour change and is left for its own commit.
	conn, err := grpc.DialContext(ctx, addr, opts...) //nolint:staticcheck // SA1019, see above
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", addr, err)
	}

	return conn, nil
}

// GetConnection returns a cached or new connection for the given address.
func (m *Manager) GetConnection(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	m.mu.RLock()
	conn, exists := m.conns[addr]
	m.mu.RUnlock()

	if exists {
		return conn, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if conn, exists := m.conns[addr]; exists {
		return conn, nil
	}

	conn, err := m.dial(ctx, addr)
	if err != nil {
		return nil, err
	}

	m.conns[addr] = conn
	return conn, nil
}

// TaxScraperConn returns connection to tax scraper service.
func (m *Manager) TaxScraperConn(ctx context.Context) (*grpc.ClientConn, error) {
	return m.GetConnection(ctx, m.config.TaxScraperAddr)
}

// InsuranceEDIConn returns connection to insurance EDI service.
func (m *Manager) InsuranceEDIConn(ctx context.Context) (*grpc.ClientConn, error) {
	return m.GetConnection(ctx, m.config.InsuranceEDIAddr)
}

// HealthCheck performs health check on a service.
//
// It uses the standard grpc.health.v1 service, which is the one the Python
// TokenAuthInterceptor exempts from authentication, so it answers even when
// the caller has no token configured.
func (m *Manager) HealthCheck(ctx context.Context, addr string) error {
	conn, err := m.GetConnection(ctx, addr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, m.config.CallTimeout)
	defer cancel()

	client := grpc_health_v1.NewHealthClient(conn)
	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("service not serving: %s", resp.Status)
	}

	return nil
}

// Close closes all connections.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for addr, conn := range m.conns {
		if err := conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close %s: %w", addr, err))
		}
		delete(m.conns, addr)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}
	return nil
}
