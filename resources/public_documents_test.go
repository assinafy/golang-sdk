package resources

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/assinafy/golang-sdk/models"
)

func TestPublicDocumentContractMethods(t *testing.T) {
	t.Run("get full document", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assertNoClientAuth(t, r)
			if r.Method != http.MethodGet || r.URL.EscapedPath() != "/public/documents/doc%2F1" {
				t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
			}
			writeJSONResponse(w, `{"status":200,"data":{"resource":"document","id":"doc/1","account_id":"acc1","name":"contract.pdf","status":"ready","is_closed":true}}`)
		})

		doc, err := NewPublicDocumentResource(httpClient).GetDocument(context.Background(), "doc/1")
		if err != nil {
			t.Fatal(err)
		}
		if doc.ID != "doc/1" || doc.AccountID != "acc1" || doc.Status != models.StatusReady || !doc.IsClosed {
			t.Fatalf("document = %+v", doc)
		}
	})

	t.Run("send email only", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assertNoClientAuth(t, r)
			if r.Method != http.MethodPut || r.URL.Path != "/public/documents/d1/send-token" {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}
			want := map[string]any{"email": "person@example.com"}
			if got := decodeContractBody(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
			writeJSONResponse(w, `{"status":200,"message":""}`)
		})

		if err := NewPublicDocumentResource(httpClient).SendTokenByEmail(context.Background(), "d1", "person@example.com"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("send email falls back for legacy sandbox", func(t *testing.T) {
		calls := 0
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls == 1 {
				writeJSONResponse(w, `{"status":400,"message":"O atributo channel é obrigatório."}`)
				return
			}
			want := map[string]any{"recipient": "person@example.com", "channel": "email"}
			if got := decodeContractBody(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("fallback body = %#v, want %#v", got, want)
			}
			writeJSONResponse(w, `{"status":200,"message":""}`)
		})

		if err := NewPublicDocumentResource(httpClient).SendTokenByEmail(context.Background(), "d1", "person@example.com"); err != nil {
			t.Fatal(err)
		}
		if calls != 2 {
			t.Fatalf("calls = %d, want 2", calls)
		}
	})

	t.Run("send token omits optional body", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			if len(body) != 0 {
				t.Errorf("body = %q, want empty", body)
			}
			writeJSONResponse(w, `{"status":200,"message":""}`)
		})

		if err := NewPublicDocumentResource(httpClient).SendTokenWithRequest(context.Background(), "d1", nil); err != nil {
			t.Fatal(err)
		}
	})
}
