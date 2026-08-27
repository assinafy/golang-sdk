package models

// WorkspaceListItem is the slim account record returned in the login response.
type WorkspaceListItem struct {
	// ID is the account identifier.
	ID string `json:"id"`
	// Name is the account display name.
	Name string `json:"name"`
	// Roles lists the authenticated user's roles in the account.
	Roles []string `json:"roles,omitempty"`
	// IsDeleteAllowed reports whether the authenticated user may delete it.
	IsDeleteAllowed bool `json:"is_delete_allowed"`
	// CreatedAt is the account creation date-time.
	CreatedAt Timestamp `json:"created_at"`
}
