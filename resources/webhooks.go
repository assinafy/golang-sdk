package resources

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// WebhookResource exposes the documented `Webhooks` endpoints.
type WebhookResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewWebhookResource constructs a WebhookResource.
func NewWebhookResource(httpClient *internal.HTTPClient, accountID string) *WebhookResource {
	return &WebhookResource{http: httpClient, accountID: accountID}
}

// UpdateSubscription registers or updates the workspace webhook subscription.
// PUT /accounts/{account_id}/webhooks/subscriptions.
func (r *WebhookResource) UpdateSubscription(ctx context.Context, accountID string, body *models.UpdateWebhookSubscriptionRequest) (*models.WebhookSubscription, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.WebhookSubscription
	path := "/accounts/" + url.PathEscape(accountID) + "/webhooks/subscriptions"
	_, err := r.http.NewRequest(http.MethodPut, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetSubscription retrieves the current webhook subscription.
// GET /accounts/{account_id}/webhooks/subscriptions.
func (r *WebhookResource) GetSubscription(ctx context.Context, accountID string) (*models.WebhookSubscription, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.WebhookSubscription
	path := "/accounts/" + url.PathEscape(accountID) + "/webhooks/subscriptions"
	_, err := r.http.NewRequest(http.MethodGet, path).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteSubscription removes the workspace webhook subscription.
// DELETE /accounts/{account_id}/webhooks/subscriptions.
func (r *WebhookResource) DeleteSubscription(ctx context.Context, accountID string) (*models.WebhookSubscription, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.WebhookSubscription
	path := "/accounts/" + url.PathEscape(accountID) + "/webhooks/subscriptions"
	_, err := r.http.NewRequest(http.MethodDelete, path).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Inactivate disables the workspace webhook subscription.
// PUT /accounts/{account_id}/webhooks/inactivate.
func (r *WebhookResource) Inactivate(ctx context.Context, accountID string) (*models.WebhookSubscription, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.WebhookSubscription
	path := "/accounts/" + url.PathEscape(accountID) + "/webhooks/inactivate"
	_, err := r.http.NewRequest(http.MethodPut, path).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListEventTypes returns the available webhook event types.
// GET /webhooks/event-types.
func (r *WebhookResource) ListEventTypes(ctx context.Context) ([]models.WebhookEventType, error) {
	var out []models.WebhookEventType
	_, err := r.http.NewRequest(http.MethodGet, "/webhooks/event-types").Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListDispatches returns webhook delivery attempts for the account.
// GET /accounts/{account_id}/webhooks.
func (r *WebhookResource) ListDispatches(ctx context.Context, accountID string, params *models.WebhookDispatchListParams) (*models.PaginatedResult[models.WebhookDispatch], error) {
	accountID = resolveAccountID(accountID, r.accountID)
	if params == nil {
		params = &models.WebhookDispatchListParams{}
	}
	params.SetDefaults()

	var out []models.WebhookDispatch
	req := r.http.NewRequest(http.MethodGet, "/accounts/"+url.PathEscape(accountID)+"/webhooks").
		WithQuery("page", strconv.Itoa(params.Page)).
		WithQuery("per-page", strconv.Itoa(params.PerPage)).
		WithQuery("event", params.Event)
	if params.Delivered != nil {
		req.WithQuery("delivered", strconv.FormatBool(*params.Delivered))
	}
	if params.From != 0 {
		req.WithQuery("from", strconv.FormatInt(params.From, 10))
	}
	if params.To != 0 {
		req.WithQuery("to", strconv.FormatInt(params.To, 10))
	}

	resp, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return paginated(out, resp), nil
}

// RetryDispatch retries a failed webhook dispatch.
// POST /accounts/{account_id}/webhooks/{dispatch_id}/retry.
func (r *WebhookResource) RetryDispatch(ctx context.Context, accountID, dispatchID string) (*models.WebhookDispatch, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.WebhookDispatch
	path := "/accounts/" + url.PathEscape(accountID) + "/webhooks/" + url.PathEscape(dispatchID) + "/retry"
	_, err := r.http.NewRequest(http.MethodPost, path).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
