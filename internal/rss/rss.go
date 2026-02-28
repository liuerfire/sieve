// Package rss provides RSS feed fetching and parsing functionality.
package rss

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/liuerfire/sieve/internal/storage"
)

// FetchItems fetches and parses an RSS feed, returning items with the given source name.
// The provided context controls cancellation and timeout of the HTTP request.
func FetchItems(ctx context.Context, url string, sourceName string) ([]*storage.Item, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURLWithContext(url, ctx)
	if err != nil {
		return nil, err
	}

	var items []*storage.Item
	for _, entry := range feed.Items {
		item := &storage.Item{
			ID:          generateID(sourceName, entry.Link),
			Source:      sourceName,
			Title:       entry.Title,
			Link:        entry.Link,
			Description: entry.Description,
			Content:     entry.Content,
		}

		if entry.PublishedParsed != nil {
			item.PublishedAt = *entry.PublishedParsed
		} else if entry.UpdatedParsed != nil {
			item.PublishedAt = *entry.UpdatedParsed
		} else {
			item.PublishedAt = time.Now()
		}

		items = append(items, item)
	}

	return items, nil
}

func generateID(source, link string) string {
	// Use SHA-256 with source+link to prevent collisions across different sources
	h := sha256.New()
	h.Write([]byte(source + "|" + link))
	return fmt.Sprintf("%x", h.Sum(nil))
}
