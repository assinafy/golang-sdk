package models

type WebhookSubscription struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	Email     string   `json:"email"`
	Events    []string `json:"events"`
	IsActive  bool     `json:"is_active"`
	Secret    string   `json:"secret,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
	UpdatedAt string   `json:"updated_at,omitempty"`
}

type RegisterWebhookRequest struct {
	URL    string   `json:"url"`
	Email  string   `json:"email"`
	Events []string `json:"events"`
}

type WebhookDispatch struct {
	ID          string  `json:"id"`
	WebhookID   string  `json:"webhook_id"`
	Event       string  `json:"event"`
	Status      string  `json:"status"`
	Attempts    int     `json:"attempts"`
	Response    string  `json:"response,omitempty"`
	CreatedAt   string  `json:"created_at"`
	DeliveredAt *string `json:"delivered_at,omitempty"`
}

type WebhookDispatchListParams struct {
	Page      int    `json:"page,omitempty"`
	PerPage   int    `json:"per-page,omitempty"`
	Delivered *bool  `json:"delivered,omitempty"`
	Event     string `json:"event,omitempty"`
}

type WebhookEventType struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type WebhookPayload struct {
	Event     string                 `json:"event"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}
