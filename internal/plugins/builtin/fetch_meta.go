package builtin

import (
	"context"
	"net/http"

	"github.com/PuerkitoBio/goquery"

	"github.com/liuerfire/sieve/internal/config"
	httpx "github.com/liuerfire/sieve/internal/http"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

type FetchMetaPlugin struct {
	plugins.BasePlugin
}

func (FetchMetaPlugin) ProcessItems(ctx context.Context, items []types.FeedItem, _ config.PluginEntry, _ plugins.Context) ([]types.FeedItem, error) {
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
				if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
					_ = resp.Body.Close()
					item.Extra["meta"] = ""
					result = append(result, item)
					continue
				}
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
	plugins.Register("builtin/fetch-meta", FetchMetaPlugin{})
}
