package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouteResponsePolicyHandlerOverridesInnerHeaders(t *testing.T) {
	policies, policiesErr := NewRouteResponsePolicies([]string{"/=Cache-Control:no-store"})
	if policiesErr != nil {
		t.Fatalf("create response policies: %v", policiesErr)
	}

	innerHandler := http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Cache-Control", "public, max-age=60")
		_, _ = responseWriter.Write([]byte("ok"))
	})
	handler := newRouteResponsePolicyHandler(innerHandler, policies)

	request := httptest.NewRequest(http.MethodGet, "http://example.com/index.html", nil)
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, request)

	if responseRecorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected no-store cache control, got %q", responseRecorder.Header().Get("Cache-Control"))
	}
}

func TestRouteResponsePolicyHandlerSkipsUnmatchedPaths(t *testing.T) {
	policies, policiesErr := NewRouteResponsePolicies([]string{"/app/=Cache-Control:no-store"})
	if policiesErr != nil {
		t.Fatalf("create response policies: %v", policiesErr)
	}

	innerHandler := http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		_, _ = responseWriter.Write([]byte("ok"))
	})
	handler := newRouteResponsePolicyHandler(innerHandler, policies)

	request := httptest.NewRequest(http.MethodGet, "http://example.com/assets/app.js", nil)
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, request)

	if responseRecorder.Header().Get("Cache-Control") != "" {
		t.Fatalf("expected no cache-control header, got %q", responseRecorder.Header().Get("Cache-Control"))
	}
}
