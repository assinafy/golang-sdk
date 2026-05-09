package resources

import (
	"context"
	"net/http"
	"net/url"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// PublicDocumentResource exposes the documented unauthenticated public document endpoints.
type PublicDocumentResource struct {
	http *internal.HTTPClient
}

// NewPublicDocumentResource constructs a PublicDocumentResource.
func NewPublicDocumentResource(httpClient *internal.HTTPClient) *PublicDocumentResource {
	return &PublicDocumentResource{http: httpClient}
}

// Get returns the limited public information for a document.
// GET /public/documents/{document_id}.
func (r *PublicDocumentResource) Get(ctx context.Context, documentID string) (*models.PublicDocumentInfo, error) {
	var out models.PublicDocumentInfo
	_, err := r.http.NewRequest(http.MethodGet, "/public/documents/"+url.PathEscape(documentID)).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SendToken delivers a signing access token to the recipient.
// PUT /public/documents/{document_id}/send-token.
func (r *PublicDocumentResource) SendToken(ctx context.Context, documentID string, body *models.SendDocumentTokenRequest) (*models.SendDocumentTokenResult, error) {
	var out models.SendDocumentTokenResult
	path := "/public/documents/" + url.PathEscape(documentID) + "/send-token"
	_, err := r.http.NewRequest(http.MethodPut, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
