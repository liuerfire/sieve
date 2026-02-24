package rss

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchItems(t *testing.T) {
	// Mock RSS feed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8" ?>
<rss version="2.0">
<channel>
  <title>Test Feed</title>
  <item>
    <title>Test Item 1</title>
    <link>http://example.com/1</link>
    <description>Description 1</description>
    <pubDate>Mon, 24 Feb 2026 12:00:00 GMT</pubDate>
  </item>
</channel>
</rss>`)
	}))
	defer server.Close()

	items, err := FetchItems(server.URL, "test-source")
	if err != nil {
		t.Fatalf("failed to fetch items: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	if items[0].Title != "Test Item 1" {
		t.Errorf("expected title 'Test Item 1', got '%s'", items[0].Title)
	}
	if items[0].Source != "test-source" {
		t.Errorf("expected source 'test-source', got '%s'", items[0].Source)
	}
}
