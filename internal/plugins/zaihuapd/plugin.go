package zaihuapd

import (
	"context"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

type Plugin struct {
	plugins.BasePlugin
}

func (Plugin) ProcessItems(_ context.Context, items []types.FeedItem, _ config.PluginEntry, _ plugins.Context) ([]types.FeedItem, error) {
	result := make([]types.FeedItem, 0, len(items))
	for _, item := range items {
		item.Title = trimLeadingEmoji(item.Title)
		if item.Description == "" {
			result = append(result, item)
			continue
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(item.Description))
		if err != nil {
			result = append(result, item)
			continue
		}
		doc.Find("a").Each(func(_ int, sel *goquery.Selection) {
			href, _ := sel.Attr("href")
			if strings.Contains(href, "t.me/zaihuanews") {
				parent := sel.Parent()
				parent.SetHtml("")
			}
		})
		for {
			last := doc.Selection.Children().Last()
			if last.Length() == 0 || goquery.NodeName(last) != "img" {
				break
			}
			last.Remove()
		}
		html := ""
		doc.Selection.ChildrenFiltered("p,div").Each(func(_ int, sel *goquery.Selection) {
			if chunk, err := goquery.OuterHtml(sel); err == nil {
				html += chunk
			}
		})
		item.Description = html
		result = append(result, item)
	}
	return result, nil
}

func trimLeadingEmoji(s string) string {
	return strings.TrimLeftFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
	})
}

func init() {
	plugins.Register("zaihuapd", Plugin{})
}
