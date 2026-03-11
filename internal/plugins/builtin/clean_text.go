package builtin

import (
	"context"
	"strings"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/plugin"
	"github.com/liuerfire/sieve/internal/types"
)

type CleanTextPlugin struct {
	plugin.BaseWorkflowPlugin
}

func (CleanTextPlugin) ProcessItems(_ context.Context, items []types.FeedItem, _ config.WorkflowPluginEntry, _ plugin.WorkflowContext) ([]types.FeedItem, error) {
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
	plugin.RegisterWorkflow("builtin/clean-text", CleanTextPlugin{})
}
