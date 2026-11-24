package resources

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/models"
)

const scheduledSendData = `{"resource":"scheduled_send","id":"send","type":"assignment","document_ids":["doc"],"status":"pending","send_at":"2026-12-01T12:00:00Z","assignment_id":null,"envelope_id":null,"failure_reason":null,"created_at":"2026-11-24T12:00:00Z","processed_at":null}`

func TestScheduledSendsWire(t *testing.T) {
	ctx := context.Background()
	zero := 0.0
	expiry := "2026-12-02T12:00:00Z"
	for _, tc := range []struct {
		name, method, path, body string
		call                     func(*ScheduledSendResource) (*models.ScheduledSend, error)
	}{
		{"assignment", "POST", "/documents/doc%2F1/scheduled-sends", `{"method":"virtual","signers":[{"id":"signer","sign_as":"Witness"}],"max_total_credits":0,"expires_at":"2026-12-02T12:00:00Z","send_at":"2026-12-01T12:00:00Z"}`, func(r *ScheduledSendResource) (*models.ScheduledSend, error) {
			return r.CreateAssignment(ctx, "doc/1", &models.ScheduleAssignmentRequest{CreateAssignmentRequest: models.CreateAssignmentRequest{Method: models.MethodVirtual, Signers: []models.SignerReference{{ID: "signer", SignAs: models.SignerCapacityWitness}}, MaxTotalCredits: &zero, ExpiresAt: &expiry}, SendAt: "2026-12-01T12:00:00Z"})
		}},
		{"envelope", "POST", "/accounts/default/scheduled-sends", `{"name":"Batch","signers":[{"id":"signer"}],"documents":[{"id":"doc","signer_ids":["signer"]}],"max_total_credits":0,"send_at":1796126400}`, func(r *ScheduledSendResource) (*models.ScheduledSend, error) {
			return r.CreateEnvelope(ctx, "", &models.ScheduleEnvelopeRequest{CreateEnvelopeRequest: models.CreateEnvelopeRequest{Name: "Batch", Signers: []models.EnvelopeSigner{{ID: "signer"}}, Documents: []models.EnvelopeDocumentRequest{{ID: "doc", SignerIDs: []string{"signer"}}}, MaxTotalCredits: &zero, IdempotencyKey: "ignored"}, SendAt: int64(1796126400)})
		}},
		{"send", "PUT", "/accounts/account%2F1/scheduled-sends/send%2F1/send", "", func(r *ScheduledSendResource) (*models.ScheduledSend, error) {
			return r.SendNow(ctx, "account/1", "send/1")
		}},
		{"cancel", "PUT", "/accounts/default/scheduled-sends/send%2F1/cancel", "", func(r *ScheduledSendResource) (*models.ScheduledSend, error) { return r.Cancel(ctx, "", "send/1") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != tc.method || req.URL.EscapedPath() != tc.path || req.URL.RawQuery != "" || req.Header.Get("X-Api-Key") != "key" || req.Header.Get("Idempotency-Key") != "" {
					t.Errorf("request = %s %s", req.Method, req.URL)
				}
				if tc.body != "" {
					var want any
					if err := json.Unmarshal([]byte(tc.body), &want); err != nil {
						t.Fatal(err)
					}
					if got := decodeContractBody(t, req); !reflect.DeepEqual(got, want) {
						t.Errorf("body = %#v, want %#v", got, want)
					}
				} else if req.ContentLength != 0 {
					t.Error("unexpected body")
				}
				writeJSONResponse(w, `{"status":200,"data":`+scheduledSendData+`}`)
			})
			got, err := tc.call(NewScheduledSendResource(h, "default"))
			if err != nil {
				t.Fatal(err)
			}
			if got.ID != "send" || got.Type != "assignment" || got.Status != "pending" || !reflect.DeepEqual(got.DocumentIDs, []string{"doc"}) || got.AssignmentID != nil || got.EnvelopeID != nil || got.FailureReason != nil || got.ProcessedAt != nil || got.SendAt != "2026-12-01T12:00:00Z" || got.CreatedAt == "" {
				t.Fatalf("result = %+v", got)
			}
		})
	}
}

func TestScheduledSendsList(t *testing.T) {
	for _, params := range []*models.ScheduledSendListParams{nil, {}, {Page: 2, PerPage: 10, DocumentID: "doc/1"}} {
		h, _ := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
			want := ""
			if params != nil && params.Page == 2 {
				want = "document_id=doc%2F1&page=2&per-page=10"
			}
			if req.Method != http.MethodGet || req.URL.Path != "/accounts/default/scheduled-sends" || req.URL.RawQuery != want || req.Header.Get("X-Api-Key") != "key" {
				t.Errorf("request = %s %s", req.Method, req.URL)
			}
			w.Header().Set("X-Pagination-Total-Count", "21")
			writeJSONResponse(w, `{"status":200,"data":[`+scheduledSendData+`]}`)
		})
		got, err := NewScheduledSendResource(h, "default").List(context.Background(), "", params)
		if err != nil {
			t.Fatal(err)
		}
		if got.Pagination.TotalCount != 21 || len(got.Data) != 1 || got.Data[0].ID != "send" {
			t.Fatalf("result = %+v", got)
		}
	}
}

func TestScheduledSendsErrors(t *testing.T) {
	ctx := context.Background()
	calls := []func(*ScheduledSendResource, string) error{
		func(r *ScheduledSendResource, id string) error {
			_, err := r.CreateAssignment(ctx, id, nil)
			return err
		},
		func(r *ScheduledSendResource, id string) error { _, err := r.CreateEnvelope(ctx, id, nil); return err },
		func(r *ScheduledSendResource, id string) error { _, err := r.List(ctx, id, nil); return err },
		func(r *ScheduledSendResource, id string) error { _, err := r.SendNow(ctx, id, "send"); return err },
		func(r *ScheduledSendResource, id string) error { _, err := r.Cancel(ctx, id, "send"); return err },
	}
	h, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		writeJSONResponse(w, `{"status":409,"code":"CreditCeilingExceeded","total_credits":1,"max_total_credits":0,"message":"Ceiling exceeded"}`)
	})
	r := NewScheduledSendResource(h, "")
	for i, call := range calls {
		var apiErr *sdkerrors.APIError
		if err := call(r, "id"); !errors.As(err, &apiErr) || apiErr.Code != "CreditCeilingExceeded" || apiErr.MaxTotalCredits == nil || *apiErr.MaxTotalCredits != 0 {
			t.Errorf("call %d error = %v", i, err)
		}
		for _, id := range []string{"", "..", " "} {
			if err := call(r, id); !errors.Is(err, sdkerrors.ErrInvalidInput) {
				t.Errorf("call %d invalid path = %v", i, err)
			}
		}
	}
	for _, call := range []func() error{
		func() error { _, err := r.SendNow(ctx, "account", ""); return err },
		func() error { _, err := r.Cancel(ctx, "account", ".."); return err },
	} {
		if err := call(); !errors.Is(err, sdkerrors.ErrInvalidInput) {
			t.Errorf("invalid send ID = %v", err)
		}
	}
}
