// Package clickupv3 wraps the ogen-generated ClickUp V3 API client with the
// provider's authentication and base-URL configuration. The generated code
// lives in the same package (oas_*.go) and is not modified.
package clickupv3

import (
	"context"
	"fmt"
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

// New wraps the ogen-generated NewClient with the provider's API token
// and an optional base-URL override. The base URL defaults to
// https://api.clickup.com/ (V3 routes are under /api/v3/...).
func New(apiToken, baseURL string) (*Client, error) {
	if apiToken == "" {
		return nil, fmt.Errorf("ClickUp V3 API token must not be empty")
	}
	if baseURL == "" {
		baseURL = defaultV3BaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return NewClient(baseURL, &apiKeySecuritySource{apiToken: apiToken})
}
