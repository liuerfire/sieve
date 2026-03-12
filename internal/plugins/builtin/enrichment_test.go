package builtin

import (
	"context"
	"encoding/xml"
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

func TestFetchMeta_StoresMetaDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<html><head><meta name="description" content="meta description"></head></html>`)
	}))
	defer server.Close()

	items := []types.FeedItem{
		types.FeedItem{Link: server.URL}.WithDefaults(),
	}

	got, err := FetchMetaPlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{Name: "builtin/fetch-meta"}, testRunContext("source"))
	if err != nil {
		t.Fatalf("ProcessItems returned error: %v", err)
	}
	if got[0].Extra["meta"] != "meta description" {
		t.Fatalf("expected meta description, got %#v", got[0].Extra["meta"])
	}
}

func TestFetchContent_FiltersImagesAndCapturesText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<html><body><article>
			<img src="https://example.com/category-icon.png" alt="filtered" />
			<p>Hello</p>
			<img src="https://example.com/photo.jpg" alt="hero" width="100" />
			<p>World</p>
		</article></body></html>`)
	}))
	defer server.Close()

	items := []types.FeedItem{
		types.FeedItem{Link: server.URL}.WithDefaults(),
	}

	got, err := FetchContentPlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{Name: "builtin/fetch-content"}, testRunContext("source"))
	if err != nil {
		t.Fatalf("ProcessItems returned error: %v", err)
	}

	content, _ := got[0].Extra["content"].(string)
	if !strings.Contains(content, "Hello") || !strings.Contains(content, "[IMAGE_0]") || !strings.Contains(content, "World") {
		t.Fatalf("expected text with placeholder, got %q", content)
	}
	images, _ := got[0].Extra["images"].([]map[string]any)
	if len(images) != 1 {
		t.Fatalf("expected 1 kept image, got %#v", got[0].Extra["images"])
	}
	if images[0]["src"] != "https://example.com/photo.jpg" {
		t.Fatalf("unexpected image src: %#v", images[0]["src"])
	}
}

func TestReporterRSS_FormatsStarsAndExcludesRejected(t *testing.T) {
	items := []types.FeedItem{
		types.FeedItem{
			Title:       "Critical",
			Link:        "https://example.com/critical",
			PubDate:     "2026-03-11T12:00:00Z",
			Description: "Body",
			GUID:        "critical-guid",
			Level:       types.LevelCritical,
			Reason:      "Must read",
		}.WithDefaults(),
		types.FeedItem{
			Title:       "Rejected",
			Link:        "https://example.com/rejected",
			PubDate:     "2026-03-11T12:00:00Z",
			Description: "Skip",
			GUID:        "rejected-guid",
			Level:       types.LevelRejected,
			Reason:      "Nope",
		}.WithDefaults(),
	}

	formatted := FormatRSSItems(items, true)
	if len(formatted) != 1 {
		t.Fatalf("expected 1 formatted item, got %d", len(formatted))
	}
	if formatted[0].Title != "⭐⭐ Critical" {
		t.Fatalf("unexpected title %q", formatted[0].Title)
	}
	if !strings.Contains(formatted[0].Description, "[critical] Must read") {
		t.Fatalf("unexpected description %q", formatted[0].Description)
	}
}

func TestReporterRSS_AppendsExistingItemsAndLimitsTo50(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "feed.xml")

	existing := make([]rssItem, 0, 49)
	for range 49 {
		existing = append(existing, rssItem{
			Title:       "Old",
			Link:        "https://example.com/old",
			GUID:        rssGUID{Value: "old-guid"},
			Description: "old",
			PubDate:     "2026-03-10T12:00:00Z",
		})
	}
	if err := writeRSS(path, rssFeed{
		Channel: rssChannel{
			Title: "Existing",
			Items: existing,
		},
	}); err != nil {
		t.Fatalf("writeRSS: %v", err)
	}

	items := []types.FeedItem{
		types.FeedItem{
			Title:       "New",
			Link:        "https://example.com/new",
			PubDate:     "2026-03-11T12:00:00Z",
			Description: "new",
			GUID:        "new-guid",
			Level:       types.LevelRecommended,
			Reason:      "Worth it",
		}.WithDefaults(),
	}

	entry := config.PluginEntry{
		Name: "builtin/reporter-rss",
		Options: mustJSON(map[string]any{
			"outputPath": path,
			"sourceName": "source",
			"title":      "Source Feed",
		}),
	}
	err := ReporterRSSPlugin{}.Report(context.Background(), items, entry, plugins.Context{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var feed rssFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(feed.Channel.Items) != 50 {
		t.Fatalf("expected 50 items, got %d", len(feed.Channel.Items))
	}
	if feed.Channel.Items[0].Title != "⭐ New" {
		t.Fatalf("expected newest item first, got %q", feed.Channel.Items[0].Title)
	}
}
