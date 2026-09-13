package grpcclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// The Python services read the same variables from their own environment (see
// python-services/shared/grpc_security.py), so one value configures both ends
// of the connection.
const (
	// EnvAuthToken carries the shared bearer token the Python services'
	// TokenAuthInterceptor compares against.
	EnvAuthToken = "GRPC_AUTH_TOKEN"

	// EnvTLSCAFile is the CA bundle used to verify the server certificate.
	// Unset means the system trust store.
	EnvTLSCAFile = "GRPC_TLS_CA_FILE"

	// EnvTLSClientCert and EnvTLSClientKey are the client key pair presented
	// when the server sets GRPC_TLS_CLIENT_CA and therefore requires mTLS.
	EnvTLSClientCert = "GRPC_TLS_CLIENT_CERT_FILE"
	EnvTLSClientKey  = "GRPC_TLS_CLIENT_KEY_FILE"

	// EnvTLSServerName overrides the name verified against the server
	// certificate, for deployments dialled by container name or IP.
	EnvTLSServerName = "GRPC_TLS_SERVER_NAME"

	// EnvTLSEnabled turns TLS on without any client-side material, for a
	// server whose certificate chains to a publicly trusted CA.
	EnvTLSEnabled = "GRPC_TLS_ENABLED"

	// EnvAllowInsecure is the explicit opt-in to a plaintext, unauthenticated
	// connection. Development only.
	EnvAllowInsecure = "GRPC_ALLOW_INSECURE"
)

// authMetadataKey is the metadata key the Python TokenAuthInterceptor reads.
// gRPC lowercases metadata keys; "authorization" is already lowercase.
const authMetadataKey = "authorization"

// ErrInsecureConfiguration is returned when a connection would be opened with
// neither transport security nor a caller credential outside development.
//
// This mirrors shared/grpc_security.py's InsecureConfigurationError on the
// server side. Failing here rather than dialling anyway is the point: the
// tax scraper accepts 세금계산서 발급 over this channel, and a plaintext,
// tokenless channel is one that anything on the container network can imitate.
var ErrInsecureConfiguration = errors.New("grpcclient: refusing to open a plaintext, unauthenticated connection")

// ClientSecurity holds the transport-security and caller-credential settings
// used for every outbound gRPC connection.
type ClientSecurity struct {
	// AuthToken is presented as "authorization: Bearer <token>" on every RPC.
	AuthToken string

	// TLSEnabled forces TLS even with no CA or client key pair configured.
	// Setting any of the TLS file fields implies it.
	TLSEnabled bool

	// CACertFile verifies the server certificate. Empty means system roots.
	CACertFile string

	// ClientCertFile and ClientKeyFile are presented for mTLS.
	ClientCertFile string
	ClientKeyFile  string

	// ServerName overrides the hostname verified in the server certificate.
	ServerName string

	// AllowInsecure permits a plaintext connection with no token. It exists
	// so a developer can say so out loud rather than have it happen quietly.
	AllowInsecure bool

	// Environment names the deployment. Anything outside the development set
	// is treated as production for the purposes of Validate.
	Environment string
}

// SecurityFromEnv reads the client security settings from the process
// environment.
//
// environment should be the application's own environment name (cfg.App.Env);
// it decides whether a plaintext, tokenless connection is refused.
func SecurityFromEnv(environment string) *ClientSecurity {
	if environment == "" {
		environment = firstNonEmptyEnv("APP_ENV", "KERP_APP_ENV", "ENVIRONMENT")
	}

	return &ClientSecurity{
		AuthToken:      os.Getenv(EnvAuthToken),
		TLSEnabled:     envFlag(EnvTLSEnabled, false),
		CACertFile:     os.Getenv(EnvTLSCAFile),
		ClientCertFile: os.Getenv(EnvTLSClientCert),
		ClientKeyFile:  os.Getenv(EnvTLSClientKey),
		ServerName:     os.Getenv(EnvTLSServerName),
		AllowInsecure:  envFlag(EnvAllowInsecure, false),
		Environment:    environment,
	}
}

// UseTLS reports whether the connection should be encrypted.
func (s *ClientSecurity) UseTLS() bool {
	if s == nil {
		return false
	}
	return s.TLSEnabled || s.CACertFile != "" || s.ClientCertFile != "" || s.ClientKeyFile != ""
}

// IsProduction reports whether this looks like a non-development deployment.
// The set of development names matches shared/grpc_security.py.
func (s *ClientSecurity) IsProduction() bool {
	if s == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(s.Environment)) {
	case "development", "dev", "local", "test", "":
		return false
	default:
		return true
	}
}

// Validate refuses a configuration that offers the server nothing to
// authenticate and nothing to encrypt with.
//
// The rule is deliberately the same as the one the Python server applies to
// itself: TLS or a token is enough; neither is refused outside development
// unless AllowInsecure says so explicitly.
func (s *ClientSecurity) Validate() error {
	if s == nil {
		return fmt.Errorf("%w: no security configuration was supplied", ErrInsecureConfiguration)
	}

	if s.ClientCertFile != "" && s.ClientKeyFile == "" {
		return fmt.Errorf("grpcclient: %s is set without %s; a client certificate needs its private key", EnvTLSClientCert, EnvTLSClientKey)
	}
	if s.ClientKeyFile != "" && s.ClientCertFile == "" {
		return fmt.Errorf("grpcclient: %s is set without %s; a private key needs its certificate", EnvTLSClientKey, EnvTLSClientCert)
	}

	if s.UseTLS() || s.AuthToken != "" {
		return nil
	}

	if s.AllowInsecure {
		return nil
	}

	if s.IsProduction() {
		return fmt.Errorf(
			"%w in environment %q. Set %s, or %s/%s/%s, or set %s=true to accept the risk deliberately",
			ErrInsecureConfiguration, s.Environment,
			EnvAuthToken, EnvTLSEnabled, EnvTLSCAFile, EnvTLSClientCert,
			EnvAllowInsecure,
		)
	}

	// Development: allowed, because a local tax-scraper started without
	// GRPC_AUTH_TOKEN has no token to present. The server logs the same
	// warning from its own side.
	return nil
}

// transportCredentials builds the transport credentials for the connection.
func (s *ClientSecurity) transportCredentials() (credentials.TransportCredentials, error) {
	if !s.UseTLS() {
		return insecure.NewCredentials(), nil
	}

	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: s.ServerName,
	}

	if s.CACertFile != "" {
		pem, err := os.ReadFile(s.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("grpcclient: failed to read %s %q: %w", EnvTLSCAFile, s.CACertFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("grpcclient: %q contains no PEM certificate", s.CACertFile)
		}
		cfg.RootCAs = pool
	}

	if s.ClientCertFile != "" {
		pair, err := tls.LoadX509KeyPair(s.ClientCertFile, s.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("grpcclient: failed to load client key pair (%s / %s): %w", s.ClientCertFile, s.ClientKeyFile, err)
		}
		cfg.Certificates = []tls.Certificate{pair}
	}

	return credentials.NewTLS(cfg), nil
}

// tokenCallCredentials returns per-RPC credentials carrying the shared bearer
// token, or nil when no token is configured.
func (s *ClientSecurity) tokenCallCredentials() credentials.PerRPCCredentials {
	if s == nil || s.AuthToken == "" {
		return nil
	}
	return bearerToken{token: s.AuthToken, requireTLS: s.UseTLS()}
}

// bearerToken presents the shared token the Python TokenAuthInterceptor
// expects.
type bearerToken struct {
	token      string
	requireTLS bool
}

// GetRequestMetadata returns the authorization header for a call.
func (b bearerToken) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{authMetadataKey: "Bearer " + b.token}, nil
}

// RequireTransportSecurity reports whether gRPC must refuse to send the token
// over a plaintext connection.
//
// It returns false when TLS is not configured, because the development
// deployment described in deployments/docker runs the Python services on a
// plaintext port with a token. Returning true there would make the token
// unusable rather than make the connection safer. Validate is what keeps that
// combination out of production.
func (b bearerToken) RequireTransportSecurity() bool {
	return b.requireTLS
}

// dialOptions returns the security-related dial options for a connection.
func (s *ClientSecurity) dialOptions() ([]grpc.DialOption, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}

	creds, err := s.transportCredentials()
	if err != nil {
		return nil, err
	}

	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}

	if callCreds := s.tokenCallCredentials(); callCreds != nil {
		opts = append(opts, grpc.WithPerRPCCredentials(callCreds))
	}

	return opts, nil
}

// AuthorizeContext attaches the bearer token to an outgoing context.
//
// The dial options already do this for every RPC on the channel; this exists
// for callers that build their own context for a one-off connection.
func (s *ClientSecurity) AuthorizeContext(ctx context.Context) context.Context {
	if s == nil || s.AuthToken == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, authMetadataKey, "Bearer "+s.AuthToken)
}

// envFlag reads a boolean environment variable using the same spellings the
// Python side accepts.
func envFlag(name string, def bool) bool {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off", "":
		return false
	default:
		return def
	}
}

// firstNonEmptyEnv returns the value of the first variable that is set and
// non-empty.
func firstNonEmptyEnv(names ...string) string {
	for _, name := range names {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	return ""
}
