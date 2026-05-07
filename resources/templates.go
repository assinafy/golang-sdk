package resources

import (
	"context"
	"fmt"
	"net/http"

	"github.com/assinafy/assinafy-go/internal"
	"github.com/assinafy/assinafy-go/models"
)

type TemplateResource struct {
	httpClient *internal.HTTPClient
	accountID  string
}

func NewTemplateResource(httpClient *internal.HTTPClient, accountID string) *TemplateResource {
	return &TemplateResource{
		httpClient: httpClient,
		accountID:  accountID,
	}
}

func (r *TemplateResource) List(ctx context.Context, accountID string, params *models.ListParams) (*models.PaginatedResult[models.TemplateListItem], error) {
	if accountID == "" {
		accountID = r.accountID
	}
	if params == nil {
		params = &models.ListParams{}
	}
	params.SetDefaults()

	var result []models.TemplateListItem
	req := r.httpClient.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%s/templates", accountID))
	req.WithQuery("page", fmt.Sprintf("%d", params.Page))
	req.WithQuery("per-page", fmt.Sprintf("%d", params.PerPage))
	if params.Search != "" {
		req.WithQuery("search", params.Search)
	}

	resp, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &models.PaginatedResult[models.TemplateListItem]{
		Data:       result,
		Pagination: extractPagination(resp.Headers),
	}, nil
}

func (r *TemplateResource) Get(ctx context.Context, accountID, templateID string) (*models.TemplateDetailsResponse, error) {
	var result models.TemplateDetailsResponse
	req := r.httpClient.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%s/templates/%s", accountID, templateID))
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
