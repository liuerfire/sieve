package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/liuerfire/sieve/internal/config"
	httpx "github.com/liuerfire/sieve/internal/http"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

type CollectRSSHubPlugin struct {
	plugins.BasePlugin
}

type collectRSSHubOptions struct {
	Route    string `json:"route"`
	MaxItems int    `json:"maxItems"`
}

func (CollectRSSHubPlugin) Collect(ctx context.Context, entry config.PluginEntry, runCtx plugins.Context) (plugins.CollectResult, error) {
	var opts collectRSSHubOptions
	if err := json.Unmarshal(entry.Options, &opts); err != nil {
		return plugins.CollectResult{}, err
	}
	if opts.Route == "" {
		return plugins.CollectResult{}, fmt.Errorf("collect-rsshub: route is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.Route, nil)
	if err != nil {
		return plugins.CollectResult{}, err
	}
	resp, err := httpx.NewClient().Do(req)
	if err != nil {
		return plugins.CollectResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return plugins.CollectResult{}, fmt.Errorf("collect-rsshub: unexpected status %d", resp.StatusCode)
	}

	var payload struct {
		Title string `json:"title"`
		Item  []struct {
			Title       string `json:"title"`
			Link        string `json:"link"`
			PubDate     string `json:"pubDate"`
			Description string `json:"description"`
			GUID        struct {
				Value string `json:"value"`
			} `json:"guid"`
		} `json:"item"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return plugins.CollectResult{}, err
	}

	items := make([]types.FeedItem, 0, len(payload.Item))
	for _, item := range payload.Item {
		guid := item.GUID.Value
		if guid == "" {
			guid = item.Link
		}
		items = append(items, types.FeedItem{
			Title:       item.Title,
			Link:        item.Link,
			PubDate:     item.PubDate,
			Description: item.Description,
			GUID:        guid,
		}.WithDefaults())
	}
	if opts.MaxItems > 0 && len(items) > opts.MaxItems {
		items = items[:opts.MaxItems]
	}
	if runCtx.IsDryRun && len(items) > 3 {
		items = items[:3]
	}

	return plugins.CollectResult{Title: payload.Title, Items: items}, nil
}

func init() {
	plugins.Register("builtin/collect-rsshub", CollectRSSHubPlugin{})
}
