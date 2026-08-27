package resources

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/assinafy/golang-sdk/models"
)

func TestTemplatesListEncodesTags(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("tags"); got != "t1,t2" {
			t.Errorf("tags = %q, want t1,t2", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	r := NewTemplateResource(httpClient, "acc1")
	if _, err := r.List(context.Background(), "", &models.ListParams{Tags: []string{"t1", "t2"}}); err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestTemplatesListEncodesStatusFilter(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("status"); got != "ready" {
			t.Errorf("status = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	r := NewTemplateResource(httpClient, "acc1")
	_, err := r.List(context.Background(), "", &models.ListParams{Status: "ready"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
}
