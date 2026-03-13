package hacker_news

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

func TestHackerNews_ExtractsNestedComments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{
			"children": [
				{"author":"a","text":"top","points":10,"children":[
					{"author":"b","text":"child","points":3,"children":[
						{"author":"c","text":"grandchild","points":1}
					]}
				]}
			]
		}`)
	}))
	defer server.Close()

	oldURL := algoliaItemURL
	algoliaItemURL = server.URL + "/%s"
	defer func() { algoliaItemURL = oldURL }()

	items := []types.FeedItem{
		types.FeedItem{GUID: "https://news.ycombinator.com/item?id=123"}.WithDefaults(),
	}
	got, err := Plugin{}.ProcessItems(context.Background(), items, config.PluginEntry{Name: "hacker-news"}, plugins.Context{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("ProcessItems: %v", err)
	}
	comments := got[0].Extra["comments"].([]map[string]any)
	if len(comments) != 3 {
		t.Fatalf("expected 3 comments, got %#v", comments)
	}
	if comments[2]["depth"] != 2 {
		t.Fatalf("expected nested depth 2, got %#v", comments[2]["depth"])
	}
}

func TestHackerNews_StoresEmptyCommentsOnHTTPStatusFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer server.Close()

	oldURL := algoliaItemURL
	algoliaItemURL = server.URL + "/%s"
	defer func() { algoliaItemURL = oldURL }()

	items := []types.FeedItem{
		types.FeedItem{GUID: "https://news.ycombinator.com/item?id=123"}.WithDefaults(),
	}
	got, err := Plugin{}.ProcessItems(context.Background(), items, config.PluginEntry{Name: "hacker-news"}, plugins.Context{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("ProcessItems: %v", err)
	}
	comments := got[0].Extra["comments"].([]map[string]any)
	if len(comments) != 0 {
		t.Fatalf("expected no comments on HTTP error, got %#v", comments)
	}
}
