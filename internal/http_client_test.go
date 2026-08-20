package internal

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type failingReadCloser struct{}

func (failingReadCloser) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (failingReadCloser) Close() error             { return nil }

func TestHTTPClientParsesWrappedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":200,"message":"","data":{"id":"doc_123"}}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	var result struct {
		ID string `json:"id"`
	}

	_, err := client.NewRequest(http.MethodGet, "/documents/doc_123").Execute(context.Background(), &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "doc_123" {
		t.Fatalf("unexpected id: %s", result.ID)
	}
}

func TestHTTPClientBaseURL(t *testing.T) {
	if got := NewHTTPClient("https://example.com/v1/", "", "", time.Second).BaseURL(); got != "https://example.com/v1" {
		t.Fatalf("BaseURL() = %q", got)
	}
}

func TestHTTPClientEncodesHeadersQueryAndJSONBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/documents" || r.URL.Query().Get("search") != "a b" {
			t.Errorf("request = %s %s", r.Method, r.URL.String())
		}
		if _, exists := r.URL.Query()["empty"]; exists {
			t.Error("empty query value must be omitted")
		}
		if r.Header.Get("Authorization") != "Bearer token" || r.Header.Get("X-Api-Key") != "" {
			t.Errorf("authentication headers = %v", r.Header)
		}
		if r.Header.Get("Accept") != "application/json" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("User-Agent") == "" {
			t.Errorf("content headers = %v", r.Header)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["name"] != "contract" {
			t.Errorf("body = %v, error = %v", body, err)
		}
		_, _ = w.Write([]byte(`{"status":200,"data":null}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "token", time.Second)
	_, err := client.NewRequest(http.MethodPost, "/documents").
		WithQuery("search", "a b").
		WithQuery("empty", "").
		WithBody(map[string]string{"name": "contract"}).
		Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestHTTPClientPrefersAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "key" || r.Header.Get("Authorization") != "" {
			t.Errorf("authentication headers = %v", r.Header)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "token", time.Second)
	if _, err := client.NewRequest(http.MethodGet, "/").Execute(context.Background(), nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestHTTPClientOmitsConfiguredAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "" || r.Header.Get("Authorization") != "" {
			t.Errorf("authentication headers = %v", r.Header)
		}
		if r.URL.Path == "/download" {
			_, _ = w.Write([]byte("file"))
			return
		}
		_, _ = w.Write([]byte(`{"status":200,"data":null}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "secret", "", time.Second)
	if _, err := client.NewRequest(http.MethodGet, "/public").WithoutAuth().Execute(context.Background(), nil); err != nil {
		t.Fatalf("WithoutAuth: %v", err)
	}
	got, err := client.DownloadUnauthenticated(context.Background(), "/download", nil)
	if err != nil || string(got) != "file" {
		t.Fatalf("DownloadUnauthenticated = %q, %v", got, err)
	}
}

func TestHTTPClientRejectsTerminalRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	for _, call := range []struct {
		name string
		fn   func() error
	}{
		{"json", func() error {
			_, err := client.NewRequest(http.MethodGet, "/json").Execute(context.Background(), nil)
			return err
		}},
		{"download", func() error {
			_, err := client.Download(context.Background(), "/download", nil)
			return err
		}},
	} {
		t.Run(call.name, func(t *testing.T) {
			var apiErr *sdkerrors.APIError
			if err := call.fn(); !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusFound {
				t.Fatalf("error = %#v, want APIError status 302", err)
			}
		})
	}
}

func TestHTTPClientRedactsSignerCodeFromNetworkErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()
	client := NewHTTPClient(server.URL, "", "", time.Second)
	const secret = "signer-secret-code"

	for _, call := range []struct {
		name string
		fn   func(context.Context) error
	}{
		{"json", func(ctx context.Context) error {
			_, err := client.NewRequest(http.MethodGet, "/sign").
				WithQuery("signer-access-code", secret).
				Execute(ctx, nil)
			return err
		}},
		{"download", func(ctx context.Context) error {
			_, err := client.DownloadUnauthenticated(ctx, "/signature/signature", map[string]string{"signer-access-code": secret})
			return err
		}},
	} {
		t.Run(call.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err := call.fn(ctx)
			var networkErr *sdkerrors.NetworkError
			if !errors.As(err, &networkErr) || !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %#v, want NetworkError wrapping context.Canceled", err)
			}
			if strings.Contains(err.Error(), secret) {
				t.Fatalf("network error leaked signer code: %v", err)
			}
		})
	}
}

func TestHTTPClientClassifiesResponseBodyReadFailuresAsNetworkErrors(t *testing.T) {
	client := NewHTTPClient("https://example.test", "key", "", time.Second)
	client.httpc.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       failingReadCloser{},
			Request:    req,
		}, nil
	})

	for _, call := range []struct {
		name string
		fn   func() error
	}{
		{"json", func() error {
			_, err := client.NewRequest(http.MethodGet, "/json").Execute(context.Background(), nil)
			return err
		}},
		{"download", func() error {
			_, err := client.Download(context.Background(), "/download", nil)
			return err
		}},
	} {
		t.Run(call.name, func(t *testing.T) {
			err := call.fn()
			var networkErr *sdkerrors.NetworkError
			if !errors.As(err, &networkErr) || !errors.Is(err, io.ErrUnexpectedEOF) || !sdkerrors.IsRetryable(err) {
				t.Fatalf("error = %#v, want retryable NetworkError wrapping io.ErrUnexpectedEOF", err)
			}
		})
	}
}

func TestHTTPClientParsesBareObjectWithStatusField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"doc_123","status":"uploaded"}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}

	_, err := client.NewRequest(http.MethodGet, "/documents/doc_123").Execute(context.Background(), &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "uploaded" {
		t.Fatalf("unexpected status: %s", result.Status)
	}
}

func TestHTTPClientParsesBareArrayResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"doc_123"}]`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	var result []struct {
		ID string `json:"id"`
	}

	_, err := client.NewRequest(http.MethodGet, "/documents").Execute(context.Background(), &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || result[0].ID != "doc_123" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestHTTPClientParsesBareObjectWithNumericStatusField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"job_123","status":1}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	var result struct {
		ID     string `json:"id"`
		Status int    `json:"status"`
	}

	_, err := client.NewRequest(http.MethodGet, "/jobs/job_123").Execute(context.Background(), &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "job_123" || result.Status != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestHTTPClientReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":400,"message":"bad request","data":[]}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	resp, err := client.NewRequest(http.MethodGet, "/bad").Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *sdkerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest || apiErr.Message != "bad request" {
		t.Fatalf("unexpected APIError: %#v", apiErr)
	}
	if resp == nil || resp.StatusCode != http.StatusBadRequest || apiErr.Headers.Get("Retry-After") != "2" {
		t.Fatalf("missing error response metadata: response=%#v error=%#v", resp, apiErr)
	}
}

func TestHTTPClientKeepsHTTPErrorStatusWhenEnvelopeStatusIsSuccessful(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":200,"message":"upstream failure","data":null}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	_, err := client.NewRequest(http.MethodGet, "/bad").Execute(context.Background(), nil)
	var apiErr *sdkerrors.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("error = %#v, want APIError status 500", err)
	}
}

func TestHTTPClientKeepsRetryableHTTPStatusWhenEnvelopeAlsoFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":400,"message":"upstream request failed","data":null}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	_, err := client.NewRequest(http.MethodGet, "/bad").Execute(context.Background(), nil)
	var apiErr *sdkerrors.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusInternalServerError || !sdkerrors.IsRetryable(err) {
		t.Fatalf("error = %#v, want retryable APIError status 500", err)
	}
}

func TestHTTPClientUsesActualHTTPStatusText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":200,"message":"","data":null}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	_, err := client.NewRequest(http.MethodGet, "/bad").Execute(context.Background(), nil)
	var apiErr *sdkerrors.APIError
	if !errors.As(err, &apiErr) || apiErr.Message != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("error = %#v, want HTTP 500 status text", err)
	}
}

func TestHTTPClientStripsCredentialsOnCrossOriginRedirect(t *testing.T) {
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Api-Key"); got != "" {
			t.Errorf("redirect leaked X-Api-Key %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("redirect leaked Authorization %q", got)
		}
		if got := r.Header.Get("Referer"); got != "" {
			t.Errorf("redirect leaked signer code in Referer %q", got)
		}
		_, _ = w.Write([]byte(`{"id":"ok"}`))
	}))
	defer destination.Close()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusFound)
	}))
	defer source.Close()

	client := NewHTTPClient(source.URL, "key", "", time.Second)
	var result struct {
		ID string `json:"id"`
	}
	if _, err := client.NewRequest(http.MethodGet, "/redirect").
		WithQuery("signer-access-code", "secret").
		Execute(context.Background(), &result); err != nil {
		t.Fatalf("redirect request: %v", err)
	}
	if result.ID != "ok" {
		t.Errorf("redirect result = %+v", result)
	}
}

func TestHTTPClientLimitsRedirects(t *testing.T) {
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		http.Redirect(w, r, server.URL+"/loop", http.StatusFound)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	_, err := client.NewRequest(http.MethodGet, "/loop").Execute(context.Background(), nil)
	var networkErr *sdkerrors.NetworkError
	if !errors.As(err, &networkErr) || !strings.Contains(err.Error(), "stopped after 10 consecutive requests") {
		t.Fatalf("error = %#v, want redirect-limit NetworkError", err)
	}
	if requests != 10 {
		t.Fatalf("requests = %d, want 10", requests)
	}
}

func TestRequestWithRawBody(t *testing.T) {
	body := []byte{0x01, 0x02, 0x03}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		if string(got) != string(body) {
			t.Errorf("unexpected body: %v", got)
			http.Error(w, "unexpected body", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Content-Type") != "image/png" {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
			http.Error(w, "unexpected content type", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":200,"message":"","data":[]}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	req := client.NewRequest(http.MethodPost, "/signature")
	req.WithHeader("Content-Type", "image/png")
	req.WithRawBody(body)

	if _, err := req.Execute(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPClientUploadMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/accounts/a/documents" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
			http.Error(w, "invalid multipart form", http.StatusBadRequest)
			return
		}
		if got := r.FormValue("name"); got != "contract.pdf" {
			t.Errorf("name = %q", got)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Errorf("FormFile: %v", err)
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		defer func() { _ = file.Close() }()
		contents, _ := io.ReadAll(file)
		if header.Filename != "contract.pdf" || string(contents) != "PDF" {
			t.Errorf("file = %q %q", header.Filename, contents)
		}
		_, _ = w.Write([]byte(`{"status":200,"data":{"id":"doc1"}}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	var result struct {
		ID string `json:"id"`
	}
	_, err := client.UploadMultipart(context.Background(), "/accounts/a/documents", "file", "contract.pdf", []byte("PDF"), map[string]string{"name": "contract.pdf"}, &result)
	if err != nil || result.ID != "doc1" {
		t.Fatalf("UploadMultipart result = %+v, error = %v", result, err)
	}
}

func TestHTTPClientDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/artifact" || r.URL.Query().Get("kind") != "original" {
			t.Errorf("URL = %s", r.URL.String())
		}
		if _, exists := r.URL.Query()["empty"]; exists {
			t.Error("empty query value must be omitted")
		}
		_, _ = w.Write([]byte("PDF"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	data, err := client.Download(context.Background(), "/artifact", map[string]string{"kind": "original", "empty": ""})
	if err != nil || string(data) != "PDF" {
		t.Fatalf("Download = %q, error = %v", data, err)
	}
}

func TestHTTPClientDownloadReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":404,"message":"missing","data":null}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	_, err := client.Download(context.Background(), "/missing", nil)
	var apiErr *sdkerrors.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("Download error = %#v", err)
	}
}

func TestHTTPClientRejectsUnencodableBody(t *testing.T) {
	client := NewHTTPClient("https://example.com", "key", "", time.Second)
	_, err := client.NewRequest(http.MethodPost, "/bad").WithBody(make(chan int)).Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected JSON marshal error")
	}
}

func TestHTTPClientReturnsApplicationLevelAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Request-ID", "request-1")
		_, _ = w.Write([]byte(`{"status":429,"message":"slow down","data":{"limit":1}}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	resp, err := client.NewRequest(http.MethodGet, "/limited").Execute(context.Background(), nil)
	var apiErr *sdkerrors.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusTooManyRequests || apiErr.Headers.Get("X-Request-ID") != "request-1" {
		t.Fatalf("response = %#v, error = %#v", resp, err)
	}
	if apiErr.Data == nil {
		t.Fatal("expected API error data")
	}
}
