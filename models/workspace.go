package models

type Workspace struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Roles           []string `json:"roles,omitempty"`
	IsDeleteAllowed bool     `json:"is_delete_allowed,omitempty"`
	CreatedAt       string   `json:"created_at"`
}

type CreateWorkspaceRequest struct {
	Name         string `json:"name"`
	PrimaryColor string `json:"primary_color,omitempty"`
}

type UpdateWorkspaceRequest struct {
	Name         string `json:"name,omitempty"`
	PrimaryColor string `json:"primary_color,omitempty"`
}

type WorkspaceListItem struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles,omitempty"`
	CreatedAt string   `json:"created_at"`
}
