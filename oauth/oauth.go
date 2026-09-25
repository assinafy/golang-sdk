// Package oauth implements the Assinafy OAuth 2.1 authorization-code flow with
// mandatory PKCE, for applications that act in other people's workspaces with
// those people's permission.
//
// Use it instead of an API key when you build a product that many Assinafy
// customers connect to their own workspace: the user approves your application
// once and you receive a token limited to the scopes they approved and to the
// single workspace they chose. To automate your own workspace, keep using
// ClientOptions.APIKey and ignore this package.
//
// The flow has four steps. Step two happens in the user's browser; every other
// step runs on your server:
//
//  1. Create a PKCE pair and a state value, store both in the user's session,
//     and build the authorization URL.
//  2. Send the browser to that URL. The user signs in, picks one workspace, and
//     approves the application.
//  3. The browser returns to your redirect URI with a one-time code. Validate
//     the callback and exchange the code for a token.
//  4. Call the API with that token, refreshing it as it expires.
//
// A complete server-side integration:
//
//	cfg := &oauth.Config{
//	    ClientID:     os.Getenv("ASSINAFY_CLIENT_ID"),
//	    ClientSecret: os.Getenv("ASSINAFY_CLIENT_SECRET"),
//	    RedirectURI:  "https://myapp.example/oauth/callback",
//	    Scopes:       []string{oauth.ScopeDocumentsRead, oauth.ScopeOfflineAccess},
//	}
//
//	// Step 1: start the connection.
//	pkce, err := oauth.NewPKCE()
//	state, err := oauth.NewState()
//	// Persist pkce.Verifier and state in the user's session, then:
//	authURL, err := cfg.AuthorizationURL(oauth.AuthorizationRequest{
//	    State:         state,
//	    CodeChallenge: pkce.Challenge,
//	})
//
//	// Step 3: the browser comes back to the redirect URI.
//	code, err := cfg.ParseCallback(r.URL, state)
//	token, err := cfg.Exchange(ctx, code, pkce.Verifier)
//
//	// Step 4: call the API with an auto-refreshing token source.
//	source := oauth.NewTokenSource(cfg, token, saveTokenToDatabase)
//	client, err := assinafy.NewClient(assinafy.ClientOptions{TokenSource: source})
//
// Token and revocation responses are plain OAuth JSON objects, not the
// {status, message, data} envelope the rest of the API uses, so this package
// speaks to those endpoints directly rather than through the SDK's HTTP client.
// Failures are returned as *Error carrying the RFC 6749 error code.
package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/assinafy/golang-sdk/internal"
)

// Default endpoints for the Assinafy production environment. The authorization
// page lives on the authorization server; every endpoint your code calls lives
// on the API host.
const (
	// DefaultIssuer is the authorization server that owns the browser-facing flow.
	DefaultIssuer = "https://auth.assinafy.com.br"
	// DefaultResource is the RFC 8707 resource indicator naming the API a token is for.
	DefaultResource = "https://api.assinafy.com.br"
)

// Scopes an application can request. Request the minimum your product needs:
// the user reads every one of them before deciding.
const (
	// ScopeDocumentsRead reads documents, their pages, tags, signers, assignments and activity.
	ScopeDocumentsRead = "documents:read"
	// ScopeDocumentsWrite creates, updates and deletes documents and manages their
	// signers and assignments. It can spend the workspace's notification credits,
	// because sending a document for signature notifies its signers.
	ScopeDocumentsWrite = "documents:write"
	// ScopeTemplatesRead reads reusable document templates, their pages, roles, fields and tags.
	ScopeTemplatesRead = "templates:read"
	// ScopeTemplatesWrite creates, updates and deletes templates and their contents.
	ScopeTemplatesWrite = "templates:write"
	// ScopeAccountRead reads the workspace's profile, theme and logo.
	ScopeAccountRead = "account:read"
	// ScopeWebhooksWrite configures and deactivates the workspace webhook subscription.
	ScopeWebhooksWrite = "webhooks:write"
	// ScopeOpenID identifies the authenticated user and enables Config.UserInfo.
	ScopeOpenID = "openid"
	// ScopeProfile includes the user's name in the id_token and userinfo claims.
	ScopeProfile = "profile"
	// ScopeEmail includes the user's email and its verification status in those claims.
	ScopeEmail = "email"
	// ScopeOfflineAccess requests a refresh token, so the application keeps working
	// after the user's session expires. Without it, Token.RefreshToken is empty and
	// the user must approve the application again once the access token expires.
	ScopeOfflineAccess = "offline_access"
)

// ChallengeMethod is the only PKCE code-challenge method Assinafy accepts.
const ChallengeMethod = "S256"

// RFC 6749 and RFC 8707 error codes returned by the authorization server.
// Compare them against Error.Code.
const (
	// ErrCodeAccessDenied means the user declined the approval screen.
	ErrCodeAccessDenied = "access_denied"
	// ErrCodeInvalidClient means the client_id is unknown or disabled, or client
	// authentication failed. The description never reveals which.
	ErrCodeInvalidClient = "invalid_client"
	// ErrCodeInvalidGrant means the code or refresh token was expired, already
	// used, issued to another client, or paired with the wrong code_verifier or
	// redirect_uri. On a refresh it also means the user reconnected with
	// different permissions.
	ErrCodeInvalidGrant = "invalid_grant"
	// ErrCodeInvalidRequest means a PKCE or protocol parameter was missing or malformed.
	ErrCodeInvalidRequest = "invalid_request"
	// ErrCodeInvalidScope means a scope the application is not registered for, or none.
	ErrCodeInvalidScope = "invalid_scope"
	// ErrCodeInvalidTarget means the resource indicator is not one this server
	// issues tokens for, or disagrees with the authorized value.
	ErrCodeInvalidTarget = "invalid_target"
	// ErrCodeUnsupportedGrantType means a grant other than authorization_code or refresh_token.
	ErrCodeUnsupportedGrantType = "unsupported_grant_type"
	// ErrCodeUnsupportedResponseType means a response_type other than code.
	ErrCodeUnsupportedResponseType = "unsupported_response_type"
)

// Token type hints accepted by Config.Revoke.
const (
	// HintAccessToken tells the server the revoked value is an access token.
	HintAccessToken = "access_token"
	// HintRefreshToken tells the server the revoked value is a refresh token.
	HintRefreshToken = "refresh_token"
)

// expiryLeeway is how long before its stated expiry a token is treated as expired.
const expiryLeeway = 60 * time.Second

var (
	// ErrMissingClientID is returned when Config.ClientID is empty.
	ErrMissingClientID = errors.New("assinafy/oauth: ClientID is required")
	// ErrMissingRedirectURI is returned when Config.RedirectURI is empty.
	ErrMissingRedirectURI = errors.New("assinafy/oauth: RedirectURI is required")
	// ErrMissingState is returned when AuthorizationRequest.State is empty. A
	// fresh random state per connection attempt is the CSRF protection for the flow.
	ErrMissingState = errors.New("assinafy/oauth: State is required")
	// ErrMissingCodeChallenge is returned when AuthorizationRequest.CodeChallenge
	// is empty. PKCE is mandatory for every application, confidential ones included.
	ErrMissingCodeChallenge = errors.New("assinafy/oauth: CodeChallenge is required")
	// ErrStateMismatch is returned by Config.ParseCallback when the returned state
	// is not the one that was sent. The response is not yours; do not use it.
	ErrStateMismatch = errors.New("assinafy/oauth: callback state does not match")
	// ErrIssuerMismatch is returned by Config.ParseCallback when the iss parameter
	// is not the configured authorization server.
	ErrIssuerMismatch = errors.New("assinafy/oauth: callback issuer does not match")
	// ErrNoCode is returned by Config.ParseCallback when the callback carries
	// neither an authorization code nor an error.
	ErrNoCode = errors.New("assinafy/oauth: callback carries no authorization code")
	// ErrNoRefreshToken is returned when a refresh is attempted without a refresh
	// token. Request ScopeOfflineAccess to receive one.
	ErrNoRefreshToken = errors.New("assinafy/oauth: no refresh token; request the offline_access scope")
	// ErrMissingToken is returned when a required token argument is empty.
	ErrMissingToken = errors.New("assinafy/oauth: token is required")
)

// Error is an OAuth error response. The authorization server answers a failed
// token, refresh or revocation request with a flat {error, error_description}
// object rather than the API's usual envelope.
type Error struct {
	// Code is the RFC 6749 error code, such as invalid_grant or invalid_client.
	Code string `json:"error"`
	// Description is the server's human-readable explanation. It never reveals
	// whether a client_id or token exists.
	Description string `json:"error_description"`
	// StatusCode is the HTTP status that carried the error.
	StatusCode int `json:"-"`
}

// Error formats the OAuth error code, description and status.
func (e *Error) Error() string {
	if e.Description == "" {
		return fmt.Sprintf("assinafy/oauth: %s (status %d)", e.Code, e.StatusCode)
	}
	return fmt.Sprintf("assinafy/oauth: %s: %s (status %d)", e.Code, e.Description, e.StatusCode)
}

// ErrorCode returns the RFC 6749 error code carried by err, or "" when err is
// not an OAuth error. Use it to tell a user declining (ErrCodeAccessDenied)
// apart from a connection that must be re-established (ErrCodeInvalidGrant).
func ErrorCode(err error) string {
	var oauthErr *Error
	if errors.As(err, &oauthErr) && oauthErr != nil {
		return oauthErr.Code
	}
	return ""
}

// Endpoint holds the authorization server's URLs. The zero value resolves to
// the Assinafy production endpoints; DiscoverAuthorizationServer fills it from
// the server's own RFC 8414 metadata.
type Endpoint struct {
	// Issuer identifies the authorization server and is checked against the
	// callback's iss parameter.
	Issuer string
	// AuthorizationURL is the browser-facing approval page.
	AuthorizationURL string
	// TokenURL exchanges authorization codes and refresh tokens for access tokens.
	TokenURL string
	// RevocationURL revokes an access or refresh token.
	RevocationURL string
	// UserInfoURL returns OpenID Connect claims about the authorizing user.
	UserInfoURL string
}

// ProductionEndpoint returns the Assinafy production OAuth endpoints. They are
// the defaults for a Config whose Endpoint is left zero.
func ProductionEndpoint() Endpoint {
	return Endpoint{
		Issuer:           DefaultIssuer,
		AuthorizationURL: DefaultIssuer + "/oauth/authorize",
		TokenURL:         DefaultResource + "/v1/oauth/token",
		RevocationURL:    DefaultResource + "/v1/oauth/revoke",
		UserInfoURL:      DefaultResource + "/v1/oauth/userinfo",
	}
}

// resolve fills any empty field from the production defaults, so a Config that
// overrides only one URL keeps working.
func (e Endpoint) resolve() Endpoint {
	defaults := ProductionEndpoint()
	if e.Issuer == "" {
		e.Issuer = defaults.Issuer
	}
	if e.AuthorizationURL == "" {
		e.AuthorizationURL = defaults.AuthorizationURL
	}
	if e.TokenURL == "" {
		e.TokenURL = defaults.TokenURL
	}
	if e.RevocationURL == "" {
		e.RevocationURL = defaults.RevocationURL
	}
	if e.UserInfoURL == "" {
		e.UserInfoURL = defaults.UserInfoURL
	}
	return e
}

// Config describes one registered Assinafy application. Create the application
// in the Assinafy app under Settings, OAuth applications; it cannot be created
// through the API. A Config is safe for concurrent use once built.
type Config struct {
	// ClientID is the public identifier issued when the application was registered.
	ClientID string
	// ClientSecret authenticates a confidential application, whose code runs on a
	// server you control. Leave it empty for a public application, which
	// authenticates with PKCE alone and is never issued a secret. Never ship a
	// secret in mobile, browser or repository code.
	ClientSecret string
	// RedirectURI is where the browser returns after approval. It must be one of
	// the application's registered URIs, character for character: a trailing
	// slash makes it a different URI.
	RedirectURI string
	// Scopes are requested when Scopes is not set on the authorization request.
	// They must be within what the application is registered for.
	Scopes []string
	// Endpoint overrides individual OAuth URLs. The zero value uses production.
	Endpoint Endpoint
	// Resource is the RFC 8707 resource indicator sent to the authorization and
	// token endpoints. An empty value sends DefaultResource; set it to "-" to
	// send no resource indicator at all.
	Resource string
	// HTTPClient performs the server-to-server calls. A nil value uses a client
	// with a 30-second timeout.
	HTTPClient *http.Client
}

// ResourceIndicator returns the RFC 8707 resource value this Config sends, or
// "" when the indicator is suppressed.
func (c *Config) ResourceIndicator() string {
	switch c.Resource {
	case "":
		return DefaultResource
	case "-":
		return ""
	default:
		return c.Resource
	}
}

func (c *Config) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Transport: internal.Transport(), Timeout: 30 * time.Second}
}

// AuthorizationRequest carries the per-attempt values for one authorization URL.
// Create a fresh State and PKCE pair for every attempt and keep both in the
// user's session: the callback and the code exchange need them.
type AuthorizationRequest struct {
	// State is the required random per-attempt value echoed back on the callback.
	State string
	// CodeChallenge is the required S256 challenge from NewPKCE.
	CodeChallenge string
	// Scopes overrides Config.Scopes for this attempt. Ask for less than the
	// application is registered for when a user does not need every permission.
	Scopes []string
	// Nonce is an optional OpenID Connect value echoed in the id_token. Validate
	// it there when you request ScopeOpenID.
	Nonce string
}

// AuthorizationURL builds the URL to send the user's browser to. Use a full page
// navigation, a link or a redirect, never an AJAX call.
//
// If the client_id or redirect_uri is wrong the user is not sent back to you:
// the authorization server shows an error on its own page, because redirecting
// to an unverified address would be unsafe.
func (c *Config) AuthorizationURL(req AuthorizationRequest) (string, error) {
	switch {
	case c.ClientID == "":
		return "", ErrMissingClientID
	case c.RedirectURI == "":
		return "", ErrMissingRedirectURI
	case req.State == "":
		return "", ErrMissingState
	case req.CodeChallenge == "":
		return "", ErrMissingCodeChallenge
	}

	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = c.Scopes
	}

	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", c.ClientID)
	query.Set("redirect_uri", c.RedirectURI)
	query.Set("state", req.State)
	query.Set("code_challenge", req.CodeChallenge)
	query.Set("code_challenge_method", ChallengeMethod)
	if len(scopes) > 0 {
		query.Set("scope", strings.Join(scopes, " "))
	}
	if req.Nonce != "" {
		query.Set("nonce", req.Nonce)
	}
	if resource := c.ResourceIndicator(); resource != "" {
		query.Set("resource", resource)
	}

	authorizeURL, err := url.Parse(c.Endpoint.resolve().AuthorizationURL)
	if err != nil {
		return "", fmt.Errorf("assinafy/oauth: parse authorization URL: %w", err)
	}
	authorizeURL.RawQuery = query.Encode()
	return authorizeURL.String(), nil
}

// ParseCallback validates the redirect back from the authorization server and
// returns the one-time authorization code.
//
// It checks, in order, that the state equals wantState, that the iss parameter
// names the configured authorization server, and that the server did not report
// an error. A declined approval returns an *Error whose Code is
// ErrCodeAccessDenied. Pass the request's URL directly:
//
//	code, err := cfg.ParseCallback(r.URL, sessionState)
//
// The code is single-use and expires 60 seconds after approval, so exchange it
// immediately.
func (c *Config) ParseCallback(callbackURL *url.URL, wantState string) (string, error) {
	if callbackURL == nil {
		return "", fmt.Errorf("%w: callback URL is nil", ErrNoCode)
	}
	query := callbackURL.Query()

	if wantState == "" || !constantTimeEqual(query.Get("state"), wantState) {
		return "", ErrStateMismatch
	}
	if issuer := query.Get("iss"); issuer != "" && issuer != c.Endpoint.resolve().Issuer {
		return "", ErrIssuerMismatch
	}
	if code := query.Get("error"); code != "" {
		return "", &Error{
			Code:        code,
			Description: query.Get("error_description"),
			StatusCode:  http.StatusBadRequest,
		}
	}
	code := query.Get("code")
	if code == "" {
		return "", ErrNoCode
	}
	return code, nil
}

// Token is an OAuth token response. Persist every field: a refresh is only
// possible with RefreshToken, and Expiry tells a TokenSource when to renew.
type Token struct {
	// AccessToken authenticates API calls as Authorization: Bearer. A token sent
	// as X-Api-Key or in the query string is refused.
	AccessToken string `json:"access_token"`
	// TokenType is always "Bearer".
	TokenType string `json:"token_type"`
	// ExpiresIn is the access token's lifetime in seconds, normally 3600.
	ExpiresIn int `json:"expires_in"`
	// RefreshToken renews the access token without the user. It is present only
	// when ScopeOfflineAccess was requested and approved, and every refresh
	// replaces it with a new one.
	RefreshToken string `json:"refresh_token,omitempty"`
	// Scope is what was actually granted, space-separated. Read it instead of
	// assuming you received everything you asked for. ScopeOfflineAccess is a
	// request-time signal, so it never appears here.
	Scope string `json:"scope"`
	// IDToken is a signed OpenID Connect assertion (RS256), present only when
	// ScopeOpenID was granted. Validate it with an OpenID Connect library against
	// the issuer's JWKS before trusting its claims.
	IDToken string `json:"id_token,omitempty"`
	// Expiry is when AccessToken stops working, computed from ExpiresIn when the
	// token was issued. A zero value means the lifetime is unknown.
	Expiry time.Time `json:"expiry,omitempty"`
}

// Valid reports whether the token has an access token that is not within a
// minute of expiring. A zero Expiry is treated as valid, since the lifetime is
// then unknown.
func (t *Token) Valid() bool {
	if t == nil || t.AccessToken == "" {
		return false
	}
	return t.Expiry.IsZero() || time.Now().Add(expiryLeeway).Before(t.Expiry)
}

// Scopes splits Scope into the individual permissions that were granted.
func (t *Token) Scopes() []string {
	if t == nil || strings.TrimSpace(t.Scope) == "" {
		return nil
	}
	return strings.Fields(t.Scope)
}

// HasScope reports whether the granted scopes include scope.
func (t *Token) HasScope(scope string) bool {
	for _, granted := range t.Scopes() {
		if granted == scope {
			return true
		}
	}
	return false
}

// Exchange trades a one-time authorization code for a token. Call it from your
// server, with the code verifier stored alongside the state in step 1.
//
// The code expires 60 seconds after approval and can be used once; a stale,
// replayed or mismatched code returns an *Error whose Code is ErrCodeInvalidGrant.
func (c *Config) Exchange(ctx context.Context, code, codeVerifier string) (*Token, error) {
	if code == "" {
		return nil, fmt.Errorf("%w: authorization code is empty", ErrNoCode)
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("code_verifier", codeVerifier)
	if c.RedirectURI != "" {
		form.Set("redirect_uri", c.RedirectURI)
	}
	return c.token(ctx, form)
}

// Refresh exchanges a refresh token for a new access token and a new refresh
// token, and retires the one it was given.
//
// Save the returned RefreshToken before doing anything else with the response,
// and refresh one connection at a time. A reused refresh token cannot be told
// apart from a stolen one being replayed, so the server ends the whole
// connection and the user must approve the application again. Treat a timeout
// as "maybe it worked": re-read the token you saved rather than retrying with
// the old one. Connections last 30 days from approval and refreshing does not
// extend them.
func (c *Config) Refresh(ctx context.Context, refreshToken string) (*Token, error) {
	if refreshToken == "" {
		return nil, ErrNoRefreshToken
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	return c.token(ctx, form)
}

func (c *Config) token(ctx context.Context, form url.Values) (*Token, error) {
	if c.ClientID == "" {
		return nil, ErrMissingClientID
	}
	form.Set("client_id", c.ClientID)
	if c.ClientSecret != "" {
		form.Set("client_secret", c.ClientSecret)
	}
	if resource := c.ResourceIndicator(); resource != "" {
		form.Set("resource", resource)
	}

	issuedAt := time.Now()
	var token Token
	if err := c.postForm(ctx, c.Endpoint.resolve().TokenURL, form, &token); err != nil {
		return nil, err
	}
	if token.ExpiresIn > 0 {
		token.Expiry = issuedAt.Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	return &token, nil
}

// Revoke revokes an access or refresh token, which disconnects the user from
// the application. Call it when a user disconnects in your product instead of
// only deleting the token; revoking a refresh token also ends its access tokens.
//
// hint is optional and may be HintAccessToken or HintRefreshToken. Every token
// outcome answers 200, including an unknown, malformed or already-revoked
// token, so the endpoint can never be used to probe whether a token exists.
// Only failed client authentication returns an *Error.
func (c *Config) Revoke(ctx context.Context, token, hint string) error {
	if token == "" {
		return ErrMissingToken
	}
	if c.ClientID == "" {
		return ErrMissingClientID
	}
	form := url.Values{}
	form.Set("token", token)
	form.Set("client_id", c.ClientID)
	if hint != "" {
		form.Set("token_type_hint", hint)
	}
	if c.ClientSecret != "" {
		form.Set("client_secret", c.ClientSecret)
	}
	return c.postForm(ctx, c.Endpoint.resolve().RevocationURL, form, nil)
}

// UserInfo describes the user who approved the application. Claims beyond sub
// depend on the scopes granted.
type UserInfo struct {
	// Sub is the user's stable identifier, granted with ScopeOpenID.
	Sub string `json:"sub"`
	// Name is the user's display name, nil unless ScopeProfile was granted.
	Name *string `json:"name,omitempty"`
	// Email is the user's address, nil unless ScopeEmail was granted.
	Email *string `json:"email,omitempty"`
	// EmailVerified reports whether that address is verified, nil unless
	// ScopeEmail was granted.
	EmailVerified *bool `json:"email_verified,omitempty"`
}

// UserInfo returns the OpenID Connect claims for accessToken. It requires
// ScopeOpenID; Name additionally requires ScopeProfile and Email requires
// ScopeEmail. Like the token endpoint, the response is a flat claims object
// rather than the API's usual envelope.
//
// GET /v1/oauth/userinfo.
func (c *Config) UserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	if accessToken == "" {
		return nil, ErrMissingToken
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Endpoint.resolve().UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("assinafy/oauth: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	var info UserInfo
	if err := c.do(req, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *Config) postForm(ctx context.Context, endpoint string, form url.Values, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("assinafy/oauth: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.do(req, result)
}

func (c *Config) do(req *http.Request, result any) error {
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("assinafy/oauth: %s %s: %w", req.Method, redactQuery(req.URL), redactTransportError(err))
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("assinafy/oauth: read response body: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return newResponseError(resp.StatusCode, body)
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("assinafy/oauth: decode response: %w", err)
	}
	return nil
}

// newResponseError decodes the flat OAuth error object, falling back to the
// API's own envelope for failures raised before the OAuth handler runs.
func newResponseError(statusCode int, body []byte) error {
	oauthErr := &Error{StatusCode: statusCode}
	if err := json.Unmarshal(body, oauthErr); err == nil && oauthErr.Code != "" {
		return oauthErr
	}

	var envelope struct {
		Message string `json:"message"`
	}
	oauthErr.Code = "invalid_request"
	switch {
	case json.Unmarshal(body, &envelope) == nil && envelope.Message != "":
		oauthErr.Description = envelope.Message
	case len(body) > 0:
		oauthErr.Description = string(body)
	default:
		oauthErr.Description = http.StatusText(statusCode)
	}
	return oauthErr
}

// redactQuery hides any credential a transport error would otherwise print.
func redactQuery(u *url.URL) string {
	if u == nil {
		return ""
	}
	clean := *u
	clean.RawQuery = ""
	clean.User = nil
	return clean.String()
}

// redactTransportError strips the query string net/http records on a *url.Error,
// so a failed dial cannot print a credential carried in the URL.
func redactTransportError(err error) error {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return err
	}
	clean := *urlErr
	if parsed, parseErr := url.Parse(clean.URL); parseErr == nil {
		clean.URL = redactQuery(parsed)
	} else if i := strings.IndexByte(clean.URL, '?'); i >= 0 {
		clean.URL = clean.URL[:i]
	}
	return &clean
}
