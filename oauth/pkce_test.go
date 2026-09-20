package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

// rfc7636Unreserved is the code-verifier alphabet: values outside it, or shorter
// than 43 characters, are rejected with invalid_grant.
const rfc7636Unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"

func TestNewPKCE(t *testing.T) {
	first, err := NewPKCE()
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Verifier) < 43 || len(first.Verifier) > 128 {
		t.Fatalf("verifier length = %d, want 43..128", len(first.Verifier))
	}
	if strings.ContainsFunc(first.Verifier, func(r rune) bool {
		return !strings.ContainsRune(rfc7636Unreserved, r)
	}) {
		t.Fatalf("verifier %q leaves the RFC 7636 alphabet", first.Verifier)
	}

	sum := sha256.Sum256([]byte(first.Verifier))
	if want := base64.RawURLEncoding.EncodeToString(sum[:]); first.Challenge != want {
		t.Fatalf("challenge = %q, want %q", first.Challenge, want)
	}

	// Every connection attempt needs its own pair.
	second, err := NewPKCE()
	if err != nil {
		t.Fatal(err)
	}
	if second.Verifier == first.Verifier {
		t.Fatal("two pairs shared a verifier")
	}
}

func TestChallengeFor(t *testing.T) {
	// The worked example from RFC 7636 appendix B.
	const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	const challenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if got := ChallengeFor(verifier); got != challenge {
		t.Fatalf("ChallengeFor = %q, want %q", got, challenge)
	}
}

func TestNewState(t *testing.T) {
	first, err := NewState()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) < 16 {
		t.Fatalf("state = %q, too short to be a CSRF value", first)
	}
	second, err := NewState()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two attempts shared a state")
	}
}

func TestConstantTimeEqual(t *testing.T) {
	for _, tc := range []struct {
		got, want string
		equal     bool
	}{
		{"abc", "abc", true},
		{"abc", "abd", false},
		{"abc", "abcd", false},
		{"", "", true},
	} {
		if got := constantTimeEqual(tc.got, tc.want); got != tc.equal {
			t.Errorf("constantTimeEqual(%q, %q) = %v", tc.got, tc.want, got)
		}
	}
}
