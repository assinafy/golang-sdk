package oauth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

// newTestConfig points a Config at srv while keeping the production behaviour
// of every value the flow does not override.
func newTestConfig(srv *httptest.Server) *Config {
	return &Config{
		ClientID:     "client-1",
		ClientSecret: "secret-1",
		RedirectURI:  "https://app.example/oauth/callback",
		Scopes:       []string{ScopeDocumentsRead, ScopeOfflineAccess},
		Endpoint: Endpoint{
			Issuer:           srv.URL,
			AuthorizationURL: srv.URL + "/oauth/authorize",
			TokenURL:         srv.URL + "/v1/oauth/token",
			RevocationURL:    srv.URL + "/v1/oauth/revoke",
			UserInfoURL:      srv.URL + "/v1/oauth/userinfo",
		},
		HTTPClient: srv.Client(),
	}
}

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func TestProductionEndpointAndResolve(t *testing.T) {
	production := ProductionEndpoint()
	if production.Issuer != DefaultIssuer {
		t.Errorf("issuer = %q", production.Issuer)
	}
	if production.TokenURL != DefaultResource+"/v1/oauth/token" {
		t.Errorf("token URL = %q", production.TokenURL)
	}

	// The browser page is served by the authorization server; everything the
	// integration calls is served by the API host.
	if !strings.HasPrefix(production.AuthorizationURL, DefaultIssuer) {
		t.Errorf("authorization URL = %q", production.AuthorizationURL)
	}
	for _, endpoint := range []string{production.TokenURL, production.RevocationURL, production.UserInfoURL} {
		if !strings.HasPrefix(endpoint, DefaultResource) {
			t.Errorf("endpoint %q is not served by the API host", endpoint)
		}
	}

	// Overriding one URL must not blank the others.
	partial := Endpoint{TokenURL: "https://token.example"}.resolve()
	if partial.TokenURL != "https://token.example" {
		t.Errorf("token URL = %q", partial.TokenURL)
	}
	if partial.Issuer != DefaultIssuer || partial.UserInfoURL != production.UserInfoURL {
		t.Errorf("partial = %+v", partial)
	}
}

func TestResourceIndicator(t *testing.T) {
	for _, tc := range []struct{ resource, want string }{
		{"", DefaultResource},
		{"-", ""},
		{"https://other.example", "https://other.example"},
	} {
		if got := (&Config{Resource: tc.resource}).ResourceIndicator(); got != tc.want {
			t.Errorf("ResourceIndicator(%q) = %q, want %q", tc.resource, got, tc.want)
		}
	}
}

func TestHTTPClientDefaultsToATimeout(t *testing.T) {
	if got := (&Config{}).httpClient(); got.Timeout != 30*time.Second {
		t.Errorf("timeout = %v", got.Timeout)
	}
	custom := &http.Client{Timeout: time.Second}
	if got := (&Config{HTTPClient: custom}).httpClient(); got != custom {
		t.Error("configured client was replaced")
	}
}

func TestAuthorizationURL(t *testing.T) {
	cfg := &Config{
		ClientID:    "client-1",
		RedirectURI: "https://app.example/oauth/callback",
		Scopes:      []string{ScopeDocumentsRead},
	}

	raw, err := cfg.AuthorizationURL(AuthorizationRequest{
		State:         "state-1",
		CodeChallenge: "challenge-1",
		Scopes:        []string{ScopeDocumentsWrite, ScopeWebhooksWrite, ScopeOfflineAccess},
		Nonce:         "nonce-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme+"://"+parsed.Host+parsed.Path != DefaultIssuer+"/oauth/authorize" {
		t.Errorf("authorize URL = %q", raw)
	}
	want := map[string]string{
		"response_type":         "code",
		"client_id":             "client-1",
		"redirect_uri":          "https://app.example/oauth/callback",
		"state":                 "state-1",
		"code_challenge":        "challenge-1",
		"code_challenge_method": ChallengeMethod,
		"scope":                 ScopeDocumentsWrite + " " + ScopeWebhooksWrite + " " + ScopeOfflineAccess,
		"nonce":                 "nonce-1",
		"resource":              DefaultResource,
	}
	for key, value := range want {
		if got := parsed.Query().Get(key); got != value {
			t.Errorf("%s = %q, want %q", key, got, value)
		}
	}

	// Without per-request scopes the configured ones are requested.
	raw, err = cfg.AuthorizationURL(AuthorizationRequest{State: "s", CodeChallenge: "c"})
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ = url.Parse(raw)
	if got := parsed.Query().Get("scope"); got != ScopeDocumentsRead {
		t.Errorf("scope = %q", got)
	}

	// Suppressing the resource indicator drops the parameter entirely.
	noResource := *cfg
	noResource.Resource = "-"
	raw, err = noResource.AuthorizationURL(AuthorizationRequest{State: "s", CodeChallenge: "c"})
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ = url.Parse(raw)
	if _, ok := parsed.Query()["resource"]; ok {
		t.Errorf("resource was sent: %q", raw)
	}
}

func TestAuthorizationURLRejectsIncompleteRequests(t *testing.T) {
	full := Config{ClientID: "c", RedirectURI: "https://app.example/cb"}
	request := AuthorizationRequest{State: "s", CodeChallenge: "c"}

	for _, tc := range []struct {
		name string
		cfg  Config
		req  AuthorizationRequest
		want error
	}{
		{"no client id", Config{RedirectURI: full.RedirectURI}, request, ErrMissingClientID},
		{"no redirect uri", Config{ClientID: "c"}, request, ErrMissingRedirectURI},
		{"no state", full, AuthorizationRequest{CodeChallenge: "c"}, ErrMissingState},
		{"no challenge", full, AuthorizationRequest{State: "s"}, ErrMissingCodeChallenge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.cfg.AuthorizationURL(tc.req); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}

	broken := Config{ClientID: "c", RedirectURI: "https://app.example/cb", Endpoint: Endpoint{AuthorizationURL: "://"}}
	if _, err := broken.AuthorizationURL(request); err == nil {
		t.Fatal("expected an error for an unparsable authorization URL")
	}
}

func TestParseCallback(t *testing.T) {
	cfg := &Config{Endpoint: Endpoint{Issuer: DefaultIssuer}}

	callback := func(query string) *url.URL {
		u, err := url.Parse("https://app.example/oauth/callback?" + query)
		if err != nil {
			t.Fatal(err)
		}
		return u
	}

	code, err := cfg.ParseCallback(callback("code=abc&state=s1&iss="+url.QueryEscape(DefaultIssuer)), "s1")
	if err != nil || code != "abc" {
		t.Fatalf("code = %q, err = %v", code, err)
	}

	// An absent iss is tolerated; a wrong one is not.
	if _, err := cfg.ParseCallback(callback("code=abc&state=s1"), "s1"); err != nil {
		t.Fatalf("missing iss: %v", err)
	}
	if _, err := cfg.ParseCallback(callback("code=abc&state=s1&iss=https://evil.example"), "s1"); !errors.Is(err, ErrIssuerMismatch) {
		t.Fatalf("err = %v, want ErrIssuerMismatch", err)
	}

	for _, tc := range []struct {
		name  string
		query string
		state string
		want  error
	}{
		{"state mismatch", "code=abc&state=other", "s1", ErrStateMismatch},
		{"empty expected state", "code=abc&state=", "", ErrStateMismatch},
		{"no code", "state=s1", "s1", ErrNoCode},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := cfg.ParseCallback(callback(tc.query), tc.state); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}

	if _, err := cfg.ParseCallback(nil, "s1"); !errors.Is(err, ErrNoCode) {
		t.Fatalf("nil URL err = %v", err)
	}

	// A declined approval arrives as an OAuth error on the redirect URI.
	_, err = cfg.ParseCallback(callback("error=access_denied&error_description=no&state=s1"), "s1")
	if ErrorCode(err) != ErrCodeAccessDenied {
		t.Fatalf("err = %v", err)
	}
}

func TestTokenValidityAndScopes(t *testing.T) {
	var nilToken *Token
	if nilToken.Valid() || nilToken.Scopes() != nil || nilToken.HasScope(ScopeOpenID) {
		t.Error("nil token reported as usable")
	}
	if (&Token{}).Valid() {
		t.Error("token without an access token reported as valid")
	}
	// An unknown lifetime is treated as valid; the API is the real authority.
	if !(&Token{AccessToken: "a"}).Valid() {
		t.Error("token with zero expiry reported as invalid")
	}
	if (&Token{AccessToken: "a", Expiry: time.Now().Add(30 * time.Second)}).Valid() {
		t.Error("token inside the refresh leeway reported as valid")
	}
	if !(&Token{AccessToken: "a", Expiry: time.Now().Add(time.Hour)}).Valid() {
		t.Error("token valid for an hour reported as invalid")
	}

	token := &Token{Scope: ScopeDocumentsRead + " " + ScopeDocumentsWrite}
	if got := token.Scopes(); !reflect.DeepEqual(got, []string{ScopeDocumentsRead, ScopeDocumentsWrite}) {
		t.Errorf("scopes = %v", got)
	}
	if !token.HasScope(ScopeDocumentsWrite) || token.HasScope(ScopeTemplatesWrite) {
		t.Errorf("HasScope is wrong for %v", token.Scopes())
	}
	if got := (&Token{Scope: "  "}).Scopes(); got != nil {
		t.Errorf("blank scope = %v", got)
	}
}

func TestExchange(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/oauth/token" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("content type = %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		want := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {"the-code"},
			"code_verifier": {"the-verifier"},
			"redirect_uri":  {"https://app.example/oauth/callback"},
			"client_id":     {"client-1"},
			"client_secret": {"secret-1"},
			"resource":      {DefaultResource},
		}
		if !reflect.DeepEqual(r.PostForm, want) {
			t.Errorf("form = %v, want %v", r.PostForm, want)
		}
		// Token responses are plain OAuth JSON, not the API's envelope.
		_, _ = io.WriteString(w, `{"access_token":"at","token_type":"Bearer","expires_in":3600,`+
			`"refresh_token":"rt","scope":"documents:read","id_token":"idt"}`)
	})

	token, err := newTestConfig(srv).Exchange(context.Background(), "the-code", "the-verifier")
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "at" || token.RefreshToken != "rt" || token.IDToken != "idt" {
		t.Fatalf("token = %+v", token)
	}
	if !token.Valid() || time.Until(token.Expiry) > time.Hour {
		t.Fatalf("expiry = %v", token.Expiry)
	}
}

func TestExchangeAndRefreshRejectEmptyGrants(t *testing.T) {
	cfg := &Config{ClientID: "c"}
	if _, err := cfg.Exchange(context.Background(), "", "v"); !errors.Is(err, ErrNoCode) {
		t.Errorf("Exchange err = %v", err)
	}
	if _, err := cfg.Refresh(context.Background(), ""); !errors.Is(err, ErrNoRefreshToken) {
		t.Errorf("Refresh err = %v", err)
	}
	if _, err := (&Config{}).Exchange(context.Background(), "code", "v"); !errors.Is(err, ErrMissingClientID) {
		t.Errorf("no client id err = %v", err)
	}
}

func TestRefreshSendsOnlyTheRefreshGrant(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.PostForm.Get("grant_type") != "refresh_token" || r.PostForm.Get("refresh_token") != "old-rt" {
			t.Errorf("form = %v", r.PostForm)
		}
		if _, ok := r.PostForm["redirect_uri"]; ok {
			t.Errorf("refresh sent redirect_uri: %v", r.PostForm)
		}
		_, _ = io.WriteString(w, `{"access_token":"at2","token_type":"Bearer","expires_in":3600,"refresh_token":"new-rt"}`)
	})

	token, err := newTestConfig(srv).Refresh(context.Background(), "old-rt")
	if err != nil {
		t.Fatal(err)
	}
	if token.RefreshToken != "new-rt" {
		t.Fatalf("token = %+v", token)
	}
}

func TestPublicClientOmitsTheSecret(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if _, ok := r.PostForm["client_secret"]; ok {
			t.Errorf("public client sent a secret: %v", r.PostForm)
		}
		_, _ = io.WriteString(w, `{"access_token":"at","token_type":"Bearer"}`)
	})

	cfg := newTestConfig(srv)
	cfg.ClientSecret = ""
	token, err := cfg.Exchange(context.Background(), "code", "verifier")
	if err != nil {
		t.Fatal(err)
	}
	// No expires_in means an unknown lifetime rather than an expired token.
	if !token.Expiry.IsZero() || !token.Valid() {
		t.Fatalf("token = %+v", token)
	}
}

func TestTokenEndpointErrors(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"invalid_grant","error_description":"Code expired."}`)
	})

	_, err := newTestConfig(srv).Exchange(context.Background(), "code", "verifier")
	if ErrorCode(err) != ErrCodeInvalidGrant {
		t.Fatalf("err = %v", err)
	}
	var oauthErr *Error
	if !errors.As(err, &oauthErr) || oauthErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("err = %#v", err)
	}
	if !strings.Contains(err.Error(), "Code expired.") {
		t.Errorf("message = %q", err.Error())
	}
}

func TestRevoke(t *testing.T) {
	var form url.Values
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/oauth/revoke" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		form = r.PostForm
		w.WriteHeader(http.StatusOK)
	})

	cfg := newTestConfig(srv)
	if err := cfg.Revoke(context.Background(), "rt", HintRefreshToken); err != nil {
		t.Fatal(err)
	}
	if form.Get("token") != "rt" || form.Get("token_type_hint") != HintRefreshToken || form.Get("client_id") != "client-1" {
		t.Fatalf("form = %v", form)
	}
	// The resource indicator is a token-request parameter, not a revocation one.
	if _, ok := form["resource"]; ok {
		t.Errorf("revocation sent a resource indicator: %v", form)
	}

	if err := cfg.Revoke(context.Background(), "rt", ""); err != nil {
		t.Fatal(err)
	}
	if _, ok := form["token_type_hint"]; ok {
		t.Errorf("empty hint was sent: %v", form)
	}

	if err := cfg.Revoke(context.Background(), "", HintAccessToken); !errors.Is(err, ErrMissingToken) {
		t.Errorf("err = %v", err)
	}
	if err := (&Config{}).Revoke(context.Background(), "rt", ""); !errors.Is(err, ErrMissingClientID) {
		t.Errorf("err = %v", err)
	}
}

func TestUserInfo(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/oauth/userinfo" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer at" {
			t.Errorf("authorization = %q", got)
		}
		// Claims are returned flat, per OpenID Connect Core 5.3.2.
		_, _ = io.WriteString(w, `{"sub":"u1","name":"Maria Silva","email":"maria@example.test","email_verified":true}`)
	})

	info, err := newTestConfig(srv).UserInfo(context.Background(), "at")
	if err != nil {
		t.Fatal(err)
	}
	if info.Sub != "u1" || info.Name == nil || *info.Name != "Maria Silva" {
		t.Fatalf("info = %+v", info)
	}
	if info.EmailVerified == nil || !*info.EmailVerified {
		t.Fatalf("email_verified = %v", info.EmailVerified)
	}

	if _, err := newTestConfig(srv).UserInfo(context.Background(), ""); !errors.Is(err, ErrMissingToken) {
		t.Errorf("err = %v", err)
	}
}

func TestUserInfoSurfacesTheAPIEnvelopeError(t *testing.T) {
	// An expired token is refused before the OAuth handler runs, so the body is
	// the API's own envelope rather than a flat OAuth error.
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"status":401,"data":null,"message":"Credenciais inválidas."}`)
	})

	_, err := newTestConfig(srv).UserInfo(context.Background(), "expired")
	var oauthErr *Error
	if !errors.As(err, &oauthErr) {
		t.Fatalf("err = %#v", err)
	}
	if oauthErr.StatusCode != http.StatusUnauthorized || oauthErr.Description != "Credenciais inválidas." {
		t.Fatalf("err = %#v", oauthErr)
	}
}

func TestErrorFormattingAndFallbacks(t *testing.T) {
	bare := &Error{Code: ErrCodeInvalidClient, StatusCode: http.StatusUnauthorized}
	if got := bare.Error(); !strings.Contains(got, ErrCodeInvalidClient) || !strings.Contains(got, "401") {
		t.Errorf("Error() = %q", got)
	}
	if ErrorCode(errors.New("other")) != "" {
		t.Error("non-OAuth error reported a code")
	}

	// A body that is neither an OAuth error nor an envelope is still reported.
	err := newResponseError(http.StatusBadGateway, []byte("upstream down"))
	var oauthErr *Error
	if !errors.As(err, &oauthErr) || oauthErr.Description != "upstream down" {
		t.Fatalf("err = %#v", err)
	}
	// So is an empty one.
	err = newResponseError(http.StatusServiceUnavailable, nil)
	if !errors.As(err, &oauthErr) || oauthErr.Description != http.StatusText(http.StatusServiceUnavailable) {
		t.Fatalf("err = %#v", err)
	}
}

func TestTransportErrorsHideTheQueryString(t *testing.T) {
	cfg := &Config{
		ClientID: "c",
		Endpoint: Endpoint{TokenURL: "http://127.0.0.1:1/v1/oauth/token?signer-access-code=secret"},
	}
	_, err := cfg.Exchange(context.Background(), "code", "verifier")
	if err == nil {
		t.Fatal("expected a transport error")
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("error leaked the query string: %v", err)
	}

	if got := redactQuery(nil); got != "" {
		t.Errorf("redactQuery(nil) = %q", got)
	}
}

func TestMalformedSuccessBodyIsReported(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "not json")
	})
	if _, err := newTestConfig(srv).Exchange(context.Background(), "code", "verifier"); err == nil {
		t.Fatal("expected a decoding error")
	}
}

func TestRequestConstructionErrorsAreWrapped(t *testing.T) {
	cfg := &Config{ClientID: "c", Endpoint: Endpoint{TokenURL: "://", UserInfoURL: "://"}}
	if _, err := cfg.Exchange(context.Background(), "code", "v"); err == nil {
		t.Error("expected an error for an unparsable token URL")
	}
	if _, err := cfg.UserInfo(context.Background(), "at"); err == nil {
		t.Error("expected an error for an unparsable userinfo URL")
	}
}
