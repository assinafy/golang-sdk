package models

// TemplateStatus enumerates the documented template processing states.
type TemplateStatus string

// Documented values for TemplateStatus.
const (
	// TemplateStatusUploading means the source upload is in progress.
	TemplateStatusUploading TemplateStatus = "uploading"
	// TemplateStatusUploaded means the source was accepted for processing.
	TemplateStatusUploaded TemplateStatus = "uploaded"
	// TemplateStatusProcessing means page and field metadata is being generated.
	TemplateStatusProcessing TemplateStatus = "processing"
	// TemplateStatusReady means documents may be created from the template.
	TemplateStatusReady TemplateStatus = "ready"
	// TemplateStatusFailed means template processing failed.
	TemplateStatusFailed TemplateStatus = "failed"
)

// Template models the response of GET /accounts/{id}/templates and
// GET /accounts/{id}/templates/{tid}.
type Template struct {
	// Resource is the API resource discriminator when present.
	Resource string `json:"resource,omitempty"`
	// ID is the template identifier.
	ID string `json:"id"`
	// Name is the template display name.
	Name string `json:"name"`
	// DocumentName is the nullable default name for generated documents.
	DocumentName *string `json:"document_name,omitempty"`
	// Message is the nullable default signing-invitation message.
	Message *string `json:"message,omitempty"`
	// Status is the template processing lifecycle code.
	Status TemplateStatus `json:"status"`
	// Pages contains rendered pages and field placements when included.
	Pages []TemplatePage `json:"pages,omitempty"`
	// Roles contains the template's required signer roles.
	Roles []TemplateRole `json:"roles"`
	// Tags are the template's own tags (always present, possibly empty). Each
	// inline entry carries only ID and Name.
	Tags []Tag `json:"tags"`
	// DefaultDocumentTags are applied to every document created from this
	// template. Only the single-template endpoint populates this field.
	DefaultDocumentTags []Tag `json:"default_document_tags,omitempty"`
	// CreatedAt is the template creation date-time.
	CreatedAt Timestamp `json:"created_at"`
	// UpdatedAt is the template last-update date-time.
	UpdatedAt Timestamp `json:"updated_at"`
}

// TemplateRole describes a signing role declared on a template.
type TemplateRole struct {
	// ID is the role identifier referenced by TemplateSigner.RoleID.
	ID string `json:"id"`
	// Name is the role's display name.
	Name string `json:"name"`
	// Description is optional explanatory text for the role.
	Description string `json:"description,omitempty"`
	// AssignmentType is the endpoint-provided assignment-flow code when present.
	AssignmentType string `json:"assignment_type,omitempty"`
	// CreatedAt is the creation date-time when included.
	CreatedAt Timestamp `json:"created_at,omitempty"`
	// UpdatedAt is the last-update date-time when included.
	UpdatedAt Timestamp `json:"updated_at,omitempty"`
}

// TemplatePage is a single page inside a Template.
type TemplatePage struct {
	// ID is the page identifier referenced by template field placements.
	ID string `json:"id"`
	// Number is the one-based page number.
	Number int `json:"number"`
	// Height is the rendered page height in the API's 150-DPI coordinate system.
	Height int `json:"height"`
	// Width is the rendered page width in the API's 150-DPI coordinate system.
	Width int `json:"width"`
	// DownloadURL is the absolute URL returned for the rendered page.
	DownloadURL string `json:"download_url"`
	// Fields contains the template placements on this page when included.
	Fields []TemplateFieldPlacement `json:"fields,omitempty"`
}

// TemplateFieldPlacement is a single field placement on a TemplatePage.
type TemplateFieldPlacement struct {
	// ID is the placement identifier.
	ID string `json:"id"`
	// FieldID is the field-definition identifier.
	FieldID string `json:"field_id"`
	// RoleID is the signer-role identifier responsible for this field.
	RoleID string `json:"role_id"`
	// Label is the field label shown in the template editor.
	Label string `json:"label"`
	// DisplaySettings is the placement object, commonly DisplaySettings.
	DisplaySettings any `json:"display_settings,omitempty"`
	// CreatedAt is the placement creation date-time.
	CreatedAt Timestamp `json:"created_at"`
	// UpdatedAt is the placement last-update date-time.
	UpdatedAt Timestamp `json:"updated_at"`
}
