package grpcclient

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSecurityValidateRefusesPlaintextInProduction(t *testing.T) {
	sec := &ClientSecurity{Environment: "production"}

	err := sec.Validate()
	if err == nil {
		t.Fatal("expected production without TLS or a token to be refused")
	}
	if !errors.Is(err, ErrInsecureConfiguration) {
		t.Errorf("error = %v, want ErrInsecureConfiguration", err)
	}
}

func TestSecurityValidateAcceptsTokenInProduction(t *testing.T) {
	sec := &ClientSecurity{Environment: "production", AuthToken: "token"}

	if err := sec.Validate(); err != nil {
		t.Fatalf("a token should be enough: %v", err)
	}
}

func TestSecurityValidateAcceptsTLSInProduction(t *testing.T) {
	sec := &ClientSecurity{Environment: "production", TLSEnabled: true}

	if err := sec.Validate(); err != nil {
		t.Fatalf("TLS should be enough: %v", err)
	}
}

func TestSecurityValidateHonoursExplicitOptIn(t *testing.T) {
	sec := &ClientSecurity{Environment: "production", AllowInsecure: true}

	if err := sec.Validate(); err != nil {
		t.Fatalf("an explicit opt-in should be honoured: %v", err)
	}
}

func TestSecurityValidateAllowsDevelopmentPlaintext(t *testing.T) {
	for _, env := range []string{"", "development", "dev", "local", "test"} {
		sec := &ClientSecurity{Environment: env}
		if err := sec.Validate(); err != nil {
			t.Errorf("environment %q should allow plaintext: %v", env, err)
		}
	}
}

func TestSecurityValidateRejectsHalfAKeyPair(t *testing.T) {
	if err := (&ClientSecurity{ClientCertFile: "cert.pem"}).Validate(); err == nil {
		t.Error("a client certificate without its key should be refused")
	}
	if err := (&ClientSecurity{ClientKeyFile: "key.pem"}).Validate(); err == nil {
		t.Error("a client key without its certificate should be refused")
	}
}

func TestSecurityValidateRejectsNil(t *testing.T) {
	var sec *ClientSecurity
	if err := sec.Validate(); err == nil {
		t.Fatal("a nil security configuration should be refused")
	}
}

func TestUseTLSIsImpliedByAnyTLSFile(t *testing.T) {
	cases := []struct {
		name string
		sec  ClientSecurity
		want bool
	}{
		{"nothing", ClientSecurity{}, false},
		{"explicit", ClientSecurity{TLSEnabled: true}, true},
		{"ca", ClientSecurity{CACertFile: "ca.pem"}, true},
		{"client cert", ClientSecurity{ClientCertFile: "cert.pem"}, true},
		{"client key", ClientSecurity{ClientKeyFile: "key.pem"}, true},
	}

	for _, tc := range cases {
		if got := tc.sec.UseTLS(); got != tc.want {
			t.Errorf("%s: UseTLS() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestBearerTokenMetadata(t *testing.T) {
	sec := &ClientSecurity{AuthToken: "abc123"}

	creds := sec.tokenCallCredentials()
	if creds == nil {
		t.Fatal("expected per-RPC credentials when a token is configured")
	}

	md, err := creds.GetRequestMetadata(context.Background())
	if err != nil {
		t.Fatalf("GetRequestMetadata: %v", err)
	}
	if md[authMetadataKey] != "Bearer abc123" {
		t.Errorf("%s = %q, want %q", authMetadataKey, md[authMetadataKey], "Bearer abc123")
	}

	// Plaintext development connections must still be able to present the
	// token, otherwise gRPC refuses to send it and every call fails
	// UNAUTHENTICATED against a token-protected dev server.
	if creds.RequireTransportSecurity() {
		t.Error("a plaintext connection should still carry the token")
	}

	if (&ClientSecurity{AuthToken: "abc123", TLSEnabled: true}).tokenCallCredentials().RequireTransportSecurity() != true {
		t.Error("a TLS connection should require transport security for the token")
	}
}

func TestNoTokenMeansNoCallCredentials(t *testing.T) {
	if creds := (&ClientSecurity{}).tokenCallCredentials(); creds != nil {
		t.Error("expected no per-RPC credentials without a token")
	}
	var nilSec *ClientSecurity
	if creds := nilSec.tokenCallCredentials(); creds != nil {
		t.Error("expected no per-RPC credentials from a nil configuration")
	}
}

func TestAuthorizeContextIsANoOpWithoutAToken(t *testing.T) {
	ctx := context.Background()
	if got := (&ClientSecurity{}).AuthorizeContext(ctx); got != ctx {
		t.Error("expected the context to be returned unchanged")
	}
}

func TestSecurityFromEnvReadsTheSameVariablesAsThePythonServer(t *testing.T) {
	t.Setenv(EnvAuthToken, "shared-token")
	t.Setenv(EnvTLSCAFile, "/etc/ssl/ca.pem")
	t.Setenv(EnvAllowInsecure, "true")

	sec := SecurityFromEnv("staging")

	if sec.AuthToken != "shared-token" {
		t.Errorf("AuthToken = %q", sec.AuthToken)
	}
	if sec.CACertFile != "/etc/ssl/ca.pem" {
		t.Errorf("CACertFile = %q", sec.CACertFile)
	}
	if !sec.AllowInsecure {
		t.Error("AllowInsecure should be true")
	}
	if !sec.IsProduction() {
		t.Error("staging should count as production")
	}
}

func TestDialFailsClosedOnAnInsecureProductionConfiguration(t *testing.T) {
	manager := NewManager(&ClientConfig{
		TaxScraperAddr: "127.0.0.1:1",
		Security:       &ClientSecurity{Environment: "production"},
	})
	t.Cleanup(func() { _ = manager.Close() })

	_, err := manager.TaxScraperConn(context.Background())
	if err == nil {
		t.Fatal("expected the dial to be refused")
	}
	if !errors.Is(err, ErrInsecureConfiguration) {
		t.Errorf("error = %v, want ErrInsecureConfiguration", err)
	}
}

func TestServiceConfigDropsRetriesForIssuance(t *testing.T) {
	manager := NewManager(&ClientConfig{
		MaxRetryAttempts: 3,
		Security:         &ClientSecurity{Environment: "test"},
	})

	cfg := manager.serviceConfigJSON()

	// The read-only methods are retried.
	if !strings.Contains(cfg, "GetTaxInvoices") {
		t.Error("GetTaxInvoices should carry a retry policy")
	}
	// The filings are not: a silent second attempt is a duplicate 세금계산서.
	for _, method := range []string{"IssueTaxInvoice", "CancelTaxInvoice", "Login"} {
		if strings.Contains(cfg, method) {
			t.Errorf("%s must not carry a retry policy", method)
		}
	}
}

func TestServiceConfigWithoutRetries(t *testing.T) {
	manager := NewManager(&ClientConfig{
		MaxRetryAttempts: 1,
		Security:         &ClientSecurity{Environment: "test"},
	})

	if strings.Contains(manager.serviceConfigJSON(), "retryPolicy") {
		t.Error("a single attempt should produce no retry policy")
	}
}

func TestServiceAddrFromEnv(t *testing.T) {
	t.Setenv("TAX_SCRAPER_HOST", "tax-scraper")
	t.Setenv("TAX_SCRAPER_PORT", "50051")

	if got := serviceAddrFromEnv("TAX_SCRAPER_ADDR", "TAX_SCRAPER_HOST", "TAX_SCRAPER_PORT"); got != "tax-scraper:50051" {
		t.Errorf("addr = %q, want tax-scraper:50051", got)
	}

	t.Setenv("TAX_SCRAPER_ADDR", "override:9999")
	if got := serviceAddrFromEnv("TAX_SCRAPER_ADDR", "TAX_SCRAPER_HOST", "TAX_SCRAPER_PORT"); got != "override:9999" {
		t.Errorf("addr = %q, want override:9999", got)
	}
}
