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

func decodeContractBody(t *testing.T, r *http.Request) any {
	t.Helper()
	var body any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("decode body: %v", err)
	}
	return body
}

func writeContractResponse(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, body)
}

func assertNoClientAuth(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Header.Get("X-Api-Key") != "" || r.Header.Get("Authorization") != "" {
		t.Errorf("public/signer request included client authentication: %v", r.Header)
	}
}

func TestPublicDocumentContractMethods(t *testing.T) {
	t.Run("get full document", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assertNoClientAuth(t, r)
			if r.Method != http.MethodGet || r.URL.EscapedPath() != "/public/documents/doc%2F1" {
				t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
			}
			writeContractResponse(w, `{"status":200,"data":{"resource":"document","id":"doc/1","account_id":"acc1","name":"contract.pdf","status":"ready","is_closed":true}}`)
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
			writeContractResponse(w, `{"status":200,"message":""}`)
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
				writeContractResponse(w, `{"status":400,"message":"O atributo channel é obrigatório."}`)
				return
			}
			want := map[string]any{"recipient": "person@example.com", "channel": "email"}
			if got := decodeContractBody(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("fallback body = %#v, want %#v", got, want)
			}
			writeContractResponse(w, `{"status":200,"message":""}`)
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
			writeContractResponse(w, `{"status":200,"message":""}`)
		})

		if err := NewPublicDocumentResource(httpClient).SendTokenWithRequest(context.Background(), "d1", nil); err != nil {
			t.Fatal(err)
		}
	})
}

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
			writeContractResponse(w, `{"status":200,"data":{"documents":1,"credits":0}}`)
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
			writeContractResponse(w, `{"status":200,"data":{"id":"d1","name":"x","status":"ready"}}`)
		})

		if _, err := NewAssignmentResource(httpClient).GetSigningInfoWithTerms(context.Background(), "code1", true); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSignerContractMethods(t *testing.T) {
	t.Run("confirm data returns signer", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut || r.URL.EscapedPath() != "/documents/d1/signers/confirm-data" {
				t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
			}
			assertNoClientAuth(t, r)
			if r.URL.Query().Get("signer-access-code") != "code1" {
				t.Errorf("query = %v", r.URL.Query())
			}
			want := map[string]any{"full_name": "New Name"}
			if got := decodeContractBody(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
			writeContractResponse(w, `{"status":200,"data":{"id":"s1","full_name":"New Name","has_accepted_terms":false}}`)
		})

		name := "New Name"
		signer, err := NewSignerResource(httpClient, "acc1").ConfirmDataAndGet(context.Background(), "d1", "code1", &models.ConfirmSignerDataRequest{FullName: &name})
		if err != nil {
			t.Fatal(err)
		}
		if signer.ID != "s1" || signer.FullName != name {
			t.Fatalf("signer = %+v", signer)
		}
	})

	t.Run("accept terms envelope only", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut || r.URL.Path != "/signers/accept-terms" || r.URL.Query().Get("signer-access-code") != "code1" {
				t.Errorf("request = %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
			}
			writeContractResponse(w, `{"status":200,"message":""}`)
		})

		if err := NewSignerResource(httpClient, "").AcceptTermsOnly(context.Background(), "code1"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("upload png with reuse", func(t *testing.T) {
		image := []byte{1, 2, 3}
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.EscapedPath() != "/signature" {
				t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
			}
			assertNoClientAuth(t, r)
			if r.Header.Get("Content-Type") != "image/png" {
				t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
			}
			if got := r.URL.Query(); !reflect.DeepEqual(got, url.Values{
				"reuse": {"true"}, "signer-access-code": {"code1"}, "type": {"signature"},
			}) {
				t.Errorf("query = %v", got)
			}
			got, _ := io.ReadAll(r.Body)
			if !reflect.DeepEqual(got, image) {
				t.Errorf("body = %v", got)
			}
			writeContractResponse(w, `{"status":200,"message":""}`)
		})

		if err := NewSignerResource(httpClient, "").UploadSignatureWithReuse(context.Background(), "code1", "signature", true, image); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSignerDocumentSearchAllUsesOnlyOfficialQuery(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/signers/s1/documents/search" {
			t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
		}
		assertNoClientAuth(t, r)
		want := url.Values{"search": {"contract"}, "signer-access-code": {"code1"}}
		if got := r.URL.Query(); !reflect.DeepEqual(got, want) {
			t.Errorf("query = %v, want %v", got, want)
		}
		writeContractResponse(w, `{"status":200,"data":[{"id":"d1","name":"contract.pdf","status":"ready"}]}`)
	})

	docs, err := NewSignerDocumentResource(httpClient).SearchAll(context.Background(), "s1", "code1", "contract")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || docs[0].ID != "d1" {
		t.Fatalf("documents = %+v", docs)
	}
}

func TestAuthenticatedFieldValidationOmitsSignerCode(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.EscapedPath() != "/accounts/acc1/fields/f1/validate" {
				t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
			}
			if len(r.URL.Query()) != 0 {
				t.Errorf("query = %v", r.URL.Query())
			}
			if r.Header.Get("X-Api-Key") != "key" {
				t.Errorf("X-Api-Key = %q", r.Header.Get("X-Api-Key"))
			}
			want := map[string]any{"value": "ok"}
			if got := decodeContractBody(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
			writeContractResponse(w, `{"status":200,"data":{"type":"text","success":true,"error_message":""}}`)
		})

		result, err := NewFieldResource(httpClient, "acc1").ValidateAuthenticated(context.Background(), "", "f1", &models.ValidateFieldRequest{Value: "ok"})
		if err != nil {
			t.Fatal(err)
		}
		if !result.Success {
			t.Fatalf("result = %+v", result)
		}
	})

	t.Run("multiple", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.EscapedPath() != "/accounts/acc1/fields/validate-multiple" {
				t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
			}
			if len(r.URL.Query()) != 0 {
				t.Errorf("query = %v", r.URL.Query())
			}
			want := []any{map[string]any{"field_id": "f1", "value": "ok"}}
			if got := decodeContractBody(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
			writeContractResponse(w, `{"status":200,"data":[{"field_id":"f1","type":"text","success":true,"error_message":""}]}`)
		})

		result, err := NewFieldResource(httpClient, "acc1").ValidateMultipleAuthenticated(context.Background(), "", []models.ValidateMultipleFieldsRequest{{FieldID: "f1", Value: "ok"}})
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 || !result[0].Success {
			t.Fatalf("result = %+v", result)
		}
	})
}
