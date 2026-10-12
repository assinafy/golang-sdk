package resources

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// PublicDocumentResource exposes unauthenticated public-document endpoints. Its
// methods follow the package-level response and error contract.
type PublicDocumentResource struct {
	http *internal.HTTPClient
}

// NewPublicDocumentResource constructs a PublicDocumentResource.
func NewPublicDocumentResource(httpClient *internal.HTTPClient) *PublicDocumentResource {
	return &PublicDocumentResource{http: httpClient}
}

// Get returns the legacy limited PublicDocumentInfo payload without requiring a
// client credential. An unknown or non-public document produces an API error.
// GET /public/documents/{document_id}.
//
// Deprecated: use GetDocument when the deployment returns the full Document
// shape. Get retains page_count and created_by from the reduced public shape.
func (r *PublicDocumentResource) Get(ctx context.Context, documentID string) (*models.PublicDocumentInfo, error) {
	var out models.PublicDocumentInfo
	if err := r.get(ctx, documentID, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDocument returns the documented public Document payload without requiring a
// client credential. An unknown or non-public document produces an API error.
// Deployments returning the reduced public shape populate only matching fields;
// use Get to retain that shape's page_count and created_by.
// GET /public/documents/{document_id}.
func (r *PublicDocumentResource) GetDocument(ctx context.Context, documentID string) (*models.Document, error) {
	var out models.Document
	if err := r.get(ctx, documentID, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *PublicDocumentResource) get(ctx context.Context, documentID string, out any) error {
	_, err := r.http.NewRequest(http.MethodGet, "/public/documents/"+url.PathEscape(documentID)).WithoutAuth().Execute(ctx, out)
	return err
}

// SendToken sends a one-time access token to a signer of a public document so
// they can view it, without a client credential, and returns the delivery
// result. The document must be pending signature and the recipient must be a
// signer who has not signed yet.
//
// Request: {"channel": "email", "recipient": string}, both required; an empty
// Channel is sent as "email", and Email stands in for an empty Recipient. A nil
// body, an empty recipient or a channel other than "email" is rejected with
// errors.ErrInvalidInput before any request is sent.
//
// Response data: {"document": {"resource", "id", "name", "page_count",
// "created_by"}, "channel": string, "recipient": string}.
//
// PUT /public/documents/{document_id}/send-token.
func (r *PublicDocumentResource) SendToken(ctx context.Context, documentID string, body *models.SendDocumentTokenRequest) (*models.SendDocumentTokenResult, error) {
	if body == nil {
		return nil, fmt.Errorf("%w: send-token body is nil", sdkerrors.ErrInvalidInput)
	}
	req := models.SendDocumentTokenRequest{Recipient: body.Recipient, Channel: body.Channel}
	if req.Recipient == "" {
		req.Recipient = body.Email
	}
	if req.Channel == "" {
		req.Channel = models.SendTokenChannelEmail
	}
	if strings.TrimSpace(req.Recipient) == "" {
		return nil, fmt.Errorf("%w: send-token recipient is empty", sdkerrors.ErrInvalidInput)
	}
	if req.Channel != models.SendTokenChannelEmail {
		return nil, fmt.Errorf("%w: send-token channel %q is not supported; use %q", sdkerrors.ErrInvalidInput, req.Channel, models.SendTokenChannelEmail)
	}

	var out models.SendDocumentTokenResult
	path := "/public/documents/" + url.PathEscape(documentID) + "/send-token"
	if _, err := r.http.NewRequest(http.MethodPut, path).WithoutAuth().WithBody(req).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SendTokenByEmail sends a one-time access token to the signer whose email is
// email, as {"channel": "email", "recipient": email}, without a client
// credential. Use SendToken to read the delivery result.
//
// PUT /public/documents/{document_id}/send-token.
func (r *PublicDocumentResource) SendTokenByEmail(ctx context.Context, documentID, email string) error {
	_, err := r.SendToken(ctx, documentID, &models.SendDocumentTokenRequest{Recipient: email})
	return err
}

// SendTokenWithRequest behaves like SendToken but returns only the error.
//
// PUT /public/documents/{document_id}/send-token.
func (r *PublicDocumentResource) SendTokenWithRequest(ctx context.Context, documentID string, body *models.SendDocumentTokenRequest) error {
	_, err := r.SendToken(ctx, documentID, body)
	return err
}
