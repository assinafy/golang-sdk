package resources

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// SignerResource exposes authenticated account-signer operations and
// signer-access-code session operations. Its methods follow the package-level
// account-selection and error contract.
type SignerResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewSignerResource constructs a SignerResource bound to the given HTTPClient
// and default account ID.
func NewSignerResource(httpClient *internal.HTTPClient, accountID string) *SignerResource {
	return &SignerResource{http: httpClient, accountID: accountID}
}

// Create sends CreateSignerRequest and returns the created Signer. It requires
// client authentication; an empty accountID uses the configured default.
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

// List returns Signer payloads and X-Pagination metadata using ListParams. Sort
// is a compatibility query; the other populated fields applicable to this route
// follow the current contract. It requires client authentication; an empty
// accountID uses the configured default.
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

// Get returns one authenticated Signer payload. An empty accountID uses the
// configured default.
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

// Update sends UpdateSignerRequest and returns the updated Signer. It requires
// client authentication; an empty accountID uses the default and nil request
// fields leave their attributes unchanged.
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

// Delete removes an authenticated workspace signer and returns only error,
// discarding the documented empty data array. An empty accountID uses the
// configured default; a signer still in use may produce an API error.
// DELETE /accounts/{account_id}/signers/{signer_id}.
func (r *SignerResource) Delete(ctx context.Context, accountID, signerID string) error {
	accountID = resolveAccountID(accountID, r.accountID)

	path := "/accounts/" + url.PathEscape(accountID) + "/signers/" + url.PathEscape(signerID)
	_, err := r.http.NewRequest(http.MethodDelete, path).Execute(ctx, nil)
	return err
}

// GetSelf authenticates with signerAccessCode and returns the current Signer,
// including saved-signature flags, without requiring a client credential.
// GET /signers/self.
func (r *SignerResource) GetSelf(ctx context.Context, signerAccessCode string) (*models.Signer, error) {
	var out models.Signer
	_, err := r.http.NewRequest(http.MethodGet, "/signers/self").
		WithoutAuth().
		WithQuery("signer-access-code", signerAccessCode).
		Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// AcceptTerms authenticates with signerAccessCode and records terms acceptance.
// It retains the legacy Signer result; the current contract has no data payload,
// so the returned Signer can be zero-valued.
// PUT /signers/accept-terms.
//
// Deprecated: use AcceptTermsOnly for the current envelope-only response.
func (r *SignerResource) AcceptTerms(ctx context.Context, signerAccessCode string) (*models.Signer, error) {
	var out models.Signer
	if err := r.acceptTerms(ctx, signerAccessCode, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AcceptTermsOnly authenticates with signerAccessCode, records terms acceptance,
// and returns the envelope-only response documented by the current API contract.
// PUT /signers/accept-terms.
func (r *SignerResource) AcceptTermsOnly(ctx context.Context, signerAccessCode string) error {
	return r.acceptTerms(ctx, signerAccessCode, nil)
}

func (r *SignerResource) acceptTerms(ctx context.Context, signerAccessCode string, out any) error {
	_, err := r.http.NewRequest(http.MethodPut, "/signers/accept-terms").
		WithoutAuth().
		WithQuery("signer-access-code", signerAccessCode).
		Execute(ctx, out)
	return err
}

// VerifyEmail verifies an emailed code and returns an envelope-only success. The
// signer credential is supplied as the signer-access-code query parameter; only
// the required verification-code travels in the body.
// POST /verify.
func (r *SignerResource) VerifyEmail(ctx context.Context, signerAccessCode, verificationCode string) error {
	body := map[string]string{"verification-code": verificationCode}
	_, err := r.http.NewRequest(http.MethodPost, "/verify").
		WithoutAuth().
		WithQuery("signer-access-code", signerAccessCode).
		WithBody(body).
		Execute(ctx, nil)
	return err
}

// ConfirmData authenticates with signerAccessCode, sends ConfirmSignerDataRequest,
// updates the signing-session data, and discards the documented Signer response.
// Use ConfirmDataAndGet when the updated payload is needed.
// PUT /documents/{document_id}/signers/confirm-data.
//
// Deprecated: use ConfirmDataAndGet to retain the documented response.
func (r *SignerResource) ConfirmData(ctx context.Context, documentID, signerAccessCode string, body *models.ConfirmSignerDataRequest) error {
	return r.confirmData(ctx, documentID, signerAccessCode, body, nil)
}

// ConfirmDataAndGet updates a signer's contact data and returns the updated
// Signer from the documented response payload. It authenticates with
// signerAccessCode and sends ConfirmSignerDataRequest.
// PUT /documents/{document_id}/signers/confirm-data.
func (r *SignerResource) ConfirmDataAndGet(ctx context.Context, documentID, signerAccessCode string, body *models.ConfirmSignerDataRequest) (*models.Signer, error) {
	var out models.Signer
	if err := r.confirmData(ctx, documentID, signerAccessCode, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *SignerResource) confirmData(ctx context.Context, documentID, signerAccessCode string, body *models.ConfirmSignerDataRequest, out any) error {
	path := "/documents/" + url.PathEscape(documentID) + "/signers/confirm-data"
	_, err := r.http.NewRequest(http.MethodPut, path).
		WithoutAuth().
		WithQuery("signer-access-code", signerAccessCode).
		WithBody(body).
		Execute(ctx, out)
	return err
}

// UploadSignature authenticates with signerAccessCode, sends raw PNG bytes for
// the requested signature type (for example, signature or initial), stores the
// image for this signing process, and returns an envelope-only success.
// POST /signature.
func (r *SignerResource) UploadSignature(ctx context.Context, signerAccessCode, signatureType string, image []byte) error {
	return r.uploadSignature(ctx, signerAccessCode, signatureType, nil, "image/png", image)
}

// UploadSignatureWithContentType authenticates with signerAccessCode and sends
// raw image bytes with a caller-selected content type.
// POST /signature.
//
// Deprecated: the current API contract only documents image/png. Use
// UploadSignature or UploadSignatureWithReuse.
func (r *SignerResource) UploadSignatureWithContentType(ctx context.Context, signerAccessCode, signatureType, contentType string, image []byte) error {
	return r.uploadSignature(ctx, signerAccessCode, signatureType, nil, contentType, image)
}

// UploadSignatureWithReuse uploads a PNG signature/initial image and records
// whether it may be reused in future signing processes. It authenticates with
// signerAccessCode and returns an envelope-only success.
// POST /signature?type={type}&reuse={value}.
func (r *SignerResource) UploadSignatureWithReuse(ctx context.Context, signerAccessCode, signatureType string, reuse bool, image []byte) error {
	return r.uploadSignature(ctx, signerAccessCode, signatureType, &reuse, "image/png", image)
}

func (r *SignerResource) uploadSignature(ctx context.Context, signerAccessCode, signatureType string, reuse *bool, contentType string, image []byte) error {
	req := r.http.NewRequest(http.MethodPost, "/signature").
		WithoutAuth().
		WithQuery("signer-access-code", signerAccessCode).
		WithQuery("type", signatureType).
		WithHeader("Content-Type", contentType).
		WithRawBody(image)
	if reuse != nil {
		req.WithQuery("reuse", strconv.FormatBool(*reuse))
	}
	_, err := req.Execute(ctx, nil)
	return err
}

// DownloadSignature authenticates with signerAccessCode and returns raw bytes for
// the requested signature type. A missing saved image produces an API error.
// GET /signature/{type}.
func (r *SignerResource) DownloadSignature(ctx context.Context, signerAccessCode, signatureType string) ([]byte, error) {
	return r.http.DownloadUnauthenticated(ctx, "/signature/"+url.PathEscape(signatureType), map[string]string{"signer-access-code": signerAccessCode})
}
