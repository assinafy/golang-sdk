package resources

import (
	"context"
	"fmt"
	"net/http"

	"github.com/assinafy/assinafy-go/internal"
	"github.com/assinafy/assinafy-go/models"
)

type WebhookResource struct {
	httpClient *internal.HTTPClient
	accountID  string
}

func NewWebhookResource(httpClient *internal.HTTPClient, accountID string) *WebhookResource {
	return &WebhookResource{
		httpClient: httpClient,
		accountID:  accountID,
	}
}

func (r *WebhookResource) Register(ctx context.Context, accountID string, req *models.RegisterWebhookRequest) (*models.WebhookSubscription, error) {
	if accountID == "" {
		accountID = r.accountID
	}

	var result models.WebhookSubscription
	httpReq := r.httpClient.NewRequest(http.MethodPut, fmt.Sprintf("/accounts/%s/webhooks/subscriptions", accountID))
	httpReq.WithBody(req)
	_, err := httpReq.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *WebhookResource) Get(ctx context.Context, accountID string) (*models.WebhookSubscription, error) {
	if accountID == "" {
		accountID = r.accountID
	}

	var result models.WebhookSubscription
	req := r.httpClient.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%s/webhooks/subscriptions", accountID))
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *WebhookResource) Update(ctx context.Context, accountID string, req *models.RegisterWebhookRequest) (*models.WebhookSubscription, error) {
	return r.Register(ctx, accountID, req)
}

func (r *WebhookResource) Inactivate(ctx context.Context, accountID string) error {
	if accountID == "" {
		accountID = r.accountID
	}

	req := r.httpClient.NewRequest(http.MethodPut, fmt.Sprintf("/accounts/%s/webhooks/inactivate", accountID))
	_, err := req.Execute(ctx, nil)
	return err
}

func (r *WebhookResource) Delete(ctx context.Context, accountID string) error {
	if accountID == "" {
		accountID = r.accountID
	}

	req := r.httpClient.NewRequest(http.MethodDelete, fmt.Sprintf("/accounts/%s/webhooks/subscriptions", accountID))
	_, err := req.Execute(ctx, nil)
	return err
}

func (r *WebhookResource) ListEventTypes(ctx context.Context) ([]models.WebhookEventType, error) {
	var result []models.WebhookEventType
	req := r.httpClient.NewRequest(http.MethodGet, "/webhooks/event-types")
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *WebhookResource) ListDispatches(ctx context.Context, accountID string, params *models.WebhookDispatchListParams) (*models.PaginatedResult[models.WebhookDispatch], error) {
	if accountID == "" {
		accountID = r.accountID
	}
	if params == nil {
		params = &models.WebhookDispatchListParams{}
	}

	var result []models.WebhookDispatch
	req := r.httpClient.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%s/webhooks", accountID))
	req.WithQuery("page", fmt.Sprintf("%d", params.Page))
	req.WithQuery("per-page", fmt.Sprintf("%d", params.PerPage))
	if params.Delivered != nil {
		req.WithQuery("delivered", fmt.Sprintf("%t", *params.Delivered))
	}
	if params.Event != "" {
		req.WithQuery("event", params.Event)
	}

	resp, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &models.PaginatedResult[models.WebhookDispatch]{
		Data:       result,
		Pagination: extractPagination(resp.Headers),
	}, nil
}

func (r *WebhookResource) RetryDispatch(ctx context.Context, accountID, dispatchID string) error {
	req := r.httpClient.NewRequest(http.MethodPost, fmt.Sprintf("/accounts/%s/webhooks/%s/retry", accountID, dispatchID))
	_, err := req.Execute(ctx, nil)
	return err
}
