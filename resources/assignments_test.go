package resources

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/models"
)

func TestAssignmentContractMethods(t *testing.T) {
	t.Run("certificate estimate permits omitted method", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/documents/d1/assignments/estimate-cost" {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}
			want := map[string]any{"signers": []any{map[string]any{"verification_method": "DigitalCertificate"}}}
			if got := decodeContractBody(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
			writeJSONResponse(w, `{"status":200,"data":{"documents":1,"total_credits":0.5}}`)
		})
		result, err := NewAssignmentResource(httpClient).EstimateCostWithRequest(context.Background(), "d1", &models.EstimateAssignmentCostRequest{
			Signers: []models.EstimateAssignmentCostSigner{{VerificationMethod: models.VerificationMethodDigitalCertificate}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.Documents != 1 || result.TotalCredits != 0.5 {
			t.Fatalf("estimate = %+v", result)
		}
	})

	t.Run("virtual certificate estimate has no signer id", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/documents/d1/assignments/estimate-cost" {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}
			want := map[string]any{"method": "virtual", "signers": []any{map[string]any{"verification_method": "DigitalCertificate"}}}
			if got := decodeContractBody(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
			writeJSONResponse(w, `{"status":200,"data":{"documents":1,"credits":0}}`)
		})

		result, err := NewAssignmentResource(httpClient).EstimateCostWithRequest(context.Background(), "d1", &models.EstimateAssignmentCostRequest{
			Method: models.MethodVirtual, Signers: []models.EstimateAssignmentCostSigner{{VerificationMethod: models.VerificationMethodDigitalCertificate}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.Documents != 1 {
			t.Fatalf("result = %+v", result)
		}
	})

	t.Run("collect estimate carries signers alongside entries", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			want := map[string]any{
				"method":  "collect",
				"signers": []any{map[string]any{"verification_method": "Email"}},
				"entries": []any{map[string]any{"page_id": "p1"}},
			}
			if got := decodeContractBody(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
			writeJSONResponse(w, `{"status":200,"data":{"documents":1,"credits":2}}`)
		})

		if _, err := NewAssignmentResource(httpClient).EstimateCostWithRequest(context.Background(), "d1", &models.EstimateAssignmentCostRequest{
			Method:  models.MethodCollect,
			Signers: []models.EstimateAssignmentCostSigner{{VerificationMethod: models.VerificationMethodEmail}},
			Entries: []any{map[string]any{"page_id": "p1"}},
		}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("certificate requires virtual method", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(http.ResponseWriter, *http.Request) {
			t.Error("request must not be sent")
		})
		resource := NewAssignmentResource(httpClient)
		create := &models.CreateAssignmentRequest{Method: models.MethodCollect, Signers: []models.SignerReference{{VerificationMethod: models.VerificationMethodDigitalCertificate}}}
		estimate := &models.EstimateAssignmentCostRequest{Method: models.MethodCollect, Signers: []models.EstimateAssignmentCostSigner{{VerificationMethod: models.VerificationMethodDigitalCertificate}}}

		_, createErr := resource.Create(context.Background(), "d1", create)
		_, legacyErr := resource.EstimateCost(context.Background(), "d1", create)
		_, estimateErr := resource.EstimateCostWithRequest(context.Background(), "d1", estimate)
		for name, err := range map[string]error{"create": createErr, "legacy estimate": legacyErr, "estimate": estimateErr} {
			if !errors.Is(err, sdkerrors.ErrDigitalCertificateRequiresVirtual) || !errors.Is(err, sdkerrors.ErrInvalidInput) {
				t.Errorf("%s: err = %v, want ErrDigitalCertificateRequiresVirtual and ErrInvalidInput", name, err)
			}
		}
	})

	// The API refuses a signer-less estimate in either mode, and omitempty would drop the key.
	t.Run("estimate without signers fails before the network", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			t.Error("request must not be sent")
		})
		resource := NewAssignmentResource(httpClient)

		for name, body := range map[string]any{
			"dedicated body": &models.EstimateAssignmentCostRequest{Method: models.MethodCollect,
				Entries: []any{map[string]any{"page_id": "p1"}}},
			"legacy body":   &models.CreateAssignmentRequest{Method: models.MethodVirtual},
			"nil dedicated": (*models.EstimateAssignmentCostRequest)(nil),
			"nil legacy":    (*models.CreateAssignmentRequest)(nil),
		} {
			var err error
			switch request := body.(type) {
			case *models.EstimateAssignmentCostRequest:
				_, err = resource.EstimateCostWithRequest(context.Background(), "d1", request)
			case *models.CreateAssignmentRequest:
				_, err = resource.EstimateCost(context.Background(), "d1", request)
			}
			if !errors.Is(err, sdkerrors.ErrInvalidInput) {
				t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
			}
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

func TestAssignmentsListWorkspaceSelection(t *testing.T) {
	for _, tc := range []struct {
		name, fallback, explicit, want string
	}{
		{"default workspace", "default", "", "default"},
		{"explicit workspace", "default", "other", "other"},
		{"OAuth workspace", "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				if r.Method != http.MethodGet || r.URL.Path != "/assignments" || q.Get("accountId") != tc.want || q.Has("account_id") {
					t.Errorf("request = %s %s", r.Method, r.URL)
				}
				if tc.want == "" && q.Has("accountId") {
					t.Error("empty workspace must be omitted for OAuth")
				}
				w.Header().Set("X-Pagination-Total-Count", "2")
				writeJSONResponse(w, `{"status":200,"data":[{"id":"assignment","document_id":"document"}]}`)
			})
			params := models.ListParams{AccountID: tc.explicit, Sort: "-created_at"}
			before := params
			got, err := NewAssignmentResource(h, tc.fallback).List(context.Background(), &params)
			if err != nil || len(got.Data) != 1 || got.Data[0].DocumentID != "document" || got.Pagination.TotalCount != 2 {
				t.Fatalf("List = %+v, %v", got, err)
			}
			if !reflect.DeepEqual(params, before) {
				t.Errorf("params changed: %+v", params)
			}
		})
	}
}

func TestResetExpirationRequestShapes(t *testing.T) {
	expiresAt := "2027-01-02T03:04:05Z"
	for _, tc := range []struct {
		name string
		call func(*AssignmentResource) error
		want map[string]any
	}{
		{"nil clears", func(r *AssignmentResource) error {
			_, err := r.ResetExpiration(context.Background(), "d1", "a1", nil)
			return err
		}, map[string]any{"expires_at": nil}},
		{"request nil clears", func(r *AssignmentResource) error {
			_, err := r.ResetExpirationWithRequest(context.Background(), "d1", "a1", models.ResetAssignmentExpirationRequest{})
			return err
		}, map[string]any{"expires_at": nil}},
		{"value", func(r *AssignmentResource) error {
			_, err := r.ResetExpiration(context.Background(), "d1", "a1", &expiresAt)
			return err
		}, map[string]any{"expires_at": expiresAt}},
		{"request value", func(r *AssignmentResource) error {
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
