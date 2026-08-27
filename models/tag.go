package models

import "encoding/json"

// Tag is a workspace-scoped label that can be attached to documents and
// templates. The full shape is returned by the Tag endpoints; when a tag
// appears inline on a document or template only ID and Name are declared.
type Tag struct {
	// Resource is the API resource discriminator when present.
	Resource string `json:"resource,omitempty"`
	// ID is the account tag identifier.
	ID string `json:"id"`
	// Name is the case-insensitively unique workspace label.
	Name string `json:"name"`
	// Color is a nullable 6-character hexadecimal color.
	Color *string `json:"color,omitempty"`
	// CreatedAt is the creation date-time when included.
	CreatedAt Timestamp `json:"created_at,omitempty"`
	// UpdatedAt is the last-update date-time when included.
	UpdatedAt Timestamp `json:"updated_at,omitempty"`
}

// CreateTagRequest is the body for POST /accounts/{account_id}/tags.
type CreateTagRequest struct {
	// Name is the required workspace-unique label. The API trims it, collapses
	// internal whitespace, and limits it to 64 characters.
	Name string `json:"name"`
	// Color is a 6-character hex color (with or without leading "#"). Leave nil
	// to omit color. To send the API's explicit null form, set ClearColor.
	Color *string `json:"color,omitempty"`
	// ClearColor, when true, sends color:null and takes precedence over Color.
	ClearColor bool `json:"-"`
}

// MarshalJSON preserves the API's omitted, null, and string color states.
func (r CreateTagRequest) MarshalJSON() ([]byte, error) {
	m := map[string]any{"name": r.Name}
	switch {
	case r.ClearColor:
		m["color"] = nil
	case r.Color != nil:
		m["color"] = *r.Color
	}
	return json.Marshal(m)
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
// document-tag endpoints. The current contract specifies IDs; the live API also
// accepts names and creates unknown names case-insensitively.
type SetDocumentTagsRequest struct {
	// Tags is the optional API list of tag IDs. The SDK's document helpers also
	// accept names as a live compatibility extension; an empty list clears on PUT.
	Tags []string `json:"tags"`
}

// TagDeleteResult is returned after deleting a workspace tag.
type TagDeleteResult struct {
	// Deleted reports whether the tag was deleted.
	Deleted bool `json:"deleted"`
}

// DocumentTagDetachResult is returned after detaching a tag from a document.
type DocumentTagDetachResult struct {
	// Detached reports whether the tag was attached and removed.
	Detached bool `json:"detached"`
}
