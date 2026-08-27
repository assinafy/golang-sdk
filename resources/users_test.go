package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/assinafy/golang-sdk/models"
)

func TestUserGetSelf(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/users/self" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		writeJSONResponse(w, `{"status":200,"data":{"id":"u1","name":"Ada","email":"ada@example.com"}}`)
	})

	user, err := NewUserResource(httpClient).GetSelf(context.Background())
	if err != nil {
		t.Fatalf("GetSelf: %v", err)
	}
	if user.ID != "u1" || user.Email != "ada@example.com" {
		t.Errorf("user = %+v", user)
	}
}

func TestUserGetSelfAcceptsSandboxSessionShape(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSONResponse(w, `{"status":200,"data":{"user":{"id":"u1","name":"Ada","email":"ada@example.com"},"accounts":[]}}`)
	})

	user, err := NewUserResource(httpClient).GetSelf(context.Background())
	if err != nil {
		t.Fatalf("GetSelf: %v", err)
	}
	if user.ID != "u1" || user.Email != "ada@example.com" {
		t.Errorf("user = %+v", user)
	}
}

func TestUserStatsEncodesParamsWithoutMutation(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/users/self/stats" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		if got := r.URL.Query(); got.Get("granularity") != "monthly" || got.Has("month") || len(got) != 1 {
			t.Errorf("query = %v", got)
		}
		writeJSONResponse(w, `{"status":200,"data":[{"period":"2026-06","signature_requests":3}]}`)
	})

	params := models.StatsParams{Granularity: models.StatsGranularityMonthly}
	wantParams := params
	rows, err := NewUserResource(httpClient).Stats(context.Background(), &params)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if !reflect.DeepEqual(params, wantParams) {
		t.Errorf("params mutated: got %+v want %+v", params, wantParams)
	}
	if len(rows) != 1 || rows[0].SignatureRequests != 3 {
		t.Errorf("rows = %+v", rows)
	}
}

func TestUserGetNotificationPreferences(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/users/self/notification-preferences" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		writeJSONResponse(w, `{"status":200,"data":{"DocumentCompleted":true,"SignerDeclined":false}}`)
	})

	prefs, err := NewUserResource(httpClient).GetNotificationPreferences(context.Background())
	if err != nil {
		t.Fatalf("GetNotificationPreferences: %v", err)
	}
	if !prefs[models.NotificationPreferenceDocumentCompleted] || prefs[models.NotificationPreferenceSignerDeclined] {
		t.Errorf("preferences = %v", prefs)
	}
}

func TestUserUpdateNotificationPreferences(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/users/self/notification-preferences" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		var body models.NotificationPreferences
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if len(body) != 1 || body[models.NotificationPreferenceDocumentExpired] {
			t.Errorf("body = %v", body)
		}
		writeJSONResponse(w, `{"status":200,"data":{"DocumentCompleted":true,"DocumentExpired":false}}`)
	})

	changes := models.NotificationPreferences{models.NotificationPreferenceDocumentExpired: false}
	prefs, err := NewUserResource(httpClient).UpdateNotificationPreferences(context.Background(), changes)
	if err != nil {
		t.Fatalf("UpdateNotificationPreferences: %v", err)
	}
	if !prefs[models.NotificationPreferenceDocumentCompleted] || prefs[models.NotificationPreferenceDocumentExpired] {
		t.Errorf("preferences = %v", prefs)
	}
}
