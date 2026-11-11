package resources

import (
	"context"
	"net/http"
	"net/url"

	"github.com/assinafy/golang-sdk/v2/internal"
	"github.com/assinafy/golang-sdk/v2/models"
)

// EnvelopeResource manages workspace envelopes using API-key or bearer
// authentication. Reads require documents:read; writes require documents:write.
// Creation and additions also require the workspace's Envelopes plan feature.
type EnvelopeResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewEnvelopeResource constructs an EnvelopeResource with a default workspace.
func NewEnvelopeResource(httpClient *internal.HTTPClient, accountID string) *EnvelopeResource {
	return &EnvelopeResource{http: httpClient, accountID: accountID}
}

func (r *EnvelopeResource) path(accountID string) string {
	return "/accounts/" + url.PathEscape(resolveAccountID(accountID, r.accountID)) + "/envelopes"
}

// List returns authenticated envelopes newest first with X-Pagination metadata.
// Page, PerPage, Search and Status (open or closed) are supported. An empty
// accountID uses the configured workspace.
// GET /accounts/{account_id}/envelopes.
func (r *EnvelopeResource) List(ctx context.Context, accountID string, params *models.ListParams) (*models.PaginatedResult[models.DocumentEnvelope], error) {
	req := r.http.NewRequest(http.MethodGet, r.path(accountID))
	p := models.ListParams{}
	if params != nil {
		p = *params
	}
	p.Sort = ""
	applyListParams(req, &p)
	req.WithQuery("status", p.Status)
	var out []models.DocumentEnvelope
	resp, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return paginated(out, resp), nil
}

// Create sends CreateEnvelopeRequest with client authentication and returns the
// envelope. IdempotencyKey makes retries after uncertain responses safe.
// Success does not guarantee invitation delivery.
// An empty accountID uses the configured workspace.
// POST /accounts/{account_id}/envelopes.
func (r *EnvelopeResource) Create(ctx context.Context, accountID string, body *models.CreateEnvelopeRequest) (*models.DocumentEnvelope, error) {
	req := r.http.NewRequest(http.MethodPost, r.path(accountID)).WithBody(body)
	if body != nil && body.IdempotencyKey != "" {
		req.WithHeader("Idempotency-Key", body.IdempotencyKey)
	}
	var out models.DocumentEnvelope
	if _, err := req.Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// EstimateCost uses client authentication and the creation body to price an
// envelope without creating or charging it. MaxTotalCredits is ignored by the
// API. An empty accountID uses the configured workspace.
// POST /accounts/{account_id}/envelopes/estimate-cost.
func (r *EnvelopeResource) EstimateCost(ctx context.Context, accountID string, body *models.CreateEnvelopeRequest) (*models.CostEstimate, error) {
	var out models.CostEstimate
	if _, err := r.http.NewRequest(http.MethodPost, r.path(accountID)+"/estimate-cost").WithBody(body).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get returns an authenticated envelope's documents, statuses and signers.
// An empty accountID uses the configured workspace.
// GET /accounts/{account_id}/envelopes/{envelope_id}.
func (r *EnvelopeResource) Get(ctx context.Context, accountID, envelopeID string) (*models.DocumentEnvelope, error) {
	var out models.DocumentEnvelope
	if _, err := r.http.NewRequest(http.MethodGet, r.path(accountID)+"/"+url.PathEscape(envelopeID)).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update uses client authentication to change the supplied name, message or
// document order and returns the envelope. It sends no notifications and also
// works on closed envelopes. An empty accountID uses the configured workspace.
// PATCH /accounts/{account_id}/envelopes/{envelope_id}.
func (r *EnvelopeResource) Update(ctx context.Context, accountID, envelopeID string, body *models.UpdateEnvelopeRequest) (*models.DocumentEnvelope, error) {
	var out models.DocumentEnvelope
	if _, err := r.http.NewRequest(http.MethodPatch, r.path(accountID)+"/"+url.PathEscape(envelopeID)).WithBody(body).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete uses client authentication to remove a closed envelope, retaining its
// documents and artifacts. Open envelopes return 409 EnvelopeOpen; cancel them
// first. An empty accountID uses the configured workspace.
// DELETE /accounts/{account_id}/envelopes/{envelope_id}.
func (r *EnvelopeResource) Delete(ctx context.Context, accountID, envelopeID string) error {
	_, err := r.http.NewRequest(http.MethodDelete, r.path(accountID)+"/"+url.PathEscape(envelopeID)).Execute(ctx, nil)
	return err
}

// Activities returns the authenticated envelope's document activity trail and
// X-Pagination metadata, including removed documents. Nil params uses 20 items
// newest first; Page, PerPage and Sort (id or -id) are supported. An empty
// accountID uses the configured workspace.
// GET /accounts/{account_id}/envelopes/{envelope_id}/activities.
func (r *EnvelopeResource) Activities(ctx context.Context, accountID, envelopeID string, params *models.ListParams) (*models.PaginatedResult[models.EnvelopeActivity], error) {
	req := r.http.NewRequest(http.MethodGet, r.path(accountID)+"/"+url.PathEscape(envelopeID)+"/activities")
	if params != nil {
		p := *params
		p.Search = ""
		applyListParams(req, &p)
	}
	var out []models.EnvelopeActivity
	resp, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return paginated(out, resp), nil
}

// Download uses client authentication to return a ZIP of each document's chosen
// artifact in display order: original, certificated, certificate-page, pades or
// bundle. Only available artifacts are included; none yields 404. An empty
// accountID uses the configured workspace.
// GET /accounts/{account_id}/envelopes/{envelope_id}/download/{artifact}.
func (r *EnvelopeResource) Download(ctx context.Context, accountID, envelopeID, artifact string) ([]byte, error) {
	return r.http.Download(ctx, r.path(accountID)+"/"+url.PathEscape(envelopeID)+"/download/"+url.PathEscape(artifact), nil)
}

// Cancel uses client authentication to cancel eligible documents, notifying
// owners, signers and copy receivers. Closed or fully signed documents are left
// unchanged. Repeating the call is safe. An empty accountID uses the default.
// PUT /accounts/{account_id}/envelopes/{envelope_id}/cancel.
func (r *EnvelopeResource) Cancel(ctx context.Context, accountID, envelopeID string) ([]models.EnvelopeCancellation, error) {
	var out []models.EnvelopeCancellation
	if _, err := r.http.NewRequest(http.MethodPut, r.path(accountID)+"/"+url.PathEscape(envelopeID)+"/cancel").Execute(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ResendNotification uses client authentication to remind one signer about all
// their pending envelope documents. Paid notification channels are charged per
// message sent. An empty accountID uses the configured workspace.
// PUT /accounts/{account_id}/envelopes/{envelope_id}/signers/{signer_id}/resend.
func (r *EnvelopeResource) ResendNotification(ctx context.Context, accountID, envelopeID, signerID string) (*models.EnvelopeResendResult, error) {
	var out models.EnvelopeResendResult
	path := r.path(accountID) + "/" + url.PathEscape(envelopeID) + "/signers/" + url.PathEscape(signerID) + "/resend"
	if _, err := r.http.NewRequest(http.MethodPut, path).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddDocument uses client authentication to append a processed document for
// existing signers to an open envelope. Those signers receive an invitation;
// only additional certificate signatures are charged. An empty accountID uses
// the configured workspace.
// POST /accounts/{account_id}/envelopes/{envelope_id}/documents.
func (r *EnvelopeResource) AddDocument(ctx context.Context, accountID, envelopeID string, body *models.AddEnvelopeDocumentRequest) (*models.DocumentEnvelope, error) {
	var out models.DocumentEnvelope
	if _, err := r.http.NewRequest(http.MethodPost, r.path(accountID)+"/"+url.PathEscape(envelopeID)+"/documents").WithBody(body).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RemoveDocument uses client authentication to cancel and detach one open
// document, notifying its owner, signers and copy receivers. Other documents
// continue signing. A closed document returns 409 DocumentClosed. An empty
// accountID uses the configured workspace.
// DELETE /accounts/{account_id}/envelopes/{envelope_id}/documents/{document_id}.
func (r *EnvelopeResource) RemoveDocument(ctx context.Context, accountID, envelopeID, documentID string) (*models.DocumentEnvelope, error) {
	var out models.DocumentEnvelope
	path := r.path(accountID) + "/" + url.PathEscape(envelopeID) + "/documents/" + url.PathEscape(documentID)
	if _, err := r.http.NewRequest(http.MethodDelete, path).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
