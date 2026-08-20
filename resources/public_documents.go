package resources

import (
	"context"
	stderrors "errors"
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
// Deprecated: use GetDocument, which decodes the full response documented by
// the current API contract.
func (r *PublicDocumentResource) Get(ctx context.Context, documentID string) (*models.PublicDocumentInfo, error) {
	var out models.PublicDocumentInfo
	if err := r.get(ctx, documentID, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDocument returns the documented public Document payload without requiring a
// client credential. An unknown or non-public document produces an API error.
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

// SendToken sends the legacy SendDocumentTokenRequest without a client credential,
// triggers token delivery, and decodes the legacy SendDocumentTokenResult.
// PUT /public/documents/{document_id}/send-token.
//
// Deprecated: use SendTokenWithRequest or SendTokenByEmail for the request and
// envelope-only response documented by the current API contract.
func (r *PublicDocumentResource) SendToken(ctx context.Context, documentID string, body *models.SendDocumentTokenRequest) (*models.SendDocumentTokenResult, error) {
	var out models.SendDocumentTokenResult
	err := r.sendToken(ctx, documentID, body, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SendTokenByEmail sends the required email body without a client credential,
// triggers access-token delivery, and returns an envelope-only success. Invalid
// destinations and delivery failures are API errors.
// PUT /public/documents/{document_id}/send-token.
func (r *PublicDocumentResource) SendTokenByEmail(ctx context.Context, documentID, email string) error {
	return r.SendTokenWithRequest(ctx, documentID, &models.SendDocumentTokenRequest{Email: email})
}

// SendTokenWithRequest triggers token delivery with the current optional
// request body. Pass nil to omit the body entirely. For older sandbox
// compatibility, an email-bearing request retries with recipient/channel only
// after a 400 response whose message mentions channel.
// PUT /public/documents/{document_id}/send-token.
func (r *PublicDocumentResource) SendTokenWithRequest(ctx context.Context, documentID string, body *models.SendDocumentTokenRequest) error {
	if body == nil {
		return r.sendToken(ctx, documentID, nil, nil)
	}
	err := r.sendToken(ctx, documentID, body, nil)
	var apiErr *sdkerrors.APIError
	if !stderrors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest ||
		!strings.Contains(strings.ToLower(apiErr.Message), "channel") || body.Email == "" {
		return err
	}
	// Older sandbox deployments require the former recipient/channel shape.
	return r.sendToken(ctx, documentID, &models.SendDocumentTokenRequest{
		Recipient: body.Email,
		Channel:   "email",
	}, nil)
}

func (r *PublicDocumentResource) sendToken(ctx context.Context, documentID string, body, out any) error {
	path := "/public/documents/" + url.PathEscape(documentID) + "/send-token"
	req := r.http.NewRequest(http.MethodPut, path).WithoutAuth()
	if body != nil {
		req.WithBody(body)
	}
	_, err := req.Execute(ctx, out)
	return err
}
