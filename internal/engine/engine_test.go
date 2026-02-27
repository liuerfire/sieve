package engine

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
						"text": "{\"thought\": \"Reasoning\", \"type\": \"high_interest\", \"reason\": \"matched keywords\"}"
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
	a := ai.NewClient()
	a.AddProvider(ai.Gemini, "dummy-key")
	ai.WithBaseURL(ai.Gemini, aiServer.URL)(a)

	// Setup Engine
	eng := NewEngine(cfg, s, a)

	// Run Engine
	if err := eng.Run(ctx); err != nil {
		t.Fatalf("Engine.Run failed: %v", err)
	}
	defer os.Remove("index.json")

	// Verify items in storage
	var items []*storage.Item
	for it, err := range s.AllItems(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, it)
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

func TestEngine_ProcessItem_Pipeline(t *testing.T) {
	ctx := context.Background()
	dbPath := "test_pipeline.db"
	defer os.Remove(dbPath)
	s, _ := storage.InitDB(ctx, dbPath)
	defer s.Close()

	// Mock AI - First call returns high_interest, second returns interest
	aiCalls := 0
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		aiCalls++
		w.Header().Set("Content-Type", "application/json")
		level := "high_interest"
		if aiCalls > 1 {
			level = "interest" // Changed in second pass
		}

		// Distinguish between classification (which uses isJSON/JSON response) and summarization (which returns raw text)
		var text string
		if aiCalls == 2 {
			// Second call in this test sequence is actually the Summarize call
			text = "Summary text"
		} else {
			// First and third calls are Classify
			text = fmt.Sprintf(`{"thought": "Thought %d", "type": "%s", "reason": "pass %d"}`, aiCalls, level, aiCalls)
		}

		resp := fmt.Sprintf(`{"candidates": [{"content": {"parts": [{"text": "%s"}]}}]}`, strings.ReplaceAll(text, `"`, `\"`))
		w.Write([]byte(resp))
	}))
	defer aiServer.Close()

	a := ai.NewClient()
	a.AddProvider(ai.Gemini, "dummy-key")
	ai.WithBaseURL(ai.Gemini, aiServer.URL)(a)
	cfg := &config.Config{Global: config.GlobalConfig{PreferredLanguage: "en"}}
	eng := NewEngine(cfg, s, a)

	src := config.SourceConfig{Name: "test", Summarize: true}
	item := &storage.Item{ID: "unique-1", Title: "Title", Description: "Long enough description for summarization"}

	// 1. First run: Should do Phase 1 -> Summarize -> Phase 2
	err := eng.processItem(ctx, src, item, "")
	if err != nil {
		t.Fatal(err)
	}

	// 3 AI calls: 1 (classify) + 1 (summarize) + 1 (classify again)
	if aiCalls != 3 {
		t.Errorf("expected 3 AI calls (classify, summarize, re-classify), got %d", aiCalls)
	}

	// 2. Second run: Should skip immediately due to GUID check
	err = eng.processItem(ctx, src, item, "")
	if err != nil {
		t.Fatal(err)
	}
	if aiCalls != 3 {
		t.Errorf("expected still 3 AI calls after second run (early exit), got %d", aiCalls)
	}
}
