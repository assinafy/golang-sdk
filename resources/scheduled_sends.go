package resources

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/assinafy/golang-sdk/v2/internal"
	"github.com/assinafy/golang-sdk/v2/models"
)

// ScheduledSendResource schedules assignment and envelope invitations.
// All methods require workspace credentials or OAuth documents:write, including List.
type ScheduledSendResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewScheduledSendResource constructs a resource with a default workspace.
func NewScheduledSendResource(httpClient *internal.HTTPClient, accountID string) *ScheduledSendResource {
	return &ScheduledSendResource{http: httpClient, accountID: accountID}
}

func (r *ScheduledSendResource) path(accountID string) string {
	return "/accounts/" + url.PathEscape(resolveAccountID(accountID, r.accountID)) + "/scheduled-sends"
}

// CreateAssignment schedules an assignment and returns its pending ScheduledSend.
// It uses workspace credentials or OAuth documents:write. Nothing is sent or
// charged until execution. Do not repeat an uncertain scheduling request; list
// the document's sends first. The API validates methods, plan and signer membership.
// POST /documents/{document_id}/scheduled-sends.
func (r *ScheduledSendResource) CreateAssignment(ctx context.Context, documentID string, body *models.ScheduleAssignmentRequest) (*models.ScheduledSend, error) {
	var out models.ScheduledSend
	path := "/documents/" + url.PathEscape(documentID) + "/scheduled-sends"
	if _, err := r.http.NewRequest(http.MethodPost, path).WithBody(body).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateEnvelope schedules an envelope and returns its pending ScheduledSend.
// It requires workspace credentials or OAuth documents:write and the Envelopes
// feature. Empty accountID uses the default. IdempotencyKey is not sent; never
// repeat an uncertain call without reading List. Nothing is sent or charged yet.
// POST /accounts/{account_id}/scheduled-sends.
func (r *ScheduledSendResource) CreateEnvelope(ctx context.Context, accountID string, body *models.ScheduleEnvelopeRequest) (*models.ScheduledSend, error) {
	var out models.ScheduledSend
	if _, err := r.http.NewRequest(http.MethodPost, r.path(accountID)).WithBody(body).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns scheduled sends of all statuses, newest first, with X-Pagination
// metadata. It requires workspace credentials or OAuth documents:write. Empty
// accountID uses the default; nil params uses the API's default page of 20.
// GET /accounts/{account_id}/scheduled-sends.
func (r *ScheduledSendResource) List(ctx context.Context, accountID string, params *models.ScheduledSendListParams) (*models.PaginatedResult[models.ScheduledSend], error) {
	var out []models.ScheduledSend
	req := r.http.NewRequest(http.MethodGet, r.path(accountID))
	if params != nil {
		if params.Page > 0 {
			req.WithQuery("page", strconv.Itoa(params.Page))
		}
		if params.PerPage > 0 {
			req.WithQuery("per-page", strconv.Itoa(params.PerPage))
		}
		req.WithQuery("document_id", params.DocumentID)
	}
	resp, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return paginated(out, resp), nil
}

// SendNow executes a pending send as the member who scheduled it and returns
// the sent ScheduledSend. It requires workspace credentials or OAuth documents:write;
// empty accountID uses the default. Execution rechecks plan, balances and the stored
// credit ceiling. A refusal leaves it pending; a non-pending send returns 409.
// PUT /accounts/{account_id}/scheduled-sends/{scheduled_send_id}/send.
func (r *ScheduledSendResource) SendNow(ctx context.Context, accountID, scheduledSendID string) (*models.ScheduledSend, error) {
	var out models.ScheduledSend
	path := r.path(accountID) + "/" + url.PathEscape(scheduledSendID) + "/send"
	if _, err := r.http.NewRequest(http.MethodPut, path).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Cancel returns the cancelled ScheduledSend without sending or charging.
// It requires workspace credentials or OAuth documents:write; empty accountID
// uses the default. Only pending sends can be cancelled; other states return 409.
// PUT /accounts/{account_id}/scheduled-sends/{scheduled_send_id}/cancel.
func (r *ScheduledSendResource) Cancel(ctx context.Context, accountID, scheduledSendID string) (*models.ScheduledSend, error) {
	var out models.ScheduledSend
	path := r.path(accountID) + "/" + url.PathEscape(scheduledSendID) + "/cancel"
	if _, err := r.http.NewRequest(http.MethodPut, path).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
