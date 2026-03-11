package workflow

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/plugin"
	"github.com/liuerfire/sieve/internal/types"
)

type recorderPlugin struct {
	collectResult plugin.CollectResult
	events        *[]string
	reportTitle   *string
}

func (p recorderPlugin) Collect(_ context.Context, entry config.WorkflowPluginEntry, _ plugin.WorkflowContext) (plugin.CollectResult, error) {
	if p.events != nil {
		*p.events = append(*p.events, "collect:"+entry.Name)
	}
	return p.collectResult, nil
}

func (p recorderPlugin) ProcessItems(_ context.Context, items []types.FeedItem, entry config.WorkflowPluginEntry, _ plugin.WorkflowContext) ([]types.FeedItem, error) {
	if p.events != nil {
		*p.events = append(*p.events, "process:"+entry.Name)
	}
	return append(items, types.FeedItem{Title: entry.Name}.WithDefaults()), nil
}

func (p recorderPlugin) Report(_ context.Context, _ []types.FeedItem, entry config.WorkflowPluginEntry, _ plugin.WorkflowContext) error {
	if p.events != nil {
		*p.events = append(*p.events, "report:"+entry.Name)
	}
	if p.reportTitle != nil {
		var payload struct {
			Title string `json:"title"`
		}
		if err := json.Unmarshal(entry.Options, &payload); err != nil {
			return err
		}
		*p.reportTitle = payload.Title
	}
	return nil
}

func TestRunWorkflow_RunsCollectPrefixSourceAndReportInOrder(t *testing.T) {
	var events []string

	plugin.RegisterWorkflow("builtin/deduplicate", recorderPlugin{events: &events})
	plugin.RegisterWorkflow("builtin/clean-text", recorderPlugin{events: &events})
	plugin.RegisterWorkflow("source/test", recorderPlugin{
		events: &events,
		collectResult: plugin.CollectResult{
			Title: "Collected Title",
			Items: []types.FeedItem{types.FeedItem{Title: "first"}.WithDefaults()},
		},
	})

	err := Run(context.Background(), Params{
		SourceName: "hacker-news",
		SourceConfig: config.WorkflowSourceConfig{
			Name:    "hacker-news",
			Context: "context",
			Plugins: []config.WorkflowPluginEntry{{Name: "source/test"}},
		},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	want := []string{
		"collect:source/test",
		"process:builtin/deduplicate",
		"process:builtin/clean-text",
		"process:source/test",
		"report:source/test",
	}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events mismatch\n got: %#v\nwant: %#v", events, want)
	}
}

func TestRunWorkflow_MergesCollectResultsAndReporterTitle(t *testing.T) {
	var gotTitle string

	plugin.RegisterWorkflow("builtin/deduplicate", recorderPlugin{})
	plugin.RegisterWorkflow("builtin/clean-text", recorderPlugin{})
	plugin.RegisterWorkflow("source/title", recorderPlugin{
		collectResult: plugin.CollectResult{
			Title: "Collected Title",
			Items: []types.FeedItem{types.FeedItem{Title: "first"}.WithDefaults()},
		},
		reportTitle: &gotTitle,
	})

	err := Run(context.Background(), Params{
		SourceName: "hacker-news",
		SourceConfig: config.WorkflowSourceConfig{
			Name:    "hacker-news",
			Plugins: []config.WorkflowPluginEntry{{Name: "source/title"}},
		},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if gotTitle != "Collected Title" {
		t.Fatalf("expected collected title fallback, got %q", gotTitle)
	}
}

func TestRunWorkflow_LogsProgressAndSummary(t *testing.T) {
	var logs strings.Builder

	plugin.RegisterWorkflow("builtin/deduplicate", recorderPlugin{})
	plugin.RegisterWorkflow("builtin/clean-text", recorderPlugin{})
	plugin.RegisterWorkflow("source/test", recorderPlugin{
		collectResult: plugin.CollectResult{
			Title: "Collected Title",
			Items: []types.FeedItem{
				types.FeedItem{Title: "visible", Level: types.LevelRecommended}.WithDefaults(),
				types.FeedItem{Title: "hidden", Level: types.LevelRejected}.WithDefaults(),
			},
		},
	})

	err := Run(context.Background(), Params{
		SourceName: "hacker-news",
		SourceConfig: config.WorkflowSourceConfig{
			Name:    "hacker-news",
			Context: "context",
			Plugins: []config.WorkflowPluginEntry{{Name: "source/test"}},
		},
		Logger: slog.New(slog.NewTextHandler(&logs, nil)),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	output := logs.String()
	for _, want := range []string{
		"starting workflow",
		"running collect plugin",
		"collect completed",
		"running process plugin",
		"processing completed",
		"running report plugin",
		"workflow completed",
		"source=hacker-news",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected log output to contain %q, got %s", want, output)
		}
	}
}
