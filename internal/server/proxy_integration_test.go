package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/tyemirov/ghttp/internal/serverdetails"
	"github.com/tyemirov/ghttp/pkg/logging"
)

// Integration tests for proxy functionality.
// All tests go through the full FileServer handler chain including logging middleware,
// matching the exact code path used in production.

func TestIntegrationProxyForwardsMatchingRequests(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "reached")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))
	defer backend.Close()

	handler := newTestProxyHandler(t, backend.URL, "/api/")

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Backend") != "reached" {
		t.Error("request did not reach backend")
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "backend response" {
		t.Errorf("expected 'backend response', got %q", string(body))
	}
}

func TestIntegrationProxyFallsBackForNonMatchingPaths(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("backend"))
	}))
	defer backend.Close()

	tempDir := t.TempDir()
	writeTestFile(t, tempDir, "static/file.js", "fallback content")

	handler := newTestProxyHandlerWithDir(t, tempDir, backend.URL, "/api/")

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/static/file.js")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "fallback content") {
		t.Errorf("expected fallback content, got %q", string(body))
	}
}

func TestIntegrationProxyPreservesQueryParams(t *testing.T) {
	var receivedQuery string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	handler := newTestProxyHandler(t, backend.URL, "/api/")

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/test?foo=bar&baz=qux")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if receivedQuery != "foo=bar&baz=qux" {
		t.Errorf("expected query 'foo=bar&baz=qux', got %q", receivedQuery)
	}
}

func TestIntegrationProxyForwardsRequestBody(t *testing.T) {
	var receivedBody string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	handler := newTestProxyHandler(t, backend.URL, "/api/")

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/test", "application/json", strings.NewReader(`{"key":"value"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if receivedBody != `{"key":"value"}` {
		t.Errorf("expected body '{\"key\":\"value\"}', got %q", receivedBody)
	}
}

func TestIntegrationProxyReturns502OnBackendError(t *testing.T) {
	handler := newTestProxyHandler(t, "http://localhost:1", "/api/")

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", resp.StatusCode)
	}
}

func TestIntegrationProxyPrefersMostSpecificPrefix(t *testing.T) {
	backendPrimary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "primary")
		w.WriteHeader(http.StatusOK)
	}))
	defer backendPrimary.Close()

	backendSecondary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "secondary")
		w.WriteHeader(http.StatusOK)
	}))
	defer backendSecondary.Close()

	proxyRoutes, routeErr := NewProxyRoutes([]string{
		"/api=" + backendPrimary.URL,
		"/api/internal=" + backendSecondary.URL,
	})
	if routeErr != nil {
		t.Fatalf("proxy routes: %v", routeErr)
	}

	tempDir := t.TempDir()
	fileServer := NewFileServer(logging.NewTestService(logging.TypeJSON), serverdetails.NewServingAddressFormatter())
	config := FileServerConfiguration{
		DirectoryPath: tempDir,
		ProxyRoutes:   proxyRoutes,
		LoggingType:   logging.TypeJSON,
	}
	handler := fileServer.FullHandler(config)

	proxyServer := httptest.NewServer(handler)
	defer proxyServer.Close()

	primaryResponse, primaryErr := http.Get(proxyServer.URL + "/api/status")
	if primaryErr != nil {
		t.Fatalf("primary request failed: %v", primaryErr)
	}
	primaryResponse.Body.Close()
	if primaryResponse.Header.Get("X-Backend") != "primary" {
		t.Errorf("expected primary backend, got %s", primaryResponse.Header.Get("X-Backend"))
	}

	secondaryResponse, secondaryErr := http.Get(proxyServer.URL + "/api/internal/status")
	if secondaryErr != nil {
		t.Fatalf("secondary request failed: %v", secondaryErr)
	}
	secondaryResponse.Body.Close()
	if secondaryResponse.Header.Get("X-Backend") != "secondary" {
		t.Errorf("expected secondary backend, got %s", secondaryResponse.Header.Get("X-Backend"))
	}
}

func TestIntegrationProxyWebSocketThroughFullStack(t *testing.T) {
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

	// Use FullHandler to get the complete handler chain including logging middleware
	handler := newTestProxyHandler(t, backend.URL, "/api/")

	proxy := httptest.NewServer(handler)
	defer proxy.Close()

	wsURL := "ws" + strings.TrimPrefix(proxy.URL, "http") + "/api/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect websocket through full handler stack: %v", err)
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

func TestIntegrationProxyWebSocketWithLoggingMiddleware(t *testing.T) {
	// This test specifically verifies that WebSocket connections work through
	// the logging middleware's statusRecorder wrapper (which must implement http.Hijacker)
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	messageReceived := make(chan string, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		messageReceived <- string(msg)
		conn.WriteMessage(websocket.TextMessage, []byte("ack"))
	}))
	defer backend.Close()

	// Create handler with JSON logging (uses statusRecorder wrapper)
	tempDir := t.TempDir()
	fileServer := NewFileServer(logging.NewTestService(logging.TypeJSON), serverdetails.NewServingAddressFormatter())
	proxyRoutes, routeErr := NewProxyRoutesFromLegacy("/api/", backend.URL)
	if routeErr != nil {
		t.Fatalf("proxy routes: %v", routeErr)
	}
	config := FileServerConfiguration{
		DirectoryPath: tempDir,
		ProxyRoutes:   proxyRoutes,
		LoggingType:   logging.TypeJSON,
	}
	handler := fileServer.FullHandler(config)

	proxy := httptest.NewServer(handler)
	defer proxy.Close()

	wsURL := "ws" + strings.TrimPrefix(proxy.URL, "http") + "/api/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed through logging middleware: %v", err)
	}
	defer conn.Close()

	testMessage := "test through logging middleware"
	err = conn.WriteMessage(websocket.TextMessage, []byte(testMessage))
	if err != nil {
		t.Fatalf("failed to write message: %v", err)
	}

	// Read the ack response to ensure round-trip completes
	_, _, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read ack: %v", err)
	}

	select {
	case received := <-messageReceived:
		if received != testMessage {
			t.Errorf("expected %q, got %q", testMessage, received)
		}
	default:
		t.Error("message was not received by backend")
	}
}

func TestIntegrationProxyWebSocketBackendUnavailable(t *testing.T) {
	proxyRoutes, routeErr := NewProxyRoutes([]string{"/api=http://127.0.0.1:1"})
	if routeErr != nil {
		t.Fatalf("proxy routes: %v", routeErr)
	}

	tempDir := t.TempDir()
	fileServer := NewFileServer(logging.NewTestService(logging.TypeJSON), serverdetails.NewServingAddressFormatter())
	config := FileServerConfiguration{
		DirectoryPath: tempDir,
		ProxyRoutes:   proxyRoutes,
		LoggingType:   logging.TypeJSON,
	}
	handler := fileServer.FullHandler(config)

	proxy := httptest.NewServer(handler)
	defer proxy.Close()

	request, requestErr := http.NewRequest(http.MethodGet, proxy.URL+"/api/ws", nil)
	if requestErr != nil {
		t.Fatalf("request: %v", requestErr)
	}
	request.Header.Set(headerConnection, "Upgrade")
	request.Header.Set(headerUpgrade, valueWebSocket)

	response, responseErr := http.DefaultClient.Do(request)
	if responseErr != nil {
		t.Fatalf("request failed: %v", responseErr)
	}
	response.Body.Close()

	if response.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, response.StatusCode)
	}
}

// Helper functions

func newTestProxyHandler(t *testing.T, backendURL, pathPrefix string) http.Handler {
	t.Helper()
	tempDir := t.TempDir()
	return newTestProxyHandlerWithDir(t, tempDir, backendURL, pathPrefix)
}

func newTestProxyHandlerWithDir(t *testing.T, dir, backendURL, pathPrefix string) http.Handler {
	t.Helper()
	proxyRoutes, routeErr := NewProxyRoutesFromLegacy(pathPrefix, backendURL)
	if routeErr != nil {
		t.Fatalf("proxy routes: %v", routeErr)
	}
	fileServer := NewFileServer(logging.NewTestService(logging.TypeJSON), serverdetails.NewServingAddressFormatter())
	config := FileServerConfiguration{
		DirectoryPath: dir,
		ProxyRoutes:   proxyRoutes,
		LoggingType:   logging.TypeJSON,
	}
	return fileServer.FullHandler(config)
}

func writeTestFile(t *testing.T, baseDir, relativePath, content string) {
	t.Helper()
	fullPath := baseDir + "/" + relativePath
	// Create parent directories
	parts := strings.Split(relativePath, "/")
	if len(parts) > 1 {
		parentDir := baseDir + "/" + strings.Join(parts[:len(parts)-1], "/")
		if err := os.MkdirAll(parentDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", parentDir, err)
		}
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write file %s: %v", fullPath, err)
	}
}
