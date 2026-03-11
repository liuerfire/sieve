package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/types"
)

type WorkflowContext struct {
	SourceName    string
	SourceContext string
	IsDryRun      bool
	Logger        *slog.Logger
	LLM           func(tier string) any
}

type CollectResult struct {
	Title string
	Items []types.FeedItem
}

type WorkflowPlugin interface {
	Collect(ctx context.Context, entry config.WorkflowPluginEntry, runCtx WorkflowContext) (CollectResult, error)
	ProcessItems(ctx context.Context, items []types.FeedItem, entry config.WorkflowPluginEntry, runCtx WorkflowContext) ([]types.FeedItem, error)
	Report(ctx context.Context, items []types.FeedItem, entry config.WorkflowPluginEntry, runCtx WorkflowContext) error
}

type BaseWorkflowPlugin struct{}

func (BaseWorkflowPlugin) Collect(context.Context, config.WorkflowPluginEntry, WorkflowContext) (CollectResult, error) {
	return CollectResult{}, nil
}

func (BaseWorkflowPlugin) ProcessItems(_ context.Context, items []types.FeedItem, _ config.WorkflowPluginEntry, _ WorkflowContext) ([]types.FeedItem, error) {
	return items, nil
}

func (BaseWorkflowPlugin) Report(context.Context, []types.FeedItem, config.WorkflowPluginEntry, WorkflowContext) error {
	return nil
}

type LoadedWorkflowPlugin struct {
	Name   string
	Plugin WorkflowPlugin
	Entry  config.WorkflowPluginEntry
}

var (
	workflowRegistry   = map[string]WorkflowPlugin{}
	workflowRegistryMu sync.RWMutex
)

func RegisterWorkflow(name string, plugin WorkflowPlugin) {
	workflowRegistryMu.Lock()
	defer workflowRegistryMu.Unlock()
	workflowRegistry[name] = plugin
}

func LoadWorkflowPlugins(entries []config.WorkflowPluginEntry) ([]LoadedWorkflowPlugin, error) {
	workflowRegistryMu.RLock()
	defer workflowRegistryMu.RUnlock()

	loaded := make([]LoadedWorkflowPlugin, 0, len(entries))
	for _, entry := range entries {
		plugin, ok := workflowRegistry[entry.Name]
		if !ok {
			return nil, fmt.Errorf("workflow plugin %q not found", entry.Name)
		}
		loaded = append(loaded, LoadedWorkflowPlugin{
			Name:   entry.Name,
			Plugin: plugin,
			Entry:  entry,
		})
	}
	return loaded, nil
}

func ApplyProcessItems(ctx context.Context, items []types.FeedItem, loaded LoadedWorkflowPlugin, runCtx WorkflowContext) []types.FeedItem {
	nextItems, err := loaded.Plugin.ProcessItems(ctx, items, loaded.Entry, runCtx)
	if err != nil {
		if runCtx.Logger != nil {
			runCtx.Logger.Warn("process items failed", "plugin", loaded.Name, "error", err)
		}
		return items
	}
	return nextItems
}

func swapWorkflowRegistry(next map[string]WorkflowPlugin) func() {
	workflowRegistryMu.Lock()
	prev := workflowRegistry
	workflowRegistry = next
	workflowRegistryMu.Unlock()

	return func() {
		workflowRegistryMu.Lock()
		workflowRegistry = prev
		workflowRegistryMu.Unlock()
	}
}
