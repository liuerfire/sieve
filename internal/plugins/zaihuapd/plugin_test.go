package zaihuapd

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

func TestZaihuapd_CleansTrailingPromoAndEmojiPrefix(t *testing.T) {
	items := []types.FeedItem{
		types.FeedItem{
			Title:       "🍀🚀  Example title",
			Description: `<p>Hello</p><div><span>🍀</span><a href="https://t.me/zaihuanews">promo</a><br/></div><img src="tail.jpg"/>`,
		}.WithDefaults(),
	}
	got, err := Plugin{}.ProcessItems(context.Background(), items, config.PluginEntry{Name: "zaihuapd"}, plugins.Context{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("ProcessItems: %v", err)
	}
	if strings.HasPrefix(got[0].Title, "🍀") || strings.HasPrefix(got[0].Title, "🚀") {
		t.Fatalf("expected emoji prefix trimmed, got %q", got[0].Title)
	}
	if strings.Contains(got[0].Description, "t.me/zaihuanews") || strings.Contains(got[0].Description, "tail.jpg") {
		t.Fatalf("expected promo and trailing image removed, got %q", got[0].Description)
	}
}
