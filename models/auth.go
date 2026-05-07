package models

type LoginResponse struct {
	AccessToken string      `json:"access_token"`
	User        User        `json:"user"`
	Accounts    []Workspace `json:"accounts"`
}

type User struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Email            string  `json:"email"`
	Telephone        string  `json:"telephone,omitempty"`
	GovernmentID     string  `json:"government_id,omitempty"`
	IsEmailVerified  bool    `json:"is_email_verified"`
	HasAcceptedTerms bool    `json:"has_accepted_terms"`
	CreatedAt        string  `json:"created_at"`
	ToBeDeletedAt    *string `json:"to_be_deleted_at,omitempty"`
}

type ChangePasswordRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	NewPassword string `json:"new_password"`
}

type RequestPasswordResetRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email"`
	Token       string `json:"token,omitempty"`
	NewPassword string `json:"new_password"`
}

type SocialLoginRequest struct {
	Provider         string `json:"provider"`
	Token            string `json:"token"`
	HasAcceptedTerms bool   `json:"has_accepted_terms"`
}

type CreateAPIKeyRequest struct {
	Password string `json:"password"`
}

type APIKeyResponse struct {
	APIKey string `json:"api_key"`
}
