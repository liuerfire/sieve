package hacker_news

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/liuerfire/sieve/internal/config"
	httpx "github.com/liuerfire/sieve/internal/http"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

type Plugin struct {
	plugins.BasePlugin
}

var algoliaItemURL = "https://hn.algolia.com/api/v1/items/%s"

var itemIDPattern = regexp.MustCompile(`id=(\d+)`)

func (Plugin) ProcessItems(ctx context.Context, items []types.FeedItem, _ config.PluginEntry, runCtx plugins.Context) ([]types.FeedItem, error) {
	client := httpx.NewClient()
	result := make([]types.FeedItem, 0, len(items))
	for _, item := range items {
		if item.Level == types.LevelRejected {
			result = append(result, item)
			continue
		}
		match := itemIDPattern.FindStringSubmatch(item.GUID)
		if !strings.Contains(item.GUID, "news.ycombinator.com") || len(match) < 2 {
			result = append(result, item)
			continue
		}
		itemID := match[1]
		item.Link = item.GUID
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(algoliaItemURL, itemID), nil)
		if err != nil {
			item.Extra["comments"] = []map[string]any{}
			result = append(result, item)
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			item.Extra["comments"] = []map[string]any{}
			result = append(result, item)
			continue
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			_ = resp.Body.Close()
			item.Extra["comments"] = []map[string]any{}
			result = append(result, item)
			continue
		}
		var payload struct {
			Children []hnChild `json:"children"`
		}
		err = json.NewDecoder(resp.Body).Decode(&payload)
		_ = resp.Body.Close()
		if err != nil {
			item.Extra["comments"] = []map[string]any{}
			result = append(result, item)
			continue
		}
		item.Extra["comments"] = extractComments(payload.Children, 0, 2)
		result = append(result, item)
	}
	_ = runCtx
	return result, nil
}

type hnChild struct {
	Author   string    `json:"author"`
	Text     string    `json:"text"`
	Points   *int      `json:"points"`
	Children []hnChild `json:"children"`
}

func extractComments(children []hnChild, depth int, maxDepth int) []map[string]any {
	comments := make([]map[string]any, 0)
	limit := 2
	if depth == 0 {
		limit = 10
	}
	for i, child := range children {
		if i >= limit {
			break
		}
		if child.Text != "" {
			points := 0
			if child.Points != nil {
				points = *child.Points
			}
			comments = append(comments, map[string]any{
				"author": child.Author,
				"text":   child.Text,
				"points": points,
				"depth":  depth,
			})
		}
		if depth < maxDepth && len(child.Children) > 0 {
			comments = append(comments, extractComments(child.Children, depth+1, maxDepth)...)
		}
	}
	return comments
}

func init() {
	plugins.Register("hacker-news", Plugin{})
}
