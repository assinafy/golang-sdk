package models

// CreateEnvelopeRequest sends processed documents to shared signers. The API
// accepts at most 20 documents and 20 signers; every signer must be on a document.
type CreateEnvelopeRequest struct {
	// Name is the invitation title, 1–255 characters, without '<'.
	Name string `json:"name"`
	// Message is the optional shared invitation message.
	Message *string `json:"message,omitempty"`
	// Signers lists each participant once. Envelope signers have no signing steps.
	Signers []EnvelopeSigner `json:"signers"`
	// Documents lists the documents in their intended display order.
	Documents []EnvelopeDocumentRequest `json:"documents"`
	// MaxTotalCredits limits creation spending; zero is a valid ceiling.
	// Estimates ignore it. A higher cost returns 409 CreditCeilingExceeded.
	MaxTotalCredits *float64 `json:"max_total_credits,omitempty"`
	// IdempotencyKey is sent as the Idempotency-Key header on creation only.
	// Reuse the key after an uncertain response to retrieve the same envelope
	// without charging again. Keys are workspace-scoped and do not expire.
	IdempotencyKey string `json:"-"`
}

// EnvelopeSigner defines verification and notification for one participant.
type EnvelopeSigner struct {
	// ID identifies an existing signer in the workspace.
	ID string `json:"id"`
	// VerificationMethod selects Email, Whatsapp, DigitalCertificate, or IdDocumentFaceMatchBasic.
	VerificationMethod string `json:"verification_method,omitempty"`
	// NotificationMethods contains one compatible channel; omission infers it.
	NotificationMethods []string `json:"notification_methods,omitempty"`
	// Embedded uses an embed session for every document, including later additions.
	Embedded bool `json:"embedded,omitempty"`
}

// EnvelopeDocumentRequest assigns a metadata_ready document with no assignment.
type EnvelopeDocumentRequest struct {
	// ID identifies the uploaded document in the workspace.
	ID string `json:"id"`
	// Method is virtual by default; collect requires Entries.
	Method AssignmentMethod `json:"method,omitempty"`
	// Entries places fields on pages for collect signing. Every SignerID must
	// own a field. Certificate signers require virtual and must sign alone.
	Entries []AssignmentEntry `json:"entries,omitempty"`
	// SignerIDs contains participants from the envelope, each exactly once.
	SignerIDs []string `json:"signer_ids"`
	// ExpiresAt accepts an ISO-8601 string or Unix seconds. Nil, zero, an empty
	// string or omission means no expiration; otherwise allow at least one hour.
	ExpiresAt any `json:"expires_at,omitempty"`
	// CopyReceiverIDs lists signers receiving only the certified copy.
	CopyReceiverIDs []string `json:"copy_receiver_ids,omitempty"`
}

// AddEnvelopeDocumentRequest adds a document for existing envelope signers.
type AddEnvelopeDocumentRequest struct {
	// EnvelopeDocumentRequest supplies the document, participants and placements.
	EnvelopeDocumentRequest
	// MaxTotalCredits limits the additional document's certificate-signature cost.
	MaxTotalCredits *float64 `json:"max_total_credits,omitempty"`
}

// UpdateEnvelopeRequest changes only the supplied fields, without notifications.
type UpdateEnvelopeRequest struct {
	// Name replaces the invitation title, 1–255 characters, without '<'.
	Name *string `json:"name,omitempty"`
	// Message is omitted when nil; an empty Nullable clears the message.
	Message *Nullable[string] `json:"message,omitempty"`
	// DocumentIDs lists every current document once, in the desired order.
	DocumentIDs []string `json:"document_ids,omitzero"`
}

// DocumentEnvelope groups documents that retain individual signing lifecycles.
type DocumentEnvelope struct {
	// Resource is present on single-resource responses.
	Resource string `json:"resource,omitempty"`
	// ID identifies the envelope.
	ID string `json:"id"`
	// Name is the shared invitation title.
	Name string `json:"name"`
	// Message is the nullable shared invitation message.
	Message *string `json:"message"`
	// Documents contains each current document in display order.
	Documents []EnvelopeDocument `json:"documents"`
	// Counts maps document status to count; null means no documents remain.
	Counts map[string]int `json:"counts"`
	// SigningURLs contains one entry per signer; embedded signers have no link.
	SigningURLs []SigningURL `json:"signing_urls"`
	// CreatedAt is the creation date-time.
	CreatedAt Timestamp `json:"created_at"`
}

// EnvelopeDocument is one document in an envelope response.
type EnvelopeDocument struct {
	// ID identifies the document.
	ID string `json:"id"`
	// Position is its one-based display order.
	Position int `json:"position"`
	// Name is the document name.
	Name string `json:"name"`
	// Status is the document's lifecycle state.
	Status DocumentStatus `json:"status"`
	// Artifacts maps artifact names to download URLs. Envelope lists include
	// only thumbnail, whose URL may return 404 until generated.
	Artifacts map[string]string `json:"artifacts"`
	// Signers lists only participants assigned to this document.
	Signers []EnvelopeDocumentSigner `json:"signers"`
}

// EnvelopeDocumentSigner reports one participant's completion on a document.
type EnvelopeDocumentSigner struct {
	// ID identifies the signer.
	ID string `json:"id"`
	// FullName is the signer's display name.
	FullName string `json:"full_name"`
	// Email is the nullable contact address.
	Email *string `json:"email"`
	// WhatsAppPhoneNumber is the nullable WhatsApp contact number.
	WhatsAppPhoneNumber *string `json:"whatsapp_phone_number"`
	// Completed reports whether the signer has signed this document.
	Completed bool `json:"completed"`
	// SignedAt is null until the participant signs.
	SignedAt *Timestamp `json:"signed_at"`
}

// EnvelopeActivity is a document activity annotated with its envelope document.
type EnvelopeActivity struct {
	// DocumentActivity supplies the event, payload, origin and timestamp.
	DocumentActivity
	// DocumentID identifies the document the activity concerns.
	DocumentID string `json:"document_id"`
	// DocumentName is that document's name.
	DocumentName string `json:"document_name"`
}

// EnvelopeCancellation reports the outcome for one document when cancelling.
type EnvelopeCancellation struct {
	// ID identifies the document.
	ID string `json:"id"`
	// Status is the document's lifecycle state after the call.
	Status DocumentStatus `json:"status"`
	// Cancelled is true only when this call cancelled the document.
	Cancelled bool `json:"cancelled"`
}

// EnvelopeResendResult reports whether all notification methods sent a reminder.
type EnvelopeResendResult struct {
	// IsSent is true when every notification method sent the invitation.
	IsSent bool `json:"is_sent"`
	// EnvelopeID identifies the envelope.
	EnvelopeID string `json:"envelope_id"`
	// SignerID identifies the notified participant.
	SignerID string `json:"signer_id"`
}
