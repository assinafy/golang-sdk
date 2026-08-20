package resources

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// FieldResource exposes account field-definition and value-validation endpoints.
// Its methods follow the package-level account-selection and error contract.
type FieldResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewFieldResource constructs a FieldResource.
func NewFieldResource(httpClient *internal.HTTPClient, accountID string) *FieldResource {
	return &FieldResource{http: httpClient, accountID: accountID}
}

// Create sends CreateFieldDefinitionRequest and returns the created
// FieldDefinition. It requires client authentication; an empty accountID uses
// the configured default.
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

// List returns FieldDefinition payloads filtered by optional
// ListFieldDefinitionsParams. It requires client authentication; an empty
// accountID uses the configured default.
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

// Get returns one authenticated FieldDefinition payload. An empty accountID uses
// the configured default.
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

// Update sends UpdateFieldDefinitionRequest and returns the updated
// FieldDefinition. It requires client authentication; an empty accountID uses
// the configured default and nil request fields remain unchanged.
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

// Delete removes an authenticated field definition and returns only error,
// discarding the documented empty data array. An empty accountID uses the
// configured default; definitions still in use may produce an API error.
// DELETE /accounts/{account_id}/fields/{field_id}.
func (r *FieldResource) Delete(ctx context.Context, accountID, fieldID string) error {
	accountID = resolveAccountID(accountID, r.accountID)

	path := "/accounts/" + url.PathEscape(accountID) + "/fields/" + url.PathEscape(fieldID)
	_, err := r.http.NewRequest(http.MethodDelete, path).Execute(ctx, nil)
	return err
}

// Validate is the signer-code compatibility form of field validation. It sends
// ValidateFieldRequest and returns FieldValidationResult; an empty accountID uses
// the configured default. The current OpenAPI requires client authentication and
// does not declare signerAccessCode; prefer ValidateAuthenticated for that contract.
// POST /accounts/{account_id}/fields/{field_id}/validate.
//
// Deprecated: use ValidateAuthenticated for the documented authentication shape.
func (r *FieldResource) Validate(ctx context.Context, accountID, fieldID, signerAccessCode string, body *models.ValidateFieldRequest) (*models.FieldValidationResult, error) {
	return r.validate(ctx, accountID, fieldID, signerAccessCode, body)
}

// ValidateAuthenticated sends ValidateFieldRequest using the client's API key or
// bearer token and returns FieldValidationResult. An empty accountID uses the default.
// POST /accounts/{account_id}/fields/{field_id}/validate.
func (r *FieldResource) ValidateAuthenticated(ctx context.Context, accountID, fieldID string, body *models.ValidateFieldRequest) (*models.FieldValidationResult, error) {
	return r.validate(ctx, accountID, fieldID, "", body)
}

func (r *FieldResource) validate(ctx context.Context, accountID, fieldID, signerAccessCode string, body *models.ValidateFieldRequest) (*models.FieldValidationResult, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.FieldValidationResult
	path := "/accounts/" + url.PathEscape(accountID) + "/fields/" + url.PathEscape(fieldID) + "/validate"
	req := r.http.NewRequest(http.MethodPost, path).
		WithQuery("signer-access-code", signerAccessCode).
		WithBody(body)
	if signerAccessCode != "" {
		req.WithoutAuth()
	}
	_, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ValidateMultiple is the signer-code compatibility form of bulk validation. It
// sends ValidateMultipleFieldsRequest values and returns FieldValidationResult
// values; an empty accountID uses the default. The current OpenAPI requires client
// authentication; prefer ValidateMultipleAuthenticated for that contract.
// POST /accounts/{account_id}/fields/validate-multiple.
//
// Deprecated: use ValidateMultipleAuthenticated for the documented authentication shape.
func (r *FieldResource) ValidateMultiple(ctx context.Context, accountID, signerAccessCode string, body []models.ValidateMultipleFieldsRequest) ([]models.FieldValidationResult, error) {
	return r.validateMultiple(ctx, accountID, signerAccessCode, body)
}

// ValidateMultipleAuthenticated sends an array of ValidateMultipleFieldsRequest
// values using the client's API key or bearer token and returns validation
// results. An empty accountID uses the configured default.
// POST /accounts/{account_id}/fields/validate-multiple.
func (r *FieldResource) ValidateMultipleAuthenticated(ctx context.Context, accountID string, body []models.ValidateMultipleFieldsRequest) ([]models.FieldValidationResult, error) {
	return r.validateMultiple(ctx, accountID, "", body)
}

func (r *FieldResource) validateMultiple(ctx context.Context, accountID, signerAccessCode string, body []models.ValidateMultipleFieldsRequest) ([]models.FieldValidationResult, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out []models.FieldValidationResult
	path := "/accounts/" + url.PathEscape(accountID) + "/fields/validate-multiple"
	req := r.http.NewRequest(http.MethodPost, path).
		WithQuery("signer-access-code", signerAccessCode).
		WithBody(body)
	if signerAccessCode != "" {
		req.WithoutAuth()
	}
	_, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListTypes uses client authentication and returns the available FieldType payloads.
// GET /field-types.
func (r *FieldResource) ListTypes(ctx context.Context) ([]models.FieldType, error) {
	var out []models.FieldType
	_, err := r.http.NewRequest(http.MethodGet, "/field-types").Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
