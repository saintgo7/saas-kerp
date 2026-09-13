// Package grpcclient provides gRPC client for tax scraper service.
package grpcclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	taxv1 "github.com/saintgo7/saas-kerp/api/proto/tax/v1"
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
	SessionID string

	// CompanyID identifies the tenant that owns the session. REQUIRED.
	// The tax scraper cannot otherwise tell whether the caller owns the
	// session it named, and rejects a request without it with
	// UNAUTHENTICATED. See api/proto/tax/v1/tax.proto.
	CompanyID string

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
	SessionID string

	// CompanyID identifies the tenant that owns the session. REQUIRED - see
	// GetTaxInvoicesRequest.CompanyID.
	CompanyID string

	Invoice             TaxInvoice
	TransmitImmediately bool

	// IdempotencyKey must be stable across retries of the same logical
	// issuance. The tax scraper keys on it so that a retry cannot produce a
	// second 세금계산서 at the NTS.
	IdempotencyKey string

	// Provider selects the issuance route: "" or "hometax" for Hometax
	// scraping, "popbill" for the Popbill ASP API. The empty value is the
	// proto zero value, which the scraper reads as Hometax.
	Provider string
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

// errMissingCompanyID is returned before a call leaves the process when the
// tenant is not set. The tax scraper would reject it with UNAUTHENTICATED
// anyway (a deliberate fail-closed on the Python side); failing here names the
// actual problem instead of surfacing a transport error.
var errMissingCompanyID = errors.New("company_id is required for tax scraper calls: without it the service cannot verify the caller owns the session")

// errMissingSessionID is returned when no Hometax session was supplied.
var errMissingSessionID = errors.New("session_id is required for tax scraper calls")

// errMissingIdempotencyKey is returned when an issuance carries no idempotency
// key. Without one the scraper cannot recognise a retry, so a timeout followed
// by a retry files the 세금계산서 twice at the NTS - and the second filing can
// only be undone with a 수정세금계산서.
var errMissingIdempotencyKey = errors.New("idempotency_key is required for tax invoice issuance: without it a retry can file the invoice twice")

// dateLayouts are the formats the tax scraper emits for a date or timestamp.
//
// The proto comments say YYYY-MM-DD, but the Python service serialises with
// datetime.isoformat(), which produces a full timestamp with no zone offset
// and a variable-length fractional second. All the shapes it can produce are
// listed here; time.Parse matches the fraction against the "9" form.
var dateLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"20060102",
}

// client returns the generated stub bound to the tax scraper connection.
func (c *TaxInvoiceClient) client(ctx context.Context) (taxv1.TaxInvoiceServiceClient, error) {
	if c == nil || c.manager == nil {
		return nil, errors.New("tax scraper client is not configured")
	}

	conn, err := c.manager.TaxScraperConn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	return taxv1.NewTaxInvoiceServiceClient(conn), nil
}

// callContext bounds a single RPC.
//
// context.WithTimeout keeps whichever deadline expires first, so a caller that
// is already close to giving up is not extended by this.
func (c *TaxInvoiceClient) callContext(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := 30 * time.Second
	if c != nil && c.manager != nil && c.manager.config != nil && c.manager.config.CallTimeout > 0 {
		timeout = c.manager.config.CallTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// rpcError turns a gRPC status into an error that names the failure.
//
// The status code matters to the caller: the Python servicer maps its own
// error codes onto UNAUTHENTICATED, UNIMPLEMENTED, INVALID_ARGUMENT and
// UNAVAILABLE, and UNKNOWN specifically means the result of a 발급 attempt is
// undetermined - which must never be read as either success or a clean
// failure.
func rpcError(op string, err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("%s failed: %w", op, err)
	}

	switch st.Code() {
	case codes.Unauthenticated:
		return fmt.Errorf("%s rejected by tax scraper (%s): %s. The call must carry a valid GRPC_AUTH_TOKEN and a company_id that owns the session: %w",
			op, st.Code(), st.Message(), err)
	case codes.Unknown:
		return fmt.Errorf("%s returned an undetermined result from the tax scraper: %s. Do not retry without the same idempotency key: %w",
			op, st.Message(), err)
	default:
		return fmt.Errorf("%s failed (%s): %s: %w", op, st.Code(), st.Message(), err)
	}
}

// Login authenticates with Hometax.
func (c *TaxInvoiceClient) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	if req == nil {
		return nil, errors.New("login request is nil")
	}
	if req.CompanyID == "" {
		return nil, errMissingCompanyID
	}
	if req.BusinessNumber == "" {
		return nil, errors.New("business_number is required for Hometax login")
	}

	authType, err := authTypeToProto(req.AuthType)
	if err != nil {
		return nil, err
	}

	pbReq := &taxv1.LoginRequest{
		BusinessNumber: req.BusinessNumber,
		AuthType:       authType,
		CompanyId:      req.CompanyID,
	}
	// The optional fields are presence-tracked: sending an empty string is not
	// the same as not sending the field, and the servicer branches on
	// HasField.
	if req.CertPassword != "" {
		pbReq.CertPassword = &req.CertPassword
	}
	if len(req.CertData) > 0 {
		pbReq.CertData = req.CertData
	}
	if req.UserID != "" {
		pbReq.UserId = &req.UserID
	}
	if req.Password != "" {
		pbReq.Password = &req.Password
	}

	stub, err := c.client(ctx)
	if err != nil {
		return nil, err
	}

	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	resp, err := stub.Login(callCtx, pbReq)
	if err != nil {
		return nil, rpcError("Hometax login", err)
	}
	if resp == nil {
		return nil, errors.New("Hometax login returned an empty response")
	}

	// A success with no session id is not a success: every later call is
	// scoped by that id, so accepting it would hand the caller a session it
	// cannot use and hide the real failure.
	if resp.GetSuccess() && resp.GetSessionId() == "" {
		return nil, errors.New("Hometax login reported success without a session id")
	}

	expiresAt, err := parseScraperTime(resp.GetExpiresAt())
	if err != nil {
		return nil, fmt.Errorf("Hometax login returned an unreadable session expiry: %w", err)
	}

	return &LoginResponse{
		Success:      resp.GetSuccess(),
		SessionID:    resp.GetSessionId(),
		ExpiresAt:    expiresAt,
		CompanyName:  resp.GetCompanyName(),
		ErrorMessage: resp.GetErrorMessage(),
		ErrorCode:    resp.GetErrorCode(),
	}, nil
}

// Logout terminates Hometax session.
func (c *TaxInvoiceClient) Logout(ctx context.Context, companyID, sessionID string) error {
	if companyID == "" {
		return errMissingCompanyID
	}
	if sessionID == "" {
		return errMissingSessionID
	}

	stub, err := c.client(ctx)
	if err != nil {
		return err
	}

	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	resp, err := stub.Logout(callCtx, &taxv1.LogoutRequest{
		SessionId: sessionID,
		CompanyId: companyID,
	})
	if err != nil {
		return rpcError("Hometax logout", err)
	}
	if resp == nil {
		return errors.New("Hometax logout returned an empty response")
	}

	// Report a refused logout rather than swallowing it. A caller that
	// believes the session is closed will not try again, and the scraper keeps
	// a live Hometax session open on the caller's behalf.
	if !resp.GetSuccess() {
		msg := resp.GetErrorMessage()
		if msg == "" {
			msg = "no reason given"
		}
		return fmt.Errorf("Hometax logout was refused: %s", msg)
	}

	return nil
}

// GetTaxInvoices retrieves tax invoices from Hometax.
func (c *TaxInvoiceClient) GetTaxInvoices(ctx context.Context, req *GetTaxInvoicesRequest) (*GetTaxInvoicesResponse, error) {
	if req == nil {
		return nil, errors.New("get tax invoices request is nil")
	}
	if req.CompanyID == "" {
		return nil, errMissingCompanyID
	}
	if req.SessionID == "" {
		return nil, errMissingSessionID
	}

	pbReq := &taxv1.GetTaxInvoicesRequest{
		SessionId: req.SessionID,
		CompanyId: req.CompanyID,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}

	if req.InvoiceType != "" {
		invoiceType, err := invoiceTypeToProto(req.InvoiceType)
		if err != nil {
			return nil, err
		}
		pbReq.InvoiceType = &invoiceType
	}
	if req.BusinessNumber != "" {
		pbReq.BusinessNumber = &req.BusinessNumber
	}

	stub, err := c.client(ctx)
	if err != nil {
		return nil, err
	}

	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	resp, err := stub.GetTaxInvoices(callCtx, pbReq)
	if err != nil {
		return nil, rpcError("Hometax invoice retrieval", err)
	}
	if resp == nil {
		return nil, errors.New("Hometax invoice retrieval returned an empty response")
	}

	invoices := make([]TaxInvoice, 0, len(resp.GetInvoices()))
	for _, pbInvoice := range resp.GetInvoices() {
		invoice, err := invoiceFromProto(pbInvoice)
		if err != nil {
			return nil, fmt.Errorf("Hometax invoice retrieval returned an unusable invoice: %w", err)
		}
		invoices = append(invoices, invoice)
	}

	return &GetTaxInvoicesResponse{
		Success:      resp.GetSuccess(),
		Invoices:     invoices,
		TotalCount:   resp.GetTotalCount(),
		Page:         resp.GetPage(),
		PageSize:     resp.GetPageSize(),
		ErrorMessage: resp.GetErrorMessage(),
	}, nil
}

// IssueTaxInvoice issues a new tax invoice via Hometax.
func (c *TaxInvoiceClient) IssueTaxInvoice(ctx context.Context, req *IssueTaxInvoiceRequest) (*IssueTaxInvoiceResponse, error) {
	if req == nil {
		return nil, errors.New("issue tax invoice request is nil")
	}
	if req.CompanyID == "" {
		return nil, errMissingCompanyID
	}
	if req.SessionID == "" {
		return nil, errMissingSessionID
	}
	if req.IdempotencyKey == "" {
		return nil, errMissingIdempotencyKey
	}

	provider, err := providerToProto(req.Provider)
	if err != nil {
		return nil, err
	}

	pbInvoice, err := invoiceToProto(req.Invoice)
	if err != nil {
		return nil, err
	}

	stub, err := c.client(ctx)
	if err != nil {
		return nil, err
	}

	callCtx, cancel := c.callContext(ctx)
	defer cancel()

	resp, err := stub.IssueTaxInvoice(callCtx, &taxv1.IssueTaxInvoiceRequest{
		SessionId:           req.SessionID,
		CompanyId:           req.CompanyID,
		IdempotencyKey:      req.IdempotencyKey,
		Invoice:             pbInvoice,
		TransmitImmediately: req.TransmitImmediately,
		Provider:            provider,
	})
	if err != nil {
		return nil, rpcError("tax invoice issuance", err)
	}
	if resp == nil {
		return nil, errors.New("tax invoice issuance returned an empty response")
	}

	// A transmitted invoice with no 승인번호 is not a transmitted invoice. The
	// caller records that number and uses its presence as the guard against
	// filing the same invoice twice, so accepting an empty one would defeat
	// the guard on the next retry. internal/service also refuses this; the
	// check is repeated here so any future caller of this package inherits it.
	if resp.GetSuccess() && req.TransmitImmediately && resp.GetNtsConfirmNumber() == "" {
		return nil, errors.New("tax scraper reported a successful transmission without an NTS confirmation number")
	}

	issueDate, err := parseScraperTime(resp.GetIssueDate())
	if err != nil {
		return nil, fmt.Errorf("tax invoice issuance returned an unreadable issue date: %w", err)
	}

	return &IssueTaxInvoiceResponse{
		Success:          resp.GetSuccess(),
		InvoiceNumber:    resp.GetInvoiceNumber(),
		IssueDate:        issueDate,
		NTSConfirmNumber: resp.GetNtsConfirmNumber(),
		ErrorMessage:     resp.GetErrorMessage(),
		ErrorCode:        resp.GetErrorCode(),
	}, nil
}

// HealthCheck checks if tax scraper service is healthy.
func (c *TaxInvoiceClient) HealthCheck(ctx context.Context) error {
	if c == nil || c.manager == nil {
		return errors.New("tax scraper client is not configured")
	}
	return c.manager.HealthCheck(ctx, c.manager.config.TaxScraperAddr)
}

// authTypeToProto maps the caller's auth type string onto the proto enum.
//
// An unrecognised value is an error rather than a silent fall back to
// certificate authentication: quietly changing how a caller authenticates is
// how a credential ends up sent down a path it was not meant for.
func authTypeToProto(authType string) (taxv1.AuthType, error) {
	switch strings.ToLower(strings.TrimSpace(authType)) {
	case "", "certificate":
		return taxv1.AuthType_AUTH_TYPE_CERTIFICATE, nil
	case "simple_auth":
		return taxv1.AuthType_AUTH_TYPE_SIMPLE_AUTH, nil
	case "id_password":
		return taxv1.AuthType_AUTH_TYPE_ID_PASSWORD, nil
	default:
		return taxv1.AuthType_AUTH_TYPE_UNSPECIFIED, fmt.Errorf("unknown Hometax auth type %q (want certificate, simple_auth or id_password)", authType)
	}
}

// invoiceTypeToProto maps "sales"/"purchase" onto the proto enum.
func invoiceTypeToProto(invoiceType string) (taxv1.InvoiceType, error) {
	switch strings.ToLower(strings.TrimSpace(invoiceType)) {
	case "":
		return taxv1.InvoiceType_INVOICE_TYPE_UNSPECIFIED, nil
	case "sales":
		return taxv1.InvoiceType_INVOICE_TYPE_SALES, nil
	case "purchase":
		return taxv1.InvoiceType_INVOICE_TYPE_PURCHASE, nil
	default:
		return taxv1.InvoiceType_INVOICE_TYPE_UNSPECIFIED, fmt.Errorf("unknown invoice type %q (want sales or purchase)", invoiceType)
	}
}

// invoiceTypeFromProto maps the proto enum back onto the domain's strings.
func invoiceTypeFromProto(invoiceType taxv1.InvoiceType) string {
	switch invoiceType {
	case taxv1.InvoiceType_INVOICE_TYPE_SALES:
		return "sales"
	case taxv1.InvoiceType_INVOICE_TYPE_PURCHASE:
		return "purchase"
	default:
		return ""
	}
}

// statusFromProto maps the proto status enum onto the domain's strings.
func statusFromProto(s taxv1.InvoiceStatus) string {
	switch s {
	case taxv1.InvoiceStatus_INVOICE_STATUS_DRAFT:
		return "draft"
	case taxv1.InvoiceStatus_INVOICE_STATUS_ISSUED:
		return "issued"
	case taxv1.InvoiceStatus_INVOICE_STATUS_TRANSMITTED:
		return "transmitted"
	case taxv1.InvoiceStatus_INVOICE_STATUS_CONFIRMED:
		return "confirmed"
	case taxv1.InvoiceStatus_INVOICE_STATUS_CANCELLED:
		return "cancelled"
	case taxv1.InvoiceStatus_INVOICE_STATUS_REJECTED:
		return "rejected"
	default:
		return ""
	}
}

// providerToProto maps the caller's provider string onto the proto enum.
func providerToProto(provider string) (taxv1.ProviderType, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "":
		// The zero value. The scraper reads it as Hometax.
		return taxv1.ProviderType_PROVIDER_TYPE_UNSPECIFIED, nil
	case "hometax":
		return taxv1.ProviderType_PROVIDER_TYPE_HOMETAX, nil
	case "popbill":
		return taxv1.ProviderType_PROVIDER_TYPE_POPBILL, nil
	default:
		return taxv1.ProviderType_PROVIDER_TYPE_UNSPECIFIED, fmt.Errorf("unknown issuance provider %q (want hometax or popbill)", provider)
	}
}

// invoiceToProto converts an outbound invoice.
func invoiceToProto(invoice TaxInvoice) (*taxv1.TaxInvoice, error) {
	invoiceType, err := invoiceTypeToProto(invoice.InvoiceType)
	if err != nil {
		return nil, err
	}

	items := make([]*taxv1.TaxInvoiceItem, 0, len(invoice.Items))
	for _, item := range invoice.Items {
		items = append(items, &taxv1.TaxInvoiceItem{
			Sequence:      item.Sequence,
			SupplyDate:    formatScraperDate(item.SupplyDate),
			Description:   item.Description,
			Specification: item.Specification,
			Quantity:      item.Quantity,
			UnitPrice:     item.UnitPrice,
			Amount:        item.Amount,
			TaxAmount:     item.TaxAmount,
			Remarks:       item.Remarks,
		})
	}

	return &taxv1.TaxInvoice{
		InvoiceNumber:          invoice.InvoiceNumber,
		IssueDate:              formatScraperDate(invoice.IssueDate),
		InvoiceType:            invoiceType,
		SupplierBusinessNumber: invoice.SupplierBusinessNumber,
		SupplierName:           invoice.SupplierName,
		SupplierCeoName:        invoice.SupplierCEOName,
		SupplierAddress:        invoice.SupplierAddress,
		BuyerBusinessNumber:    invoice.BuyerBusinessNumber,
		BuyerName:              invoice.BuyerName,
		BuyerCeoName:           invoice.BuyerCEOName,
		BuyerAddress:           invoice.BuyerAddress,
		SupplyAmount:           invoice.SupplyAmount,
		TaxAmount:              invoice.TaxAmount,
		TotalAmount:            invoice.TotalAmount,
		Items:                  items,
		NtsConfirmNumber:       invoice.NTSConfirmNumber,
		Remarks:                invoice.Remarks,
	}, nil
}

// invoiceFromProto converts an inbound invoice.
//
// An unreadable date is an error rather than a zero time. A 세금계산서 whose
// 작성일자 silently became 0001-01-01 lands in the wrong VAT period, and
// nothing downstream can tell that from a date that was genuinely absent.
func invoiceFromProto(pb *taxv1.TaxInvoice) (TaxInvoice, error) {
	if pb == nil {
		return TaxInvoice{}, errors.New("tax scraper returned a nil invoice")
	}

	issueDate, err := parseScraperTime(pb.GetIssueDate())
	if err != nil {
		return TaxInvoice{}, fmt.Errorf("invoice %q has an unreadable issue date: %w", pb.GetInvoiceNumber(), err)
	}

	items := make([]TaxInvoiceItem, 0, len(pb.GetItems()))
	for _, pbItem := range pb.GetItems() {
		supplyDate, err := parseScraperTime(pbItem.GetSupplyDate())
		if err != nil {
			return TaxInvoice{}, fmt.Errorf("invoice %q item %d has an unreadable supply date: %w",
				pb.GetInvoiceNumber(), pbItem.GetSequence(), err)
		}
		items = append(items, TaxInvoiceItem{
			Sequence:      pbItem.GetSequence(),
			SupplyDate:    supplyDate,
			Description:   pbItem.GetDescription(),
			Specification: pbItem.GetSpecification(),
			Quantity:      pbItem.GetQuantity(),
			UnitPrice:     pbItem.GetUnitPrice(),
			Amount:        pbItem.GetAmount(),
			TaxAmount:     pbItem.GetTaxAmount(),
			Remarks:       pbItem.GetRemarks(),
		})
	}

	return TaxInvoice{
		InvoiceNumber:          pb.GetInvoiceNumber(),
		IssueDate:              issueDate,
		InvoiceType:            invoiceTypeFromProto(pb.GetInvoiceType()),
		Status:                 statusFromProto(pb.GetStatus()),
		SupplierBusinessNumber: pb.GetSupplierBusinessNumber(),
		SupplierName:           pb.GetSupplierName(),
		SupplierCEOName:        pb.GetSupplierCeoName(),
		SupplierAddress:        pb.GetSupplierAddress(),
		BuyerBusinessNumber:    pb.GetBuyerBusinessNumber(),
		BuyerName:              pb.GetBuyerName(),
		BuyerCEOName:           pb.GetBuyerCeoName(),
		BuyerAddress:           pb.GetBuyerAddress(),
		SupplyAmount:           pb.GetSupplyAmount(),
		TaxAmount:              pb.GetTaxAmount(),
		TotalAmount:            pb.GetTotalAmount(),
		Items:                  items,
		NTSConfirmNumber:       pb.GetNtsConfirmNumber(),
		Remarks:                pb.GetRemarks(),
	}, nil
}

// formatScraperDate renders a date for the wire. A zero time becomes the empty
// string, which is how the proto spells "absent".
func formatScraperDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// parseScraperTime parses a date or timestamp emitted by the tax scraper.
//
// An empty string yields the zero time and no error: the field is genuinely
// optional. A non-empty value that matches no known layout is an error, so a
// format change on the Python side surfaces instead of quietly zeroing a date.
func parseScraperTime(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, nil
	}

	for _, layout := range dateLayouts {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse %q as a date or timestamp", value)
}
