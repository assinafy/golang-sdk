package internal

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
)

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
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":400,"message":"bad request","data":[]}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", "", time.Second)
	_, err := client.NewRequest(http.MethodGet, "/bad").Execute(context.Background(), nil)
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
}

func TestRequestWithRawBody(t *testing.T) {
	body := []byte{0x01, 0x02, 0x03}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if string(got) != string(body) {
			t.Fatalf("unexpected body: %v", got)
		}
		if r.Header.Get("Content-Type") != "image/png" {
			t.Fatalf("unexpected content type: %s", r.Header.Get("Content-Type"))
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
