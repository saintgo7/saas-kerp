// Package grpcclient provides gRPC client for tax scraper service.
package grpcclient

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
)

// TaxInvoiceClient provides methods to interact with tax scraper gRPC service.
type TaxInvoiceClient struct {
	manager *Manager
}

// NewTaxInvoiceClient creates a new tax invoice client.
func NewTaxInvoiceClient(manager *Manager) *TaxInvoiceClient {
	return &TaxInvoiceClient{manager: manager}
}

// LoginRequest represents a login request to Hometax.
type LoginRequest struct {
	BusinessNumber string
	AuthType       string // "certificate", "simple_auth", "id_password"
	CertPassword   string
	CertData       []byte
	UserID         string
	Password       string
	CompanyID      string
}

// LoginResponse represents a login response from Hometax.
type LoginResponse struct {
	Success      bool
	SessionID    string
	ExpiresAt    time.Time
	CompanyName  string
	ErrorMessage string
	ErrorCode    string
}

// TaxInvoice represents a tax invoice.
type TaxInvoice struct {
	InvoiceNumber          string
	IssueDate              time.Time
	InvoiceType            string // "sales", "purchase"
	Status                 string
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
	TotalAmount            int64
	Items                  []TaxInvoiceItem
	NTSConfirmNumber       string
	Remarks                string
}

// TaxInvoiceItem represents a line item in a tax invoice.
type TaxInvoiceItem struct {
	Sequence      int32
	SupplyDate    time.Time
	Description   string
	Specification string
	Quantity      int32
	UnitPrice     int64
	Amount        int64
	TaxAmount     int64
	Remarks       string
}

// GetTaxInvoicesRequest represents a request to get tax invoices.
type GetTaxInvoicesRequest struct {
	SessionID      string
	StartDate      string
	EndDate        string
	InvoiceType    string
	BusinessNumber string
	Page           int32
	PageSize       int32
}

// GetTaxInvoicesResponse represents a response with tax invoices.
type GetTaxInvoicesResponse struct {
	Success      bool
	Invoices     []TaxInvoice
	TotalCount   int32
	Page         int32
	PageSize     int32
	ErrorMessage string
}

// IssueTaxInvoiceRequest represents a request to issue a tax invoice.
type IssueTaxInvoiceRequest struct {
	SessionID           string
	Invoice             TaxInvoice
	TransmitImmediately bool
}

// IssueTaxInvoiceResponse represents a response from issuing a tax invoice.
type IssueTaxInvoiceResponse struct {
	Success          bool
	InvoiceNumber    string
	IssueDate        time.Time
	NTSConfirmNumber string
	ErrorMessage     string
	ErrorCode        string
}

// getConn returns the gRPC connection to tax scraper service.
func (c *TaxInvoiceClient) getConn(ctx context.Context) (*grpc.ClientConn, error) {
	return c.manager.TaxScraperConn(ctx)
}

// Login authenticates with Hometax.
func (c *TaxInvoiceClient) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// TODO: Use generated proto client
	// client := taxv1.NewTaxInvoiceServiceClient(conn)
	// resp, err := client.Login(ctx, &taxv1.LoginRequest{...})
	_ = conn

	// Placeholder implementation
	return &LoginResponse{
		Success:      false,
		ErrorMessage: "gRPC client not yet implemented - run buf generate",
	}, nil
}

// Logout terminates Hometax session.
func (c *TaxInvoiceClient) Logout(ctx context.Context, sessionID string) error {
	conn, err := c.getConn(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}

	// TODO: Use generated proto client
	_ = conn

	return nil
}

// GetTaxInvoices retrieves tax invoices from Hometax.
func (c *TaxInvoiceClient) GetTaxInvoices(ctx context.Context, req *GetTaxInvoicesRequest) (*GetTaxInvoicesResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// TODO: Use generated proto client
	_ = conn

	return &GetTaxInvoicesResponse{
		Success:      false,
		ErrorMessage: "gRPC client not yet implemented - run buf generate",
	}, nil
}

// IssueTaxInvoice issues a new tax invoice via Hometax.
func (c *TaxInvoiceClient) IssueTaxInvoice(ctx context.Context, req *IssueTaxInvoiceRequest) (*IssueTaxInvoiceResponse, error) {
	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// TODO: Use generated proto client
	_ = conn

	return &IssueTaxInvoiceResponse{
		Success:      false,
		ErrorMessage: "gRPC client not yet implemented - run buf generate",
	}, nil
}

// HealthCheck checks if tax scraper service is healthy.
func (c *TaxInvoiceClient) HealthCheck(ctx context.Context) error {
	return c.manager.HealthCheck(ctx, c.manager.config.TaxScraperAddr)
}
