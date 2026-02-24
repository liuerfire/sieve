package engine

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/liuerfire/sieve/internal/ai"
	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/storage"
)

func TestEngine_GenerateHTML(t *testing.T) {
	ctx := context.Background()
	dbPath := "test_html.db"
	defer os.Remove(dbPath)

	s, err := storage.InitDB(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Add mock items
	item := &storage.Item{
		ID:            "test-1",
		Title:         "High Interest Item",
		Source:        "test-source",
		InterestLevel: "high_interest",
		Summary:       "<p>This is a summary</p>",
		Content:       "This is full content",
		PublishedAt:   time.Now(),
	}
	if err := s.SaveItem(ctx, item); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{}
	a := ai.NewClient(ai.Gemini, "dummy")
	eng := NewEngine(cfg, s, a)

	outputPath := "test.html"
	defer os.Remove(outputPath)

	if err := eng.GenerateHTML(ctx, outputPath); err != nil {
		t.Fatalf("GenerateHTML failed: %v", err)
	}

	// Verify output
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	// After fix, summary is used as Description in htmlItem
	expected := []string{"<html", "High Interest Item", "⭐⭐", "This is a summary"}
	for _, exp := range expected {
		if !strings.Contains(content, exp) {
			t.Errorf("expected HTML to contain '%s'", exp)
		}
	}
}
