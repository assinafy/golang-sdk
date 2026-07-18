package resources

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// DocumentResource exposes the documented `Document` endpoints.
type DocumentResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewDocumentResource constructs a DocumentResource bound to the given
// HTTPClient and default account ID.
func NewDocumentResource(httpClient *internal.HTTPClient, accountID string) *DocumentResource {
	return &DocumentResource{http: httpClient, accountID: accountID}
}

// Upload uploads a file as a new document. POST /accounts/{account_id}/documents.
func (r *DocumentResource) Upload(ctx context.Context, accountID string, fileContent []byte, fileName string, metadata map[string]string) (*models.Document, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	form := map[string]string{"name": fileName}
	for k, v := range metadata {
		form[k] = v
	}

	var doc models.Document
	path := "/accounts/" + url.PathEscape(accountID) + "/documents"
	if _, err := r.http.UploadMultipart(ctx, path, "file", fileName, fileContent, form, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// List returns the workspace documents page.
// GET /accounts/{account_id}/documents.
func (r *DocumentResource) List(ctx context.Context, accountID string, params *models.ListParams) (*models.PaginatedResult[models.Document], error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var docs []models.Document
	req := r.http.NewRequest(http.MethodGet, "/accounts/"+url.PathEscape(accountID)+"/documents")
	applyListParams(req, params)
	if params != nil {
		req.WithQuery("status", params.Status)
		req.WithQuery("method", params.Method)
		if len(params.Tags) > 0 {
			req.WithQuery("tags", strings.Join(params.Tags, ","))
		}
	}

	resp, err := req.Execute(ctx, &docs)
	if err != nil {
		return nil, err
	}
	return paginated(docs, resp), nil
}

// Search returns the workspace documents matching the search term, ordered by
// relevance and paginated via the X-Pagination-* response headers. It accepts
// the same status/method/tag filters as List.
// GET /accounts/{account_id}/documents/search.
func (r *DocumentResource) Search(ctx context.Context, accountID string, params *models.ListParams) (*models.PaginatedResult[models.Document], error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var docs []models.Document
	req := r.http.NewRequest(http.MethodGet, "/accounts/"+url.PathEscape(accountID)+"/documents/search")
	applyListParams(req, params)
	if params != nil {
		req.WithQuery("status", params.Status)
		req.WithQuery("method", params.Method)
		if len(params.Tags) > 0 {
			req.WithQuery("tags", strings.Join(params.Tags, ","))
		}
	}

	resp, err := req.Execute(ctx, &docs)
	if err != nil {
		return nil, err
	}
	return paginated(docs, resp), nil
}

// Get retrieves a single document. GET /documents/{document_id}.
func (r *DocumentResource) Get(ctx context.Context, documentID string) (*models.Document, error) {
	var doc models.Document
	_, err := r.http.NewRequest(http.MethodGet, "/documents/"+url.PathEscape(documentID)).Execute(ctx, &doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// Delete removes a document. DELETE /documents/{document_id}.
func (r *DocumentResource) Delete(ctx context.Context, documentID string) error {
	_, err := r.http.NewRequest(http.MethodDelete, "/documents/"+url.PathEscape(documentID)).Execute(ctx, nil)
	return err
}

// Rename changes a document's name and returns the updated document. Renaming is
// only allowed before the signature process starts (while the document is in
// uploaded or metadata_ready status with no signers); once signing has begun or
// the document is certificated the API rejects the change with a 400. The name
// is normalized server-side (diacritics removed, unsupported characters replaced
// with dashes).
// PATCH /documents/{document_id}.
func (r *DocumentResource) Rename(ctx context.Context, documentID, name string) (*models.Document, error) {
	var out models.Document
	path := "/documents/" + url.PathEscape(documentID)
	if _, err := r.http.NewRequest(http.MethodPatch, path).
		WithBody(models.RenameDocumentRequest{Name: name}).
		Execute(ctx, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Activities lists the audit-trail entries on a document.
// GET /documents/{document_id}/activities.
func (r *DocumentResource) Activities(ctx context.Context, documentID string) ([]models.DocumentActivity, error) {
	var out []models.DocumentActivity
	_, err := r.http.NewRequest(http.MethodGet, "/documents/"+url.PathEscape(documentID)+"/activities").Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Download fetches a document artifact. GET /documents/{document_id}/download/{artifact}.
func (r *DocumentResource) Download(ctx context.Context, documentID, artifact string) ([]byte, error) {
	return r.http.Download(ctx, "/documents/"+url.PathEscape(documentID)+"/download/"+url.PathEscape(artifact), nil)
}

// Thumbnail downloads the document thumbnail JPEG.
// GET /documents/{document_id}/thumbnail.
func (r *DocumentResource) Thumbnail(ctx context.Context, documentID string) ([]byte, error) {
	return r.http.Download(ctx, "/documents/"+url.PathEscape(documentID)+"/thumbnail", nil)
}

// DownloadPage downloads a single rendered page.
// GET /documents/{document_id}/pages/{page_id}/download.
func (r *DocumentResource) DownloadPage(ctx context.Context, documentID, pageID string) ([]byte, error) {
	return r.http.Download(ctx, "/documents/"+url.PathEscape(documentID)+"/pages/"+url.PathEscape(pageID)+"/download", nil)
}

// Verify validates a document by signature hash.
// GET /documents/{signature_hash}/verify.
func (r *DocumentResource) Verify(ctx context.Context, signatureHash string) (*models.VerifyDocumentResult, error) {
	var out models.VerifyDocumentResult
	_, err := r.http.NewRequest(http.MethodGet, "/documents/"+url.PathEscape(signatureHash)+"/verify").Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateFromTemplate creates a document from a template.
// POST /accounts/{account_id}/templates/{template_id}/documents.
func (r *DocumentResource) CreateFromTemplate(ctx context.Context, accountID, templateID string, signers []models.TemplateSigner, opts *models.CreateDocumentFromTemplateOptions) (*models.Document, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	body := models.CreateDocumentFromTemplateOptions{Signers: signers}
	if opts != nil {
		body = *opts
		body.Signers = signers
	}

	var out models.Document
	path := "/accounts/" + url.PathEscape(accountID) + "/templates/" + url.PathEscape(templateID) + "/documents"
	_, err := r.http.NewRequest(http.MethodPost, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// EstimateCostFromTemplate returns the cost estimate of creating a document from a template.
// POST /accounts/{account_id}/templates/{template_id}/documents/estimate-cost.
func (r *DocumentResource) EstimateCostFromTemplate(ctx context.Context, accountID, templateID string, signers []models.TemplateSigner) (*models.CostEstimate, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out models.CostEstimate
	body := map[string]any{"signers": signers}
	path := "/accounts/" + url.PathEscape(accountID) + "/templates/" + url.PathEscape(templateID) + "/documents/estimate-cost"
	_, err := r.http.NewRequest(http.MethodPost, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListTags lists the tags currently attached to a document.
// GET /accounts/{account_id}/documents/{document_id}/tags.
func (r *DocumentResource) ListTags(ctx context.Context, accountID, documentID string) ([]models.Tag, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var out []models.Tag
	path := "/accounts/" + url.PathEscape(accountID) + "/documents/" + url.PathEscape(documentID) + "/tags"
	if _, err := r.http.NewRequest(http.MethodGet, path).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ReplaceTags replaces a document's tag set with the provided names, returning
// the resulting tags. An empty slice detaches all tags. Unknown names are
// created automatically (case-insensitive lookup).
// PUT /accounts/{account_id}/documents/{document_id}/tags.
func (r *DocumentResource) ReplaceTags(ctx context.Context, accountID, documentID string, tags []string) ([]models.Tag, error) {
	return r.writeTags(ctx, http.MethodPut, accountID, documentID, tags)
}

// AppendTags attaches additional tags to a document without removing existing
// ones, returning the resulting tags. Re-attaching a present tag is a no-op and
// unknown names are created automatically.
// POST /accounts/{account_id}/documents/{document_id}/tags.
func (r *DocumentResource) AppendTags(ctx context.Context, accountID, documentID string, tags []string) ([]models.Tag, error) {
	return r.writeTags(ctx, http.MethodPost, accountID, documentID, tags)
}

func (r *DocumentResource) writeTags(ctx context.Context, method, accountID, documentID string, tags []string) ([]models.Tag, error) {
	accountID = resolveAccountID(accountID, r.accountID)
	if tags == nil {
		tags = []string{}
	}

	var out []models.Tag
	path := "/accounts/" + url.PathEscape(accountID) + "/documents/" + url.PathEscape(documentID) + "/tags"
	if _, err := r.http.NewRequest(method, path).WithBody(models.SetDocumentTagsRequest{Tags: tags}).Execute(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DetachTag detaches a single tag from a document. The tag itself is not
// deleted, and detaching a tag that was not attached is a no-op.
// DELETE /accounts/{account_id}/documents/{document_id}/tags/{tag_id}.
func (r *DocumentResource) DetachTag(ctx context.Context, accountID, documentID, tagID string) error {
	accountID = resolveAccountID(accountID, r.accountID)

	path := "/accounts/" + url.PathEscape(accountID) + "/documents/" + url.PathEscape(documentID) + "/tags/" + url.PathEscape(tagID)
	_, err := r.http.NewRequest(http.MethodDelete, path).Execute(ctx, nil)
	return err
}

// ListStatuses returns the documented status codes.
// GET /documents/statuses.
func (r *DocumentResource) ListStatuses(ctx context.Context) ([]models.DocumentStatusInfo, error) {
	var out []models.DocumentStatusInfo
	_, err := r.http.NewRequest(http.MethodGet, "/documents/statuses").Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
