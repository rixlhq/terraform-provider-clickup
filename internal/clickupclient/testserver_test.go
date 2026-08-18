package clickupclient_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
)

func TestTestServerStatic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	ts.RegisterStatic("GET", "/v2/space/123", 200, map[string]any{"id": "123", "name": "test"})

	client := clickupclient.New("token", ts.URL, nil)
	body, err := client.Get(context.Background(), "/v2/space/123", nil)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != `{"id":"123","name":"test"}` {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestTestServerRegister(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	ts.Register("POST", "/v2/team/1/space", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if string(body) != `{"name":"space"}` {
			t.Errorf("unexpected request body: %s", string(body))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"abc"}`))
	})

	client := clickupclient.New("token", ts.URL, nil)
	resp, err := client.Post(context.Background(), "/v2/team/1/space", []byte(`{"name":"space"}`))
	if err != nil {
		t.Fatal(err)
	}

	if string(resp) != `{"id":"abc"}` {
		t.Fatalf("unexpected body: %s", string(resp))
	}
}

func TestTestServerNotFound(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	client := clickupclient.New("token", ts.URL, nil)
	_, err := client.Get(context.Background(), "/v2/unknown", url.Values{})
	if err == nil {
		t.Fatal("expected error for unregistered path")
	}
	if !clickupclient.IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}
