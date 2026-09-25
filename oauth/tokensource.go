package oauth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// stepTimeout bounds each step a TokenSource runs in the background: the token
// request, and the save of the token it returns.
const stepTimeout = time.Minute

// TokenSource keeps one user's connection usable: it hands out the current
// access token and renews it with the refresh token when it expires. Pass it to
// assinafy.ClientOptions.TokenSource and the SDK authenticates every request
// with a fresh token.
//
// One TokenSource holds one connection, which is one user in one workspace. A
// customer who uses several workspaces connects each one separately, so keep a
// token, a workspace ID and a TokenSource per connection.
//
// A TokenSource is safe for concurrent use and runs one refresh at a time, so
// concurrent callers share one. That covers a single process. When several
// processes or replicas can refresh the same connection, implement
// assinafy.TokenSource over Config.Refresh instead, holding a lock in your token
// store while you re-read the saved token, refresh it and save the result.
type TokenSource struct {
	config *Config
	// save persists a renewed token before it is used. Rotation makes this
	// mandatory rather than advisory.
	save func(context.Context, *Token) error

	mu    sync.Mutex
	token Token
	// unsaved is set while token holds a renewed token save has not stored.
	unsaved bool
	// err stops every later refresh once the refresh token is refused or may
	// have been retired.
	err error
	// inFlight is the save or refresh under way, shared by every caller waiting
	// for it. While it runs, only its goroutine changes token and unsaved.
	inFlight *refreshCall
}

// refreshCall is one run of saving and refreshing; done is closed once its
// outcome is set.
type refreshCall struct {
	done        chan struct{}
	accessToken string
	err         error
}

// NewTokenSource builds a TokenSource for one connection from the token a
// previous Exchange or Refresh produced.
//
// onRefresh is called with the renewed token before that token is handed to any
// caller. Store the new RefreshToken there: every refresh retires the previous
// one, and replaying a retired refresh token disconnects the user entirely. It
// runs on a goroutine of the source's own, without its lock held and never
// concurrently with itself, and finishes even if the caller that started the
// refresh stops waiting. Until it returns, the refresh stays in flight: callers
// wait for it with their own contexts, and no other refresh or save starts.
// onRefresh gets no context, so keep it bounded, or use NewTokenSourceContext.
//
// When onRefresh returns an error, or panics, Token returns that error, or one
// wrapping the panic value, without an access token, and the next call saves
// again before anything else, refreshing included; the source keeps the renewed
// token and never sends the retired one again. Until a save succeeds the
// renewed token exists only in this process: if it exits first, your storage
// still holds the retired token and the user must connect again. Save to
// durable storage, and treat a failed save as an incident. A nil onRefresh is
// allowed only when the token carries no refresh token.
func NewTokenSource(config *Config, token *Token, onRefresh func(*Token) error) *TokenSource {
	var save func(context.Context, *Token) error
	if onRefresh != nil {
		save = func(_ context.Context, renewed *Token) error { return onRefresh(renewed) }
	}
	return NewTokenSourceContext(config, token, save)
}

// NewTokenSourceContext is NewTokenSource with a save callback that receives a
// context. The context ends a minute after the save starts and carries the
// values of the context passed to the Token call that started the refresh. Pass
// it to your storage calls, so that a stalled save gives up and is tried again
// by the next call instead of keeping the refresh in flight.
func NewTokenSourceContext(config *Config, token *Token, save func(context.Context, *Token) error) *TokenSource {
	source := &TokenSource{config: config, save: save}
	if token != nil {
		source.token = *token
	}
	return source
}

// Token returns a valid access token, refreshing first when the current one is
// within a minute of expiring, and saving first a renewed token that is not
// stored yet.
//
// That work runs in the background and does not depend on ctx: the token
// request, and a save made through NewTokenSourceContext, each get a context
// that ends after a minute, and their outcome is kept and handed to later calls
// even when ctx ends first, in which case Token returns ctx's error without
// waiting for it.
//
// It returns ErrNoRefreshToken when the access token has expired and the
// connection has none, which means the user must approve the application again;
// request ScopeOfflineAccess to avoid that. A refresh rejected with
// ErrCodeInvalidGrant means the connection is over: the refresh token was
// already used, 30 days passed without a refresh, or the user reconnected with
// different permissions. Ask the user to connect again.
//
// A refresh whose outcome is unknown returns an error wrapping
// ErrRefreshIndeterminate: the server may have rotated the refresh token this
// source sent, which Current still returns. After that error, or after
// ErrCodeInvalidGrant, the source sends nothing more and every call returns the
// same error. Compare Current().RefreshToken with the one in your storage: build
// a new TokenSource only when storage holds a different one, saved since, and
// otherwise ask the user to connect again rather than send that token again. A
// panic during the token request, in a custom transport for instance, counts as
// an unknown outcome. A refresh that failed before anything was sent, such as a
// DNS, connection or TLS handshake failure, is tried again on the next call.
func (s *TokenSource) Token(ctx context.Context) (string, error) {
	call, accessToken, err := s.start(ctx)
	if call == nil {
		return accessToken, err
	}
	select {
	case <-call.done:
		return call.accessToken, call.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// start returns the access token when nothing needs doing, or else the work in
// flight, starting it when none is.
func (s *TokenSource) start(ctx context.Context) (*refreshCall, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case s.inFlight != nil:
		return s.inFlight, "", nil
	case s.err != nil:
		return nil, "", s.err
	case !s.unsaved && s.token.Valid():
		return nil, s.token.AccessToken, nil
	case !s.unsaved && s.token.RefreshToken == "":
		return nil, "", ErrNoRefreshToken
	case ctx.Err() != nil:
		// A caller that has already stopped waiting starts nothing.
		return nil, "", ctx.Err()
	}
	s.inFlight = &refreshCall{done: make(chan struct{})}
	go s.run(context.WithoutCancel(ctx), s.inFlight, s.token, s.unsaved)
	return s.inFlight, "", nil
}

// run saves a renewed token that is not stored yet, before anything else, so a
// refresh only ever sends the refresh token your storage holds. It then
// refreshes an expiring token and saves the result. It holds no lock while it
// waits on the network or on the save, and finishes whether or not any caller
// still waits for it.
func (s *TokenSource) run(ctx context.Context, call *refreshCall, token Token, unsaved bool) {
	var stop error // set once the refresh token must never be sent again
	requesting := false
	defer func() {
		// A panic in the caller's save or transport must not end the process. One
		// during the token request leaves the refresh's outcome unknown.
		if p := recover(); p != nil {
			cause, ok := p.(error)
			if !ok {
				cause = fmt.Errorf("%v", p)
			}
			call.err = fmt.Errorf("assinafy/oauth: token source panicked: %w", cause)
			if requesting {
				stop = fmt.Errorf("%w: %w", ErrRefreshIndeterminate, call.err)
				call.err = stop
			}
		}
		s.mu.Lock()
		s.token, s.unsaved, s.inFlight = token, unsaved, nil
		if stop != nil {
			s.err = stop
		}
		s.mu.Unlock()
		close(call.done)
	}()

	if unsaved {
		if call.err = s.persist(ctx, token); call.err != nil {
			return
		}
		unsaved = false
	}
	if token.Valid() {
		call.accessToken = token.AccessToken
		return
	}

	requesting = true
	requestCtx, cancel := context.WithTimeout(ctx, stepTimeout)
	defer cancel()
	refreshed, err := s.config.Refresh(requestCtx, token.RefreshToken)
	requesting = false
	if err != nil {
		// A refused refresh token is dead, and one whose refresh may have gone
		// through may be retired; sending either again could end the connection.
		if errors.Is(err, ErrRefreshIndeterminate) || ErrorCode(err) == ErrCodeInvalidGrant {
			stop = err
		}
		call.err = err
		return
	}
	// The renewed token replaces the old one even if saving it fails: the old
	// refresh token is retired, and sending it again would end the connection.
	token, unsaved = *refreshed, true
	if call.err = s.persist(ctx, token); call.err != nil {
		return
	}
	unsaved = false
	call.accessToken = token.AccessToken
}

// persist stores token through save, under its own time limit.
func (s *TokenSource) persist(ctx context.Context, token Token) error {
	if s.save == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, stepTimeout)
	defer cancel()
	if err := s.save(ctx, &token); err != nil {
		return fmt.Errorf("assinafy/oauth: persist refreshed token: %w", err)
	}
	return nil
}

// Current returns a copy of the token the source holds, including any refresh
// performed since it was created. It does not wait: while a refresh or save is
// in flight it returns the token being replaced, so persist the connection with
// Latest, not Current. After a refresh whose outcome is unknown it returns the
// refresh token that was sent.
func (s *TokenSource) Current() Token {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token
}

// Latest waits until no refresh or save is in flight, then returns the token
// the source holds and whether it is saved. Use it to persist the connection,
// for instance on shutdown: stop the calls that use the source first, so no
// refresh starts afterwards, and write the token only when saved is false. Make
// your storage writes ordered as well, for instance by keeping the token with
// the later Expiry, so an older token never overwrites a newer one. It returns
// ctx's error if ctx ends first.
func (s *TokenSource) Latest(ctx context.Context) (token Token, saved bool, err error) {
	for {
		s.mu.Lock()
		call := s.inFlight
		token, saved = s.token, !s.unsaved
		s.mu.Unlock()
		if call == nil {
			return token, saved, nil
		}
		select {
		case <-call.done:
		case <-ctx.Done():
			return Token{}, false, ctx.Err()
		}
	}
}
