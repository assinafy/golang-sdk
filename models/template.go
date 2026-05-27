package models

// TemplateStatus enumerates the documented template processing states.
type TemplateStatus string

// Documented values for TemplateStatus.
const (
	TemplateStatusUploading  TemplateStatus = "uploading"
	TemplateStatusUploaded   TemplateStatus = "uploaded"
	TemplateStatusProcessing TemplateStatus = "processing"
	TemplateStatusReady      TemplateStatus = "ready"
	TemplateStatusFailed     TemplateStatus = "failed"
)

// Template models the response of GET /accounts/{id}/templates and
// GET /accounts/{id}/templates/{tid}.
type Template struct {
	Resource     string         `json:"resource,omitempty"`
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	DocumentName *string        `json:"document_name,omitempty"`
	Message      *string        `json:"message,omitempty"`
	Status       TemplateStatus `json:"status"`
	Pages        []TemplatePage `json:"pages,omitempty"`
	Roles        []TemplateRole `json:"roles"`
	// Tags are the template's own tags. Each inline entry carries only ID and Name.
	Tags []Tag `json:"tags,omitempty"`
	// DefaultDocumentTags are applied to every document created from this
	// template. Only the single-template endpoint populates this field.
	DefaultDocumentTags []Tag     `json:"default_document_tags,omitempty"`
	CreatedAt           Timestamp `json:"created_at"`
	UpdatedAt           Timestamp `json:"updated_at"`
}

// TemplateRole describes a signing role declared on a template.
type TemplateRole struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	AssignmentType string    `json:"assignment_type,omitempty"`
	CreatedAt      Timestamp `json:"created_at,omitempty"`
	UpdatedAt      Timestamp `json:"updated_at,omitempty"`
}

// TemplatePage is a single page inside a Template.
type TemplatePage struct {
	ID          string                   `json:"id"`
	Number      int                      `json:"number"`
	Height      int                      `json:"height"`
	Width       int                      `json:"width"`
	DownloadURL string                   `json:"download_url"`
	Fields      []TemplateFieldPlacement `json:"fields,omitempty"`
}

// TemplateFieldPlacement is a single field placement on a TemplatePage.
type TemplateFieldPlacement struct {
	ID              string    `json:"id"`
	FieldID         string    `json:"field_id"`
	RoleID          string    `json:"role_id"`
	Label           string    `json:"label"`
	DisplaySettings any       `json:"display_settings,omitempty"`
	CreatedAt       Timestamp `json:"created_at"`
	UpdatedAt       Timestamp `json:"updated_at"`
}
