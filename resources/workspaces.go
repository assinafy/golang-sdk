package resources

import (
	"context"
	"net/http"

	"github.com/assinafy/assinafy-go/internal"
	"github.com/assinafy/assinafy-go/models"
)

type WorkspaceResource struct {
	httpClient *internal.HTTPClient
}

func NewWorkspaceResource(httpClient *internal.HTTPClient) *WorkspaceResource {
	return &WorkspaceResource{
		httpClient: httpClient,
	}
}

func (r *WorkspaceResource) Create(ctx context.Context, req *models.CreateWorkspaceRequest) (*models.Workspace, error) {
	var result models.Workspace
	httpReq := r.httpClient.NewRequest(http.MethodPost, "/accounts")
	httpReq.WithBody(req)
	_, err := httpReq.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *WorkspaceResource) List(ctx context.Context) ([]models.WorkspaceListItem, error) {
	var result []models.WorkspaceListItem
	req := r.httpClient.NewRequest(http.MethodGet, "/accounts")
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *WorkspaceResource) Get(ctx context.Context, accountID string) (*models.Workspace, error) {
	var result models.Workspace
	req := r.httpClient.NewRequest(http.MethodGet, "/accounts/"+accountID)
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *WorkspaceResource) Update(ctx context.Context, accountID string, req *models.UpdateWorkspaceRequest) (*models.Workspace, error) {
	var result models.Workspace
	httpReq := r.httpClient.NewRequest(http.MethodPut, "/accounts/"+accountID)
	httpReq.WithBody(req)
	_, err := httpReq.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *WorkspaceResource) Delete(ctx context.Context, accountID string) error {
	req := r.httpClient.NewRequest(http.MethodDelete, "/accounts/"+accountID)
	_, err := req.Execute(ctx, nil)
	return err
}
