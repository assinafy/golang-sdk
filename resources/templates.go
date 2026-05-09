package resources

import (
	"context"
	"net/http"
	"net/url"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// TemplateResource exposes the documented `Template` endpoints.
type TemplateResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewTemplateResource constructs a TemplateResource.
func NewTemplateResource(httpClient *internal.HTTPClient, accountID string) *TemplateResource {
	return &TemplateResource{http: httpClient, accountID: accountID}
}

// List returns the workspace templates page.
// GET /accounts/{account_id}/templates.
func (r *TemplateResource) List(ctx context.Context, accountID string, params *models.ListParams) (*models.PaginatedResult[models.Template], error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out []models.Template
	req := r.http.NewRequest(http.MethodGet, "/accounts/"+url.PathEscape(accountID)+"/templates")
	applyListParams(req, params)
	if params != nil {
		req.WithQuery("status", params.Status)
	}

	resp, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return paginated(out, resp), nil
}

// Get retrieves a template's details.
// GET /accounts/{account_id}/templates/{template_id}.
func (r *TemplateResource) Get(ctx context.Context, accountID, templateID string) (*models.Template, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.Template
	path := "/accounts/" + url.PathEscape(accountID) + "/templates/" + url.PathEscape(templateID)
	_, err := r.http.NewRequest(http.MethodGet, path).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
