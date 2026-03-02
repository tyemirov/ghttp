package server

import (
	"net/url"
	"testing"
)

func TestProxyRouteHandlerResolveHTTPProxyUsesUnbufferedPolicy(t *testing.T) {
	backendURL, parseErr := url.Parse("http://127.0.0.1:8080")
	if parseErr != nil {
		t.Fatalf("parse backend url: %v", parseErr)
	}
	routeHandler := newProxyRouteHandler(proxyRoute{
		pathPrefix: "/api",
		backendURL: backendURL,
	})
	streamingPolicies, policiesErr := NewProxyStreamingPolicies([]string{"/api/events=unbuffered"})
	if policiesErr != nil {
		t.Fatalf("create proxy streaming policies: %v", policiesErr)
	}

	selectedProxy := routeHandler.resolveHTTPProxy("/api/events/stream", streamingPolicies)
	if selectedProxy.FlushInterval != -1 {
		t.Fatalf("expected unbuffered proxy flush interval -1, got %v", selectedProxy.FlushInterval)
	}

	defaultProxy := routeHandler.resolveHTTPProxy("/api/users", streamingPolicies)
	if defaultProxy.FlushInterval != 0 {
		t.Fatalf("expected default proxy flush interval 0, got %v", defaultProxy.FlushInterval)
	}
}
