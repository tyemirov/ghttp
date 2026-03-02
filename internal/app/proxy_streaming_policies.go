package app

import (
	"fmt"

	"github.com/spf13/viper"

	"github.com/tyemirov/ghttp/internal/server"
)

func resolveProxyStreamingPolicies(configurationManager *viper.Viper) (server.ProxyStreamingPolicies, error) {
	streamingMappings := normalizeCommaDelimitedMappings(configurationManager.GetStringSlice(configKeyServeProxyStreaming))
	streamingPolicies, streamingErr := server.NewProxyStreamingPolicies(streamingMappings)
	if streamingErr != nil {
		return server.ProxyStreamingPolicies{}, fmt.Errorf("parse proxy streaming mappings: %w", streamingErr)
	}
	return streamingPolicies, nil
}
