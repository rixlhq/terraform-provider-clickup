package clickupclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// APIError is returned when the ClickUp API responds with a non-2xx status code.
type APIError struct {
	StatusCode int
	Body       string
}

// Error formats the APIError including status code and parsed body details.
func (e *APIError) Error() string {
	formatted := formatErrorBody(e.Body)
	if formatted != "" {
		return fmt.Sprintf("clickup API error %d: %s", e.StatusCode, formatted)
	}
	return fmt.Sprintf("clickup API error %d: %s", e.StatusCode, e.Body)
}

type rawAPIError struct {
	Err     string          `json:"err"`
	Error   string          `json:"error"`
	Message string          `json:"message"`
	ECODE   string          `json:"ECODE"`
	Code    string          `json:"code"`
	TraceID string          `json:"trace_id"`
	Errors  json.RawMessage `json:"errors"`
}

type fieldError struct {
	Field   string `json:"field"`
	Param   string `json:"param"`
	Key     string `json:"key"`
	Message string `json:"message"`
	Msg     string `json:"msg"`
	Err     string `json:"err"`
}

// extractPrimaryMessage returns the primary human-readable error message from rawAPIError.
func extractPrimaryMessage(raw *rawAPIError) string {
	if raw.Message != "" {
		return raw.Message
	}
	if raw.Err != "" {
		return raw.Err
	}
	return raw.Error
}

// extractFieldError formats a single field-level error into a descriptive message.
func extractFieldError(fe fieldError) string {
	f := fe.Field
	if f == "" {
		f = fe.Param
	}
	if f == "" {
		f = fe.Key
	}
	m := fe.Message
	if m == "" {
		m = fe.Msg
	}
	if m == "" {
		m = fe.Err
	}

	switch {
	case f != "" && m != "":
		return fmt.Sprintf("field %q: %s", f, m)
	case m != "":
		return m
	case f != "":
		return fmt.Sprintf("field %q", f)
	default:
		return ""
	}
}

// extractDetailParts unmarshals and formats any nested or field-level validation errors.
func extractDetailParts(errorsRaw json.RawMessage) []string {
	if len(errorsRaw) == 0 {
		return nil
	}

	var fieldErrors []fieldError
	if err := json.Unmarshal(errorsRaw, &fieldErrors); err == nil && len(fieldErrors) > 0 {
		var details []string
		for _, fe := range fieldErrors {
			if s := extractFieldError(fe); s != "" {
				details = append(details, s)
			}
		}
		return details
	}

	var strErrors []string
	if err := json.Unmarshal(errorsRaw, &strErrors); err == nil && len(strErrors) > 0 {
		return strErrors
	}

	return nil
}

// formatErrorBody parses a raw JSON error response and returns a formatted error summary.
func formatErrorBody(body string) string {
	trimmed := strings.TrimSpace(body)
	if !strings.HasPrefix(trimmed, "{") {
		return body
	}

	var raw rawAPIError
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return body
	}

	primaryMsg := extractPrimaryMessage(&raw)
	detailParts := extractDetailParts(raw.Errors)

	var b strings.Builder
	if primaryMsg != "" {
		b.WriteString(primaryMsg)
	}

	if len(detailParts) > 0 {
		if b.Len() > 0 {
			b.WriteString(" (")
			b.WriteString(strings.Join(detailParts, "; "))
			b.WriteString(")")
		} else {
			b.WriteString(strings.Join(detailParts, "; "))
		}
	}

	code := raw.ECODE
	if code == "" {
		code = raw.Code
	}
	if code != "" {
		if b.Len() > 0 {
			b.WriteString(" [code: ")
			b.WriteString(code)
			b.WriteString("]")
		} else {
			b.WriteString("[code: ")
			b.WriteString(code)
			b.WriteString("]")
		}
	}

	if raw.TraceID != "" {
		if b.Len() > 0 {
			b.WriteString(" [trace_id: ")
			b.WriteString(raw.TraceID)
			b.WriteString("]")
		} else {
			b.WriteString("[trace_id: ")
			b.WriteString(raw.TraceID)
			b.WriteString("]")
		}
	}

	if b.Len() == 0 {
		return body
	}
	return b.String()
}

// IsNotFound reports whether err is an APIError with HTTP 404 status.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

const (
	// ClickUp V2 API endpoints are hosted under /api/v2 (not at the root domain).
	defaultBaseURL     = "https://api.clickup.com/api"
	defaultHTTPTimeout = 30 * time.Second
)

// Client interacts with the ClickUp API.
type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

// New creates a ClickUp API client.
func New(apiToken, baseURL string, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiToken:   apiToken,
		httpClient: httpClient,
	}
}

// Get performs a GET request against the ClickUp API and returns the response body.
func (c *Client) Get(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return c.do(ctx, http.MethodGet, path, query, nil)
}

// Post performs a POST request against the ClickUp API and returns the response body.
func (c *Client) Post(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.do(ctx, http.MethodPost, path, nil, body)
}

// Patch performs a PATCH request against the ClickUp API and returns the response body.
func (c *Client) Patch(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.do(ctx, http.MethodPatch, path, nil, body)
}

// Put performs a PUT request against the ClickUp API and returns the response body.
func (c *Client) Put(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.do(ctx, http.MethodPut, path, nil, body)
}

// Delete performs a DELETE request against the ClickUp API and returns the response body.
func (c *Client) Delete(ctx context.Context, path string, query ...url.Values) ([]byte, error) {
	var q url.Values
	if len(query) > 0 {
		q = query[0]
	}
	return c.do(ctx, http.MethodDelete, path, q, nil)
}

// do executes an HTTP request with authentication and standard ClickUp headers.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body []byte) ([]byte, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, err
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}
	if c.apiToken != "" {
		req.Header.Set("Authorization", c.apiToken)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	return respBody, nil
}
