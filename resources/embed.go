package resources

import (
	"context"
	"net/http"
	"net/url"

	"github.com/assinafy/golang-sdk/v2/internal"
	"github.com/assinafy/golang-sdk/v2/models"
)

// EmbedResource manages allowed origins and single-use embedded signing sessions.
// Embedded signing requires the workspace's plan to include the feature.
type EmbedResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewEmbedResource constructs an EmbedResource with a default workspace ID.
func NewEmbedResource(httpClient *internal.HTTPClient, accountID string) *EmbedResource {
	return &EmbedResource{http: httpClient, accountID: accountID}
}

func (r *EmbedResource) originsPath(accountID string) string {
	return "/accounts/" + url.PathEscape(resolveAccountID(accountID, r.accountID)) + "/embed-origins"
}

// ListOrigins returns the allowed origins, oldest first. API-key or user-token
// authentication is required; OAuth applications cannot manage origins.
// An empty accountID uses the configured workspace. A plan without embedded
// signing returns 403 with APIError.Code PlanRestricted.
// GET /accounts/{account_id}/embed-origins.
func (r *EmbedResource) ListOrigins(ctx context.Context, accountID string) ([]models.EmbedOrigin, error) {
	var out []models.EmbedOrigin
	if _, err := r.http.NewRequest(http.MethodGet, r.originsPath(accountID)).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateOrigin registers an origin and returns its normalized representation.
// It requires an API key or user token, not OAuth. An empty accountID uses the
// configured workspace. Invalid, duplicate or an eleventh origin returns 400;
// a plan without embedded signing returns 403 with code PlanRestricted.
// POST /accounts/{account_id}/embed-origins.
func (r *EmbedResource) CreateOrigin(ctx context.Context, accountID string, body *models.EmbedOriginRequest) (*models.EmbedOrigin, error) {
	var out models.EmbedOrigin
	if _, err := r.http.NewRequest(http.MethodPost, r.originsPath(accountID)).WithBody(body).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateOrigin replaces an allowed origin while preserving its ID. It requires
// an API key or user token, not OAuth, and returns the normalized EmbedOrigin.
// An empty accountID uses the configured workspace. Invalid or duplicate origins
// return 400, a missing ID returns 404, and a plan restriction returns 403.
// PUT /accounts/{account_id}/embed-origins/{origin_id}.
func (r *EmbedResource) UpdateOrigin(ctx context.Context, accountID, originID string, body *models.EmbedOriginRequest) (*models.EmbedOrigin, error) {
	var out models.EmbedOrigin
	path := r.originsPath(accountID) + "/" + url.PathEscape(originID)
	if _, err := r.http.NewRequest(http.MethodPut, path).WithBody(body).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteOrigin removes an allowed origin with API-key or user-token authentication;
// OAuth applications cannot manage origins. An empty accountID uses the configured
// workspace. Removing the last origin prevents new sessions with 409 and code
// embed_origins_missing. A successful removal returns no payload.
// DELETE /accounts/{account_id}/embed-origins/{origin_id}.
func (r *EmbedResource) DeleteOrigin(ctx context.Context, accountID, originID string) error {
	path := r.originsPath(accountID) + "/" + url.PathEscape(originID)
	_, err := r.http.NewRequest(http.MethodDelete, path).Execute(ctx, nil)
	return err
}

// CreateSession uses an API key or user token, not OAuth, to mint a single-use
// iframe URL for a signer configured with embedded:true. Open it on an allowed origin within
// five minutes and request a fresh URL each time the signer opens the page.
// The signer access code is not returned. APIError.Code identifies
// embed_not_enabled_for_signer (400), already_signed, document_not_ready,
// document_unavailable or embed_origins_missing (409), and PlanRestricted (403).
// Only document_not_ready calls for waiting until document processing completes;
// more than thirty sessions for one signer per hour returns 429.
// POST /documents/{document_id}/assignments/{assignment_id}/signers/{signer_id}/embed-sessions.
func (r *EmbedResource) CreateSession(ctx context.Context, documentID, assignmentID, signerID string) (*models.EmbedSession, error) {
	var out models.EmbedSession
	path := "/documents/" + url.PathEscape(documentID) + "/assignments/" + url.PathEscape(assignmentID) + "/signers/" + url.PathEscape(signerID) + "/embed-sessions"
	if _, err := r.http.NewRequest(http.MethodPost, path).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ExchangeSession consumes the session token without sending workspace credentials
// and returns the signer context. It is intended for Assinafy's embed page;
// integrations should load CreateSession's URL instead. A nil body sends {}.
// The token is consumed even on a 409 refusal, and unknown, expired or used tokens
// return 410 with code embed_session_invalid. Never retry an uncertain exchange.
// POST /embed/sessions/{token}/exchange.
func (r *EmbedResource) ExchangeSession(ctx context.Context, token string, body *models.ExchangeEmbedSessionRequest) (*models.EmbedSessionExchange, error) {
	var out models.EmbedSessionExchange
	req := r.http.NewRequest(http.MethodPost, "/embed/sessions/"+url.PathEscape(token)+"/exchange").WithoutAuth().WithoutReplay()
	if body == nil {
		body = &models.ExchangeEmbedSessionRequest{}
	}
	if _, err := req.WithBody(body).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
