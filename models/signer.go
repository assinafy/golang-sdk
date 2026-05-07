package models

type Signer struct {
	Resource            string  `json:"resource,omitempty"`
	ID                  string  `json:"id"`
	FullName            string  `json:"full_name"`
	Email               *string `json:"email,omitempty"`
	WhatsAppPhoneNumber *string `json:"whatsapp_phone_number,omitempty"`
	HasAcceptedTerms    bool    `json:"has_accepted_terms"`
	HasSignature        bool    `json:"has_signature,omitempty"`
	HasInitial          bool    `json:"has_initial,omitempty"`
}

type CreateSignerRequest struct {
	FullName            string  `json:"full_name"`
	Email               *string `json:"email,omitempty"`
	WhatsAppPhoneNumber *string `json:"whatsapp_phone_number,omitempty"`
}

type UpdateSignerRequest struct {
	FullName            *string `json:"full_name,omitempty"`
	Email               *string `json:"email,omitempty"`
	WhatsAppPhoneNumber *string `json:"whatsapp_phone_number,omitempty"`
}

type SignerReference struct {
	ID                  string   `json:"id"`
	VerificationMethod  string   `json:"verification_method,omitempty"`
	NotificationMethods []string `json:"notification_methods,omitempty"`
}
