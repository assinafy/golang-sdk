package models

// WebhookSubscription models GET /accounts/{id}/webhooks/subscriptions.
type WebhookSubscription struct {
	// Events lists the subscribed event codes from GET /webhooks/event-types.
	Events []string `json:"events"`
	// IsActive reports whether delivery is enabled.
	IsActive bool `json:"is_active"`
	// URL is the nullable URI delivery endpoint.
	URL *string `json:"url,omitempty"`
	// Email is the nullable notification email configured for the subscription.
	Email *string `json:"email,omitempty"`
	// UpdatedAt is the nullable subscription last-update date-time.
	UpdatedAt *Timestamp `json:"updated_at,omitempty"`
}

// UpdateWebhookSubscriptionRequest is the body for
// PUT /accounts/{id}/webhooks/subscriptions. All four fields are required by the
// API, so url and email are always serialized (even when empty) for the server
// to validate.
type UpdateWebhookSubscriptionRequest struct {
	// Events is the required set of event codes from GET /webhooks/event-types.
	Events []string `json:"events"`
	// IsActive is required and enables or disables delivery.
	IsActive bool `json:"is_active"`
	// URL is the required webhook endpoint value; the API validates its format.
	URL string `json:"url"`
	// Email is the required notification email value; the API validates its format.
	Email string `json:"email"`
}

// WebhookEndpoint is a URL that receives the account's webhook events, returned
// by the /accounts/{id}/webhooks/endpoints operations. Every active endpoint
// subscribed to an event receives it independently. An account has one endpoint,
// or up to three on paid plans.
type WebhookEndpoint struct {
	// ID identifies the endpoint in the endpoint, secret and dispatch operations.
	ID string `json:"id"`
	// Name is the nullable label that tells endpoints apart.
	Name *string `json:"name"`
	// URL receives the events over http or https; it is unique per workspace.
	URL string `json:"url"`
	// Email receives delivery-failure notices.
	Email string `json:"email"`
	// Events lists the delivered event codes from GET /webhooks/event-types.
	Events []string `json:"events"`
	// IsActive reports whether events are delivered.
	IsActive bool `json:"is_active"`
	// SigningEnabled reports whether deliveries carry a Standard Webhooks
	// webhook-signature header; see WebhookVerifier.VerifyRequest.
	SigningEnabled bool `json:"signing_enabled"`
	// CreatedAt is the endpoint creation date-time.
	CreatedAt Timestamp `json:"created_at"`
	// UpdatedAt is the endpoint last-update date-time.
	UpdatedAt Timestamp `json:"updated_at"`
}

// CreateWebhookEndpointRequest is the body for
// POST /accounts/{id}/webhooks/endpoints.
type CreateWebhookEndpointRequest struct {
	// URL is required and must differ from every other endpoint of the workspace.
	URL string `json:"url"`
	// Email is the required contact for delivery-failure notices.
	Email string `json:"email"`
	// Events is the required set of event codes from GET /webhooks/event-types.
	Events []string `json:"events"`
	// Name is an optional label.
	Name string `json:"name,omitempty"`
	// IsActive enables delivery; nil lets the API default to true.
	IsActive *bool `json:"is_active,omitempty"`
	// SigningEnabled generates a signing secret and signs every delivery. The API
	// defaults to false.
	SigningEnabled bool `json:"signing_enabled,omitempty"`
}

// UpdateWebhookEndpointRequest is the body for
// PUT /accounts/{id}/webhooks/endpoints/{endpointId}. Only non-nil fields are
// sent and changed.
type UpdateWebhookEndpointRequest struct {
	// URL replaces the delivery URL; it must not match another endpoint's URL.
	URL *string `json:"url,omitempty"`
	// Email replaces the delivery-failure contact.
	Email *string `json:"email,omitempty"`
	// Events replaces the subscribed event codes; nil leaves them unchanged.
	Events []string `json:"events,omitempty"`
	// Name replaces the label.
	Name *string `json:"name,omitempty"`
	// IsActive enables or pauses delivery.
	IsActive *bool `json:"is_active,omitempty"`
	// SigningEnabled true generates a secret when the endpoint has none and keeps
	// the current one otherwise; false discards the secret.
	SigningEnabled *bool `json:"signing_enabled,omitempty"`
}

// WebhookDispatch is a single delivery attempt entry from
// GET /accounts/{id}/webhooks.
type WebhookDispatch struct {
	// Resource is the API resource discriminator when present.
	Resource string `json:"resource,omitempty"`
	// ID is the dispatch identifier used by the retry endpoint.
	ID string `json:"id"`
	// Event is the machine-readable webhook event code.
	Event string `json:"event"`
	// ActivityID is the source activity identifier.
	ActivityID int `json:"activity_id"`
	// EndpointID identifies the WebhookEndpoint the delivery was sent to; nil
	// once that endpoint is deleted.
	EndpointID *string `json:"endpoint_id,omitempty"`
	// Endpoint is the nullable URL to which delivery was attempted.
	Endpoint *string `json:"endpoint,omitempty"`
	// Payload contains event-specific JSON data.
	Payload Payload `json:"payload,omitempty"`
	// Delivered reports whether the attempt succeeded.
	Delivered bool `json:"delivered"`
	// HTTPStatus is the nullable response status from the subscriber endpoint.
	HTTPStatus *int `json:"http_status,omitempty"`
	// ResponseBody is the nullable subscriber response captured by Assinafy.
	ResponseBody *string `json:"response_body,omitempty"`
	// Error is the nullable delivery error reported by Assinafy.
	Error *string `json:"error,omitempty"`
	// CreatedAt is the dispatch creation date-time.
	CreatedAt Timestamp `json:"created_at"`
	// UpdatedAt is the dispatch last-update date-time.
	UpdatedAt Timestamp `json:"updated_at"`
}

// WebhookDispatchListParams are the query parameters for GET /accounts/{id}/webhooks.
type WebhookDispatchListParams struct {
	// Page is one-based; values less than one default to 1.
	Page int
	// PerPage defaults to 20 and is clamped to the API maximum of 100.
	PerPage int
	// Delivered optionally filters by success or failure; nil applies no filter.
	Delivered *bool
	// EndpointID optionally limits results to deliveries sent to one WebhookEndpoint.
	EndpointID string
	// Event optionally filters by a webhook event code.
	Event string
	// From optionally filters for entries after this Unix timestamp.
	From int64
	// To optionally filters for entries before this Unix timestamp.
	To int64
}

// SetDefaults normalises Page/PerPage and clamps PerPage to the API maximum.
func (p *WebhookDispatchListParams) SetDefaults() {
	if p.Page <= 0 {
		p.Page = defaultPage
	}
	if p.PerPage <= 0 {
		p.PerPage = defaultWebhookPerPage
	}
	if p.PerPage > maxPerPage {
		p.PerPage = maxPerPage
	}
}

// WebhookEventType is one entry of GET /webhooks/event-types.
type WebhookEventType struct {
	// ID is the event code used in subscription and dispatch filters.
	ID string `json:"id"`
	// Description is the event's human-readable label.
	Description string `json:"description"`
}

// WebhookPayload is the JSON envelope delivered to webhook subscribers.
type WebhookPayload struct {
	// ID is the source activity identifier.
	ID int `json:"id"`
	// Event is the machine-readable webhook event code.
	Event string `json:"event"`
	// Message is an optional human-readable event description.
	Message *string `json:"message,omitempty"`
	// Payload contains event-specific JSON data.
	Payload Payload `json:"payload,omitempty"`
	// Origin is nullable network context for the source request.
	Origin *RequestOrigin `json:"origin,omitempty"`
	// CreatedAt is the event creation time as Unix seconds. Timestamp preserves
	// the numeric value in base-10 string form.
	CreatedAt Timestamp `json:"created_at"`
	// Subject is the optional event actor or subject object.
	Subject map[string]any `json:"subject,omitempty"`
	// Object is the optional event target object.
	Object map[string]any `json:"object,omitempty"`
	// AccountID identifies the account that emitted the event.
	AccountID string `json:"account_id"`
}
