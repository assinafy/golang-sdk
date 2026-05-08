package resources

import (
	"context"
	"fmt"
	"net/http"

	"github.com/assinafy/assinafy-go/internal"
	"github.com/assinafy/assinafy-go/models"
)

type AssignmentResource struct {
	httpClient *internal.HTTPClient
	accountID  string
}

func NewAssignmentResource(httpClient *internal.HTTPClient, accountID string) *AssignmentResource {
	return &AssignmentResource{
		httpClient: httpClient,
		accountID:  accountID,
	}
}

func (r *AssignmentResource) Create(ctx context.Context, documentID string, req *models.CreateAssignmentRequest) (*models.Assignment, error) {
	var result models.Assignment
	httpReq := r.httpClient.NewRequest(http.MethodPost, fmt.Sprintf("/documents/%s/assignments", documentID))
	httpReq.WithBody(req)
	_, err := httpReq.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *AssignmentResource) EstimateCost(ctx context.Context, documentID string, req *models.CreateAssignmentRequest) (*models.EstimateCostResult, error) {
	var result models.EstimateCostResult
	httpReq := r.httpClient.NewRequest(http.MethodPost, fmt.Sprintf("/documents/%s/assignments/estimate-cost", documentID))
	httpReq.WithBody(req)
	_, err := httpReq.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *AssignmentResource) ResendNotification(ctx context.Context, documentID, assignmentID, signerID string) error {
	req := r.httpClient.NewRequest(http.MethodPut, fmt.Sprintf("/documents/%s/assignments/%s/signers/%s/resend", documentID, assignmentID, signerID))
	_, err := req.Execute(ctx, nil)
	return err
}

func (r *AssignmentResource) EstimateResendCost(ctx context.Context, documentID, assignmentID, signerID string) (*models.EstimateCostResult, error) {
	var result models.EstimateCostResult
	req := r.httpClient.NewRequest(http.MethodPost, fmt.Sprintf("/documents/%s/assignments/%s/signers/%s/estimate-resend-cost", documentID, assignmentID, signerID))
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *AssignmentResource) ResetExpiration(ctx context.Context, documentID, assignmentID, expiresAt string) error {
	body := map[string]string{"expires_at": expiresAt}
	req := r.httpClient.NewRequest(http.MethodPut, fmt.Sprintf("/documents/%s/assignments/%s/reset-expiration", documentID, assignmentID))
	req.WithBody(body)
	_, err := req.Execute(ctx, nil)
	return err
}

func (r *AssignmentResource) GetSummary(ctx context.Context, documentID, assignmentID string) (*models.AssignmentSummary, error) {
	var result models.AssignmentSummary
	req := r.httpClient.NewRequest(http.MethodGet, fmt.Sprintf("/documents/%s/assignments/%s/summary", documentID, assignmentID))
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *AssignmentResource) Cancel(ctx context.Context, documentID, assignmentID, reason string) error {
	body := map[string]string{"reason": reason}
	req := r.httpClient.NewRequest(http.MethodDelete, fmt.Sprintf("/documents/%s/assignments/%s", documentID, assignmentID))
	req.WithBody(body)
	_, err := req.Execute(ctx, nil)
	return err
}

func (r *AssignmentResource) Sign(ctx context.Context, documentID, assignmentID string, opts *models.SignDocumentOptions) error {
	body := map[string]bool{}
	if opts != nil {
		body["has_accepted_terms"] = opts.HasAcceptedTerms
	}
	req := r.httpClient.NewRequest(http.MethodPost, fmt.Sprintf("/documents/%s/assignments/%s", documentID, assignmentID))
	req.WithBody(body)
	_, err := req.Execute(ctx, nil)
	return err
}

func (r *AssignmentResource) Decline(ctx context.Context, documentID, assignmentID, signerAccessCode, reason string) error {
	body := map[string]string{"reason": reason}
	req := r.httpClient.NewRequest(http.MethodPut, fmt.Sprintf("/documents/%s/assignments/%s/reject", documentID, assignmentID))
	req.WithQuery("signer-access-code", signerAccessCode)
	req.WithBody(body)
	_, err := req.Execute(ctx, nil)
	return err
}

func (r *AssignmentResource) ListWhatsAppNotifications(ctx context.Context, documentID, assignmentID string) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	req := r.httpClient.NewRequest(http.MethodGet, fmt.Sprintf("/documents/%s/assignments/%s/whatsapp-notifications", documentID, assignmentID))
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AssignmentResource) GetSigningInfo(ctx context.Context, signerAccessCode string) (*models.Document, error) {
	var result models.Document
	req := r.httpClient.NewRequest(http.MethodGet, "/sign")
	req.WithQuery("signer-access-code", signerAccessCode)
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
