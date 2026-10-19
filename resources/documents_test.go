package resources

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/internal"
	"github.com/assinafy/golang-sdk/v2/models"
)

func TestDocumentsSearch(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/acc1/documents/search" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("search"); got != "contract" {
			t.Errorf("search = %q", got)
		}
		if got := r.URL.Query().Get("status"); got != "metadata_ready" {
			t.Errorf("status = %q", got)
		}
		w.Header().Set("X-Pagination-Total-Count", "3")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[{"id":"d1","name":"contract.pdf","status":"metadata_ready"}]}`)
	})

	r := NewDocumentResource(httpClient, "acc1")
	page, err := r.Search(context.Background(), "", &models.ListParams{Search: "contract", Status: "metadata_ready"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "d1" || page.Pagination.TotalCount != 3 {
		t.Errorf("page = %+v", page)
	}
}

func TestDocumentsRename(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/documents/d1" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if body["name"] != "renamed.pdf" {
			t.Errorf("name = %v", body["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":{"id":"d1","name":"renamed.pdf","status":"metadata_ready"}}`)
	})

	r := NewDocumentResource(httpClient, "acc1")
	doc, err := r.Rename(context.Background(), "d1", "renamed.pdf")
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if doc.Name != "renamed.pdf" {
		t.Errorf("name = %q", doc.Name)
	}
}

func TestDocumentsList(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/accounts/acc1/documents" {
			t.Errorf("path = %q", got)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("page = %q", got)
		}
		if got := r.URL.Query().Get("per-page"); got != "10" {
			t.Errorf("per-page = %q", got)
		}
		if got := r.URL.Query().Get("status"); got != "metadata_ready" {
			t.Errorf("status = %q", got)
		}
		if got := r.URL.Query().Get("method"); got != "virtual" {
			t.Errorf("method = %q", got)
		}
		if got := r.Header.Get("X-Api-Key"); got != "key" {
			t.Errorf("X-Api-Key = %q", got)
		}
		w.Header().Set("X-Pagination-Total-Count", "42")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[{"id":"d1","name":"a","status":"uploaded"}]}`)
	})

	docs := NewDocumentResource(httpClient, "acc1")
	page, err := docs.List(context.Background(), "", &models.ListParams{Page: 2, PerPage: 10, Status: "metadata_ready", Method: "virtual"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "d1" {
		t.Errorf("Data = %+v", page.Data)
	}
	if page.Pagination.TotalCount != 42 {
		t.Errorf("TotalCount = %d", page.Pagination.TotalCount)
	}
}

func TestDocumentsListDoesNotMutateParams(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	params := &models.ListParams{}
	docs := NewDocumentResource(httpClient, "acc1")
	if _, err := docs.List(context.Background(), "", params); err != nil {
		t.Fatalf("List: %v", err)
	}
	if params.Page != 0 || params.PerPage != 0 {
		t.Errorf("List mutated params: %+v", params)
	}
}

func TestDocumentsListEncodesTags(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("tags"); got != "t1,t2" {
			t.Errorf("tags = %q, want t1,t2", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	docs := NewDocumentResource(httpClient, "acc1")
	if _, err := docs.List(context.Background(), "", &models.ListParams{Tags: []string{"t1", "t2"}}); err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestDocumentsListOmitsEmptyTags(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.URL.Query()["tags"]; ok {
			t.Errorf("tags param should be absent, got %q", r.URL.Query().Get("tags"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	docs := NewDocumentResource(httpClient, "acc1")
	if _, err := docs.List(context.Background(), "", &models.ListParams{}); err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestCreateFromTemplateDoesNotMutateOptions(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.EscapedPath() != "/accounts/acc1/templates/template1/documents" {
			t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
		}
		var body models.CreateDocumentFromTemplateOptions
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if len(body.Signers) != 1 || body.Signers[0].ID != "signer1" {
			t.Errorf("signers = %+v", body.Signers)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":{"id":"doc1","name":"from-template.pdf","status":"uploaded"}}`)
	})

	opts := &models.CreateDocumentFromTemplateOptions{Name: "from-template.pdf"}
	r := NewDocumentResource(httpClient, "acc1")
	if _, err := r.CreateFromTemplate(context.Background(), "", "template1", []models.TemplateSigner{{RoleID: "role1", ID: "signer1"}}, opts); err != nil {
		t.Fatalf("CreateFromTemplate: %v", err)
	}
	if len(opts.Signers) != 0 {
		t.Errorf("expected options not to be mutated, got %+v", opts.Signers)
	}
}

func TestTemplateDigitalCertificateRequiresVirtualMethod(t *testing.T) {
	for _, operation := range []string{"create", "estimate"} {
		t.Run(operation, func(t *testing.T) {
			requests := 0
			httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != http.MethodGet || r.URL.EscapedPath() != "/accounts/acc1/templates/template1" {
					t.Errorf("request = %s %s, want template detail", r.Method, r.URL.EscapedPath())
				}
				writeJSONResponse(w, `{"status":200,"data":{"id":"template1","name":"placed","status":"ready","roles":[],"tags":[],"created_at":"2026-10-19T00:00:00Z","updated_at":"2026-10-19T00:00:00Z","pages":[{"id":"p1","number":1,"height":100,"width":100,"download_url":"https://example.test/p1","fields":[{"id":"placement1","field_id":"field1","role_id":"role1","label":"Sign","created_at":"2026-10-19T00:00:00Z","updated_at":"2026-10-19T00:00:00Z"}]}]}}`)
			})
			r := NewDocumentResource(httpClient, "acc1")
			signers := []models.TemplateSigner{{RoleID: "role1", ID: "signer1", VerificationMethod: models.VerificationMethodDigitalCertificate}}
			var err error
			if operation == "create" {
				_, err = r.CreateFromTemplate(context.Background(), "", "template1", signers, nil)
			} else {
				_, err = r.EstimateCostFromTemplate(context.Background(), "", "template1", signers)
			}
			if !errors.Is(err, sdkerrors.ErrDigitalCertificateRequiresVirtual) || requests != 1 {
				t.Errorf("err = %v, requests = %d, want local certificate error after one template read", err, requests)
			}
		})
	}
}

func TestVirtualTemplateAllowsDigitalCertificate(t *testing.T) {
	requests := 0
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			writeJSONResponse(w, `{"status":200,"data":{"id":"template1","name":"virtual","status":"ready","roles":[],"tags":[],"created_at":"2026-10-19T00:00:00Z","updated_at":"2026-10-19T00:00:00Z","pages":[]}}`)
		case 2:
			if r.Method != http.MethodPost || r.URL.EscapedPath() != "/accounts/acc1/templates/template1/documents/estimate-cost" {
				t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
			}
			writeJSONResponse(w, `{"status":200,"data":{"documents":1,"credits":1}}`)
		}
	})
	r := NewDocumentResource(httpClient, "acc1")
	result, err := r.EstimateCostFromTemplate(context.Background(), "", "template1", []models.TemplateSigner{{
		RoleID: "role1", VerificationMethod: models.VerificationMethodDigitalCertificate,
	}})
	if err != nil || result.Documents != 1 || requests != 2 {
		t.Fatalf("result = %+v, err = %v, requests = %d", result, err, requests)
	}
}

func TestDocumentTagsReplaceSendsNames(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/accounts/acc1/documents/d1/tags" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		var body struct {
			Tags []string `json:"tags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if len(body.Tags) != 2 || body.Tags[0] != "A" || body.Tags[1] != "B" {
			t.Errorf("tags = %v", body.Tags)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[{"id":"t1","name":"A"},{"id":"t2","name":"B"}]}`)
	})

	r := NewDocumentResource(httpClient, "acc1")
	tags, err := r.ReplaceTags(context.Background(), "", "d1", []string{"A", "B"})
	if err != nil {
		t.Fatalf("ReplaceTags: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("tags = %+v", tags)
	}
}

func TestDocumentTagsReplaceNilBecomesEmptyArray(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Tags []string `json:"tags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if body.Tags == nil || len(body.Tags) != 0 {
			t.Errorf("expected empty (non-null) tags array, got %v", body.Tags)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	r := NewDocumentResource(httpClient, "acc1")
	if _, err := r.ReplaceTags(context.Background(), "", "d1", nil); err != nil {
		t.Fatalf("ReplaceTags: %v", err)
	}
}

func TestDocumentTagsDetach(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/accounts/acc1/documents/d1/tags/t1" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":{"detached":true}}`)
	})

	r := NewDocumentResource(httpClient, "acc1")
	result, err := r.DetachTagWithResult(context.Background(), "", "d1", "t1")
	if err != nil {
		t.Fatalf("DetachTagWithResult: %v", err)
	}
	if !result.Detached {
		t.Errorf("result = %+v", result)
	}
}

func TestDocumentUploadValidation(t *testing.T) {
	r := NewDocumentResource(internal.NewHTTPClient("http://127.0.0.1:1", "key", "", time.Second), "account")
	for _, tc := range []struct {
		name     string
		content  []byte
		fileName string
	}{
		{name: "empty file", fileName: "document.pdf"},
		{name: "empty name", content: []byte("%PDF-1.1"), fileName: " "},
		{name: "not pdf", content: []byte("plain text"), fileName: "document.pdf"},
		{name: "too large", content: make([]byte, maxDocumentSize+1), fileName: "document.pdf"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := r.Upload(context.Background(), "", tc.content, tc.fileName, nil); !errors.Is(err, sdkerrors.ErrInvalidInput) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
