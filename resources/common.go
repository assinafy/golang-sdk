// Package resources contains the typed clients for each Assinafy API resource.
//
// Each resource is a thin wrapper around internal.HTTPClient. Methods accept a
// context.Context, optional account ID, and the documented request/response
// types from the models package.
package resources

import (
	"net/http"
	"strconv"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

// resolveAccountID returns id when set, otherwise the configured fallback.
func resolveAccountID(id, fallback string) string {
	if id != "" {
		return id
	}
	return fallback
}

func applyListParams(req *internal.Request, params *models.ListParams) {
	if params == nil {
		params = &models.ListParams{}
	}
	params.SetDefaults()
	req.WithQuery("page", strconv.Itoa(params.Page))
	req.WithQuery("per-page", strconv.Itoa(params.PerPage))
	req.WithQuery("search", params.Search)
	req.WithQuery("sort", params.Sort)
}

func paginated[T any](data []T, resp *internal.Response) *models.PaginatedResult[T] {
	return &models.PaginatedResult[T]{Data: data, Pagination: extractPagination(resp.Headers)}
}

func extractPagination(headers http.Header) models.PaginationMeta {
	return models.PaginationMeta{
		CurrentPage: parseHeaderInt(headers, "X-Pagination-Current-Page"),
		TotalCount:  parseHeaderInt(headers, "X-Pagination-Total-Count"),
		PageCount:   parseHeaderInt(headers, "X-Pagination-Page-Count"),
		PerPage:     parseHeaderInt(headers, "X-Pagination-Per-Page"),
	}
}

func parseHeaderInt(headers http.Header, name string) int {
	v := headers.Get(name)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}
