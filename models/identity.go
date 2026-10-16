package models

// IdentityVerificationStatus is the signer-facing identity-document state from
// GET /signers/identity-verification.
type IdentityVerificationStatus struct {
	// CanStart is true before a session exists or after its capture link expires.
	CanStart bool `json:"can_start"`
	// Status is Pending, InReview, Approved, Declined, Expired, Abandoned, or null.
	Status *string `json:"status"`
	// URL is the nullable capture bearer link while Pending. Open it unchanged,
	// including its fragment; never log it or persist it in browser storage.
	URL *string `json:"url"`
	// ExpiresAt is when capture expires, not the review or processing deadline.
	ExpiresAt *Timestamp `json:"expires_at"`
	// CompletedAt is the nullable verification completion date-time.
	CompletedAt *Timestamp `json:"completed_at"`
}

// StartIdentityVerificationRequest is the body for
// POST /signers/identity-verification. Obtain the signer's explicit consent.
type StartIdentityVerificationRequest struct {
	// Consent must be true and records consent to biometric processing.
	Consent bool `json:"consent"`
}

// IdentityVerificationSession is the data returned by
// POST /signers/identity-verification.
type IdentityVerificationSession struct {
	// ID identifies the provider session.
	ID string `json:"id"`
	// URL is the capture bearer link. Open it unchanged, including its fragment;
	// never log it or persist it in browser storage.
	URL string `json:"url"`
	// ExpiresAt is the nullable capture-link expiration date-time.
	ExpiresAt *Timestamp `json:"expires_at"`
}

// IdentityReviewDecisionApproved approves an identity verification under review.
const IdentityReviewDecisionApproved = "Approved"

// IdentityReviewDecisionRejected rejects an identity verification under review
// and cancels the document, notifying signers and copy receivers.
const IdentityReviewDecisionRejected = "Rejected"

// DecideIdentityReviewRequest is the body for
// POST /documents/{id}/assignments/{aid}/signers/{sid}/identity-verification/review.
type DecideIdentityReviewRequest struct {
	// Decision is required: IdentityReviewDecisionApproved or IdentityReviewDecisionRejected.
	Decision string `json:"decision"`
	// Note is optional, at most 2,000 characters, and is never shown to signers.
	Note *string `json:"note,omitempty"`
}

// SignerIdentityReview is the account review of a signer's identity verification.
// Evidence contains no images or biometrics. Images can be fetched only while
// the status is InReview; provider-approved verifications have no evidence here.
type SignerIdentityReview struct {
	// ID identifies the verification.
	ID string `json:"id"`
	// Status is Pending, InReview, Approved, Declined, Expired, or Abandoned.
	Status string `json:"status"`
	// Provider identifies the verification service, currently AssinafyVerify.
	Provider string `json:"provider"`
	// VerificationMethod is IdDocumentFaceMatchBasic.
	VerificationMethod string `json:"verification_method"`
	// ReasonCodes explains why the provider requested review, in provider order.
	ReasonCodes []IdentityReviewReason `json:"reason_codes"`
	// EvidenceAvailable reports whether Assinafy has fetched an evidence summary.
	EvidenceAvailable bool `json:"evidence_available"`
	// Expected is the nullable identity registered for the signer.
	Expected *IdentityReviewExpected `json:"expected"`
	// Document is the nullable identity read from the government document.
	Document *IdentityReviewDocument `json:"document"`
	// Matches contains nullable comparisons of the registered and captured identity.
	Matches *IdentityReviewMatches `json:"matches"`
	// Checks lists the provider's checks and their results.
	Checks []IdentityReviewCheck `json:"checks"`
	// Images lists available document_front, document_back, document_pdf, or face images.
	Images []string `json:"images"`
	// RulesetVersion is the nullable provider ruleset identifier.
	RulesetVersion *string `json:"ruleset_version"`
	// EvidenceSHA256 is the nullable SHA-256 digest of the evidence summary.
	EvidenceSHA256 *string `json:"evidence_sha256"`
	// ReviewedBy identifies the account user who decided the review, or is nil.
	ReviewedBy *IdentityReviewer `json:"reviewed_by"`
	// ReviewedAt is the nullable decision date-time.
	ReviewedAt *Timestamp `json:"reviewed_at"`
	// ReviewNote is the nullable private reviewer note.
	ReviewNote *string `json:"review_note"`
	// CompletedAt is the nullable verification completion date-time.
	CompletedAt *Timestamp `json:"completed_at"`
}

// IdentityReviewReason describes a provider reason for requiring account review.
type IdentityReviewReason struct {
	// Code is the provider's machine-readable reason.
	Code string `json:"code"`
	// Description explains the reason; unknown codes use the code itself.
	Description string `json:"description"`
}

// IdentityReviewExpected is the identity registered for the signer.
type IdentityReviewExpected struct {
	// Name is the nullable registered name.
	Name *string `json:"name"`
	// CPF is the nullable registered government identifier.
	CPF *string `json:"cpf"`
}

// IdentityReviewDocument is the identity read from the captured document.
type IdentityReviewDocument struct {
	// Type is the nullable document type, such as CNH.
	Type *string `json:"type"`
	// Name is the nullable name read from the document.
	Name *string `json:"name"`
	// CPF is the nullable government identifier read from the document.
	CPF *string `json:"cpf"`
}

// IdentityReviewMatches compares the registered and captured identity.
type IdentityReviewMatches struct {
	// CPF reports whether the identifiers match, or is nil when unavailable.
	CPF *bool `json:"cpf"`
	// Name is the nullable name similarity from zero to one.
	Name *float64 `json:"name"`
	// BirthDate reports whether birth dates match, or is nil when unavailable.
	BirthDate *bool `json:"birth_date"`
}

// IdentityReviewCheck is one provider check in an identity evidence summary.
type IdentityReviewCheck struct {
	// Name identifies the check.
	Name string `json:"name"`
	// Result is pass, review, skipped, or a provider failure result.
	Result string `json:"result"`
	// Score is the nullable provider score.
	Score *float64 `json:"score"`
	// Reasons contains the machine-readable reason codes for this result.
	Reasons []string `json:"reasons"`
}

// IdentityReviewer identifies the account user who decided a review.
type IdentityReviewer struct {
	// ID is the account user's identifier.
	ID string `json:"id"`
	// Name is the account user's display name.
	Name string `json:"name"`
}
