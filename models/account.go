package models

// NotificationSenderType controls whose name signers see as the sender of the
// notifications for documents in an account.
type NotificationSenderType string

// Documented values for NotificationSenderType.
const (
	// NotificationSenderUser shows the document owner's name (the default).
	NotificationSenderUser NotificationSenderType = "User"
	// NotificationSenderAccount shows the account's own name.
	NotificationSenderAccount NotificationSenderType = "Account"
)

// Account is a workspace account (organization) returned by the Account
// endpoints. The API returns different field subsets per endpoint: the list
// endpoint (GET /accounts) includes Roles and IsDeleteAllowed; the single-account
// endpoint (GET /accounts/{id}) includes PrimaryColor and SecondaryColor. Fields
// absent from a given response stay at their zero value.
type Account struct {
	// Resource is the API resource discriminator when that endpoint includes it.
	Resource string `json:"resource,omitempty"`
	// ID is the account identifier used in account-scoped endpoint paths.
	ID string `json:"id"`
	// Name is the account's display name.
	Name string `json:"name"`
	// PrimaryColor is the optional 6-character hexadecimal brand color without
	// a leading "#"; nil means the endpoint omitted it or no color is set.
	PrimaryColor *string `json:"primary_color,omitempty"`
	// SecondaryColor is the optional 6-character hexadecimal accent color
	// without a leading "#"; nil means the endpoint omitted it or none is set.
	SecondaryColor *string `json:"secondary_color,omitempty"`
	// NotificationSenderType is "User" or "Account" when returned.
	NotificationSenderType NotificationSenderType `json:"notification_sender_type,omitempty"`
	// Roles lists the authenticated user's roles in this account.
	Roles []string `json:"roles,omitempty"`
	// IsDeleteAllowed reports whether the authenticated user may delete the account.
	IsDeleteAllowed bool `json:"is_delete_allowed"`
	// Limits contains plan limits on the single-account response; lists omit it.
	Limits *AccountLimits `json:"limits,omitempty"`
	// SigningFeatures is included only with expand=signing_features.
	SigningFeatures *SigningFeatures `json:"signing_features,omitempty"`
	// CreatedAt is the account creation date-time.
	CreatedAt Timestamp `json:"created_at"`
}

// AccountLimits describes the workspace's current plan limits.
type AccountLimits struct {
	// WebhookEndpoints is the maximum number of webhook endpoints (1 or 3).
	WebhookEndpoints int `json:"webhook_endpoints"`
}

// AccountTheme is the branding theme returned by GET /accounts/{id}/theme.
type AccountTheme struct {
	// AccountName is the display name shown in signer-facing branding.
	AccountName string `json:"account_name"`
	// PrimaryColor is a 6-character hex color without a leading "#".
	PrimaryColor string `json:"primary_color"`
	// SecondaryColor is a 6-character hex color without a leading "#", or nil.
	SecondaryColor *string `json:"secondary_color,omitempty"`
	// Logo is the URL to the account logo image, or nil when none is set.
	Logo *string `json:"logo,omitempty"`
}

// CreateAccountRequest is the body for POST /accounts.
type CreateAccountRequest struct {
	// Name is the required new workspace display name.
	Name string `json:"name"`
	// NotificationSenderType controls whose name signers see as the sender.
	// Nil uses the API default, NotificationSenderUser.
	NotificationSenderType *NotificationSenderType `json:"notification_sender_type,omitempty"`
}

// UpdateAccountRequest is the body for PUT /accounts/{account_id}. A nil field
// leaves the corresponding attribute unchanged.
type UpdateAccountRequest struct {
	// Name renames the account. Nil leaves the name unchanged.
	Name *string `json:"name,omitempty"`
	// NotificationSenderType switches the notification sender shown to signers.
	// Nil leaves the current value unchanged.
	NotificationSenderType *NotificationSenderType `json:"notification_sender_type,omitempty"`
}

// AccountGetParams selects additional fields on a single-account read.
type AccountGetParams struct {
	// Expand accepts signing_features to include current signing entitlements.
	Expand string
}

// SigningFeatures lists the workspace's current signing entitlements. Credits
// and assignment rules still apply, and a subscription change can revoke access.
type SigningFeatures struct {
	// DigitalCertificate permits the DigitalCertificate verification method.
	DigitalCertificate bool `json:"DigitalCertificate"`
	// VerificationIdDocumentFaceMatchBasic permits ID document and face matching.
	VerificationIdDocumentFaceMatchBasic bool `json:"VerificationIdDocumentFaceMatchBasic"`
	// WhatsappNotification permits WhatsApp notifications and OTP verification.
	WhatsappNotification bool `json:"WhatsappNotification"`
	// VerificationBypass permits signing without an OTP.
	VerificationBypass bool `json:"VerificationBypass"`
	// Embed permits embedded signers, allowed origins and embed sessions.
	Embed bool `json:"Embed"`
}

// Balance describes a document or credit allowance for the current billing cycle.
type Balance struct {
	// Balance is the remaining quantity; nil means unlimited.
	Balance *float64 `json:"balance"`
	// Limit is the cycle allowance; nil means unlimited or not cycle-limited.
	Limit *float64 `json:"limit"`
	// Spent is the quantity consumed in the current cycle.
	Spent float64 `json:"spent"`
}

// AccountBalances contains the workspace's document and credit allowances.
type AccountBalances struct {
	// Documents is the document allowance.
	Documents Balance `json:"documents"`
	// Credits is the credit allowance.
	Credits Balance `json:"credits"`
}

// AccountUsage is the plan and remaining allowances, without payment information.
type AccountUsage struct {
	// Plan is the plan name; nil means no plan or a retired plan.
	Plan *string `json:"plan"`
	// DocumentsRenewalDate is the next allowance date, or nil before the first
	// payment or on a contract with no billing day.
	DocumentsRenewalDate *string `json:"documents_renewal_date"`
	// Balances contains remaining documents and credits, preserving unlimited nulls.
	Balances AccountBalances `json:"balances"`
}

// NotificationPolicyEvents maps notification codes to workspace delivery rules.
// False suppresses that alert on every channel except the identity-review email.
type NotificationPolicyEvents map[NotificationPreference]bool

// NotificationPolicy is the workspace's owner-controlled notification policy.
type NotificationPolicy struct {
	// Editable reports whether the requesting user may change the policy.
	Editable bool `json:"editable"`
	// OwnersReceiveAll gives owners push alerts for all members' documents,
	// subject to each owner's preferences.
	OwnersReceiveAll bool `json:"owners_receive_all"`
	// Events contains every workspace notification rule; new codes may be added.
	Events NotificationPolicyEvents `json:"events"`
}

// UpdateNotificationPolicyRequest merges the supplied rules into the policy.
// Omitted fields stay unchanged; an empty body or unknown code is rejected.
type UpdateNotificationPolicyRequest struct {
	// Events contains only notification codes to change.
	Events NotificationPolicyEvents `json:"events,omitempty"`
	// OwnersReceiveAll changes owner push delivery for all members' documents.
	OwnersReceiveAll *bool `json:"owners_receive_all,omitempty"`
}
