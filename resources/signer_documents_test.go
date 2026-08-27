package resources

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/assinafy/golang-sdk/models"
)

func TestSignerDocumentsSearchAllUsesDocumentedQuery(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/signers/s1/documents/search" {
			t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
		}
		assertNoClientAuth(t, r)
		want := url.Values{"search": {"contract"}, "signer-access-code": {"code1"}}
		if got := r.URL.Query(); !reflect.DeepEqual(got, want) {
			t.Errorf("query = %v, want %v", got, want)
		}
		writeJSONResponse(w, `{"status":200,"data":[{"id":"d1","name":"contract.pdf","status":"ready"}]}`)
	})

	docs, err := NewSignerDocumentResource(httpClient).SearchAll(context.Background(), "s1", "code1", "contract")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || docs[0].ID != "d1" {
		t.Fatalf("documents = %+v", docs)
	}
}

func TestSignerDocumentsSearch(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/signers/s1/documents/search" {
			t.Errorf("path = %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("signer-access-code") != "code-1" || q.Get("search") != "nda" {
			t.Errorf("query = %v", q)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	r := NewSignerDocumentResource(httpClient)
	if _, err := r.Search(context.Background(), "s1", "code-1", &models.ListParams{Search: "nda"}); err != nil {
		t.Fatalf("Search: %v", err)
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
