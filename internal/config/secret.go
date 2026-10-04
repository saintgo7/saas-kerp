package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"unicode"
)

// Minimum requirements for a JWT signing secret outside development.
const (
	// MinSecretLength is the minimum accepted secret length in bytes.
	MinSecretLength = 32
	// MinSecretDistinctChars is the minimum number of distinct characters a
	// secret must contain. Repeated or patterned secrets ("aaaa...", "abcabc...")
	// fail this check even when they are long enough.
	MinSecretDistinctChars = 12
	// maxIdenticalRun is the longest run of one repeated character a secret may
	// contain. It catches a short secret padded out to the length minimum
	// ("...aaaaaaaaaaaa"), which the distinct-character and entropy checks miss
	// when the rest of the value is varied.
	//
	// Eight, not six: hex output draws on 16 symbols, so a six-character run
	// turns up in about one `openssl rand -hex 32` in eighteen thousand. At
	// eight it is one in five million.
	maxIdenticalRun = 8
	// MinSecretEntropyBits is the minimum Shannon entropy, in bits, of the whole
	// secret. A 32-character secret drawn from a 16-symbol alphabet carries 128
	// bits; anything under this bound is a hand-typed string, not a random one.
	MinSecretEntropyBits = 80
)

// exactWeakSecrets are literal values that have shipped in this repository or in
// its documentation at some point. They must never reach a deployed environment.
var exactWeakSecrets = map[string]bool{
	"change-me-in-production":                               true,
	"dev-secret-change-in-production":                       true,
	"change-this-to-a-long-random-string-at-least-32-chars": true,
	"your-secret-key":                                       true,
	"your-256-bit-secret":                                   true,
	"kerp-jwt-secret":                                       true,
	"kerp-secret-key":                                       true,
	"supersecretkey":                                        true,
	"secret":                                                true,
}

// placeholderTokens are substrings that only appear in human-authored
// placeholders. Any secret containing one of these is rejected regardless of its
// length or entropy, which is what stops `.env.example` values from booting a
// staging or production server.
//
// Every token here is at least seven characters long, so a generated secret
// contains one by chance far too rarely to matter: a lower-cased 64-character
// base64 secret hits a given seven-letter run about once in six hundred million.
// Shorter words live in placeholderWords, which only match whole words. A five
// letter token matched as a bare substring ("dummy", "fixme") turned up in about
// one generated secret in two hundred thousand and made CI flaky. Runs of a
// repeated character are caught by hasLongRun instead.
var placeholderTokens = []string{
	"changeme",
	"change-me",
	"change_me",
	"change-this",
	"change-in",
	"changein",
	"tobechanged",
	"replace-me",
	"replaceme",
	"placeholder",
	"example",
	"yoursecret",
	"yourkey",
	"secret-key",
	"secretkey",
	"jwt-secret",
	"jwtsecret",
	"mysecret",
	"topsecret",
	"supersecret",
	"insecure",
	"notsecure",
	"default",
	"password",
	"letmein",
	"deadbeef",
	"random-string",
	"randomstring",
	"at-least",
	"atleast",
	"32-chars",
	"32chars",
	"test-secret",
	"testsecret",
	"localhost",
	"kerp-api",
}

// placeholderWords are short placeholder words that are rejected only when they
// stand as a whole word, i.e. bounded on both sides by the start or end of the
// value or by a character that is not a letter or digit ("dummy-secret",
// "your_secret_here", "x/fixme/y"). A generated secret is one unbroken run of
// letters and digits (hex), or has a separator ("+", "/", or "-", "_" in the URL
// alphabet) only about once every thirty characters (base64), so it isolates one
// of these words by chance about once in ten million secrets at worst ("your"),
// while a human-written placeholder separates its words. "Kd8dummyRp2" is
// accepted: letters inside a random run carry no sign of human authorship.
var placeholderWords = map[string]bool{
	"your":   true,
	"dummy":  true,
	"sample": true,
	"passwd": true,
	"qwerty": true,
	"fixme":  true,
	"lorem":  true,
}

// separatorStripper removes the word separators a human uses inside a
// placeholder but a secret generator never emits.
var separatorStripper = strings.NewReplacer("-", "", "_", "", ".", "", " ", "")

// ValidateSecret reports why the given secret is unacceptable for the named
// setting, or nil when it passes every check. It is intentionally strict: a
// false rejection costs an operator one `openssl rand` invocation, while a false
// acceptance hands every tenant's data to anyone who has read the repository.
func ValidateSecret(name, secret string) error {
	if strings.TrimSpace(secret) == "" {
		return fmt.Errorf("%s is required", name)
	}

	normalized := strings.ToLower(strings.TrimSpace(secret))

	// The placeholder checks run before the length check on purpose. The
	// shipped placeholders are shorter than the minimum, and telling an operator
	// "add more characters" would send them to lengthen the value they were
	// supposed to replace.
	if exactWeakSecrets[normalized] {
		return fmt.Errorf("%s is the placeholder value shipped in .env.example and must be replaced (generate one with: openssl rand -base64 48)", name)
	}

	// Scan the value both as written and with the separators stripped, so
	// "my-super-secret-key" is caught by the "supersecret" token. Neither
	// `openssl rand -base64` nor `openssl rand -hex` emits "-" or "_", so
	// stripping them cannot merge two halves of a generated secret into a word.
	compact := separatorStripper.Replace(normalized)
	for _, token := range placeholderTokens {
		if strings.Contains(normalized, token) || strings.Contains(compact, token) {
			return fmt.Errorf("%s looks like a placeholder (contains %q) and must be replaced (generate one with: openssl rand -base64 48)", name, token)
		}
	}
	for _, word := range strings.FieldsFunc(normalized, isNotWordRune) {
		if placeholderWords[word] {
			return fmt.Errorf("%s looks like a placeholder (contains %q) and must be replaced (generate one with: openssl rand -base64 48)", name, word)
		}
	}

	if len(secret) < MinSecretLength {
		return fmt.Errorf("%s is too weak (must be at least %d characters, got %d)", name, MinSecretLength, len(secret))
	}

	if distinctRunes(secret) < MinSecretDistinctChars {
		return fmt.Errorf("%s is too repetitive (needs at least %d distinct characters)", name, MinSecretDistinctChars)
	}

	if isRepeatedPattern(secret) {
		return fmt.Errorf("%s repeats a short pattern and carries far less entropy than its length suggests", name)
	}

	if isSequential(secret) {
		return fmt.Errorf("%s is a character sequence, not a random secret", name)
	}

	if hasLongRun(secret, maxIdenticalRun) {
		return fmt.Errorf("%s contains a run of %d or more identical characters and is not a generated secret", name, maxIdenticalRun)
	}

	if !hasMixedCharacterClasses(secret) {
		return fmt.Errorf("%s draws on a single character class; use a generated secret (openssl rand -base64 48)", name)
	}

	if bits := shannonEntropyBits(secret); bits < MinSecretEntropyBits {
		return fmt.Errorf("%s has insufficient entropy (%.0f bits, need at least %d; generate one with: openssl rand -base64 48)", name, bits, MinSecretEntropyBits)
	}

	return nil
}

// SecretFingerprint returns a short, non-reversible identifier for a secret so
// operators can confirm which value a running process loaded without the value
// itself ever appearing in a log.
func SecretFingerprint(secret string) string {
	if secret == "" {
		return "unset"
	}
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])[:8]
}

// isNotWordRune reports whether r separates words: anything but a letter or digit.
func isNotWordRune(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

// distinctRunes counts the distinct runes in s.
func distinctRunes(s string) int {
	seen := make(map[rune]struct{}, len(s))
	for _, r := range s {
		seen[r] = struct{}{}
	}
	return len(seen)
}

// shannonEntropyBits returns the Shannon entropy of the whole string in bits.
func shannonEntropyBits(s string) float64 {
	if s == "" {
		return 0
	}
	counts := make(map[rune]int, len(s))
	total := 0
	for _, r := range s {
		counts[r]++
		total++
	}

	var perChar float64
	for _, n := range counts {
		p := float64(n) / float64(total)
		perChar -= p * math.Log2(p)
	}
	return perChar * float64(total)
}

// isRepeatedPattern reports whether s is a short unit repeated to fill the
// length requirement, e.g. "abcabcabcabc..." or "01010101...".
func isRepeatedPattern(s string) bool {
	n := len(s)
	for unit := 1; unit <= n/3; unit++ {
		if n%unit != 0 {
			continue
		}
		matches := true
		for i := unit; i < n; i++ {
			if s[i] != s[i-unit] {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

// hasLongRun reports whether s contains n or more identical characters in a row.
func hasLongRun(s string, n int) bool {
	if n < 2 || len(s) < n {
		return false
	}
	run := 1
	for i := 1; i < len(s); i++ {
		if s[i] == s[i-1] {
			run++
			if run >= n {
				return true
			}
			continue
		}
		run = 1
	}
	return false
}

// isSequential reports whether s is (almost) entirely an ascending or descending
// run of adjacent characters, e.g. "abcdefghij...".
func isSequential(s string) bool {
	if len(s) < 4 {
		return false
	}
	ascending, descending := true, true
	for i := 1; i < len(s); i++ {
		if s[i] != s[i-1]+1 {
			ascending = false
		}
		if s[i] != s[i-1]-1 {
			descending = false
		}
		if !ascending && !descending {
			return false
		}
	}
	return true
}

// hasMixedCharacterClasses is retained for callers that want a softer signal
// than ValidateSecret; it reports whether s draws on more than one class.
func hasMixedCharacterClasses(s string) bool {
	var lower, upper, digit, other bool
	for _, r := range s {
		switch {
		case unicode.IsLower(r):
			lower = true
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsDigit(r):
			digit = true
		default:
			other = true
		}
	}
	classes := 0
	for _, present := range []bool{lower, upper, digit, other} {
		if present {
			classes++
		}
	}
	return classes > 1
}
