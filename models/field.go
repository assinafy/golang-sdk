package models

import "encoding/json"

// DisplaySettings positions and styles a field on Assinafy's 150-DPI page
// image coordinate system. Left and Top may be zero; Width, Height, and
// FontSize must be greater than zero.
//
// Existing placement fields remain typed as any for source compatibility, but
// callers can use DisplaySettings as their concrete value.
type DisplaySettings struct {
	// Left is the required horizontal offset from the page's left edge.
	Left float64 `json:"left"`
	// Top is the required vertical offset from the page's top edge.
	Top float64 `json:"top"`
	// Width is the required field width and must be greater than zero.
	Width float64 `json:"width"`
	// Height is the required field height and must be greater than zero.
	Height float64 `json:"height"`
	// FontFamily optionally selects the rendered font family.
	FontFamily string `json:"fontFamily,omitempty"`
	// FontSize is the required rendered font size and must be greater than zero.
	FontSize float64 `json:"fontSize"`
	// BackgroundColor is an optional CSS-compatible background color.
	BackgroundColor string `json:"backgroundColor,omitempty"`
}

// FieldDefinition is the canonical field definition object.
type FieldDefinition struct {
	// Resource is the API resource discriminator when present.
	Resource string `json:"resource,omitempty"`
	// ID is the field-definition identifier.
	ID string `json:"id"`
	// Name is the human-readable field name.
	Name string `json:"name"`
	// Type is the API field-type code returned by GET /field-types.
	Type string `json:"type"`
	// Regex is the nullable validation expression for custom fields.
	Regex *string `json:"regex,omitempty"`
	// IsPreDefined reports whether Assinafy supplied the definition.
	IsPreDefined bool `json:"is_pre_defined"`
	// IsActive reports whether the definition may be used for new placements.
	IsActive bool `json:"is_active"`
	// IsRequired reports whether a signer must provide the field value.
	IsRequired bool `json:"is_required"`
	// IsStandard reports whether this is a standard Assinafy field.
	IsStandard bool `json:"is_standard"`
	// IsReadOnly reports whether signers may edit the rendered value.
	IsReadOnly bool `json:"is_read_only"`
	// IsVisible reports whether the field is rendered to the signer.
	IsVisible bool `json:"is_visible"`
}

// CreateFieldDefinitionRequest is the body for POST /accounts/{id}/fields.
type CreateFieldDefinitionRequest struct {
	// Type is the required code from GET /field-types.
	Type string `json:"type"`
	// Name is the required display name.
	Name string `json:"name"`
	// Regex is an optional validation expression; nil omits it.
	Regex *string `json:"regex,omitempty"`
	// ClearRegex sends regex:null. On creation, this explicitly selects no regex.
	ClearRegex bool `json:"-"`
	// IsRequired optionally overrides the API's required-value default.
	IsRequired *bool `json:"is_required,omitempty"`
	// IsActive optionally overrides the API's active-state default.
	IsActive *bool `json:"is_active,omitempty"`
}

// MarshalJSON preserves the API's three regex states: omitted, string, or null.
func (r CreateFieldDefinitionRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type       string `json:"type"`
		Name       string `json:"name"`
		Regex      any    `json:"regex,omitempty"`
		IsRequired *bool  `json:"is_required,omitempty"`
		IsActive   *bool  `json:"is_active,omitempty"`
	}{
		Type:       r.Type,
		Name:       r.Name,
		Regex:      regexRequestValue(r.Regex, r.ClearRegex),
		IsRequired: r.IsRequired,
		IsActive:   r.IsActive,
	})
}

// UpdateFieldDefinitionRequest is the body for PUT /accounts/{id}/fields/{fid}.
type UpdateFieldDefinitionRequest struct {
	// Type optionally changes the field-type code; nil leaves it unchanged.
	Type *string `json:"type,omitempty"`
	// Name optionally changes the display name; nil leaves it unchanged.
	Name *string `json:"name,omitempty"`
	// Regex optionally changes the validation expression; nil leaves it unchanged
	// unless ClearRegex is true.
	Regex *string `json:"regex,omitempty"`
	// IsRequired optionally changes required-value behavior; nil leaves it unchanged.
	IsRequired *bool `json:"is_required,omitempty"`
	// IsActive optionally changes active state; nil leaves it unchanged.
	IsActive *bool `json:"is_active,omitempty"`
	// ClearRegex sends regex:null. It takes precedence over Regex.
	ClearRegex bool `json:"-"`
}

// MarshalJSON preserves the API's three regex states: omitted, string, or null.
func (r UpdateFieldDefinitionRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type       *string `json:"type,omitempty"`
		Name       *string `json:"name,omitempty"`
		Regex      any     `json:"regex,omitempty"`
		IsRequired *bool   `json:"is_required,omitempty"`
		IsActive   *bool   `json:"is_active,omitempty"`
	}{
		Type:       r.Type,
		Name:       r.Name,
		Regex:      regexRequestValue(r.Regex, r.ClearRegex),
		IsRequired: r.IsRequired,
		IsActive:   r.IsActive,
	})
}

func regexRequestValue(regex *string, clear bool) any {
	if clear {
		return (*string)(nil)
	}
	if regex != nil {
		return *regex
	}
	return nil
}

// ListFieldDefinitionsParams are the query parameters for GET /accounts/{id}/fields.
type ListFieldDefinitionsParams struct {
	// IncludeInactive includes definitions whose IsActive value is false.
	IncludeInactive bool
	// IncludeStandard includes Assinafy's standard field definitions.
	IncludeStandard bool
}

// ValidateFieldRequest is the body for POST /accounts/{id}/fields/{fid}/validate.
// Value accepts any JSON value (string, number, boolean, …) per the spec.
type ValidateFieldRequest struct {
	// Value is the required JSON value to validate against the target definition.
	Value any `json:"value"`
}

// ValidateMultipleFieldsRequest is one entry of the body for
// POST /accounts/{id}/fields/validate-multiple. Value accepts any JSON value.
type ValidateMultipleFieldsRequest struct {
	// FieldID is the required field-definition identifier.
	FieldID string `json:"field_id"`
	// Value is the required JSON value to validate against FieldID.
	Value any `json:"value"`
}

// FieldValidationResult is a single validation result entry.
type FieldValidationResult struct {
	// FieldID identifies the validated definition when returned by bulk validation.
	FieldID string `json:"field_id,omitempty"`
	// Type is the field-type code used for validation.
	Type string `json:"type"`
	// Success reports whether the supplied value passed validation.
	Success bool `json:"success"`
	// ErrorMessage is empty on success and explains a validation failure otherwise.
	ErrorMessage string `json:"error_message"`
}

// FieldType describes one entry of GET /field-types.
type FieldType struct {
	// Type is the machine-readable field-type code used in request payloads.
	Type string `json:"type"`
	// Name is the human-readable field-type label.
	Name string `json:"name"`
}
