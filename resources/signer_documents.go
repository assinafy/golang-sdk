package resources

import (
	"context"
	"net/http"
	"net/url"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// SignerDocumentResource exposes the documented signer-facing document endpoints.
type SignerDocumentResource struct {
	http *internal.HTTPClient
}

// NewSignerDocumentResource constructs a SignerDocumentResource.
func NewSignerDocumentResource(httpClient *internal.HTTPClient) *SignerDocumentResource {
	return &SignerDocumentResource{http: httpClient}
}

// GetCurrent returns the document the signer must currently sign.
// GET /signers/{signer_id}/document.
func (r *SignerDocumentResource) GetCurrent(ctx context.Context, signerID, signerAccessCode string) (*models.Document, error) {
	var out models.Document
	_, err := r.http.NewRequest(http.MethodGet, "/signers/"+url.PathEscape(signerID)+"/document").
		WithQuery("signer-access-code", signerAccessCode).
		Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the documents accessible to a signer.
// GET /signers/{signer_id}/documents.
func (r *SignerDocumentResource) List(ctx context.Context, signerID, signerAccessCode string, params *models.ListParams) (*models.PaginatedResult[models.Document], error) {
	var out []models.Document
	req := r.http.NewRequest(http.MethodGet, "/signers/"+url.PathEscape(signerID)+"/documents").
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

// SignMultiple batch-signs documents for the signer.
// PUT /signers/documents/sign-multiple.
func (r *SignerDocumentResource) SignMultiple(ctx context.Context, signerAccessCode string, documentIDs []string) error {
	body := map[string][]string{"document_ids": documentIDs}
	_, err := r.http.NewRequest(http.MethodPut, "/signers/documents/sign-multiple").
		WithQuery("signer-access-code", signerAccessCode).
		WithBody(body).
		Execute(ctx, nil)
	return err
}

// DeclineMultiple batch-declines documents for the signer.
// PUT /signers/documents/decline-multiple.
func (r *SignerDocumentResource) DeclineMultiple(ctx context.Context, signerAccessCode string, documentIDs []string, reason string) error {
	body := map[string]any{
		"document_ids":   documentIDs,
		"decline_reason": reason,
	}
	_, err := r.http.NewRequest(http.MethodPut, "/signers/documents/decline-multiple").
		WithQuery("signer-access-code", signerAccessCode).
		WithBody(body).
		Execute(ctx, nil)
	return err
}

// Download fetches an artifact from a signer-accessible document.
// GET /signers/{signer_id}/documents/{document_id}/download/{artifact}.
func (r *SignerDocumentResource) Download(ctx context.Context, signerID, documentID, artifact, signerAccessCode string) ([]byte, error) {
	path := "/signers/" + url.PathEscape(signerID) +
		"/documents/" + url.PathEscape(documentID) +
		"/download/" + url.PathEscape(artifact)
	return r.http.Download(ctx, path, map[string]string{"signer-access-code": signerAccessCode})
}
