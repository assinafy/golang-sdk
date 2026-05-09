// Package models contains the request and response types for the Assinafy API.
package models

// AssignmentMethod selects between virtual (sender places fields) and collect
// (signer fills inputs themselves) flows.
type AssignmentMethod string

// Documented values for AssignmentMethod.
const (
	MethodVirtual AssignmentMethod = "virtual"
	MethodCollect AssignmentMethod = "collect"
)

// Assignment models the response of POST /documents/{id}/assignments and the
// embedded `assignment` field on document responses.
type Assignment struct {
	Resource      string             `json:"resource,omitempty"`
	ID            string             `json:"id"`
	DocumentID    string             `json:"document_id,omitempty"`
	SenderEmail   string             `json:"sender_email,omitempty"`
	Method        AssignmentMethod   `json:"method"`
	Status        string             `json:"status,omitempty"`
	Expiration    *string            `json:"expiration,omitempty"`
	ExpiresAt     *string            `json:"expires_at,omitempty"`
	Message       *string            `json:"message,omitempty"`
	Signers       []Signer           `json:"signers,omitempty"`
	CopyReceivers []Signer           `json:"copy_receivers,omitempty"`
	Items         []AssignmentItem   `json:"items,omitempty"`
	Summary       *AssignmentSummary `json:"summary,omitempty"`
	SigningURLs   []SigningURL       `json:"signing_urls,omitempty"`
	CompletedAt   *Timestamp         `json:"completed_at,omitempty"`
	CreatedAt     Timestamp          `json:"created_at,omitempty"`
	UpdatedAt     Timestamp          `json:"updated_at,omitempty"`
}

// AssignmentItem is a single signer/field/page tuple inside an assignment.
type AssignmentItem struct {
	ID              string           `json:"id"`
	Page            *DocumentPage    `json:"page,omitempty"`
	Signer          *Signer          `json:"signer,omitempty"`
	Field           *FieldDefinition `json:"field,omitempty"`
	DisplaySettings any              `json:"display_settings,omitempty"`
	Value           any              `json:"value,omitempty"`
	Completed       bool             `json:"completed"`
}

// AssignmentSummary is the aggregate completion summary embedded in Assignment.
type AssignmentSummary struct {
	SignerCount    int      `json:"signer_count"`
	CompletedCount int      `json:"completed_count"`
	Signers        []Signer `json:"signers"`
}

// SigningURL is one entry of the signing_urls array returned for virtual assignments.
type SigningURL struct {
	SignerID string `json:"signer_id"`
	URL      string `json:"url"`
}

// CreateAssignmentRequest is the body for POST /documents/{id}/assignments.
type CreateAssignmentRequest struct {
	Method        AssignmentMethod  `json:"method,omitempty"`
	Signers       []SignerReference `json:"signers,omitempty"`
	SignerIDs     []string          `json:"signer_ids,omitempty"`
	Entries       []AssignmentEntry `json:"entries,omitempty"`
	Message       *string           `json:"message,omitempty"`
	Expiration    *string           `json:"expiration,omitempty"`
	ExpiresAt     *string           `json:"expires_at,omitempty"`
	CopyReceivers []string          `json:"copy_receivers,omitempty"`
}

// AssignmentEntry positions a set of fields on a specific document page.
type AssignmentEntry struct {
	PageID string            `json:"page_id"`
	Fields []AssignmentField `json:"fields"`
}

// AssignmentField pairs a signer with a field definition and visual placement.
type AssignmentField struct {
	SignerID        string `json:"signer_id"`
	FieldID         string `json:"field_id"`
	DisplaySettings any    `json:"display_settings,omitempty"`
}

// CostEstimate is shared by assignment estimate-cost endpoints.
type CostEstimate struct {
	Documents              float64             `json:"documents,omitempty"`
	Credits                float64             `json:"credits,omitempty"`
	NeedsExtraDocument     bool                `json:"needs_extra_document,omitempty"`
	ExtraDocumentCost      float64             `json:"extra_document_cost,omitempty"`
	TotalCredits           float64             `json:"total_credits,omitempty"`
	Breakdown              []CostBreakdownItem `json:"breakdown,omitempty"`
	DocumentBalance        float64             `json:"document_balance,omitempty"`
	CreditBalance          float64             `json:"credit_balance,omitempty"`
	HasSufficientResources bool                `json:"has_sufficient_resources,omitempty"`
	Total                  float64             `json:"total,omitempty"`
	HasSufficientCredits   bool                `json:"has_sufficient_credits,omitempty"`
	Description            string              `json:"description,omitempty"`
	TotalCost              float64             `json:"total_cost,omitempty"`
}

// CostBreakdownItem is a single line in CostEstimate.Breakdown.
type CostBreakdownItem struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Cost     float64 `json:"cost"`
	Quantity float64 `json:"quantity,omitempty"`
	UnitCost float64 `json:"unit_cost,omitempty"`
}

// ResendNotificationResult is returned by the assignment resend endpoint.
type ResendNotificationResult struct {
	IsSent     bool   `json:"is_sent"`
	DocumentID string `json:"document_id"`
	SignerID   string `json:"signer_id"`
}

// SignDocumentItem is one filled field submitted to POST .../assignments/{aid}.
type SignDocumentItem struct {
	ItemID  string `json:"itemId"`
	FieldID string `json:"fieldId"`
	PageID  string `json:"pageId"`
	Value   string `json:"value"`
}

// WhatsAppNotification is a row from
// GET /documents/{id}/assignments/{aid}/whatsapp-notifications.
type WhatsAppNotification struct {
	SentAt      Timestamp        `json:"sent_at"`
	Header      string           `json:"header"`
	Body        string           `json:"body"`
	Buttons     []WhatsAppButton `json:"buttons"`
	PhoneNumber string           `json:"phone_number"`
	SignerID    string           `json:"signer_id"`
}

// WhatsAppButton is one of the call-to-action buttons in a WhatsAppNotification.
type WhatsAppButton struct {
	Text string  `json:"text"`
	URL  *string `json:"url,omitempty"`
}
