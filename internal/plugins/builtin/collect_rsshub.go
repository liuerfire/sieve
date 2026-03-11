package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/liuerfire/sieve/internal/config"
	httpx "github.com/liuerfire/sieve/internal/http"
	"github.com/liuerfire/sieve/internal/plugin"
	"github.com/liuerfire/sieve/internal/types"
)

type CollectRSSHubPlugin struct {
	plugin.BaseWorkflowPlugin
}

type collectRSSHubOptions struct {
	Route    string `json:"route"`
	MaxItems int    `json:"maxItems"`
}

func (CollectRSSHubPlugin) Collect(ctx context.Context, entry config.WorkflowPluginEntry, runCtx plugin.WorkflowContext) (plugin.CollectResult, error) {
	var opts collectRSSHubOptions
	if err := json.Unmarshal(entry.Options, &opts); err != nil {
		return plugin.CollectResult{}, err
	}
	if opts.Route == "" {
		return plugin.CollectResult{}, fmt.Errorf("collect-rsshub: route is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.Route, nil)
	if err != nil {
		return plugin.CollectResult{}, err
	}
	resp, err := httpx.NewClient().Do(req)
	if err != nil {
		return plugin.CollectResult{}, err
	}
	defer resp.Body.Close()

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
		return plugin.CollectResult{}, err
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

	return plugin.CollectResult{Title: payload.Title, Items: items}, nil
}

func init() {
	plugin.RegisterWorkflow("builtin/collect-rsshub", CollectRSSHubPlugin{})
}
