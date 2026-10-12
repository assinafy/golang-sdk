package resources

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
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

	t.Run("send token by email", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assertNoClientAuth(t, r)
			if r.Method != http.MethodPut || r.URL.Path != "/public/documents/d1/send-token" {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}
			want := map[string]any{"channel": "email", "recipient": "signer@example.com"}
			if got := decodeContractBody(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
			writeJSONResponse(w, `{"status":200,"message":"","data":{"document":{"resource":"document","id":"d1","name":"c.pdf","page_count":2,"created_by":"Ada"},"channel":"email","recipient":"signer@example.com"}}`)
		})
		public := NewPublicDocumentResource(httpClient)
		ctx := context.Background()

		if err := public.SendTokenByEmail(ctx, "d1", "signer@example.com"); err != nil {
			t.Fatal(err)
		}
		if err := public.SendTokenWithRequest(ctx, "d1", &models.SendDocumentTokenRequest{Email: "signer@example.com"}); err != nil {
			t.Fatal(err)
		}
		got, err := public.SendToken(ctx, "d1", &models.SendDocumentTokenRequest{Recipient: "signer@example.com", Channel: "email"})
		if err != nil {
			t.Fatal(err)
		}
		want := &models.SendDocumentTokenResult{
			Document: models.PublicDocumentInfo{Resource: "document", ID: "d1", Name: "c.pdf", PageCount: "2", CreatedBy: "Ada"},
			Channel:  "email", Recipient: "signer@example.com",
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("result = %+v, want %+v", got, want)
		}
	})

	t.Run("send token rejects invalid requests locally", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(_ http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		})
		public := NewPublicDocumentResource(httpClient)
		for name, body := range map[string]*models.SendDocumentTokenRequest{
			"nil body":        nil,
			"empty recipient": {Channel: "email"},
			"blank recipient": {Recipient: "  "},
			"whatsapp":        {Recipient: "signer@example.com", Channel: "whatsapp"},
		} {
			if err := public.SendTokenWithRequest(context.Background(), "d1", body); !errors.Is(err, sdkerrors.ErrInvalidInput) {
				t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
			}
		}
	})
}
