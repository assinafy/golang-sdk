package models

type PaginationMeta struct {
	CurrentPage int `json:"X-Pagination-Current-Page"`
	TotalCount  int `json:"X-Pagination-Total-Count"`
	PageCount   int `json:"X-Pagination-Page-Count"`
	PerPage     int `json:"X-Pagination-Per-Page"`
}

type ListParams struct {
	Page    int    `json:"page,omitempty"`
	PerPage int    `json:"per-page,omitempty"`
	Search  string `json:"search,omitempty"`
	Sort    string `json:"sort,omitempty"`
}

func (p *ListParams) SetDefaults() {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PerPage == 0 {
		p.PerPage = 25
	}
	if p.PerPage > 100 {
		p.PerPage = 100
	}
}

type PaginatedResult[T any] struct {
	Data       []T
	Pagination PaginationMeta
}
