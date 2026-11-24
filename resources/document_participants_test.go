package resources

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/internal"
	"github.com/assinafy/golang-sdk/v2/models"
)

func TestDocumentParticipantsWire(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name, method, path, data string
		call                     func(*internal.HTTPClient, string) (any, error)
	}{
		{"capacities", "GET", "/documents/doc%2F1/signer-capacities", `[{"signer_id":"signer","sign_as":"Witness"}]`, func(h *internal.HTTPClient, id string) (any, error) {
			return NewAssignmentResource(h).SignerCapacities(ctx, id)
		}},
		{"history", "GET", "/documents/doc%2F1/notification-history", `[{"signer_id":"signer","notification_history":[{"event":"signature_request","status":"opened","error_code":null,"error_message":null,"sent_at":"2026-11-24T12:00:00Z","failed_at":null,"delivered_at":"2026-11-24T12:00:05Z","opened_at":"2026-11-24T12:01:00Z"}]}]`, func(h *internal.HTTPClient, id string) (any, error) {
			return NewAssignmentResource(h).NotificationHistory(ctx, id)
		}},
		{"watchers", "GET", "/documents/doc%2F1/watchers", `[{"id":"signer","full_name":"Watcher","email":"signer@example.com"}]`, func(h *internal.HTTPClient, id string) (any, error) {
			return NewDocumentResource(h, "").ListWatchers(ctx, id)
		}},
		{"set watchers", "PUT", "/documents/doc%2F1/watchers", `[{"id":"signer","full_name":"Watcher","email":"signer@example.com"}]`, func(h *internal.HTTPClient, id string) (any, error) {
			return NewDocumentResource(h, "").SetWatchers(ctx, id, &models.SetDocumentWatchersRequest{WatcherIDs: []string{"signer"}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != tc.method || req.URL.EscapedPath() != tc.path || req.URL.RawQuery != "" || req.Header.Get("X-Api-Key") != "key" {
					t.Errorf("request = %s %s", req.Method, req.URL)
				}
				if req.Method == http.MethodPut {
					if got := decodeContractBody(t, req); !reflect.DeepEqual(got, map[string]any{"watcher_ids": []any{"signer"}}) {
						t.Errorf("body = %#v", got)
					}
				}
				writeJSONResponse(w, `{"status":200,"data":`+tc.data+`}`)
			})
			got, err := tc.call(h, "doc/1")
			if err != nil {
				t.Fatal(err)
			}
			switch v := got.(type) {
			case []models.DocumentSignerCapacity:
				if len(v) != 1 || v[0].SignerID != "signer" || v[0].SignAs != models.SignerCapacityWitness {
					t.Fatalf("capacities = %+v", v)
				}
			case []models.SignerNotificationHistory:
				if len(v) != 1 || v[0].SignerID != "signer" || len(v[0].NotificationHistory) != 1 {
					t.Fatalf("history = %+v", v)
				}
				n := v[0].NotificationHistory[0]
				if n.Status != "opened" || n.Event != "signature_request" || n.DeliveredAt == nil || n.OpenedAt == nil || n.SentAt == nil || n.ErrorCode != nil || n.FailedAt != nil {
					t.Fatalf("notification = %+v", n)
				}
			case []models.Signer:
				if len(v) != 1 || v[0].ID != "signer" || v[0].Email == nil || *v[0].Email != "signer@example.com" {
					t.Fatalf("watchers = %+v", v)
				}
			default:
				t.Fatalf("unexpected result %T", got)
			}
			h, _ = newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				writeJSONResponse(w, `{"status":403,"message":"Permission required"}`)
			})
			if _, err = tc.call(h, "doc"); !sdkerrors.IsStatusCode(err, 403) {
				t.Errorf("API error = %v", err)
			}
			if _, err = tc.call(h, ""); !errors.Is(err, sdkerrors.ErrInvalidInput) {
				t.Errorf("empty ID = %v", err)
			}
		})
	}
}

func TestClearDocumentWatchers(t *testing.T) {
	h, _ := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		if got := decodeContractBody(t, req); !reflect.DeepEqual(got, map[string]any{"watcher_ids": []any{}}) {
			t.Errorf("body = %#v", got)
		}
		writeJSONResponse(w, `{"status":200,"data":[]}`)
	})
	got, err := NewDocumentResource(h, "").SetWatchers(context.Background(), "doc", &models.SetDocumentWatchersRequest{WatcherIDs: []string{}})
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("clear = %#v, %v", got, err)
	}
}

func TestEnvelopeResendEstimate(t *testing.T) {
	h, _ := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || req.URL.EscapedPath() != "/accounts/default/envelopes/env%2F1/signers/signer%2F1/estimate-resend-cost" || req.Header.Get("X-Api-Key") != "key" || req.ContentLength != 0 {
			t.Errorf("request = %s %s", req.Method, req.URL)
		}
		writeJSONResponse(w, `{"status":200,"data":{"total":0,"credit_balance":null,"has_sufficient_credits":true,"breakdown":[]}}`)
	})
	r := NewEnvelopeResource(h, "default")
	got, err := r.EstimateResendCost(context.Background(), "", "env/1", "signer/1")
	if err != nil || !got.CreditBalanceUnlimited || !got.HasSufficientCredits || got.Total != 0 {
		t.Fatalf("estimate = %+v, %v", got, err)
	}
	if _, err := r.EstimateResendCost(context.Background(), "", "env", ""); !errors.Is(err, sdkerrors.ErrInvalidInput) {
		t.Errorf("empty signer = %v", err)
	}
}
