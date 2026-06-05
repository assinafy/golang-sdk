// Package internal contains the low-level HTTP transport used by the SDK
// resources. It is not part of the public API and may change without notice.
package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/assinafy/golang-sdk/errors"
)

const userAgent = "assinafy-go-sdk/1.0.0"

// HTTPClient is a thin JSON/multipart client around net/http.
type HTTPClient struct {
	baseURL string
	headers http.Header
	httpc   *http.Client
}

// NewHTTPClient builds an HTTPClient with the given credentials and timeout.
// If apiKey is set it takes precedence over token.
func NewHTTPClient(baseURL, apiKey, token string, timeout time.Duration) *HTTPClient {
	headers := http.Header{}
	headers.Set("Accept", "application/json")
	headers.Set("User-Agent", userAgent)
	switch {
	case apiKey != "":
		headers.Set("X-Api-Key", apiKey)
	case token != "":
		headers.Set("Authorization", "Bearer "+token)
	}

	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		headers: headers,
		httpc:   &http.Client{Timeout: timeout},
	}
}

// Request is a fluent builder for outgoing API requests.
type Request struct {
	client  *HTTPClient
	method  string
	path    string
	query   url.Values
	headers http.Header
	body    any
	rawBody []byte
}

// NewRequest creates a Request bound to this client. Path must start with "/".
func (c *HTTPClient) NewRequest(method, path string) *Request {
	return &Request{
		client:  c,
		method:  method,
		path:    path,
		query:   url.Values{},
		headers: http.Header{},
	}
}

// WithQuery sets a single query parameter, dropping it if value is empty.
func (r *Request) WithQuery(key, value string) *Request {
	if value != "" {
		r.query.Set(key, value)
	}
	return r
}

// WithHeader sets a request-specific header (overrides client defaults).
func (r *Request) WithHeader(key, value string) *Request {
	r.headers.Set(key, value)
	return r
}

// WithBody sets a JSON-encoded body.
func (r *Request) WithBody(body any) *Request {
	r.body = body
	r.rawBody = nil
	return r
}

// WithRawBody sets a raw byte body. Caller must also call WithHeader for Content-Type.
func (r *Request) WithRawBody(body []byte) *Request {
	r.rawBody = body
	r.body = nil
	return r
}

// Response carries the parsed result and surrounding HTTP metadata.
type Response struct {
	StatusCode int
	Headers    http.Header
}

// Execute runs the request and decodes the response data envelope into result.
func (r *Request) Execute(ctx context.Context, result any) (*Response, error) {
	return r.client.do(ctx, r, result)
}

func (c *HTTPClient) do(ctx context.Context, r *Request, result any) (*Response, error) {
	target := c.baseURL + r.path
	if len(r.query) > 0 {
		target += "?" + r.query.Encode()
	}

	var body io.Reader
	contentType := ""
	switch {
	case r.rawBody != nil:
		body = bytes.NewReader(r.rawBody)
	case r.body != nil:
		buf, err := json.Marshal(r.body)
		if err != nil {
			return nil, fmt.Errorf("assinafy: marshal request body: %w", err)
		}
		body = bytes.NewReader(buf)
		contentType = "application/json"
	}

	req, err := http.NewRequestWithContext(ctx, r.method, target, body)
	if err != nil {
		return nil, fmt.Errorf("assinafy: build request: %w", err)
	}

	for k, vs := range c.headers {
		req.Header[k] = append(req.Header[k], vs...)
	}
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, vs := range r.headers {
		req.Header[k] = vs
	}

	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, &errors.NetworkError{Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	return parseResponse(resp, result)
}

// UploadMultipart sends a multipart/form-data POST with a single file part.
func (c *HTTPClient) UploadMultipart(ctx context.Context, path, fieldName, fileName string, fileContent []byte, formFields map[string]string, result any) (*Response, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	part, err := w.CreateFormFile(fieldName, fileName)
	if err != nil {
		return nil, fmt.Errorf("assinafy: create form file: %w", err)
	}
	if _, err := part.Write(fileContent); err != nil {
		return nil, fmt.Errorf("assinafy: write form file: %w", err)
	}
	for k, v := range formFields {
		if err := w.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("assinafy: write form field %q: %w", k, err)
		}
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("assinafy: close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return nil, fmt.Errorf("assinafy: build request: %w", err)
	}
	for k, vs := range c.headers {
		req.Header[k] = append(req.Header[k], vs...)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, &errors.NetworkError{Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	return parseResponse(resp, result)
}

// Download fetches a path as raw bytes (used for artifact/page/thumbnail downloads).
func (c *HTTPClient) Download(ctx context.Context, path string, query map[string]string) ([]byte, error) {
	target := c.baseURL + path
	if len(query) > 0 {
		q := url.Values{}
		for k, v := range query {
			if v != "" {
				q.Set(k, v)
			}
		}
		if len(q) > 0 {
			target += "?" + q.Encode()
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("assinafy: build request: %w", err)
	}
	for k, vs := range c.headers {
		req.Header[k] = append(req.Header[k], vs...)
	}

	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, &errors.NetworkError{Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		// parseResponse always returns a non-nil error for status >= 400.
		_, err := parseResponse(resp, nil)
		return nil, err
	}

	return io.ReadAll(resp.Body)
}

// envelope mirrors the shared response wrapper documented at
// https://api.assinafy.com.br/v1/docs.
type envelope struct {
	Status  json.RawMessage `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func parseResponse(resp *http.Response, result any) (*Response, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("assinafy: read response: %w", err)
	}

	out := &Response{StatusCode: resp.StatusCode, Headers: resp.Header}
	body = bytes.TrimSpace(body)

	if len(body) == 0 {
		if resp.StatusCode >= 400 {
			return nil, &errors.APIError{StatusCode: resp.StatusCode, Message: http.StatusText(resp.StatusCode)}
		}
		return out, nil
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		if resp.StatusCode >= 400 {
			return nil, &errors.APIError{StatusCode: resp.StatusCode, Message: string(body)}
		}
		if result == nil {
			return nil, fmt.Errorf("assinafy: parse response: %w", err)
		}
		if err := json.Unmarshal(body, result); err != nil {
			return nil, fmt.Errorf("assinafy: unmarshal response: %w", err)
		}
		return out, nil
	}

	envStatus := resp.StatusCode
	hasEnvelopeStatus := false
	if len(env.Status) > 0 {
		var s int
		if err := json.Unmarshal(env.Status, &s); err == nil {
			envStatus = s
			hasEnvelopeStatus = true
		}
	}
	isEnvelope := hasEnvelopeStatus && (len(env.Data) > 0 || env.Message != "")

	if resp.StatusCode >= 400 || (isEnvelope && envStatus >= 400) {
		apiErr := &errors.APIError{StatusCode: envStatus, Message: env.Message}
		if apiErr.Message == "" {
			apiErr.Message = http.StatusText(envStatus)
		}
		if hasDataPayload(env.Data) {
			var data any
			if json.Unmarshal(env.Data, &data) == nil {
				apiErr.Data = data
			}
		}
		return nil, apiErr
	}

	if result == nil {
		return out, nil
	}

	switch {
	case isEnvelope && hasDataPayload(env.Data):
		if err := json.Unmarshal(env.Data, result); err != nil {
			return nil, fmt.Errorf("assinafy: unmarshal response data: %w", err)
		}
	case !isEnvelope:
		if err := json.Unmarshal(body, result); err != nil {
			return nil, fmt.Errorf("assinafy: unmarshal response: %w", err)
		}
	}

	return out, nil
}

func hasDataPayload(data json.RawMessage) bool {
	data = bytes.TrimSpace(data)
	return len(data) > 0 && !bytes.Equal(data, []byte("null"))
}
