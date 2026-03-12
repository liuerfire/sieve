package builtin

import (
	"context"
	"strings"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

type CleanTextPlugin struct {
	plugins.BasePlugin
}

func (CleanTextPlugin) ProcessItems(_ context.Context, items []types.FeedItem, _ config.PluginEntry, _ plugins.Context) ([]types.FeedItem, error) {
	result := make([]types.FeedItem, 0, len(items))
	for _, item := range items {
		item.Title = cleanZeroWidth(item.Title)
		item.Description = cleanZeroWidth(item.Description)
		result = append(result, item)
	}
	return result, nil
}

func cleanZeroWidth(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\u200b")
	value = strings.TrimSpace(value)
	return value
}

func init() {
	plugins.Register("builtin/clean-text", CleanTextPlugin{})
}
