package server

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

const (
	proxyMappingSeparator = "="
	proxyPathPrefixStart  = "/"
	proxySchemeHTTP       = "http"
	proxySchemeHTTPS      = "https"
)

var (
	ErrInvalidProxyRoute  = errors.New("proxy.route.invalid")
	ErrInvalidProxyRoutes = errors.New("proxy.routes.invalid")
)

type ProxyRoutes struct {
	routes []proxyRoute
}

type proxyRoute struct {
	pathPrefix string
	backendURL *url.URL
}

func NewProxyRoutes(routeMappings []string) (ProxyRoutes, error) {
	if len(routeMappings) == 0 {
		return ProxyRoutes{}, nil
	}
	seenPrefixes := map[string]struct{}{}
	parsedRoutes := make([]proxyRoute, 0, len(routeMappings))
	for _, mapping := range routeMappings {
		route, parseErr := parseProxyMapping(mapping)
		if parseErr != nil {
			return ProxyRoutes{}, parseErr
		}
		if _, exists := seenPrefixes[route.pathPrefix]; exists {
			return ProxyRoutes{}, fmt.Errorf("%w: duplicate path prefix %s", ErrInvalidProxyRoutes, route.pathPrefix)
		}
		seenPrefixes[route.pathPrefix] = struct{}{}
		parsedRoutes = append(parsedRoutes, route)
	}
	sort.SliceStable(parsedRoutes, func(leftIndex int, rightIndex int) bool {
		return len(parsedRoutes[leftIndex].pathPrefix) > len(parsedRoutes[rightIndex].pathPrefix)
	})
	return ProxyRoutes{routes: parsedRoutes}, nil
}

func NewProxyRoutesFromLegacy(pathPrefix string, backendURL string) (ProxyRoutes, error) {
	trimmedPathPrefix := strings.TrimSpace(pathPrefix)
	trimmedBackendURL := strings.TrimSpace(backendURL)
	if trimmedPathPrefix == "" && trimmedBackendURL == "" {
		return ProxyRoutes{}, nil
	}
	route, routeErr := newProxyRoute(trimmedPathPrefix, trimmedBackendURL)
	if routeErr != nil {
		return ProxyRoutes{}, routeErr
	}
	return ProxyRoutes{routes: []proxyRoute{route}}, nil
}

func (routes ProxyRoutes) IsEmpty() bool {
	return len(routes.routes) == 0
}

func (routes ProxyRoutes) Count() int {
	return len(routes.routes)
}

func parseProxyMapping(mapping string) (proxyRoute, error) {
	trimmedMapping := strings.TrimSpace(mapping)
	if trimmedMapping == "" {
		return proxyRoute{}, fmt.Errorf("%w: empty mapping", ErrInvalidProxyRoute)
	}
	parts := strings.SplitN(trimmedMapping, proxyMappingSeparator, 2)
	if len(parts) != 2 {
		return proxyRoute{}, fmt.Errorf("%w: mapping must be in /from=http://backend form", ErrInvalidProxyRoute)
	}
	pathPrefix := strings.TrimSpace(parts[0])
	backendURL := strings.TrimSpace(parts[1])
	return newProxyRoute(pathPrefix, backendURL)
}

func newProxyRoute(pathPrefix string, backendURL string) (proxyRoute, error) {
	if pathPrefix == "" {
		return proxyRoute{}, fmt.Errorf("%w: empty path prefix", ErrInvalidProxyRoute)
	}
	if !strings.HasPrefix(pathPrefix, proxyPathPrefixStart) {
		return proxyRoute{}, fmt.Errorf("%w: path prefix must start with /", ErrInvalidProxyRoute)
	}
	parsedURL, parseErr := parseProxyBackendURL(backendURL)
	if parseErr != nil {
		return proxyRoute{}, parseErr
	}
	return proxyRoute{
		pathPrefix: pathPrefix,
		backendURL: parsedURL,
	}, nil
}

func parseProxyBackendURL(backendURL string) (*url.URL, error) {
	if strings.TrimSpace(backendURL) == "" {
		return nil, fmt.Errorf("%w: empty backend url", ErrInvalidProxyRoute)
	}
	parsedURL, parseErr := url.Parse(backendURL)
	if parseErr != nil {
		return nil, fmt.Errorf("%w: parse backend url: %s", ErrInvalidProxyRoute, parseErr.Error())
	}
	if !strings.EqualFold(parsedURL.Scheme, proxySchemeHTTP) && !strings.EqualFold(parsedURL.Scheme, proxySchemeHTTPS) {
		return nil, fmt.Errorf("%w: backend url must use http or https", ErrInvalidProxyRoute)
	}
	if parsedURL.Host == "" {
		return nil, fmt.Errorf("%w: backend url must include host", ErrInvalidProxyRoute)
	}
	hostname := parsedURL.Hostname()
	if hostname == "" || strings.HasSuffix(parsedURL.Host, ":") || strings.Contains(hostname, "//") {
		return nil, fmt.Errorf("%w: backend url must include valid host", ErrInvalidProxyRoute)
	}
	return parsedURL, nil
}
