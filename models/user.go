package models

// User describes an account holder. It is returned by the login and social-login
// responses and also appears as the subject of some webhook events. Not every
// field is populated in every context — for example is_password_set is emitted
// for webhook subjects but not by the login endpoints.
type User struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Email            string     `json:"email"`
	Telephone        string     `json:"telephone,omitempty"`
	GovernmentID     string     `json:"government_id,omitempty"`
	IsEmailVerified  bool       `json:"is_email_verified"`
	HasAcceptedTerms bool       `json:"has_accepted_terms"`
	IsPasswordSet    bool       `json:"is_password_set,omitempty"`
	CreatedAt        Timestamp  `json:"created_at"`
	ToBeDeletedAt    *Timestamp `json:"to_be_deleted_at,omitempty"`
}
