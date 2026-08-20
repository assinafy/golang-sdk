package resources

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// WebhookResource exposes authenticated webhook subscription and delivery-history
// endpoints. Its methods follow the package-level account and error contract.
type WebhookResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewWebhookResource constructs a WebhookResource.
func NewWebhookResource(httpClient *internal.HTTPClient, accountID string) *WebhookResource {
	return &WebhookResource{http: httpClient, accountID: accountID}
}

// UpdateSubscription sends all fields of UpdateWebhookSubscriptionRequest and
// returns the resulting WebhookSubscription. It requires client authentication,
// uses the configured default for an empty accountID, and changes future delivery.
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

// GetSubscription returns the authenticated account's WebhookSubscription. An
// empty accountID uses the configured default.
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

// Inactivate disables the workspace webhook subscription. This is the supported
// way to stop receiving webhook deliveries; the DELETE subscriptions route the
// docs mention in passing is not implemented by the live API (returns 404).
// It requires client authentication, returns the updated WebhookSubscription,
// and uses the configured default for an empty accountID.
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

// ListEventTypes uses client authentication and returns the available
// WebhookEventType payloads.
// GET /webhooks/event-types.
func (r *WebhookResource) ListEventTypes(ctx context.Context) ([]models.WebhookEventType, error) {
	var out []models.WebhookEventType
	_, err := r.http.NewRequest(http.MethodGet, "/webhooks/event-types").Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListDispatches returns WebhookDispatch payloads with X-Pagination metadata,
// filtered by optional WebhookDispatchListParams. It requires client
// authentication and uses the configured default for an empty accountID.
// GET /accounts/{account_id}/webhooks.
func (r *WebhookResource) ListDispatches(ctx context.Context, accountID string, params *models.WebhookDispatchListParams) (*models.PaginatedResult[models.WebhookDispatch], error) {
	accountID = resolveAccountID(accountID, r.accountID)
	p := models.WebhookDispatchListParams{}
	if params != nil {
		p = *params
	}
	p.SetDefaults()

	var out []models.WebhookDispatch
	req := r.http.NewRequest(http.MethodGet, "/accounts/"+url.PathEscape(accountID)+"/webhooks").
		WithQuery("page", strconv.Itoa(p.Page)).
		WithQuery("per-page", strconv.Itoa(p.PerPage))
	if p.Event != "" {
		req.WithQuery("event", p.Event)
	}
	if p.Delivered != nil {
		req.WithQuery("delivered", strconv.FormatBool(*p.Delivered))
	}
	if p.From != 0 {
		req.WithQuery("from", strconv.FormatInt(p.From, 10))
	}
	if p.To != 0 {
		req.WithQuery("to", strconv.FormatInt(p.To, 10))
	}

	resp, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return paginated(out, resp), nil
}

// RetryDispatch uses client authentication, creates a new delivery attempt for a
// failed dispatch, and returns the resulting WebhookDispatch. An empty accountID
// uses the configured default; an ineligible dispatch produces an API error.
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
