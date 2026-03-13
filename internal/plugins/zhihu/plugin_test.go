package zhihu

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/plugins"
)

func TestCollect_ReturnsErrorOnHTTPStatusFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "blocked", http.StatusForbidden)
	}))
	defer server.Close()

	restoreURL := HotListURLForTest(server.URL)
	defer restoreURL()

	_, err := Plugin{}.Collect(context.Background(), config.PluginEntry{Name: "zhihu"}, plugins.Context{})
	if err == nil {
		t.Fatal("expected HTTP status failure to return an error")
	}
}
