package resources

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// AssignmentResource exposes authenticated assignment management and
// signer-access-code signing endpoints. Its methods follow the package-level
// response and error contract.
type AssignmentResource struct {
	http *internal.HTTPClient
}

// NewAssignmentResource constructs an AssignmentResource.
func NewAssignmentResource(httpClient *internal.HTTPClient) *AssignmentResource {
	return &AssignmentResource{http: httpClient}
}

// List returns a page of assignments for the authenticated user's current
// account, paginated via the X-Pagination-* response headers. Page and per-page
// are the current documented parameters; populated shared search/sort fields are
// retained as compatibility queries.
//
// The OpenAPI permits an API key or bearer token. The sandbox currently requires
// user current-account context, so an API key can receive a 400 there.
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

// Create sends CreateAssignmentRequest for an authenticated document and returns
// the created Assignment. It starts the virtual or collect signing workflow and
// can dispatch signer notifications; validation or balance failures are API errors.
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

// EstimateCost sends the creation-shaped CreateAssignmentRequest using client
// authentication and returns CostEstimate without creating an assignment.
// POST /documents/{document_id}/assignments/estimate-cost.
//
// Deprecated: use EstimateCostWithRequest for the endpoint's dedicated body.
func (r *AssignmentResource) EstimateCost(ctx context.Context, documentID string, body *models.CreateAssignmentRequest) (*models.CostEstimate, error) {
	return r.estimateCost(ctx, documentID, body)
}

// EstimateCostWithRequest sends EstimateAssignmentCostRequest using client
// authentication and returns CostEstimate without creating an assignment.
// POST /documents/{document_id}/assignments/estimate-cost.
func (r *AssignmentResource) EstimateCostWithRequest(ctx context.Context, documentID string, body *models.EstimateAssignmentCostRequest) (*models.CostEstimate, error) {
	return r.estimateCost(ctx, documentID, body)
}

func (r *AssignmentResource) estimateCost(ctx context.Context, documentID string, body any) (*models.CostEstimate, error) {
	var out models.CostEstimate
	path := "/documents/" + url.PathEscape(documentID) + "/assignments/estimate-cost"
	_, err := r.http.NewRequest(http.MethodPost, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ResendNotification uses client authentication, re-sends one signer's
// invitation, and returns ResendNotificationResult. Delivery or balance failures
// are returned as API errors.
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

// EstimateResendCost uses client authentication and returns CostEstimate for one
// signer notification without sending it.
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

// ResetExpiration uses client authentication, sends expires_at explicitly (JSON
// null when expiresAt is nil), and returns the updated Assignment. The null form
// is retained for compatibility with earlier SDK releases.
// PUT /documents/{document_id}/assignments/{assignment_id}/reset-expiration.
//
// Deprecated: use ResetExpirationWithRequest for the current optional,
// non-null expires_at contract.
func (r *AssignmentResource) ResetExpiration(ctx context.Context, documentID, assignmentID string, expiresAt *string) (*models.Assignment, error) {
	return r.resetExpiration(ctx, documentID, assignmentID, map[string]*string{"expires_at": expiresAt})
}

// ResetExpirationWithRequest uses client authentication, sends the required
// JSON object (omitting expires_at when body.ExpiresAt is nil), and returns the
// updated Assignment. The operation changes the signing deadline and may
// produce lifecycle notifications.
// PUT /documents/{document_id}/assignments/{assignment_id}/reset-expiration.
func (r *AssignmentResource) ResetExpirationWithRequest(ctx context.Context, documentID, assignmentID string, body models.ResetAssignmentExpirationRequest) (*models.Assignment, error) {
	return r.resetExpiration(ctx, documentID, assignmentID, body)
}

func (r *AssignmentResource) resetExpiration(ctx context.Context, documentID, assignmentID string, body any) (*models.Assignment, error) {
	var out models.Assignment
	path := "/documents/" + url.PathEscape(documentID) +
		"/assignments/" + url.PathEscape(assignmentID) +
		"/reset-expiration"
	_, err := r.http.NewRequest(http.MethodPut, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Sign authenticates with signerAccessCode, sends the SignDocumentItem array,
// records the signer's submitted values, and returns only error (the documented
// success object has no declared fields). Invalid, expired, or incomplete data
// produces an API error.
// POST /documents/{document_id}/assignments/{assignment_id}.
func (r *AssignmentResource) Sign(ctx context.Context, documentID, assignmentID, signerAccessCode string, items []models.SignDocumentItem) error {
	path := "/documents/" + url.PathEscape(documentID) + "/assignments/" + url.PathEscape(assignmentID)
	_, err := r.http.NewRequest(http.MethodPost, path).
		WithoutAuth().
		WithQuery("signer-access-code", signerAccessCode).
		WithBody(items).
		Execute(ctx, nil)
	return err
}

// Decline authenticates with signerAccessCode, sends the required decline_reason,
// closes the signer's assignment as rejected, and returns only error, discarding
// the documented empty data array.
// PUT /documents/{document_id}/assignments/{assignment_id}/reject.
func (r *AssignmentResource) Decline(ctx context.Context, documentID, assignmentID, signerAccessCode, reason string) error {
	body := map[string]string{"decline_reason": reason}
	path := "/documents/" + url.PathEscape(documentID) +
		"/assignments/" + url.PathEscape(assignmentID) +
		"/reject"
	_, err := r.http.NewRequest(http.MethodPut, path).
		WithoutAuth().
		WithQuery("signer-access-code", signerAccessCode).
		WithBody(body).
		Execute(ctx, nil)
	return err
}

// ListWhatsAppNotifications uses client authentication and returns the
// assignment's WhatsAppNotification delivery-attempt payloads.
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

// GetSigningInfo authenticates with signerAccessCode and returns the signer-facing
// Document payload without requiring a client credential.
// GET /sign.
func (r *AssignmentResource) GetSigningInfo(ctx context.Context, signerAccessCode string) (*models.Document, error) {
	return r.getSigningInfo(ctx, signerAccessCode, nil)
}

// GetSigningInfoWithTerms authenticates with signerAccessCode, sends the
// has_accepted_terms query value, and returns the signer-facing Document. The
// terms flag can update signer state; no client credential is required.
// GET /sign?has_accepted_terms={value}.
func (r *AssignmentResource) GetSigningInfoWithTerms(ctx context.Context, signerAccessCode string, hasAcceptedTerms bool) (*models.Document, error) {
	return r.getSigningInfo(ctx, signerAccessCode, &hasAcceptedTerms)
}

func (r *AssignmentResource) getSigningInfo(ctx context.Context, signerAccessCode string, hasAcceptedTerms *bool) (*models.Document, error) {
	var out models.Document
	req := r.http.NewRequest(http.MethodGet, "/sign").WithoutAuth().WithQuery("signer-access-code", signerAccessCode)
	if hasAcceptedTerms != nil {
		req.WithQuery("has_accepted_terms", strconv.FormatBool(*hasAcceptedTerms))
	}
	_, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
