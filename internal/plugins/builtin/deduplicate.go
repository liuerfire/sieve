package builtin

import (
	"context"
	"path/filepath"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/storage"
	"github.com/liuerfire/sieve/internal/types"
)

type DeduplicatePlugin struct {
	plugins.BasePlugin
}

func (DeduplicatePlugin) ProcessItems(_ context.Context, items []types.FeedItem, _ config.PluginEntry, runCtx plugins.Context) ([]types.FeedItem, error) {
	if runCtx.IsDryRun {
		return items, nil
	}

	tracker, err := storage.NewGUIDTracker(filepath.Join("output", runCtx.SourceName+"-processed.json"))
	if err != nil {
		return nil, err
	}

	newGuids := make(map[string]struct{}, len(items))
	for _, item := range items {
		if !tracker.IsProcessed(item.GUID) {
			newGuids[item.GUID] = struct{}{}
		}
	}

	processed := make([]string, 0, len(newGuids))
	for guid := range newGuids {
		processed = append(processed, guid)
	}
	tracker.MarkProcessed(processed)
	tracker.Cleanup()
	if err := tracker.Persist(); err != nil {
		return nil, err
	}

	result := make([]types.FeedItem, 0, len(items))
	for _, item := range items {
		if _, ok := newGuids[item.GUID]; ok {
			result = append(result, item)
			continue
		}
		item.Level = types.LevelRejected
		result = append(result, item)
	}
	return result, nil
}

func init() {
	plugins.Register("builtin/deduplicate", DeduplicatePlugin{})
}
