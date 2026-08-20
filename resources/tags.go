package resources

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// TagResource exposes authenticated workspace Tag endpoints. Its methods follow
// the package-level account-selection and error contract.
type TagResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewTagResource constructs a TagResource bound to the given HTTPClient and
// default account ID.
func NewTagResource(httpClient *internal.HTTPClient, accountID string) *TagResource {
	return &TagResource{http: httpClient, accountID: accountID}
}

// List returns workspace Tag payloads with the optional search term. It requires
// client authentication and uses the configured default for an empty accountID.
// GET /accounts/{account_id}/tags.
func (r *TagResource) List(ctx context.Context, accountID, search string) ([]models.Tag, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out []models.Tag
	_, err := r.http.NewRequest(http.MethodGet, "/accounts/"+url.PathEscape(accountID)+"/tags").
		WithQuery("search", search).
		Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Create creates a new tag in the workspace. The API returns 409 Conflict when
// a tag with the same name (case-insensitive) already exists. It sends
// CreateTagRequest, returns the created Tag, requires client authentication, and
// uses the configured default for an empty accountID.
// POST /accounts/{account_id}/tags.
func (r *TagResource) Create(ctx context.Context, accountID string, body *models.CreateTagRequest) (*models.Tag, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.Tag
	path := "/accounts/" + url.PathEscape(accountID) + "/tags"
	_, err := r.http.NewRequest(http.MethodPost, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Update updates a tag's name and/or color. Nil request fields are left
// unchanged; set UpdateTagRequest.ClearColor to remove an existing color. It
// returns the updated Tag, requires authentication, and uses the default account.
// PUT /accounts/{account_id}/tags/{tag_id}.
func (r *TagResource) Update(ctx context.Context, accountID, tagID string, body *models.UpdateTagRequest) (*models.Tag, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.Tag
	path := "/accounts/" + url.PathEscape(accountID) + "/tags/" + url.PathEscape(tagID)
	_, err := r.http.NewRequest(http.MethodPut, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes a tag; pass force=true to request detachment from associated
// resources. It requires client authentication, returns only error (discarding
// the documented {deleted:boolean} result), and uses the configured default for
// an empty accountID.
// DELETE /accounts/{account_id}/tags/{tag_id}.
func (r *TagResource) Delete(ctx context.Context, accountID, tagID string, force bool) error {
	return r.delete(ctx, accountID, tagID, force, nil)
}

// DeleteWithResult deletes a tag and returns the documented deletion result.
// DELETE /accounts/{account_id}/tags/{tag_id}.
func (r *TagResource) DeleteWithResult(ctx context.Context, accountID, tagID string, force bool) (*models.TagDeleteResult, error) {
	var out models.TagDeleteResult
	if err := r.delete(ctx, accountID, tagID, force, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *TagResource) delete(ctx context.Context, accountID, tagID string, force bool, out any) error {
	accountID = resolveAccountID(accountID, r.accountID)

	req := r.http.NewRequest(http.MethodDelete, "/accounts/"+url.PathEscape(accountID)+"/tags/"+url.PathEscape(tagID))
	if force {
		req.WithQuery("force", strconv.FormatBool(force))
	}
	_, err := req.Execute(ctx, out)
	return err
}
