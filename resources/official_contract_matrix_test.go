package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

func TestOfficialOperationWireMatrix(t *testing.T) {
	type wireCase struct {
		name, method, path string
		authenticated      bool
		query              url.Values
		body               any
		response           string
		responseBytes      []byte
		call               func(*internal.HTTPClient) error
	}

	ctx := context.Background()
	token := "reset-token"
	step := 2
	tests := []wireCase{
		{
			name: "authentication login", method: http.MethodPost, path: "/login",
			body:     models.LoginRequest{Email: "ada@example.com", Password: "old-secret"},
			response: `{"status":200,"data":{"access_token":"login-token","user":{"id":"u1","name":"Ada","email":"ada@example.com"},"accounts":[]}}`,
			call: func(httpClient *internal.HTTPClient) error {
				got, err := NewAuthenticationResource(httpClient).Login(ctx, &models.LoginRequest{Email: "ada@example.com", Password: "old-secret"})
				if err == nil && (got.AccessToken != "login-token" || got.User.ID != "u1") {
					return fmt.Errorf("decoded login response = %+v", got)
				}
				return err
			},
		},
		{
			name: "authentication social login", method: http.MethodPost, path: "/authentication/social-login",
			body:     models.SocialLoginRequest{Provider: "google", Token: "identity-token", HasAcceptedTerms: true},
			response: `{"status":200,"data":{"access_token":"social-token","user":{"id":"u2","name":"Grace","email":"grace@example.com"},"accounts":[]}}`,
			call: func(httpClient *internal.HTTPClient) error {
				got, err := NewAuthenticationResource(httpClient).SocialLogin(ctx, &models.SocialLoginRequest{Provider: "google", Token: "identity-token", HasAcceptedTerms: true})
				if err == nil && (got.AccessToken != "social-token" || got.User.ID != "u2") {
					return fmt.Errorf("decoded social login response = %+v", got)
				}
				return err
			},
		},
		{
			name: "authentication create API key", method: http.MethodPost, path: "/users/api-keys", authenticated: true,
			body:     models.CreateAPIKeyRequest{Password: "current-secret"},
			response: `{"status":200,"data":{"api_key":"issued-key"}}`,
			call: func(httpClient *internal.HTTPClient) error {
				got, err := NewAuthenticationResource(httpClient).CreateAPIKey(ctx, &models.CreateAPIKeyRequest{Password: "current-secret"})
				if err == nil && (got.APIKey == nil || *got.APIKey != "issued-key") {
					return fmt.Errorf("decoded API-key response = %+v", got)
				}
				return err
			},
		},
		{
			name: "authentication delete API key", method: http.MethodDelete, path: "/users/api-keys", authenticated: true,
			response: `{"status":200,"data":[]}`,
			call: func(httpClient *internal.HTTPClient) error {
				return NewAuthenticationResource(httpClient).DeleteAPIKey(ctx)
			},
		},
		{
			name: "authentication change password", method: http.MethodPut, path: "/authentication/change-password", authenticated: true,
			body:     models.ChangePasswordRequest{Email: "ada@example.com", Password: "old-secret", NewPassword: "new-secret"},
			response: `{"status":200,"data":{"email":"ada@example.com"}}`,
			call: func(httpClient *internal.HTTPClient) error {
				got, err := NewAuthenticationResource(httpClient).ChangePassword(ctx, &models.ChangePasswordRequest{Email: "ada@example.com", Password: "old-secret", NewPassword: "new-secret"})
				if err == nil && got.Email != "ada@example.com" {
					return fmt.Errorf("decoded change-password response = %+v", got)
				}
				return err
			},
		},
		{
			name: "authentication request password reset", method: http.MethodPut, path: "/authentication/request-password-reset",
			body:     models.RequestPasswordResetRequest{Email: "ada@example.com"},
			response: `{"status":200,"data":{"email":"ada@example.com"}}`,
			call: func(httpClient *internal.HTTPClient) error {
				got, err := NewAuthenticationResource(httpClient).RequestPasswordReset(ctx, &models.RequestPasswordResetRequest{Email: "ada@example.com"})
				if err == nil && got.Email != "ada@example.com" {
					return fmt.Errorf("decoded request-reset response = %+v", got)
				}
				return err
			},
		},
		{
			name: "authentication reset password", method: http.MethodPut, path: "/authentication/reset-password",
			body:     models.ResetPasswordRequest{Email: "ada@example.com", Token: &token, NewPassword: "new-secret"},
			response: `{"status":200,"data":{"email":"ada@example.com"}}`,
			call: func(httpClient *internal.HTTPClient) error {
				got, err := NewAuthenticationResource(httpClient).ResetPassword(ctx, &models.ResetPasswordRequest{Email: "ada@example.com", Token: &token, NewPassword: "new-secret"})
				if err == nil && got.Email != "ada@example.com" {
					return fmt.Errorf("decoded reset-password response = %+v", got)
				}
				return err
			},
		},
		{
			name: "assignment decline", method: http.MethodPut, path: "/documents/doc%2F1/assignments/asg%2F1/reject",
			query: url.Values{"signer-access-code": {"sign-code"}},
			body:  map[string]string{"decline_reason": "Not mine"}, response: `{"status":200,"data":[]}`,
			call: func(httpClient *internal.HTTPClient) error {
				return NewAssignmentResource(httpClient).Decline(ctx, "doc/1", "asg/1", "sign-code", "Not mine")
			},
		},
		{
			name: "document estimate template cost", method: http.MethodPost,
			path: "/accounts/acc%2F1/templates/template%2F1/documents/estimate-cost", authenticated: true,
			body: map[string]any{"signers": []map[string]any{{
				"role_id": "role-1", "verification_method": "Email", "notification_methods": []string{"Email"},
			}}},
			response: `{"status":200,"data":{"documents":1,"credits":2}}`,
			call: func(httpClient *internal.HTTPClient) error {
				got, err := NewDocumentResource(httpClient, "acc/1").EstimateCostFromTemplate(ctx, "", "template/1", []models.TemplateSigner{{
					RoleID: "role-1", ID: "not-valid-for-estimates", VerificationMethod: "Email",
					NotificationMethods: []string{"Email"}, Step: &step,
				}})
				if err == nil && (got.Documents != 1 || got.Credits != 2) {
					return fmt.Errorf("decoded template estimate = %+v", got)
				}
				return err
			},
		},
		{
			name: "signer self", method: http.MethodGet, path: "/signers/self",
			query:    url.Values{"signer-access-code": {"sign-code"}},
			response: `{"status":200,"data":{"id":"signer-1","full_name":"Ada","has_accepted_terms":true,"has_signature":true}}`,
			call: func(httpClient *internal.HTTPClient) error {
				got, err := NewSignerResource(httpClient, "").GetSelf(ctx, "sign-code")
				if err == nil && (got.ID != "signer-1" || !got.HasSignature) {
					return fmt.Errorf("decoded signer self = %+v", got)
				}
				return err
			},
		},
		{
			name: "signer signature download", method: http.MethodGet, path: "/signature/signature%2Fink",
			query:         url.Values{"signer-access-code": {"sign-code"}},
			responseBytes: []byte("PNG"),
			call: func(httpClient *internal.HTTPClient) error {
				got, err := NewSignerResource(httpClient, "").DownloadSignature(ctx, "sign-code", "signature/ink")
				if err == nil && !bytes.Equal(got, []byte("PNG")) {
					return fmt.Errorf("downloaded signature = %q", got)
				}
				return err
			},
		},
		{
			name: "signer current document", method: http.MethodGet, path: "/signers/signer%2F1/document",
			query:    url.Values{"signer-access-code": {"sign-code"}},
			response: `{"status":200,"data":{"id":"document-1","name":"Contract","status":"ready"}}`,
			call: func(httpClient *internal.HTTPClient) error {
				got, err := NewSignerDocumentResource(httpClient).GetCurrent(ctx, "signer/1", "sign-code")
				if err == nil && got.ID != "document-1" {
					return fmt.Errorf("decoded current document = %+v", got)
				}
				return err
			},
		},
		{
			name: "signer documents sign multiple", method: http.MethodPut, path: "/signers/documents/sign-multiple",
			query: url.Values{"signer-access-code": {"sign-code"}},
			body:  map[string][]string{"document_ids": {"document-1", "document-2"}}, response: `{"status":200,"data":[]}`,
			call: func(httpClient *internal.HTTPClient) error {
				return NewSignerDocumentResource(httpClient).SignMultiple(ctx, "sign-code", []string{"document-1", "document-2"})
			},
		},
		{
			name: "signer documents decline multiple", method: http.MethodPut, path: "/signers/documents/decline-multiple",
			query: url.Values{"signer-access-code": {"sign-code"}},
			body:  map[string]any{"document_ids": []string{"document-1"}, "decline_reason": "Not mine"}, response: `{"status":200,"data":[]}`,
			call: func(httpClient *internal.HTTPClient) error {
				return NewSignerDocumentResource(httpClient).DeclineMultiple(ctx, "sign-code", []string{"document-1"}, "Not mine")
			},
		},
		{
			name: "public signer document download", method: http.MethodGet,
			path:          "/signers/signer%2F1/documents/document%2F1/download/certificate-page",
			responseBytes: []byte("PDF"),
			call: func(httpClient *internal.HTTPClient) error {
				got, err := NewSignerDocumentResource(httpClient).Download(ctx, "signer/1", "document/1", "certificate-page", "")
				if err == nil && !bytes.Equal(got, []byte("PDF")) {
					return fmt.Errorf("downloaded signer document = %q", got)
				}
				return err
			},
		},
		{
			name: "webhook retry dispatch", method: http.MethodPost,
			path: "/accounts/acc%2F1/webhooks/history%2F1/retry", authenticated: true,
			response: `{"status":200,"data":{"id":"history/1","event":"document_ready","delivered":true}}`,
			call: func(httpClient *internal.HTTPClient) error {
				got, err := NewWebhookResource(httpClient, "acc/1").RetryDispatch(ctx, "", "history/1")
				if err == nil && (got.ID != "history/1" || !got.Delivered) {
					return fmt.Errorf("decoded webhook retry = %+v", got)
				}
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.EscapedPath() != tc.path {
					t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.EscapedPath(), tc.method, tc.path)
				}
				if got := r.URL.Query(); !reflect.DeepEqual(got, tc.query) && (len(got) != 0 || len(tc.query) != 0) {
					t.Errorf("query = %v, want %v", got, tc.query)
				}
				if tc.authenticated {
					if r.Header.Get("X-Api-Key") != "key" || r.Header.Get("Authorization") != "" {
						t.Errorf("authenticated headers = %v", r.Header)
					}
				} else if r.Header.Get("X-Api-Key") != "" || r.Header.Get("Authorization") != "" {
					t.Errorf("credential leaked to public/signer operation: %v", r.Header)
				}

				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read body: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if tc.body == nil {
					if len(bytes.TrimSpace(body)) != 0 {
						t.Errorf("body = %s, want empty", body)
					}
				} else {
					wantJSON, err := json.Marshal(tc.body)
					if err != nil {
						t.Errorf("marshal expected body: %v", err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					var got, want any
					if err := json.Unmarshal(body, &got); err != nil {
						t.Errorf("decode body: %v", err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if err := json.Unmarshal(wantJSON, &want); err != nil {
						t.Errorf("decode expected body: %v", err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if !reflect.DeepEqual(got, want) {
						t.Errorf("body = %#v, want %#v", got, want)
					}
				}

				if tc.responseBytes != nil {
					_, _ = w.Write(tc.responseBytes)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.response)
			})

			if err := tc.call(httpClient); err != nil {
				t.Errorf("call failed: %v", err)
			}
		})
	}
}
