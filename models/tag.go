package models

import "encoding/json"

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
// An omitted field leaves the corresponding attribute unchanged.
type UpdateTagRequest struct {
	// Name renames the tag. Nil leaves the name unchanged.
	Name *string
	// Color sets the tag's hex color (6 characters, with or without a leading
	// "#"). Nil leaves the color unchanged. To remove an existing color, set
	// ClearColor instead of Color.
	Color *string
	// ClearColor, when true, sends color:null to clear the tag's color. It takes
	// precedence over Color.
	ClearColor bool
}

// MarshalJSON encodes only the attributes that should change, honouring the
// three states the API distinguishes for color: omitted (leave unchanged), null
// (clear), and a hex string (set).
func (r UpdateTagRequest) MarshalJSON() ([]byte, error) {
	m := make(map[string]any, 2)
	if r.Name != nil {
		m["name"] = *r.Name
	}
	switch {
	case r.ClearColor:
		m["color"] = nil
	case r.Color != nil:
		m["color"] = *r.Color
	}
	return json.Marshal(m)
}

// SetDocumentTagsRequest is the body for the replace (PUT) and append (POST)
// document-tag endpoints. Names that do not yet exist in the workspace are
// created automatically (case-insensitive lookup).
type SetDocumentTagsRequest struct {
	Tags []string `json:"tags"`
}
