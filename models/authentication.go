package models

// LoginRequest is the body for POST /login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SocialLoginRequest is the body for POST /authentication/social-login.
type SocialLoginRequest struct {
	Provider         string `json:"provider"`
	Token            string `json:"token"`
	HasAcceptedTerms bool   `json:"has_accepted_terms"`
}

// AuthenticationResult is returned by login endpoints.
type AuthenticationResult struct {
	AccessToken string              `json:"access_token"`
	User        User                `json:"user"`
	Accounts    []WorkspaceListItem `json:"accounts"`
}

// CreateAPIKeyRequest is the body for POST /users/api-keys.
type CreateAPIKeyRequest struct {
	Password string `json:"password"`
}

// APIKeyResult is the response from the /users/api-keys endpoints.
type APIKeyResult struct {
	APIKey *string `json:"api_key"`
}

// ChangePasswordRequest is the body for PUT /authentication/change-password.
type ChangePasswordRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	NewPassword string `json:"new_password"`
}

// RequestPasswordResetRequest is the body for PUT /authentication/request-password-reset.
type RequestPasswordResetRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest is the body for PUT /authentication/reset-password.
type ResetPasswordRequest struct {
	Email       string  `json:"email"`
	Token       *string `json:"token,omitempty"`
	NewPassword string  `json:"new_password"`
}

// EmailResult is returned by password-reset workflow endpoints.
type EmailResult struct {
	Email string `json:"email"`
}
