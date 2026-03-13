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

func TestDeduplicate_DryRunDoesNotPersistHistory(t *testing.T) {
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
	_, err = DeduplicatePlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{Name: "builtin/deduplicate"}, plugins.Context{
		SourceName: "source",
		IsDryRun:   true,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("ProcessItems returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join("output", "source-processed.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no persisted history in dry-run, got err=%v", err)
	}
}

func TestDeduplicate_DoesNotTreatEmptyGUIDAsDuplicate(t *testing.T) {
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
		types.FeedItem{GUID: "", Title: "A", Link: "https://example.com/a"}.WithDefaults(),
		types.FeedItem{GUID: "", Title: "B", Link: "https://example.com/b"}.WithDefaults(),
	}

	_, err = DeduplicatePlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{Name: "builtin/deduplicate"}, testRunContext("source"))
	if err != nil {
		t.Fatalf("ProcessItems returned error: %v", err)
	}

	got, err := DeduplicatePlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{Name: "builtin/deduplicate"}, testRunContext("source"))
	if err != nil {
		t.Fatalf("ProcessItems returned error: %v", err)
	}
	if got[0].Level == types.LevelRejected || got[1].Level == types.LevelRejected {
		t.Fatalf("expected empty-guid items to remain visible, got %#v", got)
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

func TestReporterRSS_DryRunDoesNotWriteOutput(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "feed.xml")

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
		IsDryRun:   true,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("expected no RSS output in dry-run, got err=%v", err)
	}
}

func TestReporterRSS_ReturnsReadError(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "feed.xml")
	if err := os.WriteFile(outputPath, []byte("<rss"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

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
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err == nil {
		t.Fatal("expected invalid existing RSS to return an error")
	}
}

func TestReporterHTML_WritesVisibleItemsOnly(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "feed.html")
	var logs strings.Builder

	err := ReporterHTMLPlugin{}.Report(context.Background(), []types.FeedItem{
		types.FeedItem{
			Title:       "Visible item",
			Link:        "https://example.com/a",
			GUID:        "a",
			Description: "<p>Summary body</p>",
			Level:       types.LevelCritical,
			Reason:      "Strong match",
		}.WithDefaults(),
		types.FeedItem{
			Title:       "Rejected item",
			Link:        "https://example.com/b",
			GUID:        "b",
			Description: "<p>Should not appear</p>",
			Level:       types.LevelRejected,
			Reason:      "No match",
		}.WithDefaults(),
	}, config.PluginEntry{
		Name: "builtin/reporter-html",
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

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	output := string(data)
	for _, want := range []string{"Test Feed", "Visible item", "Summary body", "Strong match", "Critical", "Open original"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected html output to contain %q, got %s", want, output)
		}
	}
	if strings.Contains(output, "Rejected item") {
		t.Fatalf("expected rejected item to be omitted, got %s", output)
	}
	if !strings.Contains(logs.String(), "wrote html output") {
		t.Fatalf("expected html reporter log, got %s", logs.String())
	}
}

func TestReporterHTML_DryRunDoesNotWriteOutput(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "feed.html")

	err := ReporterHTMLPlugin{}.Report(context.Background(), []types.FeedItem{
		types.FeedItem{Title: "A", Link: "https://example.com/a", GUID: "a"}.WithDefaults(),
	}, config.PluginEntry{
		Name: "builtin/reporter-html",
		Options: mustJSON(map[string]any{
			"outputPath": outputPath,
			"sourceName": "test-source",
			"title":      "Test Feed",
		}),
	}, plugins.Context{
		SourceName: "test-source",
		IsDryRun:   true,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("expected no HTML output in dry-run, got err=%v", err)
	}
}

func TestReporterHTML_RequiresOutputPath(t *testing.T) {
	err := ReporterHTMLPlugin{}.Report(context.Background(), nil, config.PluginEntry{
		Name:    "builtin/reporter-html",
		Options: mustJSON(map[string]any{}),
	}, plugins.Context{
		SourceName: "test-source",
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err == nil {
		t.Fatal("expected missing outputPath to fail")
	}
}

func TestFetchMeta_ClearsMetaOnHTTPErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "blocked", http.StatusForbidden)
	}))
	defer server.Close()

	items := []types.FeedItem{
		types.FeedItem{Title: "A", Link: server.URL}.WithDefaults(),
	}

	got, err := FetchMetaPlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{Name: "builtin/fetch-meta"}, testRunContext("source"))
	if err != nil {
		t.Fatalf("ProcessItems returned error: %v", err)
	}
	if meta := got[0].Extra["meta"]; meta != "" {
		t.Fatalf("expected empty meta on HTTP error, got %#v", meta)
	}
}

func TestFetchContent_ClearsContentOnHTTPErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "blocked", http.StatusTooManyRequests)
	}))
	defer server.Close()

	items := []types.FeedItem{
		types.FeedItem{Title: "A", Link: server.URL}.WithDefaults(),
	}

	got, err := FetchContentPlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{Name: "builtin/fetch-content"}, testRunContext("source"))
	if err != nil {
		t.Fatalf("ProcessItems returned error: %v", err)
	}
	if content := got[0].Extra["content"]; content != "" {
		t.Fatalf("expected empty content on HTTP error, got %#v", content)
	}
}

func TestCollectRSSHub_ReturnsErrorOnHTTPStatusFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream failed", http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := CollectRSSHubPlugin{}.Collect(context.Background(), config.PluginEntry{
		Name:    "builtin/collect-rsshub",
		Options: mustJSON(map[string]any{"route": server.URL}),
	}, testRunContext("source"))
	if err == nil {
		t.Fatal("expected HTTP status failure to return an error")
	}
}

func mustJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
