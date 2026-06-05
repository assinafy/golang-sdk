package models

// WorkspaceListItem is the slim account record returned in the login response.
type WorkspaceListItem struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Roles           []string  `json:"roles,omitempty"`
	IsDeleteAllowed bool      `json:"is_delete_allowed"`
	CreatedAt       Timestamp `json:"created_at"`
}
