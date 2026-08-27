package resources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/assinafy/golang-sdk/models"
)

func TestAssignmentContractMethods(t *testing.T) {
	t.Run("estimate request has no signer id", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/documents/d1/assignments/estimate-cost" {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}
			want := map[string]any{"method": "virtual", "signers": []any{map[string]any{}}}
			if got := decodeContractBody(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
			writeJSONResponse(w, `{"status":200,"data":{"documents":1,"credits":0}}`)
		})

		result, err := NewAssignmentResource(httpClient).EstimateCostWithRequest(context.Background(), "d1", &models.EstimateAssignmentCostRequest{
			Method: models.MethodVirtual, Signers: []models.EstimateAssignmentCostSigner{{}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.Documents != 1 {
			t.Fatalf("result = %+v", result)
		}
	})

	t.Run("signing info includes terms", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.EscapedPath() != "/sign" {
				t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
			}
			assertNoClientAuth(t, r)
			if got := r.URL.Query(); !reflect.DeepEqual(got, url.Values{
				"has_accepted_terms": {"true"}, "signer-access-code": {"code1"},
			}) {
				t.Errorf("query = %v", got)
			}
			writeJSONResponse(w, `{"status":200,"data":{"id":"d1","name":"x","status":"ready"}}`)
		})

		if _, err := NewAssignmentResource(httpClient).GetSigningInfoWithTerms(context.Background(), "code1", true); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAssignmentsList(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/assignments" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("per-page"); got != "5" {
			t.Errorf("per-page = %q", got)
		}
		w.Header().Set("X-Pagination-Total-Count", "1")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[{"id":"asg1","method":"virtual"}]}`)
	})

	r := NewAssignmentResource(httpClient)
	page, err := r.List(context.Background(), &models.ListParams{PerPage: 5})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "asg1" {
		t.Errorf("page = %+v", page)
	}
}

func TestResetExpirationRequestShapes(t *testing.T) {
	expiresAt := "2027-01-02T03:04:05Z"
	for _, tc := range []struct {
		name string
		call func(*AssignmentResource) error
		want map[string]any
	}{
		{"legacy null", func(r *AssignmentResource) error {
			_, err := r.ResetExpiration(context.Background(), "d1", "a1", nil)
			return err
		}, map[string]any{"expires_at": nil}},
		{"exact omitted", func(r *AssignmentResource) error {
			_, err := r.ResetExpirationWithRequest(context.Background(), "d1", "a1", models.ResetAssignmentExpirationRequest{})
			return err
		}, map[string]any{}},
		{"exact value", func(r *AssignmentResource) error {
			_, err := r.ResetExpirationWithRequest(context.Background(), "d1", "a1", models.ResetAssignmentExpirationRequest{ExpiresAt: &expiresAt})
			return err
		}, map[string]any{"expires_at": expiresAt}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut || r.URL.EscapedPath() != "/documents/d1/assignments/a1/reset-expiration" {
					t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
				}
				var got map[string]any
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Errorf("decode body: %v", err)
				}
				if !reflect.DeepEqual(got, tc.want) {
					t.Errorf("body = %#v, want %#v", got, tc.want)
				}
				writeJSONResponse(w, `{"status":200,"data":{"id":"a1","method":"virtual"}}`)
			})
			if err := tc.call(NewAssignmentResource(httpClient)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAssignmentsSignSendsAccessCode(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.EscapedPath() != "/documents/did/assignments/aid" {
			t.Errorf("path = %q", r.URL.EscapedPath())
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
