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
	"strings"
	"testing"

	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

func TestDocumentedOperationWireContracts(t *testing.T) {
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

type multipartContract struct {
	field, filename string
	content         []byte
	values          map[string][]string
}

type wireCase struct {
	name, method, path string
	authenticated      bool
	query              url.Values
	jsonBody           any
	rawBody            []byte
	contentType        string
	multipart          *multipartContract
	response           string
	responseBytes      []byte
	call               func(*internal.HTTPClient) (any, error)
	want               any
}

func TestAdditionalMethodWireContracts(t *testing.T) {
	ctx := context.Background()
	fieldName := "Reference"
	signerName := "Ada Updated"
	tagName := "Priority"
	maskedKey := "issued****"
	assignmentBody := models.CreateAssignmentRequest{
		Method:  models.MethodVirtual,
		Signers: []models.SignerReference{{ID: "signer-1"}},
	}
	tests := []wireCase{
		{
			name: "assignment create", method: http.MethodPost, path: "/documents/doc%2F1/assignments", authenticated: true,
			jsonBody: assignmentBody,
			response: `{"status":200,"data":{"id":"assignment-1","method":"virtual"}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewAssignmentResource(h).Create(ctx, "doc/1", &assignmentBody)
			},
			want: &models.Assignment{ID: "assignment-1", Method: models.MethodVirtual},
		},
		{
			name: "assignment legacy estimate cost", method: http.MethodPost, path: "/documents/doc%2F1/assignments/estimate-cost", authenticated: true,
			jsonBody: assignmentBody,
			response: `{"status":200,"data":{"documents":1,"credits":2}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewAssignmentResource(h).EstimateCost(ctx, "doc/1", &assignmentBody)
			},
			want: &models.CostEstimate{Documents: 1, Credits: 2},
		},
		{
			name: "assignment estimate resend", method: http.MethodPost,
			path: "/documents/doc%2F1/assignments/asg%2F1/signers/signer%2F1/estimate-resend-cost", authenticated: true,
			response: `{"status":200,"data":{"total":1,"has_sufficient_credits":true}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewAssignmentResource(h).EstimateResendCost(ctx, "doc/1", "asg/1", "signer/1")
			},
			want: &models.CostEstimate{Total: 1, HasSufficientCredits: true},
		},
		{
			name: "assignment WhatsApp notifications", method: http.MethodGet,
			path: "/documents/doc%2F1/assignments/asg%2F1/whatsapp-notifications", authenticated: true,
			response: `{"status":200,"data":[{"header":"Sign","signer_id":"signer-1"}]}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewAssignmentResource(h).ListWhatsAppNotifications(ctx, "doc/1", "asg/1")
			},
			want: []models.WhatsAppNotification{{Header: "Sign", SignerID: "signer-1"}},
		},
		{
			name: "assignment legacy signing info", method: http.MethodGet, path: "/sign",
			query:    url.Values{"signer-access-code": {"sign-code"}},
			response: `{"status":200,"data":{"id":"document-1","name":"Contract","status":"ready"}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewAssignmentResource(h).GetSigningInfo(ctx, "sign-code")
			},
			want: &models.Document{ID: "document-1", Name: "Contract", Status: models.StatusReady},
		},
		{
			name: "authentication get API key", method: http.MethodGet, path: "/users/api-keys", authenticated: true,
			response: `{"status":200,"data":{"api_key":"issued****"}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewAuthenticationResource(h).GetAPIKey(ctx)
			},
			want: &models.APIKeyResult{APIKey: &maskedKey},
		},
		{
			name: "document upload", method: http.MethodPost, path: "/accounts/acc%2F1/documents", authenticated: true,
			multipart: &multipartContract{
				field: "file", filename: "contract.pdf", content: []byte("%PDF-1.1"),
				values: map[string][]string{"name": {"contract.pdf"}, "source": {"test"}},
			},
			response: `{"status":200,"data":{"id":"document-1","name":"contract.pdf","status":"uploaded"}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewDocumentResource(h, "acc/1").Upload(ctx, "", []byte("%PDF-1.1"), "contract.pdf", map[string]string{"source": "test"})
			},
			want: &models.Document{ID: "document-1", Name: "contract.pdf", Status: models.StatusUploaded},
		},
		{
			name: "document get", method: http.MethodGet, path: "/documents/doc%2F1", authenticated: true,
			response: `{"status":200,"data":{"id":"doc/1","name":"Contract","status":"ready"}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewDocumentResource(h, "").Get(ctx, "doc/1")
			},
			want: &models.Document{ID: "doc/1", Name: "Contract", Status: models.StatusReady},
		},
		{
			name: "document delete", method: http.MethodDelete, path: "/documents/doc%2F1", authenticated: true,
			response: `{"status":200,"data":[]}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return nil, NewDocumentResource(h, "").Delete(ctx, "doc/1")
			},
		},
		{
			name: "document activities", method: http.MethodGet, path: "/documents/doc%2F1/activities", authenticated: true,
			response: `{"status":200,"data":[{"id":7,"event":"document_ready","message":"Ready"}]}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewDocumentResource(h, "").Activities(ctx, "doc/1")
			},
			want: []models.DocumentActivity{{ID: 7, Event: "document_ready", Message: "Ready"}},
		},
		{
			name: "document artifact download", method: http.MethodGet,
			path: "/documents/doc%2F1/download/certificate-page", authenticated: true,
			responseBytes: []byte("PDF"),
			call: func(h *internal.HTTPClient) (any, error) {
				return NewDocumentResource(h, "").Download(ctx, "doc/1", "certificate-page")
			},
			want: []byte("PDF"),
		},
		{
			name: "document thumbnail", method: http.MethodGet, path: "/documents/doc%2F1/thumbnail", authenticated: true,
			responseBytes: []byte("PNG"),
			call: func(h *internal.HTTPClient) (any, error) {
				return NewDocumentResource(h, "").Thumbnail(ctx, "doc/1")
			},
			want: []byte("PNG"),
		},
		{
			name: "document page download", method: http.MethodGet,
			path: "/documents/doc%2F1/pages/page%2F1/download", authenticated: true,
			responseBytes: []byte("PAGE"),
			call: func(h *internal.HTTPClient) (any, error) {
				return NewDocumentResource(h, "").DownloadPage(ctx, "doc/1", "page/1")
			},
			want: []byte("PAGE"),
		},
		{
			name: "document verify", method: http.MethodGet, path: "/documents/hash%2F1/verify",
			response: `{"status":200,"data":{"hash":"hash/1","is_valid":true,"message":"valid"}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewDocumentResource(h, "").Verify(ctx, "hash/1")
			},
			want: &models.VerifyDocumentResult{Hash: "hash/1", IsValid: true, Message: "valid"},
		},
		{
			name: "document list tags", method: http.MethodGet,
			path: "/accounts/acc%2F1/documents/doc%2F1/tags", authenticated: true,
			response: `{"status":200,"data":[{"id":"tag-1","name":"Legal"}]}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewDocumentResource(h, "acc/1").ListTags(ctx, "", "doc/1")
			},
			want: []models.Tag{{ID: "tag-1", Name: "Legal"}},
		},
		{
			name: "document append tags", method: http.MethodPost,
			path: "/accounts/acc%2F1/documents/doc%2F1/tags", authenticated: true,
			jsonBody: models.SetDocumentTagsRequest{Tags: []string{"tag-1"}},
			response: `{"status":200,"data":[{"id":"tag-1","name":"Legal"}]}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewDocumentResource(h, "acc/1").AppendTags(ctx, "", "doc/1", []string{"tag-1"})
			},
			want: []models.Tag{{ID: "tag-1", Name: "Legal"}},
		},
		{
			name: "document legacy detach tag", method: http.MethodDelete,
			path: "/accounts/acc%2F1/documents/doc%2F1/tags/tag%2F1", authenticated: true,
			response: `{"status":200,"data":{"detached":true}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return nil, NewDocumentResource(h, "acc/1").DetachTag(ctx, "", "doc/1", "tag/1")
			},
		},
		{
			name: "document statuses", method: http.MethodGet, path: "/documents/statuses", authenticated: true,
			response: `{"status":200,"data":[{"code":"ready","deletable":true}]}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewDocumentResource(h, "").ListStatuses(ctx)
			},
			want: []models.DocumentStatusInfo{{Code: "ready", Deletable: true}},
		},
		{
			name: "field create", method: http.MethodPost, path: "/accounts/acc%2F1/fields", authenticated: true,
			jsonBody: models.CreateFieldDefinitionRequest{Type: "text", Name: "Reference"},
			response: `{"status":200,"data":{"id":"field-1","name":"Reference","type":"text"}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewFieldResource(h, "acc/1").Create(ctx, "", &models.CreateFieldDefinitionRequest{Type: "text", Name: "Reference"})
			},
			want: &models.FieldDefinition{ID: "field-1", Name: "Reference", Type: "text"},
		},
		{
			name: "field get", method: http.MethodGet, path: "/accounts/acc%2F1/fields/field%2F1", authenticated: true,
			response: `{"status":200,"data":{"id":"field/1","name":"Reference","type":"text"}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewFieldResource(h, "acc/1").Get(ctx, "", "field/1")
			},
			want: &models.FieldDefinition{ID: "field/1", Name: "Reference", Type: "text"},
		},
		{
			name: "field update", method: http.MethodPut, path: "/accounts/acc%2F1/fields/field%2F1", authenticated: true,
			jsonBody: models.UpdateFieldDefinitionRequest{Name: &fieldName},
			response: `{"status":200,"data":{"id":"field/1","name":"Reference","type":"text"}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewFieldResource(h, "acc/1").Update(ctx, "", "field/1", &models.UpdateFieldDefinitionRequest{Name: &fieldName})
			},
			want: &models.FieldDefinition{ID: "field/1", Name: "Reference", Type: "text"},
		},
		{
			name: "field delete", method: http.MethodDelete, path: "/accounts/acc%2F1/fields/field%2F1", authenticated: true,
			response: `{"status":200,"data":[]}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return nil, NewFieldResource(h, "acc/1").Delete(ctx, "", "field/1")
			},
		},
		{
			name: "field legacy validate", method: http.MethodPost, path: "/accounts/acc%2F1/fields/field%2F1/validate",
			query:    url.Values{"signer-access-code": {"sign-code"}},
			jsonBody: models.ValidateFieldRequest{Value: "ABC"},
			response: `{"status":200,"data":{"type":"text","success":true,"error_message":""}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewFieldResource(h, "acc/1").Validate(ctx, "", "field/1", "sign-code", &models.ValidateFieldRequest{Value: "ABC"})
			},
			want: &models.FieldValidationResult{Type: "text", Success: true},
		},
		{
			name: "field legacy validate multiple", method: http.MethodPost, path: "/accounts/acc%2F1/fields/validate-multiple",
			query:    url.Values{"signer-access-code": {"sign-code"}},
			jsonBody: []models.ValidateMultipleFieldsRequest{{FieldID: "field-1", Value: "ABC"}},
			response: `{"status":200,"data":[{"field_id":"field-1","type":"text","success":true,"error_message":""}]}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewFieldResource(h, "acc/1").ValidateMultiple(ctx, "", "sign-code", []models.ValidateMultipleFieldsRequest{{FieldID: "field-1", Value: "ABC"}})
			},
			want: []models.FieldValidationResult{{FieldID: "field-1", Type: "text", Success: true}},
		},
		{
			name: "field types", method: http.MethodGet, path: "/field-types", authenticated: true,
			response: `{"status":200,"data":[{"type":"text","name":"Text"}]}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewFieldResource(h, "").ListTypes(ctx)
			},
			want: []models.FieldType{{Type: "text", Name: "Text"}},
		},
		{
			name: "public document legacy get", method: http.MethodGet, path: "/public/documents/doc%2F1",
			response: `{"status":200,"data":{"id":"doc/1","name":"Contract","page_count":"1","created_by":"Ada"}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewPublicDocumentResource(h).Get(ctx, "doc/1")
			},
			want: &models.PublicDocumentInfo{ID: "doc/1", Name: "Contract", PageCount: "1", CreatedBy: "Ada"},
		},
		{
			name: "public document legacy send token", method: http.MethodPut, path: "/public/documents/doc%2F1/send-token",
			jsonBody: models.SendDocumentTokenRequest{Recipient: "ada@example.com", Channel: "email"},
			response: `{"status":200,"data":{"document":{"id":"doc/1","name":"Contract"},"channel":"email","recipient":"ada@example.com"}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewPublicDocumentResource(h).SendToken(ctx, "doc/1", &models.SendDocumentTokenRequest{Recipient: "ada@example.com", Channel: "email"})
			},
			want: &models.SendDocumentTokenResult{Document: models.PublicDocumentInfo{ID: "doc/1", Name: "Contract"}, Channel: "email", Recipient: "ada@example.com"},
		},
		{
			name: "signer list", method: http.MethodGet, path: "/accounts/acc%2F1/signers", authenticated: true,
			query:    url.Values{"page": {"2"}, "per-page": {"5"}, "search": {"ada"}},
			response: `{"status":200,"data":[{"id":"signer-1","full_name":"Ada","has_accepted_terms":false}]}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewSignerResource(h, "acc/1").List(ctx, "", &models.ListParams{Page: 2, PerPage: 5, Search: "ada"})
			},
			want: &models.PaginatedResult[models.Signer]{Data: []models.Signer{{ID: "signer-1", FullName: "Ada"}}},
		},
		{
			name: "signer update", method: http.MethodPut,
			path: "/accounts/acc%2F1/signers/signer%2F1", authenticated: true,
			jsonBody: models.UpdateSignerRequest{FullName: &signerName},
			response: `{"status":200,"data":{"id":"signer/1","full_name":"Ada Updated","has_accepted_terms":false}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewSignerResource(h, "acc/1").Update(ctx, "", "signer/1", &models.UpdateSignerRequest{FullName: &signerName})
			},
			want: &models.Signer{ID: "signer/1", FullName: "Ada Updated"},
		},
		{
			name: "signer delete", method: http.MethodDelete,
			path: "/accounts/acc%2F1/signers/signer%2F1", authenticated: true,
			response: `{"status":200,"data":[]}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return nil, NewSignerResource(h, "acc/1").Delete(ctx, "", "signer/1")
			},
		},
		{
			name: "signer legacy confirm data", method: http.MethodPut,
			path:     "/documents/doc%2F1/signers/confirm-data",
			query:    url.Values{"signer-access-code": {"sign-code"}},
			jsonBody: models.ConfirmSignerDataRequest{FullName: &signerName},
			response: `{"status":200,"data":{"id":"signer-1","full_name":"Ada Updated","has_accepted_terms":false}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return nil, NewSignerResource(h, "").ConfirmData(ctx, "doc/1", "sign-code", &models.ConfirmSignerDataRequest{FullName: &signerName})
			},
		},
		{
			name: "signer upload signature", method: http.MethodPost, path: "/signature",
			query:   url.Values{"signer-access-code": {"sign-code"}, "type": {"signature"}},
			rawBody: []byte("\x89PNG\r\n\x1a\n"), contentType: "image/png",
			response: `{"status":200,"message":"stored"}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return nil, NewSignerResource(h, "").UploadSignature(ctx, "sign-code", "signature", []byte("\x89PNG\r\n\x1a\n"))
			},
		},
		{
			name: "signer legacy upload content type", method: http.MethodPost, path: "/signature",
			query:   url.Values{"signer-access-code": {"sign-code"}, "type": {"initial"}},
			rawBody: []byte("image"), contentType: "image/webp",
			response: `{"status":200,"message":"stored"}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return nil, NewSignerResource(h, "").UploadSignatureWithContentType(ctx, "sign-code", "initial", "image/webp", []byte("image"))
			},
		},
		{
			name: "tag update", method: http.MethodPut, path: "/accounts/acc%2F1/tags/tag%2F1", authenticated: true,
			jsonBody: models.UpdateTagRequest{Name: &tagName},
			response: `{"status":200,"data":{"id":"tag/1","name":"Priority"}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewTagResource(h, "acc/1").Update(ctx, "", "tag/1", &models.UpdateTagRequest{Name: &tagName})
			},
			want: &models.Tag{ID: "tag/1", Name: "Priority"},
		},
		{
			name: "tag legacy delete", method: http.MethodDelete, path: "/accounts/acc%2F1/tags/tag%2F1", authenticated: true,
			query:    url.Values{"force": {"true"}},
			response: `{"status":200,"data":{"deleted":true}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return nil, NewTagResource(h, "acc/1").Delete(ctx, "", "tag/1", true)
			},
		},
		{
			name: "template compatibility get", method: http.MethodGet,
			path: "/accounts/acc%2F1/templates/template%2F1", authenticated: true,
			response: `{"status":200,"data":{"id":"template/1","name":"NDA","status":"ready"}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewTemplateResource(h, "acc/1").Get(ctx, "", "template/1")
			},
			want: &models.Template{ID: "template/1", Name: "NDA", Status: models.TemplateStatusReady},
		},
		{
			name: "webhook update subscription", method: http.MethodPut,
			path: "/accounts/acc%2F1/webhooks/subscriptions", authenticated: true,
			jsonBody: models.UpdateWebhookSubscriptionRequest{Events: []string{"document_ready"}, IsActive: true, URL: "https://example.test/hook", Email: "ada@example.com"},
			response: `{"status":200,"data":{"events":["document_ready"],"is_active":true}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewWebhookResource(h, "acc/1").UpdateSubscription(ctx, "", &models.UpdateWebhookSubscriptionRequest{Events: []string{"document_ready"}, IsActive: true, URL: "https://example.test/hook", Email: "ada@example.com"})
			},
			want: &models.WebhookSubscription{Events: []string{"document_ready"}, IsActive: true},
		},
		{
			name: "webhook get subscription", method: http.MethodGet,
			path: "/accounts/acc%2F1/webhooks/subscriptions", authenticated: true,
			response: `{"status":200,"data":{"events":["document_ready"],"is_active":true}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewWebhookResource(h, "acc/1").GetSubscription(ctx, "")
			},
			want: &models.WebhookSubscription{Events: []string{"document_ready"}, IsActive: true},
		},
		{
			name: "webhook inactivate", method: http.MethodPut,
			path: "/accounts/acc%2F1/webhooks/inactivate", authenticated: true,
			response: `{"status":200,"data":{"events":["document_ready"],"is_active":false}}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewWebhookResource(h, "acc/1").Inactivate(ctx, "")
			},
			want: &models.WebhookSubscription{Events: []string{"document_ready"}},
		},
		{
			name: "webhook event types", method: http.MethodGet, path: "/webhooks/event-types", authenticated: true,
			response: `{"status":200,"data":[{"id":"document_ready","description":"Document ready"}]}`,
			call: func(h *internal.HTTPClient) (any, error) {
				return NewWebhookResource(h, "").ListEventTypes(ctx)
			},
			want: []models.WebhookEventType{{ID: "document_ready", Description: "Document ready"}},
		},
	}

	runWireCases(t, tests)
}

func runWireCases(t *testing.T, tests []wireCase) {
	t.Helper()
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

				switch {
				case tc.multipart != nil:
					checkMultipartContract(t, w, r, tc.multipart)
				case tc.rawBody != nil:
					checkRawContract(t, w, r, tc.rawBody, tc.contentType)
				default:
					checkJSONContract(t, w, r, tc.jsonBody)
				}

				if tc.responseBytes != nil {
					_, _ = w.Write(tc.responseBytes)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.response)
			})

			got, err := tc.call(httpClient)
			if err != nil {
				t.Fatalf("call failed: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("decoded result = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func checkJSONContract(t *testing.T, w http.ResponseWriter, r *http.Request, wantBody any) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("read body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if wantBody == nil {
		if len(bytes.TrimSpace(body)) != 0 {
			t.Errorf("body = %s, want empty", body)
		}
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
	}
	wantJSON, err := json.Marshal(wantBody)
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

func checkRawContract(t *testing.T, w http.ResponseWriter, r *http.Request, want []byte, contentType string) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("read body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !bytes.Equal(body, want) || r.Header.Get("Content-Type") != contentType {
		t.Errorf("raw body/content type = %q %q, want %q %q", body, r.Header.Get("Content-Type"), want, contentType)
	}
}

func checkMultipartContract(t *testing.T, w http.ResponseWriter, r *http.Request, want *multipartContract) {
	t.Helper()
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data; boundary=") {
		t.Errorf("Content-Type = %q, want multipart/form-data", r.Header.Get("Content-Type"))
	}
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		t.Errorf("parse multipart: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()
	if !reflect.DeepEqual(r.MultipartForm.Value, want.values) {
		t.Errorf("multipart values = %v, want %v", r.MultipartForm.Value, want.values)
	}
	file, header, err := r.FormFile(want.field)
	if err != nil {
		t.Errorf("multipart file %q: %v", want.field, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer func() { _ = file.Close() }()
	content, err := io.ReadAll(file)
	if err != nil {
		t.Errorf("read multipart file: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if header.Filename != want.filename || !bytes.Equal(content, want.content) {
		t.Errorf("multipart file = %q %q, want %q %q", header.Filename, content, want.filename, want.content)
	}
}
