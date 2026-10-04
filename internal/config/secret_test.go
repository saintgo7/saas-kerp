package config

import (
	"encoding/base64"
	"encoding/hex"
	"math/rand"
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
		"changeme",
		"your-secret-here",
		"CHANGE_ME_TO_A_REAL_SECRET_BEFORE_DEPLOY",
		"dummy-secret",
		"FIXME",
		"lorem ipsum dolor sit amet consectetur",
		"Xq7#mZ2!pL9",
		"kerp1234kerp1234",
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

// Placeholder words padded with random-looking characters to a legal length must
// still be rejected as placeholders, not let through by the length or entropy
// checks: the word check has to hold on its own.
func TestValidateSecret_RejectsPlaceholderWordsInsideRandomValues(t *testing.T) {
	const random = "Kd8Rp2Ns6Vt9Yc3Hb7Lq1Zg5Ax0Ju4Fo8Wm2PeR6"
	for _, secret := range []string{
		"Kd8Rp2Ns-dummy-6Vt9Yc3Hb7Lq1Zg5Ax0Ju4Fo8Wm2Pe",
		"dummy-secret-" + random,
		random + "_FIXME",
		"your_" + random,
		"Lorem." + random,
		random + "/sample/x",
		"CHANGE_ME_" + random,
		random + "changeme",
	} {
		err := ValidateSecret("jwt.secret", secret)
		if err == nil {
			t.Errorf("ValidateSecret accepted %q", secret)
			continue
		}
		if !strings.Contains(err.Error(), "placeholder") {
			t.Errorf("ValidateSecret(%q) rejected for the wrong reason: %v", secret, err)
		}
	}
}

// A short placeholder word that occurs inside an unbroken random run is chance,
// not authorship. Rejecting it is what made the generated-secret test flaky.
func TestValidateSecret_AcceptsShortWordsInsideRandomRuns(t *testing.T) {
	for _, secret := range []string{
		"Kd8Rp2NsdummyVt9Yc3Hb7Lq1Zg5Ax0Ju4Fo8Wm2PeR6",
		"Kd8Rp2Ns6Vt9Yc3FIXMEHb7Lq1Zg5Ax0Ju4Fo8Wm2PeR6",
		"yourKd8Rp2Ns6Vt9Yc3Hb7Lq1Zg5Ax0Ju4Fo8Wm2PeR6",
	} {
		if err := ValidateSecret("jwt.secret", secret); err != nil {
			t.Errorf("ValidateSecret rejected a random value that merely contains a short word: %v", err)
		}
	}
}

func TestValidateSecret_AcceptsGeneratedSecrets(t *testing.T) {
	// What `openssl rand -base64 48`, `openssl rand -hex 32` and URL-safe base64
	// generators produce must pass, otherwise the check is unusable in practice.
	// The sample is large and the seed fixed: a placeholder token that collides
	// with random output once in a few hundred thousand secrets fails here every
	// time instead of failing one CI run in ten.
	const samples = 200000
	r := rand.New(rand.NewSource(20261005))
	fill := func(b []byte) {
		for i := range b {
			b[i] = byte(r.Int63())
		}
	}

	raw48 := make([]byte, 48)
	raw32 := make([]byte, 32)
	for i := 0; i < samples; i++ {
		fill(raw48)
		if secret := base64.StdEncoding.EncodeToString(raw48); ValidateSecret("jwt.secret", secret) != nil {
			t.Fatalf("ValidateSecret rejected a base64 secret %q: %v", secret, ValidateSecret("jwt.secret", secret))
		}
		if secret := base64.RawURLEncoding.EncodeToString(raw48); ValidateSecret("jwt.secret", secret) != nil {
			t.Fatalf("ValidateSecret rejected a base64url secret %q: %v", secret, ValidateSecret("jwt.secret", secret))
		}
		fill(raw32)
		if secret := hex.EncodeToString(raw32); ValidateSecret("jwt.secret", secret) != nil {
			t.Fatalf("ValidateSecret rejected a hex secret %q: %v", secret, ValidateSecret("jwt.secret", secret))
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
// chance. Bare substrings must be long; short words must be plain words, since
// they are only ever compared against whole words. The hex alphabet is the tight
// case: `openssl rand -hex 32` draws on sixteen symbols.
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
		if len(token) < 7 {
			t.Errorf("placeholder token %q is too short to match as a substring: move it to placeholderWords", token)
		}
		if len(token) < 8 && isHexOnly(token) {
			t.Errorf("placeholder token %q is short and all-hex: it collides with `openssl rand -hex` output", token)
		}
	}

	for word := range placeholderWords {
		if len(word) < 4 || strings.IndexFunc(word, isNotWordRune) >= 0 || strings.ToLower(word) != word {
			t.Errorf("placeholder word %q must be a lower-case word of at least four letters or digits", word)
		}
	}
}
