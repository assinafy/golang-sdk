package oauth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
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

func TestRefreshWithoutANewRefreshTokenIsRejected(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"missing", `{"access_token":"at-2","token_type":"Bearer","expires_in":3600}`},
		{"null", `{"access_token":"at-2","token_type":"Bearer","expires_in":3600,"refresh_token":null}`},
		{"empty", `{"access_token":"at-2","token_type":"Bearer","expires_in":3600,"refresh_token":""}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				_, _ = io.WriteString(w, tc.body)
			})
			cfg := newTestConfig(srv)

			// Every refresh rotates the token, so a success without a new one may
			// have retired the token sent: it is neither returned nor kept.
			if token, err := cfg.Refresh(context.Background(), "rt-1"); !errors.Is(err, ErrRefreshIndeterminate) || token != nil {
				t.Fatalf("token = %+v, err = %v", token, err)
			}

			source := NewTokenSource(cfg, &Token{
				AccessToken:  "at-1",
				RefreshToken: "rt-1",
				Expiry:       time.Now().Add(-time.Minute),
			}, func(*Token) error {
				t.Error("a response without a refresh token was saved")
				return nil
			})
			for range 2 {
				if _, err := source.Token(context.Background()); !errors.Is(err, ErrRefreshIndeterminate) {
					t.Fatalf("err = %v", err)
				}
			}
			// One request from Refresh, one from the source's first call only.
			if got := requests.Load(); got != 2 {
				t.Fatalf("requests = %d, want 2", got)
			}
		})
	}
}

func TestTokenSourceStopsAfterARefreshWithAnUnknownOutcome(t *testing.T) {
	for _, tc := range []struct {
		name    string
		timeout time.Duration
		handle  func(http.ResponseWriter, *http.Request)
	}{
		{"connection dropped after the request was read", 0, func(w http.ResponseWriter, r *http.Request) {
			_ = r.ParseForm()
			if conn, _, err := http.NewResponseController(w).Hijack(); err == nil {
				_ = conn.Close()
			}
		}},
		{"answer slower than the client timeout", 50 * time.Millisecond, func(_ http.ResponseWriter, r *http.Request) {
			// The server notices the client leaving only once the body is read.
			_ = r.ParseForm()
			<-r.Context().Done()
		}},
		{"unreadable success body", 0, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "not json")
		}},
		{"gateway timeout", 0, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusGatewayTimeout)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				tc.handle(w, r)
			})
			cfg := newTestConfig(srv)
			cfg.HTTPClient.Timeout = tc.timeout

			source := NewTokenSource(cfg, &Token{
				AccessToken:  "at-1",
				RefreshToken: "rt-1",
				Expiry:       time.Now().Add(-time.Minute),
			}, nil)

			// The server may have rotated rt-1, so neither the callers waiting on
			// the first refresh nor any later call may send it again.
			var wg sync.WaitGroup
			for range 4 {
				wg.Go(func() {
					if _, err := source.Token(context.Background()); !errors.Is(err, ErrRefreshIndeterminate) {
						t.Errorf("err = %v", err)
					}
				})
			}
			wg.Wait()
			if _, err := source.Token(context.Background()); !errors.Is(err, ErrRefreshIndeterminate) {
				t.Fatalf("err = %v", err)
			}
			if got := requests.Load(); got != 1 {
				t.Fatalf("requests = %d, want 1", got)
			}
			// Current names the token that must not be sent again, to compare with
			// the one in storage.
			if got := source.Current().RefreshToken; got != "rt-1" {
				t.Fatalf("current refresh token = %q", got)
			}
		})
	}
}

func TestTokenSourceFinishesARefreshItsCallerStoppedWaitingFor(t *testing.T) {
	var requests atomic.Int32
	answer := make(chan struct{})
	release := sync.OnceFunc(func() { close(answer) })
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_ = r.ParseForm()
		select {
		case <-answer:
		case <-r.Context().Done():
			return
		}
		_, _ = io.WriteString(w, `{"access_token":"at-2","token_type":"Bearer","expires_in":3600,"refresh_token":"rt-2"}`)
	})
	// Registered after the server, so it runs first and unblocks the handler.
	t.Cleanup(release)
	saved := make(chan string, 1)
	source := NewTokenSource(newTestConfig(srv), &Token{
		AccessToken:  "at-1",
		RefreshToken: "rt-1",
		Expiry:       time.Now().Add(-time.Minute),
	}, func(token *Token) error {
		saved <- token.RefreshToken
		return nil
	})

	// The API call that needed the token is canceled once its refresh is on the
	// wire. It stops waiting, but the refresh is not abandoned.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for requests.Load() == 0 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()
	if _, err := source.Token(ctx); !errors.Is(err, context.Canceled) || errors.Is(err, ErrRefreshIndeterminate) {
		t.Fatalf("err = %v", err)
	}

	waiter := make(chan string)
	go func() {
		token, _ := source.Token(context.Background())
		waiter <- token
	}()
	release()
	select {
	case got := <-saved:
		if got != "rt-2" {
			t.Fatalf("saved %q", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the refresh was not saved")
	}
	if got := <-waiter; got != "at-2" {
		t.Fatalf("waiting caller got %q", got)
	}
	if token, err := source.Token(context.Background()); err != nil || token != "at-2" {
		t.Fatalf("token = %q, err = %v", token, err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestTokenSourceRecoversFromAPanickingSave(t *testing.T) {
	var requests atomic.Int32
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = io.WriteString(w, `{"access_token":"at-2","token_type":"Bearer","expires_in":3600,"refresh_token":"rt-2"}`)
	})
	driverBug := errors.New("storage driver bug")
	var saves atomic.Int32
	source := NewTokenSource(newTestConfig(srv), &Token{
		AccessToken:  "at-1",
		RefreshToken: "rt-1",
		Expiry:       time.Now().Add(-time.Minute),
	}, func(*Token) error {
		if saves.Add(1) == 1 {
			panic(driverBug)
		}
		return nil
	})

	// The panic comes back as an error rather than ending the process, and the
	// renewed token is kept, unsaved, rather than lost.
	if _, err := source.Token(context.Background()); !errors.Is(err, driverBug) {
		t.Fatalf("err = %v", err)
	}
	if token, saved, err := source.Latest(context.Background()); err != nil || saved || token.RefreshToken != "rt-2" {
		t.Fatalf("latest = %+v, saved = %v, err = %v", token, saved, err)
	}
	if token, err := source.Token(context.Background()); err != nil || token != "at-2" {
		t.Fatalf("token = %q, err = %v", token, err)
	}
	if token, saved, err := source.Latest(context.Background()); err != nil || !saved || token.RefreshToken != "rt-2" {
		t.Fatalf("latest = %+v, saved = %v, err = %v", token, saved, err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestTokenSourceStopsAfterAPanicDuringTheTokenRequest(t *testing.T) {
	var requests atomic.Int32
	cfg := &Config{ClientID: "c", HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests.Add(1)
		panic("transport bug")
	})}}
	source := NewTokenSource(cfg, &Token{
		AccessToken:  "at-1",
		RefreshToken: "rt-1",
		Expiry:       time.Now().Add(-time.Minute),
	}, nil)

	// The panic may have come after the request was sent, so the outcome is
	// unknown and rt-1 is not sent again.
	for range 2 {
		if _, err := source.Token(context.Background()); !errors.Is(err, ErrRefreshIndeterminate) || !strings.Contains(err.Error(), "transport bug") {
			t.Fatalf("err = %v", err)
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestTokenSourceCallersDoNotWaitOnAStalledSave(t *testing.T) {
	var requests atomic.Int32
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = io.WriteString(w, `{"access_token":"at-2","token_type":"Bearer","expires_in":3600,"refresh_token":"rt-2"}`)
	})
	saving, release := make(chan struct{}), make(chan struct{})
	releaseSave := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseSave)
	source := NewTokenSourceContext(newTestConfig(srv), &Token{
		AccessToken:  "at-1",
		RefreshToken: "rt-1",
		Expiry:       time.Now().Add(-time.Minute),
	}, func(ctx context.Context, _ *Token) error {
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > time.Minute {
			t.Errorf("save context deadline = %v, %v", deadline, ok)
		}
		close(saving)
		<-release
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan error, 1)
	go func() {
		_, err := source.Token(ctx)
		started <- err
	}()
	<-saving
	cancel()

	// While the save stalls, every caller stops waiting on its own context, and
	// Current answers at once.
	returns := func(f func()) bool {
		done := make(chan struct{})
		go func() {
			f()
			close(done)
		}()
		select {
		case <-done:
			return true
		case <-time.After(2 * time.Second):
			return false
		}
	}
	if !returns(func() {
		if err := <-started; !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v", err)
		}
	}) {
		t.Fatal("the canceled caller waited on the save")
	}
	short, cancelShort := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancelShort()
	if !returns(func() {
		if _, err := source.Token(short); !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("err = %v", err)
		}
	}) {
		t.Fatal("a later caller waited on the save past its own deadline")
	}
	if !returns(func() { _ = source.Current() }) {
		t.Fatal("Current waited on the save")
	}

	releaseSave()
	if token, err := source.Token(context.Background()); err != nil || token != "at-2" {
		t.Fatalf("token = %q, err = %v", token, err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestTokenSourceLatestWaitsForTheRefreshInFlight(t *testing.T) {
	var requests atomic.Int32
	answer := make(chan struct{})
	release := sync.OnceFunc(func() { close(answer) })
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_ = r.ParseForm()
		select {
		case <-answer:
		case <-r.Context().Done():
			return
		}
		_, _ = io.WriteString(w, `{"access_token":"at-2","token_type":"Bearer","expires_in":3600,"refresh_token":"rt-2"}`)
	})
	// Registered after the server, so it runs first and unblocks the handler.
	t.Cleanup(release)
	source := NewTokenSource(newTestConfig(srv), &Token{
		AccessToken:  "at-1",
		RefreshToken: "rt-1",
		Expiry:       time.Now().Add(-time.Minute),
	}, func(*Token) error { return nil })

	go func() { _, _ = source.Token(context.Background()) }()
	for requests.Load() == 0 {
		time.Sleep(time.Millisecond)
	}

	// Current does not wait, so it still holds the token being replaced, which
	// must not be persisted; Latest waits for the refresh instead.
	if got := source.Current().RefreshToken; got != "rt-1" {
		t.Fatalf("current refresh token = %q", got)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := source.Latest(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}

	type snapshot struct {
		token Token
		saved bool
		err   error
	}
	latest := make(chan snapshot, 1)
	go func() {
		token, saved, err := source.Latest(context.Background())
		latest <- snapshot{token, saved, err}
	}()
	release()
	select {
	case got := <-latest:
		if got.err != nil || !got.saved || got.token.RefreshToken != "rt-2" {
			t.Fatalf("latest = %+v", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Latest did not return after the refresh")
	}
}

func TestTokenSourceSavesARenewedTokenBeforeRefreshingIt(t *testing.T) {
	var mu sync.Mutex
	var events []string
	record := func(event string) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
	}
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		sent := r.PostForm.Get("refresh_token")
		record("send " + sent)
		next := map[string]string{"rt-1": "2", "rt-2": "3"}[sent]
		// The renewed access token is already inside the refresh window, so the
		// next call has to refresh again.
		_, _ = io.WriteString(w, `{"access_token":"at-`+next+`","token_type":"Bearer","expires_in":30,"refresh_token":"rt-`+next+`"}`)
	})
	var failSave atomic.Bool
	failSave.Store(true)
	source := NewTokenSource(newTestConfig(srv), &Token{
		AccessToken:  "at-1",
		RefreshToken: "rt-1",
		Expiry:       time.Now().Add(-time.Minute),
	}, func(token *Token) error {
		if failSave.Load() {
			record("save failed " + token.RefreshToken)
			return errors.New("database unavailable")
		}
		record("save " + token.RefreshToken)
		return nil
	})

	for range 2 {
		if token, err := source.Token(context.Background()); err == nil {
			t.Fatalf("token %q handed out before it was saved", token)
		}
	}
	failSave.Store(false)
	if token, err := source.Token(context.Background()); err != nil || token != "at-3" {
		t.Fatalf("token = %q, err = %v", token, err)
	}

	// rt-2 is stored before it is sent, so storage never falls behind the
	// refresh token a refresh sends.
	mu.Lock()
	defer mu.Unlock()
	want := []string{"send rt-1", "save failed rt-2", "save failed rt-2", "save rt-2", "send rt-2", "save rt-3"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
}

func TestTokenSourceRetriesARefreshThatWasNeverSent(t *testing.T) {
	var requests atomic.Int32
	handler := func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = io.WriteString(w, `{"access_token":"at-2","token_type":"Bearer","expires_in":3600,"refresh_token":"rt-2"}`)
	}
	srv := newTestServer(t, handler)
	untrusted := httptest.NewTLSServer(http.HandlerFunc(handler))
	t.Cleanup(untrusted.Close)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	for _, tc := range []struct {
		name string
		ctx  context.Context
		fail func(*Config)
	}{
		{"connection refused", context.Background(), func(cfg *Config) {
			cfg.Endpoint.TokenURL = "http://127.0.0.1:1/v1/oauth/token"
		}},
		{"untrusted certificate", context.Background(), func(cfg *Config) {
			cfg.Endpoint.TokenURL = untrusted.URL + "/v1/oauth/token"
			cfg.HTTPClient = &http.Client{}
		}},
		{"context already canceled", canceled, func(*Config) {}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests.Store(0)
			cfg := newTestConfig(srv)
			working := *cfg
			tc.fail(cfg)
			source := NewTokenSource(cfg, &Token{
				AccessToken:  "at-1",
				RefreshToken: "rt-1",
				Expiry:       time.Now().Add(-time.Minute),
			}, nil)

			// Nothing reached the server, so rt-1 is still current and the next
			// call may send it.
			if _, err := source.Token(tc.ctx); err == nil || errors.Is(err, ErrRefreshIndeterminate) {
				t.Fatalf("err = %v", err)
			}
			*cfg = working
			if token, err := source.Token(context.Background()); err != nil || token != "at-2" {
				t.Fatalf("token = %q, err = %v", token, err)
			}
			if got := requests.Load(); got != 1 {
				t.Fatalf("requests = %d, want 1", got)
			}
		})
	}
}

func TestTokenSourceRetriesTheSaveNotTheRefresh(t *testing.T) {
	var refreshes int
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		refreshes++
		_, _ = io.WriteString(w, `{"access_token":"at-2","token_type":"Bearer","expires_in":3600,"refresh_token":"rt-2"}`)
	})

	saveErr := errors.New("database unavailable")
	var saved []string
	source := NewTokenSource(newTestConfig(srv), &Token{
		AccessToken:  "at-1",
		RefreshToken: "rt-1",
		Expiry:       time.Now().Add(-time.Minute),
	}, func(token *Token) error {
		saved = append(saved, token.RefreshToken)
		if len(saved) == 1 {
			return saveErr
		}
		return nil
	})

	// A failed save hands out no access token, but the source keeps the rotated
	// token: rt-1 is retired, and sending it again would end the connection.
	if token, err := source.Token(context.Background()); !errors.Is(err, saveErr) || token != "" {
		t.Fatalf("token = %q, err = %v", token, err)
	}
	if got := source.Current(); got.RefreshToken != "rt-2" || got.AccessToken != "at-2" {
		t.Fatalf("current = %+v", got)
	}

	token, err := source.Token(context.Background())
	if err != nil || token != "at-2" {
		t.Fatalf("token = %q, err = %v", token, err)
	}
	if _, err := source.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if refreshes != 1 || !reflect.DeepEqual(saved, []string{"rt-2", "rt-2"}) {
		t.Fatalf("refreshes = %d, saved = %v", refreshes, saved)
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
	for _, tc := range []struct {
		code         string
		status       int
		wantRequests int32
	}{
		// A refused refresh token is dead, so it is not sent again.
		{ErrCodeInvalidGrant, http.StatusBadRequest, 1},
		// Any other refusal left the token untouched, so the next call retries.
		{ErrCodeInvalidClient, http.StatusUnauthorized, 2},
	} {
		t.Run(tc.code, func(t *testing.T) {
			var requests atomic.Int32
			srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, `{"error":"`+tc.code+`"}`)
			})

			source := NewTokenSource(newTestConfig(srv), &Token{
				AccessToken:  "at",
				RefreshToken: "rt",
				Expiry:       time.Now().Add(-time.Minute),
			}, nil)

			for range 2 {
				if _, err := source.Token(context.Background()); ErrorCode(err) != tc.code || errors.Is(err, ErrRefreshIndeterminate) {
					t.Fatalf("err = %v", err)
				}
			}
			if got := requests.Load(); got != tc.wantRequests {
				t.Fatalf("requests = %d, want %d", got, tc.wantRequests)
			}
		})
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
