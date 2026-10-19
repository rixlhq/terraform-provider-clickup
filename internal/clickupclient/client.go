package clickupclient

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// ClickUp V2 API endpoints are hosted under /api/v2 (not at the root domain).
	defaultBaseURL     = "https://api.clickup.com/api"
	defaultHTTPTimeout = 30 * time.Second
)

// Client interacts with the ClickUp API.
type Client struct {
	baseURL      string
	apiToken     string
	userAgent    string
	httpClient   *http.Client
	maxRetries   int
	minRetryWait time.Duration
	maxRetryWait time.Duration
}

// New creates a ClickUp API client with sensible default timeouts, user agent, and retry policies.
func New(apiToken, baseURL string, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		apiToken:     apiToken,
		userAgent:    "terraform-provider-clickup/1.0.0",
		httpClient:   httpClient,
		maxRetries:   3,
		minRetryWait: 200 * time.Millisecond,
		maxRetryWait: 10 * time.Second,
	}
}

// SetUserAgent configures a custom User-Agent header value for API requests.
func (c *Client) SetUserAgent(ua string) {
	c.userAgent = ua
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

// do executes an HTTP request with authentication, standard ClickUp headers, and automatic retries.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body []byte) ([]byte, error) {
	reqURL, err := c.buildURL(path, query)
	if err != nil {
		return nil, err
	}

	maxRetries := max(0, c.maxRetries)
	for attempt := 0; ; attempt++ {
		statusCode, headers, respBody, err := c.executeHTTP(ctx, method, reqURL, body)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if attempt < maxRetries {
				if waitErr := c.waitRetry(ctx, attempt, nil); waitErr != nil {
					return nil, waitErr
				}
				continue
			}
			return nil, err
		}

		if statusCode >= 300 {
			if isRetryableStatusCode(statusCode) && attempt < maxRetries {
				if waitErr := c.waitRetry(ctx, attempt, headers); waitErr != nil {
					return nil, waitErr
				}
				continue
			}
			return nil, &APIError{StatusCode: statusCode, Body: string(respBody)}
		}

		return respBody, nil
	}
}

// buildURL joins the client baseURL with path and encodes any query parameters.
func (c *Client) buildURL(path string, query url.Values) (string, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return "", err
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return u.String(), nil
}

// executeHTTP issues the HTTP request, closes the response body, and returns the status, headers, and body.
func (c *Client) executeHTTP(ctx context.Context, method, reqURL string, body []byte) (int, http.Header, []byte, error) {
	req, err := c.buildRequest(ctx, method, reqURL, body)
	if err != nil {
		return 0, nil, nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, nil, err
	}
	return resp.StatusCode, resp.Header, respBody, nil
}

// buildRequest creates an authorized http.Request with JSON headers.
func (c *Client) buildRequest(ctx context.Context, method, reqURL string, body []byte) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
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
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	return req, nil
}
