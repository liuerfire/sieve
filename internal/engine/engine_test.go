package engine

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/liuerfire/sieve/internal/ai"
	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/storage"
)

func TestEngine_Run(t *testing.T) {
	ctx := context.Background()

	// Mock RSS feed
	rssServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8" ?>
<rss version="2.0">
<channel>
  <item>
    <title>Test Item 1</title>
    <link>http://example.com/1</link>
    <description>Description 1</description>
  </item>
</channel>
</rss>`)
	}))
	defer rssServer.Close()

	// Mock AI provider
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Simulate classification response
		w.Write([]byte(`{
			"candidates": [{
				"content": {
					"parts": [{
						"text": "{\"type\": \"high_interest\", \"reason\": \"matched keywords\"}"
					}]
				}
			}]
		}`))
	}))
	defer aiServer.Close()

	// Setup Config
	cfg := &config.Config{
		Global: config.GlobalConfig{
			HighInterest:      "test",
			PreferredLanguage: "en",
		},
		Sources: []config.SourceConfig{
			{
				Name:      "test-source",
				URL:       rssServer.URL,
				Summarize: false,
			},
		},
	}

	// Setup Storage
	dbPath := "test_engine.db"
	defer os.Remove(dbPath)
	s, err := storage.InitDB(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Setup AI Client
	a := ai.NewClient(ai.Gemini, "dummy-key", ai.WithBaseURL(aiServer.URL))

	// Setup Engine
	eng := NewEngine(cfg, s, a)

	// Run Engine
	if err := eng.Run(ctx); err != nil {
		t.Fatalf("Engine.Run failed: %v", err)
	}
	defer os.Remove("index.json")

	// Verify items in storage
	items, err := s.GetItems(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item in storage, got %d", len(items))
	}

	if items[0].InterestLevel != "high_interest" {
		t.Errorf("expected interest level 'high_interest', got '%s'", items[0].InterestLevel)
	}

	// Verify index.json exists
	if _, err := os.Stat("index.json"); os.IsNotExist(err) {
		t.Errorf("index.json was not generated")
	}
}
