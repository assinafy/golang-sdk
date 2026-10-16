package resources

import (
	"context"
	"net/http"
	"net/url"

	"github.com/assinafy/golang-sdk/v2/internal"
	"github.com/assinafy/golang-sdk/v2/models"
)

// SignerDocumentResource exposes signer-facing document endpoints.
//
// Methods that take a signerAccessCode authenticate with that code alone: the
// client's API key or bearer token is never sent with them, and an empty code
// is rejected with errors.ErrInvalidInput before any request is made. Workspace
// access to a signer's documents goes through the explicit ListAsWorkspace and
// SearchAsWorkspace methods, and DownloadPublic needs no credential at all.
type SignerDocumentResource struct {
	http *internal.HTTPClient
}

// NewSignerDocumentResource constructs a SignerDocumentResource.
func NewSignerDocumentResource(httpClient *internal.HTTPClient) *SignerDocumentResource {
	return &SignerDocumentResource{http: httpClient}
}

// GetCurrent authenticates with signerAccessCode and returns the Document the
// signer is currently signing. An invalid or expired code produces an API error.
//
// GET /signers/{signer_id}/document.
func (r *SignerDocumentResource) GetCurrent(ctx context.Context, signerID, signerAccessCode string) (*models.Document, error) {
	var out models.Document
	_, err := r.http.NewRequest(http.MethodGet, "/signers/"+url.PathEscape(signerID)+"/document").
		WithSignerAccessCode(signerAccessCode).
		Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List authenticates with signerAccessCode and returns the Documents the signer
// can access, with X-Pagination metadata. Page and per-page are documented;
// populated search, sort, status and method fields are sent as well.
//
// GET /signers/{signer_id}/documents.
func (r *SignerDocumentResource) List(ctx context.Context, signerID, signerAccessCode string, params *models.ListParams) (*models.PaginatedResult[models.Document], error) {
	return r.list(ctx, r.listRequest(signerID).WithSignerAccessCode(signerAccessCode), params)
}

// ListAsWorkspace returns the Documents of one of the workspace's signers using
// the client's API key or bearer token, with X-Pagination metadata. It sends
// the same query parameters as List.
//
// GET /signers/{signer_id}/documents.
func (r *SignerDocumentResource) ListAsWorkspace(ctx context.Context, signerID string, params *models.ListParams) (*models.PaginatedResult[models.Document], error) {
	return r.list(ctx, r.listRequest(signerID), params)
}

func (r *SignerDocumentResource) listRequest(signerID string) *internal.Request {
	return r.http.NewRequest(http.MethodGet, "/signers/"+url.PathEscape(signerID)+"/documents")
}

func (r *SignerDocumentResource) list(ctx context.Context, req *internal.Request, params *models.ListParams) (*models.PaginatedResult[models.Document], error) {
	applyListParams(req, params)
	if params != nil {
		req.WithQuery("status", params.Status)
		req.WithQuery("method", params.Method)
	}
	var out []models.Document
	resp, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return paginated(out, resp), nil
}

// Search returns the documents accessible to a signer that match
// params.Search, authenticated by signerAccessCode and returned with
// X-Pagination metadata. Other generic pagination parameters are also sent.
//
// GET /signers/{signer_id}/documents/search.
//
// Deprecated: use SearchAll for the documented query and non-paginated result.
func (r *SignerDocumentResource) Search(ctx context.Context, signerID, signerAccessCode string, params *models.ListParams) (*models.PaginatedResult[models.Document], error) {
	var out []models.Document
	req := r.searchRequest(signerID).WithSignerAccessCode(signerAccessCode)
	applyListParams(req, params)

	resp, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return paginated(out, resp), nil
}

// SearchAll authenticates with signerAccessCode and returns every Document the
// signer can access that matches search.
//
// GET /signers/{signer_id}/documents/search?search={search}.
func (r *SignerDocumentResource) SearchAll(ctx context.Context, signerID, signerAccessCode, search string) ([]models.Document, error) {
	return r.search(ctx, r.searchRequest(signerID).WithSignerAccessCode(signerAccessCode), search)
}

// SearchAsWorkspace returns every Document of one of the workspace's signers
// that matches search, using the client's API key or bearer token.
//
// GET /signers/{signer_id}/documents/search?search={search}.
func (r *SignerDocumentResource) SearchAsWorkspace(ctx context.Context, signerID, search string) ([]models.Document, error) {
	return r.search(ctx, r.searchRequest(signerID), search)
}

func (r *SignerDocumentResource) searchRequest(signerID string) *internal.Request {
	return r.http.NewRequest(http.MethodGet, "/signers/"+url.PathEscape(signerID)+"/documents/search")
}

func (r *SignerDocumentResource) search(ctx context.Context, req *internal.Request, search string) ([]models.Document, error) {
	var out []models.Document
	if _, err := req.WithQuery("search", search).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SignMultiple authenticates with signerAccessCode and signs the eligible
// documents as a batch. Request: {"document_ids": [string]}. The response data
// is an empty array, so only an error is returned.
//
// PUT /signers/documents/sign-multiple.
func (r *SignerDocumentResource) SignMultiple(ctx context.Context, signerAccessCode string, documentIDs []string) error {
	body := map[string][]string{"document_ids": nonNil(documentIDs)}
	_, err := r.http.NewRequest(http.MethodPut, "/signers/documents/sign-multiple").
		WithSignerAccessCode(signerAccessCode).
		WithBody(body).
		Execute(ctx, nil)
	return err
}

// DeclineMultiple authenticates with signerAccessCode and declines the eligible
// documents. Request: {"document_ids": [string], "decline_reason": string},
// both required. The response data is an empty array, so only an error is
// returned.
//
// PUT /signers/documents/decline-multiple.
func (r *SignerDocumentResource) DeclineMultiple(ctx context.Context, signerAccessCode string, documentIDs []string, reason string) error {
	body := map[string]any{
		"document_ids":   nonNil(documentIDs),
		"decline_reason": reason,
	}
	_, err := r.http.NewRequest(http.MethodPut, "/signers/documents/decline-multiple").
		WithSignerAccessCode(signerAccessCode).
		WithBody(body).
		Execute(ctx, nil)
	return err
}

// Download returns the raw bytes of artifact ("original", "certificated",
// "certificate-page", "pades" or "bundle"), sending signerAccessCode and no
// client credential. An empty code is rejected; use DownloadPublic to call the
// operation without one.
//
// GET /signers/{signer_id}/documents/{document_id}/download/{artifact}.
func (r *SignerDocumentResource) Download(ctx context.Context, signerID, documentID, artifact, signerAccessCode string) ([]byte, error) {
	return r.http.DownloadAsSigner(ctx, signerArtifactPath(signerID, documentID, artifact), signerAccessCode)
}

// DownloadPublic returns the raw bytes of artifact ("original", "certificated",
// "certificate-page", "pades" or "bundle") without any credential, as the API
// contract declares this operation public. Unknown documents or artifacts
// produce API errors.
//
// GET /signers/{signer_id}/documents/{document_id}/download/{artifact}.
func (r *SignerDocumentResource) DownloadPublic(ctx context.Context, signerID, documentID, artifact string) ([]byte, error) {
	return r.http.DownloadUnauthenticated(ctx, signerArtifactPath(signerID, documentID, artifact), nil)
}

func signerArtifactPath(signerID, documentID, artifact string) string {
	return "/signers/" + url.PathEscape(signerID) +
		"/documents/" + url.PathEscape(documentID) +
		"/download/" + url.PathEscape(artifact)
}
