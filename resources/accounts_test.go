package resources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/assinafy/golang-sdk/models"
)

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
		writeJSONResponse(w, `{"status":200,"data":{"id":"a1","name":"Acme","notification_sender_type":"Account"}}`)
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
		writeJSONResponse(w, `{"status":200,"data":[]}`)
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
		writeJSONResponse(w, `{"status":200,"message":"Logo updated"}`)
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
		writeJSONResponse(w, `{"status":200,"message":"Logo deleted"}`)
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
		writeJSONResponse(w, `{"status":200,"data":[{"period":"2026-06-01","documents_uploaded":2,"documents_certified":1}]}`)
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

func TestAccountsList(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/accounts" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[{"id":"a1","name":"MT","roles":["owner"],"is_delete_allowed":true,"created_at":"2026-05-12T18:05:11Z"}]}`)
	})

	r := NewAccountResource(httpClient, "a1")
	accts, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(accts) != 1 || accts[0].ID != "a1" || len(accts[0].Roles) != 1 || !accts[0].IsDeleteAllowed {
		t.Errorf("accounts = %+v", accts)
	}
}

func TestAccountsGetDecodesColors(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/a1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":{"id":"a1","name":"MT","primary_color":null,"secondary_color":"112233","created_at":"2026-05-12T18:05:11Z"}}`)
	})

	r := NewAccountResource(httpClient, "")
	acct, err := r.Get(context.Background(), "a1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if acct.PrimaryColor != nil {
		t.Errorf("primary_color = %v, want nil", acct.PrimaryColor)
	}
	if acct.SecondaryColor == nil || *acct.SecondaryColor != "112233" {
		t.Errorf("secondary_color = %v", acct.SecondaryColor)
	}
}

func TestAccountsUpdateOmitsNilFields(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/accounts/a1" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if body["name"] != "Acme" {
			t.Errorf("name = %v", body["name"])
		}
		if _, ok := body["notification_sender_type"]; ok {
			t.Errorf("notification_sender_type should be omitted, got %v", body["notification_sender_type"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":{"id":"a1","name":"Acme"}}`)
	})

	name := "Acme"
	r := NewAccountResource(httpClient, "")
	acct, err := r.Update(context.Background(), "a1", &models.UpdateAccountRequest{Name: &name})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if acct.Name != "Acme" {
		t.Errorf("name = %q", acct.Name)
	}
}

func TestAccountsGetTheme(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/a1/theme" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":{"account_name":"MT","primary_color":"2072b9","secondary_color":"ffffff","logo":null}}`)
	})

	r := NewAccountResource(httpClient, "")
	theme, err := r.GetTheme(context.Background(), "a1")
	if err != nil {
		t.Fatalf("GetTheme: %v", err)
	}
	if theme.AccountName != "MT" || theme.PrimaryColor != "2072b9" || theme.Logo != nil {
		t.Errorf("theme = %+v", theme)
	}
}

func TestAccountsDownloadLogo(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/a1/logo" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("\x89PNG\r\n"))
	})

	r := NewAccountResource(httpClient, "")
	img, err := r.DownloadLogo(context.Background(), "a1")
	if err != nil {
		t.Fatalf("DownloadLogo: %v", err)
	}
	if !strings.HasPrefix(string(img), "\x89PNG") {
		t.Errorf("logo bytes = %q", img)
	}
}
