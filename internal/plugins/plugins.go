package plugins

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/llm"
	"github.com/liuerfire/sieve/internal/types"
)

type Context struct {
	SourceName    string
	SourceContext string
	IsDryRun      bool
	Logger        *slog.Logger
	LLM           func(tier string) (llm.Provider, error)
}

type CollectResult struct {
	Title string
	Items []types.FeedItem
}

type Plugin interface {
	Collect(ctx context.Context, entry config.PluginEntry, runCtx Context) (CollectResult, error)
	ProcessItems(ctx context.Context, items []types.FeedItem, entry config.PluginEntry, runCtx Context) ([]types.FeedItem, error)
	Report(ctx context.Context, items []types.FeedItem, entry config.PluginEntry, runCtx Context) error
}

type BasePlugin struct{}

func (BasePlugin) Collect(context.Context, config.PluginEntry, Context) (CollectResult, error) {
	return CollectResult{}, nil
}

func (BasePlugin) ProcessItems(_ context.Context, items []types.FeedItem, _ config.PluginEntry, _ Context) ([]types.FeedItem, error) {
	return items, nil
}

func (BasePlugin) Report(context.Context, []types.FeedItem, config.PluginEntry, Context) error {
	return nil
}

type LoadedPlugin struct {
	Name   string
	Plugin Plugin
	Entry  config.PluginEntry
}

var (
	registry   = map[string]Plugin{}
	registryMu sync.RWMutex
)

func Register(name string, plugin Plugin) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = plugin
}

func Load(entries []config.PluginEntry) ([]LoadedPlugin, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	loaded := make([]LoadedPlugin, 0, len(entries))
	for _, entry := range entries {
		plugin, ok := registry[entry.Name]
		if !ok {
			return nil, fmt.Errorf("plugin %q not found", entry.Name)
		}
		loaded = append(loaded, LoadedPlugin{
			Name:   entry.Name,
			Plugin: plugin,
			Entry:  entry,
		})
	}
	return loaded, nil
}

func ApplyProcessItems(ctx context.Context, items []types.FeedItem, loaded LoadedPlugin, runCtx Context) ([]types.FeedItem, error) {
	nextItems, err := loaded.Plugin.ProcessItems(ctx, items, loaded.Entry, runCtx)
	if err != nil {
		if runCtx.Logger != nil {
			runCtx.Logger.Warn("process items failed", "plugin", loaded.Name, "error", err)
		}
		return items, err
	}
	return nextItems, nil
}

func swapRegistry(next map[string]Plugin) func() {
	registryMu.Lock()
	prev := registry
	registry = next
	registryMu.Unlock()

	return func() {
		registryMu.Lock()
		registry = prev
		registryMu.Unlock()
	}
}
