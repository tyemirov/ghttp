package server

import (
	"errors"
	"testing"
)

func TestNewRouteResponsePoliciesRejectsInvalidMappings(t *testing.T) {
	testCases := []struct {
		name    string
		mapping string
	}{
		{name: "missing-separator", mapping: "/api"},
		{name: "missing-path-prefix", mapping: "=Cache-Control:no-store"},
		{name: "missing-leading-slash", mapping: "api=Cache-Control:no-store"},
		{name: "missing-header-value-separator", mapping: "/api=Cache-Control"},
		{name: "empty-header-name", mapping: "/api=:no-store"},
		{name: "empty-header-value", mapping: "/api=Cache-Control:"},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(testingT *testing.T) {
			_, parseErr := NewRouteResponsePolicies([]string{testCase.mapping})
			if !errors.Is(parseErr, ErrInvalidRouteResponsePolicy) {
				testingT.Fatalf("expected ErrInvalidRouteResponsePolicy, got %v", parseErr)
			}
		})
	}
}

func TestRouteResponsePoliciesHeadersForPathUsesMostSpecificMatch(t *testing.T) {
	policies, policiesErr := NewRouteResponsePolicies([]string{
		"/=Cache-Control:no-store",
		"/assets/=Cache-Control:public, max-age=31536000, immutable",
		"/assets/=X-Asset-Policy:yes",
	})
	if policiesErr != nil {
		t.Fatalf("create response policies: %v", policiesErr)
	}

	assetHeaders := policies.HeadersForPath("/assets/app.js")
	if assetHeaders["Cache-Control"] != "public, max-age=31536000, immutable" {
		t.Fatalf("expected asset cache control override, got %q", assetHeaders["Cache-Control"])
	}
	if assetHeaders["X-Asset-Policy"] != "yes" {
		t.Fatalf("expected custom asset header, got %q", assetHeaders["X-Asset-Policy"])
	}

	rootHeaders := policies.HeadersForPath("/index.html")
	if rootHeaders["Cache-Control"] != "no-store" {
		t.Fatalf("expected root cache control no-store, got %q", rootHeaders["Cache-Control"])
	}
}
