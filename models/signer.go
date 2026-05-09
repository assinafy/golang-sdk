package models

// Signer is the canonical signer object.
type Signer struct {
	Resource            string   `json:"resource,omitempty"`
	ID                  string   `json:"id"`
	FullName            string   `json:"full_name"`
	Email               *string  `json:"email,omitempty"`
	WhatsAppPhoneNumber *string  `json:"whatsapp_phone_number,omitempty"`
	HasAcceptedTerms    bool     `json:"has_accepted_terms"`
	HasSignature        bool     `json:"has_signature,omitempty"`
	HasInitial          bool     `json:"has_initial,omitempty"`
	VerificationMethod  *string  `json:"verification_method,omitempty"`
	NotificationMethods []string `json:"notification_methods,omitempty"`
	Completed           bool     `json:"completed,omitempty"`
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
type ConfirmSignerDataRequest struct {
	Email               *string `json:"email,omitempty"`
	WhatsAppPhoneNumber *string `json:"whatsapp_phone_number,omitempty"`
	HasAcceptedTerms    *bool   `json:"has_accepted_terms,omitempty"`
}

// SignerReference identifies a signer inside a CreateAssignmentRequest.
type SignerReference struct {
	ID                  string   `json:"id"`
	VerificationMethod  string   `json:"verification_method,omitempty"`
	NotificationMethods []string `json:"notification_methods,omitempty"`
}
