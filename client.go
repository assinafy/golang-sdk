package assinafy

import (
	"context"
	"time"

	"github.com/assinafy/assinafy-go/internal"
	"github.com/assinafy/assinafy-go/models"
	"github.com/assinafy/assinafy-go/resources"
)

const (
	defaultBaseURL = "https://api.assinafy.com.br/v1"
	defaultTimeout = 30 * time.Second
)

type Client struct {
	httpClient    *internal.HTTPClient
	accountID     string
	baseURL       string
	timeout       time.Duration
	webhookSecret string

	Documents   *resources.DocumentResource
	Signers     *resources.SignerResource
	Workspaces  *resources.WorkspaceResource
	Assignments *resources.AssignmentResource
	Webhooks    *resources.WebhookResource
	Templates   *resources.TemplateResource
}

type ClientOptions struct {
	APIKey        string
	Token         string
	AccountID     string
	BaseURL       string
	WebhookSecret string
	Timeout       time.Duration
}

func NewClient(opts ClientOptions) (*Client, error) {
	if opts.APIKey == "" && opts.Token == "" {
		return nil, &ValidationError{Message: "either API key or token is required"}
	}

	if opts.BaseURL == "" {
		opts.BaseURL = defaultBaseURL
	}

	if opts.Timeout == 0 {
		opts.Timeout = defaultTimeout
	}

	httpClient := internal.NewHTTPClient(opts.BaseURL, opts.APIKey, opts.Token, opts.Timeout)

	client := &Client{
		httpClient:    httpClient,
		accountID:     opts.AccountID,
		baseURL:       opts.BaseURL,
		timeout:       opts.Timeout,
		webhookSecret: opts.WebhookSecret,
	}

	client.Documents = resources.NewDocumentResource(httpClient, opts.AccountID)
	client.Signers = resources.NewSignerResource(httpClient, opts.AccountID)
	client.Workspaces = resources.NewWorkspaceResource(httpClient)
	client.Assignments = resources.NewAssignmentResource(httpClient, opts.AccountID)
	client.Webhooks = resources.NewWebhookResource(httpClient, opts.AccountID)
	client.Templates = resources.NewTemplateResource(httpClient, opts.AccountID)

	return client, nil
}

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
		return nil, &ValidationError{Message: "at least one signer is required"}
	}

	if accountID == "" {
		accountID = c.accountID
	}

	doc, err := c.Documents.Upload(ctx, accountID, fileContent, fileName, nil)
	if err != nil {
		return nil, err
	}

	signerIDs := make([]string, len(signers))
	for i, s := range signers {
		req := &models.CreateSignerRequest{
			FullName: s.Name,
		}
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
	}

	assignmentReq := &models.CreateAssignmentRequest{
		Method:    models.MethodVirtual,
		SignerIDs: signerIDs,
	}
	if message != "" {
		assignmentReq.Message = &message
	}
	if expiresAt != nil {
		assignmentReq.ExpiresAt = expiresAt
	}

	assignment, err := c.Assignments.Create(ctx, doc.ID, assignmentReq)
	if err != nil {
		return nil, err
	}

	return &models.UploadAndRequestSignaturesResult{
		Document:   doc,
		Assignment: assignment,
		SignerIDs:  signerIDs,
	}, nil
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
