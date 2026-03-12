package producthunt

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/plugins"
)

func TestCollect_UsesLosAngelesMidnightDuringDST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		if string(body) == "" {
			t.Fatal("expected graphql request body")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected auth header %q", got)
		}
		if want := `"postedAfter":"2026-03-12T07:00:00Z"`; !strings.Contains(string(body), want) {
			t.Fatalf("expected request body to contain %s, got %s", want, string(body))
		}
		_, _ = io.WriteString(w, `{"data":{"posts":{"edges":[]}}}`)
	}))
	defer server.Close()

	restoreURL := GraphqlURLForTest(server.URL)
	defer restoreURL()

	restoreNow := swapNow(func() time.Time {
		return time.Date(2026, 3, 12, 9, 30, 0, 0, time.UTC)
	})
	defer restoreNow()

	t.Setenv("PRODUCTHUNT_API_KEY", "test-token")

	_, err := Plugin{}.Collect(context.Background(), config.PluginEntry{Name: "producthunt"}, plugins.Context{})
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
}

func TestCollect_UsesLosAngelesMidnightOutsideDST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		if want := `"postedAfter":"2026-01-12T08:00:00Z"`; !strings.Contains(string(body), want) {
			t.Fatalf("expected request body to contain %s, got %s", want, string(body))
		}
		_, _ = io.WriteString(w, `{"data":{"posts":{"edges":[]}}}`)
	}))
	defer server.Close()

	restoreURL := GraphqlURLForTest(server.URL)
	defer restoreURL()

	restoreNow := swapNow(func() time.Time {
		return time.Date(2026, 1, 12, 9, 30, 0, 0, time.UTC)
	})
	defer restoreNow()

	t.Setenv("PRODUCTHUNT_API_KEY", "test-token")

	_, err := Plugin{}.Collect(context.Background(), config.PluginEntry{Name: "producthunt"}, plugins.Context{})
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
}
