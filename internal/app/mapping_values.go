package app

import "strings"

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
