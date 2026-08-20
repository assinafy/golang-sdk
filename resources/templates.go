package resources

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// TemplateResource exposes authenticated account-template read endpoints. Its
// methods follow the package-level account-selection and error contract.
type TemplateResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewTemplateResource constructs a TemplateResource.
func NewTemplateResource(httpClient *internal.HTTPClient, accountID string) *TemplateResource {
	return &TemplateResource{http: httpClient, accountID: accountID}
}

// List returns Template payloads and X-Pagination metadata. Search, page, and
// per-page are documented; populated shared status, sort, and tag fields are
// compatibility queries. It requires client authentication; an empty accountID
// uses the configured default.
// GET /accounts/{account_id}/templates.
func (r *TemplateResource) List(ctx context.Context, accountID string, params *models.ListParams) (*models.PaginatedResult[models.Template], error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out []models.Template
	req := r.http.NewRequest(http.MethodGet, "/accounts/"+url.PathEscape(accountID)+"/templates")
	applyListParams(req, params)
	if params != nil {
		req.WithQuery("status", params.Status)
		if len(params.Tags) > 0 {
			req.WithQuery("tags", strings.Join(params.Tags, ","))
		}
	}

	resp, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return paginated(out, resp), nil
}

// Get returns the authenticated Template detail payload, including roles, pages,
// and default document tags when supplied by the API. An empty accountID uses
// the configured default. This live compatibility route is absent from the
// current OpenAPI; List is the only officially described template operation.
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
