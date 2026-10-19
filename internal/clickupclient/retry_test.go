package clickupclient_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
)

func TestClientUserAgent(t *testing.T) {
	ctx := context.Background()
	var receivedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer server.Close()

	client := clickupclient.New("pk_token", server.URL, nil)
	client.SetUserAgent("terraform-provider-clickup/1.0.0-test")
	_, err := client.Get(ctx, "/v2/user", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedUA != "terraform-provider-clickup/1.0.0-test" {
		t.Fatalf("expected User-Agent %q, got %q", "terraform-provider-clickup/1.0.0-test", receivedUA)
	}
}

func TestClientRetryOn429(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = fmt.Fprint(w, `{"err":"Rate limit reached"}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer server.Close()

	client := clickupclient.New("pk_token", server.URL, nil)
	client.SetRetryPolicy(3, 1*time.Millisecond, 5*time.Millisecond)
	body, err := client.Get(ctx, "/v2/space/1", nil)
	if err != nil {
		t.Fatalf("expected successful retry, got error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %q", string(body))
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestClientRetryOn503(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprint(w, `{"err":"Temporary outage"}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"success":true}`)
	}))
	defer server.Close()

	client := clickupclient.New("pk_token", server.URL, nil)
	client.SetRetryPolicy(3, 1*time.Millisecond, 5*time.Millisecond)
	body, err := client.Get(ctx, "/v2/task/1", nil)
	if err != nil {
		t.Fatalf("expected successful retry, got: %v", err)
	}
	if string(body) != `{"success":true}` {
		t.Fatalf("unexpected body: %q", string(body))
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestClientRetryExhausted(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = fmt.Fprint(w, `{"err":"Rate limit permanently reached"}`)
	}))
	defer server.Close()

	client := clickupclient.New("pk_token", server.URL, nil)
	client.SetRetryPolicy(2, 1*time.Millisecond, 2*time.Millisecond)
	_, err := client.Get(ctx, "/v2/space/1", nil)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if attempts != 3 { // initial attempt (0) + 2 retries = 3
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}
