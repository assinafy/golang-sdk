package resources

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
	"github.com/assinafy/golang-sdk/internal"
	"github.com/assinafy/golang-sdk/models"
)

func TestSignerContractMethods(t *testing.T) {
	t.Run("confirm data returns signer", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut || r.URL.EscapedPath() != "/documents/d1/signers/confirm-data" {
				t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
			}
			assertNoClientAuth(t, r)
			if r.URL.Query().Get("signer-access-code") != "code1" {
				t.Errorf("query = %v", r.URL.Query())
			}
			want := map[string]any{"full_name": "New Name"}
			if got := decodeContractBody(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
			writeJSONResponse(w, `{"status":200,"data":{"id":"s1","full_name":"New Name","has_accepted_terms":false}}`)
		})

		name := "New Name"
		signer, err := NewSignerResource(httpClient, "acc1").ConfirmDataAndGet(context.Background(), "d1", "code1", &models.ConfirmSignerDataRequest{FullName: &name})
		if err != nil {
			t.Fatal(err)
		}
		if signer.ID != "s1" || signer.FullName != name {
			t.Fatalf("signer = %+v", signer)
		}
	})

	t.Run("accept terms envelope only", func(t *testing.T) {
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut || r.URL.Path != "/signers/accept-terms" || r.URL.Query().Get("signer-access-code") != "code1" {
				t.Errorf("request = %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
			}
			writeJSONResponse(w, `{"status":200,"message":""}`)
		})

		if err := NewSignerResource(httpClient, "").AcceptTermsOnly(context.Background(), "code1"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("upload png with reuse", func(t *testing.T) {
		image := []byte("\x89PNG\r\n\x1a\n")
		httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.EscapedPath() != "/signature" {
				t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
			}
			assertNoClientAuth(t, r)
			if r.Header.Get("Content-Type") != "image/png" {
				t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
			}
			if got := r.URL.Query(); !reflect.DeepEqual(got, url.Values{
				"reuse": {"true"}, "signer-access-code": {"code1"}, "type": {"signature"},
			}) {
				t.Errorf("query = %v", got)
			}
			got, _ := io.ReadAll(r.Body)
			if !reflect.DeepEqual(got, image) {
				t.Errorf("body = %v", got)
			}
			writeJSONResponse(w, `{"status":200,"message":""}`)
		})

		if err := NewSignerResource(httpClient, "").UploadSignatureWithReuse(context.Background(), "code1", "signature", true, image); err != nil {
			t.Fatal(err)
		}
	})
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
			t.Errorf("decode: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
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

func TestSignerCreateUsesDefaultAccount(t *testing.T) {
	httpClient, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/acc1/signers" {
			t.Errorf("path = %q", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if body["full_name"] != "Bob" {
			t.Errorf("full_name = %v", body["full_name"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":200,"data":{"id":"s1","full_name":"Bob","has_accepted_terms":false}}`)
	})

	r := NewSignerResource(httpClient, "acc1")
	signer, err := r.Create(context.Background(), "", &models.CreateSignerRequest{FullName: "Bob"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if signer.ID != "s1" {
		t.Errorf("ID = %q", signer.ID)
	}
}

func TestSignatureUploadValidation(t *testing.T) {
	r := NewSignerResource(internal.NewHTTPClient("http://127.0.0.1:1", "key", "", time.Second), "")
	for _, image := range [][]byte{nil, []byte("not a png")} {
		if err := r.UploadSignature(context.Background(), "code", "signature", image); !errors.Is(err, sdkerrors.ErrInvalidInput) {
			t.Fatalf("error = %v", err)
		}
	}
}
