package resources

import (
	"context"
	"fmt"
	"net/http"

	"github.com/assinafy/assinafy-go/internal"
	"github.com/assinafy/assinafy-go/models"
)

type SignerResource struct {
	httpClient *internal.HTTPClient
	accountID  string
}

func NewSignerResource(httpClient *internal.HTTPClient, accountID string) *SignerResource {
	return &SignerResource{
		httpClient: httpClient,
		accountID:  accountID,
	}
}

func (r *SignerResource) Create(ctx context.Context, accountID string, req *models.CreateSignerRequest) (*models.Signer, error) {
	if accountID == "" {
		accountID = r.accountID
	}

	var result models.Signer
	httpReq := r.httpClient.NewRequest(http.MethodPost, fmt.Sprintf("/accounts/%s/signers", accountID))
	httpReq.WithBody(req)
	_, err := httpReq.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *SignerResource) List(ctx context.Context, accountID string, params *models.ListParams) (*models.PaginatedResult[models.Signer], error) {
	if accountID == "" {
		accountID = r.accountID
	}
	if params == nil {
		params = &models.ListParams{}
	}
	params.SetDefaults()

	var result []models.Signer
	req := r.httpClient.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%s/signers", accountID))
	req.WithQuery("page", fmt.Sprintf("%d", params.Page))
	req.WithQuery("per-page", fmt.Sprintf("%d", params.PerPage))
	if params.Search != "" {
		req.WithQuery("search", params.Search)
	}

	resp, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &models.PaginatedResult[models.Signer]{
		Data:       result,
		Pagination: extractPaginationMeta(resp.Headers),
	}, nil
}

func (r *SignerResource) Get(ctx context.Context, accountID, signerID string) (*models.Signer, error) {
	var result models.Signer
	req := r.httpClient.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%s/signers/%s", accountID, signerID))
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *SignerResource) Update(ctx context.Context, accountID, signerID string, req *models.UpdateSignerRequest) (*models.Signer, error) {
	var result models.Signer
	httpReq := r.httpClient.NewRequest(http.MethodPut, fmt.Sprintf("/accounts/%s/signers/%s", accountID, signerID))
	httpReq.WithBody(req)
	_, err := httpReq.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *SignerResource) Delete(ctx context.Context, accountID, signerID string) error {
	req := r.httpClient.NewRequest(http.MethodDelete, fmt.Sprintf("/accounts/%s/signers/%s", accountID, signerID))
	_, err := req.Execute(ctx, nil)
	return err
}

func (r *SignerResource) GetSelf(ctx context.Context, signerAccessCode string) (*models.Signer, error) {
	var result models.Signer
	req := r.httpClient.NewRequest(http.MethodGet, "/signers/self")
	req.WithQuery("signer-access-code", signerAccessCode)
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *SignerResource) AcceptTerms(ctx context.Context, signerAccessCode string) error {
	body := map[string]string{"signer-access-code": signerAccessCode}
	req := r.httpClient.NewRequest(http.MethodPut, "/signers/accept-terms")
	req.WithBody(body)
	_, err := req.Execute(ctx, nil)
	return err
}

func (r *SignerResource) VerifyEmail(ctx context.Context, signerAccessCode, verificationCode string) error {
	body := map[string]string{
		"signer-access-code": signerAccessCode,
		"verification-code":  verificationCode,
	}
	req := r.httpClient.NewRequest(http.MethodPost, "/verify")
	req.WithBody(body)
	_, err := req.Execute(ctx, nil)
	return err
}

func (r *SignerResource) ConfirmData(ctx context.Context, documentID, signerAccessCode string, req *models.UpdateSignerRequest) error {
	httpReq := r.httpClient.NewRequest(http.MethodPut, fmt.Sprintf("/documents/%s/signers/confirm-data", documentID))
	httpReq.WithQuery("signer-access-code", signerAccessCode)
	httpReq.WithBody(req)
	_, err := httpReq.Execute(ctx, nil)
	return err
}

func (r *SignerResource) UploadSignature(ctx context.Context, signerAccessCode, signatureType string, imageContent []byte) error {
	req := r.httpClient.NewRequest(http.MethodPost, fmt.Sprintf("/signature?signer-access-code=%s&type=%s", signerAccessCode, signatureType))
	req.WithHeader("Content-Type", "image/png")
	req.WithBody(imageContent)
	_, err := req.Execute(ctx, nil)
	return err
}

func (r *SignerResource) DownloadSignature(ctx context.Context, signerAccessCode, signatureType string) ([]byte, error) {
	return r.httpClient.Download(ctx, fmt.Sprintf("/signature/%s", signatureType), map[string]string{"signer-access-code": signerAccessCode})
}
