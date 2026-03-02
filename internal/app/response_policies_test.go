package app

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestResolveRouteResponsePoliciesReadsEnvValueWithWhitespace(t *testing.T) {
	t.Setenv("GHTTP_SERVE_RESPONSE_HEADERS", "/=Cache-Control:public, max-age=600")

	configurationManager := viper.New()
	configurationManager.SetDefault(configKeyServeResponseHeaders, []string{})
	configurationManager.SetEnvPrefix(strings.ToUpper(defaultApplicationName))
	configurationManager.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	configurationManager.AutomaticEnv()

	policies, resolveErr := resolveRouteResponsePolicies(configurationManager)
	if resolveErr != nil {
		t.Fatalf("resolve route response policies: %v", resolveErr)
	}

	resolvedHeaders := policies.HeadersForPath("/index.html")
	if resolvedHeaders["Cache-Control"] != "public, max-age=600" {
		t.Fatalf("expected cache-control policy from env, got %q", resolvedHeaders["Cache-Control"])
	}
}
