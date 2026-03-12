package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse_ValidConfig(t *testing.T) {
	cfg, err := Parse([]byte(`{
		"llm": {
			"provider": "openai",
			"baseUrl": "https://example.com/v1",
			"models": {
				"fast": "gpt-fast",
				"balanced": "gpt-balanced",
				"powerful": "gpt-powerful"
			}
		},
		"plugins": {
			"builtin/reporter-rss": {
				"outputPath": "output/hacker-news.xml"
			}
		},
		"sources": [
			{
				"name": "hacker-news",
				"title": "Hacker News",
				"context": "Best posts",
				"plugins": [
					"builtin/collect-rss",
					{
						"name": "builtin/reporter-rss",
						"options": {
							"showReason": true
						}
					}
				]
			}
		]
	}`))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if cfg.LLM.Provider != "openai" {
		t.Fatalf("expected provider openai, got %q", cfg.LLM.Provider)
	}
	if cfg.LLM.Models.Balanced != "gpt-balanced" {
		t.Fatalf("expected balanced model gpt-balanced, got %q", cfg.LLM.Models.Balanced)
	}
	if len(cfg.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(cfg.Sources))
	}
	if len(cfg.Sources[0].Plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(cfg.Sources[0].Plugins))
	}
	if cfg.Sources[0].Plugins[1].Name != "builtin/reporter-rss" {
		t.Fatalf("expected reporter plugin, got %q", cfg.Sources[0].Plugins[1].Name)
	}
}

func TestParse_InvalidPluginEntry(t *testing.T) {
	_, err := Parse([]byte(`{
		"llm": {
			"provider": "openai",
			"models": {
				"fast": "a",
				"balanced": "b",
				"powerful": "c"
			}
		},
		"sources": [
			{
				"name": "broken",
				"plugins": [
					{
						"options": {
							"showReason": true
						}
					}
				]
			}
		]
	}`))
	if err == nil {
		t.Fatal("expected invalid plugin entry to fail")
	}
}

func TestLoad_ReadsJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{
		"llm": {
			"provider": "gemini",
			"models": {
				"fast": "gemini-fast",
				"balanced": "gemini-balanced",
				"powerful": "gemini-powerful"
			}
		},
		"sources": [
			{
				"name": "cnbeta",
				"plugins": ["builtin/collect-rss"]
			}
		]
	}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Sources[0].Name != "cnbeta" {
		t.Fatalf("expected source cnbeta, got %q", cfg.Sources[0].Name)
	}
}

func TestParse_AcceptsQwenProvider(t *testing.T) {
	cfg, err := Parse([]byte(`{
		"llm": {
			"provider": "qwen",
			"models": {
				"fast": "qwen-turbo",
				"balanced": "qwen-plus",
				"powerful": "qwen-max"
			}
		},
		"sources": [
			{
				"name": "hacker-news",
				"plugins": ["builtin/reporter-rss"]
			}
		]
	}`))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cfg.LLM.Provider != "qwen" {
		t.Fatalf("expected provider qwen, got %q", cfg.LLM.Provider)
	}
}
