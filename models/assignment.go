// Package models contains the request and response types for the Assinafy API.
package models

// AssignmentMethod selects between virtual signing without input fields and
// collect signing with input fields placed on document pages.
type AssignmentMethod string

// Documented values for AssignmentMethod.
const (
	// MethodVirtual creates a signing flow without input fields.
	MethodVirtual AssignmentMethod = "virtual"
	// MethodCollect creates a signing flow with input fields on specific pages.
	MethodCollect AssignmentMethod = "collect"
)

// Assignment models the response of POST /documents/{id}/assignments and the
// embedded `assignment` field on document responses.
type Assignment struct {
	// Resource is the API resource discriminator when present.
	Resource string `json:"resource,omitempty"`
	// ID is the assignment identifier.
	ID string `json:"id"`
	// DocumentID is the assigned document identifier when included by the endpoint.
	DocumentID string `json:"document_id,omitempty"`
	// SenderEmail is the email address of the user who sent the assignment.
	SenderEmail string `json:"sender_email,omitempty"`
	// Method is the assignment flow, "virtual" or "collect".
	Method AssignmentMethod `json:"method"`
	// Status is the API assignment status code when included.
	Status string `json:"status,omitempty"`
	// Expiration is the legacy nullable expiration value returned by some endpoints.
	Expiration *string `json:"expiration,omitempty"`
	// ExpiresAt is the nullable ISO-8601 expiration date/time.
	ExpiresAt *string `json:"expires_at,omitempty"`
	// Message is the optional invitation message, or nil when none was supplied.
	Message *string `json:"message,omitempty"`
	// Signers contains the assignment's signing participants.
	Signers []Signer `json:"signers,omitempty"`
	// CopyReceivers contains recipients copied on the signing process.
	CopyReceivers []Signer `json:"copy_receivers,omitempty"`
	// Items contains field placements and their current values.
	Items []AssignmentItem `json:"items,omitempty"`
	// Summary contains aggregate signer completion data when returned.
	Summary *AssignmentSummary `json:"summary,omitempty"`
	// SigningURLs contains one signer-specific URL per virtual-flow signer.
	SigningURLs []SigningURL `json:"signing_urls,omitempty"`
	// CompletedAt is the nullable assignment-completion date-time.
	CompletedAt *Timestamp `json:"completed_at,omitempty"`
	// CreatedAt is the creation date-time when included.
	CreatedAt Timestamp `json:"created_at,omitempty"`
	// UpdatedAt is the last-update date-time when included.
	UpdatedAt Timestamp `json:"updated_at,omitempty"`
}

// AssignmentItem is a single signer/field/page tuple inside an assignment.
type AssignmentItem struct {
	// ID is the assignment-item identifier used when submitting a signature value.
	ID string `json:"id"`
	// Page identifies the document page containing this item, or nil if omitted.
	Page *DocumentPage `json:"page,omitempty"`
	// Signer is the participant responsible for the item, or nil if omitted.
	Signer *Signer `json:"signer,omitempty"`
	// Field is the field definition for this item, or nil if omitted.
	Field *FieldDefinition `json:"field,omitempty"`
	// DisplaySettings is the endpoint-provided placement object; it can be
	// decoded into DisplaySettings when the standard shape is returned.
	DisplaySettings any `json:"display_settings,omitempty"`
	// Value is the field's nullable JSON value.
	Value any `json:"value,omitempty"`
	// Completed reports whether the signer has completed this item.
	Completed bool `json:"completed"`
}

// AssignmentSummary is the aggregate completion summary embedded in Assignment.
type AssignmentSummary struct {
	// SignerCount is the total number of assignment signers.
	SignerCount int `json:"signer_count"`
	// CompletedCount is the number of signers who completed their assignments.
	CompletedCount int `json:"completed_count"`
	// Signers contains the signer records represented by the counts.
	Signers []Signer `json:"signers"`
}

// SigningURL is one entry of the signing_urls array returned for virtual assignments.
type SigningURL struct {
	// SignerID identifies the signer for whom URL was issued.
	SignerID string `json:"signer_id"`
	// URL is the signer-specific absolute signing URI.
	URL string `json:"url"`
}

// CreateAssignmentRequest is the body for POST /documents/{id}/assignments.
type CreateAssignmentRequest struct {
	// Method is required and must be MethodVirtual or MethodCollect.
	Method AssignmentMethod `json:"method,omitempty"`
	// Signers identifies required participants and their delivery settings.
	Signers []SignerReference `json:"signers,omitempty"`
	// SignerIDs is the legacy alternative to Signers for signer UUIDs.
	SignerIDs []string `json:"signer_ids,omitempty"`
	// Entries positions fields by page and is conditionally required for collect assignments.
	Entries []AssignmentEntry `json:"entries,omitempty"`
	// Message is an optional invitation message; nil omits it.
	Message *string `json:"message,omitempty"`
	// Expiration is the optional legacy expiration value; nil omits it.
	Expiration *string `json:"expiration,omitempty"`
	// ExpiresAt is an optional ISO-8601 expiration date/time; nil omits it.
	ExpiresAt *string `json:"expires_at,omitempty"`
	// CopyReceivers is an optional list of signer UUIDs copied on completion.
	CopyReceivers []string `json:"copy_receivers,omitempty"`
}

// EstimateAssignmentCostRequest is the body for
// POST /documents/{id}/assignments/estimate-cost. Unlike assignment creation,
// cost estimation does not accept signer IDs or invitation metadata.
type EstimateAssignmentCostRequest struct {
	// Method selects the "virtual" or "collect" flow being estimated.
	Method AssignmentMethod `json:"method,omitempty"`
	// Signers describes the prospective signers' verification and delivery channels.
	Signers []EstimateAssignmentCostSigner `json:"signers,omitempty"`
	// Entries supplies the API's open object entries for a collect-flow estimate.
	Entries []any `json:"entries,omitempty"`
}

// EstimateAssignmentCostSigner describes verification and notification costs
// for one prospective signer. An empty object selects the API's Email defaults.
type EstimateAssignmentCostSigner struct {
	// VerificationMethod is optionally "Email", "Whatsapp", or
	// "DigitalCertificate"; an empty value selects the Email default.
	VerificationMethod string `json:"verification_method,omitempty"`
	// NotificationMethods optionally contains "Email" or "Whatsapp"; an empty
	// slice selects the Email default.
	NotificationMethods []string `json:"notification_methods,omitempty"`
}

// ResetAssignmentExpirationRequest is the body for
// PUT /documents/{id}/assignments/{aid}/reset-expiration.
type ResetAssignmentExpirationRequest struct {
	// ExpiresAt is an optional ISO-8601 expiration date/time. Nil omits the
	// property and lets the API apply its default.
	ExpiresAt *string `json:"expires_at,omitempty"`
}

// AssignmentEntry positions a set of fields on a specific document page.
type AssignmentEntry struct {
	// PageID identifies the document page receiving the fields.
	PageID string `json:"page_id"`
	// Fields contains the placements on that page.
	Fields []AssignmentField `json:"fields"`
}

// AssignmentField pairs a signer with a field definition and visual placement.
type AssignmentField struct {
	// SignerID identifies the signer responsible for this field.
	SignerID string `json:"signer_id"`
	// FieldID identifies the field definition being placed.
	FieldID string `json:"field_id"`
	// DisplaySettings is the optional placement object, commonly DisplaySettings.
	DisplaySettings any `json:"display_settings,omitempty"`
}

// CostEstimate is shared by assignment estimate-cost endpoints.
type CostEstimate struct {
	// Documents is the number of document units consumed by assignment creation.
	Documents float64 `json:"documents,omitempty"`
	// Credits is the number of notification or verification credits consumed.
	Credits float64 `json:"credits,omitempty"`
	// NeedsExtraDocument reports whether an additional document unit is required.
	NeedsExtraDocument bool `json:"needs_extra_document,omitempty"`
	// ExtraDocumentCost is the additional cost for the required document unit.
	ExtraDocumentCost float64 `json:"extra_document_cost,omitempty"`
	// TotalCredits is the total credit cost across the estimate breakdown.
	TotalCredits float64 `json:"total_credits,omitempty"`
	// Breakdown itemizes the operation's priced components.
	Breakdown []CostBreakdownItem `json:"breakdown,omitempty"`
	// DocumentBalance is the account's available document-unit balance.
	DocumentBalance float64 `json:"document_balance,omitempty"`
	// CreditBalance is the account's available credit balance.
	CreditBalance float64 `json:"credit_balance,omitempty"`
	// HasSufficientResources reports whether both balances cover the operation.
	HasSufficientResources bool `json:"has_sufficient_resources,omitempty"`
	// BlockingReason is set when the operation cannot proceed (e.g. PendingPayment,
	// InsufficientDocuments, InsufficientCredits) and is null otherwise.
	BlockingReason *string `json:"blocking_reason"`
	// Message is the human-readable explanation for the current block state, or null.
	Message *string `json:"message"`
	// Total is the resend endpoint's total credit cost.
	Total float64 `json:"total,omitempty"`
	// HasSufficientCredits reports whether the resend can be funded.
	HasSufficientCredits bool `json:"has_sufficient_credits,omitempty"`
}

// CostBreakdownItem is a single line in CostEstimate.Breakdown.
type CostBreakdownItem struct {
	// Code is the machine-readable priced-component code.
	Code string `json:"code"`
	// Name is the human-readable priced-component label.
	Name string `json:"name"`
	// Cost is the total cost for this line.
	Cost float64 `json:"cost"`
	// Quantity is the number of charged units when the endpoint itemizes it.
	Quantity float64 `json:"quantity,omitempty"`
	// UnitCost is the cost per unit when the endpoint itemizes it.
	UnitCost float64 `json:"unit_cost,omitempty"`
}

// ResendNotificationResult is returned by the assignment resend endpoint.
type ResendNotificationResult struct {
	// IsSent reports whether the replacement invitation was dispatched.
	IsSent bool `json:"is_sent"`
	// DocumentID identifies the affected document.
	DocumentID string `json:"document_id"`
	// SignerID identifies the notified signer.
	SignerID string `json:"signer_id"`
}

// SignDocumentItem is one filled field submitted to POST .../assignments/{aid}.
type SignDocumentItem struct {
	// ItemID is the required assignment-item identifier.
	ItemID string `json:"itemId"`
	// FieldID is the required field-definition identifier.
	FieldID string `json:"fieldId"`
	// PageID is the required document-page identifier.
	PageID string `json:"pageId"`
	// Value is the required field value submitted by the signer.
	Value string `json:"value"`
}

// WhatsAppNotification is a row from
// GET /documents/{id}/assignments/{aid}/whatsapp-notifications.
type WhatsAppNotification struct {
	// SentAt is the Unix time at which the delivery was attempted.
	SentAt Timestamp `json:"sent_at"`
	// Header is the rendered WhatsApp message header.
	Header string `json:"header"`
	// Body is the rendered WhatsApp message body.
	Body string `json:"body"`
	// Buttons contains the rendered call-to-action buttons.
	Buttons []WhatsAppButton `json:"buttons"`
	// PhoneNumber is the destination telephone number returned by the API.
	PhoneNumber string `json:"phone_number"`
	// SignerID identifies the destination signer.
	SignerID string `json:"signer_id"`
}

// WhatsAppButton is one of the call-to-action buttons in a WhatsAppNotification.
type WhatsAppButton struct {
	// Text is the button label.
	Text string `json:"text"`
	// URL is the nullable target URL; nil denotes a non-link button.
	URL *string `json:"url,omitempty"`
}
