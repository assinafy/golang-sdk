package models

// ScheduledSend is an assignment or envelope request saved for later delivery.
// Scheduling reserves neither documents nor credits; execution rechecks both.
type ScheduledSend struct {
	// Resource is scheduled_send when included.
	Resource string `json:"resource,omitempty"`
	// ID identifies the scheduled send.
	ID string `json:"id"`
	// Type is assignment or envelope.
	Type string `json:"type"`
	// DocumentIDs lists the documents; deleted documents are omitted.
	DocumentIDs []string `json:"document_ids"`
	// Status is pending, processing, sent, cancelled or failed.
	Status string `json:"status"`
	// SendAt is the scheduled execution time.
	SendAt Timestamp `json:"send_at"`
	// AssignmentID identifies the created assignment once sent, or is null.
	AssignmentID *string `json:"assignment_id"`
	// EnvelopeID identifies the created envelope, or is null if absent or deleted.
	EnvelopeID *string `json:"envelope_id"`
	// FailureReason contains the Portuguese refusal message once failed, or null.
	FailureReason *string `json:"failure_reason"`
	// CreatedAt is when the request was scheduled.
	CreatedAt Timestamp `json:"created_at"`
	// ProcessedAt is when it was sent, failed or cancelled, or null while pending.
	ProcessedAt *Timestamp `json:"processed_at"`
}

// ScheduleAssignmentRequest saves an assignment request for later execution.
// Each signer must belong to the workspace; a document permits one pending send.
type ScheduleAssignmentRequest struct {
	// CreateAssignmentRequest supplies the signing and spending settings.
	// An omitted Method defaults to virtual; ExpiresAt, if set, must be at
	// least two hours after SendAt.
	CreateAssignmentRequest
	// SendAt is an ISO-8601 string or integer Unix seconds, 1 hour to 90 days ahead.
	SendAt any `json:"send_at"`
}

// ScheduleEnvelopeRequest saves an envelope request without creating or charging it.
type ScheduleEnvelopeRequest struct {
	// CreateEnvelopeRequest supplies documents, signers and MaxTotalCredits.
	// IdempotencyKey is ignored: each call schedules a separate send.
	CreateEnvelopeRequest
	// SendAt is an ISO-8601 string or integer Unix seconds, 1 hour to 90 days ahead.
	// Each document expiration must be at least two hours after this time.
	SendAt any `json:"send_at"`
}

// ScheduledSendListParams selects a page of scheduled sends, newest first.
type ScheduledSendListParams struct {
	// Page is one-based; zero uses the API default.
	Page int
	// PerPage is the page size; zero uses the API default of 20.
	PerPage int
	// DocumentID limits results to sends containing this document.
	DocumentID string
}
