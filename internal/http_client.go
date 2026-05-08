package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
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

	return c.parseResponse(resp, result)
}

func (c *HTTPClient) parseResponse(resp *http.Response, result interface{}) (*Response, error) {
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

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(fileContent); err != nil {
		return nil, fmt.Errorf("failed to write file content: %w", err)
	}

	for k, v := range formFields {
		if err := writer.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("failed to write form field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &errors.NetworkError{OriginalError: err}
	}
	defer resp.Body.Close()

	return c.parseResponse(resp, result)
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

func ParsePagination(headers map[string][]string) map[string]int {
	pagination := make(map[string]int)

	if v, ok := headers["X-Pagination-Current-Page"]; ok && len(v) > 0 {
		if page, err := strconv.Atoi(v[0]); err == nil {
			pagination["current_page"] = page
		}
	}
	if v, ok := headers["X-Pagination-Total-Count"]; ok && len(v) > 0 {
		if count, err := strconv.Atoi(v[0]); err == nil {
			pagination["total_count"] = count
		}
	}
	if v, ok := headers["X-Pagination-Page-Count"]; ok && len(v) > 0 {
		if count, err := strconv.Atoi(v[0]); err == nil {
			pagination["page_count"] = count
		}
	}
	if v, ok := headers["X-Pagination-Per-Page"]; ok && len(v) > 0 {
		if perPage, err := strconv.Atoi(v[0]); err == nil {
			pagination["per_page"] = perPage
		}
	}

	return pagination
}

func GetMimeType(filename string) string {
	mimeType := mime.TypeByExtension(filename)
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}
