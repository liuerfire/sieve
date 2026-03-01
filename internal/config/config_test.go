package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadConfig_NonExistentFile(t *testing.T) {
	_, err := LoadConfig("non_existent.json")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestLoadConfig_ValidJSON(t *testing.T) {
	content := `{
  "$schema": "./schemas/config.schema.json",
  "global": {
    "high_interest": "major_news,programming_tools,productivity",
    "interest": "market_trends,ai_software,programming_languages,open_source,science",
    "uninterested": "industry_figures,history,infrastructure,crypto,chips,iphone,autonomous_driving",
    "exclude": "nft,cars,aviation,gaming_consoles,development_boards,biographies",
    "preferred_language": "en",
    "timeout": 5
  },
  "sources": [
    {
      "name": "cnbeta",
      "title": "cnBeta.com - Tech News",
      "url": "https://www.cnbeta.com.tw/backend.php",
      "exclude": "health_tips,entertainment_gossip",
      "plugins": ["cnbeta_fetch_content"],
      "summarize": false
    },
    {
      "name": "hacker-news",
      "url": "https://hnrss.org/best",
      "uninterested": "security,privacy",
      "exclude": "government_policy,social_news,code_golf",
      "plugins": ["fetch_meta", "fetch_content", "hn_fetch_comments"],
      "summarize": true,
      "timeout": 20
    }
  ]
}`
	tmpfile, err := os.CreateTemp("", "config*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Global.PreferredLanguage != "en" {
		t.Errorf("expected preferred_language 'en', got '%s'", cfg.Global.PreferredLanguage)
	}
	if len(cfg.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(cfg.Sources))
	}
	if cfg.Sources[0].Name != "cnbeta" {
		t.Errorf("expected first source name 'cnbeta', got '%s'", cfg.Sources[0].Name)
	}
	if cfg.Sources[1].Timeout != 20 {
		t.Errorf("expected second source timeout 20, got %d", cfg.Sources[1].Timeout)
	}
}

func TestConfig_Validate_InvalidProvider(t *testing.T) {
	cfg := &Config{
		Global: GlobalConfig{
			HighInterest:      "test",
			Interest:          "test",
			Uninterested:      "test",
			Exclude:           "test",
			PreferredLanguage: "en",
			Timeout:           5,
			AI: &AIConfig{
				Provider: "invalid-provider",
				Model:    "test-model",
			},
		},
		Sources: []SourceConfig{
			{Name: "test", URL: "https://example.com/feed.xml"},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid provider, got nil")
	}
	if !strings.Contains(err.Error(), "provider") {
		t.Errorf("expected provider error, got: %v", err)
	}
}

func TestConfig_Validate_ValidProviders(t *testing.T) {
	for _, provider := range []string{"gemini", "qwen"} {
		t.Run(provider, func(t *testing.T) {
			cfg := &Config{
				Global: GlobalConfig{
					HighInterest:      "test",
					Interest:          "test",
					Uninterested:      "test",
					Exclude:           "test",
					PreferredLanguage: "en",
					Timeout:           5,
					AI:                &AIConfig{Provider: provider},
				},
				Sources: []SourceConfig{
					{Name: "test", URL: "https://example.com/feed.xml"},
				},
			}
			err := cfg.Validate()
			if err != nil {
				t.Errorf("expected no error for valid provider %s, got: %v", provider, err)
			}
		})
	}
}

func TestConfig_Validate_SourceAIProvider(t *testing.T) {
	cfg := &Config{
		Global: GlobalConfig{
			HighInterest:      "test",
			Interest:          "test",
			Uninterested:      "test",
			Exclude:           "test",
			PreferredLanguage: "en",
			Timeout:           5,
		},
		Sources: []SourceConfig{
			{
				Name: "test",
				URL:  "https://example.com/feed.xml",
				AI:   &AIConfig{Provider: "invalid"},
			},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid source AI provider, got nil")
	}
	if !strings.Contains(err.Error(), "source[0]") {
		t.Errorf("expected source index in error, got: %v", err)
	}
}

func TestConfig_Validate_GlobalAIProvider(t *testing.T) {
	cfg := &Config{
		Global: GlobalConfig{
			HighInterest:      "test",
			Interest:          "test",
			Uninterested:      "test",
			Exclude:           "test",
			PreferredLanguage: "en",
			Timeout:           5,
			AI:                &AIConfig{Provider: "invalid"},
		},
		Sources: []SourceConfig{
			{Name: "test", URL: "https://example.com/feed.xml"},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid global AI provider, got nil")
	}
	if !strings.Contains(err.Error(), "invalid AI provider") {
		t.Errorf("expected provider error in message, got: %v", err)
	}
}

func TestConfig_Validate_CaseInsensitiveProviders(t *testing.T) {
	for _, provider := range []string{"Gemini", "GEMINI", "Qwen", "QWEN"} {
		t.Run(provider, func(t *testing.T) {
			cfg := &Config{
				Global: GlobalConfig{
					HighInterest:      "test",
					Interest:          "test",
					Uninterested:      "test",
					Exclude:           "test",
					PreferredLanguage: "en",
					Timeout:           5,
					AI:                &AIConfig{Provider: provider},
				},
				Sources: []SourceConfig{
					{Name: "test", URL: "https://example.com/feed.xml"},
				},
			}
			err := cfg.Validate()
			if err != nil {
				t.Errorf("expected no error for case-insensitive provider %s, got: %v", provider, err)
			}
		})
	}
}
