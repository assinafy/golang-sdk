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
	Resource               string                 `json:"resource,omitempty"`
	ID                     string                 `json:"id"`
	Name                   string                 `json:"name"`
	PrimaryColor           *string                `json:"primary_color,omitempty"`
	SecondaryColor         *string                `json:"secondary_color,omitempty"`
	NotificationSenderType NotificationSenderType `json:"notification_sender_type,omitempty"`
	Roles                  []string               `json:"roles,omitempty"`
	IsDeleteAllowed        bool                   `json:"is_delete_allowed"`
	CreatedAt              Timestamp              `json:"created_at"`
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

// UpdateAccountRequest is the body for PUT /accounts/{account_id}. A nil field
// leaves the corresponding attribute unchanged.
type UpdateAccountRequest struct {
	// Name renames the account. Nil leaves the name unchanged.
	Name *string `json:"name,omitempty"`
	// NotificationSenderType switches the notification sender shown to signers.
	// Nil leaves the current value unchanged.
	NotificationSenderType *NotificationSenderType `json:"notification_sender_type,omitempty"`
}
