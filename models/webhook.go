package models

// WebhookSubscription models GET /accounts/{id}/webhooks/subscriptions.
type WebhookSubscription struct {
	Events    []string   `json:"events"`
	IsActive  bool       `json:"is_active"`
	URL       *string    `json:"url,omitempty"`
	Email     *string    `json:"email,omitempty"`
	UpdatedAt *Timestamp `json:"updated_at,omitempty"`
}

// UpdateWebhookSubscriptionRequest is the body for
// PUT /accounts/{id}/webhooks/subscriptions. All four fields are required by the
// API, so url and email are always serialized (even when empty) for the server
// to validate.
type UpdateWebhookSubscriptionRequest struct {
	Events   []string `json:"events"`
	IsActive bool     `json:"is_active"`
	URL      string   `json:"url"`
	Email    string   `json:"email"`
}

// WebhookDispatch is a single delivery attempt entry from
// GET /accounts/{id}/webhooks.
type WebhookDispatch struct {
	Resource     string    `json:"resource,omitempty"`
	ID           string    `json:"id"`
	Event        string    `json:"event"`
	ActivityID   int       `json:"activity_id"`
	Endpoint     *string   `json:"endpoint,omitempty"`
	Payload      Payload   `json:"payload,omitempty"`
	Delivered    bool      `json:"delivered"`
	HTTPStatus   *int      `json:"http_status,omitempty"`
	ResponseBody *string   `json:"response_body,omitempty"`
	Error        *string   `json:"error,omitempty"`
	CreatedAt    Timestamp `json:"created_at"`
	UpdatedAt    Timestamp `json:"updated_at"`
}

// WebhookDispatchListParams are the query parameters for GET /accounts/{id}/webhooks.
type WebhookDispatchListParams struct {
	Page      int
	PerPage   int
	Delivered *bool
	Event     string
	From      int64
	To        int64
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
	ID          string `json:"id"`
	Description string `json:"description"`
}

// WebhookPayload is the JSON envelope delivered to webhook subscribers.
type WebhookPayload struct {
	ID        int            `json:"id"`
	Event     string         `json:"event"`
	Message   *string        `json:"message,omitempty"`
	Payload   Payload        `json:"payload,omitempty"`
	Origin    *RequestOrigin `json:"origin,omitempty"`
	CreatedAt Timestamp      `json:"created_at"`
	Subject   map[string]any `json:"subject,omitempty"`
	Object    map[string]any `json:"object,omitempty"`
	AccountID string         `json:"account_id"`
}
