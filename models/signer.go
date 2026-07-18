package models

// Signer is the canonical signer object.
type Signer struct {
	// Base fields, returned by every account-scoped signer endpoint.
	Resource            string  `json:"resource,omitempty"`
	ID                  string  `json:"id"`
	FullName            string  `json:"full_name"`
	Email               *string `json:"email,omitempty"`
	WhatsAppPhoneNumber *string `json:"whatsapp_phone_number,omitempty"`
	HasAcceptedTerms    bool    `json:"has_accepted_terms"`
	// HasSignature, HasInitial, and IsSignatureReusable are returned only by
	// GET /signers/self.
	HasSignature        bool `json:"has_signature,omitempty"`
	HasInitial          bool `json:"has_initial,omitempty"`
	IsSignatureReusable bool `json:"is_signature_reusable,omitempty"`
	// The following fields are populated only when the signer is embedded in an
	// assignment (the Assignment Signer object).
	VerificationMethod  *string              `json:"verification_method,omitempty"`
	NotificationMethods []string             `json:"notification_methods,omitempty"`
	Completed           bool                 `json:"completed,omitempty"`
	Step                *int                 `json:"step,omitempty"`
	Notified            *bool                `json:"notified,omitempty"`
	NotificationHistory []SignerNotification `json:"notification_history,omitempty"`
}

// SignerNotification is one tracked delivery attempt inside an assignment
// signer's NotificationHistory.
type SignerNotification struct {
	Event        string     `json:"event"`
	Status       string     `json:"status"`
	ErrorCode    *string    `json:"error_code,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	SentAt       *Timestamp `json:"sent_at,omitempty"`
	FailedAt     *Timestamp `json:"failed_at,omitempty"`
}

// CreateSignerRequest is the body for POST /accounts/{id}/signers.
type CreateSignerRequest struct {
	FullName            string  `json:"full_name"`
	Email               *string `json:"email,omitempty"`
	WhatsAppPhoneNumber *string `json:"whatsapp_phone_number,omitempty"`
}

// UpdateSignerRequest is the body for PUT /accounts/{id}/signers/{sid}.
type UpdateSignerRequest struct {
	FullName            *string `json:"full_name,omitempty"`
	Email               *string `json:"email,omitempty"`
	WhatsAppPhoneNumber *string `json:"whatsapp_phone_number,omitempty"`
}

// ConfirmSignerDataRequest is the body for PUT /documents/{id}/signers/confirm-data.
// Every field is optional; a nil field is not sent and leaves the signer's
// current value unchanged. The OpenAPI reference documents full_name, email, and
// government_id, while the live API also accepts whatsapp_phone_number and
// has_accepted_terms, so all five are exposed here.
type ConfirmSignerDataRequest struct {
	FullName            *string `json:"full_name,omitempty"`
	Email               *string `json:"email,omitempty"`
	GovernmentID        *string `json:"government_id,omitempty"`
	WhatsAppPhoneNumber *string `json:"whatsapp_phone_number,omitempty"`
	HasAcceptedTerms    *bool   `json:"has_accepted_terms,omitempty"`
}

// SignerReference identifies a signer inside a CreateAssignmentRequest.
type SignerReference struct {
	ID                  string   `json:"id"`
	VerificationMethod  string   `json:"verification_method,omitempty"`
	NotificationMethods []string `json:"notification_methods,omitempty"`
	// Step controls sequential signing order. Signers sharing a step sign in
	// parallel; a step is activated only after every signer in the previous
	// step has signed. When supplied, every signer must supply it and the
	// values must form a contiguous sequence starting at 1.
	Step *int `json:"step,omitempty"`
}
