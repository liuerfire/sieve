// Package config handles loading and parsing of JSON configuration files.
package config

import (
	"encoding/json"
	"fmt"
	"os"
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
	HighInterest      string    `json:"high_interest"`
	Interest          string    `json:"interest"`
	Uninterested      string    `json:"uninterested"`
	Exclude           string    `json:"exclude"`
	PreferredLanguage string    `json:"preferred_language"`
	Timeout           int       `json:"timeout"`
	AI                *AIConfig `json:"ai,omitempty"`
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
	for i, src := range c.Sources {
		if src.Name == "" {
			return fmt.Errorf("source[%d]: name is required", i)
		}
		if src.URL == "" {
			return fmt.Errorf("source[%d]: URL is required", i)
		}
	}
	return nil
}
