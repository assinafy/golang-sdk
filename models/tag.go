package models

// Tag is a workspace-scoped label that can be attached to documents and
// templates. The full shape is returned by the Tag endpoints; when a tag
// appears inline on a document or template only ID, Name, and Color are set.
type Tag struct {
	Resource  string    `json:"resource,omitempty"`
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     *string   `json:"color,omitempty"`
	CreatedAt Timestamp `json:"created_at,omitempty"`
	UpdatedAt Timestamp `json:"updated_at,omitempty"`
}

// CreateTagRequest is the body for POST /accounts/{account_id}/tags.
type CreateTagRequest struct {
	Name string `json:"name"`
	// Color is a 6-character hex color (with or without leading "#"). Leave nil
	// for no color.
	Color *string `json:"color,omitempty"`
}

// UpdateTagRequest is the body for PUT /accounts/{account_id}/tags/{tag_id}.
// Nil fields are left unchanged.
type UpdateTagRequest struct {
	Name  *string `json:"name,omitempty"`
	Color *string `json:"color,omitempty"`
}

// SetDocumentTagsRequest is the body for the replace (PUT) and append (POST)
// document-tag endpoints. Names that do not yet exist in the workspace are
// created automatically (case-insensitive lookup).
type SetDocumentTagsRequest struct {
	Tags []string `json:"tags"`
}
