package clickupclient_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
)

func TestClientGet(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "pk_token" {
			t.Fatalf("expected Authorization header pk_token, got %q", auth)
		}
		if !strings.HasPrefix(r.URL.Path, "/api/v2/") {
			t.Fatalf("expected path prefix /api/v2/, got %q", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "value" {
			t.Fatalf("expected query key=value, got %q", r.URL.RawQuery)
		}
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer server.Close()

	client := clickupclient.New("pk_token", server.URL+"/api", nil)
	query := url.Values{"key": []string{"value"}}
	body, err := client.Get(ctx, "/v2/space/123", query)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %q", string(body))
	}
}

func TestClientPost(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"name":"test"}` {
			t.Fatalf("unexpected request body: %q", string(body))
		}
		_, _ = fmt.Fprint(w, `{"id":"abc"}`)
	}))
	defer server.Close()

	client := clickupclient.New("pk_token", server.URL+"/api", nil)
	body, err := client.Post(ctx, "/v2/space/123/list", []byte(`{"name":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"id":"abc"}` {
		t.Fatalf("unexpected body: %q", string(body))
	}
}

func TestClientDeleteNotFound(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, "not found")
	}))
	defer server.Close()

	client := clickupclient.New("pk_token", server.URL+"/api", nil)
	_, err := client.Delete(ctx, "/v2/task/123")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !clickupclient.IsNotFound(err) {
		t.Fatalf("expected IsNotFound, got %v", err)
	}
}

func TestClientAPIError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, "bad request")
	}))
	defer server.Close()

	client := clickupclient.New("pk_token", server.URL+"/api", nil)
	_, err := client.Get(ctx, "/v2/space/123", nil)
	var apiErr *clickupclient.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", apiErr.StatusCode)
	}
	if apiErr.Body != "bad request" {
		t.Fatalf("expected body \"bad request\", got %q", apiErr.Body)
	}
}
