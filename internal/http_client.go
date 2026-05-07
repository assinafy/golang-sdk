package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/assinafy/assinafy-go/errors"
)

type HTTPClient struct {
	baseURL    string
	apiKey     string
	token      string
	httpClient *http.Client
	headers    map[string]string
}

func NewHTTPClient(baseURL, apiKey, token string, timeout time.Duration) *HTTPClient {
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
		"User-Agent":   "assinafy-go-sdk",
	}

	if apiKey != "" {
		headers["X-Api-Key"] = apiKey
	} else if token != "" {
		headers["Authorization"] = "Bearer " + token
	}

	return &HTTPClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		headers: headers,
	}
}

func (c *HTTPClient) NewRequest(method, path string) *Request {
	return &Request{
		client:      c,
		method:      method,
		path:        path,
		queryParams: make(map[string]string),
		headers:     make(map[string]string),
	}
}

type Request struct {
	client      *HTTPClient
	method      string
	path        string
	queryParams map[string]string
	headers     map[string]string
	body        interface{}
}

func (r *Request) WithQuery(key, value string) *Request {
	r.queryParams[key] = value
	return r
}

func (r *Request) WithHeader(key, value string) *Request {
	r.headers[key] = value
	return r
}

func (r *Request) WithBody(body interface{}) *Request {
	r.body = body
	return r
}

func (r *Request) Execute(ctx context.Context, responseStruct interface{}) (*Response, error) {
	return r.client.Do(ctx, r, responseStruct)
}

type Response struct {
	StatusCode int
	Data       interface{}
	Headers    http.Header
}

func (c *HTTPClient) Do(ctx context.Context, req *Request, result interface{}) (*Response, error) {
	urlStr := c.baseURL + req.path
	if len(req.queryParams) > 0 {
		query := url.Values{}
		for k, v := range req.queryParams {
			if v != "" {
				query.Set(k, v)
			}
		}
		if len(query) > 0 {
			urlStr += "?" + query.Encode()
		}
	}

	var bodyReader io.Reader
	if req.body != nil {
		data, err := json.Marshal(req.body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.method, urlStr, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range c.headers {
		httpReq.Header.Set(k, v)
	}
	for k, v := range req.headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, &errors.NetworkError{OriginalError: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp struct {
		Status  int             `json:"status"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(respBody))
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Status >= 400 {
		apiError := &errors.APIError{
			StatusCode: apiResp.Status,
			Message:    apiResp.Message,
		}
		if len(apiResp.Data) > 0 && !bytes.HasPrefix(apiResp.Data, []byte("[]")) {
			var data interface{}
			if err := json.Unmarshal(apiResp.Data, &data); err == nil {
				apiError.Data = data
			}
		}
		return nil, apiError
	}

	if result != nil && len(apiResp.Data) > 0 {
		if err := json.Unmarshal(apiResp.Data, result); err != nil {
			if !bytes.HasPrefix(apiResp.Data, []byte("[]")) {
				return nil, fmt.Errorf("failed to unmarshal response data: %w", err)
			}
		}
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Data:       result,
		Headers:    resp.Header,
	}, nil
}

func (c *HTTPClient) UploadMultipart(ctx context.Context, path string, fieldName, fileName string, fileContent []byte, formFields map[string]string, result interface{}) (*Response, error) {
	urlStr := c.baseURL + path

	body, contentType, err := createMultipartBody(fieldName, fileName, fileContent, formFields)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &errors.NetworkError{OriginalError: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp struct {
		Status  int             `json:"status"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Status >= 400 {
		return nil, &errors.APIError{
			StatusCode: apiResp.Status,
			Message:    apiResp.Message,
		}
	}

	if result != nil && len(apiResp.Data) > 0 {
		if err := json.Unmarshal(apiResp.Data, result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response data: %w", err)
		}
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Data:       result,
		Headers:    resp.Header,
	}, nil
}

func (c *HTTPClient) Download(ctx context.Context, path string, queryParams map[string]string) ([]byte, error) {
	urlStr := c.baseURL + path
	if len(queryParams) > 0 {
		query := url.Values{}
		for k, v := range queryParams {
			query.Set(k, v)
		}
		urlStr += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &errors.NetworkError{OriginalError: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func createMultipartBody(fieldName, fileName string, fileContent []byte, formFields map[string]string) (io.Reader, string, error) {
	var buf bytes.Buffer
	writer := newMultipartWriter(&buf)

	writer.WriteField(fieldName, fileName)
	for k, v := range formFields {
		writer.WriteField(k, v)
	}
	writer.WriteFile("file", fileName, "application/pdf", fileContent)

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return &buf, writer.FormDataContentType(), nil
}

type multipartWriter struct {
	w   *bytes.Buffer
	pw  *io.PipeWriter
	err error
}

func newMultipartWriter(w *bytes.Buffer) *multipartWriter {
	return &multipartWriter{w: w}
}

func (m *multipartWriter) WriteField(key, value string) {
	m.w.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	m.w.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=\"%s\"\r\n\r\n", key))
	m.w.WriteString(value + "\r\n")
}

func (m *multipartWriter) WriteFile(fieldName, fileName, contentType string, data []byte) {
	m.w.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	m.w.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=\"%s\"; filename=\"%s\"\r\n", fieldName, fileName))
	m.w.WriteString(fmt.Sprintf("Content-Type: %s\r\n\r\n", contentType))
	m.w.Write(data)
	m.w.WriteString("\r\n")
}

func (m *multipartWriter) Close() error {
	m.w.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	return nil
}

func (m *multipartWriter) FormDataContentType() string {
	return fmt.Sprintf("multipart/form-data; boundary=%s", boundary)
}

const boundary = "---------------------------Boundary"
