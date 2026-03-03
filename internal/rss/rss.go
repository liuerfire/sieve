// Package rss provides RSS feed fetching and parsing functionality.
package rss

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/liuerfire/sieve/internal/storage"
)

// fetchWithRetry fetches and parses an RSS feed with exponential backoff retry.
// It retries up to maxRetries times on transient errors (timeouts, temporary errors, 5xx).
// Context cancellation is respected during backoff delays.
func fetchWithRetry(ctx context.Context, url string, maxRetries int) (*gofeed.Feed, error) {
	var lastErr error

	for attempt := range maxRetries+1 {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		fp := gofeed.NewParser()
		feed, err := fp.ParseURLWithContext(url, ctx)
		if err == nil {
			return feed, nil
		}

		lastErr = err
		if !shouldRetryRSS(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("all retries failed: %w", lastErr)
}

// shouldRetryRSS determines if an error is retryable based on error message content.
// Returns true for timeouts, temporary errors, and 5xx server errors.
func shouldRetryRSS(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "temporary") ||
		strings.Contains(errStr, "5") // 5xx errors
}

// FetchItems fetches and parses an RSS feed, returning items with the given source name.
// The provided context controls cancellation and timeout of the HTTP request.
func FetchItems(ctx context.Context, url string, sourceName string) ([]*storage.Item, error) {
	feed, err := fetchWithRetry(ctx, url, 3)
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
