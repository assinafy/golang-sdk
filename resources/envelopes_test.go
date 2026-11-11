package resources

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/models"
)

const envelopeData = `{"resource":"envelope","id":"env","name":"Contract","message":null,"documents":[{"id":"doc","position":1,"name":"contract.pdf","status":"pending_signature","artifacts":{"original":"https://example.test/original"},"signers":[{"id":"signer","full_name":"Signer","email":null,"whatsapp_phone_number":null,"completed":false,"signed_at":null}]}],"counts":{"pending_signature":1},"signing_urls":[{"signer_id":"signer","url":"https://example.test/sign"}],"created_at":"2026-11-01T12:00:00Z"}`

func TestEnvelopesWire(t *testing.T) {
	ctx := context.Background()
	zero := 0.0
	create := &models.CreateEnvelopeRequest{
		Name: "Contract", IdempotencyKey: "request-1", MaxTotalCredits: &zero,
		Signers:   []models.EnvelopeSigner{{ID: "signer", VerificationMethod: "Email", NotificationMethods: []string{"Email"}, Embedded: true}},
		Documents: []models.EnvelopeDocumentRequest{{ID: "doc", Method: models.MethodVirtual, SignerIDs: []string{"signer"}, ExpiresAt: "2027-01-01T12:00:00Z", CopyReceiverIDs: []string{"copy"}}},
	}
	createJSON := `{"name":"Contract","max_total_credits":0,"signers":[{"id":"signer","verification_method":"Email","notification_methods":["Email"],"embedded":true}],"documents":[{"id":"doc","method":"virtual","signer_ids":["signer"],"expires_at":"2027-01-01T12:00:00Z","copy_receiver_ids":["copy"]}]}`
	for _, tc := range []struct {
		name, method, suffix, body, data, key string
		call                                  func(*EnvelopeResource) (any, error)
	}{
		{"create", "POST", "", createJSON, envelopeData, "request-1", func(r *EnvelopeResource) (any, error) { return r.Create(ctx, "account/1", create) }},
		{"estimate", "POST", "/estimate-cost", createJSON, `{"documents":1,"credits":0,"total_credits":0,"has_sufficient_resources":true}`, "", func(r *EnvelopeResource) (any, error) { return r.EstimateCost(ctx, "account/1", create) }},
		{"get", "GET", "/env%2F1", "", envelopeData, "", func(r *EnvelopeResource) (any, error) { return r.Get(ctx, "account/1", "env/1") }},
		{"update", "PATCH", "/env%2F1", `{"message":null,"document_ids":["doc"]}`, envelopeData, "", func(r *EnvelopeResource) (any, error) {
			return r.Update(ctx, "account/1", "env/1", &models.UpdateEnvelopeRequest{Message: &models.Nullable[string]{}, DocumentIDs: []string{"doc"}})
		}},
		{"delete", "DELETE", "/env%2F1", "", `[]`, "", func(r *EnvelopeResource) (any, error) { return nil, r.Delete(ctx, "account/1", "env/1") }},
		{"cancel", "PUT", "/env%2F1/cancel", "", `[{"id":"doc","status":"rejected_by_user","cancelled":true}]`, "", func(r *EnvelopeResource) (any, error) { return r.Cancel(ctx, "account/1", "env/1") }},
		{"resend", "PUT", "/env%2F1/signers/signer%2F1/resend", "", `{"is_sent":true,"envelope_id":"env","signer_id":"signer"}`, "", func(r *EnvelopeResource) (any, error) {
			return r.ResendNotification(ctx, "account/1", "env/1", "signer/1")
		}},
		{"add", "POST", "/env%2F1/documents", `{"id":"doc","signer_ids":["signer"],"expires_at":0,"max_total_credits":0}`, envelopeData, "", func(r *EnvelopeResource) (any, error) {
			return r.AddDocument(ctx, "account/1", "env/1", &models.AddEnvelopeDocumentRequest{EnvelopeDocumentRequest: models.EnvelopeDocumentRequest{ID: "doc", SignerIDs: []string{"signer"}, ExpiresAt: 0}, MaxTotalCredits: &zero})
		}},
		{"remove", "DELETE", "/env%2F1/documents/doc%2F1", "", envelopeData, "", func(r *EnvelopeResource) (any, error) { return r.RemoveDocument(ctx, "account/1", "env/1", "doc/1") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != tc.method || req.URL.EscapedPath() != "/accounts/account%2F1/envelopes"+tc.suffix || req.URL.RawQuery != "" || req.Header.Get("X-Api-Key") != "key" || req.Header.Get("Idempotency-Key") != tc.key {
					t.Errorf("request = %s %s, idempotency key %q", req.Method, req.URL, req.Header.Get("Idempotency-Key"))
				}
				if tc.body != "" {
					var want any
					if err := json.Unmarshal([]byte(tc.body), &want); err != nil {
						t.Fatal(err)
					}
					if got := decodeContractBody(t, req); !reflect.DeepEqual(got, want) {
						t.Errorf("body = %#v, want %#v", got, want)
					}
				}
				writeJSONResponse(w, `{"status":200,"data":`+tc.data+`}`)
			})
			got, err := tc.call(NewEnvelopeResource(h, "default"))
			if err != nil {
				t.Fatal(err)
			}
			switch result := got.(type) {
			case *models.DocumentEnvelope:
				if result.ID != "env" || len(result.Documents) != 1 || result.Counts["pending_signature"] != 1 || result.Documents[0].Signers[0].Completed || result.Documents[0].Signers[0].SignedAt != nil || result.SigningURLs[0].SignerID != "signer" {
					t.Fatalf("result = %+v", result)
				}
			case *models.CostEstimate:
				if result.Documents != 1 || !result.HasSufficientResources {
					t.Fatalf("result = %+v", result)
				}
			case []models.EnvelopeCancellation:
				if len(result) != 1 || !result[0].Cancelled || result[0].Status != models.StatusRejectedByUser {
					t.Fatalf("result = %+v", result)
				}
			case *models.EnvelopeResendResult:
				if !result.IsSent || result.EnvelopeID != "env" || result.SignerID != "signer" {
					t.Fatalf("result = %+v", result)
				}
			}
		})
	}
}

func TestEnvelopesPagination(t *testing.T) {
	for _, activities := range []bool{false, true} {
		for _, params := range []*models.ListParams{nil, {Page: 2, PerPage: 10, Search: "Contract", Status: "open", Sort: "id"}} {
			h, _ := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
				path := "/accounts/default/envelopes"
				want := url.Values{"page": {"1"}, "per-page": {"25"}}
				data := "[" + envelopeData + "]"
				if activities {
					path += "/env/activities"
					want = url.Values{}
					data = `[{"id":7,"event":"document_created","message":"Created","payload":[],"document_id":"doc","document_name":"contract.pdf","created_at":"2026-11-01T12:00:00Z"}]`
				}
				if params != nil {
					want = url.Values{"page": {"2"}, "per-page": {"10"}}
					if activities {
						want.Set("sort", "id")
					} else {
						want.Set("search", "Contract")
						want.Set("status", "open")
					}
				}
				if req.Method != http.MethodGet || req.URL.Path != path || !reflect.DeepEqual(req.URL.Query(), want) || req.Header.Get("X-Api-Key") != "key" {
					t.Errorf("request = %s %s, want query %v", req.Method, req.URL, want)
				}
				w.Header().Set("X-Pagination-Total-Count", "12")
				writeJSONResponse(w, `{"status":200,"data":`+data+`}`)
			})
			r := NewEnvelopeResource(h, "default")
			if activities {
				got, err := r.Activities(context.Background(), "", "env", params)
				if err != nil {
					t.Fatal(err)
				}
				if got.Pagination.TotalCount != 12 || len(got.Data) != 1 || got.Data[0].DocumentID != "doc" || got.Data[0].ID != 7 || got.Data[0].Payload != nil {
					t.Fatalf("result = %+v", got)
				}
			} else {
				got, err := r.List(context.Background(), "", params)
				if err != nil {
					t.Fatal(err)
				}
				if got.Pagination.TotalCount != 12 || len(got.Data) != 1 || got.Data[0].ID != "env" {
					t.Fatalf("result = %+v", got)
				}
			}
		}
	}
}

func TestEnvelopeDownload(t *testing.T) {
	h, _ := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.EscapedPath() != "/accounts/default/envelopes/env%2F1/download/original" || req.Header.Get("X-Api-Key") != "key" {
			t.Errorf("request = %s %s", req.Method, req.URL)
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write([]byte("PK\x03\x04"))
	})
	got, err := NewEnvelopeResource(h, "default").Download(context.Background(), "", "env/1", "original")
	if err != nil || string(got) != "PK\x03\x04" {
		t.Fatalf("download = %q, %v", got, err)
	}
}

func TestEnvelopeCollectAndPlanRestriction(t *testing.T) {
	body := &models.CreateEnvelopeRequest{
		Name: "Fields", Signers: []models.EnvelopeSigner{{ID: "signer"}},
		Documents: []models.EnvelopeDocumentRequest{{
			ID: "doc", Method: models.MethodCollect, SignerIDs: []string{"signer"},
			Entries: []models.AssignmentEntry{{PageID: "page", Fields: []models.AssignmentField{{
				SignerID: "signer", FieldID: "field", DisplaySettings: models.DisplaySettings{Left: 0, Top: 0, Width: 200, Height: 50, FontSize: 14},
			}}}},
		}},
	}
	h, _ := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Idempotency-Key") != "" {
			t.Error("unexpected idempotency key")
		}
		var want any
		if err := json.Unmarshal([]byte(`{"name":"Fields","signers":[{"id":"signer"}],"documents":[{"id":"doc","method":"collect","signer_ids":["signer"],"entries":[{"page_id":"page","fields":[{"signer_id":"signer","field_id":"field","display_settings":{"left":0,"top":0,"width":200,"height":50,"fontSize":14}}]}]}]}`), &want); err != nil {
			t.Fatal(err)
		}
		if got := decodeContractBody(t, req); !reflect.DeepEqual(got, want) {
			t.Errorf("body = %#v, want %#v", got, want)
		}
		w.WriteHeader(http.StatusForbidden)
		writeJSONResponse(w, `{"status":403,"code":"PlanRestricted","message":"Envelopes feature required"}`)
	})
	got, err := NewEnvelopeResource(h, "default").Create(context.Background(), "", body)
	var apiErr *sdkerrors.APIError
	if got != nil || !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden || apiErr.Code != "PlanRestricted" {
		t.Fatalf("result = %v, error = %v", got, err)
	}
}

func TestEnvelopesErrorsAndPaths(t *testing.T) {
	ctx := context.Background()
	calls := []func(*EnvelopeResource, string) error{
		func(r *EnvelopeResource, account string) error { _, err := r.List(ctx, account, nil); return err },
		func(r *EnvelopeResource, account string) error { _, err := r.Create(ctx, account, nil); return err },
		func(r *EnvelopeResource, account string) error {
			_, err := r.EstimateCost(ctx, account, nil)
			return err
		},
		func(r *EnvelopeResource, account string) error { _, err := r.Get(ctx, account, "env"); return err },
		func(r *EnvelopeResource, account string) error {
			_, err := r.Update(ctx, account, "env", nil)
			return err
		},
		func(r *EnvelopeResource, account string) error { return r.Delete(ctx, account, "env") },
		func(r *EnvelopeResource, account string) error {
			_, err := r.Activities(ctx, account, "env", nil)
			return err
		},
		func(r *EnvelopeResource, account string) error {
			_, err := r.Download(ctx, account, "env", "original")
			return err
		},
		func(r *EnvelopeResource, account string) error { _, err := r.Cancel(ctx, account, "env"); return err },
		func(r *EnvelopeResource, account string) error {
			_, err := r.ResendNotification(ctx, account, "env", "signer")
			return err
		},
		func(r *EnvelopeResource, account string) error {
			_, err := r.AddDocument(ctx, account, "env", nil)
			return err
		},
		func(r *EnvelopeResource, account string) error {
			_, err := r.RemoveDocument(ctx, account, "env", "doc")
			return err
		},
	}
	h, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		writeJSONResponse(w, `{"status":409,"code":"EnvelopeOpen","message":"Cancel first"}`)
	})
	for i, call := range calls {
		err := call(NewEnvelopeResource(h, ""), "account")
		var apiErr *sdkerrors.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict || apiErr.Code != "EnvelopeOpen" {
			t.Errorf("operation %d error = %v", i, err)
		}
		if err := call(NewEnvelopeResource(h, ""), ""); !errors.Is(err, sdkerrors.ErrInvalidInput) {
			t.Errorf("operation %d empty account = %v", i, err)
		}
	}
	r := NewEnvelopeResource(h, "account")
	for _, call := range []func() error{
		func() error { _, err := r.Get(ctx, "", ".."); return err },
		func() error { _, err := r.ResendNotification(ctx, "", "env", ""); return err },
		func() error { _, err := r.RemoveDocument(ctx, "", "env", " "); return err },
		func() error { _, err := r.Download(ctx, "", "env", ".."); return err },
	} {
		if err := call(); !errors.Is(err, sdkerrors.ErrInvalidInput) {
			t.Errorf("invalid path error = %v", err)
		}
	}
}
