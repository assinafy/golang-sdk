package models

// EnvelopeReference identifies the envelope containing a document.
type EnvelopeReference struct {
	// ID is the envelope identifier.
	ID string `json:"id"`
	// Name is the envelope display name.
	Name string `json:"name"`
}

// DocumentGetParams selects optional workspace document detail expansions.
type DocumentGetParams struct {
	// Expand is a comma-separated selection of envelope_id and envelope.
	Expand string
}

// SigningDocumentParams selects a signer-facing document and its expansions.
type SigningDocumentParams struct {
	// Expand accepts envelope to include its identifier and name.
	Expand string
	// DocumentID selects another document in the code's envelope on GET /sign.
	DocumentID string
	// HasAcceptedTerms records terms acceptance on GET /sign when supplied.
	HasAcceptedTerms *bool
}

// CertifiedCopyStatus describes whether a signer can request another certified copy.
type CertifiedCopyStatus struct {
	// Available reports whether another copy can be requested.
	Available bool `json:"available"`
	// NotificationMethod is Email or Whatsapp, or nil when neither is available.
	NotificationMethod *string `json:"notification_method"`
	// ResendsLeft is the remaining WhatsApp resend count; nil for email.
	ResendsLeft *int `json:"resends_left"`
}

// CertifiedCopyRequest optionally selects a document from the signer's envelope.
type CertifiedCopyRequest struct {
	// DocumentID selects a document the signer is on within the same envelope.
	DocumentID string `json:"document_id,omitempty"`
}

// CertifiedCopyResult describes the copy delivery and remaining availability.
type CertifiedCopyResult struct {
	CertifiedCopyStatus
	// SentTo is the masked original signing contact used for delivery.
	SentTo string `json:"sent_to"`
}
