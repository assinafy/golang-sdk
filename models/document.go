package models

// DocumentStatus enumerates the lifecycle states documented at
// https://api.assinafy.com.br/v1/docs#statuses.
type DocumentStatus string

// Documented values for DocumentStatus.
const (
	StatusUploading          DocumentStatus = "uploading"
	StatusUploaded           DocumentStatus = "uploaded"
	StatusMetadataProcessing DocumentStatus = "metadata_processing"
	StatusMetadataReady      DocumentStatus = "metadata_ready"
	StatusExpired            DocumentStatus = "expired"
	StatusCertificating      DocumentStatus = "certificating"
	StatusCertificated       DocumentStatus = "certificated"
	StatusRejectedBySigner   DocumentStatus = "rejected_by_signer"
	StatusPendingSignature   DocumentStatus = "pending_signature"
	StatusPendingSignatures  DocumentStatus = "pending_signatures"
	StatusPartiallySigned    DocumentStatus = "partially_signed"
	StatusReady              DocumentStatus = "ready"
	StatusRejectedByUser     DocumentStatus = "rejected_by_user"
	StatusFailed             DocumentStatus = "failed"
)

// Document is the canonical document object returned by document endpoints.
type Document struct {
	Resource         string             `json:"resource,omitempty"`
	ID               string             `json:"id"`
	AccountID        string             `json:"account_id,omitempty"`
	TemplateID       *string            `json:"template_id,omitempty"`
	Name             string             `json:"name"`
	Status           DocumentStatus     `json:"status"`
	Artifacts        *DocumentArtifacts `json:"artifacts,omitempty"`
	IsClosed         bool               `json:"is_closed,omitempty"`
	SigningURL       *string            `json:"signing_url,omitempty"`
	DeclineReason    *string            `json:"decline_reason,omitempty"`
	DeclinedBy       *Signer            `json:"declined_by,omitempty"`
	Tags             []Tag              `json:"tags,omitempty"`
	CreatedAt        Timestamp          `json:"created_at,omitempty"`
	UpdatedAt        Timestamp          `json:"updated_at,omitempty"`
	Assignment       *Assignment        `json:"assignment,omitempty"`
	Pages            []DocumentPage     `json:"pages,omitempty"`
	Activities       []DocumentActivity `json:"activities,omitempty"`
	CurrentSigner    *Signer            `json:"current_signer,omitempty"`
	DownloadURL      *string            `json:"download_url,omitempty"`
	DownloadFinalURL *string            `json:"download_final_url,omitempty"`
}

// DocumentArtifacts lists the URLs exposed by the API for download endpoints.
type DocumentArtifacts struct {
	Original        *string `json:"original,omitempty"`
	Certificated    *string `json:"certificated,omitempty"`
	CertificatePage *string `json:"certificate-page,omitempty"`
	Bundle          *string `json:"bundle,omitempty"`
	Thumbnail       *string `json:"thumbnail,omitempty"`
}

// DocumentPage describes a single page rendered from the source PDF.
type DocumentPage struct {
	ID          string `json:"id"`
	Number      int    `json:"number"`
	Height      int    `json:"height"`
	Width       int    `json:"width"`
	DownloadURL string `json:"download_url"`
}

// DocumentActivity represents a single audit-trail entry on a document.
type DocumentActivity struct {
	ID        int            `json:"id"`
	Event     string         `json:"event"`
	Message   string         `json:"message"`
	Payload   Payload        `json:"payload,omitempty"`
	Origin    *RequestOrigin `json:"origin,omitempty"`
	CreatedAt Timestamp      `json:"created_at"`
}

// RequestOrigin captures the network context recorded for an activity.
type RequestOrigin struct {
	IP        string `json:"ip"`
	UserAgent string `json:"user-agent"`
}

// CreateDocumentFromTemplateOptions is the body for
// POST /accounts/{id}/templates/{tid}/documents.
type CreateDocumentFromTemplateOptions struct {
	Name         string                `json:"name,omitempty"`
	Message      string                `json:"message,omitempty"`
	EditorFields []TemplateEditorField `json:"editor_fields,omitempty"`
	ExpiresAt    string                `json:"expires_at,omitempty"`
	Signers      []TemplateSigner      `json:"signers"`
}

// TemplateEditorField is a single value bound to a template editor field.
type TemplateEditorField struct {
	FieldID string `json:"field_id"`
	Value   any    `json:"value"`
}

// TemplateSigner pairs a real signer with a template role.
type TemplateSigner struct {
	RoleID              string   `json:"role_id"`
	ID                  string   `json:"id,omitempty"`
	VerificationMethod  string   `json:"verification_method,omitempty"`
	NotificationMethods []string `json:"notification_methods,omitempty"`
}

// UploadAndRequestSignaturesSigner is a high-level signer payload accepted by
// the Client.UploadAndRequestSignatures convenience helper.
type UploadAndRequestSignaturesSigner struct {
	Name                string
	Email               string
	WhatsAppPhoneNumber string
}

// UploadAndRequestSignaturesResult is returned by Client.UploadAndRequestSignatures.
type UploadAndRequestSignaturesResult struct {
	Document   *Document
	Assignment *Assignment
	SignerIDs  []string
}

// VerifyDocumentResult is the response from GET /documents/{hash}/verify.
type VerifyDocumentResult struct {
	Hash           string     `json:"hash"`
	ID             *string    `json:"id,omitempty"`
	Status         *string    `json:"status,omitempty"`
	PageCount      *string    `json:"page_count,omitempty"`
	SignerCount    *string    `json:"signer_count,omitempty"`
	CompletedCount *int       `json:"completed_count,omitempty"`
	CompletedAt    *Timestamp `json:"completed_at,omitempty"`
	VerifiedAt     Timestamp  `json:"verified_at"`
	IsValid        bool       `json:"is_valid"`
	Message        string     `json:"message"`
}

// DocumentStatusInfo describes a single status code returned by GET /documents/statuses.
type DocumentStatusInfo struct {
	Code      string `json:"code"`
	Deletable bool   `json:"deletable"`
}

// PublicDocumentInfo is the limited data returned from GET /public/documents/{id}.
type PublicDocumentInfo struct {
	Resource  string `json:"resource,omitempty"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	PageCount string `json:"page_count"`
	CreatedBy string `json:"created_by"`
}

// SendDocumentTokenRequest is the body for PUT /public/documents/{id}/send-token.
type SendDocumentTokenRequest struct {
	Recipient string `json:"recipient"`
	Channel   string `json:"channel"`
}

// SendDocumentTokenResult is the response from PUT /public/documents/{id}/send-token.
type SendDocumentTokenResult struct {
	Document  PublicDocumentInfo `json:"document"`
	Channel   string             `json:"channel"`
	Recipient string             `json:"recipient"`
}
