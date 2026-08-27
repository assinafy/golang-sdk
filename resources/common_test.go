package resources

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
	"github.com/assinafy/golang-sdk/internal"
)

func decodeContractBody(t *testing.T, r *http.Request) any {
	t.Helper()
	var body any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("decode body: %v", err)
	}
	return body
}

func assertNoClientAuth(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Header.Get("X-Api-Key") != "" || r.Header.Get("Authorization") != "" {
		t.Errorf("public/signer request included client authentication: %v", r.Header)
	}
}

func writeJSONResponse(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, body)
}

func newTestClient(t *testing.T, handler http.HandlerFunc) (*internal.HTTPClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return internal.NewHTTPClient(srv.URL, "key", "", time.Second), srv
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

func TestRequiredIDsAreValidatedBeforeSending(t *testing.T) {
	httpClient := internal.NewHTTPClient("http://127.0.0.1:1", "key", "", time.Second)
	for _, call := range []struct {
		name string
		fn   func() error
	}{
		{"default account", func() error {
			_, err := NewAccountResource(httpClient, "").Get(context.Background(), "")
			return err
		}},
		{"document", func() error {
			_, err := NewDocumentResource(httpClient, "account").Get(context.Background(), " ")
			return err
		}},
		{"nested id", func() error {
			_, err := NewSignerResource(httpClient, "account").Get(context.Background(), "", "")
			return err
		}},
	} {
		t.Run(call.name, func(t *testing.T) {
			if err := call.fn(); !errors.Is(err, sdkerrors.ErrInvalidInput) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRequiredArrayBodiesEncodeAsArrays(t *testing.T) {
	for _, tc := range []struct {
		name, want, response string
		call                 func(*internal.HTTPClient) error
	}{
		{"sign assignment", `[]`, "", func(h *internal.HTTPClient) error {
			return NewAssignmentResource(h).Sign(context.Background(), "document", "assignment", "code", nil)
		}},
		{"sign multiple", `{"document_ids":[]}`, "", func(h *internal.HTTPClient) error {
			return NewSignerDocumentResource(h).SignMultiple(context.Background(), "code", nil)
		}},
		{"decline multiple", `{"decline_reason":"reason","document_ids":[]}`, "", func(h *internal.HTTPClient) error {
			return NewSignerDocumentResource(h).DeclineMultiple(context.Background(), "code", nil, "reason")
		}},
		{"validate multiple", `[]`, "", func(h *internal.HTTPClient) error {
			_, err := NewFieldResource(h, "account").ValidateMultipleAuthenticated(context.Background(), "", nil)
			return err
		}},
		{"create from template", `{"signers":[]}`, `{"status":200,"data":{}}`, func(h *internal.HTTPClient) error {
			_, err := NewDocumentResource(h, "account").CreateFromTemplate(context.Background(), "", "template", nil, nil)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				var got, want any
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Errorf("decode body: %v", err)
				}
				if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
					t.Fatalf("decode expected body: %v", err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("body = %#v, want %#v", got, want)
				}
				response := tc.response
				if response == "" {
					response = `{"status":200,"data":[]}`
				}
				_, _ = io.WriteString(w, response)
			})
			if err := tc.call(httpClient); err != nil {
				t.Fatalf("call: %v", err)
			}
		})
	}
}
