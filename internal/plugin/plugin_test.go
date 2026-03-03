package plugin

import (
	"context"
	"testing"

	"github.com/liuerfire/sieve/internal/storage"
)

type mockPlugin struct{}

func (p *mockPlugin) Execute(ctx context.Context, item *storage.Item) (*storage.Item, error) {
	item.Content = "Mock Content"
	return item, nil
}

func TestPluginRegistry(t *testing.T) {
	Register("mock", &mockPlugin{})

	p, err := Get("mock")
	if err != nil {
		t.Fatalf("failed to get plugin: %v", err)
	}

	item := &storage.Item{Title: "Test"}
	updatedItem, err := p.Execute(context.Background(), item)
	if err != nil {
		t.Fatalf("plugin execution failed: %v", err)
	}

	if updatedItem.Content != "Mock Content" {
		t.Errorf("expected content 'Mock Content', got '%s'", updatedItem.Content)
	}

	_, err = Get("non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent plugin")
	}
}
