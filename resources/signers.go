package resources

import (
	"context"
	"net/http"
	"net/url"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// SignerResource exposes the documented `Signer` endpoints.
type SignerResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewSignerResource constructs a SignerResource bound to the given HTTPClient
// and default account ID.
func NewSignerResource(httpClient *internal.HTTPClient, accountID string) *SignerResource {
	return &SignerResource{http: httpClient, accountID: accountID}
}

// Create registers a new signer in the workspace.
// POST /accounts/{account_id}/signers.
func (r *SignerResource) Create(ctx context.Context, accountID string, body *models.CreateSignerRequest) (*models.Signer, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.Signer
	path := "/accounts/" + url.PathEscape(accountID) + "/signers"
	_, err := r.http.NewRequest(http.MethodPost, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the workspace signers page.
// GET /accounts/{account_id}/signers.
func (r *SignerResource) List(ctx context.Context, accountID string, params *models.ListParams) (*models.PaginatedResult[models.Signer], error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out []models.Signer
	req := r.http.NewRequest(http.MethodGet, "/accounts/"+url.PathEscape(accountID)+"/signers")
	applyListParams(req, params)

	resp, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return paginated(out, resp), nil
}

// Get retrieves a signer.
// GET /accounts/{account_id}/signers/{signer_id}.
func (r *SignerResource) Get(ctx context.Context, accountID, signerID string) (*models.Signer, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.Signer
	path := "/accounts/" + url.PathEscape(accountID) + "/signers/" + url.PathEscape(signerID)
	_, err := r.http.NewRequest(http.MethodGet, path).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Update updates a signer.
// PUT /accounts/{account_id}/signers/{signer_id}.
func (r *SignerResource) Update(ctx context.Context, accountID, signerID string, body *models.UpdateSignerRequest) (*models.Signer, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.Signer
	path := "/accounts/" + url.PathEscape(accountID) + "/signers/" + url.PathEscape(signerID)
	_, err := r.http.NewRequest(http.MethodPut, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete removes a signer.
// DELETE /accounts/{account_id}/signers/{signer_id}.
func (r *SignerResource) Delete(ctx context.Context, accountID, signerID string) error {
	accountID = resolveAccountID(accountID, r.accountID)

	path := "/accounts/" + url.PathEscape(accountID) + "/signers/" + url.PathEscape(signerID)
	_, err := r.http.NewRequest(http.MethodDelete, path).Execute(ctx, nil)
	return err
}

// GetSelf returns the currently authenticated signer (signer-facing flow).
// GET /signers/self.
func (r *SignerResource) GetSelf(ctx context.Context, signerAccessCode string) (*models.Signer, error) {
	var out models.Signer
	_, err := r.http.NewRequest(http.MethodGet, "/signers/self").
		WithQuery("signer-access-code", signerAccessCode).
		Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// AcceptTerms accepts terms of use for a signer (signer-facing flow).
// PUT /signers/accept-terms.
func (r *SignerResource) AcceptTerms(ctx context.Context, signerAccessCode string) (*models.Signer, error) {
	var out models.Signer
	body := map[string]string{"signer-access-code": signerAccessCode}
	_, err := r.http.NewRequest(http.MethodPut, "/signers/accept-terms").
		WithBody(body).
		Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// VerifyEmail verifies an emailed access code (signer-facing flow).
// POST /verify.
func (r *SignerResource) VerifyEmail(ctx context.Context, signerAccessCode, verificationCode string) error {
	body := map[string]string{
		"signer-access-code": signerAccessCode,
		"verification-code":  verificationCode,
	}
	_, err := r.http.NewRequest(http.MethodPost, "/verify").WithBody(body).Execute(ctx, nil)
	return err
}

// ConfirmData updates a signer's contact data within a signing session.
// PUT /documents/{document_id}/signers/confirm-data.
func (r *SignerResource) ConfirmData(ctx context.Context, documentID, signerAccessCode string, body *models.ConfirmSignerDataRequest) error {
	path := "/documents/" + url.PathEscape(documentID) + "/signers/confirm-data"
	_, err := r.http.NewRequest(http.MethodPut, path).
		WithQuery("signer-access-code", signerAccessCode).
		WithBody(body).
		Execute(ctx, nil)
	return err
}

// UploadSignature uploads a signature/initial image (defaults to image/png).
// POST /signature.
func (r *SignerResource) UploadSignature(ctx context.Context, signerAccessCode, signatureType string, image []byte) error {
	return r.UploadSignatureWithContentType(ctx, signerAccessCode, signatureType, "image/png", image)
}

// UploadSignatureWithContentType uploads a signature/initial image with a custom content type.
// POST /signature.
func (r *SignerResource) UploadSignatureWithContentType(ctx context.Context, signerAccessCode, signatureType, contentType string, image []byte) error {
	_, err := r.http.NewRequest(http.MethodPost, "/signature").
		WithQuery("signer-access-code", signerAccessCode).
		WithQuery("type", signatureType).
		WithHeader("Content-Type", contentType).
		WithRawBody(image).
		Execute(ctx, nil)
	return err
}

// DownloadSignature downloads a signature/initial image.
// GET /signature/{type}.
func (r *SignerResource) DownloadSignature(ctx context.Context, signerAccessCode, signatureType string) ([]byte, error) {
	return r.http.Download(ctx, "/signature/"+url.PathEscape(signatureType), map[string]string{"signer-access-code": signerAccessCode})
}
