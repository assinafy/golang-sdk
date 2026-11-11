package internal

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/v2/errors"
)

func TestTransportWithApplicationDefaults(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	http.DefaultTransport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("wrapped transport must not be invoked during construction")
		return nil, nil
	})
	transport := Transport()
	if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 || transport.DialContext == nil || !transport.ForceAttemptHTTP2 {
		t.Fatal("fallback does not retain secure standard transport defaults")
	}
	configured := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, ServerName: "example.test"}}
	http.DefaultTransport = configured
	transport = Transport()
	if transport.TLSClientConfig.MinVersion != tls.VersionTLS13 || transport.TLSClientConfig.ServerName != "example.test" || transport.TLSClientConfig == configured.TLSClientConfig {
		t.Fatal("transport did not preserve and clone the application's TLS settings")
	}
	configured.TLSClientConfig.MinVersion = tls.VersionTLS11
	if Transport().TLSClientConfig.MinVersion != tls.VersionTLS12 || configured.TLSClientConfig.MinVersion != tls.VersionTLS11 {
		t.Fatal("minimum TLS version must be raised only on the clone")
	}
}

func TestSingleUseRequestCannotRewindBody(t *testing.T) {
	c := NewHTTPClient("https://example.test", "", "", time.Second)
	c.httpc.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.GetBody != nil || req.Body == nil || req.Body == http.NoBody {
			t.Fatal("single-use request can be replayed")
		}
		body, err := io.ReadAll(req.Body)
		if err != nil || string(body) != "{}" {
			t.Fatalf("body = %q, error = %v", body, err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"status":200,"data":{}}`))}, nil
	})
	if _, err := c.NewRequest(http.MethodPost, "/embed/sessions/dummy/exchange").WithoutReplay().WithBody(struct{}{}).Execute(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
}

func TestTransportErrorsRedactPathTokens(t *testing.T) {
	for _, path := range []string{"/v1/embed/sessions/private-token/exchange", "/v1/signers/downloads/private-token/resend"} {
		cause := fmt.Errorf("bad redirect to https://example.test%s?other=hidden: %w", path, io.ErrUnexpectedEOF)
		for _, err := range []error{cause, &url.Error{Op: "Post", URL: "https://example.test" + path + "?key=hidden", Err: cause}} {
			clean := RedactTransportError(err)
			if strings.Contains(clean.Error(), "private-token") || !errors.Is(clean, io.ErrUnexpectedEOF) {
				t.Fatalf("unsafe error or lost cause: %v", clean)
			}
		}
	}
	if RedactTransportError(nil) != nil {
		t.Fatal("nil error changed")
	}
}

func TestTokenDownloadRetainsRedirect(t *testing.T) {
	var targetCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/expired" {
			targetCalls++
		}
		if r.Header.Get("X-Api-Key") != "" {
			t.Error("workspace key sent on token download")
		}
		http.Redirect(w, r, "/expired?token=dummy", http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	c := NewHTTPClient(srv.URL, "workspace-key", "", time.Second)
	if _, err := c.DownloadUnauthenticatedNoRedirect(context.Background(), "/signers/downloads/dummy"); err == nil || targetCalls != 0 {
		t.Fatalf("redirect followed: calls=%d err=%v", targetCalls, err)
	} else {
		var apiErr *sdkerrors.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusFound || apiErr.Headers.Get("Location") != "" || strings.Contains(err.Error(), "dummy") {
			t.Fatalf("redirect leaked its token or lost status: %v", err)
		}
	}
}
