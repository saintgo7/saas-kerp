package grpcclient

import (
	"context"
	"os"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Environment variables that turn the live probe on. They are deliberately not
// read from the normal configuration: this test talks to a real tax scraper
// and must never run by accident.
const (
	envLiveAddr  = "KERP_LIVE_SCRAPER_ADDR"
	envLiveToken = "KERP_LIVE_SCRAPER_TOKEN"

	// A session id that the scraper will recognise. Without it the probe can
	// only reach the authentication check, which is still worth knowing.
	envLiveSession = "KERP_LIVE_SCRAPER_SESSION"
	envLiveCompany = "KERP_LIVE_SCRAPER_COMPANY"
)

// liveClient builds a client pointed at a running tax scraper, or skips.
func liveClient(t *testing.T, token string) *TaxInvoiceClient {
	t.Helper()

	addr := os.Getenv(envLiveAddr)
	if addr == "" {
		t.Skipf("set %s to run this against a live tax scraper", envLiveAddr)
	}

	manager := NewManager(&ClientConfig{
		TaxScraperAddr:   addr,
		DialTimeout:      3 * time.Second,
		KeepAliveTime:    30 * time.Second,
		KeepAliveTimeout: 10 * time.Second,
		MaxRetryAttempts: 1,
		CallTimeout:      10 * time.Second,
		Security: &ClientSecurity{
			Environment:   "test",
			AuthToken:     token,
			AllowInsecure: token == "",
		},
	})
	t.Cleanup(func() { _ = manager.Close() })

	return NewTaxInvoiceClient(manager)
}

// TestLiveIssueTaxInvoice drives IssueTaxInvoice against a running scraper.
//
//	KERP_LIVE_SCRAPER_ADDR=127.0.0.1:50051 \
//	KERP_LIVE_SCRAPER_TOKEN=$GRPC_AUTH_TOKEN \
//	go test ./internal/grpcclient/ -run TestLive -v
func TestLiveIssueTaxInvoice(t *testing.T) {
	client := liveClient(t, os.Getenv(envLiveToken))

	companyID := os.Getenv(envLiveCompany)
	if companyID == "" {
		companyID = "11111111-1111-1111-1111-111111111111"
	}
	sessionID := os.Getenv(envLiveSession)
	if sessionID == "" {
		sessionID = "live-probe-session"
	}

	req := validIssueRequest()
	req.CompanyID = companyID
	req.SessionID = sessionID

	resp, err := client.IssueTaxInvoice(context.Background(), req)
	if err != nil {
		t.Fatalf("IssueTaxInvoice against %s: %v", os.Getenv(envLiveAddr), err)
	}
	if !resp.Success {
		t.Fatalf("scraper reported failure: %s (%s)", resp.ErrorMessage, resp.ErrorCode)
	}
	if resp.NTSConfirmNumber == "" {
		t.Fatal("scraper reported success with no NTS confirmation number")
	}
	t.Logf("승인번호 %s, 발급일 %s", resp.NTSConfirmNumber, resp.IssueDate)
}

// TestLiveRejectsWrongToken checks that the server's own token interceptor
// turns a bad credential into UNAUTHENTICATED, and that the client surfaces it
// rather than absorbing it.
func TestLiveRejectsWrongToken(t *testing.T) {
	client := liveClient(t, "definitely-not-the-token")

	req := validIssueRequest()

	_, err := client.IssueTaxInvoice(context.Background(), req)
	if err == nil {
		t.Fatal("expected the scraper to reject a bad token")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("status = %v, want Unauthenticated: %v", status.Code(err), err)
	}
	t.Logf("rejected as expected: %v", err)
}
