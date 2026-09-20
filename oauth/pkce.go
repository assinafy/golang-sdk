package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
)

// verifierBytes is 32 random bytes, which base64url-encode to the 43-character
// minimum RFC 7636 allows. Shorter values are rejected with invalid_grant.
const verifierBytes = 32

// stateBytes is 16 random bytes, enough entropy for a per-attempt CSRF value.
const stateBytes = 16

// PKCE is one Proof Key for Code Exchange pair. PKCE is mandatory for every
// Assinafy application, confidential ones included.
//
// Keep Verifier in the user's session and send Challenge to the authorization
// server. The token exchange proves the client that redeems the code is the one
// that started the flow by presenting the verifier the challenge was derived from.
type PKCE struct {
	// Verifier is the secret kept by your server and replayed to Config.Exchange.
	// It is 43 characters from the RFC 7636 unreserved set.
	Verifier string
	// Challenge is the base64url-encoded SHA-256 of Verifier, sent in the
	// authorization URL with code_challenge_method=S256.
	Challenge string
}

// NewPKCE creates a fresh verifier and its S256 challenge from the system's
// cryptographic random source. Create a new pair for every connection attempt
// and never reuse one.
func NewPKCE() (PKCE, error) {
	verifier, err := randomURLSafe(verifierBytes)
	if err != nil {
		return PKCE{}, fmt.Errorf("assinafy/oauth: generate code verifier: %w", err)
	}
	return PKCE{Verifier: verifier, Challenge: ChallengeFor(verifier)}, nil
}

// ChallengeFor returns the S256 code challenge for an existing verifier. Use it
// when the verifier comes from your own storage rather than from NewPKCE.
func ChallengeFor(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// NewState creates the random per-attempt state value that protects the
// callback against cross-site request forgery. Store it in the user's session
// and pass it back to Config.ParseCallback.
func NewState() (string, error) {
	state, err := randomURLSafe(stateBytes)
	if err != nil {
		return "", fmt.Errorf("assinafy/oauth: generate state: %w", err)
	}
	return state, nil
}

func randomURLSafe(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// constantTimeEqual compares two values without leaking their contents through
// timing. Lengths are compared first because ConstantTimeCompare returns 0 for
// mismatched lengths regardless of content.
func constantTimeEqual(got, want string) bool {
	if len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
