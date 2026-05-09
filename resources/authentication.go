package resources

import (
	"context"
	"net/http"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// AuthenticationResource exposes the documented `Authentication` endpoints.
type AuthenticationResource struct {
	http *internal.HTTPClient
}

// NewAuthenticationResource constructs an AuthenticationResource.
func NewAuthenticationResource(httpClient *internal.HTTPClient) *AuthenticationResource {
	return &AuthenticationResource{http: httpClient}
}

// Login authenticates a user with email and password.
// POST /login.
func (r *AuthenticationResource) Login(ctx context.Context, body *models.LoginRequest) (*models.AuthenticationResult, error) {
	var out models.AuthenticationResult
	_, err := r.http.NewRequest(http.MethodPost, "/login").WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SocialLogin authenticates a user via a social provider.
// POST /authentication/social-login.
func (r *AuthenticationResource) SocialLogin(ctx context.Context, body *models.SocialLoginRequest) (*models.AuthenticationResult, error) {
	var out models.AuthenticationResult
	_, err := r.http.NewRequest(http.MethodPost, "/authentication/social-login").WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateAPIKey generates a new API key.
// POST /users/api-keys.
func (r *AuthenticationResource) CreateAPIKey(ctx context.Context, body *models.CreateAPIKeyRequest) (*models.APIKeyResult, error) {
	var out models.APIKeyResult
	_, err := r.http.NewRequest(http.MethodPost, "/users/api-keys").WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAPIKey retrieves the masked API key.
// GET /users/api-keys.
func (r *AuthenticationResource) GetAPIKey(ctx context.Context) (*models.APIKeyResult, error) {
	var out models.APIKeyResult
	_, err := r.http.NewRequest(http.MethodGet, "/users/api-keys").Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteAPIKey deletes the API key.
// DELETE /users/api-keys.
func (r *AuthenticationResource) DeleteAPIKey(ctx context.Context) error {
	_, err := r.http.NewRequest(http.MethodDelete, "/users/api-keys").Execute(ctx, nil)
	return err
}

// ChangePassword changes the user's password.
// PUT /authentication/change-password.
func (r *AuthenticationResource) ChangePassword(ctx context.Context, body *models.ChangePasswordRequest) (*models.EmailResult, error) {
	var out models.EmailResult
	_, err := r.http.NewRequest(http.MethodPut, "/authentication/change-password").WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// RequestPasswordReset triggers a password-reset email.
// PUT /authentication/request-password-reset.
func (r *AuthenticationResource) RequestPasswordReset(ctx context.Context, body *models.RequestPasswordResetRequest) (*models.EmailResult, error) {
	var out models.EmailResult
	_, err := r.http.NewRequest(http.MethodPut, "/authentication/request-password-reset").WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ResetPassword completes a password reset.
// PUT /authentication/reset-password.
func (r *AuthenticationResource) ResetPassword(ctx context.Context, body *models.ResetPasswordRequest) (*models.EmailResult, error) {
	var out models.EmailResult
	_, err := r.http.NewRequest(http.MethodPut, "/authentication/reset-password").WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
