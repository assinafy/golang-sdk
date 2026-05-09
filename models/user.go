package models

// User describes an account holder returned by login responses.
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
