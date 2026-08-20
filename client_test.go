package assinafy

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/assinafy/golang-sdk/models"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name string
		opts ClientOptions
	}{
		{"with API key", ClientOptions{APIKey: "test-api-key", AccountID: "acc"}},
		{"with token", ClientOptions{Token: "test-token", AccountID: "acc"}},
		{"no credentials (public/auth flows only)", ClientOptions{AccountID: "acc"}},
		{"sandbox base URL", ClientOptions{APIKey: "k", BaseURL: SandboxBaseURL, AccountID: "acc"}},
		{"custom timeout", ClientOptions{APIKey: "k", Timeout: 60 * time.Second, AccountID: "acc"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, err := NewClient(tc.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c == nil {
				t.Fatal("expected non-nil client")
			}
			if c.Documents == nil || c.Signers == nil || c.Assignments == nil ||
				c.Webhooks == nil || c.Templates == nil || c.Fields == nil ||
				c.Authentication == nil || c.PublicDocuments == nil || c.SignerDocuments == nil {
				t.Errorf("missing resource: %+v", c)
			}
		})
	}
}

func TestNewClientRejectsInvalidBaseURL(t *testing.T) {
	for _, baseURL := range []string{
		"relative/path",
		"ftp://api.assinafy.com.br/v1",
		"https://user:password@api.assinafy.com.br/v1",
		"https://api.assinafy.com.br/v1?token=secret",
		"https://api.assinafy.com.br/v1?",
		"https://api.assinafy.com.br/v1#fragment",
		"https://api.assinafy.com.br/v1#",
	} {
		t.Run(baseURL, func(t *testing.T) {
			if _, err := NewClient(ClientOptions{BaseURL: baseURL}); !errors.Is(err, ErrInvalidBaseURL) {
				t.Errorf("NewClient error = %v, want ErrInvalidBaseURL", err)
			}
		})
	}
}

func TestUploadAndRequestSignaturesValidatesSigners(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	c, err := NewClient(ClientOptions{APIKey: "k", AccountID: "acc", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		signers []models.UploadAndRequestSignaturesSigner
		want    error
	}{
		{name: "none", want: ErrNoSigners},
		{name: "blank name", signers: []models.UploadAndRequestSignaturesSigner{{Name: "  ", Email: "a@example.com"}}, want: ErrInvalidSigner},
		{name: "no contact", signers: []models.UploadAndRequestSignaturesSigner{{Name: "Alice"}}, want: ErrInvalidSigner},
		{
			name: "validates all before upload",
			signers: []models.UploadAndRequestSignaturesSigner{
				{Name: "Alice", Email: "a@example.com"},
				{Name: "Bob", Email: " ", WhatsAppPhoneNumber: "\t"},
			},
			want: ErrInvalidSigner,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, got := c.UploadAndRequestSignatures(context.Background(), []byte("pdf"), "doc.pdf", tc.signers, "", nil, "")
			if !errors.Is(got, tc.want) {
				t.Errorf("error = %v, want errors.Is(_, %v)", got, tc.want)
			}
		})
	}
	if requests != 0 {
		t.Errorf("made %d requests before validating all signers", requests)
	}
}

func TestUploadAndRequestSignaturesSuccess(t *testing.T) {
	signerCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/accounts/acc/documents":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("ParseMultipartForm: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			defer func() { _ = r.MultipartForm.RemoveAll() }()
			file, header, err := r.FormFile("file")
			if err != nil {
				t.Errorf("FormFile: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			defer func() { _ = file.Close() }()
			content, err := io.ReadAll(file)
			if err != nil || string(content) != "pdf" || header.Filename != "doc.pdf" || r.FormValue("name") != "doc.pdf" {
				t.Errorf("upload = content %q, filename %q, name %q, error %v", content, header.Filename, r.FormValue("name"), err)
			}
			_, _ = io.WriteString(w, `{"status":200,"data":{"id":"doc-1","name":"doc.pdf","status":"uploaded"}}`)

		case r.Method == http.MethodPost && r.URL.Path == "/accounts/acc/signers":
			signerCalls++
			var body struct {
				FullName            string  `json:"full_name"`
				Email               *string `json:"email"`
				WhatsAppPhoneNumber *string `json:"whatsapp_phone_number"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode signer: %v", err)
			}
			if signerCalls == 1 {
				if body.FullName != "Alice" || body.Email == nil || *body.Email != "alice@example.com" || body.WhatsAppPhoneNumber == nil || *body.WhatsAppPhoneNumber != "+5511000000001" {
					t.Errorf("first signer body = %+v", body)
				}
				_, _ = io.WriteString(w, `{"status":200,"data":{"id":"signer-1","full_name":"Alice","has_accepted_terms":false}}`)
				return
			}
			if body.FullName != "Bob" || body.Email != nil || body.WhatsAppPhoneNumber == nil || *body.WhatsAppPhoneNumber != "+5511000000002" {
				t.Errorf("second signer body = %+v", body)
			}
			_, _ = io.WriteString(w, `{"status":200,"data":{"id":"signer-2","full_name":"Bob","has_accepted_terms":false}}`)

		case r.Method == http.MethodPost && r.URL.Path == "/documents/doc-1/assignments":
			var body struct {
				Method  string `json:"method"`
				Signers []struct {
					ID                  string   `json:"id"`
					VerificationMethod  string   `json:"verification_method"`
					NotificationMethods []string `json:"notification_methods"`
				} `json:"signers"`
				Message   *string `json:"message"`
				ExpiresAt *string `json:"expires_at"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode assignment: %v", err)
			}
			if body.Method != "virtual" || len(body.Signers) != 2 || body.Message == nil || *body.Message != "Please sign" || body.ExpiresAt == nil || *body.ExpiresAt != "2026-12-31T23:59:59Z" {
				t.Errorf("assignment body = %+v", body)
			} else if body.Signers[0].ID != "signer-1" || body.Signers[0].VerificationMethod != "Email" || len(body.Signers[0].NotificationMethods) != 1 || body.Signers[0].NotificationMethods[0] != "Email" ||
				body.Signers[1].ID != "signer-2" || body.Signers[1].VerificationMethod != "Whatsapp" || len(body.Signers[1].NotificationMethods) != 1 || body.Signers[1].NotificationMethods[0] != "Whatsapp" {
				t.Errorf("assignment signers = %+v", body.Signers)
			}
			_, _ = io.WriteString(w, `{"status":200,"data":{"id":"assignment-1","method":"virtual"}}`)

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c, err := NewClient(ClientOptions{APIKey: "k", AccountID: "acc", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	expiresAt := "2026-12-31T23:59:59Z"
	result, err := c.UploadAndRequestSignatures(context.Background(), []byte("pdf"), "doc.pdf", []models.UploadAndRequestSignaturesSigner{
		{Name: " Alice ", Email: " alice@example.com ", WhatsAppPhoneNumber: " +5511000000001 "},
		{Name: "Bob", WhatsAppPhoneNumber: "+5511000000002"},
	}, "Please sign", &expiresAt, "")
	if err != nil {
		t.Fatalf("UploadAndRequestSignatures: %v", err)
	}
	if result.Document == nil || result.Document.ID != "doc-1" || result.Assignment == nil || result.Assignment.ID != "assignment-1" || len(result.SignerIDs) != 2 || result.SignerIDs[0] != "signer-1" || result.SignerIDs[1] != "signer-2" {
		t.Errorf("result = %+v", result)
	}
}

func TestUploadAndRequestSignaturesReturnsPartialResult(t *testing.T) {
	signerCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/accounts/acc/documents":
			_, _ = io.WriteString(w, `{"status":200,"data":{"id":"doc-1","status":"uploaded"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/accounts/acc/signers":
			signerCalls++
			if signerCalls == 1 {
				_, _ = io.WriteString(w, `{"status":200,"data":{"id":"signer-1","full_name":"Alice","has_accepted_terms":false}}`)
				return
			}
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = io.WriteString(w, `{"status":422,"message":"signer rejected","data":[]}`)
		default:
			t.Errorf("unexpected request after signer failure: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c, err := NewClient(ClientOptions{APIKey: "k", AccountID: "acc", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := c.UploadAndRequestSignatures(context.Background(), []byte("pdf"), "doc.pdf", []models.UploadAndRequestSignaturesSigner{
		{Name: "Alice", Email: "alice@example.com"},
		{Name: "Bob", Email: "bob@example.com"},
	}, "", nil, "")
	if err == nil {
		t.Fatal("expected signer creation error")
	}
	if result == nil || result.Document == nil || result.Document.ID != "doc-1" || result.Assignment != nil || len(result.SignerIDs) != 1 || result.SignerIDs[0] != "signer-1" {
		t.Errorf("partial result = %+v", result)
	}
}
