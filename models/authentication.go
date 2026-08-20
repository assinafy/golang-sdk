package models

// LoginRequest is the body for POST /login.
type LoginRequest struct {
	// Email is the required email-formatted account address.
	Email string `json:"email"`
	// Password is the required plaintext password sent over HTTPS.
	Password string `json:"password"`
}

// SocialLoginRequest is the body for POST /authentication/social-login.
type SocialLoginRequest struct {
	// Provider is the required provider key; the current documented value is "google".
	Provider string `json:"provider"`
	// Token is the required identity token issued by Provider.
	Token string `json:"token"`
	// HasAcceptedTerms records whether the user accepted Assinafy's terms.
	HasAcceptedTerms bool `json:"has_accepted_terms"`
}

// LinkSocialLoginRequest is the body for POST /auth/link-social-login.
type LinkSocialLoginRequest struct {
	// Provider is the social provider key (currently only "google").
	Provider string `json:"provider"`
	// Token is the token issued by the provider.
	Token string `json:"token"`
}

// AuthenticationResult is returned by login endpoints.
type AuthenticationResult struct {
	// AccessToken is the JWT bearer token for authenticated user endpoints.
	AccessToken string `json:"access_token"`
	// User is the authenticated account holder.
	User User `json:"user"`
	// Accounts lists the workspaces available to the authenticated user.
	Accounts []WorkspaceListItem `json:"accounts"`
}

// CreateAPIKeyRequest is the body for POST /users/api-keys.
type CreateAPIKeyRequest struct {
	// Password is the authenticated user's required current password.
	Password string `json:"password"`
}

// APIKeyResult is the response from the /users/api-keys endpoints.
type APIKeyResult struct {
	// APIKey is the newly issued key on creation, the masked key on retrieval,
	// or nil when no API key exists.
	APIKey *string `json:"api_key"`
}

// ChangePasswordRequest is the body for PUT /authentication/change-password.
type ChangePasswordRequest struct {
	// Email is the required email-formatted account address.
	Email string `json:"email"`
	// Password is the required current password.
	Password string `json:"password"`
	// NewPassword is the required replacement password.
	NewPassword string `json:"new_password"`
}

// RequestPasswordResetRequest is the body for PUT /authentication/request-password-reset.
type RequestPasswordResetRequest struct {
	// Email is the required email-formatted destination account address.
	Email string `json:"email"`
}

// ResetPasswordRequest is the body for PUT /authentication/reset-password.
type ResetPasswordRequest struct {
	// Email is the required email-formatted account address.
	Email string `json:"email"`
	// Token is the optional reset token supplied by the workflow; nil omits it.
	Token *string `json:"token,omitempty"`
	// NewPassword is the required replacement password.
	NewPassword string `json:"new_password"`
}

// EmailResult is returned by password-reset workflow endpoints.
type EmailResult struct {
	// Email is the email-formatted account address acted on by the password workflow.
	Email string `json:"email"`
}
