package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"github.com/tyemirov/ghttp/internal/server"
)

var errInvalidProxyConfiguration = errors.New("proxy.configuration.invalid")

func resolveProxyRoutes(configurationManager *viper.Viper) (server.ProxyRoutes, error) {
	proxyMappings := normalizeProxyMappings(configurationManager.GetStringSlice(configKeyServeProxies))
	legacyBackend := strings.TrimSpace(configurationManager.GetString(configKeyProxyBackend))
	legacyPathPrefix := strings.TrimSpace(configurationManager.GetString(configKeyProxyPathPrefix))

	if len(proxyMappings) > 0 {
		if legacyBackend != "" || legacyPathPrefix != "" {
			return server.ProxyRoutes{}, fmt.Errorf("%w: proxy mappings cannot be combined with legacy proxy flags", errInvalidProxyConfiguration)
		}
		proxyRoutes, proxyErr := server.NewProxyRoutes(proxyMappings)
		if proxyErr != nil {
			return server.ProxyRoutes{}, fmt.Errorf("parse proxy mappings: %w", proxyErr)
		}
		return proxyRoutes, nil
	}

	if legacyBackend == "" && legacyPathPrefix == "" {
		return server.ProxyRoutes{}, nil
	}
	if legacyBackend == "" || legacyPathPrefix == "" {
		return server.ProxyRoutes{}, fmt.Errorf("%w: both proxy-backend and proxy-path must be set", errInvalidProxyConfiguration)
	}

	proxyRoutes, proxyErr := server.NewProxyRoutesFromLegacy(legacyPathPrefix, legacyBackend)
	if proxyErr != nil {
		return server.ProxyRoutes{}, fmt.Errorf("parse legacy proxy mapping: %w", proxyErr)
	}
	return proxyRoutes, nil
}

func normalizeProxyMappings(proxyMappings []string) []string {
	if len(proxyMappings) == 0 {
		return proxyMappings
	}
	normalized := make([]string, 0, len(proxyMappings))
	for _, mapping := range proxyMappings {
		trimmedMapping := strings.TrimSpace(mapping)
		if trimmedMapping == "" {
			continue
		}
		if strings.Contains(trimmedMapping, ",") {
			segments := strings.Split(trimmedMapping, ",")
			for _, segment := range segments {
				trimmedSegment := strings.TrimSpace(segment)
				if trimmedSegment != "" {
					normalized = append(normalized, trimmedSegment)
				}
			}
			continue
		}
		normalized = append(normalized, trimmedMapping)
	}
	return normalized
}
