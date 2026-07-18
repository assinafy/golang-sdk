package resources

import (
	"context"
	"net/http"
	"net/url"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// AccountResource exposes the documented, non-destructive workspace `Account`
// endpoints. Account creation, deletion, and logo upload/removal are
// administrative, production-only operations and are intentionally not exposed
// by this SDK.
type AccountResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewAccountResource constructs an AccountResource bound to the given HTTPClient
// and default account ID.
func NewAccountResource(httpClient *internal.HTTPClient, accountID string) *AccountResource {
	return &AccountResource{http: httpClient, accountID: accountID}
}

// List returns the workspace accounts the authenticated user belongs to. Each
// item carries the slim shape (id, name, roles, is_delete_allowed, created_at).
// GET /accounts.
func (r *AccountResource) List(ctx context.Context) ([]models.Account, error) {
	var out []models.Account
	if _, err := r.http.NewRequest(http.MethodGet, "/accounts").Execute(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get retrieves a single workspace account. The response carries id, name,
// primary_color, secondary_color, and created_at.
// GET /accounts/{account_id}.
func (r *AccountResource) Get(ctx context.Context, accountID string) (*models.Account, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.Account
	path := "/accounts/" + url.PathEscape(accountID)
	if _, err := r.http.NewRequest(http.MethodGet, path).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update changes an account's profile (name and/or notification sender type).
// Nil request fields are left unchanged.
// PUT /accounts/{account_id}.
func (r *AccountResource) Update(ctx context.Context, accountID string, body *models.UpdateAccountRequest) (*models.Account, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.Account
	path := "/accounts/" + url.PathEscape(accountID)
	if _, err := r.http.NewRequest(http.MethodPut, path).WithBody(body).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetTheme returns an account's branding theme (name, colors, logo URL).
// GET /accounts/{account_id}/theme.
func (r *AccountResource) GetTheme(ctx context.Context, accountID string) (*models.AccountTheme, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.AccountTheme
	path := "/accounts/" + url.PathEscape(accountID) + "/theme"
	if _, err := r.http.NewRequest(http.MethodGet, path).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DownloadLogo fetches the account logo image bytes. It returns a 404 APIError
// when the account has no logo configured.
// GET /accounts/{account_id}/logo.
func (r *AccountResource) DownloadLogo(ctx context.Context, accountID string) ([]byte, error) {
	accountID = resolveAccountID(accountID, r.accountID)
	return r.http.Download(ctx, "/accounts/"+url.PathEscape(accountID)+"/logo", nil)
}
