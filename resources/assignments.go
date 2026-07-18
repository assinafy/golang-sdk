package resources

import (
	"context"
	"net/http"
	"net/url"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// AssignmentResource exposes the documented `Assignment` endpoints.
type AssignmentResource struct {
	http *internal.HTTPClient
}

// NewAssignmentResource constructs an AssignmentResource.
func NewAssignmentResource(httpClient *internal.HTTPClient) *AssignmentResource {
	return &AssignmentResource{http: httpClient}
}

// List returns a page of assignments for the authenticated user's current
// account, paginated via the X-Pagination-* response headers. Only the page and
// per-page parameters of params are used.
//
// This endpoint resolves the account from the caller's session, so it requires a
// bearer-token login (which selects a current account). Calling it with only an
// API key returns a 400 "account context required" error in the sandbox.
// GET /assignments.
func (r *AssignmentResource) List(ctx context.Context, params *models.ListParams) (*models.PaginatedResult[models.Assignment], error) {
	var out []models.Assignment
	req := r.http.NewRequest(http.MethodGet, "/assignments")
	applyListParams(req, params)

	resp, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return paginated(out, resp), nil
}

// Create creates an assignment (virtual or collect).
// POST /documents/{document_id}/assignments.
func (r *AssignmentResource) Create(ctx context.Context, documentID string, body *models.CreateAssignmentRequest) (*models.Assignment, error) {
	var out models.Assignment
	path := "/documents/" + url.PathEscape(documentID) + "/assignments"
	_, err := r.http.NewRequest(http.MethodPost, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// EstimateCost returns the cost estimate for an assignment.
// POST /documents/{document_id}/assignments/estimate-cost.
func (r *AssignmentResource) EstimateCost(ctx context.Context, documentID string, body *models.CreateAssignmentRequest) (*models.CostEstimate, error) {
	var out models.CostEstimate
	path := "/documents/" + url.PathEscape(documentID) + "/assignments/estimate-cost"
	_, err := r.http.NewRequest(http.MethodPost, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ResendNotification re-sends the signing invitation for one signer.
// PUT /documents/{document_id}/assignments/{assignment_id}/signers/{signer_id}/resend.
func (r *AssignmentResource) ResendNotification(ctx context.Context, documentID, assignmentID, signerID string) (*models.ResendNotificationResult, error) {
	var out models.ResendNotificationResult
	path := "/documents/" + url.PathEscape(documentID) +
		"/assignments/" + url.PathEscape(assignmentID) +
		"/signers/" + url.PathEscape(signerID) +
		"/resend"
	_, err := r.http.NewRequest(http.MethodPut, path).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// EstimateResendCost returns the cost of resending one signer notification.
// POST /documents/{document_id}/assignments/{assignment_id}/signers/{signer_id}/estimate-resend-cost.
func (r *AssignmentResource) EstimateResendCost(ctx context.Context, documentID, assignmentID, signerID string) (*models.CostEstimate, error) {
	var out models.CostEstimate
	path := "/documents/" + url.PathEscape(documentID) +
		"/assignments/" + url.PathEscape(assignmentID) +
		"/signers/" + url.PathEscape(signerID) +
		"/estimate-resend-cost"
	_, err := r.http.NewRequest(http.MethodPost, path).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ResetExpiration updates an assignment's expiration date.
// PUT /documents/{document_id}/assignments/{assignment_id}/reset-expiration.
func (r *AssignmentResource) ResetExpiration(ctx context.Context, documentID, assignmentID string, expiresAt *string) (*models.Assignment, error) {
	var out models.Assignment
	body := map[string]*string{"expires_at": expiresAt}
	path := "/documents/" + url.PathEscape(documentID) +
		"/assignments/" + url.PathEscape(assignmentID) +
		"/reset-expiration"
	_, err := r.http.NewRequest(http.MethodPut, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Sign submits filled fields for the signer's assignment.
// POST /documents/{document_id}/assignments/{assignment_id}.
func (r *AssignmentResource) Sign(ctx context.Context, documentID, assignmentID, signerAccessCode string, items []models.SignDocumentItem) error {
	path := "/documents/" + url.PathEscape(documentID) + "/assignments/" + url.PathEscape(assignmentID)
	_, err := r.http.NewRequest(http.MethodPost, path).
		WithQuery("signer-access-code", signerAccessCode).
		WithBody(items).
		Execute(ctx, nil)
	return err
}

// Decline rejects an assignment.
// PUT /documents/{document_id}/assignments/{assignment_id}/reject.
func (r *AssignmentResource) Decline(ctx context.Context, documentID, assignmentID, signerAccessCode, reason string) error {
	body := map[string]string{"decline_reason": reason}
	path := "/documents/" + url.PathEscape(documentID) +
		"/assignments/" + url.PathEscape(assignmentID) +
		"/reject"
	_, err := r.http.NewRequest(http.MethodPut, path).
		WithQuery("signer-access-code", signerAccessCode).
		WithBody(body).
		Execute(ctx, nil)
	return err
}

// ListWhatsAppNotifications lists WhatsApp delivery attempts for an assignment.
// GET /documents/{document_id}/assignments/{assignment_id}/whatsapp-notifications.
func (r *AssignmentResource) ListWhatsAppNotifications(ctx context.Context, documentID, assignmentID string) ([]models.WhatsAppNotification, error) {
	var out []models.WhatsAppNotification
	path := "/documents/" + url.PathEscape(documentID) +
		"/assignments/" + url.PathEscape(assignmentID) +
		"/whatsapp-notifications"
	_, err := r.http.NewRequest(http.MethodGet, path).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetSigningInfo returns the signing surface for the signer's access code.
// GET /sign.
func (r *AssignmentResource) GetSigningInfo(ctx context.Context, signerAccessCode string) (*models.Document, error) {
	var out models.Document
	_, err := r.http.NewRequest(http.MethodGet, "/sign").
		WithQuery("signer-access-code", signerAccessCode).
		Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
