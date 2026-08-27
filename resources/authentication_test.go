package resources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/assinafy/golang-sdk/models"
)

func TestAuthenticationLinkSocialLogin(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/auth/link-social-login" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if body["provider"] != "google" || body["token"] != "tok" {
			t.Errorf("body = %v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"message":""}`)
	})

	r := NewAuthenticationResource(httpClient)
	if err := r.LinkSocialLogin(context.Background(), &models.LinkSocialLoginRequest{Provider: "google", Token: "tok"}); err != nil {
		t.Fatalf("LinkSocialLogin: %v", err)
	}
}

func TestAuthenticationSocialLoginURL(t *testing.T) {
	httpClient, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
	r := NewAuthenticationResource(httpClient)
	got := r.SocialLoginURL("google")
	want := srv.URL + "/auth/authenticate?authclient=google"
	if got != want {
		t.Errorf("SocialLoginURL = %q, want %q", got, want)
	}
	if bare := r.SocialLoginURL(""); bare != srv.URL+"/auth/authenticate" {
		t.Errorf("SocialLoginURL(\"\") = %q", bare)
	}
}
