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

// UpdateSubscription changes the account's oldest WebhookEndpoint, creating it
// when the account has none. Accounts with several endpoints use UpdateEndpoint.
// It sends all fields of UpdateWebhookSubscriptionRequest and
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

// GetSubscription returns the account's oldest WebhookEndpoint as a
// WebhookSubscription. Accounts with several endpoints use ListEndpoints. An
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

// Inactivate disables the account webhook subscription. It requires client
// authentication, returns the updated WebhookSubscription, and uses the
// configured default for an empty accountID.
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
// filtered by optional WebhookDispatchListParams, including EndpointID. It requires client
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
	req.WithQuery("event", p.Event).WithQuery("endpoint_id", p.EndpointID)
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

func (r *WebhookResource) endpointsPath(accountID string) string {
	return "/accounts/" + url.PathEscape(resolveAccountID(accountID, r.accountID)) + "/webhooks/endpoints"
}

// endpointPath returns one endpoint's path followed by suffix. An empty
// endpointID leaves an empty segment, which the transport rejects as
// errors.ErrInvalidInput.
func (r *WebhookResource) endpointPath(accountID, endpointID, suffix string) string {
	return r.endpointsPath(accountID) + "/" + url.PathEscape(endpointID) + suffix
}

// ListEndpoints returns every WebhookEndpoint of the account, oldest first. It
// requires the account:read scope for OAuth clients and uses the configured
// default for an empty accountID.
// GET /accounts/{account_id}/webhooks/endpoints.
func (r *WebhookResource) ListEndpoints(ctx context.Context, accountID string) ([]models.WebhookEndpoint, error) {
	var out []models.WebhookEndpoint
	if _, err := r.http.NewRequest(http.MethodGet, r.endpointsPath(accountID)).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateEndpoint registers a WebhookEndpoint and returns it. Accounts hold one
// endpoint, or three on paid plans; one past the limit is a 403 APIError and a
// URL already used by the workspace is a 400. With SigningEnabled, read the new
// secret with GetEndpointSecret. OAuth clients need the webhooks:write scope.
// POST /accounts/{account_id}/webhooks/endpoints.
func (r *WebhookResource) CreateEndpoint(ctx context.Context, accountID string, body *models.CreateWebhookEndpointRequest) (*models.WebhookEndpoint, error) {
	return r.endpoint(ctx, http.MethodPost, r.endpointsPath(accountID), body)
}

// GetEndpoint returns one WebhookEndpoint. OAuth clients need the account:read
// scope.
// GET /accounts/{account_id}/webhooks/endpoints/{endpoint_id}.
func (r *WebhookResource) GetEndpoint(ctx context.Context, accountID, endpointID string) (*models.WebhookEndpoint, error) {
	return r.endpoint(ctx, http.MethodGet, r.endpointPath(accountID, endpointID, ""), nil)
}

// UpdateEndpoint changes only the non-nil fields of the request and returns the
// updated WebhookEndpoint. OAuth clients need the webhooks:write scope.
// PUT /accounts/{account_id}/webhooks/endpoints/{endpoint_id}.
func (r *WebhookResource) UpdateEndpoint(ctx context.Context, accountID, endpointID string, body *models.UpdateWebhookEndpointRequest) (*models.WebhookEndpoint, error) {
	return r.endpoint(ctx, http.MethodPut, r.endpointPath(accountID, endpointID, ""), body)
}

// DeleteEndpoint stops delivery to an endpoint and frees its slot. Its dispatch
// history remains with a nil EndpointID. OAuth clients need the webhooks:write
// scope.
// DELETE /accounts/{account_id}/webhooks/endpoints/{endpoint_id}.
func (r *WebhookResource) DeleteEndpoint(ctx context.Context, accountID, endpointID string) error {
	_, err := r.http.NewRequest(http.MethodDelete, r.endpointPath(accountID, endpointID, "")).Execute(ctx, nil)
	return err
}

// GetEndpointSecret returns the endpoint's Standard Webhooks signing secret,
// "whsec_" followed by the base64 key, for NewWebhookVerifier. Signing must be
// enabled (otherwise a 400 APIError). API key or user token only; OAuth
// applications cannot read secrets.
// GET /accounts/{account_id}/webhooks/endpoints/{endpoint_id}/secret.
func (r *WebhookResource) GetEndpointSecret(ctx context.Context, accountID, endpointID string) (string, error) {
	return r.secret(ctx, http.MethodGet, r.endpointPath(accountID, endpointID, "/secret"))
}

// RotateEndpointSecret replaces the endpoint's signing secret and returns the
// new one. The old secret stops working immediately, so deploy the new secret to
// the receiver at once. Signing must be enabled. API key or user token only.
// POST /accounts/{account_id}/webhooks/endpoints/{endpoint_id}/secret/rotate.
func (r *WebhookResource) RotateEndpointSecret(ctx context.Context, accountID, endpointID string) (string, error) {
	return r.secret(ctx, http.MethodPost, r.endpointPath(accountID, endpointID, "/secret/rotate"))
}

func (r *WebhookResource) endpoint(ctx context.Context, method, path string, body any) (*models.WebhookEndpoint, error) {
	var out models.WebhookEndpoint
	if _, err := r.http.NewRequest(method, path).WithBody(body).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *WebhookResource) secret(ctx context.Context, method, path string) (string, error) {
	var out struct {
		Secret string `json:"secret"`
	}
	if _, err := r.http.NewRequest(method, path).Execute(ctx, &out); err != nil {
		return "", err
	}
	return out.Secret, nil
}
