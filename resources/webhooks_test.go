package resources

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/assinafy/golang-sdk/models"
)

func TestWebhooksListDispatchesOmitsEmptyEvent(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.URL.Query()["event"]; ok {
			t.Errorf("event param should be absent when unset, got %q", r.URL.Query().Get("event"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	r := NewWebhookResource(httpClient, "acc1")
	if _, err := r.ListDispatches(context.Background(), "", &models.WebhookDispatchListParams{PerPage: 5}); err != nil {
		t.Fatalf("ListDispatches: %v", err)
	}
}

func TestWebhookEndpointCanClearEvents(t *testing.T) {
	h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkJSONContract(t, w, r, map[string]any{"events": []string{}})
		writeJSONResponse(w, `{"status":200,"data":{"id":"endpoint","events":[]}}`)
	})
	got, err := NewWebhookResource(h, "account").UpdateEndpoint(context.Background(), "", "endpoint", &models.UpdateWebhookEndpointRequest{Events: []string{}})
	if err != nil || got == nil || got.Events == nil || len(got.Events) != 0 {
		t.Fatalf("clear events: result=%v, error=%v", got, err)
	}
}

func TestWebhooksListDispatchesDoesNotMutateParams(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	params := &models.WebhookDispatchListParams{}
	r := NewWebhookResource(httpClient, "acc1")
	if _, err := r.ListDispatches(context.Background(), "", params); err != nil {
		t.Fatalf("ListDispatches: %v", err)
	}
	if *params != (models.WebhookDispatchListParams{}) {
		t.Errorf("ListDispatches mutated params: %+v", params)
	}
}

func TestWebhooksListDispatchesEncodesParams(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("page") != "1" || q.Get("per-page") != "5" || q.Get("delivered") != "true" || q.Get("from") != "1700000000" {
			t.Errorf("query = %v", q)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	r := NewWebhookResource(httpClient, "acc1")
	delivered := true
	_, err := r.ListDispatches(context.Background(), "", &models.WebhookDispatchListParams{
		PerPage: 5, Delivered: &delivered, From: 1700000000,
	})
	if err != nil {
		t.Fatalf("ListDispatches: %v", err)
	}
}
