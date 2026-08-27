package resources

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/assinafy/golang-sdk/models"
)

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
			writeJSONResponse(w, `{"status":200,"data":{"type":"text","success":true,"error_message":""}}`)
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
			writeJSONResponse(w, `{"status":200,"data":[{"field_id":"f1","type":"text","success":true,"error_message":""}]}`)
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

func TestFieldsListWithFlags(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("include_inactive") != "true" || q.Get("include_standard") != "false" {
			t.Errorf("query = %v", q)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[{"id":"f1","name":"x","type":"text","is_active":true,"is_required":false,"is_standard":false,"is_read_only":false,"is_visible":true}]}`)
	})

	r := NewFieldResource(httpClient, "acc1")
	out, err := r.List(context.Background(), "", &models.ListFieldDefinitionsParams{IncludeInactive: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(out) != 1 || out[0].ID != "f1" {
		t.Errorf("got %+v", out)
	}
}
