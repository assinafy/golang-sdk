package resources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*internal.HTTPClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return internal.NewHTTPClient(srv.URL, "key", "", time.Second), srv
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

func TestSignerCreateUsesDefaultAccount(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/acc1/signers" {
			t.Errorf("path = %q", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body["full_name"] != "Bob" {
			t.Errorf("full_name = %v", body["full_name"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":{"id":"s1","full_name":"Bob","has_accepted_terms":false}}`)
	})

	r := NewSignerResource(httpClient, "acc1")
	signer, err := r.Create(context.Background(), "", &models.CreateSignerRequest{FullName: "Bob"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if signer.ID != "s1" {
		t.Errorf("ID = %q", signer.ID)
	}
}

func TestAssignmentsSignSendsAccessCode(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/assignments/aid") {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("signer-access-code"); got != "code-1" {
			t.Errorf("access code = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	a := NewAssignmentResource(httpClient)
	err := a.Sign(context.Background(), "did", "aid", "code-1", []models.SignDocumentItem{{ItemID: "i", FieldID: "f", PageID: "p", Value: "v"}})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
}

func TestAssignmentsResendNotificationReturnsResult(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		if got := r.URL.Path; got != "/documents/doc1/assignments/asg1/signers/signer1/resend" {
			t.Errorf("path = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":{"is_sent":true,"document_id":"doc1","signer_id":"signer1"}}`)
	})

	r := NewAssignmentResource(httpClient)
	out, err := r.ResendNotification(context.Background(), "doc1", "asg1", "signer1")
	if err != nil {
		t.Fatalf("ResendNotification: %v", err)
	}
	if !out.IsSent || out.DocumentID != "doc1" || out.SignerID != "signer1" {
		t.Errorf("result = %+v", out)
	}
}

func TestWebhooksListDispatchesEncodesParams(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("page") != "1" || q.Get("per-page") != "5" || q.Get("delivered") != "true" || q.Get("from") != "1700000000" {
			t.Errorf("query = %v", q)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	r := NewWebhookResource(httpClient, "acc1")
	delivered := true
	_, err := r.ListDispatches(context.Background(), "", &models.WebhookDispatchListParams{
		PerPage: 5, Delivered: &delivered, From: 1700000000,
	})
	if err != nil {
		t.Fatalf("ListDispatches: %v", err)
	}
}

func TestTemplatesListEncodesStatusFilter(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("status"); got != "ready" {
			t.Errorf("status = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	r := NewTemplateResource(httpClient, "acc1")
	_, err := r.List(context.Background(), "", &models.ListParams{Status: "ready"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestCreateFromTemplateDoesNotMutateOptions(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body models.CreateDocumentFromTemplateOptions
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
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

func TestSignerDocumentsListEncodesDocumentFilters(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("signer-access-code") != "code-1" || q.Get("status") != "pending_signature" || q.Get("method") != "collect" {
			t.Errorf("query = %v", q)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	r := NewSignerDocumentResource(httpClient)
	_, err := r.List(context.Background(), "signer1", "code-1", &models.ListParams{Status: "pending_signature", Method: "collect"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestFieldsListWithFlags(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("include_inactive") != "true" || q.Get("include_standard") != "false" {
			t.Errorf("query = %v", q)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[{"id":"f1","name":"x","type":"text","is_active":true,"is_required":false,"is_standard":false,"is_read_only":false,"is_visible":true}]}`)
	})

	r := NewFieldResource(httpClient, "acc1")
	out, err := r.List(context.Background(), "", &models.ListFieldDefinitionsParams{IncludeInactive: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(out) != 1 || out[0].ID != "f1" {
		t.Errorf("got %+v", out)
	}
}

func TestPathEscaping(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.EscapedPath(), "/accounts/acc%2F1/signers/s%201") {
			t.Errorf("path = %q", r.URL.EscapedPath())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":{"id":"s","full_name":"a","has_accepted_terms":false}}`)
	})

	r := NewSignerResource(httpClient, "")
	if _, err := r.Get(context.Background(), "acc/1", "s 1"); err != nil {
		t.Fatalf("Get: %v", err)
	}
}
