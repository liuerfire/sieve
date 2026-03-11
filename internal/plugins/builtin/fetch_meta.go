package builtin

import (
	"context"
	"net/http"

	"github.com/PuerkitoBio/goquery"

	"github.com/liuerfire/sieve/internal/config"
	httpx "github.com/liuerfire/sieve/internal/http"
	"github.com/liuerfire/sieve/internal/plugin"
	"github.com/liuerfire/sieve/internal/types"
)

type FetchMetaPlugin struct {
	plugin.BaseWorkflowPlugin
}

func (FetchMetaPlugin) ProcessItems(ctx context.Context, items []types.FeedItem, _ config.WorkflowPluginEntry, _ plugin.WorkflowContext) ([]types.FeedItem, error) {
	client := httpx.NewClient()
	result := make([]types.FeedItem, 0, len(items))
	for _, item := range items {
		if item.Level == types.LevelRejected || item.Link == "" {
			result = append(result, item)
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.Link, nil)
		if err == nil {
			resp, err := client.Do(req)
			if err == nil {
				doc, err := goquery.NewDocumentFromReader(resp.Body)
				_ = resp.Body.Close()
				if err == nil {
					meta := doc.Find(`meta[name="description"]`).AttrOr("content", "")
					if meta == "" {
						meta = doc.Find(`meta[property="og:description"]`).AttrOr("content", "")
					}
					item.Extra["meta"] = meta
					result = append(result, item)
					continue
				}
			}
		}
		item.Extra["meta"] = ""
		result = append(result, item)
	}
	return result, nil
}

func init() {
	plugin.RegisterWorkflow("builtin/fetch-meta", FetchMetaPlugin{})
}
