package resources

import (
	"context"
	"net/http"
	"net/url"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// AccountResource exposes authenticated workspace Account endpoints. Its
// methods follow the package-level account-selection and error contract.
type AccountResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewAccountResource constructs an AccountResource bound to the given HTTPClient
// and default account ID.
func NewAccountResource(httpClient *internal.HTTPClient, accountID string) *AccountResource {
	return &AccountResource{http: httpClient, accountID: accountID}
}

// List returns the Account payloads for workspaces available to the
// authenticated user. It requires a configured client credential; each item
// carries the list endpoint's slim account shape.
// GET /accounts.
func (r *AccountResource) List(ctx context.Context) ([]models.Account, error) {
	var out []models.Account
	if _, err := r.http.NewRequest(http.MethodGet, "/accounts").Execute(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Create sends the required CreateAccountRequest and returns the created Account.
// It requires an authenticated user and creates a new owned workspace.
// POST /accounts.
func (r *AccountResource) Create(ctx context.Context, body *models.CreateAccountRequest) (*models.Account, error) {
	var out models.Account
	if _, err := r.http.NewRequest(http.MethodPost, "/accounts").WithBody(body).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get returns one Account payload. It requires client authentication; an empty
// accountID uses the resource's configured default.
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

// Update sends UpdateAccountRequest and returns the updated Account. It requires
// client authentication; an empty accountID uses the configured default, and
// nil request fields leave their account attributes unchanged.
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

// Delete permanently deletes a workspace and returns only error, discarding the
// documented empty data array. It requires client authentication; an empty
// accountID uses the configured default. Setting force also cancels an active
// paid subscription.
// DELETE /accounts/{account_id}.
func (r *AccountResource) Delete(ctx context.Context, accountID string, force bool) error {
	accountID = resolveAccountID(accountID, r.accountID)
	path := "/accounts/" + url.PathEscape(accountID)
	_, err := r.http.NewRequest(http.MethodDelete, path).WithBody(map[string]bool{"force": force}).Execute(ctx, nil)
	return err
}

// GetTheme returns the AccountTheme payload (name, colors, and nullable logo
// URL). It requires client authentication; an empty accountID uses the default.
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

// DownloadLogo returns the account logo's raw image bytes. It requires client
// authentication; an empty accountID uses the default. A missing logo produces
// a 404 *errors.APIError.
// GET /accounts/{account_id}/logo.
func (r *AccountResource) DownloadLogo(ctx context.Context, accountID string) ([]byte, error) {
	accountID = resolveAccountID(accountID, r.accountID)
	return r.http.Download(ctx, "/accounts/"+url.PathEscape(accountID)+"/logo", nil)
}

// UploadLogo sends fileContent as the multipart "file" part and returns an
// envelope-only success. It requires client authentication; an empty accountID
// uses the default. An existing logo is replaced.
// POST /accounts/{account_id}/logo.
func (r *AccountResource) UploadLogo(ctx context.Context, accountID string, fileContent []byte, fileName string) error {
	accountID = resolveAccountID(accountID, r.accountID)
	path := "/accounts/" + url.PathEscape(accountID) + "/logo"
	_, err := r.http.UploadMultipart(ctx, path, "file", fileName, fileContent, nil, nil)
	return err
}

// DeleteLogo removes the account logo and returns an envelope-only success. It
// requires client authentication; an empty accountID uses the default.
// DELETE /accounts/{account_id}/logo.
func (r *AccountResource) DeleteLogo(ctx context.Context, accountID string) error {
	accountID = resolveAccountID(accountID, r.accountID)
	path := "/accounts/" + url.PathEscape(accountID) + "/logo"
	_, err := r.http.NewRequest(http.MethodDelete, path).Execute(ctx, nil)
	return err
}

// Stats returns DocumentStatsRow payloads filtered by optional StatsParams. It
// requires client authentication; an empty accountID uses the default. Daily
// granularity requires a YYYY-MM Month.
// GET /accounts/{account_id}/stats.
func (r *AccountResource) Stats(ctx context.Context, accountID string, params *models.StatsParams) ([]models.DocumentStatsRow, error) {
	accountID = resolveAccountID(accountID, r.accountID)
	var out []models.DocumentStatsRow
	req := r.http.NewRequest(http.MethodGet, "/accounts/"+url.PathEscape(accountID)+"/stats")
	applyStatsParams(req, params)
	if _, err := req.Execute(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}
