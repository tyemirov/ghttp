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

func TestRouteResponsePolicyHandlerAppliesHeadersOnFlush(t *testing.T) {
	policies, policiesErr := NewRouteResponsePolicies([]string{"/=Cache-Control:no-store"})
	if policiesErr != nil {
		t.Fatalf("create response policies: %v", policiesErr)
	}

	innerHandler := http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Cache-Control", "public, max-age=60")
		responseFlusher, supportsFlush := responseWriter.(http.Flusher)
		if !supportsFlush {
			t.Fatalf("expected wrapped writer to support flush")
		}
		responseFlusher.Flush()
	})
	handler := newRouteResponsePolicyHandler(innerHandler, policies)

	request := httptest.NewRequest(http.MethodGet, "http://example.com/stream", nil)
	trackingWriter := &routePolicyFlushTrackingResponseWriter{header: http.Header{}}

	handler.ServeHTTP(trackingWriter, request)

	if trackingWriter.flushCount != 1 {
		t.Fatalf("expected flush pass-through once, got %d", trackingWriter.flushCount)
	}
	if trackingWriter.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected no-store cache-control on flush, got %q", trackingWriter.Header().Get("Cache-Control"))
	}
}

type routePolicyFlushTrackingResponseWriter struct {
	header     http.Header
	flushCount int
}

func (writer *routePolicyFlushTrackingResponseWriter) Header() http.Header {
	return writer.header
}

func (writer *routePolicyFlushTrackingResponseWriter) Write(content []byte) (int, error) {
	return len(content), nil
}

func (writer *routePolicyFlushTrackingResponseWriter) WriteHeader(_ int) {
}

func (writer *routePolicyFlushTrackingResponseWriter) Flush() {
	writer.flushCount++
}
