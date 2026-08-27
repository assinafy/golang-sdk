package resources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/assinafy/golang-sdk/models"
)

func TestTagsCreateEncodesBody(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/accounts/acc1/tags" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if body["name"] != "Contracts" || body["color"] != "ff8800" {
			t.Errorf("body = %v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":{"resource":"tag","id":"t1","name":"Contracts","color":"ff8800"}}`)
	})

	color := "ff8800"
	r := NewTagResource(httpClient, "acc1")
	tag, err := r.Create(context.Background(), "", &models.CreateTagRequest{Name: "Contracts", Color: &color})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if tag.ID != "t1" || tag.Color == nil || *tag.Color != "ff8800" {
		t.Errorf("tag = %+v", tag)
	}
}

func TestTagsListEncodesSearch(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("search"); got != "contract" {
			t.Errorf("search = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[{"id":"t1","name":"Contracts"}]}`)
	})

	r := NewTagResource(httpClient, "acc1")
	tags, err := r.List(context.Background(), "", "contract")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tags) != 1 || tags[0].ID != "t1" {
		t.Errorf("tags = %+v", tags)
	}
}

func TestTagsDeleteSendsForce(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/accounts/acc1/tags/t1" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("force"); got != "true" {
			t.Errorf("force = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":{"deleted":true}}`)
	})

	r := NewTagResource(httpClient, "acc1")
	result, err := r.DeleteWithResult(context.Background(), "", "t1", true)
	if err != nil {
		t.Fatalf("DeleteWithResult: %v", err)
	}
	if !result.Deleted {
		t.Errorf("result = %+v", result)
	}
}
