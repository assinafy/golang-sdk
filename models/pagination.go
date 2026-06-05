package models

const (
	defaultPage           = 1
	defaultPerPage        = 25
	defaultWebhookPerPage = 20
	maxPerPage            = 100
)

// PaginationMeta is read from the X-Pagination-* response headers.
type PaginationMeta struct {
	CurrentPage int
	TotalCount  int
	PageCount   int
	PerPage     int
}

// ListParams covers the query parameters supported by paginated list endpoints.
// Not every field applies to every endpoint; resources send only the parameters
// the corresponding endpoint documents.
type ListParams struct {
	Page    int
	PerPage int
	Search  string
	Sort    string
	Status  string
	Method  string
	// Tags filters by tag ID with AND semantics (a record must carry all of
	// them). Sent as a comma-separated list. Honoured by the documents and
	// templates list endpoints; ignored elsewhere.
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
	Data       []T
	Pagination PaginationMeta
}
