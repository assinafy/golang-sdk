package resources

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// TagResource exposes the documented workspace `Tag` endpoints.
type TagResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewTagResource constructs a TagResource bound to the given HTTPClient and
// default account ID.
func NewTagResource(httpClient *internal.HTTPClient, accountID string) *TagResource {
	return &TagResource{http: httpClient, accountID: accountID}
}

// List returns the workspace tags, ordered alphabetically by name. The
// optional search argument is a case-insensitive substring filter on the name.
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
// a tag with the same name (case-insensitive) already exists.
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
// unchanged. The API returns 409 Conflict when the new name collides with
// another tag.
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

// Delete deletes a tag. By default the API returns 409 Conflict if the tag is
// still attached to a document or template; pass force=true to detach it
// everywhere and delete it.
// DELETE /accounts/{account_id}/tags/{tag_id}.
func (r *TagResource) Delete(ctx context.Context, accountID, tagID string, force bool) error {
	accountID = resolveAccountID(accountID, r.accountID)

	req := r.http.NewRequest(http.MethodDelete, "/accounts/"+url.PathEscape(accountID)+"/tags/"+url.PathEscape(tagID))
	if force {
		req.WithQuery("force", strconv.FormatBool(force))
	}
	_, err := req.Execute(ctx, nil)
	return err
}
