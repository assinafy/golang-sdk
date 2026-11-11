package resources

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
	"github.com/assinafy/golang-sdk/v2/internal"
	"github.com/assinafy/golang-sdk/v2/models"
)

func TestEmbedOrigins(t *testing.T) {
	ctx := context.Background()
	const originJSON = `{"id":"origin/1","origin":"https://app.example.com","created_at":"2026-11-04T12:05:00Z"}`
	want := models.EmbedOrigin{ID: "origin/1", Origin: "https://app.example.com", CreatedAt: "2026-11-04T12:05:00Z"}
	for _, tc := range []struct {
		name, method, path, response string
		body                         any
		call                         func(*EmbedResource) error
	}{
		{
			name: "list uses default workspace", method: http.MethodGet,
			path: "/accounts/default/embed-origins", response: `[` + originJSON + `]`,
			call: func(r *EmbedResource) error {
				out, err := r.ListOrigins(ctx, "")
				if err == nil && !reflect.DeepEqual(out, []models.EmbedOrigin{want}) {
					t.Errorf("ListOrigins = %#v", out)
				}
				return err
			},
		},
		{
			name: "create overrides and escapes workspace", method: http.MethodPost,
			path: "/accounts/account%2F1/embed-origins", response: originJSON,
			body: map[string]any{"origin": "https://app.example.com"},
			call: func(r *EmbedResource) error {
				out, err := r.CreateOrigin(ctx, "account/1", &models.EmbedOriginRequest{Origin: want.Origin})
				if err == nil && *out != want {
					t.Errorf("CreateOrigin = %#v", out)
				}
				return err
			},
		},
		{
			name: "update escapes origin ID", method: http.MethodPut,
			path: "/accounts/default/embed-origins/origin%2F1", response: originJSON,
			body: map[string]any{"origin": "https://app.example.com"},
			call: func(r *EmbedResource) error {
				out, err := r.UpdateOrigin(ctx, "", want.ID, &models.EmbedOriginRequest{Origin: want.Origin})
				if err == nil && *out != want {
					t.Errorf("UpdateOrigin = %#v", out)
				}
				return err
			},
		},
		{
			name: "delete", method: http.MethodDelete,
			path: "/accounts/default/embed-origins/origin%2F1", response: `[]`,
			call: func(r *EmbedResource) error { return r.DeleteOrigin(ctx, "", want.ID) },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != tc.method || req.URL.EscapedPath() != tc.path || req.URL.RawQuery != "" {
					t.Errorf("request = %s %s", req.Method, req.URL)
				}
				if req.Header.Get("X-Api-Key") != "key" {
					t.Error("missing client authentication")
				}
				if tc.body != nil {
					if got := decodeContractBody(t, req); !reflect.DeepEqual(got, tc.body) {
						t.Errorf("body = %#v, want %#v", got, tc.body)
					}
				} else if body, err := io.ReadAll(req.Body); err != nil || len(body) != 0 {
					t.Errorf("unexpected body %q, error %v", body, err)
				}
				writeJSONResponse(w, `{"status":200,"data":`+tc.response+`}`)
			})
			if err := tc.call(NewEmbedResource(h, "default")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEmbedCreateSession(t *testing.T) {
	h, _ := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || req.URL.EscapedPath() != "/documents/doc%2F1/assignments/assignment%2F1/signers/signer%2F1/embed-sessions" || req.URL.RawQuery != "" {
			t.Errorf("request = %s %s", req.Method, req.URL)
		}
		if req.Header.Get("X-Api-Key") != "key" {
			t.Error("missing client authentication")
		}
		if body, err := io.ReadAll(req.Body); err != nil || len(body) != 0 {
			t.Errorf("unexpected body %q, error %v", body, err)
		}
		writeJSONResponse(w, `{"status":200,"data":{"url":"https://app.example.com/embed/fixture-token","expires_at":"2026-11-04T12:05:00Z"}}`)
	})
	out, err := NewEmbedResource(h, "").CreateSession(context.Background(), "doc/1", "assignment/1", "signer/1")
	if err != nil || out.URL != "https://app.example.com/embed/fixture-token" || out.ExpiresAt != "2026-11-04T12:05:00Z" {
		t.Fatalf("CreateSession = %#v, %v", out, err)
	}
}

func TestEmbedExchangeSessionOmitsWorkspaceCredentials(t *testing.T) {
	for _, withBody := range []bool{false, true} {
		var body *models.ExchangeEmbedSessionRequest
		if withBody {
			body = &models.ExchangeEmbedSessionRequest{ParentOrigin: "https://app.example.com"}
		}
		h, server := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
			assertNoClientAuth(t, req)
			if req.Method != http.MethodPost || req.URL.EscapedPath() != "/embed/sessions/token%2F1/exchange" || req.URL.RawQuery != "" {
				t.Errorf("request = %s %s", req.Method, req.URL)
			}
			if withBody {
				if got := decodeContractBody(t, req); !reflect.DeepEqual(got, map[string]any{"parent_origin": "https://app.example.com"}) {
					t.Errorf("body = %#v", got)
				}
			} else if got := decodeContractBody(t, req); !reflect.DeepEqual(got, map[string]any{}) {
				t.Errorf("body = %#v, want empty object", got)
			}
			writeJSONResponse(w, `{"status":200,"data":{"signer_access_code":"fixture-code","signer_id":"signer-1","document_id":"doc-1","verification_method":"Email","embed_origins":["https://app.example.com"]}}`)
		})
		for _, client := range []*internal.HTTPClient{
			h,
			internal.NewHTTPClient(server.URL, "", "fixture-bearer", time.Second),
			internal.NewHTTPClientWithTokenSource(server.URL, func(context.Context) (string, error) {
				t.Error("exchange consulted workspace token source")
				return "fixture-bearer", nil
			}, time.Second),
		} {
			out, err := NewEmbedResource(client, "").ExchangeSession(context.Background(), "token/1", body)
			want := &models.EmbedSessionExchange{SignerAccessCode: "fixture-code", SignerID: "signer-1", DocumentID: "doc-1", VerificationMethod: "Email", EmbedOrigins: []string{"https://app.example.com"}}
			if err != nil || !reflect.DeepEqual(out, want) {
				t.Fatalf("ExchangeSession = %#v, %v", out, err)
			}
		}
	}
}

func TestEmbedRejectsMissingIDs(t *testing.T) {
	h, _ := newTestClient(t, func(_ http.ResponseWriter, _ *http.Request) { t.Error("sent invalid request") })
	r := NewEmbedResource(h, "")
	ctx := context.Background()
	for name, call := range map[string]func() error{
		"list workspace":       func() error { _, err := r.ListOrigins(ctx, ""); return err },
		"create workspace":     func() error { _, err := r.CreateOrigin(ctx, "", nil); return err },
		"update origin":        func() error { _, err := r.UpdateOrigin(ctx, "account", "", nil); return err },
		"delete origin":        func() error { return r.DeleteOrigin(ctx, "account", "") },
		"session document":     func() error { _, err := r.CreateSession(ctx, "", "assignment", "signer"); return err },
		"session assignment":   func() error { _, err := r.CreateSession(ctx, "document", "", "signer"); return err },
		"session signer":       func() error { _, err := r.CreateSession(ctx, "document", "assignment", " "); return err },
		"exchange token":       func() error { _, err := r.ExchangeSession(ctx, "", nil); return err },
		"blank exchange token": func() error { _, err := r.ExchangeSession(ctx, " \t", nil); return err },
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, sdkerrors.ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestEmbedPreservesSessionErrors(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
	}{
		{http.StatusForbidden, "PlanRestricted"},
		{http.StatusConflict, "document_not_ready"},
		{http.StatusGone, "embed_session_invalid"},
	} {
		h, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			writeJSONResponse(w, `{"status":`+strconv.Itoa(tc.status)+`,"message":"session unavailable","code":"`+tc.code+`"}`)
		})
		r := NewEmbedResource(h, "account")
		for _, call := range []func() error{
			func() error {
				_, err := r.CreateSession(context.Background(), "document", "assignment", "signer")
				return err
			},
			func() error { _, err := r.ExchangeSession(context.Background(), "token", nil); return err },
		} {
			var apiErr *sdkerrors.APIError
			if err := call(); !errors.As(err, &apiErr) || apiErr.StatusCode != tc.status || apiErr.Code != tc.code {
				t.Fatalf("error = %#v", err)
			}
		}
	}
}
