package resources

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// FieldResource exposes the documented `Field Definition` endpoints.
type FieldResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewFieldResource constructs a FieldResource.
func NewFieldResource(httpClient *internal.HTTPClient, accountID string) *FieldResource {
	return &FieldResource{http: httpClient, accountID: accountID}
}

// Create defines a new field.
// POST /accounts/{account_id}/fields.
func (r *FieldResource) Create(ctx context.Context, accountID string, body *models.CreateFieldDefinitionRequest) (*models.FieldDefinition, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.FieldDefinition
	path := "/accounts/" + url.PathEscape(accountID) + "/fields"
	_, err := r.http.NewRequest(http.MethodPost, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the workspace field definitions.
// GET /accounts/{account_id}/fields.
func (r *FieldResource) List(ctx context.Context, accountID string, params *models.ListFieldDefinitionsParams) ([]models.FieldDefinition, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out []models.FieldDefinition
	req := r.http.NewRequest(http.MethodGet, "/accounts/"+url.PathEscape(accountID)+"/fields")
	if params != nil {
		req.WithQuery("include_inactive", strconv.FormatBool(params.IncludeInactive))
		req.WithQuery("include_standard", strconv.FormatBool(params.IncludeStandard))
	}
	if _, err := req.Execute(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get retrieves a field definition.
// GET /accounts/{account_id}/fields/{field_id}.
func (r *FieldResource) Get(ctx context.Context, accountID, fieldID string) (*models.FieldDefinition, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.FieldDefinition
	path := "/accounts/" + url.PathEscape(accountID) + "/fields/" + url.PathEscape(fieldID)
	_, err := r.http.NewRequest(http.MethodGet, path).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Update updates a field definition.
// PUT /accounts/{account_id}/fields/{field_id}.
func (r *FieldResource) Update(ctx context.Context, accountID, fieldID string, body *models.UpdateFieldDefinitionRequest) (*models.FieldDefinition, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.FieldDefinition
	path := "/accounts/" + url.PathEscape(accountID) + "/fields/" + url.PathEscape(fieldID)
	_, err := r.http.NewRequest(http.MethodPut, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes a field definition.
// DELETE /accounts/{account_id}/fields/{field_id}.
func (r *FieldResource) Delete(ctx context.Context, accountID, fieldID string) error {
	accountID = resolveAccountID(accountID, r.accountID)

	path := "/accounts/" + url.PathEscape(accountID) + "/fields/" + url.PathEscape(fieldID)
	_, err := r.http.NewRequest(http.MethodDelete, path).Execute(ctx, nil)
	return err
}

// Validate validates a single field value (signer-facing flow).
// POST /accounts/{account_id}/fields/{field_id}/validate.
func (r *FieldResource) Validate(ctx context.Context, accountID, fieldID, signerAccessCode string, body *models.ValidateFieldRequest) (*models.FieldValidationResult, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.FieldValidationResult
	path := "/accounts/" + url.PathEscape(accountID) + "/fields/" + url.PathEscape(fieldID) + "/validate"
	_, err := r.http.NewRequest(http.MethodPost, path).
		WithQuery("signer-access-code", signerAccessCode).
		WithBody(body).
		Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ValidateMultiple validates multiple fields at once (signer-facing flow).
// POST /accounts/{account_id}/fields/validate-multiple.
func (r *FieldResource) ValidateMultiple(ctx context.Context, accountID, signerAccessCode string, body []models.ValidateMultipleFieldsRequest) ([]models.FieldValidationResult, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out []models.FieldValidationResult
	path := "/accounts/" + url.PathEscape(accountID) + "/fields/validate-multiple"
	_, err := r.http.NewRequest(http.MethodPost, path).
		WithQuery("signer-access-code", signerAccessCode).
		WithBody(body).
		Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListTypes returns the available field types.
// GET /field-types.
func (r *FieldResource) ListTypes(ctx context.Context) ([]models.FieldType, error) {
	var out []models.FieldType
	_, err := r.http.NewRequest(http.MethodGet, "/field-types").Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
