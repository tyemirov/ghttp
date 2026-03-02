package server

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	routeResponsePolicyMappingSeparator = "="
	routeResponsePolicyHeaderSeparator  = ":"
)

var ErrInvalidRouteResponsePolicy = errors.New("route.response.policy.invalid")

type RouteResponsePolicies struct {
	policies []routeResponsePolicy
}

type routeResponsePolicy struct {
	pathPrefix string
	headers    map[string]string
}

func NewRouteResponsePolicies(mappings []string) (RouteResponsePolicies, error) {
	if len(mappings) == 0 {
		return RouteResponsePolicies{}, nil
	}
	headersByPathPrefix := map[string]map[string]string{}
	for _, mapping := range mappings {
		pathPrefix, headerName, headerValue, parseErr := parseRouteResponsePolicyMapping(mapping)
		if parseErr != nil {
			return RouteResponsePolicies{}, parseErr
		}
		existingHeaders, exists := headersByPathPrefix[pathPrefix]
		if !exists {
			existingHeaders = map[string]string{}
			headersByPathPrefix[pathPrefix] = existingHeaders
		}
		existingHeaders[headerName] = headerValue
	}

	policies := make([]routeResponsePolicy, 0, len(headersByPathPrefix))
	for pathPrefix, headers := range headersByPathPrefix {
		policyHeaders := make(map[string]string, len(headers))
		for headerName, headerValue := range headers {
			policyHeaders[headerName] = headerValue
		}
		policies = append(policies, routeResponsePolicy{
			pathPrefix: pathPrefix,
			headers:    policyHeaders,
		})
	}
	sort.SliceStable(policies, func(leftIndex int, rightIndex int) bool {
		return len(policies[leftIndex].pathPrefix) < len(policies[rightIndex].pathPrefix)
	})

	return RouteResponsePolicies{policies: policies}, nil
}

func (policies RouteResponsePolicies) IsEmpty() bool {
	return len(policies.policies) == 0
}

func (policies RouteResponsePolicies) HeadersForPath(requestPath string) map[string]string {
	if len(policies.policies) == 0 {
		return nil
	}
	resolvedHeaders := map[string]string{}
	for _, policy := range policies.policies {
		if strings.HasPrefix(requestPath, policy.pathPrefix) {
			for headerName, headerValue := range policy.headers {
				resolvedHeaders[headerName] = headerValue
			}
		}
	}
	if len(resolvedHeaders) == 0 {
		return nil
	}
	return resolvedHeaders
}

func parseRouteResponsePolicyMapping(mapping string) (string, string, string, error) {
	trimmedMapping := strings.TrimSpace(mapping)
	if trimmedMapping == "" {
		return "", "", "", fmt.Errorf("%w: empty mapping", ErrInvalidRouteResponsePolicy)
	}
	parts := strings.SplitN(trimmedMapping, routeResponsePolicyMappingSeparator, 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("%w: mapping must be in /path=Header-Name:Header-Value form", ErrInvalidRouteResponsePolicy)
	}
	pathPrefix := strings.TrimSpace(parts[0])
	if pathPrefix == "" {
		return "", "", "", fmt.Errorf("%w: empty path prefix", ErrInvalidRouteResponsePolicy)
	}
	if !strings.HasPrefix(pathPrefix, proxyPathPrefixStart) {
		return "", "", "", fmt.Errorf("%w: path prefix must start with /", ErrInvalidRouteResponsePolicy)
	}
	headerParts := strings.SplitN(strings.TrimSpace(parts[1]), routeResponsePolicyHeaderSeparator, 2)
	if len(headerParts) != 2 {
		return "", "", "", fmt.Errorf("%w: header policy must be in Header-Name:Header-Value form", ErrInvalidRouteResponsePolicy)
	}
	headerName := strings.TrimSpace(headerParts[0])
	headerValue := strings.TrimSpace(headerParts[1])
	if headerName == "" {
		return "", "", "", fmt.Errorf("%w: empty header name", ErrInvalidRouteResponsePolicy)
	}
	if headerValue == "" {
		return "", "", "", fmt.Errorf("%w: empty header value", ErrInvalidRouteResponsePolicy)
	}
	return pathPrefix, headerName, headerValue, nil
}
