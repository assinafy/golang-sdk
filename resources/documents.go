package resources

import (
	"context"
	"fmt"
	"net/http"

	"github.com/assinafy/assinafy-go/internal"
	"github.com/assinafy/assinafy-go/models"
)

type DocumentResource struct {
	httpClient *internal.HTTPClient
	accountID  string
}

func NewDocumentResource(httpClient *internal.HTTPClient, accountID string) *DocumentResource {
	return &DocumentResource{
		httpClient: httpClient,
		accountID:  accountID,
	}
}

func (r *DocumentResource) Upload(ctx context.Context, accountID string, fileContent []byte, fileName string, metadata map[string]string) (*models.DocumentUploadResponse, error) {
	if accountID == "" {
		accountID = r.accountID
	}

	formFields := make(map[string]string)
	formFields["name"] = fileName
	for k, v := range metadata {
		formFields[k] = v
	}

	var result models.DocumentUploadResponse
	_, err := r.httpClient.UploadMultipart(ctx, fmt.Sprintf("/accounts/%s/documents", accountID), "file", fileName, fileContent, formFields, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *DocumentResource) List(ctx context.Context, accountID string, params *models.ListParams) (*models.PaginatedResult[models.DocumentListItem], error) {
	if accountID == "" {
		accountID = r.accountID
	}
	if params == nil {
		params = &models.ListParams{}
	}
	params.SetDefaults()

	var result []models.DocumentListItem
	req := r.httpClient.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%s/documents", accountID))
	req.WithQuery("page", fmt.Sprintf("%d", params.Page))
	req.WithQuery("per-page", fmt.Sprintf("%d", params.PerPage))
	if params.Search != "" {
		req.WithQuery("search", params.Search)
	}
	if params.Sort != "" {
		req.WithQuery("sort", params.Sort)
	}

	resp, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &models.PaginatedResult[models.DocumentListItem]{
		Data:       result,
		Pagination: extractPagination(resp.Headers),
	}, nil
}

func (r *DocumentResource) Get(ctx context.Context, documentID string) (*models.Document, error) {
	var result models.Document
	req := r.httpClient.NewRequest(http.MethodGet, fmt.Sprintf("/documents/%s", documentID))
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *DocumentResource) Delete(ctx context.Context, documentID string) error {
	req := r.httpClient.NewRequest(http.MethodDelete, fmt.Sprintf("/documents/%s", documentID))
	_, err := req.Execute(ctx, nil)
	return err
}

func (r *DocumentResource) Activities(ctx context.Context, documentID string) ([]models.DocumentActivity, error) {
	var result []models.DocumentActivity
	req := r.httpClient.NewRequest(http.MethodGet, fmt.Sprintf("/documents/%s/activities", documentID))
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *DocumentResource) Download(ctx context.Context, documentID string, artifactName string) ([]byte, error) {
	return r.httpClient.Download(ctx, fmt.Sprintf("/documents/%s/download/%s", documentID, artifactName), nil)
}

func (r *DocumentResource) Thumbnail(ctx context.Context, documentID string) ([]byte, error) {
	return r.httpClient.Download(ctx, fmt.Sprintf("/documents/%s/thumbnail", documentID), nil)
}

func (r *DocumentResource) DownloadPage(ctx context.Context, documentID, pageID string) ([]byte, error) {
	return r.httpClient.Download(ctx, fmt.Sprintf("/documents/%s/pages/%s/download", documentID, pageID), nil)
}

func (r *DocumentResource) Verify(ctx context.Context, signatureHash string) (*models.VerifyDocumentResult, error) {
	var result models.VerifyDocumentResult
	req := r.httpClient.NewRequest(http.MethodGet, fmt.Sprintf("/documents/%s/verify", signatureHash))
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *DocumentResource) CreateFromTemplate(ctx context.Context, accountID, templateID string, signers []models.TemplateSigner, opts *models.CreateDocumentFromTemplateOptions) (*models.Document, error) {
	if accountID == "" {
		accountID = r.accountID
	}

	if opts == nil {
		opts = &models.CreateDocumentFromTemplateOptions{}
	}
	opts.Signers = signers

	var result models.Document
	req := r.httpClient.NewRequest(http.MethodPost, fmt.Sprintf("/accounts/%s/templates/%s/documents", accountID, templateID))
	req.WithBody(opts)
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *DocumentResource) EstimateCostFromTemplate(ctx context.Context, accountID, templateID string, signers []models.TemplateSigner) (*models.EstimateCostResult, error) {
	if accountID == "" {
		accountID = r.accountID
	}

	body := map[string]interface{}{"signers": signers}

	var result models.EstimateCostResult
	req := r.httpClient.NewRequest(http.MethodPost, fmt.Sprintf("/accounts/%s/templates/%s/documents/estimate-cost", accountID, templateID))
	req.WithBody(body)
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *DocumentResource) GetSigningProgress(ctx context.Context, documentID string) (*models.SigningProgress, error) {
	var result models.SigningProgress
	req := r.httpClient.NewRequest(http.MethodGet, fmt.Sprintf("/documents/%s/signing-progress", documentID))
	_, err := req.Execute(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func extractPagination(headers map[string][]string) models.PaginationMeta {
	meta := models.PaginationMeta{}
	if v, ok := headers["X-Pagination-Current-Page"]; ok && len(v) > 0 {
		fmt.Sscanf(v[0], "%d", &meta.CurrentPage)
	}
	if v, ok := headers["X-Pagination-Total-Count"]; ok && len(v) > 0 {
		fmt.Sscanf(v[0], "%d", &meta.TotalCount)
	}
	if v, ok := headers["X-Pagination-Page-Count"]; ok && len(v) > 0 {
		fmt.Sscanf(v[0], "%d", &meta.PageCount)
	}
	if v, ok := headers["X-Pagination-Per-Page"]; ok && len(v) > 0 {
		fmt.Sscanf(v[0], "%d", &meta.PerPage)
	}
	return meta
}
