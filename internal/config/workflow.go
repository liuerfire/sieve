package config

import (
	"encoding/json"
	"fmt"
	"os"
)

var validWorkflowProviders = map[string]struct{}{
	"anthropic": {},
	"openai":    {},
	"gemini":    {},
	"qwen":      {},
	"openrouter": {},
	"grok":      {},
}

type WorkflowConfig struct {
	LLM     WorkflowLLMConfig          `json:"llm"`
	Plugins map[string]json.RawMessage `json:"plugins,omitempty"`
	Sources []WorkflowSourceConfig     `json:"sources"`
}

type WorkflowLLMConfig struct {
	Provider string            `json:"provider"`
	BaseURL  string            `json:"baseUrl,omitempty"`
	Models   WorkflowLLMModels `json:"models"`
}

type WorkflowLLMModels struct {
	Fast     string `json:"fast"`
	Balanced string `json:"balanced"`
	Powerful string `json:"powerful"`
}

type WorkflowSourceConfig struct {
	Name    string                `json:"name"`
	Title   string                `json:"title,omitempty"`
	Context string                `json:"context,omitempty"`
	Plugins []WorkflowPluginEntry `json:"plugins"`
}

type WorkflowPluginEntry struct {
	Name    string          `json:"name"`
	Options json.RawMessage `json:"options,omitempty"`
}

func (e *WorkflowPluginEntry) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		e.Name = name
		e.Options = nil
		return nil
	}

	type pluginAlias WorkflowPluginEntry
	var entry pluginAlias
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("invalid plugin entry: %w", err)
	}
	if entry.Name == "" {
		return fmt.Errorf("invalid plugin entry: name is required")
	}
	*e = WorkflowPluginEntry(entry)
	return nil
}

func ParseWorkflowConfig(data []byte) (*WorkflowConfig, error) {
	var cfg WorkflowConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid workflow config: %w", err)
	}
	return &cfg, nil
}

func LoadWorkflowConfig(path string) (*WorkflowConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseWorkflowConfig(data)
}

func (c *WorkflowConfig) Validate() error {
	if _, ok := validWorkflowProviders[c.LLM.Provider]; !ok {
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
