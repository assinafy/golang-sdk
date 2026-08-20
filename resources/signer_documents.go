package resources

import (
	"context"
	"net/http"
	"net/url"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// SignerDocumentResource exposes signer-facing document endpoints. Most methods
// authenticate with a signer access code rather than a client credential; Download
// is public in the current OpenAPI. All follow the package-level error contract.
type SignerDocumentResource struct {
	http *internal.HTTPClient
}

// NewSignerDocumentResource constructs a SignerDocumentResource.
func NewSignerDocumentResource(httpClient *internal.HTTPClient) *SignerDocumentResource {
	return &SignerDocumentResource{http: httpClient}
}

// GetCurrent authenticates with signerAccessCode and returns the signer's current
// Document payload. An invalid or expired code produces an API error.
// GET /signers/{signer_id}/document.
func (r *SignerDocumentResource) GetCurrent(ctx context.Context, signerID, signerAccessCode string) (*models.Document, error) {
	var out models.Document
	_, err := r.http.NewRequest(http.MethodGet, "/signers/"+url.PathEscape(signerID)+"/document").
		WithoutAuth().
		WithQuery("signer-access-code", signerAccessCode).
		Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List authenticates with signerAccessCode and returns accessible Document
// payloads with X-Pagination metadata. Page and per-page are documented;
// populated shared search, sort, status, and method fields are compatibility
// queries.
// GET /signers/{signer_id}/documents.
func (r *SignerDocumentResource) List(ctx context.Context, signerID, signerAccessCode string, params *models.ListParams) (*models.PaginatedResult[models.Document], error) {
	var out []models.Document
	req := r.http.NewRequest(http.MethodGet, "/signers/"+url.PathEscape(signerID)+"/documents").
		WithoutAuth().
		WithQuery("signer-access-code", signerAccessCode)
	applyListParams(req, params)
	if params != nil {
		req.WithQuery("status", params.Status)
		req.WithQuery("method", params.Method)
	}

	resp, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return paginated(out, resp), nil
}

// Search returns the documents accessible to a signer that match
// params.Search, authenticated by signerAccessCode and returned with
// X-Pagination metadata. Other generic pagination parameters are also sent.
// GET /signers/{signer_id}/documents/search.
//
// Deprecated: use SearchAll for the documented query and non-paginated result.
func (r *SignerDocumentResource) Search(ctx context.Context, signerID, signerAccessCode string, params *models.ListParams) (*models.PaginatedResult[models.Document], error) {
	var out []models.Document
	req := r.newSearchRequest(signerID, signerAccessCode)
	applyListParams(req, params)

	resp, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return paginated(out, resp), nil
}

// SearchAll authenticates with signerAccessCode and returns all Document payloads
// matching search using only the query parameter documented for this endpoint.
// GET /signers/{signer_id}/documents/search.
func (r *SignerDocumentResource) SearchAll(ctx context.Context, signerID, signerAccessCode, search string) ([]models.Document, error) {
	var out []models.Document
	_, err := r.newSearchRequest(signerID, signerAccessCode).
		WithQuery("search", search).
		Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SignerDocumentResource) newSearchRequest(signerID, signerAccessCode string) *internal.Request {
	return r.http.NewRequest(http.MethodGet, "/signers/"+url.PathEscape(signerID)+"/documents/search").
		WithoutAuth().
		WithQuery("signer-access-code", signerAccessCode)
}

// SignMultiple authenticates with signerAccessCode, sends the required
// document_ids body, signs the eligible documents as a batch, and returns only
// error, discarding the documented empty data array.
// PUT /signers/documents/sign-multiple.
func (r *SignerDocumentResource) SignMultiple(ctx context.Context, signerAccessCode string, documentIDs []string) error {
	body := map[string][]string{"document_ids": documentIDs}
	_, err := r.http.NewRequest(http.MethodPut, "/signers/documents/sign-multiple").
		WithoutAuth().
		WithQuery("signer-access-code", signerAccessCode).
		WithBody(body).
		Execute(ctx, nil)
	return err
}

// DeclineMultiple authenticates with signerAccessCode, sends document_ids and the
// required decline_reason, rejects the eligible documents, and returns an
// error-only result after discarding the documented empty data array.
// PUT /signers/documents/decline-multiple.
func (r *SignerDocumentResource) DeclineMultiple(ctx context.Context, signerAccessCode string, documentIDs []string, reason string) error {
	body := map[string]any{
		"document_ids":   documentIDs,
		"decline_reason": reason,
	}
	_, err := r.http.NewRequest(http.MethodPut, "/signers/documents/decline-multiple").
		WithoutAuth().
		WithQuery("signer-access-code", signerAccessCode).
		WithBody(body).
		Execute(ctx, nil)
	return err
}

// Download returns raw bytes for artifact ("original", "certificated",
// "certificate-page", "pades", or "bundle"). The current OpenAPI declares this
// operation public, so pass an empty signerAccessCode; a non-empty value is sent
// only for compatibility. Unknown documents or artifacts produce API errors.
// GET /signers/{signer_id}/documents/{document_id}/download/{artifact}.
func (r *SignerDocumentResource) Download(ctx context.Context, signerID, documentID, artifact, signerAccessCode string) ([]byte, error) {
	path := "/signers/" + url.PathEscape(signerID) +
		"/documents/" + url.PathEscape(documentID) +
		"/download/" + url.PathEscape(artifact)
	return r.http.DownloadUnauthenticated(ctx, path, map[string]string{"signer-access-code": signerAccessCode})
}
