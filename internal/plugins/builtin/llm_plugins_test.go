package builtin

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/llm"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

func TestLLMGrade_AppliesValidatedResults(t *testing.T) {
	items := []types.FeedItem{
		types.FeedItem{
			Title: "A",
			GUID:  "g1",
			Extra: map[string]any{"meta": "desc"},
		}.WithDefaults(),
	}

	got, err := LLMGradePlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{
		Name: "builtin/llm-grade",
		Options: mustJSON(map[string]any{
			"globalHighInterest": "go",
		}),
	}, plugins.Context{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		LLM: func(string) any {
			return llm.StaticProvider{
				GradeResults: []llm.GradeResult{{GUID: "g1", Level: "critical", Reason: "fit"}},
			}
		},
	})
	if err != nil {
		t.Fatalf("ProcessItems: %v", err)
	}
	if got[0].Level != types.LevelCritical || got[0].Reason != "fit" {
		t.Fatalf("unexpected graded item: %#v", got[0])
	}
}

func TestLLMGrade_RejectsUnknownLevel(t *testing.T) {
	items := []types.FeedItem{
		types.FeedItem{
			Title: "A",
			GUID:  "g1",
			Extra: map[string]any{"meta": "desc"},
		}.WithDefaults(),
	}

	_, err := LLMGradePlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{
		Name: "builtin/llm-grade",
	}, plugins.Context{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		LLM: func(string) any {
			return llm.StaticProvider{
				GradeResults: []llm.GradeResult{{GUID: "g1", Level: "recommend", Reason: "fit"}},
			}
		},
	})
	if err == nil {
		t.Fatal("expected invalid LLM level to return an error")
	}
}

func TestLLMSummarize_UpdatesTitleAndDescription(t *testing.T) {
	items := []types.FeedItem{
		types.FeedItem{
			Title:       "Old",
			GUID:        "g1",
			Description: "body",
			Extra:       map[string]any{"content": "full"},
		}.WithDefaults(),
	}

	got, err := LLMSummarizePlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{
		Name: "builtin/llm-summarize",
		Options: mustJSON(map[string]any{
			"preferredLanguage": "en",
		}),
	}, plugins.Context{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		LLM: func(string) any {
			return llm.StaticProvider{
				SummaryResult: llm.SummaryResult{
					GUID:        "g1",
					Title:       "New",
					Description: "<p>summary</p>",
				},
			}
		},
	})
	if err != nil {
		t.Fatalf("ProcessItems: %v", err)
	}
	if got[0].Title != "New" || got[0].Description != "<p>summary</p>" {
		t.Fatalf("unexpected summarized item: %#v", got[0])
	}
}

func TestLLMSummarize_RejectedSummaryMarksItemRejected(t *testing.T) {
	items := []types.FeedItem{
		types.FeedItem{
			Title: "Old",
			GUID:  "g1",
			Extra: map[string]any{"content": "full"},
		}.WithDefaults(),
	}

	got, err := LLMSummarizePlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{
		Name:    "builtin/llm-summarize",
		Options: json.RawMessage(`{}`),
	}, plugins.Context{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		LLM: func(string) any {
			return llm.StaticProvider{
				SummaryResult: llm.SummaryResult{
					GUID:     "g1",
					Rejected: true,
				},
			}
		},
	})
	if err != nil {
		t.Fatalf("ProcessItems: %v", err)
	}
	if got[0].Level != types.LevelRejected {
		t.Fatalf("expected rejected item, got %#v", got[0])
	}
}
