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
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
	"github.com/assinafy/golang-sdk/resources"
)

var (
	// ErrInvalidBaseURL is returned when ClientOptions.BaseURL is not an
	// absolute HTTP(S) URL suitable for API requests.
	ErrInvalidBaseURL = errors.New("assinafy: BaseURL must be an absolute HTTP(S) URL without credentials, query, or fragment")
	// ErrNoSigners is returned by UploadAndRequestSignatures when no signers are provided.
	ErrNoSigners = errors.New("assinafy: at least one signer is required")
	// ErrInvalidSigner is returned by UploadAndRequestSignatures when a signer
	// has no name, no contact, an invalid email, or a non-E.164 WhatsApp number.
	ErrInvalidSigner = errors.New("assinafy: signer must have a name and an email or WhatsApp contact")
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

	// Accounts provides authenticated workspace profile, theme, logo, and stats operations.
	Accounts *resources.AccountResource
	// Documents provides authenticated document upload, lifecycle, artifact, and tag operations.
	Documents *resources.DocumentResource
	// Signers provides account-scoped and signer-access-code signer operations.
	Signers *resources.SignerResource
	// SignerDocuments provides signer-access-code document and batch-signing operations.
	SignerDocuments *resources.SignerDocumentResource
	// Assignments provides authenticated assignment management and signer-facing signing operations.
	Assignments *resources.AssignmentResource
	// Webhooks provides authenticated subscription and delivery-history operations.
	Webhooks *resources.WebhookResource
	// Templates provides authenticated account-template read operations.
	Templates *resources.TemplateResource
	// Tags provides authenticated workspace-tag operations.
	Tags *resources.TagResource
	// Fields provides account field-definition and signer-facing validation operations.
	Fields *resources.FieldResource
	// Users provides authenticated current-user profile, statistics, and preference operations.
	Users *resources.UserResource
	// Authentication provides login, password, social-login, and API-key operations.
	Authentication *resources.AuthenticationResource
	// PublicDocuments provides unauthenticated public-document lookup and token delivery.
	PublicDocuments *resources.PublicDocumentResource
}

// ClientOptions configures Client construction.
type ClientOptions struct {
	// APIKey is the permanent credential sent in X-Api-Key. It takes precedence
	// over Token when both are set; leave both empty for public or signer flows.
	APIKey string
	// Token is a JWT access token sent as Authorization: Bearer when APIKey is empty.
	Token string
	// AccountID is the optional default account identifier. Account-scoped methods
	// use it when their accountID argument is empty; an empty result is rejected locally.
	AccountID string
	// BaseURL overrides the API root. It must be an absolute HTTP(S) URL without
	// credentials, query, or fragment and defaults to DefaultBaseURL. Use
	// SandboxBaseURL for sandbox calls.
	BaseURL string
	// Timeout is the per-request HTTP timeout. Zero or negative values use 30 seconds.
	Timeout time.Duration
}

// NewClient validates opts and builds a concurrency-safe Client without making
// a network request. Credentials are optional for public document, login, and
// signer-access-code endpoints. It returns ErrInvalidBaseURL for an invalid API
// root; authentication and account selection are otherwise validated per call.
func NewClient(opts ClientOptions) (*Client, error) {
	if opts.BaseURL == "" {
		opts.BaseURL = DefaultBaseURL
	}
	baseURL, err := url.Parse(opts.BaseURL)
	if err != nil || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.Host == "" ||
		baseURL.User != nil || baseURL.RawQuery != "" || baseURL.ForceQuery || strings.Contains(opts.BaseURL, "#") {
		return nil, ErrInvalidBaseURL
	}
	if opts.Timeout <= 0 {
		opts.Timeout = defaultTimeout
	}

	httpClient := internal.NewHTTPClient(opts.BaseURL, opts.APIKey, opts.Token, opts.Timeout)

	return &Client{
		accountID:       opts.AccountID,
		Accounts:        resources.NewAccountResource(httpClient, opts.AccountID),
		Documents:       resources.NewDocumentResource(httpClient, opts.AccountID),
		Signers:         resources.NewSignerResource(httpClient, opts.AccountID),
		SignerDocuments: resources.NewSignerDocumentResource(httpClient),
		Assignments:     resources.NewAssignmentResource(httpClient),
		Webhooks:        resources.NewWebhookResource(httpClient, opts.AccountID),
		Templates:       resources.NewTemplateResource(httpClient, opts.AccountID),
		Tags:            resources.NewTagResource(httpClient, opts.AccountID),
		Fields:          resources.NewFieldResource(httpClient, opts.AccountID),
		Users:           resources.NewUserResource(httpClient),
		Authentication:  resources.NewAuthenticationResource(httpClient),
		PublicDocuments: resources.NewPublicDocumentResource(httpClient),
	}, nil
}

// UploadAndRequestSignatures uploads fileContent, creates each requested signer,
// and creates a virtual assignment, returning all three response groups. It
// requires client API-key or bearer authentication; an empty accountID uses the
// Client default. expiresAt is an optional ISO-8601 date/time.
//
// The calls are sequential and are not transactional: a later failure does not
// remove the document or signers already created. It returns ErrNoSigners or
// ErrInvalidSigner before making a request. After upload, an error is accompanied
// by a partial result containing the document and any created signer IDs so the
// caller can clean them up.
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
	for i, signer := range signers {
		name := strings.TrimSpace(signer.Name)
		email := strings.TrimSpace(signer.Email)
		phone := strings.TrimSpace(signer.WhatsAppPhoneNumber)
		if name == "" {
			return nil, fmt.Errorf("%w: signer %d has no name", ErrInvalidSigner, i+1)
		}
		if email == "" && phone == "" {
			return nil, fmt.Errorf("%w: signer %d has no email or WhatsApp contact", ErrInvalidSigner, i+1)
		}
		if email != "" {
			address, err := mail.ParseAddress(email)
			if err != nil || address.Address != email {
				return nil, fmt.Errorf("%w: signer %d has an invalid email", ErrInvalidSigner, i+1)
			}
		}
		if phone != "" && !isE164(phone) {
			return nil, fmt.Errorf("%w: signer %d has a non-E.164 WhatsApp number", ErrInvalidSigner, i+1)
		}
	}
	var normalizedExpiresAt *string
	if expiresAt != nil {
		value := strings.TrimSpace(*expiresAt)
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return nil, fmt.Errorf("%w: expiresAt must be an RFC 3339 date-time", sdkerrors.ErrInvalidInput)
		}
		normalizedExpiresAt = &value
	}
	if accountID == "" {
		accountID = c.accountID
	}

	doc, err := c.Documents.Upload(ctx, accountID, fileContent, fileName, nil)
	if err != nil {
		return nil, err
	}

	result := &models.UploadAndRequestSignaturesResult{
		Document:  doc,
		SignerIDs: make([]string, 0, len(signers)),
	}
	signerRefs := make([]models.SignerReference, 0, len(signers))
	for _, s := range signers {
		name := strings.TrimSpace(s.Name)
		email := strings.TrimSpace(s.Email)
		phone := strings.TrimSpace(s.WhatsAppPhoneNumber)
		req := &models.CreateSignerRequest{FullName: name}
		method := "Whatsapp"
		if email != "" {
			req.Email = &email
			method = "Email"
		}
		if phone != "" {
			req.WhatsAppPhoneNumber = &phone
		}
		signer, err := c.Signers.Create(ctx, accountID, req)
		if err != nil {
			return result, err
		}
		result.SignerIDs = append(result.SignerIDs, signer.ID)
		signerRefs = append(signerRefs, models.SignerReference{
			ID:                  signer.ID,
			VerificationMethod:  method,
			NotificationMethods: []string{method},
		})
	}

	body := &models.CreateAssignmentRequest{
		Method:  models.MethodVirtual,
		Signers: signerRefs,
	}
	if message != "" {
		body.Message = &message
	}
	if normalizedExpiresAt != nil {
		body.ExpiresAt = normalizedExpiresAt
	}

	assignment, err := c.Assignments.Create(ctx, doc.ID, body)
	if err != nil {
		return result, err
	}
	result.Assignment = assignment
	return result, nil
}

func isE164(phone string) bool {
	if len(phone) < 2 || len(phone) > 16 || phone[0] != '+' || phone[1] == '0' {
		return false
	}
	for _, digit := range phone[1:] {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}
