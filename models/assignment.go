package models

type AssignmentMethod string

const (
	MethodVirtual AssignmentMethod = "virtual"
	MethodCollect AssignmentMethod = "collect"
)

type Assignment struct {
	Resource    string           `json:"resource,omitempty"`
	ID          string           `json:"id"`
	DocumentID  string           `json:"document_id"`
	Method      AssignmentMethod `json:"method"`
	Status      string           `json:"status"`
	ExpiresAt   *string          `json:"expires_at,omitempty"`
	Message     *string          `json:"message,omitempty"`
	CompletedAt *string          `json:"completed_at,omitempty"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
}

type CreateAssignmentRequest struct {
	Method        AssignmentMethod  `json:"method"`
	SignerIDs     []string          `json:"signerIds,omitempty"`
	Signers       []SignerReference `json:"signers,omitempty"`
	Message       *string           `json:"message,omitempty"`
	ExpiresAt     *string           `json:"expires_at,omitempty"`
	CopyReceivers []string          `json:"copy_receivers,omitempty"`
}

type EstimateCostResult struct {
	TotalCost   float64 `json:"total_cost"`
	Description string  `json:"description"`
}

type AssignmentSummary struct {
	DocumentID   string `json:"document_id"`
	AssignmentID string `json:"assignment_id"`
	Status       string `json:"status"`
}

type SigningProgress struct {
	TotalSigners  int `json:"total_signers"`
	SignedCount   int `json:"signed_count"`
	PendingCount  int `json:"pending_count"`
	DeclinedCount int `json:"declined_count"`
}
