package config

import (
	"encoding/json"
	"fmt"
	"os"
)

var validProviders = map[string]struct{}{
	"anthropic":  {},
	"openai":     {},
	"gemini":     {},
	"qwen":       {},
	"openrouter": {},
	"grok":       {},
}

type Config struct {
	LLM     LLMConfig                  `json:"llm"`
	Plugins map[string]json.RawMessage `json:"plugins,omitempty"`
	Sources []SourceConfig             `json:"sources"`
}

type LLMConfig struct {
	Provider string    `json:"provider"`
	BaseURL  string    `json:"baseUrl,omitempty"`
	Models   LLMModels `json:"models"`
}

type LLMModels struct {
	Fast     string `json:"fast"`
	Balanced string `json:"balanced"`
	Powerful string `json:"powerful"`
}

type SourceConfig struct {
	Name    string        `json:"name"`
	Title   string        `json:"title,omitempty"`
	Context string        `json:"context,omitempty"`
	Plugins []PluginEntry `json:"plugins"`
}

type PluginEntry struct {
	Name    string          `json:"name"`
	Options json.RawMessage `json:"options,omitempty"`
}

func (e *PluginEntry) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		e.Name = name
		e.Options = nil
		return nil
	}

	type pluginAlias PluginEntry
	var entry pluginAlias
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("invalid plugin entry: %w", err)
	}
	if entry.Name == "" {
		return fmt.Errorf("invalid plugin entry: name is required")
	}
	*e = PluginEntry(entry)
	return nil
}

func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

func (c *Config) Validate() error {
	if _, ok := validProviders[c.LLM.Provider]; !ok {
		return fmt.Errorf("unsupported llm.provider %q", c.LLM.Provider)
	}
	if c.LLM.Models.Fast == "" || c.LLM.Models.Balanced == "" || c.LLM.Models.Powerful == "" {
		return fmt.Errorf("llm.models.fast, llm.models.balanced, and llm.models.powerful are required")
	}
	if len(c.Sources) == 0 {
		return fmt.Errorf("at least one source is required")
	}
	for i, src := range c.Sources {
		if src.Name == "" {
			return fmt.Errorf("source[%d]: name is required", i)
		}
		if len(src.Plugins) == 0 {
			return fmt.Errorf("source[%d]: at least one plugin is required", i)
		}
		for j, plugin := range src.Plugins {
			if plugin.Name == "" {
				return fmt.Errorf("source[%d].plugins[%d]: name is required", i, j)
			}
		}
	}
	return nil
}
