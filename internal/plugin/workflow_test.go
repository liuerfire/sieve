package plugin

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/types"
)

type workflowMockPlugin struct {
	process func(context.Context, []types.FeedItem, config.WorkflowPluginEntry, WorkflowContext) ([]types.FeedItem, error)
}

func (p workflowMockPlugin) Collect(_ context.Context, _ config.WorkflowPluginEntry, _ WorkflowContext) (CollectResult, error) {
	return CollectResult{}, nil
}

func (p workflowMockPlugin) ProcessItems(ctx context.Context, items []types.FeedItem, entry config.WorkflowPluginEntry, runCtx WorkflowContext) ([]types.FeedItem, error) {
	if p.process == nil {
		return items, nil
	}
	return p.process(ctx, items, entry, runCtx)
}

func (p workflowMockPlugin) Report(_ context.Context, _ []types.FeedItem, _ config.WorkflowPluginEntry, _ WorkflowContext) error {
	return nil
}

func TestRegistry_LoadPluginsByName(t *testing.T) {
	name := "test/load"
	restore := swapWorkflowRegistry(map[string]WorkflowPlugin{
		name: workflowMockPlugin{},
	})
	defer restore()

	loaded, err := LoadWorkflowPlugins([]config.WorkflowPluginEntry{{Name: name}})
	if err != nil {
		t.Fatalf("LoadWorkflowPlugins returned error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(loaded))
	}
	if loaded[0].Name != name {
		t.Fatalf("expected plugin %q, got %q", name, loaded[0].Name)
	}
}

func TestRegistry_LoadPlugins_UnknownPluginFails(t *testing.T) {
	restore := swapWorkflowRegistry(map[string]WorkflowPlugin{})
	defer restore()

	_, err := LoadWorkflowPlugins([]config.WorkflowPluginEntry{{Name: "missing"}})
	if err == nil {
		t.Fatal("expected unknown plugin to fail")
	}
}

func TestApplyProcessItems_OnErrorReturnsOriginalItems(t *testing.T) {
	loaded := LoadedWorkflowPlugin{
		Name: "test/fail",
		Plugin: workflowMockPlugin{
			process: func(_ context.Context, _ []types.FeedItem, _ config.WorkflowPluginEntry, _ WorkflowContext) ([]types.FeedItem, error) {
				return nil, errors.New("boom")
			},
		},
		Entry: config.WorkflowPluginEntry{Name: "test/fail"},
	}

	items := []types.FeedItem{{Title: "kept"}}
	got := ApplyProcessItems(
		context.Background(),
		items,
		loaded,
		WorkflowContext{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	)

	if len(got) != 1 || got[0].Title != "kept" {
		t.Fatalf("expected original items to be returned, got %#v", got)
	}
}
