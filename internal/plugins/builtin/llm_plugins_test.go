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

type staticProvider struct {
	gradeResults  []llm.GradeResult
	summaryResult llm.SummaryResult
	gradeErr      error
	summaryErr    error
}

func (p staticProvider) Grade(_ context.Context, _ llm.GradeRequest) ([]llm.GradeResult, error) {
	return p.gradeResults, p.gradeErr
}

func (p staticProvider) Summarize(_ context.Context, _ llm.SummaryRequest) (llm.SummaryResult, error) {
	return p.summaryResult, p.summaryErr
}

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
		LLM: func(string) (llm.Provider, error) {
			return staticProvider{
				gradeResults: []llm.GradeResult{{GUID: "g1", Level: "critical", Reason: "fit"}},
			}, nil
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
		LLM: func(string) (llm.Provider, error) {
			return staticProvider{
				gradeResults: []llm.GradeResult{{GUID: "g1", Level: "recommend", Reason: "fit"}},
			}, nil
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
		LLM: func(string) (llm.Provider, error) {
			return staticProvider{
				summaryResult: llm.SummaryResult{
					GUID:        "g1",
					Title:       "New",
					Description: "<p>summary</p>",
				},
			}, nil
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
		LLM: func(string) (llm.Provider, error) {
			return staticProvider{
				summaryResult: llm.SummaryResult{
					GUID:     "g1",
					Rejected: true,
				},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("ProcessItems: %v", err)
	}
	if got[0].Level != types.LevelRejected {
		t.Fatalf("expected rejected item, got %#v", got[0])
	}
}
