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
	// MFAToken is the single-use two-factor challenge that login returns for a
	// user with an enrolled two-factor method. Pass it to
	// AuthenticationResource.VerifyMFA within five minutes to obtain the access
	// token.
	MFAToken string `json:"mfa_token,omitempty"`
	// User is the authenticated account holder.
	User User `json:"user"`
	// Accounts lists the workspaces available to the authenticated user.
	Accounts []WorkspaceListItem `json:"accounts"`
}

// VerifyMFARequest is the body for POST /authentication/mfa/verify.
type VerifyMFARequest struct {
	// MFAToken is the required challenge from AuthenticationResult.MFAToken.
	MFAToken string `json:"mfa_token"`
	// Code is a required 6-digit authenticator code or a recovery code such as
	// ABCD-EFGH-JKMN.
	Code string `json:"code"`
}

// MFAMethod is one enrolled two-factor method.
type MFAMethod struct {
	// ID identifies the method for UserResource.RemoveMFAMethod.
	ID string `json:"id"`
	// Type is the method kind, such as "Totp".
	Type string `json:"type"`
	// Label is the name given at enrollment.
	Label string `json:"label"`
	// ConfirmedAt is the confirmation date-time; nil while unconfirmed.
	ConfirmedAt *Timestamp `json:"confirmed_at,omitempty"`
	// LastUsedAt is the date-time of the last successful use, if any.
	LastUsedAt *Timestamp `json:"last_used_at,omitempty"`
}

// MFAStatus is returned by GET /users/self/mfa.
type MFAStatus struct {
	// Methods lists the enrolled two-factor methods.
	Methods []MFAMethod `json:"methods"`
	// RecoveryCodesRemaining counts the unused recovery codes.
	RecoveryCodesRemaining int `json:"recovery_codes_remaining"`
}

// TOTPEnrollment is returned by POST /users/self/mfa/totp. Secret and
// ProvisioningURI are returned only once.
type TOTPEnrollment struct {
	// ID identifies the unconfirmed method for ConfirmTOTPRequest.
	ID string `json:"id"`
	// Secret is the base32 shared secret for manual entry in an authenticator.
	Secret string `json:"secret"`
	// ProvisioningURI is the otpauth:// URI, usually rendered as a QR code.
	ProvisioningURI string `json:"provisioning_uri"`
}

// ConfirmTOTPRequest is the body for PUT /users/self/mfa/totp/confirm.
type ConfirmTOTPRequest struct {
	// ID is the required TOTPEnrollment.ID.
	ID string `json:"id"`
	// Code is a required live code from the device being enrolled.
	Code string `json:"code"`
	// Password re-authenticates the user; required only when the confirmation
	// replaces an existing confirmed method. ReauthCode is the alternative.
	Password string `json:"password,omitempty"`
	// ReauthCode is a live code from the current device or a recovery code;
	// required only when replacing an existing confirmed method.
	ReauthCode string `json:"reauth_code,omitempty"`
}

// MFAReauthRequest proves the user's identity before a two-factor change, for
// POST /users/self/mfa/recovery-codes and DELETE /users/self/mfa/{id}. Set
// Password or Code; a recovery code used as Code is consumed.
type MFAReauthRequest struct {
	// Password is the user's current password.
	Password string `json:"password,omitempty"`
	// Code is a live 6-digit authenticator code or an unused recovery code.
	Code string `json:"code,omitempty"`
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
