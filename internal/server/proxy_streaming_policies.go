package server

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	proxyStreamingPolicyMappingSeparator = "="
	proxyStreamingModeBuffered           = "buffered"
	proxyStreamingModeUnbuffered         = "unbuffered"
)

var ErrInvalidProxyStreamingPolicy = errors.New("proxy.streaming.policy.invalid")

type ProxyStreamingPolicies struct {
	policies []proxyStreamingPolicy
}

type proxyStreamingPolicy struct {
	pathPrefix   string
	unbufferedIO bool
}

func NewProxyStreamingPolicies(mappings []string) (ProxyStreamingPolicies, error) {
	if len(mappings) == 0 {
		return ProxyStreamingPolicies{}, nil
	}
	policyByPathPrefix := map[string]proxyStreamingPolicy{}
	for _, mapping := range mappings {
		parsedPolicy, parseErr := parseProxyStreamingPolicy(mapping)
		if parseErr != nil {
			return ProxyStreamingPolicies{}, parseErr
		}
		policyByPathPrefix[parsedPolicy.pathPrefix] = parsedPolicy
	}

	policies := make([]proxyStreamingPolicy, 0, len(policyByPathPrefix))
	for _, policy := range policyByPathPrefix {
		policies = append(policies, policy)
	}
	sort.SliceStable(policies, func(leftIndex int, rightIndex int) bool {
		return len(policies[leftIndex].pathPrefix) > len(policies[rightIndex].pathPrefix)
	})
	return ProxyStreamingPolicies{policies: policies}, nil
}

func (policies ProxyStreamingPolicies) IsUnbuffered(requestPath string) bool {
	for _, policy := range policies.policies {
		if strings.HasPrefix(requestPath, policy.pathPrefix) {
			return policy.unbufferedIO
		}
	}
	return false
}

func parseProxyStreamingPolicy(mapping string) (proxyStreamingPolicy, error) {
	trimmedMapping := strings.TrimSpace(mapping)
	if trimmedMapping == "" {
		return proxyStreamingPolicy{}, fmt.Errorf("%w: empty mapping", ErrInvalidProxyStreamingPolicy)
	}
	parts := strings.SplitN(trimmedMapping, proxyStreamingPolicyMappingSeparator, 2)
	if len(parts) != 2 {
		return proxyStreamingPolicy{}, fmt.Errorf("%w: mapping must be in /path=unbuffered|buffered form", ErrInvalidProxyStreamingPolicy)
	}

	pathPrefix := strings.TrimSpace(parts[0])
	if pathPrefix == "" {
		return proxyStreamingPolicy{}, fmt.Errorf("%w: empty path prefix", ErrInvalidProxyStreamingPolicy)
	}
	if !strings.HasPrefix(pathPrefix, proxyPathPrefixStart) {
		return proxyStreamingPolicy{}, fmt.Errorf("%w: path prefix must start with /", ErrInvalidProxyStreamingPolicy)
	}

	mode := strings.ToLower(strings.TrimSpace(parts[1]))
	switch mode {
	case proxyStreamingModeBuffered:
		return proxyStreamingPolicy{pathPrefix: pathPrefix, unbufferedIO: false}, nil
	case proxyStreamingModeUnbuffered:
		return proxyStreamingPolicy{pathPrefix: pathPrefix, unbufferedIO: true}, nil
	default:
		return proxyStreamingPolicy{}, fmt.Errorf("%w: unsupported mode %s", ErrInvalidProxyStreamingPolicy, mode)
	}
}
