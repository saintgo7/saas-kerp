package grpcclient

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthsvc "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	taxv1 "github.com/saintgo7/saas-kerp/api/proto/tax/v1"
)

// fakeTaxServer records what the client actually put on the wire and answers
// with whatever the test set up. It stands in for
// python-services/tax-scraper: these tests prove the Go client's half of the
// contract, not that the Python service honours it.
type fakeTaxServer struct {
	taxv1.UnimplementedTaxInvoiceServiceServer

	mu sync.Mutex

	lastIssue    *taxv1.IssueTaxInvoiceRequest
	lastGet      *taxv1.GetTaxInvoicesRequest
	lastLogout   *taxv1.LogoutRequest
	lastMetadata metadata.MD

	issueResp  *taxv1.IssueTaxInvoiceResponse
	issueErr   error
	getResp    *taxv1.GetTaxInvoicesResponse
	getErr     error
	loginResp  *taxv1.LoginResponse
	logoutResp *taxv1.LogoutResponse
}

func (s *fakeTaxServer) record(ctx context.Context) {
	md, _ := metadata.FromIncomingContext(ctx)
	s.lastMetadata = md.Copy()
}

func (s *fakeTaxServer) Login(ctx context.Context, req *taxv1.LoginRequest) (*taxv1.LoginResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.record(ctx)
	if s.loginResp == nil {
		return &taxv1.LoginResponse{Success: true, SessionId: "sess-1"}, nil
	}
	return s.loginResp, nil
}

func (s *fakeTaxServer) Logout(ctx context.Context, req *taxv1.LogoutRequest) (*taxv1.LogoutResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.record(ctx)
	s.lastLogout = req
	if s.logoutResp == nil {
		return &taxv1.LogoutResponse{Success: true}, nil
	}
	return s.logoutResp, nil
}

func (s *fakeTaxServer) GetTaxInvoices(ctx context.Context, req *taxv1.GetTaxInvoicesRequest) (*taxv1.GetTaxInvoicesResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.record(ctx)
	s.lastGet = req
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getResp == nil {
		return &taxv1.GetTaxInvoicesResponse{Success: true}, nil
	}
	return s.getResp, nil
}

func (s *fakeTaxServer) IssueTaxInvoice(ctx context.Context, req *taxv1.IssueTaxInvoiceRequest) (*taxv1.IssueTaxInvoiceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.record(ctx)
	s.lastIssue = req
	if s.issueErr != nil {
		return nil, s.issueErr
	}
	if s.issueResp == nil {
		return &taxv1.IssueTaxInvoiceResponse{
			Success:          true,
			InvoiceNumber:    "20260907-0001",
			NtsConfirmNumber: "202609071234567890",
		}, nil
	}
	return s.issueResp, nil
}

func (s *fakeTaxServer) snapshotIssue() *taxv1.IssueTaxInvoiceRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastIssue
}

func (s *fakeTaxServer) snapshotMetadata() metadata.MD {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastMetadata
}

// startFakeServer runs the fake on a loopback port and returns a client
// pointed at it.
func startFakeServer(t *testing.T, srv *fakeTaxServer, security *ClientSecurity) *TaxInvoiceClient {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	taxv1.RegisterTaxInvoiceServiceServer(grpcServer, srv)

	// The channel's healthCheckConfig makes grpc-go call Health/Watch. A
	// server that answers it keeps the test deterministic.
	health := healthsvc.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, health)
	health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	if security == nil {
		security = &ClientSecurity{Environment: "test", AllowInsecure: true}
	}

	manager := NewManager(&ClientConfig{
		TaxScraperAddr:   lis.Addr().String(),
		DialTimeout:      2 * time.Second,
		KeepAliveTime:    30 * time.Second,
		KeepAliveTimeout: 10 * time.Second,
		MaxRetryAttempts: 1,
		CallTimeout:      3 * time.Second,
		Security:         security,
	})

	t.Cleanup(func() {
		_ = manager.Close()
		grpcServer.Stop()
	})

	return NewTaxInvoiceClient(manager)
}

// unreachableClient points at a port nothing is listening on.
func unreachableClient(t *testing.T) *TaxInvoiceClient {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := lis.Addr().String()
	// Close it again: the address is now known to be free.
	if err := lis.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	manager := NewManager(&ClientConfig{
		TaxScraperAddr:   addr,
		DialTimeout:      500 * time.Millisecond,
		KeepAliveTime:    30 * time.Second,
		KeepAliveTimeout: 10 * time.Second,
		MaxRetryAttempts: 1,
		CallTimeout:      1500 * time.Millisecond,
		Security:         &ClientSecurity{Environment: "test", AllowInsecure: true},
	})
	t.Cleanup(func() { _ = manager.Close() })

	return NewTaxInvoiceClient(manager)
}

func validIssueRequest() *IssueTaxInvoiceRequest {
	return &IssueTaxInvoiceRequest{
		SessionID:           "sess-1",
		CompanyID:           "11111111-1111-1111-1111-111111111111",
		IdempotencyKey:      "22222222-2222-2222-2222-222222222222",
		TransmitImmediately: true,
		Invoice: TaxInvoice{
			InvoiceNumber:          "20260907-0001",
			IssueDate:              time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC),
			InvoiceType:            "sales",
			SupplierBusinessNumber: "1234567890",
			SupplierName:           "공급자",
			BuyerBusinessNumber:    "0987654321",
			BuyerName:              "공급받는자",
			SupplyAmount:           1000000,
			TaxAmount:              100000,
			TotalAmount:            1100000,
		},
	}
}

func TestIssueTaxInvoiceSendsCompanyIDAndIdempotencyKey(t *testing.T) {
	srv := &fakeTaxServer{}
	client := startFakeServer(t, srv, nil)

	req := validIssueRequest()
	resp, err := client.IssueTaxInvoice(context.Background(), req)
	if err != nil {
		t.Fatalf("IssueTaxInvoice: %v", err)
	}
	if !resp.Success || resp.NTSConfirmNumber == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	sent := srv.snapshotIssue()
	if sent == nil {
		t.Fatal("server received no request")
	}
	if sent.GetCompanyId() != req.CompanyID {
		t.Errorf("company_id = %q, want %q", sent.GetCompanyId(), req.CompanyID)
	}
	if sent.GetIdempotencyKey() != req.IdempotencyKey {
		t.Errorf("idempotency_key = %q, want %q", sent.GetIdempotencyKey(), req.IdempotencyKey)
	}
	if !sent.GetTransmitImmediately() {
		t.Error("transmit_immediately was not sent")
	}
	if got := sent.GetInvoice().GetIssueDate(); got != "2026-09-07" {
		t.Errorf("issue_date = %q, want 2026-09-07", got)
	}
	if sent.GetInvoice().GetInvoiceType() != taxv1.InvoiceType_INVOICE_TYPE_SALES {
		t.Errorf("invoice_type = %v, want SALES", sent.GetInvoice().GetInvoiceType())
	}
}

func TestIssueTaxInvoiceRequiresCompanyID(t *testing.T) {
	client := startFakeServer(t, &fakeTaxServer{}, nil)

	req := validIssueRequest()
	req.CompanyID = ""

	if _, err := client.IssueTaxInvoice(context.Background(), req); err == nil {
		t.Fatal("expected an error when company_id is absent")
	}
}

func TestIssueTaxInvoiceRequiresIdempotencyKey(t *testing.T) {
	client := startFakeServer(t, &fakeTaxServer{}, nil)

	req := validIssueRequest()
	req.IdempotencyKey = ""

	_, err := client.IssueTaxInvoice(context.Background(), req)
	if err == nil {
		t.Fatal("expected an error when idempotency_key is absent")
	}
	if !strings.Contains(err.Error(), "idempotency_key") {
		t.Errorf("error should name the missing key, got: %v", err)
	}
}

// A transmission the NTS supposedly accepted, with no 승인번호 to show for it,
// must not come back as a success: the caller uses that number as its guard
// against filing the same invoice twice.
func TestIssueTaxInvoiceRejectsSuccessWithoutConfirmNumber(t *testing.T) {
	srv := &fakeTaxServer{
		issueResp: &taxv1.IssueTaxInvoiceResponse{
			Success:          true,
			InvoiceNumber:    "20260907-0001",
			NtsConfirmNumber: "",
		},
	}
	client := startFakeServer(t, srv, nil)

	_, err := client.IssueTaxInvoice(context.Background(), validIssueRequest())
	if err == nil {
		t.Fatal("expected an error for a success with no NTS confirmation number")
	}
	if !strings.Contains(err.Error(), "confirmation number") {
		t.Errorf("error should name the missing confirmation number, got: %v", err)
	}
}

// An invoice issued but deliberately not transmitted has no confirmation
// number yet, and that is not an error.
func TestIssueTaxInvoiceAllowsNoConfirmNumberWhenNotTransmitting(t *testing.T) {
	srv := &fakeTaxServer{
		issueResp: &taxv1.IssueTaxInvoiceResponse{
			Success:       true,
			InvoiceNumber: "20260907-0001",
		},
	}
	client := startFakeServer(t, srv, nil)

	req := validIssueRequest()
	req.TransmitImmediately = false

	resp, err := client.IssueTaxInvoice(context.Background(), req)
	if err != nil {
		t.Fatalf("IssueTaxInvoice: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success")
	}
}

// The point of the whole exercise: a server that cannot be reached must not
// produce a response that reads as anything but a failure.
func TestIssueTaxInvoiceFailsClosedWhenServerUnreachable(t *testing.T) {
	client := unreachableClient(t)

	resp, err := client.IssueTaxInvoice(context.Background(), validIssueRequest())
	if err == nil {
		t.Fatalf("expected an error from an unreachable tax scraper, got response %+v", resp)
	}
	if resp != nil {
		t.Errorf("expected a nil response alongside the error, got %+v", resp)
	}
}

func TestIssueTaxInvoiceSurfacesUnauthenticated(t *testing.T) {
	srv := &fakeTaxServer{
		issueErr: status.Error(codes.Unauthenticated, "Missing or invalid authentication token"),
	}
	client := startFakeServer(t, srv, nil)

	_, err := client.IssueTaxInvoice(context.Background(), validIssueRequest())
	if err == nil {
		t.Fatal("expected an error")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("status code = %v, want Unauthenticated", status.Code(err))
	}
}

// UNKNOWN is how the Python servicer reports RESULT_UNDETERMINED: the 발급 may
// or may not have happened. It must never look like a clean failure the caller
// can retry blindly.
func TestIssueTaxInvoiceSurfacesUndeterminedResult(t *testing.T) {
	srv := &fakeTaxServer{
		issueErr: status.Error(codes.Unknown, "RESULT_UNDETERMINED"),
	}
	client := startFakeServer(t, srv, nil)

	_, err := client.IssueTaxInvoice(context.Background(), validIssueRequest())
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "undetermined") {
		t.Errorf("error should flag the undetermined result, got: %v", err)
	}
}

func TestIssueTaxInvoiceAttachesBearerToken(t *testing.T) {
	srv := &fakeTaxServer{}
	client := startFakeServer(t, srv, &ClientSecurity{
		Environment: "test",
		AuthToken:   "s3cr3t-token",
	})

	if _, err := client.IssueTaxInvoice(context.Background(), validIssueRequest()); err != nil {
		t.Fatalf("IssueTaxInvoice: %v", err)
	}

	md := srv.snapshotMetadata()
	got := md.Get(authMetadataKey)
	if len(got) != 1 {
		t.Fatalf("authorization metadata = %v, want exactly one value", got)
	}
	if got[0] != "Bearer s3cr3t-token" {
		t.Errorf("authorization = %q, want %q", got[0], "Bearer s3cr3t-token")
	}
}

func TestGetTaxInvoicesRequiresCompanyID(t *testing.T) {
	client := startFakeServer(t, &fakeTaxServer{}, nil)

	_, err := client.GetTaxInvoices(context.Background(), &GetTaxInvoicesRequest{
		SessionID: "sess-1",
	})
	if err == nil {
		t.Fatal("expected an error when company_id is absent")
	}
	if !strings.Contains(err.Error(), "company_id") {
		t.Errorf("error should name company_id, got: %v", err)
	}
}

func TestGetTaxInvoicesSendsFiltersAndConvertsInvoices(t *testing.T) {
	srv := &fakeTaxServer{
		getResp: &taxv1.GetTaxInvoicesResponse{
			Success:    true,
			TotalCount: 1,
			Page:       2,
			PageSize:   50,
			Invoices: []*taxv1.TaxInvoice{
				{
					InvoiceNumber: "20260907-0001",
					// datetime.isoformat() on the Python side, not YYYY-MM-DD.
					IssueDate:        "2026-09-07T13:45:12.123456",
					InvoiceType:      taxv1.InvoiceType_INVOICE_TYPE_PURCHASE,
					Status:           taxv1.InvoiceStatus_INVOICE_STATUS_CONFIRMED,
					SupplyAmount:     1000,
					TaxAmount:        100,
					TotalAmount:      1100,
					NtsConfirmNumber: "202609071234567890",
				},
			},
		},
	}
	client := startFakeServer(t, srv, nil)

	resp, err := client.GetTaxInvoices(context.Background(), &GetTaxInvoicesRequest{
		SessionID:      "sess-1",
		CompanyID:      "11111111-1111-1111-1111-111111111111",
		StartDate:      "2026-09-01",
		EndDate:        "2026-09-30",
		InvoiceType:    "purchase",
		BusinessNumber: "1234567890",
		Page:           2,
		PageSize:       50,
	})
	if err != nil {
		t.Fatalf("GetTaxInvoices: %v", err)
	}

	srv.mu.Lock()
	sent := srv.lastGet
	srv.mu.Unlock()

	if sent.GetCompanyId() != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("company_id = %q", sent.GetCompanyId())
	}
	if sent.InvoiceType == nil || *sent.InvoiceType != taxv1.InvoiceType_INVOICE_TYPE_PURCHASE {
		t.Errorf("invoice_type = %v, want PURCHASE", sent.InvoiceType)
	}
	if sent.BusinessNumber == nil || *sent.BusinessNumber != "1234567890" {
		t.Errorf("business_number = %v", sent.BusinessNumber)
	}

	if len(resp.Invoices) != 1 {
		t.Fatalf("got %d invoices, want 1", len(resp.Invoices))
	}
	inv := resp.Invoices[0]
	if inv.InvoiceType != "purchase" {
		t.Errorf("invoice type = %q, want purchase", inv.InvoiceType)
	}
	if inv.Status != "confirmed" {
		t.Errorf("status = %q, want confirmed", inv.Status)
	}
	want := time.Date(2026, 9, 7, 13, 45, 12, 123456000, time.UTC)
	if !inv.IssueDate.Equal(want) {
		t.Errorf("issue date = %v, want %v", inv.IssueDate, want)
	}
}

// A date the client cannot read must not become 0001-01-01: that lands the
// 세금계산서 in the wrong VAT period and nothing downstream can tell.
func TestGetTaxInvoicesRejectsUnreadableIssueDate(t *testing.T) {
	srv := &fakeTaxServer{
		getResp: &taxv1.GetTaxInvoicesResponse{
			Success: true,
			Invoices: []*taxv1.TaxInvoice{
				{InvoiceNumber: "20260907-0001", IssueDate: "7 September 2026"},
			},
		},
	}
	client := startFakeServer(t, srv, nil)

	_, err := client.GetTaxInvoices(context.Background(), &GetTaxInvoicesRequest{
		SessionID: "sess-1",
		CompanyID: "11111111-1111-1111-1111-111111111111",
	})
	if err == nil {
		t.Fatal("expected an error for an unparseable issue date")
	}
}

func TestLoginRejectsSuccessWithoutSessionID(t *testing.T) {
	srv := &fakeTaxServer{
		loginResp: &taxv1.LoginResponse{Success: true, SessionId: ""},
	}
	client := startFakeServer(t, srv, nil)

	_, err := client.Login(context.Background(), &LoginRequest{
		CompanyID:      "11111111-1111-1111-1111-111111111111",
		BusinessNumber: "1234567890",
		AuthType:       "certificate",
	})
	if err == nil {
		t.Fatal("expected an error for a login success with no session id")
	}
}

func TestLoginRejectsUnknownAuthType(t *testing.T) {
	client := startFakeServer(t, &fakeTaxServer{}, nil)

	_, err := client.Login(context.Background(), &LoginRequest{
		CompanyID:      "11111111-1111-1111-1111-111111111111",
		BusinessNumber: "1234567890",
		AuthType:       "carrier_pigeon",
	})
	if err == nil {
		t.Fatal("expected an error for an unknown auth type")
	}
}

func TestLoginRequiresCompanyID(t *testing.T) {
	client := startFakeServer(t, &fakeTaxServer{}, nil)

	_, err := client.Login(context.Background(), &LoginRequest{
		BusinessNumber: "1234567890",
		AuthType:       "certificate",
	})
	if err == nil {
		t.Fatal("expected an error when company_id is absent")
	}
}

func TestLogoutRequiresCompanyIDAndSession(t *testing.T) {
	client := startFakeServer(t, &fakeTaxServer{}, nil)

	if err := client.Logout(context.Background(), "", "sess-1"); err == nil {
		t.Error("expected an error when company_id is absent")
	}
	if err := client.Logout(context.Background(), "company-1", ""); err == nil {
		t.Error("expected an error when session_id is absent")
	}
}

// A refused logout must be reported. Swallowing it leaves a live Hometax
// session open that nobody knows about.
func TestLogoutReportsRefusal(t *testing.T) {
	srv := &fakeTaxServer{
		logoutResp: &taxv1.LogoutResponse{Success: false, ErrorMessage: "session not found"},
	}
	client := startFakeServer(t, srv, nil)

	err := client.Logout(context.Background(), "company-1", "sess-1")
	if err == nil {
		t.Fatal("expected an error for a refused logout")
	}
	if !strings.Contains(err.Error(), "session not found") {
		t.Errorf("error should carry the server's reason, got: %v", err)
	}
}

func TestLogoutSendsCompanyID(t *testing.T) {
	srv := &fakeTaxServer{}
	client := startFakeServer(t, srv, nil)

	if err := client.Logout(context.Background(), "company-1", "sess-1"); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.lastLogout.GetCompanyId() != "company-1" {
		t.Errorf("company_id = %q, want company-1", srv.lastLogout.GetCompanyId())
	}
}

func TestNilClientFailsClosed(t *testing.T) {
	client := NewTaxInvoiceClient(nil)

	if _, err := client.IssueTaxInvoice(context.Background(), validIssueRequest()); err == nil {
		t.Error("expected an error from a client with no connection manager")
	}
	if err := client.HealthCheck(context.Background()); err == nil {
		t.Error("expected an error from HealthCheck with no connection manager")
	}
}

func TestParseScraperTime(t *testing.T) {
	cases := []struct {
		in      string
		want    time.Time
		wantErr bool
	}{
		{in: "", want: time.Time{}},
		{in: "   ", want: time.Time{}},
		{in: "2026-09-07", want: time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)},
		{in: "20260907", want: time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)},
		{in: "2026-09-07T13:45:12", want: time.Date(2026, 9, 7, 13, 45, 12, 0, time.UTC)},
		{in: "2026-09-07T13:45:12.123456", want: time.Date(2026, 9, 7, 13, 45, 12, 123456000, time.UTC)},
		{in: "2026-09-07T13:45:12Z", want: time.Date(2026, 9, 7, 13, 45, 12, 0, time.UTC)},
		{in: "not a date", wantErr: true},
	}

	for _, tc := range cases {
		got, err := parseScraperTime(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseScraperTime(%q): expected an error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseScraperTime(%q): %v", tc.in, err)
			continue
		}
		if !got.Equal(tc.want) {
			t.Errorf("parseScraperTime(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestInvoiceTypeRoundTrip(t *testing.T) {
	for _, name := range []string{"sales", "purchase"} {
		pb, err := invoiceTypeToProto(name)
		if err != nil {
			t.Fatalf("invoiceTypeToProto(%q): %v", name, err)
		}
		if got := invoiceTypeFromProto(pb); got != name {
			t.Errorf("round trip of %q gave %q", name, got)
		}
	}

	if _, err := invoiceTypeToProto("refund"); err == nil {
		t.Error("expected an error for an unknown invoice type")
	}
}

func TestAuthTypeMapping(t *testing.T) {
	if _, err := authTypeToProto("carrier_pigeon"); err == nil {
		t.Error("expected an error for an unknown auth type")
	}
	got, err := authTypeToProto("id_password")
	if err != nil {
		t.Fatalf("authTypeToProto: %v", err)
	}
	if got != taxv1.AuthType_AUTH_TYPE_ID_PASSWORD {
		t.Errorf("auth type = %v", got)
	}
}

func TestProviderMapping(t *testing.T) {
	if got, _ := providerToProto(""); got != taxv1.ProviderType_PROVIDER_TYPE_UNSPECIFIED {
		t.Errorf("empty provider = %v, want UNSPECIFIED", got)
	}
	if got, _ := providerToProto("popbill"); got != taxv1.ProviderType_PROVIDER_TYPE_POPBILL {
		t.Errorf("popbill = %v", got)
	}
	if _, err := providerToProto("barcode"); err == nil {
		t.Error("expected an error for an unknown provider")
	}
}
