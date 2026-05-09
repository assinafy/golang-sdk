// Package assinafy is the Go SDK for the Assinafy API.
//
// See https://api.assinafy.com.br/v1/docs for the upstream documentation.
//
// Typical usage:
//
//	client, err := assinafy.NewClient(assinafy.ClientOptions{
//	    APIKey:    os.Getenv("ASSINAFY_API_KEY"),
//	    AccountID: os.Getenv("ASSINAFY_ACCOUNT_ID"),
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	docs, err := client.Documents.List(ctx, "", nil)
package assinafy

import (
	"context"
	"errors"
	"time"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
	"github.com/assinafy/golang-sdk/resources"
)

const (
	// DefaultBaseURL is the production API base URL.
	DefaultBaseURL = "https://api.assinafy.com.br/v1"

	// SandboxBaseURL is the sandbox environment base URL.
	SandboxBaseURL = "https://sandbox.assinafy.com.br/v1"

	defaultTimeout = 30 * time.Second
)

// Client is the top-level SDK entry point. It groups the per-resource clients
// and is safe for concurrent use.
type Client struct {
	accountID string

	Documents       *resources.DocumentResource
	Signers         *resources.SignerResource
	SignerDocuments *resources.SignerDocumentResource
	Assignments     *resources.AssignmentResource
	Webhooks        *resources.WebhookResource
	Templates       *resources.TemplateResource
	Fields          *resources.FieldResource
	Authentication  *resources.AuthenticationResource
	PublicDocuments *resources.PublicDocumentResource
}

// ClientOptions configures Client construction.
type ClientOptions struct {
	// APIKey is the permanent X-Api-Key credential. Takes precedence over Token.
	APIKey string
	// Token is a JWT access token sent as Authorization: Bearer.
	Token string
	// AccountID is the default workspace identifier used for account-scoped resources
	// when the per-call accountID argument is empty.
	AccountID string
	// BaseURL overrides the API base URL. Defaults to DefaultBaseURL.
	BaseURL string
	// Timeout is the per-request HTTP timeout. Defaults to 30 seconds.
	Timeout time.Duration
}

// NewClient builds a new Client. Credentials are optional: unauthenticated
// public document endpoints, login flows, and signer-access-code endpoints can
// be used without an API key or access token.
func NewClient(opts ClientOptions) (*Client, error) {
	if opts.BaseURL == "" {
		opts.BaseURL = DefaultBaseURL
	}
	if opts.Timeout <= 0 {
		opts.Timeout = defaultTimeout
	}

	httpClient := internal.NewHTTPClient(opts.BaseURL, opts.APIKey, opts.Token, opts.Timeout)

	return &Client{
		accountID:       opts.AccountID,
		Documents:       resources.NewDocumentResource(httpClient, opts.AccountID),
		Signers:         resources.NewSignerResource(httpClient, opts.AccountID),
		SignerDocuments: resources.NewSignerDocumentResource(httpClient),
		Assignments:     resources.NewAssignmentResource(httpClient),
		Webhooks:        resources.NewWebhookResource(httpClient, opts.AccountID),
		Templates:       resources.NewTemplateResource(httpClient, opts.AccountID),
		Fields:          resources.NewFieldResource(httpClient, opts.AccountID),
		Authentication:  resources.NewAuthenticationResource(httpClient),
		PublicDocuments: resources.NewPublicDocumentResource(httpClient),
	}, nil
}

// ErrNoSigners is returned by UploadAndRequestSignatures when no signers are provided.
var ErrNoSigners = errors.New("assinafy: at least one signer is required")

// UploadAndRequestSignatures is a convenience helper that uploads a file,
// creates each signer, and creates a virtual assignment in one call.
//
// If accountID is empty, the Client's default AccountID is used.
func (c *Client) UploadAndRequestSignatures(
	ctx context.Context,
	fileContent []byte,
	fileName string,
	signers []models.UploadAndRequestSignaturesSigner,
	message string,
	expiresAt *string,
	accountID string,
) (*models.UploadAndRequestSignaturesResult, error) {
	if len(signers) == 0 {
		return nil, ErrNoSigners
	}
	if accountID == "" {
		accountID = c.accountID
	}

	doc, err := c.Documents.Upload(ctx, accountID, fileContent, fileName, nil)
	if err != nil {
		return nil, err
	}

	signerIDs := make([]string, len(signers))
	signerRefs := make([]models.SignerReference, len(signers))
	for i, s := range signers {
		req := &models.CreateSignerRequest{FullName: s.Name}
		if s.Email != "" {
			req.Email = &s.Email
		}
		if s.WhatsAppPhoneNumber != "" {
			req.WhatsAppPhoneNumber = &s.WhatsAppPhoneNumber
		}
		signer, err := c.Signers.Create(ctx, accountID, req)
		if err != nil {
			return nil, err
		}
		signerIDs[i] = signer.ID
		signerRefs[i] = models.SignerReference{ID: signer.ID}
	}

	body := &models.CreateAssignmentRequest{
		Method:  models.MethodVirtual,
		Signers: signerRefs,
	}
	if message != "" {
		body.Message = &message
	}
	if expiresAt != nil {
		body.ExpiresAt = expiresAt
	}

	assignment, err := c.Assignments.Create(ctx, doc.ID, body)
	if err != nil {
		return nil, err
	}

	return &models.UploadAndRequestSignaturesResult{
		Document:   doc,
		Assignment: assignment,
		SignerIDs:  signerIDs,
	}, nil
}
