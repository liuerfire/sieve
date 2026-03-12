package builtin

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

func testRunContext(source string) plugins.Context {
	return plugins.Context{
		SourceName: source,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestCollectRSS_ParsesFeedItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Example Feed</title>
    <item>
      <title>Item 1</title>
      <link>https://example.com/1</link>
      <pubDate>Wed, 11 Mar 2026 12:00:00 GMT</pubDate>
      <description>First description</description>
      <guid>guid-1</guid>
    </item>
  </channel>
</rss>`)
	}))
	defer server.Close()

	entry := config.PluginEntry{
		Name:    "builtin/collect-rss",
		Options: mustJSON(map[string]any{"url": server.URL}),
	}

	result, err := CollectRSSPlugin{}.Collect(context.Background(), entry, testRunContext("feed"))
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if result.Title != "Example Feed" {
		t.Fatalf("expected title Example Feed, got %q", result.Title)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].GUID != "guid-1" {
		t.Fatalf("expected guid-1, got %q", result.Items[0].GUID)
	}
}

func TestCollectRSS_DryRunLimitsItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Example Feed</title>
    <item><title>1</title><link>https://e/1</link><pubDate>Wed, 11 Mar 2026 12:00:00 GMT</pubDate><description>1</description><guid>1</guid></item>
    <item><title>2</title><link>https://e/2</link><pubDate>Wed, 11 Mar 2026 12:00:00 GMT</pubDate><description>2</description><guid>2</guid></item>
    <item><title>3</title><link>https://e/3</link><pubDate>Wed, 11 Mar 2026 12:00:00 GMT</pubDate><description>3</description><guid>3</guid></item>
    <item><title>4</title><link>https://e/4</link><pubDate>Wed, 11 Mar 2026 12:00:00 GMT</pubDate><description>4</description><guid>4</guid></item>
  </channel>
</rss>`)
	}))
	defer server.Close()

	entry := config.PluginEntry{
		Name:    "builtin/collect-rss",
		Options: mustJSON(map[string]any{"url": server.URL}),
	}

	result, err := CollectRSSPlugin{}.Collect(context.Background(), entry, plugins.Context{
		SourceName: "feed",
		IsDryRun:   true,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("expected dry run to limit to 3 items, got %d", len(result.Items))
	}
}

func TestDeduplicate_MarksProcessedItemsRejected(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() { _ = os.Chdir(prevWD) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	items := []types.FeedItem{
		types.FeedItem{GUID: "a", Title: "A"}.WithDefaults(),
		types.FeedItem{GUID: "b", Title: "B"}.WithDefaults(),
	}

	got, err := DeduplicatePlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{Name: "builtin/deduplicate"}, testRunContext("source"))
	if err != nil {
		t.Fatalf("ProcessItems returned error: %v", err)
	}
	if got[0].Level == types.LevelRejected || got[1].Level == types.LevelRejected {
		t.Fatal("expected first pass items to remain non-rejected")
	}

	got, err = DeduplicatePlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{Name: "builtin/deduplicate"}, testRunContext("source"))
	if err != nil {
		t.Fatalf("ProcessItems returned error: %v", err)
	}
	if got[0].Level != types.LevelRejected || got[1].Level != types.LevelRejected {
		t.Fatalf("expected both items to be rejected on second pass, got %#v", got)
	}
}

func TestDeduplicate_DryRunReturnsOriginalItems(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "output"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() { _ = os.Chdir(prevWD) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	items := []types.FeedItem{
		types.FeedItem{GUID: "a", Title: "A"}.WithDefaults(),
	}
	got, err := DeduplicatePlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{Name: "builtin/deduplicate"}, plugins.Context{
		SourceName: "source",
		IsDryRun:   true,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("ProcessItems returned error: %v", err)
	}
	if got[0].Level == types.LevelRejected {
		t.Fatalf("expected dry-run item to remain visible, got %#v", got[0])
	}
}

func TestCleanText_NormalizesWhitespace(t *testing.T) {
	items := []types.FeedItem{
		types.FeedItem{
			Title:       "\u200b  Hello  \n",
			Description: "\u200b world \n",
		}.WithDefaults(),
	}

	got, err := CleanTextPlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{Name: "builtin/clean-text"}, testRunContext("source"))
	if err != nil {
		t.Fatalf("ProcessItems returned error: %v", err)
	}
	if got[0].Title != "Hello" {
		t.Fatalf("expected cleaned title, got %q", got[0].Title)
	}
	if got[0].Description != "world" {
		t.Fatalf("expected cleaned description, got %q", got[0].Description)
	}
}

func TestReporterRSS_LogsOutputPath(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "feed.xml")
	var logs strings.Builder

	err := ReporterRSSPlugin{}.Report(context.Background(), []types.FeedItem{
		types.FeedItem{Title: "A", Link: "https://example.com/a", GUID: "a"}.WithDefaults(),
	}, config.PluginEntry{
		Name: "builtin/reporter-rss",
		Options: mustJSON(map[string]any{
			"outputPath": outputPath,
			"sourceName": "test-source",
			"title":      "Test Feed",
		}),
	}, plugins.Context{
		SourceName: "test-source",
		Logger:     slog.New(slog.NewTextHandler(&logs, nil)),
	})
	if err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	output := logs.String()
	if !strings.Contains(output, "wrote rss output") {
		t.Fatalf("expected reporter log, got %s", output)
	}
	if !strings.Contains(output, outputPath) {
		t.Fatalf("expected output path in logs, got %s", output)
	}
}

func mustJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
