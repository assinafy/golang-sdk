package models

import "encoding/json"

// Nullable is a nullable request field. A nil *Nullable omits the field, an
// empty Nullable sends null, and NullableValue sends a value, including false
// or zero. Construct fields in Go; this encoder does not support decoding
// persisted request JSON. Keep encoded JSON bytes when persisting a request.
type Nullable[T any] struct {
	// Value is the value to send; nil sends JSON null.
	Value *T
}

// NullableValue constructs a request field containing value.
func NullableValue[T any](value T) *Nullable[T] { return &Nullable[T]{Value: &value} }

// MarshalJSON encodes the value or an explicit JSON null.
func (n Nullable[T]) MarshalJSON() ([]byte, error) { return json.Marshal(n.Value) }

// ContractStatus is the lifecycle state of tracked contract terms.
type ContractStatus string

const (
	// ContractStatusDraft identifies terms that are not yet in force.
	ContractStatusDraft ContractStatus = "draft"
	// ContractStatusActive identifies terms in force.
	ContractStatusActive ContractStatus = "active"
	// ContractStatusExpired identifies terms whose end date passed without renewal.
	ContractStatusExpired ContractStatus = "expired"
	// ContractStatusTerminated identifies a closed contract.
	ContractStatusTerminated ContractStatus = "terminated"
)

// ContractRelationType describes how a signed instrument changes its parent.
type ContractRelationType string

const (
	// ContractRelationAmendment changes the permitted terms of a contract.
	ContractRelationAmendment ContractRelationType = "amendment"
	// ContractRelationRenewal changes only end_date and value.
	ContractRelationRenewal ContractRelationType = "renewal"
	// ContractRelationTermination changes only the effective end_date.
	ContractRelationTermination ContractRelationType = "termination"
)

// ContractExternalReference identifies a contract in an external system.
type ContractExternalReference struct {
	// System is required, at most 64 letters, digits, underscores, hyphens or
	// dots; the API stores it in lower case.
	System string `json:"system"`
	// ID is required, at most 255 characters, without emoji or 4-byte characters.
	ID string `json:"id"`
}

// DocumentContract is the tracked contract returned by list, read, update,
// related, renew and terminate operations. Monetary values use json.Number to
// retain the API's decimal representation without binary floating-point rounding.
type DocumentContract struct {
	// Resource is document_contract.
	Resource string `json:"resource"`
	// DocumentID identifies the document these terms belong to.
	DocumentID string `json:"document_id"`
	// Status is the contract lifecycle state, separate from document status.
	Status ContractStatus `json:"status"`
	// CounterpartyName is the nullable counterparty display name.
	CounterpartyName *string `json:"counterparty_name"`
	// CounterpartyTaxpayerID is the nullable CPF or CNPJ, digits only.
	CounterpartyTaxpayerID *string `json:"counterparty_taxpayer_id"`
	// Value is the nullable monetary amount, with at most two decimal places.
	Value *json.Number `json:"value"`
	// Currency is the nullable three-letter currency code.
	Currency *string `json:"currency"`
	// StartDate is the nullable YYYY-MM-DD start date.
	StartDate *string `json:"start_date"`
	// EndDate is the nullable YYYY-MM-DD end date.
	EndDate *string `json:"end_date"`
	// AutoRenew enables automatic renewal after the end date.
	AutoRenew bool `json:"auto_renew"`
	// RenewalTermMonths is the nullable renewal interval; unset means 12 months.
	RenewalTermMonths *int `json:"renewal_term_months"`
	// NoticePeriodDays is the nullable number of days before the renewal notice.
	NoticePeriodDays *int `json:"notice_period_days"`
	// Term starts at one and increases on each renewal or changed end date.
	Term int `json:"term"`
	// OwnerID is the nullable workspace member who owns the contract.
	OwnerID *string `json:"owner_id"`
	// Metadata contains up to 50 text values, or null when none are set.
	Metadata map[string]string `json:"metadata"`
	// ExternalReferences lists external IDs, ordered by system and ID.
	ExternalReferences []ContractExternalReference `json:"external_references"`
	// Document is present only when the list query requests expand=document.
	Document *Document `json:"document,omitempty"`
	// Amendments is included on single-contract and related-family reads.
	Amendments []DocumentContractAmendment `json:"amendments,omitzero"`
	// CreatedAt is the creation timestamp.
	CreatedAt Timestamp `json:"created_at"`
	// UpdatedAt is the concurrency precondition to send with a subsequent write.
	UpdatedAt Timestamp `json:"updated_at"`
}

// ContractChange is a parent's field value before and after an instrument.
// RawMessage preserves decimals, nulls, objects and booleans without conversion.
type ContractChange struct {
	// From is the original field value.
	From json.RawMessage `json:"from"`
	// To is the resulting field value.
	To json.RawMessage `json:"to"`
}

// DocumentContractAmendment links an instrument to the root contract. Its
// delta applies only after certification, in the API's recorded signing order.
type DocumentContractAmendment struct {
	// Resource is document_contract_amendment when included.
	Resource string `json:"resource,omitempty"`
	// DocumentID identifies the instrument.
	DocumentID string `json:"document_id"`
	// DocumentName is the instrument's display name.
	DocumentName string `json:"document_name"`
	// ParentDocumentID identifies the root contract document.
	ParentDocumentID string `json:"parent_document_id"`
	// RelationType is amendment, renewal or termination.
	RelationType ContractRelationType `json:"relation_type"`
	// Number is the applied signing-order position; null while pending.
	Number *int `json:"number"`
	// Status is pending or applied.
	Status string `json:"status"`
	// DocumentStatus is the instrument's own document lifecycle state.
	DocumentStatus DocumentStatus `json:"document_status"`
	// ApplyError explains a blocked application, or is null.
	ApplyError *string `json:"apply_error"`
	// Currency is the nullable currency of amounts in Terms and Changes.
	Currency *string `json:"currency"`
	// Terms contains only changed fields, or null when there are none.
	Terms map[string]json.RawMessage `json:"terms"`
	// Changes previews parent changes while pending and records them once applied.
	Changes map[string]ContractChange `json:"changes"`
	// AppliedAt is null until the delta is applied.
	AppliedAt *Timestamp `json:"applied_at"`
	// CreatedAt is the link creation timestamp.
	CreatedAt Timestamp `json:"created_at"`
	// UpdatedAt is the link's concurrency precondition for a subsequent write.
	UpdatedAt Timestamp `json:"updated_at"`
}

// ContractResult holds either root contract terms or an amendment link, as
// selected by the response's resource discriminator. Exactly one field is set
// when a non-null API payload is decoded.
type ContractResult struct {
	// Contract is set for a document_contract payload.
	Contract *DocumentContract
	// Amendment is set for a document_contract_amendment payload.
	Amendment *DocumentContractAmendment
}

// UnmarshalJSON decodes a contract or amendment response without flattening its fields.
func (r *ContractResult) UnmarshalJSON(data []byte) error {
	var discriminator struct {
		Resource string `json:"resource"`
	}
	if err := json.Unmarshal(data, &discriminator); err != nil {
		return err
	}
	var result ContractResult
	if discriminator.Resource == "document_contract_amendment" {
		if err := json.Unmarshal(data, &result.Amendment); err != nil {
			return err
		}
	} else if err := json.Unmarshal(data, &result.Contract); err != nil {
		return err
	}
	*r = result
	return nil
}

// MarshalJSON encodes the populated variant in the API's flat response shape.
func (r ContractResult) MarshalJSON() ([]byte, error) {
	if r.Amendment != nil {
		return json.Marshal(r.Amendment)
	}
	return json.Marshal(r.Contract)
}

// SetDocumentContractRequest sets only the supplied terms, or links an amendment.
// Nil fields leave stored values unchanged. An empty Nullable sends null to clear
// a root term; on a pending amendment it removes that term from the delta.
// The API validates state transitions, dates, membership and plan entitlement.
type SetDocumentContractRequest struct {
	// Status accepts any ContractStatus at creation; subsequent edits permit
	// draft or active only. Active cannot return to draft.
	Status *ContractStatus `json:"status,omitempty"`
	// CounterpartyName accepts up to 255 characters without HTML.
	CounterpartyName *Nullable[string] `json:"counterparty_name,omitempty"`
	// CounterpartyTaxpayerID accepts a valid formatted or unformatted CPF/CNPJ.
	CounterpartyTaxpayerID *Nullable[string] `json:"counterparty_taxpayer_id,omitempty"`
	// Value accepts 0 through 9999999999999.99 with at most two decimal places.
	Value *Nullable[json.Number] `json:"value,omitempty"`
	// Currency is three letters; null resets it to BRL while Value is set.
	Currency *Nullable[string] `json:"currency,omitempty"`
	// StartDate is YYYY-MM-DD, from year 1000.
	StartDate *Nullable[string] `json:"start_date,omitempty"`
	// EndDate must be on or after StartDate. A change starts a new term.
	EndDate *Nullable[string] `json:"end_date,omitempty"`
	// AutoRenew enables automatic renewal; null resets it to false.
	AutoRenew *Nullable[bool] `json:"auto_renew,omitempty"`
	// RenewalTermMonths is 1 through 600; null restores the 12-month default.
	RenewalTermMonths *Nullable[int] `json:"renewal_term_months,omitempty"`
	// NoticePeriodDays is 0 through 3650; null removes the notice period.
	NoticePeriodDays *Nullable[int] `json:"notice_period_days,omitempty"`
	// OwnerID must name an active workspace member, or null to clear ownership.
	OwnerID *Nullable[string] `json:"owner_id,omitempty"`
	// Metadata replaces up to 50 entries. Keys are 1–40 letters, digits, _, -
	// or .; values are plain text up to 500 characters. Numeric text under an
	// _at key or stuck_since is rejected by the API.
	Metadata *Nullable[map[string]string] `json:"metadata,omitempty"`
	// ExternalReferences replaces the whole list (at most 20); [] or null clears it.
	ExternalReferences *Nullable[[]ContractExternalReference] `json:"external_references,omitempty"`
	// ParentDocumentID links to a contract family; null unlinks a pending amendment.
	ParentDocumentID *Nullable[string] `json:"parent_document_id,omitempty"`
	// RelationType is required on the first link. Applied amendments are read-only.
	RelationType *ContractRelationType `json:"relation_type,omitempty"`
	// UpdatedAt is the root or link timestamp read from Get. A stale value returns
	// 409 ContractChanged; omit for a first write. The API compares to the second.
	UpdatedAt *Timestamp `json:"updated_at,omitempty"`
}

// RenewContractRequest starts the next term. Send Term or a later EndDate so a
// repeated request cannot renew twice. Terminated contracts cannot be renewed.
type RenewContractRequest struct {
	// EndDate is a later YYYY-MM-DD date. Nil, empty or null uses the renewal
	// interval, stepped into the future, and then requires Term.
	EndDate *Nullable[string] `json:"end_date,omitempty"`
	// Value replaces the amount, or clears it with an empty Nullable.
	Value *Nullable[json.Number] `json:"value,omitempty"`
	// Term is the current term (at least one), required without a later EndDate.
	Term *int `json:"term,omitempty"`
	// UpdatedAt rejects concurrent changes with a 409 ContractChanged response.
	UpdatedAt *Timestamp `json:"updated_at,omitempty"`
}

// TerminateContractRequest closes a contract and stops lifecycle alerts.
type TerminateContractRequest struct {
	// Reason is optional text up to 500 characters, retained in the event; HTML is stripped.
	Reason *string `json:"reason,omitempty"`
	// UpdatedAt rejects concurrent changes with a 409 ContractChanged response.
	UpdatedAt *Timestamp `json:"updated_at,omitempty"`
}

// ContractListParams filters and paginates workspace contract terms.
type ContractListParams struct {
	// ListParams supplies page, per-page, search (document name), sort and status
	// (comma-separated ContractStatus values). Method, Tags and AccountID are ignored.
	ListParams
	// Counterparty matches part of the name or the exact CPF/CNPJ.
	Counterparty string
	// EndDateFrom is the inclusive lower YYYY-MM-DD end-date bound.
	EndDateFrom string
	// EndDateTo is the inclusive upper YYYY-MM-DD end-date bound.
	EndDateTo string
	// EndingWithinDays is 0–3650 days from today in Brasília, inclusive; zero means today.
	EndingWithinDays *int
	// ExternalSystem selects an external reference's system.
	ExternalSystem string
	// ExternalID selects an external reference's ID.
	ExternalID string
	// Expand accepts document to include each contract's document.
	Expand string
}
