package cnbeta

import (
	"context"
	"net/http"

	"github.com/PuerkitoBio/goquery"

	"github.com/liuerfire/sieve/internal/config"
	httpx "github.com/liuerfire/sieve/internal/http"
	"github.com/liuerfire/sieve/internal/plugin"
	"github.com/liuerfire/sieve/internal/types"
)

type Plugin struct {
	plugin.BaseWorkflowPlugin
}

func (Plugin) ProcessItems(ctx context.Context, items []types.FeedItem, _ config.WorkflowPluginEntry, _ plugin.WorkflowContext) ([]types.FeedItem, error) {
	client := httpx.NewClient()
	result := make([]types.FeedItem, 0, len(items))
	for _, item := range items {
		if item.Level == types.LevelRejected || item.Link == "" {
			result = append(result, item)
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.Link, nil)
		if err != nil {
			result = append(result, item)
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			result = append(result, item)
			continue
		}
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			result = append(result, item)
			continue
		}
		summary := doc.Find(".article-summary").First()
		summary.Find(".topic").Remove()
		content := doc.Find(".article-content").First()
		parts := ""
		if summary.Length() > 0 {
			if html, err := summary.Html(); err == nil {
				parts += html
			}
		}
		if content.Length() > 0 {
			if html, err := content.Html(); err == nil {
				parts += html
			}
		}
		if parts != "" {
			item.Description = parts
		}
		result = append(result, item)
	}
	return result, nil
}

func init() {
	plugin.RegisterWorkflow("cnbeta", Plugin{})
}
