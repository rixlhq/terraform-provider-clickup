// Package clickupv3 wraps the ogen-generated ClickUp V3 API client.

// The generated code lives in the same package (oas_*.go) and is not modified.

package clickupv3

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
)

const defaultV3BaseURL = "https://api.clickup.com/"

// apiKeySecuritySource implements ogen SecuritySource for the Authorization
// header scheme used by every V3 operation.
type apiKeySecuritySource struct {
	apiToken string
}

// AuthHeader provides the Authorization header value for V3 operations.
func (s *apiKeySecuritySource) AuthHeader(_ context.Context, _ OperationName) (AuthHeader, error) {
	return AuthHeader{APIKey: s.apiToken}, nil
}

// noContent204Transport wraps an http.RoundTripper to inject a "null" body
// into 204 No Content responses. ogen's generated decoder for 204 responses
// expects an application/json body, but HTTP 204 responses have no body by
// definition. This transport bridges that gap so the decoder succeeds.
type noContent204Transport struct {
	base http.RoundTripper
}

func (t *noContent204Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	// HTTP 204 responses have no body, but ogen's decoder expects one.
	// Inject a "null" body so the jx.Raw decoder succeeds.
	if resp.StatusCode == http.StatusNoContent {
		// Drain and close the original body.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(strings.NewReader("null"))
		resp.Header.Set("Content-Type", "application/json")
		resp.ContentLength = 4
	}
	return resp, nil
}

// New wraps the ogen-generated NewClient with the provider's API token
// and an optional base-URL override. The base URL defaults to
// https://api.clickup.com/ (V3 routes are under /api/v3/...).
func New(apiToken, baseURL string) (*Client, error) {
	if apiToken == "" {
		return nil, errors.New("ClickUp V3 API token must not be empty")
	}
	if baseURL == "" {
		baseURL = defaultV3BaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	httpClient := &http.Client{
		Transport: &noContent204Transport{base: http.DefaultTransport},
	}

	return NewClient(baseURL, &apiKeySecuritySource{apiToken: apiToken}, WithClient(httpClient))
}
