package models

// FieldDefinition is the canonical field definition object.
type FieldDefinition struct {
	Resource     string  `json:"resource,omitempty"`
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Regex        *string `json:"regex,omitempty"`
	IsPreDefined bool    `json:"is_pre_defined"`
	IsActive     bool    `json:"is_active"`
	IsRequired   bool    `json:"is_required"`
	IsStandard   bool    `json:"is_standard"`
	IsReadOnly   bool    `json:"is_read_only"`
	IsVisible    bool    `json:"is_visible"`
}

// CreateFieldDefinitionRequest is the body for POST /accounts/{id}/fields.
type CreateFieldDefinitionRequest struct {
	Type       string  `json:"type"`
	Name       string  `json:"name"`
	Regex      *string `json:"regex,omitempty"`
	IsRequired *bool   `json:"is_required,omitempty"`
	IsActive   *bool   `json:"is_active,omitempty"`
}

// UpdateFieldDefinitionRequest is the body for PUT /accounts/{id}/fields/{fid}.
type UpdateFieldDefinitionRequest struct {
	Type       *string `json:"type,omitempty"`
	Name       *string `json:"name,omitempty"`
	Regex      *string `json:"regex,omitempty"`
	IsRequired *bool   `json:"is_required,omitempty"`
	IsActive   *bool   `json:"is_active,omitempty"`
}

// ListFieldDefinitionsParams are the query parameters for GET /accounts/{id}/fields.
type ListFieldDefinitionsParams struct {
	IncludeInactive bool
	IncludeStandard bool
}

// ValidateFieldRequest is the body for POST /accounts/{id}/fields/{fid}/validate.
// Value accepts any JSON value (string, number, boolean, …) per the spec.
type ValidateFieldRequest struct {
	Value any `json:"value"`
}

// ValidateMultipleFieldsRequest is one entry of the body for
// POST /accounts/{id}/fields/validate-multiple. Value accepts any JSON value.
type ValidateMultipleFieldsRequest struct {
	FieldID string `json:"field_id"`
	Value   any    `json:"value"`
}

// FieldValidationResult is a single validation result entry.
type FieldValidationResult struct {
	FieldID      string `json:"field_id,omitempty"`
	Type         string `json:"type"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message"`
}

// FieldType describes one entry of GET /field-types.
type FieldType struct {
	Type string `json:"type"`
	Name string `json:"name"`
}
