package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mmcdole/gofeed"

	"github.com/liuerfire/sieve/internal/config"
	httpx "github.com/liuerfire/sieve/internal/http"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

type CollectRSSPlugin struct {
	plugins.BasePlugin
}

type collectRSSOptions struct {
	URL      string `json:"url"`
	MaxItems int    `json:"maxItems"`
}

func (CollectRSSPlugin) Collect(ctx context.Context, entry config.PluginEntry, runCtx plugins.Context) (plugins.CollectResult, error) {
	var opts collectRSSOptions
	if err := json.Unmarshal(entry.Options, &opts); err != nil {
		return plugins.CollectResult{}, err
	}
	if opts.URL == "" {
		return plugins.CollectResult{}, fmt.Errorf("collect-rss: url is required")
	}

	req, err := gofeed.NewParser().ParseURLWithContext(opts.URL, ctx)
	if err != nil {
		return plugins.CollectResult{}, err
	}

	items := make([]types.FeedItem, 0, len(req.Items))
	for _, item := range req.Items {
		guid := item.GUID
		if guid == "" {
			guid = item.Link
		}
		items = append(items, types.FeedItem{
			Title:       item.Title,
			Link:        item.Link,
			PubDate:     item.Published,
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

	return plugins.CollectResult{
		Title: req.Title,
		Items: items,
	}, nil
}

func init() {
	plugins.Register("builtin/collect-rss", CollectRSSPlugin{})
	_ = httpx.DefaultUserAgent
}
