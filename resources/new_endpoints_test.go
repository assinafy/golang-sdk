package resources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/assinafy/golang-sdk/models"
)

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
			t.Fatalf("decode: %v", err)
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

func TestDocumentsSearch(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/acc1/documents/search" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("search"); got != "contract" {
			t.Errorf("search = %q", got)
		}
		if got := r.URL.Query().Get("status"); got != "metadata_ready" {
			t.Errorf("status = %q", got)
		}
		w.Header().Set("X-Pagination-Total-Count", "3")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[{"id":"d1","name":"contract.pdf","status":"metadata_ready"}]}`)
	})

	r := NewDocumentResource(httpClient, "acc1")
	page, err := r.Search(context.Background(), "", &models.ListParams{Search: "contract", Status: "metadata_ready"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "d1" || page.Pagination.TotalCount != 3 {
		t.Errorf("page = %+v", page)
	}
}

func TestDocumentsRename(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/documents/d1" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body["name"] != "renamed.pdf" {
			t.Errorf("name = %v", body["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":{"id":"d1","name":"renamed.pdf","status":"metadata_ready"}}`)
	})

	r := NewDocumentResource(httpClient, "acc1")
	doc, err := r.Rename(context.Background(), "d1", "renamed.pdf")
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if doc.Name != "renamed.pdf" {
		t.Errorf("name = %q", doc.Name)
	}
}

func TestAssignmentsList(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/assignments" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("per-page"); got != "5" {
			t.Errorf("per-page = %q", got)
		}
		w.Header().Set("X-Pagination-Total-Count", "1")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[{"id":"asg1","method":"virtual"}]}`)
	})

	r := NewAssignmentResource(httpClient)
	page, err := r.List(context.Background(), &models.ListParams{PerPage: 5})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "asg1" {
		t.Errorf("page = %+v", page)
	}
}

func TestSignerDocumentsSearch(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/signers/s1/documents/search" {
			t.Errorf("path = %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("signer-access-code") != "code-1" || q.Get("search") != "nda" {
			t.Errorf("query = %v", q)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	r := NewSignerDocumentResource(httpClient)
	if _, err := r.Search(context.Background(), "s1", "code-1", &models.ListParams{Search: "nda"}); err != nil {
		t.Fatalf("Search: %v", err)
	}
}

func TestAuthenticationLinkSocialLogin(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/auth/link-social-login" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
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

func TestAcceptTermsSendsAccessCodeAsQuery(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/signers/accept-terms" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("signer-access-code"); got != "code-1" {
			t.Errorf("signer-access-code = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if len(strings.TrimSpace(string(body))) != 0 {
			t.Errorf("expected empty body, got %q", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":{"id":"s1","full_name":"Bob","has_accepted_terms":true}}`)
	})

	r := NewSignerResource(httpClient, "")
	signer, err := r.AcceptTerms(context.Background(), "code-1")
	if err != nil {
		t.Fatalf("AcceptTerms: %v", err)
	}
	if !signer.HasAcceptedTerms {
		t.Errorf("signer = %+v", signer)
	}
}

func TestVerifyEmailSendsAccessCodeAsQuery(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/verify" {
			t.Errorf("method/path = %s %q", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("signer-access-code"); got != "code-1" {
			t.Errorf("signer-access-code = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body["verification-code"] != "123456" {
			t.Errorf("verification-code = %v", body["verification-code"])
		}
		if _, ok := body["signer-access-code"]; ok {
			t.Errorf("signer-access-code must not be in body")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":[]}`)
	})

	r := NewSignerResource(httpClient, "")
	if err := r.VerifyEmail(context.Background(), "code-1", "123456"); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
}
