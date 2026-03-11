package plugins

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/liuerfire/sieve/internal/config"
	_ "github.com/liuerfire/sieve/internal/plugins/cnbeta"
	_ "github.com/liuerfire/sieve/internal/plugins/producthunt"
	producthunt "github.com/liuerfire/sieve/internal/plugins/producthunt"
	"github.com/liuerfire/sieve/internal/plugins/zhihu"
	"github.com/liuerfire/sieve/internal/plugin"
	"github.com/liuerfire/sieve/internal/types"
)

func TestCNBeta_RewritesDescriptionFromArticlePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<div class="article-summary"><span class="topic">Topic</span><p>Summary</p></div><div class="article-content"><p>Content</p></div>`)
	}))
	defer server.Close()

	loaded, err := plugin.LoadWorkflowPlugins([]config.WorkflowPluginEntry{{Name: "cnbeta"}})
	if err != nil {
		t.Fatalf("LoadWorkflowPlugins: %v", err)
	}
	items := []types.FeedItem{types.FeedItem{Link: server.URL}.WithDefaults()}
	got, err := loaded[0].Plugin.ProcessItems(context.Background(), items, loaded[0].Entry, plugin.WorkflowContext{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("ProcessItems: %v", err)
	}
	if got[0].Description == "" {
		t.Fatal("expected description to be rewritten")
	}
}

func TestProductHunt_CollectsRankVotesAndTopics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"data":{"posts":{"edges":[{"node":{"id":"1","name":"Prod","tagline":"Tag","description":"Desc","url":"https://producthunt.com/posts/prod","website":"https://example.com?ref=ph","votesCount":42,"createdAt":"2026-03-11T00:00:00Z","topics":{"edges":[{"node":{"name":"AI"}}]}}}]}}}`)
	}))
	defer server.Close()

	oldURL := producthunt.GraphqlURLForTest(server.URL)
	defer oldURL()
	_ = os.Setenv("PRODUCTHUNT_API_KEY", "test")
	defer os.Unsetenv("PRODUCTHUNT_API_KEY")

	loaded, err := plugin.LoadWorkflowPlugins([]config.WorkflowPluginEntry{{Name: "producthunt"}})
	if err != nil {
		t.Fatalf("LoadWorkflowPlugins: %v", err)
	}
	result, err := loaded[0].Plugin.Collect(context.Background(), loaded[0].Entry, plugin.WorkflowContext{})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Extra["votesCount"] != 42 {
		t.Fatalf("unexpected producthunt items: %#v", result.Items)
	}
}

func TestZhihu_CollectsHotListItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"data":[{"card_id":"card-1","detail_text":"hot","target":{"id":1,"title":"Title","url":"https://api.zhihu.com/questions/123","excerpt":"Excerpt","created":1710000000}}]}`)
	}))
	defer server.Close()

	oldURL := zhihu.HotListURLForTest(server.URL)
	defer oldURL()

	loaded, err := plugin.LoadWorkflowPlugins([]config.WorkflowPluginEntry{{Name: "zhihu"}})
	if err != nil {
		t.Fatalf("LoadWorkflowPlugins: %v", err)
	}
	result, err := loaded[0].Plugin.Collect(context.Background(), loaded[0].Entry, plugin.WorkflowContext{})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if result.Items[0].Link != "https://www.zhihu.com/question/123" {
		t.Fatalf("unexpected zhihu link: %#v", result.Items[0].Link)
	}
}
