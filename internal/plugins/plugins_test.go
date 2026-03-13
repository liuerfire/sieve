package plugins

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
	process func(context.Context, []types.FeedItem, config.PluginEntry, Context) ([]types.FeedItem, error)
}

func (p workflowMockPlugin) Collect(_ context.Context, _ config.PluginEntry, _ Context) (CollectResult, error) {
	return CollectResult{}, nil
}

func (p workflowMockPlugin) ProcessItems(ctx context.Context, items []types.FeedItem, entry config.PluginEntry, runCtx Context) ([]types.FeedItem, error) {
	if p.process == nil {
		return items, nil
	}
	return p.process(ctx, items, entry, runCtx)
}

func (p workflowMockPlugin) Report(_ context.Context, _ []types.FeedItem, _ config.PluginEntry, _ Context) error {
	return nil
}

func TestRegistry_LoadPluginsByName(t *testing.T) {
	name := "test/load"
	restore := swapRegistry(map[string]Plugin{
		name: workflowMockPlugin{},
	})
	defer restore()

	loaded, err := Load([]config.PluginEntry{{Name: name}})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(loaded))
	}
	if loaded[0].Name != name {
		t.Fatalf("expected plugin %q, got %q", name, loaded[0].Name)
	}
}

func TestRegistry_LoadPlugins_UnknownPluginFails(t *testing.T) {
	restore := swapRegistry(map[string]Plugin{})
	defer restore()

	_, err := Load([]config.PluginEntry{{Name: "missing"}})
	if err == nil {
		t.Fatal("expected unknown plugin to fail")
	}
}

func TestApplyProcessItems_OnErrorReturnsOriginalItems(t *testing.T) {
	loaded := LoadedPlugin{
		Name: "test/fail",
		Plugin: workflowMockPlugin{
			process: func(_ context.Context, _ []types.FeedItem, _ config.PluginEntry, _ Context) ([]types.FeedItem, error) {
				return nil, errors.New("boom")
			},
		},
		Entry: config.PluginEntry{Name: "test/fail"},
	}

	items := []types.FeedItem{{Title: "kept"}}
	got, err := ApplyProcessItems(
		context.Background(),
		items,
		loaded,
		Context{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	)
	if err == nil {
		t.Fatal("expected process error")
	}

	if len(got) != 1 || got[0].Title != "kept" {
		t.Fatalf("expected original items to be returned, got %#v", got)
	}
}
