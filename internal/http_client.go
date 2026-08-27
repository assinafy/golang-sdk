// Package internal contains the low-level HTTP transport used by the SDK
// resources. It is not part of the public API and may change without notice.
package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	sdkerrors "github.com/assinafy/golang-sdk/errors"
)

const userAgent = "assinafy-go-sdk"

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
		httpc: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return errors.New("stopped after 10 consecutive requests")
				}
				if len(via) > 0 && (req.URL.Scheme != via[0].URL.Scheme || !strings.EqualFold(req.URL.Host, via[0].URL.Host)) {
					downgrade := strings.EqualFold(via[0].URL.Scheme, "https") && !strings.EqualFold(req.URL.Scheme, "https")
					if downgrade || (via[0].Method != http.MethodGet && via[0].Method != http.MethodHead) || req.Body != nil {
						return sdkerrors.ErrUnsafeRedirect
					}
					req.Header.Del("Authorization")
					req.Header.Del("X-Api-Key")
					req.Header.Del("Referer")
					query := req.URL.Query()
					query.Del("signer-access-code")
					req.URL.RawQuery = query.Encode()
				}
				return nil
			},
		},
	}
}

// BaseURL returns the API base URL the client was constructed with (with any
// trailing slash trimmed). It is used to build browser-facing URLs.
func (c *HTTPClient) BaseURL() string { return c.baseURL }

// Request is a fluent builder for outgoing API requests.
type Request struct {
	client  *HTTPClient
	method  string
	path    string
	query   url.Values
	headers http.Header
	noAuth  bool
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

// WithoutAuth omits the client's API-key and bearer-token headers. It is used
// for public and signer-access-code operations.
func (r *Request) WithoutAuth() *Request {
	r.noAuth = true
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
	if err := validatePath(r.path); err != nil {
		return nil, err
	}
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
		if r.noAuth && (k == "Authorization" || k == "X-Api-Key") {
			continue
		}
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
		return nil, newNetworkError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	return parseResponse(resp, result)
}

// UploadMultipart sends a multipart/form-data POST with a single file part.
func (c *HTTPClient) UploadMultipart(ctx context.Context, path, fieldName, fileName string, fileContent []byte, formFields map[string]string, result any) (*Response, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}
	if strings.TrimSpace(fieldName) == "" {
		return nil, fmt.Errorf("%w: multipart field name is empty", sdkerrors.ErrInvalidInput)
	}
	if strings.TrimSpace(fileName) == "" {
		return nil, fmt.Errorf("%w: upload file name is empty", sdkerrors.ErrInvalidInput)
	}
	if len(fileContent) == 0 {
		return nil, fmt.Errorf("%w: upload file is empty", sdkerrors.ErrInvalidInput)
	}

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
		return nil, newNetworkError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	return parseResponse(resp, result)
}

// Download fetches a path as raw bytes (used for artifact/page/thumbnail downloads).
func (c *HTTPClient) Download(ctx context.Context, path string, query map[string]string) ([]byte, error) {
	return c.download(ctx, path, query, false)
}

// DownloadUnauthenticated downloads raw bytes without the client's API-key or
// bearer-token header.
func (c *HTTPClient) DownloadUnauthenticated(ctx context.Context, path string, query map[string]string) ([]byte, error) {
	return c.download(ctx, path, query, true)
}

func (c *HTTPClient) download(ctx context.Context, path string, query map[string]string, noAuth bool) ([]byte, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}
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
		if noAuth && (k == "Authorization" || k == "X-Api-Key") {
			continue
		}
		req.Header[k] = append(req.Header[k], vs...)
	}
	req.Header.Set("Accept", "*/*")

	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, newNetworkError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		// parseResponse always returns a non-nil error outside the 2xx range.
		_, err := parseResponse(resp, nil)
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newNetworkError(fmt.Errorf("read response body: %w", err))
	}
	return body, nil
}

func newNetworkError(err error) *sdkerrors.NetworkError {
	if urlErr, ok := err.(*url.Error); ok {
		clean := *urlErr
		clean.URL = redactSignerAccessCode(clean.URL)
		err = &clean
	}
	return &sdkerrors.NetworkError{Err: err}
}

func redactSignerAccessCode(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		if i := strings.IndexByte(rawURL, '?'); i >= 0 {
			return rawURL[:i] + "?[REDACTED]"
		}
		return rawURL
	}
	query := u.Query()
	if _, ok := query["signer-access-code"]; !ok {
		return rawURL
	}
	query.Set("signer-access-code", "REDACTED")
	u.RawQuery = query.Encode()
	return u.String()
}

func validatePath(path string) error {
	if path == "/" {
		return nil
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("%w: request path must start with a slash", sdkerrors.ErrInvalidInput)
	}
	for _, segment := range strings.Split(path[1:], "/") {
		decoded, err := url.PathUnescape(segment)
		if err != nil {
			return fmt.Errorf("%w: request path contains an invalid parameter", sdkerrors.ErrInvalidInput)
		}
		for _, part := range strings.Split(strings.ReplaceAll(decoded, "\\", "/"), "/") {
			if strings.TrimSpace(part) == "" || part == "." || part == ".." {
				return fmt.Errorf("%w: request path contains an invalid parameter", sdkerrors.ErrInvalidInput)
			}
		}
	}
	return nil
}

// envelope mirrors the shared response wrapper documented at
// https://api.assinafy.com.br/v1/docs.
type envelope struct {
	Status       json.RawMessage                 `json:"status"`
	Message      string                          `json:"message"`
	Data         json.RawMessage                 `json:"data"`
	Restrictions []sdkerrors.DeletionRestriction `json:"restrictions"`
}

func parseResponse(resp *http.Response, result any) (*Response, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newNetworkError(fmt.Errorf("read response body: %w", err))
	}

	out := &Response{StatusCode: resp.StatusCode, Headers: resp.Header}
	body = bytes.TrimSpace(body)

	if len(body) == 0 {
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return out, &sdkerrors.APIError{StatusCode: resp.StatusCode, Message: http.StatusText(resp.StatusCode), Headers: resp.Header.Clone()}
		}
		return out, nil
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return out, &sdkerrors.APIError{StatusCode: resp.StatusCode, Message: string(body), Headers: resp.Header.Clone()}
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

	httpFailed := resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices
	envelopeFailed := isEnvelope && (envStatus < http.StatusOK || envStatus >= http.StatusMultipleChoices)
	if httpFailed || envelopeFailed {
		statusCode := resp.StatusCode
		if !httpFailed && envelopeFailed {
			statusCode = envStatus
		}
		apiErr := &sdkerrors.APIError{
			StatusCode:   statusCode,
			Message:      env.Message,
			Restrictions: env.Restrictions,
			Headers:      resp.Header.Clone(),
		}
		if apiErr.Message == "" {
			apiErr.Message = http.StatusText(statusCode)
		}
		if hasDataPayload(env.Data) {
			var data any
			if json.Unmarshal(env.Data, &data) == nil {
				apiErr.Data = data
			}
		}
		return out, apiErr
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
