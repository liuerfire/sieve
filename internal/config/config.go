// Package config handles loading and parsing of JSON configuration files.
package config

import (
	"encoding/json"
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
	return &cfg, nil
}
