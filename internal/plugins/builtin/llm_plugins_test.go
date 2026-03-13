package builtin

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/llm"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

type staticProvider struct {
	gradeResults   []llm.GradeResult
	summaryResult  llm.SummaryResult
	summaryResults []llm.SummaryResult
	gradeErr       error
	summaryErr     error
	summaryIndex   int
}

func withWorkingDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
	return dir
}

func (p *staticProvider) Grade(ctx context.Context, req llm.GradeRequest) ([]llm.GradeResult, error) {
	if p.gradeErr != nil {
		return nil, p.gradeErr
	}
	if req.WriteGradeResults != nil {
		if err := req.WriteGradeResults(ctx, p.gradeResults); err != nil {
			return nil, err
		}
	}
	return p.gradeResults, nil
}

func (p *staticProvider) Summarize(ctx context.Context, req llm.SummaryRequest) (llm.SummaryResult, error) {
	if p.summaryErr != nil {
		return llm.SummaryResult{}, p.summaryErr
	}
	result := p.summaryResult
	if len(p.summaryResults) > 0 {
		if p.summaryIndex >= len(p.summaryResults) {
			return llm.SummaryResult{}, io.EOF
		}
		result = p.summaryResults[p.summaryIndex]
		p.summaryIndex++
	}
	if req.WriteSummary != nil {
		if err := req.WriteSummary(ctx, result); err != nil {
			return llm.SummaryResult{}, err
		}
	}
	return result, nil
}

func TestLLMGrade_AppliesValidatedResults(t *testing.T) {
	dir := withWorkingDir(t)
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
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		SourceName: "source",
		LLM: func(string) (llm.Provider, error) {
			return &staticProvider{
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
	data, err := os.ReadFile(filepath.Join(dir, "output", "source-llm-grade.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var payload struct {
		Items []llm.GradeResult `json:"items"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(payload.Items) != 1 || payload.Items[0].GUID != "g1" {
		t.Fatalf("unexpected grade output: %#v", payload)
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
			return &staticProvider{
				gradeResults: []llm.GradeResult{{GUID: "g1", Level: "recommend", Reason: "fit"}},
			}, nil
		},
	})
	if err == nil {
		t.Fatal("expected invalid LLM level to return an error")
	}
}

func TestLLMSummarize_UpdatesTitleAndDescription(t *testing.T) {
	dir := withWorkingDir(t)
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
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		SourceName: "source",
		LLM: func(string) (llm.Provider, error) {
			return &staticProvider{
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
	data, err := os.ReadFile(filepath.Join(dir, "output", "source-llm-summary.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var payload struct {
		Items []llm.SummaryResult `json:"items"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(payload.Items) != 1 || payload.Items[0].GUID != "g1" {
		t.Fatalf("unexpected summary output: %#v", payload)
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
			return &staticProvider{
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

func TestLLMGrade_DryRunDoesNotWriteOutput(t *testing.T) {
	dir := withWorkingDir(t)
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
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		SourceName: "source",
		IsDryRun:   true,
		LLM: func(string) (llm.Provider, error) {
			return &staticProvider{
				gradeResults: []llm.GradeResult{{GUID: "g1", Level: "critical", Reason: "fit"}},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("ProcessItems: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "output", "source-llm-grade.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no output file, got err=%v", err)
	}
}

func TestLLMSummarize_DryRunDoesNotWriteOutput(t *testing.T) {
	dir := withWorkingDir(t)
	items := []types.FeedItem{
		types.FeedItem{
			Title:       "Old",
			GUID:        "g1",
			Description: "body",
			Extra:       map[string]any{"content": "full"},
		}.WithDefaults(),
	}

	_, err := LLMSummarizePlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{
		Name: "builtin/llm-summarize",
	}, plugins.Context{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		SourceName: "source",
		IsDryRun:   true,
		LLM: func(string) (llm.Provider, error) {
			return &staticProvider{
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
	if _, err := os.Stat(filepath.Join(dir, "output", "source-llm-summary.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no output file, got err=%v", err)
	}
}

func TestLLMSummarize_WritesAggregateOutputForRun(t *testing.T) {
	dir := withWorkingDir(t)
	items := []types.FeedItem{
		types.FeedItem{
			Title:       "Old 1",
			GUID:        "g1",
			Description: "body 1",
			Extra:       map[string]any{"content": "full 1"},
		}.WithDefaults(),
		types.FeedItem{
			Title:       "Old 2",
			GUID:        "g2",
			Description: "body 2",
			Extra:       map[string]any{"content": "full 2"},
		}.WithDefaults(),
	}
	_, err := LLMSummarizePlugin{}.ProcessItems(context.Background(), items, config.PluginEntry{
		Name: "builtin/llm-summarize",
	}, plugins.Context{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		SourceName: "source",
		LLM: func(string) (llm.Provider, error) {
			return &staticProvider{
				summaryResults: []llm.SummaryResult{
					{
						GUID:        "g1",
						Title:       "New 1",
						Description: "<p>summary 1</p>",
					},
					{
						GUID:        "g2",
						Title:       "New 2",
						Description: "<p>summary 2</p>",
					},
				},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("ProcessItems: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "output", "source-llm-summary.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var payload struct {
		Items []llm.SummaryResult `json:"items"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(payload.Items) != 2 || payload.Items[0].GUID != "g1" || payload.Items[1].GUID != "g2" {
		t.Fatalf("unexpected summary aggregate output: %#v", payload)
	}
}
