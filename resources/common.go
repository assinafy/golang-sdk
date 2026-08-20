// Package resources contains the typed clients for each Assinafy API resource.
//
// Unless a method says otherwise, it uses the API key or bearer token configured
// on assinafy.Client. Account-scoped methods accept an account ID and fall back
// to ClientOptions.AccountID when it is empty. Signer-facing methods authenticate
// with their signer access code, and public/authentication methods need no client
// credential unless their comments say otherwise.
//
// JSON methods encode the documented models request and decode the response's
// data envelope into the documented result. Download methods return raw bytes.
// Non-success responses are returned as *errors.APIError; transport failures are
// *errors.NetworkError, while local encoding/decoding errors wrap their cause.
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
	p := models.ListParams{}
	if params != nil {
		p = *params
	}
	p.SetDefaults()
	req.WithQuery("page", strconv.Itoa(p.Page))
	req.WithQuery("per-page", strconv.Itoa(p.PerPage))
	req.WithQuery("search", p.Search)
	req.WithQuery("sort", p.Sort)
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
