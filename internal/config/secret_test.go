package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
)

func TestValidateSecret_RejectsShippedPlaceholders(t *testing.T) {
	// Every value below has appeared in this repository, its .env.example or its
	// compose files at some point. All of them are long enough to have passed the
	// previous "at least 32 characters" check.
	placeholders := []string{
		"change-this-to-a-long-random-string-at-least-32-chars",
		"change-me-in-production",
		"dev-secret-change-in-production",
		"CHANGE-THIS-TO-A-LONG-RANDOM-STRING-AT-LEAST-32-CHARS",
		"your-secret-key-please-change-this-value",
		"this-is-an-example-secret-for-documentation",
		"placeholder-secret-value-for-local-testing",
		"my-super-secret-jwt-signing-key-1234567890",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"Kd8Rp2Nsaaaaaaaa6Vt9Yc3Hb7Lq1Zg5Ax0Ju4Fo8",
		"abababababababababababababababababababab",
		"abcdefghijklmnopqrstuvwxyzabcdefghijklmn",
		"0123456789012345678901234567890123456789",
		"password-password-password-password-1234",
	}

	for _, secret := range placeholders {
		if err := ValidateSecret("jwt.secret", secret); err == nil {
			t.Errorf("ValidateSecret accepted a placeholder secret: %q", secret)
		}
	}
}

// The infrastructure branch standardised every secret in .env.example on this
// exact string. It must be rejected, and the message must say "replace it"
// rather than "make it longer".
func TestValidateSecret_RejectsTheEnvExamplePlaceholder(t *testing.T) {
	const shipped = "change-me-in-production"

	for _, name := range []string{"jwt.secret", "database.password", "redis.password"} {
		err := ValidateSecret(name, shipped)
		if err == nil {
			t.Fatalf("ValidateSecret(%s) accepted the .env.example placeholder", name)
		}
		if !strings.Contains(err.Error(), "replaced") {
			t.Errorf("ValidateSecret(%s) error does not tell the operator to replace the value: %v", name, err)
		}
		if strings.Contains(err.Error(), "at least 32 characters") {
			t.Errorf("ValidateSecret(%s) reported a length problem for a placeholder: %v", name, err)
		}
	}

	// Padding the placeholder out to a legal length must not help.
	padded := shipped + "-0123456789abcdef0123456789"
	if len(padded) < MinSecretLength {
		t.Fatalf("test setup: padded value is only %d characters", len(padded))
	}
	if err := ValidateSecret("jwt.secret", padded); err == nil {
		t.Error("ValidateSecret accepted the placeholder padded to the minimum length")
	}
}

func TestValidateSecret_RejectsShortSecrets(t *testing.T) {
	if err := ValidateSecret("jwt.secret", "Xq7#mZ2!pL9"); err == nil {
		t.Error("ValidateSecret accepted a secret shorter than the minimum length")
	}
}

func TestValidateSecret_RejectsEmpty(t *testing.T) {
	if err := ValidateSecret("jwt.secret", ""); err == nil {
		t.Error("ValidateSecret accepted an empty secret")
	}
	if err := ValidateSecret("jwt.secret", "                                    "); err == nil {
		t.Error("ValidateSecret accepted a whitespace-only secret")
	}
}

func TestValidateSecret_AcceptsGeneratedSecrets(t *testing.T) {
	// What `openssl rand -base64 48` and `openssl rand -hex 32` produce must pass,
	// otherwise the check is unusable in practice. The iteration count is high on
	// purpose: an earlier version of the placeholder list contained "xxx", which
	// a random base64 secret hits roughly once in four thousand, and a small
	// sample would not have caught it.
	for i := 0; i < 20000; i++ {
		raw := make([]byte, 48)
		if _, err := rand.Read(raw); err != nil {
			t.Fatalf("rand.Read: %v", err)
		}
		secret := base64.StdEncoding.EncodeToString(raw)
		if err := ValidateSecret("jwt.secret", secret); err != nil {
			t.Fatalf("ValidateSecret rejected a generated secret %q: %v", secret, err)
		}
	}

	for i := 0; i < 20000; i++ {
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			t.Fatalf("rand.Read: %v", err)
		}
		secret := hex.EncodeToString(raw)
		if err := ValidateSecret("jwt.secret", secret); err != nil {
			t.Fatalf("ValidateSecret rejected a hex secret %q: %v", secret, err)
		}
	}
}

func TestValidateSecret_RunsInValidatorForNonDevelopment(t *testing.T) {
	base := func(env, secret string) *Config {
		return &Config{
			App:       AppConfig{Env: env, Port: 8080},
			Database:  DatabaseConfig{Host: "db", Name: "kerp", User: "kerp", MaxOpenConns: 10},
			JWT:       JWTConfig{Secret: secret, AccessTokenTTL: 1, RefreshTokenTTL: 1},
			CORS:      CORSConfig{AllowedOrigins: []string{"http://localhost:3000"}},
			RateLimit: RateLimitConfig{Enabled: true, RequestsPerSecond: 10, Burst: 20},
			Log:       LogConfig{Level: "info", Format: "json"},
		}
	}

	const placeholder = "change-this-to-a-long-random-string-at-least-32-chars"

	for _, env := range []string{"staging", "production"} {
		if err := base(env, placeholder).Validate(); err == nil {
			t.Errorf("Validate accepted the .env.example JWT secret in %s", env)
		}
	}

	// Development keeps working with the checked-in default, so nobody has to
	// generate a secret to run the test suite.
	if err := base("development", placeholder).Validate(); err != nil {
		t.Errorf("Validate rejected the development default: %v", err)
	}
}

func TestSecretFingerprint(t *testing.T) {
	const secret = "a-secret-value-that-should-never-be-logged"

	fp := SecretFingerprint(secret)
	if len(fp) != 8 {
		t.Fatalf("fingerprint should be 8 characters, got %q", fp)
	}
	if fp == secret[:8] {
		t.Error("fingerprint leaks the beginning of the secret")
	}
	if SecretFingerprint(secret) != fp {
		t.Error("fingerprint is not stable")
	}
	if SecretFingerprint("") != "unset" {
		t.Error("empty secret should report as unset")
	}
}

// A placeholder token must not be something a generated secret can contain by
// chance. The hex alphabet is the tight case: `openssl rand -hex 32` draws on
// sixteen symbols, so a short all-hex token collides often enough to reject good
// secrets.
func TestPlaceholderTokensCannotCollide(t *testing.T) {
	const hexAlphabet = "0123456789abcdef"

	isHexOnly := func(s string) bool {
		for _, r := range s {
			if !strings.ContainsRune(hexAlphabet, r) {
				return false
			}
		}
		return true
	}

	for _, token := range placeholderTokens {
		if len(token) < 5 {
			t.Errorf("placeholder token %q is too short: it will reject generated secrets by chance", token)
		}
		if len(token) < 8 && isHexOnly(token) {
			t.Errorf("placeholder token %q is short and all-hex: it collides with `openssl rand -hex` output", token)
		}
	}
}
