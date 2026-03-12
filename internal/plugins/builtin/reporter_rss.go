package builtin

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

type ReporterRSSPlugin struct {
	plugins.BasePlugin
}

type reporterRSSOptions struct {
	OutputPath string `json:"outputPath"`
	SourceName string `json:"sourceName,omitempty"`
	Title      string `json:"title,omitempty"`
	ShowReason *bool  `json:"showReason,omitempty"`
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr,omitempty"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Description   string    `xml:"description,omitempty"`
	LastBuildDate string    `xml:"lastBuildDate,omitempty"`
	Items         []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	Description string  `xml:"description"`
	GUID        rssGUID `xml:"guid"`
	PubDate     string  `xml:"pubDate"`
}

type rssGUID struct {
	IsPermaLink bool   `xml:"isPermaLink,attr,omitempty"`
	Value       string `xml:",chardata"`
}

func FormatRSSItems(items []types.FeedItem, showReason bool) []rssItem {
	result := make([]rssItem, 0, len(items))
	for _, item := range items {
		if item.Level == types.LevelRejected {
			continue
		}
		title := item.Title
		switch item.Level {
		case types.LevelCritical:
			title = "⭐⭐ " + title
		case types.LevelRecommended:
			title = "⭐ " + title
		}
		note := ""
		if showReason {
			note = fmt.Sprintf(`<p><small style="opacity: 0.7;">[%s] %s</small></p>`, item.Level, item.Reason)
		}
		guid := item.GUID
		if guid == "" {
			guid = item.Link
		}
		result = append(result, rssItem{
			Title:       title,
			Link:        item.Link,
			Description: note + item.Description,
			GUID:        rssGUID{Value: guid},
			PubDate:     item.PubDate,
		})
	}
	return result
}

func (ReporterRSSPlugin) Report(_ context.Context, items []types.FeedItem, entry config.PluginEntry, runCtx plugins.Context) error {
	var opts reporterRSSOptions
	if err := json.Unmarshal(entry.Options, &opts); err != nil {
		return err
	}
	if opts.OutputPath == "" {
		return fmt.Errorf("reporter-rss: outputPath is required")
	}
	showReason := true
	if opts.ShowReason != nil {
		showReason = *opts.ShowReason
	}

	existing, err := readRSS(opts.OutputPath)
	if err != nil {
		return err
	}
	formatted := FormatRSSItems(items, showReason)
	allItems := append(formatted, existing.Channel.Items...)
	if len(allItems) > 50 {
		allItems = allItems[:50]
	}

	if runCtx.IsDryRun {
		return nil
	}

	if err := writeRSS(opts.OutputPath, rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:         opts.Title,
			Description:   fmt.Sprintf("Filtered content for %s", opts.SourceName),
			LastBuildDate: "",
			Items:         allItems,
		},
	}); err != nil {
		return err
	}
	if runCtx.Logger != nil {
		runCtx.Logger.Info("wrote rss output", "source", runCtx.SourceName, "path", opts.OutputPath, "items", len(allItems))
	}
	return nil
}

func readRSS(path string) (rssFeed, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return rssFeed{}, nil
		}
		return rssFeed{}, err
	}
	var feed rssFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return rssFeed{}, err
	}
	return feed, nil
}

func writeRSS(path string, feed rssFeed) error {
	data, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return err
	}
	data = append([]byte(xml.Header), data...)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func init() {
	plugins.Register("builtin/reporter-rss", ReporterRSSPlugin{})
}
