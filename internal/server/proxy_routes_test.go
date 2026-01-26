package server

import (
	"errors"
	"testing"
)

func TestNewProxyRoutes(testingInstance *testing.T) {
	testCases := []struct {
		name          string
		mappings      []string
		expectedCount int
		expectError   bool
		expectedErr   error
	}{
		{
			name:          "empty list",
			mappings:      nil,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "valid mappings",
			mappings:      []string{"/api=http://backend:8081", "/ws=https://backend:8443"},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "missing separator",
			mappings:      []string{"/apihttp://backend:8081"},
			expectedCount: 0,
			expectError:   true,
			expectedErr:   ErrInvalidProxyRoute,
		},
		{
			name:          "empty path prefix",
			mappings:      []string{"=http://backend:8081"},
			expectedCount: 0,
			expectError:   true,
			expectedErr:   ErrInvalidProxyRoute,
		},
		{
			name:          "path prefix without slash",
			mappings:      []string{"api=http://backend:8081"},
			expectedCount: 0,
			expectError:   true,
			expectedErr:   ErrInvalidProxyRoute,
		},
		{
			name:          "empty backend",
			mappings:      []string{"/api="},
			expectedCount: 0,
			expectError:   true,
			expectedErr:   ErrInvalidProxyRoute,
		},
		{
			name:          "invalid scheme",
			mappings:      []string{"/api=ftp://backend"},
			expectedCount: 0,
			expectError:   true,
			expectedErr:   ErrInvalidProxyRoute,
		},
		{
			name:          "missing host",
			mappings:      []string{"/api=http://"},
			expectedCount: 0,
			expectError:   true,
			expectedErr:   ErrInvalidProxyRoute,
		},
		{
			name:          "host with trailing colon",
			mappings:      []string{"/api=http://backend:"},
			expectedCount: 0,
			expectError:   true,
			expectedErr:   ErrInvalidProxyRoute,
		},
		{
			name:          "parse error",
			mappings:      []string{"/api=http://[::1"},
			expectedCount: 0,
			expectError:   true,
			expectedErr:   ErrInvalidProxyRoute,
		},
		{
			name:          "duplicate prefix",
			mappings:      []string{"/api=http://backend:8081", "/api=http://backend:8082"},
			expectedCount: 0,
			expectError:   true,
			expectedErr:   ErrInvalidProxyRoutes,
		},
	}

	for _, testCase := range testCases {
		testingInstance.Run(testCase.name, func(testingInstance *testing.T) {
			proxyRoutes, err := NewProxyRoutes(testCase.mappings)
			if testCase.expectError {
				if err == nil {
					testingInstance.Fatalf("expected error")
				}
				if testCase.expectedErr != nil && !errors.Is(err, testCase.expectedErr) {
					testingInstance.Fatalf("expected error %s, got %v", testCase.expectedErr.Error(), err)
				}
				return
			}
			if err != nil {
				testingInstance.Fatalf("unexpected error: %v", err)
			}
			if proxyRoutes.Count() != testCase.expectedCount {
				testingInstance.Fatalf("expected %d routes, got %d", testCase.expectedCount, proxyRoutes.Count())
			}
		})
	}
}

func TestNewProxyRoutesFromLegacy(testingInstance *testing.T) {
	testCases := []struct {
		name          string
		pathPrefix    string
		backendURL    string
		expectedCount int
		expectError   bool
	}{
		{
			name:          "empty legacy config",
			pathPrefix:    "",
			backendURL:    "",
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "valid legacy config",
			pathPrefix:    "/api",
			backendURL:    "http://backend:8081",
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:        "missing backend",
			pathPrefix:  "/api",
			backendURL:  "",
			expectError: true,
		},
		{
			name:        "missing path prefix",
			pathPrefix:  "",
			backendURL:  "http://backend:8081",
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		testingInstance.Run(testCase.name, func(testingInstance *testing.T) {
			proxyRoutes, err := NewProxyRoutesFromLegacy(testCase.pathPrefix, testCase.backendURL)
			if testCase.expectError {
				if err == nil {
					testingInstance.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				testingInstance.Fatalf("unexpected error: %v", err)
			}
			if proxyRoutes.Count() != testCase.expectedCount {
				testingInstance.Fatalf("expected %d routes, got %d", testCase.expectedCount, proxyRoutes.Count())
			}
		})
	}
}
