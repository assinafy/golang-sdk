package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/assinafy/golang-sdk/models"
)

func TestResetExpirationRequestShapes(t *testing.T) {
	expiresAt := "2027-01-02T03:04:05Z"
	for _, tc := range []struct {
		name string
		call func(*AssignmentResource) error
		want map[string]any
	}{
		{"legacy null", func(r *AssignmentResource) error {
			_, err := r.ResetExpiration(context.Background(), "d1", "a1", nil)
			return err
		}, map[string]any{"expires_at": nil}},
		{"exact omitted", func(r *AssignmentResource) error {
			_, err := r.ResetExpirationWithRequest(context.Background(), "d1", "a1", models.ResetAssignmentExpirationRequest{})
			return err
		}, map[string]any{}},
		{"exact value", func(r *AssignmentResource) error {
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
				writeContractResponse(w, `{"status":200,"data":{"id":"a1","method":"virtual"}}`)
			})
			if err := tc.call(NewAssignmentResource(httpClient)); err != nil {
				t.Fatal(err)
			}
		})
	}
}
