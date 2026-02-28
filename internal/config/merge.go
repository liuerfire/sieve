// Package config handles loading and parsing of JSON configuration files.
package config

import "fmt"

// mergeStrings combines global and source-specific string values.
// Source-specific values take precedence when both are non-empty.
// When both are non-empty, they are concatenated with a comma separator.
func mergeStrings(global, specific string) string {
	if specific != "" {
		if global == "" {
			return specific
		}
		return global + "," + specific
	}
	return global
}

// MergeRules returns a formatted string combining global and source-specific
// interest rules. Source-specific rules are appended to global rules.
func MergeRules(global, specific string) string {
	return mergeStrings(global, specific)
}

// BuildRulesString creates a formatted rules string for AI classification
// by merging global and source-specific interest levels.
func BuildRulesString(global GlobalConfig, src SourceConfig) string {
	return fmt.Sprintf("High: %s, Interest: %s, Uninterested: %s, Exclude: %s",
		MergeRules(global.HighInterest, src.HighInterest),
		MergeRules(global.Interest, src.Interest),
		MergeRules(global.Uninterested, src.Uninterested),
		MergeRules(global.Exclude, src.Exclude),
	)
}

// ResolveAIConfig returns the effective AI configuration by merging
// global defaults with source-specific overrides.
// Source-specific values override global values when both are set.
func ResolveAIConfig(global *AIConfig, source *AIConfig) *AIConfig {
	// If no source-specific config, use global
	if source == nil {
		return global
	}

	// If no global config, use source-specific
	if global == nil {
		return source
	}

	// Merge: start with global, override with source-specific non-empty fields
	merged := *global
	if source.Provider != "" {
		merged.Provider = source.Provider
	}
	if source.Model != "" {
		merged.Model = source.Model
	}
	return &merged
}
