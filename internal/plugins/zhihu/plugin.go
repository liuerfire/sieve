package zhihu

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/liuerfire/sieve/internal/config"
	httpx "github.com/liuerfire/sieve/internal/http"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

type Plugin struct {
	plugins.BasePlugin
}

var hotListURL = "https://api.zhihu.com/topstory/hot-lists/total"

type collectOptions struct {
	Limit int `json:"limit"`
}

func (Plugin) Collect(ctx context.Context, entry config.PluginEntry, _ plugins.Context) (plugins.CollectResult, error) {
	var opts collectOptions
	_ = json.Unmarshal(entry.Options, &opts)
	if opts.Limit == 0 {
		opts.Limit = 5
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hotListURL, nil)
	if err != nil {
		return plugins.CollectResult{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := httpx.NewClient().Do(req)
	if err != nil {
		return plugins.CollectResult{}, err
	}
	defer resp.Body.Close()

	var payload struct {
		Data []struct {
			CardID     string `json:"card_id"`
			DetailText string `json:"detail_text"`
			Target     struct {
				ID      any    `json:"id"`
				Title   string `json:"title"`
				URL     string `json:"url"`
				Excerpt string `json:"excerpt"`
				Created int64  `json:"created"`
			} `json:"target"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return plugins.CollectResult{}, err
	}
	if len(payload.Data) > opts.Limit {
		payload.Data = payload.Data[:opts.Limit]
	}
	items := make([]types.FeedItem, 0, len(payload.Data))
	for i, entry := range payload.Data {
		items = append(items, types.FeedItem{
			Title:       entry.Target.Title,
			Link:        toQuestionURL(entry.Target.URL),
			PubDate:     time.Unix(entry.Target.Created, 0).UTC().Format(time.RFC3339),
			Description: entry.Target.Excerpt,
			GUID:        entry.CardID,
			Extra: map[string]any{
				"hotness": entry.DetailText,
				"rank":    i + 1,
				"id":      stringify(entry.Target.ID),
			},
		}.WithDefaults())
	}
	return plugins.CollectResult{Title: "知乎热榜", Items: items}, nil
}

func toQuestionURL(apiURL string) string {
	return strings.Replace(apiURL, "https://api.zhihu.com/questions/", "https://www.zhihu.com/question/", 1)
}

func stringify(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	default:
		return ""
	}
}

func HotListURLForTest(next string) func() {
	prev := hotListURL
	hotListURL = next
	return func() {
		hotListURL = prev
	}
}

func init() {
	plugins.Register("zhihu", Plugin{})
}
