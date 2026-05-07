package models

type DocumentStatus string

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
	StatusRejectedByUser     DocumentStatus = "rejected_by_user"
	StatusFailed             DocumentStatus = "failed"
)

type Document struct {
	Resource      string             `json:"resource,omitempty"`
	ID            string             `json:"id"`
	AccountID     string             `json:"account_id,omitempty"`
	TemplateID    *string            `json:"template_id,omitempty"`
	Name          string             `json:"name"`
	Status        DocumentStatus     `json:"status"`
	Artifacts     *DocumentArtifacts `json:"artifacts,omitempty"`
	IsClosed      bool               `json:"is_closed"`
	SigningURL    *string            `json:"signing_url,omitempty"`
	DeclineReason *string            `json:"decline_reason,omitempty"`
	DeclinedBy    *Signer            `json:"declined_by,omitempty"`
	CreatedAt     string             `json:"created_at"`
	UpdatedAt     string             `json:"updated_at"`
	Assignment    *Assignment        `json:"assignment,omitempty"`
	Pages         []DocumentPage     `json:"pages,omitempty"`
}

type DocumentArtifacts struct {
	Original     *string `json:"original,omitempty"`
	Certificated *string `json:"certificated,omitempty"`
}

type DocumentPage struct {
	ID          string `json:"id"`
	Number      int    `json:"number"`
	Height      int    `json:"height"`
	Width       int    `json:"width"`
	DownloadURL string `json:"download_url"`
}

type DocumentListItem struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Status    DocumentStatus `json:"status"`
	CreatedAt string         `json:"created_at"`
}

type DocumentUploadResponse struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Status     DocumentStatus     `json:"status"`
	Assignment *Assignment        `json:"assignment,omitempty"`
	Artifacts  *DocumentArtifacts `json:"artifacts,omitempty"`
	Pages      []DocumentPage     `json:"pages,omitempty"`
	CreatedAt  string             `json:"created_at"`
	UpdatedAt  string             `json:"updated_at"`
	IsClosed   bool               `json:"is_closed"`
}

type DocumentActivity struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	Actor       *Actor `json:"actor,omitempty"`
}

type Actor struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email,omitempty"`
	IsSigner bool   `json:"is_signer,omitempty"`
}

type CreateDocumentFromTemplateOptions struct {
	Name      string           `json:"name"`
	Message   string           `json:"message,omitempty"`
	ExpiresAt string           `json:"expires_at,omitempty"`
	Signers   []TemplateSigner `json:"signers"`
}

type TemplateSigner struct {
	RoleID              string   `json:"role_id"`
	ID                  string   `json:"id"`
	VerificationMethod  string   `json:"verification_method"`
	NotificationMethods []string `json:"notification_methods,omitempty"`
}

type UploadAndRequestSignaturesSigner struct {
	Name                string `json:"name"`
	Email               string `json:"email"`
	WhatsAppPhoneNumber string `json:"whatsapp_phone_number,omitempty"`
}

type UploadAndRequestSignaturesResult struct {
	Document   *DocumentUploadResponse
	Assignment *Assignment
	SignerIDs  []string
}

type SignDocumentOptions struct {
	HasAcceptedTerms bool `json:"has_accepted_terms"`
}

type VerifyDocumentResult struct {
	Valid        bool   `json:"valid"`
	DocumentID   string `json:"document_id,omitempty"`
	DocumentName string `json:"document_name,omitempty"`
	CertifiedAt  string `json:"certified_at,omitempty"`
	SignerCount  int    `json:"signer_count,omitempty"`
	CompletedAt  string `json:"completed_at,omitempty"`
}
