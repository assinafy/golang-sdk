package models

// User describes an account holder. It is returned by the login and social-login
// responses and also appears as the subject of some webhook events. Not every
// field is populated in every context — for example is_password_set is emitted
// for webhook subjects but not by the login endpoints.
type User struct {
	// ID is the user identifier.
	ID string `json:"id"`
	// Name is the user's display name.
	Name string `json:"name"`
	// Email is the user's email-formatted account address.
	Email string `json:"email"`
	// Telephone is nullable in the API; JSON null decodes to the empty string. It
	// remains a string rather than *string for v1.0.0 source compatibility.
	Telephone string `json:"telephone,omitempty"`
	// GovernmentID is the nullable government identifier represented as an
	// empty string when the API returns null; see Telephone above.
	GovernmentID string `json:"government_id,omitempty"`
	// IsEmailVerified reports whether the account email has been verified.
	IsEmailVerified bool `json:"is_email_verified"`
	// HasAcceptedTerms reports whether the user accepted Assinafy's terms.
	HasAcceptedTerms bool `json:"has_accepted_terms"`
	// IsPasswordSet reports whether the webhook subject has a password configured.
	IsPasswordSet bool `json:"is_password_set,omitempty"`
	// CreatedAt is the account creation date-time.
	CreatedAt Timestamp `json:"created_at"`
	// ToBeDeletedAt is the nullable scheduled account-deletion date-time.
	ToBeDeletedAt *Timestamp `json:"to_be_deleted_at,omitempty"`
}
