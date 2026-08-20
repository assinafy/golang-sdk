package models

const (
	defaultPage           = 1
	defaultPerPage        = 25
	defaultWebhookPerPage = 20
	maxPerPage            = 100
)

// PaginationMeta is read from the X-Pagination-* response headers.
type PaginationMeta struct {
	// CurrentPage is the one-based page returned by the API.
	CurrentPage int
	// TotalCount is the number of matching records across all pages.
	TotalCount int
	// PageCount is the number of available pages.
	PageCount int
	// PerPage is the page size reported by the API.
	PerPage int
}

// ListParams covers the query parameters supported by paginated list endpoints.
// Not every field applies to every endpoint. Page/PerPage are shared; consult
// each resource method before using compatibility filters on another route.
type ListParams struct {
	// Page is one-based; values less than one default to 1.
	Page int
	// PerPage defaults to 25 and is clamped to the API maximum of 100.
	PerPage int
	// Search is an optional endpoint-specific text query.
	Search string
	// Sort is an optional endpoint-specific sort expression.
	Sort string
	// Status is an optional document, template, or signer-document status filter.
	Status string
	// Method is an optional assignment-method filter such as "virtual" or "collect".
	Method string
	// Tags filters by tag ID with AND semantics (a record must carry all of
	// them). Sent as a comma-separated list. It is documented for documents;
	// templates accept it as a compatibility extension. It is ignored elsewhere.
	Tags []string
}

// SetDefaults normalises Page/PerPage and clamps PerPage to the API maximum.
func (p *ListParams) SetDefaults() {
	if p.Page <= 0 {
		p.Page = defaultPage
	}
	if p.PerPage <= 0 {
		p.PerPage = defaultPerPage
	}
	if p.PerPage > maxPerPage {
		p.PerPage = maxPerPage
	}
}

// PaginatedResult bundles a list response with its pagination metadata.
type PaginatedResult[T any] struct {
	// Data contains the records in the current response page.
	Data []T
	// Pagination contains values decoded from the X-Pagination-* headers.
	Pagination PaginationMeta
}
