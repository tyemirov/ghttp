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
	proxyMappings := configurationManager.GetStringSlice(configKeyServeProxies)
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
