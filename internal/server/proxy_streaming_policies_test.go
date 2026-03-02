package server

import (
	"errors"
	"testing"
)

func TestNewProxyStreamingPoliciesRejectsInvalidMappings(t *testing.T) {
	testCases := []struct {
		name    string
		mapping string
	}{
		{name: "missing-separator", mapping: "/api"},
		{name: "missing-path-prefix", mapping: "=unbuffered"},
		{name: "missing-leading-slash", mapping: "api=unbuffered"},
		{name: "invalid-mode", mapping: "/api=invalid"},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(testingT *testing.T) {
			_, parseErr := NewProxyStreamingPolicies([]string{testCase.mapping})
			if !errors.Is(parseErr, ErrInvalidProxyStreamingPolicy) {
				testingT.Fatalf("expected ErrInvalidProxyStreamingPolicy, got %v", parseErr)
			}
		})
	}
}

func TestProxyStreamingPoliciesUsesMostSpecificMatchingPath(t *testing.T) {
	policies, policiesErr := NewProxyStreamingPolicies([]string{
		"/api=buffered",
		"/api/events=unbuffered",
	})
	if policiesErr != nil {
		t.Fatalf("create proxy streaming policies: %v", policiesErr)
	}

	if !policies.IsUnbuffered("/api/events/stream") {
		t.Fatalf("expected /api/events/stream to be unbuffered")
	}
	if policies.IsUnbuffered("/api/users") {
		t.Fatalf("expected /api/users to be buffered")
	}
	if policies.IsUnbuffered("/other/path") {
		t.Fatalf("expected unmatched path to use buffered behavior")
	}
}
