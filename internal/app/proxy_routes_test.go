package app

import (
	"testing"

	"github.com/spf13/viper"
)

func TestResolveProxyRoutesSplitsCommaDelimitedMappings(testingInstance *testing.T) {
	configurationManager := viper.New()
	configurationManager.Set(configKeyServeProxies, "/auth=http://tauth:8081,/me=http://tauth:8081,/api=http://tauth:8081,/tauth.js=http://tauth:8081")

	proxyRoutes, err := resolveProxyRoutes(configurationManager)
	if err != nil {
		testingInstance.Fatalf("resolve proxy routes: %v", err)
	}
	if proxyRoutes.Count() != 4 {
		testingInstance.Fatalf("expected 4 proxy routes, got %d", proxyRoutes.Count())
	}
}
