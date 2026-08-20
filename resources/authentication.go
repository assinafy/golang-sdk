package resources

import (
	"context"
	"net/http"
	"net/url"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// AuthenticationResource exposes login, password, social-login, and API-key
// endpoints. Its methods follow the package-level response and error contract.
type AuthenticationResource struct {
	http *internal.HTTPClient
}

// NewAuthenticationResource constructs an AuthenticationResource.
func NewAuthenticationResource(httpClient *internal.HTTPClient) *AuthenticationResource {
	return &AuthenticationResource{http: httpClient}
}

// Login sends LoginRequest without a client credential and returns an
// AuthenticationResult containing the bearer token, user, and accounts.
// POST /login.
func (r *AuthenticationResource) Login(ctx context.Context, body *models.LoginRequest) (*models.AuthenticationResult, error) {
	var out models.AuthenticationResult
	_, err := r.http.NewRequest(http.MethodPost, "/login").WithoutAuth().WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SocialLogin sends SocialLoginRequest without a client credential and returns
// an AuthenticationResult. Provider and provider token failures are API errors.
// POST /authentication/social-login.
func (r *AuthenticationResource) SocialLogin(ctx context.Context, body *models.SocialLoginRequest) (*models.AuthenticationResult, error) {
	var out models.AuthenticationResult
	_, err := r.http.NewRequest(http.MethodPost, "/authentication/social-login").WithoutAuth().WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SocialLoginURL returns, without making a network request, the browser URL that starts the OAuth flow with a
// social provider (e.g. "google"). GET /auth/authenticate is a 302 redirect to
// the provider's consent screen, so it is meant to be opened in a user's
// browser rather than called from a backend; the SDK only builds the URL. After
// the user authorizes, the provider redirects to the front-end /login-callback,
// which receives the resulting access token.
func (r *AuthenticationResource) SocialLoginURL(provider string) string {
	u := r.http.BaseURL() + "/auth/authenticate"
	if provider != "" {
		u += "?authclient=" + url.QueryEscape(provider)
	}
	return u
}

// LinkSocialLogin sends LinkSocialLoginRequest using client authentication (API
// key or bearer token) and links the provider identity to the current user. It
// returns an empty success envelope; an invalid identity produces an API error.
// POST /auth/link-social-login.
func (r *AuthenticationResource) LinkSocialLogin(ctx context.Context, body *models.LinkSocialLoginRequest) error {
	_, err := r.http.NewRequest(http.MethodPost, "/auth/link-social-login").WithBody(body).Execute(ctx, nil)
	return err
}

// CreateAPIKey sends CreateAPIKeyRequest using client authentication (API key or
// bearer token) and returns APIKeyResult with the new full key. Store it securely;
// later retrieval returns only a masked value.
// POST /users/api-keys.
func (r *AuthenticationResource) CreateAPIKey(ctx context.Context, body *models.CreateAPIKeyRequest) (*models.APIKeyResult, error) {
	var out models.APIKeyResult
	_, err := r.http.NewRequest(http.MethodPost, "/users/api-keys").WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAPIKey uses client authentication (API key or bearer token) and returns
// APIKeyResult containing the masked key, or a nil APIKey when none exists.
// GET /users/api-keys.
func (r *AuthenticationResource) GetAPIKey(ctx context.Context) (*models.APIKeyResult, error) {
	var out models.APIKeyResult
	_, err := r.http.NewRequest(http.MethodGet, "/users/api-keys").Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteAPIKey uses client authentication (API key or bearer token), permanently
// revokes the current API key, and returns only error, discarding the documented
// empty data array.
// DELETE /users/api-keys.
func (r *AuthenticationResource) DeleteAPIKey(ctx context.Context) error {
	_, err := r.http.NewRequest(http.MethodDelete, "/users/api-keys").Execute(ctx, nil)
	return err
}

// ChangePassword sends ChangePasswordRequest using client authentication (API key
// or bearer token), changes the password, and returns EmailResult. Invalid current
// credentials are API errors.
// PUT /authentication/change-password.
func (r *AuthenticationResource) ChangePassword(ctx context.Context, body *models.ChangePasswordRequest) (*models.EmailResult, error) {
	var out models.EmailResult
	_, err := r.http.NewRequest(http.MethodPut, "/authentication/change-password").WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// RequestPasswordReset sends RequestPasswordResetRequest without a client
// credential, triggers a reset email when accepted, and returns EmailResult.
// PUT /authentication/request-password-reset.
func (r *AuthenticationResource) RequestPasswordReset(ctx context.Context, body *models.RequestPasswordResetRequest) (*models.EmailResult, error) {
	var out models.EmailResult
	_, err := r.http.NewRequest(http.MethodPut, "/authentication/request-password-reset").WithoutAuth().WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ResetPassword sends ResetPasswordRequest without a client credential, changes
// the password for a valid reset token, and returns EmailResult.
// PUT /authentication/reset-password.
func (r *AuthenticationResource) ResetPassword(ctx context.Context, body *models.ResetPasswordRequest) (*models.EmailResult, error) {
	var out models.EmailResult
	_, err := r.http.NewRequest(http.MethodPut, "/authentication/reset-password").WithoutAuth().WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
