package models

// Signer is the canonical signer object.
type Signer struct {
	// Resource is the API resource discriminator when present.
	Resource string `json:"resource,omitempty"`
	// ID is the signer identifier.
	ID string `json:"id"`
	// FullName is the signer's display name.
	FullName string `json:"full_name"`
	// Email is the nullable signer email address.
	Email *string `json:"email,omitempty"`
	// WhatsAppPhoneNumber is the nullable signer WhatsApp telephone number.
	WhatsAppPhoneNumber *string `json:"whatsapp_phone_number,omitempty"`
	// HasAcceptedTerms reports whether the signer accepted Assinafy's terms.
	HasAcceptedTerms bool `json:"has_accepted_terms"`
	// HasSignature reports whether GET /signers/self found a saved signature.
	HasSignature bool `json:"has_signature,omitempty"`
	// HasInitial reports whether GET /signers/self found saved initials.
	HasInitial bool `json:"has_initial,omitempty"`
	// IsSignatureReusable reports whether the saved signature may be reused.
	IsSignatureReusable bool `json:"is_signature_reusable,omitempty"`
	// VerificationMethod is the nullable API verification-method code on an assignment.
	VerificationMethod *string `json:"verification_method,omitempty"`
	// NotificationMethods contains API delivery-channel codes on an assignment.
	NotificationMethods []string `json:"notification_methods,omitempty"`
	// Completed reports whether this signer completed the assignment. The API
	// permits null; null and absence both decode to false for source compatibility.
	Completed bool `json:"completed,omitempty"`
	// Step is the nullable one-based sequential-signing step.
	Step *int `json:"step,omitempty"`
	// Notified is nullable and reports whether the invitation was dispatched.
	Notified *bool `json:"notified,omitempty"`
	// NotificationHistory contains tracked invitation delivery attempts.
	NotificationHistory []SignerNotification `json:"notification_history,omitempty"`
}

// SignerNotification is one tracked delivery attempt inside an assignment
// signer's NotificationHistory.
type SignerNotification struct {
	// Event is the machine-readable notification event code.
	Event string `json:"event"`
	// Status is the delivery status, "sent" or "failed".
	Status string `json:"status"`
	// ErrorCode is the nullable provider error code for a failed attempt.
	ErrorCode *string `json:"error_code,omitempty"`
	// ErrorMessage is the nullable provider explanation for a failed attempt.
	ErrorMessage *string `json:"error_message,omitempty"`
	// SentAt is the nullable successful-send date-time.
	SentAt *Timestamp `json:"sent_at,omitempty"`
	// FailedAt is the nullable failure date-time.
	FailedAt *Timestamp `json:"failed_at,omitempty"`
}

// CreateSignerRequest is the body for POST /accounts/{id}/signers.
type CreateSignerRequest struct {
	// FullName is the required signer display name.
	FullName string `json:"full_name"`
	// Email is an optional email-formatted address; nil omits it.
	Email *string `json:"email,omitempty"`
	// WhatsAppPhoneNumber is an optional E.164 telephone number; nil omits it.
	WhatsAppPhoneNumber *string `json:"whatsapp_phone_number,omitempty"`
}

// UpdateSignerRequest is the body for PUT /accounts/{id}/signers/{sid}.
type UpdateSignerRequest struct {
	// FullName optionally replaces the display name; nil leaves it unchanged.
	FullName *string `json:"full_name,omitempty"`
	// Email optionally replaces the email-formatted address; nil leaves it unchanged.
	Email *string `json:"email,omitempty"`
	// WhatsAppPhoneNumber optionally replaces the E.164 number; nil leaves it unchanged.
	WhatsAppPhoneNumber *string `json:"whatsapp_phone_number,omitempty"`
	// GovernmentID optionally replaces the government identifier; nil leaves it unchanged.
	GovernmentID *string `json:"government_id,omitempty"`
}

// ConfirmSignerDataRequest is the body for PUT /documents/{id}/signers/confirm-data.
// Every field is optional; a nil field is not sent and leaves the signer's
// current value unchanged. The OpenAPI reference documents full_name, email, and
// government_id, while the live API also accepts whatsapp_phone_number and
// has_accepted_terms, so all five are exposed here.
type ConfirmSignerDataRequest struct {
	// FullName optionally confirms or replaces the display name.
	FullName *string `json:"full_name,omitempty"`
	// Email optionally confirms or replaces the email-formatted address.
	Email *string `json:"email,omitempty"`
	// GovernmentID optionally confirms or replaces the government identifier.
	GovernmentID *string `json:"government_id,omitempty"`
	// WhatsAppPhoneNumber optionally confirms or replaces the telephone number.
	WhatsAppPhoneNumber *string `json:"whatsapp_phone_number,omitempty"`
	// HasAcceptedTerms optionally records terms acceptance for this signing session.
	HasAcceptedTerms *bool `json:"has_accepted_terms,omitempty"`
}

// SignerReference identifies a signer inside a CreateAssignmentRequest.
type SignerReference struct {
	// ID is the required existing signer identifier.
	ID string `json:"id"`
	// VerificationMethod is an optional API verification-method code.
	VerificationMethod string `json:"verification_method,omitempty"`
	// NotificationMethods contains optional API delivery-channel codes.
	NotificationMethods []string `json:"notification_methods,omitempty"`
	// Step controls sequential signing order. Signers sharing a step sign in
	// parallel; a step is activated only after every signer in the previous
	// step has signed. When supplied, every signer must supply it and the
	// values must form a contiguous sequence starting at 1.
	Step *int `json:"step,omitempty"`
}
