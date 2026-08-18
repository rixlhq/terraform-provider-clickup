package clickupclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
)

// TestServer is a mock ClickUp API server for unit and acceptance tests.
type TestServer struct {
	*httptest.Server

	mu       sync.Mutex
	handlers map[string]map[string]http.HandlerFunc
}

// NewTestServer creates a mock ClickUp server with an empty handler map.
func NewTestServer() *TestServer {
	ts := &TestServer{handlers: make(map[string]map[string]http.HandlerFunc)}
	ts.Server = httptest.NewServer(ts)
	return ts
}

// Register sets a handler for the given HTTP method and path.
func (ts *TestServer) Register(method, path string, fn http.HandlerFunc) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.handlers[method] == nil {
		ts.handlers[method] = make(map[string]http.HandlerFunc)
	}
	ts.handlers[method][path] = fn
}

// RegisterStatic responds with the given status and JSON body for the
// specified method and path.
func (ts *TestServer) RegisterStatic(method, path string, status int, body any) {
	ts.Register(method, path, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		if err := json.NewEncoder(w).Encode(body); err != nil {
			_ = fmt.Errorf("test server encode: %w", err)
		}
	})
}

func (ts *TestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ts.mu.Lock()
	methodHandlers := ts.handlers[r.Method]
	if methodHandlers == nil {
		ts.mu.Unlock()
		http.NotFound(w, r)
		return
	}
	h := methodHandlers[r.URL.Path]
	ts.mu.Unlock()

	if h == nil {
		http.NotFound(w, r)
		return
	}

	h(w, r)
}
