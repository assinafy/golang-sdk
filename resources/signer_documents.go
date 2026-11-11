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
	return r.GetCurrentWithParams(ctx, signerID, signerAccessCode, nil)
}

// GetCurrentWithParams returns the signer-facing document using only its access
// code. Params.Expand accepts envelope; other params apply only to GET /sign.
// GET /signers/{signer_id}/document.
func (r *SignerDocumentResource) GetCurrentWithParams(ctx context.Context, signerID, signerAccessCode string, params *models.SigningDocumentParams) (*models.Document, error) {
	var out models.Document
	req := r.http.NewRequest(http.MethodGet, "/signers/"+url.PathEscape(signerID)+"/document").WithSignerAccessCode(signerAccessCode)
	if params != nil {
		req.WithQuery("expand", params.Expand)
	}
	_, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List authenticates with signerAccessCode and returns the Documents the signer
// can access, with X-Pagination metadata. Page and per-page are documented;
// envelope_id filters the envelope; populated search, sort, status and method
// fields are sent as well.
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
		req.WithQuery("envelope_id", params.EnvelopeID)
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
	if params != nil {
		req.WithQuery("envelope_id", params.EnvelopeID)
	}

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
	return r.SearchAllWithParams(ctx, signerID, signerAccessCode, &models.ListParams{Search: search})
}

// SearchAllWithParams returns matching documents using only signerAccessCode.
// Params.Search selects the search text and EnvelopeID limits the envelope;
// other ListParams fields are ignored because the response is not paginated.
// GET /signers/{signer_id}/documents/search.
func (r *SignerDocumentResource) SearchAllWithParams(ctx context.Context, signerID, signerAccessCode string, params *models.ListParams) ([]models.Document, error) {
	return r.search(ctx, r.searchRequest(signerID).WithSignerAccessCode(signerAccessCode), params)
}

// SearchAsWorkspace returns every Document of one of the workspace's signers
// that matches search, using the client's API key or bearer token.
//
// GET /signers/{signer_id}/documents/search?search={search}.
func (r *SignerDocumentResource) SearchAsWorkspace(ctx context.Context, signerID, search string) ([]models.Document, error) {
	return r.SearchAsWorkspaceWithParams(ctx, signerID, &models.ListParams{Search: search})
}

// SearchAsWorkspaceWithParams returns matching documents with the client's API
// key or bearer token. Params.Search selects the search text and EnvelopeID limits
// the envelope; other ListParams fields are ignored. The signer must belong to
// the authenticated workspace.
// GET /signers/{signer_id}/documents/search.
func (r *SignerDocumentResource) SearchAsWorkspaceWithParams(ctx context.Context, signerID string, params *models.ListParams) ([]models.Document, error) {
	return r.search(ctx, r.searchRequest(signerID), params)
}

func (r *SignerDocumentResource) searchRequest(signerID string) *internal.Request {
	return r.http.NewRequest(http.MethodGet, "/signers/"+url.PathEscape(signerID)+"/documents/search")
}

func (r *SignerDocumentResource) search(ctx context.Context, req *internal.Request, params *models.ListParams) ([]models.Document, error) {
	var out []models.Document
	if params != nil {
		req.WithQuery("search", params.Search).WithQuery("envelope_id", params.EnvelopeID)
	}
	if _, err := req.Execute(ctx, &out); err != nil {
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

// RequestCertifiedCopy sends a 30-minute download link to the original signing
// contact using only the signer code, including a closed code. Email allows ten
// resends per day spaced two minutes apart; WhatsApp allows one per request.
// POST /sign/certified-copy.
func (r *SignerDocumentResource) RequestCertifiedCopy(ctx context.Context, signerAccessCode string, body *models.CertifiedCopyRequest) (*models.CertifiedCopyResult, error) {
	var out models.CertifiedCopyResult
	_, err := r.http.NewRequest(http.MethodPost, "/sign/certified-copy").WithSignerAccessCode(signerAccessCode).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DownloadByToken fetches the certified PDF using only its download token; pades
// selects the PAdES artifact. Expired links return a 302 APIError without following
// the redirect to the expired-link HTML page. Tokens must not be logged.
// GET /signers/downloads/{token}.
func (r *SignerDocumentResource) DownloadByToken(ctx context.Context, token string, pades bool) ([]byte, error) {
	path := "/signers/downloads/" + url.PathEscape(token)
	if pades {
		path += "/pades"
	}
	return r.http.DownloadUnauthenticatedNoRedirect(ctx, path)
}

// ResendCertifiedCopy sends a new link using only the original download token,
// even when expired. It shares RequestCertifiedCopy's channel and resend limits.
// POST /signers/downloads/{token}/resend.
func (r *SignerDocumentResource) ResendCertifiedCopy(ctx context.Context, token string) (*models.CertifiedCopyResult, error) {
	var out models.CertifiedCopyResult
	_, err := r.http.NewRequest(http.MethodPost, "/signers/downloads/"+url.PathEscape(token)+"/resend").WithoutAuth().Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
