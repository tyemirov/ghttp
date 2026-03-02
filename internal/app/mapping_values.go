package app

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func normalizeTrimmedMappings(mappings []string) []string {
	if len(mappings) == 0 {
		return mappings
	}
	normalized := make([]string, 0, len(mappings))
	for _, mapping := range mappings {
		trimmedMapping := strings.TrimSpace(mapping)
		if trimmedMapping == "" {
			continue
		}
		normalized = append(normalized, trimmedMapping)
	}
	return normalized
}

func normalizeCommaDelimitedMappings(mappings []string) []string {
	if len(mappings) == 0 {
		return mappings
	}
	normalized := make([]string, 0, len(mappings))
	for _, mapping := range mappings {
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

func resolveMappingValues(configurationManager *viper.Viper, configurationKey string) []string {
	rawValue := configurationManager.Get(configurationKey)
	switch typedValue := rawValue.(type) {
	case []string:
		return normalizeTrimmedMappings(typedValue)
	case string:
		trimmedValue := strings.TrimSpace(typedValue)
		if trimmedValue == "" {
			return nil
		}
		return []string{trimmedValue}
	case []interface{}:
		normalized := make([]string, 0, len(typedValue))
		for _, item := range typedValue {
			itemValue := strings.TrimSpace(fmt.Sprintf("%v", item))
			if itemValue != "" {
				normalized = append(normalized, itemValue)
			}
		}
		return normalized
	default:
		return normalizeTrimmedMappings(configurationManager.GetStringSlice(configurationKey))
	}
}
