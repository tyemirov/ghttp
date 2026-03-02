package app

import (
	"fmt"

	"github.com/spf13/viper"

	"github.com/tyemirov/ghttp/internal/server"
)

func resolveRouteResponsePolicies(configurationManager *viper.Viper) (server.RouteResponsePolicies, error) {
	responseHeaderMappings := normalizeTrimmedMappings(configurationManager.GetStringSlice(configKeyServeResponseHeaders))
	responsePolicies, responsePolicyErr := server.NewRouteResponsePolicies(responseHeaderMappings)
	if responsePolicyErr != nil {
		return server.RouteResponsePolicies{}, fmt.Errorf("parse response header mappings: %w", responsePolicyErr)
	}
	return responsePolicies, nil
}
