package resources

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/internal"
	"github.com/assinafy/golang-sdk/v2/models"
)

// AssignmentResource exposes authenticated assignment management and
// signer-access-code signing endpoints. Its methods follow the package-level
// response and error contract.
type AssignmentResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewAssignmentResource constructs an AssignmentResource. The first optional
// accountID supplies the default workspace for List.
func NewAssignmentResource(httpClient *internal.HTTPClient, accountID ...string) *AssignmentResource {
	r := &AssignmentResource{http: httpClient}
	if len(accountID) > 0 {
		r.accountID = accountID[0]
	}
	return r
}

// List returns a page of assignments, paginated via X-Pagination-* headers.
// params.AccountID overrides the configured account ID, sent as accountId.
// API keys and user tokens require it; OAuth can use its token's workspace.
// It sends page, per-page, and any populated shared search and sort parameters.
// GET /assignments.
func (r *AssignmentResource) List(ctx context.Context, params *models.ListParams) (*models.PaginatedResult[models.Assignment], error) {
	var out []models.Assignment
	req := r.http.NewRequest(http.MethodGet, "/assignments")
	accountID := r.accountID
	if params != nil {
		accountID = resolveAccountID(params.AccountID, accountID)
	}
	req.WithQuery("accountId", accountID)
	applyListParams(req, params)

	resp, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return paginated(out, resp), nil
}

// Create sends CreateAssignmentRequest for an authenticated document and returns
// the created Assignment. It starts the virtual or collect signing workflow and
// can dispatch signer notifications. A DigitalCertificate signer with any method
// other than virtual returns errors.ErrDigitalCertificateRequiresVirtual before
// a request; other validation or balance failures are API errors.
// POST /documents/{document_id}/assignments.
func (r *AssignmentResource) Create(ctx context.Context, documentID string, body *models.CreateAssignmentRequest) (*models.Assignment, error) {
	if err := validateAssignmentCertificateMethod(body); err != nil {
		return nil, err
	}
	var out models.Assignment
	path := "/documents/" + url.PathEscape(documentID) + "/assignments"
	_, err := r.http.NewRequest(http.MethodPost, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// EstimateCost sends the creation-shaped CreateAssignmentRequest using client
// authentication and returns CostEstimate without creating an assignment. A
// non-virtual DigitalCertificate signer returns
// errors.ErrDigitalCertificateRequiresVirtual before a request.
// POST /documents/{document_id}/assignments/estimate-cost.
//
// Deprecated: use EstimateCostWithRequest for the endpoint's dedicated body.
func (r *AssignmentResource) EstimateCost(ctx context.Context, documentID string, body *models.CreateAssignmentRequest) (*models.CostEstimate, error) {
	return r.estimateCost(ctx, documentID, body)
}

// EstimateCostWithRequest sends EstimateAssignmentCostRequest using client
// authentication and returns CostEstimate without creating an assignment. An
// explicitly non-virtual DigitalCertificate method returns
// errors.ErrDigitalCertificateRequiresVirtual before a request. Method may be
// omitted; set it to the intended assignment method for a matching estimate.
// POST /documents/{document_id}/assignments/estimate-cost.
func (r *AssignmentResource) EstimateCostWithRequest(ctx context.Context, documentID string, body *models.EstimateAssignmentCostRequest) (*models.CostEstimate, error) {
	return r.estimateCost(ctx, documentID, body)
}

func (r *AssignmentResource) estimateCost(ctx context.Context, documentID string, body any) (*models.CostEstimate, error) {
	if err := requireEstimateSigners(body); err != nil {
		return nil, err
	}
	if err := validateEstimateCertificateMethod(body); err != nil {
		return nil, err
	}
	var out models.CostEstimate
	path := "/documents/" + url.PathEscape(documentID) + "/assignments/estimate-cost"
	_, err := r.http.NewRequest(http.MethodPost, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func validateAssignmentCertificateMethod(body *models.CreateAssignmentRequest) error {
	if body == nil {
		return nil
	}
	for _, signer := range body.Signers {
		if err := requireVirtualDigitalCertificate(body.Method, signer.VerificationMethod); err != nil {
			return err
		}
	}
	return nil
}

func validateEstimateCertificateMethod(body any) error {
	switch request := body.(type) {
	case *models.EstimateAssignmentCostRequest:
		if request.Method == "" {
			return nil
		}
		for _, signer := range request.Signers {
			if err := requireVirtualDigitalCertificate(request.Method, signer.VerificationMethod); err != nil {
				return err
			}
		}
	case *models.CreateAssignmentRequest:
		return validateAssignmentCertificateMethod(request)
	}
	return nil
}

// requireEstimateSigners rejects a signer-less cost estimate before it reaches the network.
//
// The published contract marks signers as required only for the virtual method, but the API
// prices per signer in both modes and answers a signer-less body with
// 400 "Pelo menos um signatários precisa ser informado." The request structs tag Signers
// omitempty, so an empty slice would drop the key entirely and the call could never succeed.
func requireEstimateSigners(body any) error {
	var count int
	switch request := body.(type) {
	case *models.EstimateAssignmentCostRequest:
		if request == nil {
			return fmt.Errorf("%w: cost estimate body is nil", sdkerrors.ErrInvalidInput)
		}
		count = len(request.Signers)
	case *models.CreateAssignmentRequest:
		if request == nil {
			return fmt.Errorf("%w: cost estimate body is nil", sdkerrors.ErrInvalidInput)
		}
		count = len(request.Signers) + len(request.SignerIDs)
	default:
		return nil
	}
	if count == 0 {
		return fmt.Errorf("%w: at least one signer is required for a cost estimate", sdkerrors.ErrInvalidInput)
	}
	return nil
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

// ResetExpiration sets the assignment's expiration to expiresAt, an ISO-8601
// date-time at least one hour in the future, or removes the expiration when
// expiresAt is nil. It uses client authentication and returns the updated
// Assignment. Closed documents produce an API error.
//
// Request: {"expires_at": string | null}; the key is always sent.
//
// PUT /documents/{document_id}/assignments/{assignment_id}/reset-expiration.
func (r *AssignmentResource) ResetExpiration(ctx context.Context, documentID, assignmentID string, expiresAt *string) (*models.Assignment, error) {
	return r.ResetExpirationWithRequest(ctx, documentID, assignmentID, models.ResetAssignmentExpirationRequest{ExpiresAt: expiresAt})
}

// ResetExpirationWithRequest sends body as {"expires_at": string | null} and
// returns the updated Assignment. A nil body.ExpiresAt sends null, which
// removes the expiration. It uses client authentication.
//
// PUT /documents/{document_id}/assignments/{assignment_id}/reset-expiration.
func (r *AssignmentResource) ResetExpirationWithRequest(ctx context.Context, documentID, assignmentID string, body models.ResetAssignmentExpirationRequest) (*models.Assignment, error) {
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
		WithSignerAccessCode(signerAccessCode).
		WithBody(nonNil(items)).
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
		WithSignerAccessCode(signerAccessCode).
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

// GetIdentityReview uses client API-key or bearer authentication and returns
// the signer's identity evidence summary for account review. An unavailable
// verification produces a 404 APIError. Images are available only InReview.
// GET /documents/{document_id}/assignments/{assignment_id}/signers/{signer_id}/identity-verification.
func (r *AssignmentResource) GetIdentityReview(ctx context.Context, documentID, assignmentID, signerID string) (*models.SignerIdentityReview, error) {
	var out models.SignerIdentityReview
	_, err := r.http.NewRequest(http.MethodGet, identityReviewPath(documentID, assignmentID, signerID)).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DownloadIdentityReviewImage uses client API-key or bearer authentication and
// returns raw JPEG, PNG or PDF evidence bytes. name must appear in the review's
// Images (document_front, document_back, document_pdf, face). Images are fetched
// from the provider only while InReview; once decided the API returns 404.
// GET /documents/{document_id}/assignments/{assignment_id}/signers/{signer_id}/identity-verification/images/{name}.
func (r *AssignmentResource) DownloadIdentityReviewImage(ctx context.Context, documentID, assignmentID, signerID, name string) ([]byte, error) {
	return r.http.Download(ctx, identityReviewPath(documentID, assignmentID, signerID)+"/images/"+url.PathEscape(name), nil)
}

// DecideIdentityReview sends DecideIdentityReviewRequest using client API-key
// or bearer authentication and returns the decided review. The verification
// must be InReview and the document pending_signature or pending_identity_review.
// Approval releases certification when every signer has signed and no review
// remains. Rejection cancels the document and notifies signers and copy receivers.
// POST /documents/{document_id}/assignments/{assignment_id}/signers/{signer_id}/identity-verification/review.
func (r *AssignmentResource) DecideIdentityReview(ctx context.Context, documentID, assignmentID, signerID string, body *models.DecideIdentityReviewRequest) (*models.SignerIdentityReview, error) {
	var out models.SignerIdentityReview
	_, err := r.http.NewRequest(http.MethodPost, identityReviewPath(documentID, assignmentID, signerID)+"/review").WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func identityReviewPath(documentID, assignmentID, signerID string) string {
	return "/documents/" + url.PathEscape(documentID) + "/assignments/" + url.PathEscape(assignmentID) +
		"/signers/" + url.PathEscape(signerID) + "/identity-verification"
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
	req := r.http.NewRequest(http.MethodGet, "/sign").WithSignerAccessCode(signerAccessCode)
	if hasAcceptedTerms != nil {
		req.WithQuery("has_accepted_terms", strconv.FormatBool(*hasAcceptedTerms))
	}
	_, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
