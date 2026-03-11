package builtin

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/liuerfire/sieve/internal/config"
	httpx "github.com/liuerfire/sieve/internal/http"
	"github.com/liuerfire/sieve/internal/plugin"
	"github.com/liuerfire/sieve/internal/types"
)

type FetchContentPlugin struct {
	plugin.BaseWorkflowPlugin
}

var filterKeywords = []string{"category", "categories", "tag", "topic", "icon", "avatar"}

func shouldFilterImage(src string, isFirstImage bool) bool {
	if strings.HasPrefix(src, "data:") {
		return true
	}
	if !isFirstImage {
		return false
	}
	lower := strings.ToLower(src)
	for _, keyword := range filterKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func (FetchContentPlugin) ProcessItems(ctx context.Context, items []types.FeedItem, _ config.WorkflowPluginEntry, _ plugin.WorkflowContext) ([]types.FeedItem, error) {
	client := httpx.NewClient()
	result := make([]types.FeedItem, 0, len(items))
	for _, item := range items {
		if item.Level == types.LevelRejected || item.Link == "" {
			result = append(result, item)
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.Link, nil)
		if err != nil {
			item.Extra["content"] = ""
			item.Extra["images"] = []map[string]any{}
			result = append(result, item)
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			item.Extra["content"] = ""
			item.Extra["images"] = []map[string]any{}
			result = append(result, item)
			continue
		}
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			item.Extra["content"] = ""
			item.Extra["images"] = []map[string]any{}
			result = append(result, item)
			continue
		}

		doc.Find("script,style,nav,header,footer,aside").Each(func(_ int, s *goquery.Selection) {
			s.Remove()
		})
		main := doc.Find("article").First()
		if main.Length() == 0 {
			main = doc.Find("main").First()
		}
		if main.Length() == 0 {
			main = doc.Find("body").First()
		}

		images := make([]map[string]any, 0)
		imageIndex := 0
		main.Find("img").Each(func(idx int, sel *goquery.Selection) {
			src, _ := sel.Attr("src")
			if shouldFilterImage(src, idx == 0) {
				sel.Remove()
				return
			}
			image := map[string]any{
				"src": src,
				"alt": sel.AttrOr("alt", ""),
			}
			if width, ok := sel.Attr("width"); ok {
				image["width"] = width
			}
			if height, ok := sel.Attr("height"); ok {
				image["height"] = height
			}
			if title, ok := sel.Attr("title"); ok {
				image["title"] = title
			}
			images = append(images, image)
			sel.ReplaceWithHtml(fmt.Sprintf("[IMAGE_%d]", imageIndex))
			imageIndex++
		})

		content := strings.Join(strings.Fields(main.Text()), " ")
		if len(content) > 15000 {
			content = content[:15000]
		}
		item.Extra["content"] = content
		item.Extra["images"] = images
		result = append(result, item)
	}
	return result, nil
}

func init() {
	plugin.RegisterWorkflow("builtin/fetch-content", FetchContentPlugin{})
}
