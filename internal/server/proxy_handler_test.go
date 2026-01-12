package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestProxyHandlerForwardsMatchingRequests(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "reached")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))
	defer backend.Close()

	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fallback response"))
	})

	handler, err := newProxyHandler(fallback, backend.URL, "/api/")
	if err != nil {
		t.Fatalf("failed to create proxy handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Backend") != "reached" {
		t.Error("request did not reach backend")
	}
	body := rec.Body.String()
	if body != "backend response" {
		t.Errorf("expected 'backend response', got %q", body)
	}
}

func TestProxyHandlerFallsBackForNonMatchingPaths(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("backend"))
	}))
	defer backend.Close()

	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fallback"))
	})

	handler, err := newProxyHandler(fallback, backend.URL, "/api/")
	if err != nil {
		t.Fatalf("failed to create proxy handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/static/file.js", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if body != "fallback" {
		t.Errorf("expected 'fallback', got %q", body)
	}
}

func TestProxyHandlerPreservesQueryParams(t *testing.T) {
	var receivedQuery string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	handler, err := newProxyHandler(http.NotFoundHandler(), backend.URL, "/api/")
	if err != nil {
		t.Fatalf("failed to create proxy handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test?foo=bar&baz=qux", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if receivedQuery != "foo=bar&baz=qux" {
		t.Errorf("expected query 'foo=bar&baz=qux', got %q", receivedQuery)
	}
}

func TestProxyHandlerForwardsRequestBody(t *testing.T) {
	var receivedBody string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	handler, err := newProxyHandler(http.NotFoundHandler(), backend.URL, "/api/")
	if err != nil {
		t.Fatalf("failed to create proxy handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if receivedBody != `{"key":"value"}` {
		t.Errorf("expected body '{\"key\":\"value\"}', got %q", receivedBody)
	}
}

func TestProxyHandlerReturns502OnBackendError(t *testing.T) {
	handler, err := newProxyHandler(http.NotFoundHandler(), "http://localhost:1", "/api/")
	if err != nil {
		t.Fatalf("failed to create proxy handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}
}

func TestProxyHandlerRejectsInvalidURLs(t *testing.T) {
	tests := []struct {
		name        string
		backendURL  string
		expectError bool
	}{
		{"valid http", "http://localhost:8080", false},
		{"valid https", "https://backend.example.com:443", false},
		{"valid with path", "http://backend:8001/api", false},
		{"double http scheme", "http://http://localhost:8080", true},
		{"missing scheme", "localhost:8080", true},
		{"ftp scheme", "ftp://localhost:8080", true},
		{"empty host", "http://", true},
		{"just scheme", "http:", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newProxyHandler(http.NotFoundHandler(), tc.backendURL, "/api/")
			if tc.expectError && err == nil {
				t.Errorf("expected error for URL %q, got none", tc.backendURL)
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error for URL %q: %v", tc.backendURL, err)
			}
		})
	}
}

func TestProxyHandlerDetectsWebSocketUpgrade(t *testing.T) {
	handler := &proxyHandler{}

	tests := []struct {
		name          string
		connectionHdr string
		upgradeHdr    string
		expectUpgrade bool
	}{
		{"websocket upgrade", "Upgrade", "websocket", true},
		{"mixed case", "upgrade", "WebSocket", true},
		{"keep-alive, upgrade", "keep-alive, Upgrade", "websocket", true},
		{"no upgrade header", "keep-alive", "", false},
		{"no connection header", "", "websocket", false},
		{"wrong upgrade value", "Upgrade", "h2c", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/ws", nil)
			if tc.connectionHdr != "" {
				req.Header.Set("Connection", tc.connectionHdr)
			}
			if tc.upgradeHdr != "" {
				req.Header.Set("Upgrade", tc.upgradeHdr)
			}

			result := handler.isWebSocketUpgrade(req)
			if result != tc.expectUpgrade {
				t.Errorf("expected %v, got %v", tc.expectUpgrade, result)
			}
		})
	}
}

func TestProxyHandlerWebSocketIntegration(t *testing.T) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("backend upgrade error: %v", err)
			return
		}
		defer conn.Close()

		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		conn.WriteMessage(websocket.TextMessage, append([]byte("echo: "), msg...))
	}))
	defer backend.Close()

	handler, err := newProxyHandler(http.NotFoundHandler(), backend.URL, "/api/")
	if err != nil {
		t.Fatalf("failed to create proxy handler: %v", err)
	}

	proxy := httptest.NewServer(handler)
	defer proxy.Close()

	wsURL := "ws" + strings.TrimPrefix(proxy.URL, "http") + "/api/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect websocket: %v", err)
	}
	defer conn.Close()

	err = conn.WriteMessage(websocket.TextMessage, []byte("hello"))
	if err != nil {
		t.Fatalf("failed to write message: %v", err)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	expected := "echo: hello"
	if string(msg) != expected {
		t.Errorf("expected %q, got %q", expected, string(msg))
	}
}
