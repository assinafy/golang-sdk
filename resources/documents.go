package resources

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

const maxDocumentSize = 25 * 1024 * 1024

// DocumentResource exposes authenticated document lifecycle, artifact, and tag
// endpoints. Its methods follow the package-level account and error contract;
// Verify is the unauthenticated exception.
type DocumentResource struct {
	http      *internal.HTTPClient
	accountID string
}

// NewDocumentResource constructs a DocumentResource bound to the given
// HTTPClient and default account ID.
func NewDocumentResource(httpClient *internal.HTTPClient, accountID string) *DocumentResource {
	return &DocumentResource{http: httpClient, accountID: accountID}
}

// Upload sends fileContent as the required multipart "file" part and returns the
// created Document. The SDK also sends fileName as a compatibility "name" part
// plus optional metadata fields. It requires client authentication; an empty
// accountID uses the default. Processing continues asynchronously after response.
// POST /accounts/{account_id}/documents.
func (r *DocumentResource) Upload(ctx context.Context, accountID string, fileContent []byte, fileName string, metadata map[string]string) (*models.Document, error) {
	if err := validateDocumentUpload(fileContent, fileName); err != nil {
		return nil, err
	}
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

func validateDocumentUpload(fileContent []byte, fileName string) error {
	if strings.TrimSpace(fileName) == "" {
		return fmt.Errorf("%w: document file name is empty", sdkerrors.ErrInvalidInput)
	}
	if len(fileContent) == 0 {
		return fmt.Errorf("%w: document file is empty", sdkerrors.ErrInvalidInput)
	}
	if len(fileContent) > maxDocumentSize {
		return fmt.Errorf("%w: document exceeds the 25 MB API limit", sdkerrors.ErrInvalidInput)
	}
	header := fileContent
	if len(header) > 1024 {
		header = header[:1024]
	}
	if !bytes.Contains(header, []byte("%PDF-")) {
		return fmt.Errorf("%w: document is not a PDF", sdkerrors.ErrInvalidInput)
	}
	return nil
}

// List returns a page of Document payloads and X-Pagination metadata using the
// applicable ListParams filters. It requires client authentication; an empty
// accountID uses the configured default.
// GET /accounts/{account_id}/documents.
func (r *DocumentResource) List(ctx context.Context, accountID string, params *models.ListParams) (*models.PaginatedResult[models.Document], error) {
	return r.listDocuments(ctx, accountID, "/documents", params)
}

// Search returns matching Document payloads ordered by relevance with
// X-Pagination metadata. The current contract uses search and status; method,
// tags, and sort are compatibility filters carried by ListParams. It requires
// client authentication and uses the default for an empty accountID.
// GET /accounts/{account_id}/documents/search.
func (r *DocumentResource) Search(ctx context.Context, accountID string, params *models.ListParams) (*models.PaginatedResult[models.Document], error) {
	return r.listDocuments(ctx, accountID, "/documents/search", params)
}

// listDocuments issues an account-scoped document listing at suffix, applying the
// shared pagination parameters plus the status, method, and tag filters.
func (r *DocumentResource) listDocuments(ctx context.Context, accountID, suffix string, params *models.ListParams) (*models.PaginatedResult[models.Document], error) {
	accountID = resolveAccountID(accountID, r.accountID)

	var docs []models.Document
	req := r.http.NewRequest(http.MethodGet, "/accounts/"+url.PathEscape(accountID)+suffix)
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

// Get returns the authenticated Document payload for documentID.
// GET /documents/{document_id}.
func (r *DocumentResource) Get(ctx context.Context, documentID string) (*models.Document, error) {
	var doc models.Document
	_, err := r.http.NewRequest(http.MethodGet, "/documents/"+url.PathEscape(documentID)).Execute(ctx, &doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// Delete permanently removes an authenticated, deletable document and returns
// only error, discarding the documented empty data array. A non-deletable
// lifecycle state produces an API error.
// DELETE /documents/{document_id}.
func (r *DocumentResource) Delete(ctx context.Context, documentID string) error {
	_, err := r.http.NewRequest(http.MethodDelete, "/documents/"+url.PathEscape(documentID)).Execute(ctx, nil)
	return err
}

// Rename sends RenameDocumentRequest with name and returns the authenticated,
// updated Document. Renaming is only allowed before the signature process starts
// (while the document is in
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

// Activities returns the authenticated document's DocumentActivity audit-trail payloads.
// GET /documents/{document_id}/activities.
func (r *DocumentResource) Activities(ctx context.Context, documentID string) ([]models.DocumentActivity, error) {
	var out []models.DocumentActivity
	_, err := r.http.NewRequest(http.MethodGet, "/documents/"+url.PathEscape(documentID)+"/activities").Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Download returns raw bytes for the authenticated document artifact named by
// artifact. An unavailable artifact or invalid artifact code produces an API error.
// GET /documents/{document_id}/download/{artifact}.
func (r *DocumentResource) Download(ctx context.Context, documentID, artifact string) ([]byte, error) {
	return r.http.Download(ctx, "/documents/"+url.PathEscape(documentID)+"/download/"+url.PathEscape(artifact), nil)
}

// Thumbnail returns the authenticated document thumbnail as raw image bytes. A
// thumbnail that is not yet available produces an API error.
// GET /documents/{document_id}/thumbnail.
func (r *DocumentResource) Thumbnail(ctx context.Context, documentID string) ([]byte, error) {
	return r.http.Download(ctx, "/documents/"+url.PathEscape(documentID)+"/thumbnail", nil)
}

// DownloadPage returns one authenticated rendered page as raw bytes. An unknown
// document or page produces an API error.
// GET /documents/{document_id}/pages/{page_id}/download.
func (r *DocumentResource) DownloadPage(ctx context.Context, documentID, pageID string) ([]byte, error) {
	return r.http.Download(ctx, "/documents/"+url.PathEscape(documentID)+"/pages/"+url.PathEscape(pageID)+"/download", nil)
}

// Verify validates signatureHash without requiring a client credential and
// returns VerifyDocumentResult; an unknown or malformed hash is represented by
// the API's verification payload or error response.
// GET /documents/{signature_hash}/verify.
func (r *DocumentResource) Verify(ctx context.Context, signatureHash string) (*models.VerifyDocumentResult, error) {
	var out models.VerifyDocumentResult
	_, err := r.http.NewRequest(http.MethodGet, "/documents/"+url.PathEscape(signatureHash)+"/verify").WithoutAuth().Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateFromTemplate sends CreateDocumentFromTemplateOptions with the required
// signer-role mappings and returns the created Document. It requires client
// authentication; an empty accountID uses the configured default. Generation
// continues asynchronously after the response.
// POST /accounts/{account_id}/templates/{template_id}/documents.
func (r *DocumentResource) CreateFromTemplate(ctx context.Context, accountID, templateID string, signers []models.TemplateSigner, opts *models.CreateDocumentFromTemplateOptions) (*models.Document, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	body := models.CreateDocumentFromTemplateOptions{Signers: nonNil(signers)}
	if opts != nil {
		body = *opts
		body.Signers = nonNil(signers)
	}

	var out models.Document
	path := "/accounts/" + url.PathEscape(accountID) + "/templates/" + url.PathEscape(templateID) + "/documents"
	_, err := r.http.NewRequest(http.MethodPost, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// EstimateCostFromTemplate sends the required TemplateSigner list and returns a
// CostEstimate without creating a document. It requires client authentication;
// an empty accountID uses the configured default.
// POST /accounts/{account_id}/templates/{template_id}/documents/estimate-cost.
func (r *DocumentResource) EstimateCostFromTemplate(ctx context.Context, accountID, templateID string, signers []models.TemplateSigner) (*models.CostEstimate, error) {
	accountID = resolveAccountID(accountID, r.accountID)

	type estimateSigner struct {
		RoleID              string   `json:"role_id"`
		VerificationMethod  string   `json:"verification_method,omitempty"`
		NotificationMethods []string `json:"notification_methods,omitempty"`
	}
	estimateSigners := make([]estimateSigner, len(signers))
	for i, signer := range signers {
		estimateSigners[i] = estimateSigner{
			RoleID:              signer.RoleID,
			VerificationMethod:  signer.VerificationMethod,
			NotificationMethods: signer.NotificationMethods,
		}
	}

	var out models.CostEstimate
	body := map[string]any{"signers": estimateSigners}
	path := "/accounts/" + url.PathEscape(accountID) + "/templates/" + url.PathEscape(templateID) + "/documents/estimate-cost"
	_, err := r.http.NewRequest(http.MethodPost, path).WithBody(body).Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListTags returns Tag payloads currently attached to the authenticated
// document. An empty accountID uses the configured default.
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

// ReplaceTags replaces a document's tag set with the provided tag IDs and
// returns the resulting tags. An empty slice detaches all tags. The live API
// also accepts names and creates unknown names as a compatibility extension. It
// requires client authentication and an empty accountID uses the default.
// PUT /accounts/{account_id}/documents/{document_id}/tags.
func (r *DocumentResource) ReplaceTags(ctx context.Context, accountID, documentID string, tags []string) ([]models.Tag, error) {
	return r.writeTags(ctx, http.MethodPut, accountID, documentID, tags)
}

// AppendTags attaches tag IDs without removing existing tags and returns the
// resulting set. Re-attaching a present tag is a no-op. The live API also
// accepts names and creates unknown names as a compatibility extension. It
// requires client authentication and an empty accountID uses the default.
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
// deleted, and detaching a tag that was not attached is a no-op. It requires
// client authentication, returns only error (discarding the documented
// {detached:boolean} result), and uses the default for an empty accountID.
// DELETE /accounts/{account_id}/documents/{document_id}/tags/{tag_id}.
func (r *DocumentResource) DetachTag(ctx context.Context, accountID, documentID, tagID string) error {
	return r.detachTag(ctx, accountID, documentID, tagID, nil)
}

// DetachTagWithResult detaches a tag and returns the documented result.
// DELETE /accounts/{account_id}/documents/{document_id}/tags/{tag_id}.
func (r *DocumentResource) DetachTagWithResult(ctx context.Context, accountID, documentID, tagID string) (*models.DocumentTagDetachResult, error) {
	var out models.DocumentTagDetachResult
	if err := r.detachTag(ctx, accountID, documentID, tagID, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *DocumentResource) detachTag(ctx context.Context, accountID, documentID, tagID string, out any) error {
	accountID = resolveAccountID(accountID, r.accountID)

	path := "/accounts/" + url.PathEscape(accountID) + "/documents/" + url.PathEscape(documentID) + "/tags/" + url.PathEscape(tagID)
	_, err := r.http.NewRequest(http.MethodDelete, path).Execute(ctx, out)
	return err
}

// ListStatuses returns the authenticated endpoint's DocumentStatusInfo payloads.
// GET /documents/statuses.
func (r *DocumentResource) ListStatuses(ctx context.Context) ([]models.DocumentStatusInfo, error) {
	var out []models.DocumentStatusInfo
	_, err := r.http.NewRequest(http.MethodGet, "/documents/statuses").Execute(ctx, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
