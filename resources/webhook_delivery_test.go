package resources

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/models"
)

func TestWebhookSecretRotationGrace(t *testing.T) {
	for _, body := range []*models.RotateWebhookSecretRequest{nil, {}, {KeepPrevious: true}} {
		h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.EscapedPath() != "/accounts/acc%2F1/webhooks/endpoints/ep%2F1/secret/rotate" || r.Header.Get("X-Api-Key") != "key" || r.URL.RawQuery != "" {
				t.Errorf("request = %s %s", r.Method, r.URL)
			}
			if body != nil {
				checkJSONContract(t, w, r, map[string]any{"keep_previous": body.KeepPrevious})
			} else {
				data, err := io.ReadAll(r.Body)
				if err != nil || len(data) != 0 {
					t.Errorf("nil request body = %s, error %v", data, err)
				}
			}
			if body != nil && body.KeepPrevious {
				writeJSONResponse(w, `{"status":200,"data":{"secret":"whsec_new","previous_secret_expires_at":"2026-11-12T12:00:00Z"}}`)
			} else {
				writeJSONResponse(w, `{"status":200,"data":{"secret":"whsec_new"}}`)
			}
		})
		got, err := NewWebhookResource(h, "acc/1").RotateEndpointSecretWithRequest(context.Background(), "", "ep/1", body)
		if err != nil {
			t.Fatal(err)
		}
		if got.Secret != "whsec_new" {
			t.Fatalf("secret = %q", got.Secret)
		}
		if body != nil && body.KeepPrevious {
			if got.PreviousSecretExpiresAt == nil || *got.PreviousSecretExpiresAt != "2026-11-12T12:00:00Z" {
				t.Fatalf("expiry = %v", got.PreviousSecretExpiresAt)
			}
		} else if got.PreviousSecretExpiresAt != nil {
			t.Fatalf("expiry = %v", got.PreviousSecretExpiresAt)
		}
	}
}

func TestWebhookTestEndpoint(t *testing.T) {
	h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.EscapedPath() != "/accounts/acc%2F1/webhooks/endpoints/ep%2F1/test" || r.Header.Get("X-Api-Key") != "key" || r.URL.RawQuery != "" {
			t.Errorf("request = %s %s", r.Method, r.URL)
		}
		writeJSONResponse(w, `{"status":200,"data":{"id":"attempt","event":"webhook_test","delivered":false,"kind":"manual","will_retry":false,"duration_ms":0,"http_status":503,"payload":null}}`)
	})
	got, err := NewWebhookResource(h, "acc/1").TestEndpoint(context.Background(), "", "ep/1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "attempt" || got.Event != "webhook_test" || got.Kind != "manual" || got.Delivered || got.WillRetry || got.DurationMS == nil || *got.DurationMS != 0 || got.HTTPStatus == nil || *got.HTTPStatus != 503 || got.Payload != nil {
		t.Fatalf("dispatch = %+v", got)
	}
}

func TestWebhookDeliveryErrors(t *testing.T) {
	h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Header().Set("Retry-After", "60")
		writeJSONResponse(w, `{"status":429,"message":"Rate limited"}`)
	})
	r := NewWebhookResource(h, "account")
	ctx := context.Background()
	for name, call := range map[string]func(string) error{
		"rotation": func(id string) error {
			_, err := r.RotateEndpointSecretWithRequest(ctx, "", id, &models.RotateWebhookSecretRequest{KeepPrevious: true})
			return err
		},
		"rotation compatibility": func(id string) error { _, err := r.RotateEndpointSecret(ctx, "", id); return err },
		"test":                   func(id string) error { _, err := r.TestEndpoint(ctx, "", id); return err },
	} {
		if err := call("endpoint"); !sdkerrors.IsStatusCode(err, http.StatusTooManyRequests) {
			t.Errorf("%s: %v", name, err)
		}
		if err := call(""); !errors.Is(err, sdkerrors.ErrInvalidInput) {
			t.Errorf("%s blank id: %v", name, err)
		}
	}
}
