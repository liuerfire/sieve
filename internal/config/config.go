// Package config handles loading and parsing of JSON configuration files.
package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

var validProviders = map[string]bool{"gemini": true, "qwen": true}

// InterestLevel represents the classification level for an item
type InterestLevel string

const (
	HighInterest InterestLevel = "high_interest"
	Interest     InterestLevel = "interest"
	Uninterested InterestLevel = "uninterested"
	Exclude      InterestLevel = "exclude"
)

type Config struct {
	Schema  string         `json:"$schema"`
	Global  GlobalConfig   `json:"global"`
	Sources []SourceConfig `json:"sources"`
}

type AIConfig struct {
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
}

type GlobalConfig struct {
	HighInterest          string    `json:"high_interest"`
	Interest              string    `json:"interest"`
	Uninterested          string    `json:"uninterested"`
	Exclude               string    `json:"exclude"`
	PreferredLanguage     string    `json:"preferred_language"`
	Timeout               int       `json:"timeout"`
	AI                    *AIConfig `json:"ai,omitempty"`
	AITimeBetweenRequests int       `json:"ai_time_between_ms,omitempty"`
	AIBurstLimit          int       `json:"ai_burst_limit,omitempty"`
	AIMaxConcurrency      int       `json:"ai_max_concurrency,omitempty"`
}

type SourceConfig struct {
	Name         string    `json:"name"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	HighInterest string    `json:"high_interest"`
	Interest     string    `json:"interest"`
	Uninterested string    `json:"uninterested"`
	Exclude      string    `json:"exclude"`
	Plugins      []string  `json:"plugins"`
	Summarize    bool      `json:"summarize"`
	Timeout      int       `json:"timeout"`
	AI           *AIConfig `json:"ai,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

// Validate checks that the configuration is valid and complete.
func (c *Config) Validate() error {
	if len(c.Sources) == 0 {
		return fmt.Errorf("at least one source is required")
	}

	if c.Global.AI != nil && c.Global.AI.Provider != "" {
		if !validProviders[strings.ToLower(c.Global.AI.Provider)] {
			return fmt.Errorf("invalid AI provider %q, must be 'gemini' or 'qwen'", c.Global.AI.Provider)
		}
	}

	for i, src := range c.Sources {
		if src.Name == "" {
			return fmt.Errorf("source[%d]: name is required", i)
		}
		if src.URL == "" {
			return fmt.Errorf("source[%d]: URL is required", i)
		}
		// Validate URL format
		if _, err := url.Parse(src.URL); err != nil {
			return fmt.Errorf("source[%d]: invalid URL %q: %w", i, src.URL, err)
		}
		if src.AI != nil && src.AI.Provider != "" {
			if !validProviders[strings.ToLower(src.AI.Provider)] {
				return fmt.Errorf("source[%d]: invalid AI provider %q, must be 'gemini' or 'qwen'", i, src.AI.Provider)
			}
		}
	}

	return nil
}
