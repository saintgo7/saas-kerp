// Package grpcclient provides gRPC client connections to microservices.
package grpcclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

// ClientConfig holds configuration for gRPC client connections.
type ClientConfig struct {
	// Service addresses
	TaxScraperAddr    string
	InsuranceEDIAddr  string

	// Connection settings
	DialTimeout       time.Duration
	KeepAliveTime     time.Duration
	KeepAliveTimeout  time.Duration
	MaxRetryAttempts  int
}

// DefaultConfig returns default client configuration.
func DefaultConfig() *ClientConfig {
	return &ClientConfig{
		TaxScraperAddr:   "localhost:50051",
		InsuranceEDIAddr: "localhost:50052",
		DialTimeout:      5 * time.Second,
		KeepAliveTime:    30 * time.Second,
		KeepAliveTimeout: 10 * time.Second,
		MaxRetryAttempts: 3,
	}
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
	return &Manager{
		config: config,
		conns:  make(map[string]*grpc.ClientConn),
	}
}

// dial creates a new gRPC connection with standard options.
func (m *Manager) dial(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                m.config.KeepAliveTime,
			Timeout:             m.config.KeepAliveTimeout,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultServiceConfig(`{
			"loadBalancingPolicy": "round_robin",
			"healthCheckConfig": {
				"serviceName": ""
			}
		}`),
	}

	ctx, cancel := context.WithTimeout(ctx, m.config.DialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr, opts...)
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
func (m *Manager) HealthCheck(ctx context.Context, addr string) error {
	conn, err := m.GetConnection(ctx, addr)
	if err != nil {
		return err
	}

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
