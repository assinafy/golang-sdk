package oauth

import (
	"context"
	"fmt"
	"sync"
)

// TokenSource keeps one user's connection usable: it hands out the current
// access token and renews it with the refresh token when it expires. Pass it to
// assinafy.ClientOptions.TokenSource and the SDK authenticates every request
// with a fresh token.
//
// One TokenSource holds one connection, which is one user in one workspace. A
// customer who uses several workspaces connects each one separately, so keep a
// token, a workspace ID and a TokenSource per connection.
//
// A TokenSource is safe for concurrent use. Refreshes are serialized, which also
// satisfies the API's rule that a connection refreshes one at a time.
type TokenSource struct {
	config *Config
	// onRefresh persists a renewed token before it is used. Rotation makes this
	// mandatory rather than advisory.
	onRefresh func(*Token) error

	mu    sync.Mutex
	token Token
}

// NewTokenSource builds a TokenSource for one connection from the token a
// previous Exchange or Refresh produced.
//
// onRefresh is called with the renewed token, while the internal lock is held,
// before that token is handed to any caller. Store the new RefreshToken there:
// every refresh retires the previous one, and replaying a retired refresh token
// disconnects the user entirely. Returning an error from onRefresh aborts the
// refresh, so a failed save never leaves your storage holding a dead token. A
// nil onRefresh is allowed only when the token carries no refresh token.
func NewTokenSource(config *Config, token *Token, onRefresh func(*Token) error) *TokenSource {
	source := &TokenSource{config: config, onRefresh: onRefresh}
	if token != nil {
		source.token = *token
	}
	return source
}

// Token returns a valid access token, refreshing first when the current one is
// within a minute of expiring.
//
// It returns ErrNoRefreshToken when the access token has expired and the
// connection has none, which means the user must approve the application again;
// request ScopeOfflineAccess to avoid that. A refresh rejected with
// ErrCodeInvalidGrant means the connection is over: the refresh token was
// already used or expired, the 30-day connection lifetime ran out, or the user
// reconnected with different permissions. Ask the user to connect again.
func (s *TokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token.Valid() {
		return s.token.AccessToken, nil
	}
	if s.token.RefreshToken == "" {
		return "", ErrNoRefreshToken
	}

	refreshed, err := s.config.Refresh(ctx, s.token.RefreshToken)
	if err != nil {
		return "", err
	}
	// The server issues a new refresh token on every refresh and retires the old
	// one, so carry it forward when the response omits it and persist it before
	// the access token is used.
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = s.token.RefreshToken
	}
	if s.onRefresh != nil {
		if err := s.onRefresh(refreshed); err != nil {
			return "", fmt.Errorf("assinafy/oauth: persist refreshed token: %w", err)
		}
	}
	s.token = *refreshed
	return s.token.AccessToken, nil
}

// Current returns a copy of the token the source holds, including any refresh
// performed since it was created. Use it to persist the connection on shutdown.
func (s *TokenSource) Current() Token {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token
}
