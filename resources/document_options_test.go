package resources

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/assinafy/golang-sdk/v2/internal"
	"github.com/assinafy/golang-sdk/v2/models"
)

func TestDocumentListsFilterEnvelope(t *testing.T) {
	ctx := context.Background()
	params := &models.ListParams{EnvelopeID: "envelope/1", Search: "contract", Page: 2, PerPage: 3}
	for _, tc := range []struct {
		name, path string
		signer     bool
		call       func(*internal.HTTPClient) (*models.PaginatedResult[models.Document], error)
	}{
		{"workspace list", "/accounts/default/documents", false, func(h *internal.HTTPClient) (*models.PaginatedResult[models.Document], error) {
			return NewDocumentResource(h, "default").List(ctx, "", params)
		}},
		{"workspace search", "/accounts/default/documents/search", false, func(h *internal.HTTPClient) (*models.PaginatedResult[models.Document], error) {
			return NewDocumentResource(h, "default").Search(ctx, "", params)
		}},
		{"signer list", "/signers/signer%2F1/documents", true, func(h *internal.HTTPClient) (*models.PaginatedResult[models.Document], error) {
			return NewSignerDocumentResource(h).List(ctx, "signer/1", "fixture-code", params)
		}},
		{"signer workspace list", "/signers/signer%2F1/documents", false, func(h *internal.HTTPClient) (*models.PaginatedResult[models.Document], error) {
			return NewSignerDocumentResource(h).ListAsWorkspace(ctx, "signer/1", params)
		}},
		{"signer paginated search", "/signers/signer%2F1/documents/search", true, func(h *internal.HTTPClient) (*models.PaginatedResult[models.Document], error) {
			return NewSignerDocumentResource(h).Search(ctx, "signer/1", "fixture-code", params)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				want := url.Values{"envelope_id": {"envelope/1"}, "search": {"contract"}, "page": {"2"}, "per-page": {"3"}}
				if tc.signer {
					assertNoClientAuth(t, r)
					want.Set("signer-access-code", "fixture-code")
				} else if r.Header.Get("X-Api-Key") != "key" {
					t.Error("missing workspace credentials")
				}
				if r.Method != http.MethodGet || r.URL.EscapedPath() != tc.path || !reflect.DeepEqual(r.URL.Query(), want) {
					t.Errorf("request = %s %s", r.Method, r.URL)
				}
				w.Header().Set("X-Pagination-Total-Count", "8")
				w.Header().Set("X-Pagination-Current-Page", "2")
				writeJSONResponse(w, `{"status":200,"data":[{"id":"document-1","envelope_id":"envelope/1"}]}`)
			})
			out, err := tc.call(h)
			if err != nil || len(out.Data) != 1 || out.Data[0].ID != "document-1" || out.Data[0].EnvelopeID == nil || *out.Data[0].EnvelopeID != "envelope/1" || out.Pagination.TotalCount != 8 || out.Pagination.CurrentPage != 2 {
				t.Fatalf("result = %#v, %v", out, err)
			}
		})
	}
}

func TestSignerSearchOptions(t *testing.T) {
	for _, params := range []*models.ListParams{nil, {Search: "contract", EnvelopeID: "envelope/1", Page: 3, PerPage: 5, Sort: "name"}} {
		for _, workspace := range []bool{false, true} {
			h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				want := url.Values{}
				if params != nil {
					want.Set("search", params.Search)
					want.Set("envelope_id", params.EnvelopeID)
				}
				if workspace {
					if r.Header.Get("X-Api-Key") != "key" {
						t.Error("missing workspace authentication")
					}
				} else {
					assertNoClientAuth(t, r)
					want.Set("signer-access-code", "fixture-code")
				}
				if r.Method != http.MethodGet || r.URL.EscapedPath() != "/signers/signer%2F1/documents/search" || !reflect.DeepEqual(r.URL.Query(), want) {
					t.Errorf("request = %s %s", r.Method, r.URL)
				}
				writeJSONResponse(w, `{"status":200,"data":[{"id":"doc-1","name":"contract"}]}`)
			})
			r := NewSignerDocumentResource(h)
			var out []models.Document
			var err error
			if workspace {
				out, err = r.SearchAsWorkspaceWithParams(context.Background(), "signer/1", params)
			} else {
				out, err = r.SearchAllWithParams(context.Background(), "signer/1", "fixture-code", params)
			}
			if err != nil || len(out) != 1 || out[0].ID != "doc-1" || out[0].Name != "contract" {
				t.Fatalf("result = %#v, %v", out, err)
			}
		}
	}
}

func TestContractRelatedExpansion(t *testing.T) {
	h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/accounts/default/documents/document%2F1/contract/related" || r.Header.Get("X-Api-Key") != "key" || !reflect.DeepEqual(r.URL.Query(), url.Values{"expand": {"document"}}) {
			t.Errorf("request = %s %s", r.Method, r.URL)
		}
		writeJSONResponse(w, `{"status":200,"data":{"document_id":"root","document":{"id":"root","name":"Service agreement"},"amendments":[{"document_id":"document/1","document_name":"Amendment"}]}}`)
	})
	out, err := NewContractResource(h, "default").RelatedWithParams(context.Background(), "", "document/1", &models.ContractListParams{Expand: "document", ListParams: models.ListParams{Page: 2, Search: "unused"}})
	if err != nil || out.DocumentID != "root" || out.Document == nil || out.Document.Name != "Service agreement" || len(out.Amendments) != 1 || out.Amendments[0].DocumentID != "document/1" {
		t.Fatalf("result = %#v, %v", out, err)
	}
}

func TestSignedDocumentRenders(t *testing.T) {
	for _, page := range []bool{false, true} {
		h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			path := "/documents/document%2F1/thumbnail"
			if page {
				path = "/documents/document%2F1/pages/page%2F1/download"
			}
			if r.Method != http.MethodGet || r.URL.EscapedPath() != path || r.Header.Get("X-Api-Key") != "key" || !reflect.DeepEqual(r.URL.Query(), url.Values{"artifact": {"certificated"}}) {
				t.Errorf("request = %s %s", r.Method, r.URL)
			}
			_, _ = w.Write([]byte("signed-image"))
		})
		r := NewDocumentResource(h, "")
		var data []byte
		var err error
		if page {
			data, err = r.DownloadPageWithArtifact(context.Background(), "document/1", "page/1", "certificated")
		} else {
			data, err = r.ThumbnailWithArtifact(context.Background(), "document/1", "certificated")
		}
		if err != nil || string(data) != "signed-image" {
			t.Fatalf("render = %q, %v", data, err)
		}
	}
}

func TestDocumentVerificationCertificationTime(t *testing.T) {
	for _, stamp := range []string{`null`, `"2026-11-04T12:05:00Z"`} {
		h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assertNoClientAuth(t, r)
			if r.Method != http.MethodGet || r.URL.EscapedPath() != "/documents/hash%2F1/verify" {
				t.Errorf("request = %s %s", r.Method, r.URL)
			}
			writeJSONResponse(w, `{"status":200,"data":{"hash":"hash/1","is_valid":true,"certificated_at":`+stamp+`}}`)
		})
		out, err := NewDocumentResource(h, "").Verify(context.Background(), "hash/1")
		if err != nil || out.Hash != "hash/1" || !out.IsValid {
			t.Fatalf("Verify = %#v, %v", out, err)
		}
		if stamp == `null` && out.CertificatedAt != nil || stamp != `null` && (out.CertificatedAt == nil || *out.CertificatedAt != "2026-11-04T12:05:00Z") {
			t.Fatalf("CertificatedAt = %v", out.CertificatedAt)
		}
	}
}
