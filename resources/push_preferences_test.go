package resources

import (
	"context"
	"net/http"
	"testing"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/models"
)

func TestUserPushPreferences(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != method || r.URL.Path != "/users/self/push-preferences" || r.Header.Get("X-Api-Key") != "key" || r.URL.RawQuery != "" {
					t.Errorf("request = %s %s", r.Method, r.URL)
				}
				if method == http.MethodPut {
					checkJSONContract(t, w, r, map[string]any{"SignerViewed": false})
				}
				writeJSONResponse(w, `{"status":200,"data":{"SignerViewed":false,"SignerSigned":true,"FutureCode":true}}`)
			})
			r := NewUserResource(h)
			var got models.PushPreferences
			var err error
			if method == http.MethodGet {
				got, err = r.GetPushPreferences(context.Background())
			} else {
				got, err = r.UpdatePushPreferences(context.Background(), models.PushPreferences{models.NotificationPreferenceSignerViewed: false})
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 3 || got[models.NotificationPreferenceSignerViewed] || !got[models.NotificationPreferenceSignerSigned] || !got["FutureCode"] {
				t.Fatalf("push preferences = %v", got)
			}
		})
	}
}

func TestUserPushPreferenceErrors(t *testing.T) {
	h, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		writeJSONResponse(w, `{"status":403,"message":"Forbidden"}`)
	})
	r := NewUserResource(h)
	if _, err := r.GetPushPreferences(context.Background()); !sdkerrors.IsStatusCode(err, http.StatusForbidden) {
		t.Fatalf("get: %v", err)
	}
	if _, err := r.UpdatePushPreferences(context.Background(), nil); !sdkerrors.IsStatusCode(err, http.StatusForbidden) {
		t.Fatalf("update: %v", err)
	}
}
