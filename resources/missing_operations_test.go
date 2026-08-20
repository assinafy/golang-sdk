package resources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/assinafy/golang-sdk/models"
)

func writeMissingOperationJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, body)
}

func TestAccountCreate(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/accounts" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		var body models.CreateAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if body.Name != "Acme" || body.NotificationSenderType == nil || *body.NotificationSenderType != models.NotificationSenderAccount {
			t.Errorf("body = %+v", body)
		}
		writeMissingOperationJSON(w, `{"status":200,"data":{"id":"a1","name":"Acme","notification_sender_type":"Account"}}`)
	})

	sender := models.NotificationSenderAccount
	account, err := NewAccountResource(httpClient, "").Create(context.Background(), &models.CreateAccountRequest{
		Name: "Acme", NotificationSenderType: &sender,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if account.ID != "a1" || account.NotificationSenderType != sender {
		t.Errorf("account = %+v", account)
	}
}

func TestAccountDeleteSendsForceBody(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/accounts/a1" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		var body map[string]bool
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if len(body) != 1 || !body["force"] {
			t.Errorf("body = %v", body)
		}
		writeMissingOperationJSON(w, `{"status":200,"data":[]}`)
	})

	if err := NewAccountResource(httpClient, "a1").Delete(context.Background(), "", true); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestAccountUploadLogo(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/accounts/a1/logo" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
			http.Error(w, "invalid multipart form", http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Errorf("FormFile: %v", err)
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		defer func() { _ = file.Close() }()
		content, err := io.ReadAll(file)
		if err != nil {
			t.Errorf("read file: %v", err)
			http.Error(w, "failed to read file", http.StatusBadRequest)
			return
		}
		if header.Filename != "logo.png" || string(content) != "png" || len(r.MultipartForm.Value) != 0 {
			t.Errorf("filename/content/fields = %q %q %v", header.Filename, content, r.MultipartForm.Value)
		}
		writeMissingOperationJSON(w, `{"status":200,"message":"Logo updated"}`)
	})

	if err := NewAccountResource(httpClient, "a1").UploadLogo(context.Background(), "", []byte("png"), "logo.png"); err != nil {
		t.Fatalf("UploadLogo: %v", err)
	}
}

func TestAccountDeleteLogo(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/accounts/a1/logo" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		writeMissingOperationJSON(w, `{"status":200,"message":"Logo deleted"}`)
	})

	if err := NewAccountResource(httpClient, "a1").DeleteLogo(context.Background(), ""); err != nil {
		t.Fatalf("DeleteLogo: %v", err)
	}
}

func TestAccountStatsEncodesParamsWithoutMutation(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/accounts/a1/stats" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		if got := r.URL.Query(); got.Get("granularity") != "daily" || got.Get("month") != "2026-06" || len(got) != 2 {
			t.Errorf("query = %v", got)
		}
		writeMissingOperationJSON(w, `{"status":200,"data":[{"period":"2026-06-01","documents_uploaded":2,"documents_certified":1}]}`)
	})

	params := models.StatsParams{Granularity: models.StatsGranularityDaily, Month: "2026-06"}
	wantParams := params
	rows, err := NewAccountResource(httpClient, "a1").Stats(context.Background(), "", &params)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if !reflect.DeepEqual(params, wantParams) {
		t.Errorf("params mutated: got %+v want %+v", params, wantParams)
	}
	if len(rows) != 1 || rows[0].DocumentsUploaded != 2 || rows[0].DocumentsCertified != 1 {
		t.Errorf("rows = %+v", rows)
	}
}

func TestUserGetSelf(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/users/self" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		writeMissingOperationJSON(w, `{"status":200,"data":{"id":"u1","name":"Ada","email":"ada@example.com"}}`)
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
		writeMissingOperationJSON(w, `{"status":200,"data":{"user":{"id":"u1","name":"Ada","email":"ada@example.com"},"accounts":[]}}`)
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
		writeMissingOperationJSON(w, `{"status":200,"data":[{"period":"2026-06","signature_requests":3}]}`)
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
		writeMissingOperationJSON(w, `{"status":200,"data":{"DocumentCompleted":true,"SignerDeclined":false}}`)
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
		writeMissingOperationJSON(w, `{"status":200,"data":{"DocumentCompleted":true,"DocumentExpired":false}}`)
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
