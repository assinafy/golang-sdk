package oauth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestTokenSourceReusesAValidToken(t *testing.T) {
	srv := newTestServer(t, func(http.ResponseWriter, *http.Request) {
		t.Error("a valid token was refreshed")
	})

	source := NewTokenSource(newTestConfig(srv), &Token{
		AccessToken:  "at",
		RefreshToken: "rt",
		Expiry:       time.Now().Add(time.Hour),
	}, nil)

	token, err := source.Token(context.Background())
	if err != nil || token != "at" {
		t.Fatalf("token = %q, err = %v", token, err)
	}
	if got := source.Current(); got.AccessToken != "at" {
		t.Fatalf("current = %+v", got)
	}
}

func TestTokenSourceRefreshesAndPersistsBeforeUse(t *testing.T) {
	var refreshes int
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		refreshes++
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.PostForm.Get("refresh_token"); got != "rt-1" {
			t.Errorf("refresh_token = %q", got)
		}
		_, _ = io.WriteString(w, `{"access_token":"at-2","token_type":"Bearer","expires_in":3600,"refresh_token":"rt-2"}`)
	})

	var saved *Token
	source := NewTokenSource(newTestConfig(srv), &Token{
		AccessToken:  "at-1",
		RefreshToken: "rt-1",
		Expiry:       time.Now().Add(-time.Minute),
	}, func(token *Token) error {
		// Rotation makes saving mandatory: the new refresh token must be stored
		// before the access token it came with is used.
		saved = token
		return nil
	})

	token, err := source.Token(context.Background())
	if err != nil || token != "at-2" {
		t.Fatalf("token = %q, err = %v", token, err)
	}
	if saved == nil || saved.RefreshToken != "rt-2" {
		t.Fatalf("saved = %+v", saved)
	}
	if got := source.Current(); got.RefreshToken != "rt-2" {
		t.Fatalf("current = %+v", got)
	}

	// The renewed token is now valid, so a second call must not refresh again.
	if _, err := source.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if refreshes != 1 {
		t.Fatalf("refreshes = %d, want 1", refreshes)
	}
}

func TestTokenSourceCarriesTheRefreshTokenForward(t *testing.T) {
	// A response that omits refresh_token leaves the current one in force; losing
	// it would strand the connection.
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"access_token":"at-2","token_type":"Bearer","expires_in":3600}`)
	})

	source := NewTokenSource(newTestConfig(srv), &Token{
		AccessToken:  "at-1",
		RefreshToken: "rt-1",
		Expiry:       time.Now().Add(-time.Minute),
	}, nil)

	if _, err := source.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := source.Current(); got.RefreshToken != "rt-1" {
		t.Fatalf("current = %+v", got)
	}
}

func TestTokenSourceAbortsWhenTheTokenCannotBeSaved(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"access_token":"at-2","token_type":"Bearer","expires_in":3600,"refresh_token":"rt-2"}`)
	})

	saveErr := errors.New("database unavailable")
	source := NewTokenSource(newTestConfig(srv), &Token{
		AccessToken:  "at-1",
		RefreshToken: "rt-1",
		Expiry:       time.Now().Add(-time.Minute),
	}, func(*Token) error { return saveErr })

	if _, err := source.Token(context.Background()); !errors.Is(err, saveErr) {
		t.Fatalf("err = %v", err)
	}
	// The unsaved token must not become the source's state, or the stored
	// refresh token and the live one would disagree.
	if got := source.Current(); got.RefreshToken != "rt-1" || got.AccessToken != "at-1" {
		t.Fatalf("current = %+v", got)
	}
}

func TestTokenSourceWithoutARefreshToken(t *testing.T) {
	source := NewTokenSource(&Config{ClientID: "c"}, &Token{
		AccessToken: "at",
		Expiry:      time.Now().Add(-time.Minute),
	}, nil)

	if _, err := source.Token(context.Background()); !errors.Is(err, ErrNoRefreshToken) {
		t.Fatalf("err = %v", err)
	}

	// A source built with no token at all reports the same thing.
	empty := NewTokenSource(&Config{ClientID: "c"}, nil, nil)
	if _, err := empty.Token(context.Background()); !errors.Is(err, ErrNoRefreshToken) {
		t.Fatalf("err = %v", err)
	}
	if got := empty.Current(); got != (Token{}) {
		t.Fatalf("current = %+v", got)
	}
}

func TestTokenSourceSurfacesARejectedRefresh(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"invalid_grant","error_description":"Refresh token already used."}`)
	})

	source := NewTokenSource(newTestConfig(srv), &Token{
		AccessToken:  "at",
		RefreshToken: "rt",
		Expiry:       time.Now().Add(-time.Minute),
	}, nil)

	_, err := source.Token(context.Background())
	if ErrorCode(err) != ErrCodeInvalidGrant {
		t.Fatalf("err = %v", err)
	}
}

func TestTokenSourceRefreshesOnceUnderConcurrency(t *testing.T) {
	var mu sync.Mutex
	var refreshes int
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		refreshes++
		mu.Unlock()
		_, _ = io.WriteString(w, `{"access_token":"at-2","token_type":"Bearer","expires_in":3600,"refresh_token":"rt-2"}`)
	})

	source := NewTokenSource(newTestConfig(srv), &Token{
		AccessToken:  "at-1",
		RefreshToken: "rt-1",
		Expiry:       time.Now().Add(-time.Minute),
	}, nil)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := source.Token(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	// The API ends a connection whose refresh token is replayed, so concurrent
	// callers must share one refresh rather than race.
	mu.Lock()
	defer mu.Unlock()
	if refreshes != 1 {
		t.Fatalf("refreshes = %d, want 1", refreshes)
	}
}
