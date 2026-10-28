package resources

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/assinafy/golang-sdk/v2/internal"
	"github.com/assinafy/golang-sdk/v2/models"
)

// ContractResource manages document contract terms and amendment families.
// Methods use client API-key or bearer authentication and the configured account
// fallback. OAuth reads need documents:read and writes need documents:write;
// writes also need a workspace plan with contract tracking.
type ContractResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewContractResource constructs a ContractResource.
func NewContractResource(httpClient *internal.HTTPClient, accountID string) *ContractResource {
	return &ContractResource{http: httpClient, accountID: accountID}
}

// List returns tracked terms and X-Pagination metadata using client authentication.
// The default order is -updated_at; page until records predate the last sync to
// recover missed webhooks. Reads remain available when plan entitlement lapses.
// GET /accounts/{account_id}/contracts.
func (r *ContractResource) List(ctx context.Context, accountID string, params *models.ContractListParams) (*models.PaginatedResult[models.DocumentContract], error) {
	accountID = resolveAccountID(accountID, r.accountID)
	req := r.http.NewRequest(http.MethodGet, "/accounts/"+url.PathEscape(accountID)+"/contracts")
	var list *models.ListParams
	if params != nil {
		list = &params.ListParams
		req.WithQuery("status", params.Status).
			WithQuery("counterparty", params.Counterparty).
			WithQuery("end_date_from", params.EndDateFrom).
			WithQuery("end_date_to", params.EndDateTo).
			WithQuery("external_system", params.ExternalSystem).
			WithQuery("external_id", params.ExternalID).
			WithQuery("expand", params.Expand)
		if params.EndingWithinDays != nil {
			req.WithQuery("ending_within_days", strconv.Itoa(*params.EndingWithinDays))
		}
	}
	applyListParams(req, list)
	var out []models.DocumentContract
	resp, err := req.Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return paginated(out, resp), nil
}

// Get returns contract terms with amendments, or this document's amendment link,
// using client authentication. A document without terms returns a 404 APIError.
// GET /accounts/{account_id}/documents/{document_id}/contract.
func (r *ContractResource) Get(ctx context.Context, accountID, documentID string) (*models.ContractResult, error) {
	var out models.ContractResult
	_, err := r.http.NewRequest(http.MethodGet, r.contractPath(accountID, documentID)).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Set sends SetDocumentContractRequest using client authentication and returns
// root terms or an amendment link. Only supplied fields change. Linking an
// already certificated instrument can immediately change its parent contract.
// Send the read UpdatedAt to reject concurrent edits with 409 ContractChanged.
// PUT /accounts/{account_id}/documents/{document_id}/contract.
func (r *ContractResource) Set(ctx context.Context, accountID, documentID string, body *models.SetDocumentContractRequest) (*models.ContractResult, error) {
	var out models.ContractResult
	_, err := r.http.NewRequest(http.MethodPut, r.contractPath(accountID, documentID)).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Related returns the root contract and its amendment family using client
// authentication, starting from either the root document or an instrument.
// GET /accounts/{account_id}/documents/{document_id}/contract/related.
func (r *ContractResource) Related(ctx context.Context, accountID, documentID string) (*models.DocumentContract, error) {
	var out models.DocumentContract
	_, err := r.http.NewRequest(http.MethodGet, r.contractPath(accountID, documentID)+"/related").Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Renew sends RenewContractRequest using client authentication and returns the
// next term. Send the current Term or a later EndDate to prevent duplicate
// renewal. UpdatedAt optionally rejects concurrent changes with 409.
// POST /accounts/{account_id}/documents/{document_id}/contract/renew.
func (r *ContractResource) Renew(ctx context.Context, accountID, documentID string, body *models.RenewContractRequest) (*models.DocumentContract, error) {
	var out models.DocumentContract
	_, err := r.http.NewRequest(http.MethodPost, r.contractPath(accountID, documentID)+"/renew").WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Terminate sends the optional TerminateContractRequest using client authentication,
// returns the closed contract, and stops lifecycle alerts. The reason is kept in
// document activities; UpdatedAt optionally rejects concurrent changes with 409.
// POST /accounts/{account_id}/documents/{document_id}/contract/terminate.
func (r *ContractResource) Terminate(ctx context.Context, accountID, documentID string, body *models.TerminateContractRequest) (*models.DocumentContract, error) {
	if body == nil {
		body = &models.TerminateContractRequest{}
	}
	var out models.DocumentContract
	_, err := r.http.NewRequest(http.MethodPost, r.contractPath(accountID, documentID)+"/terminate").WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *ContractResource) contractPath(accountID, documentID string) string {
	accountID = resolveAccountID(accountID, r.accountID)
	return "/accounts/" + url.PathEscape(accountID) + "/documents/" + url.PathEscape(documentID) + "/contract"
}
