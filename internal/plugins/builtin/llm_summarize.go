package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/llm"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

type LLMSummarizePlugin struct {
	plugins.BasePlugin
}

type llmSummarizeOptions struct {
	PreferredLanguage string `json:"preferredLanguage"`
	Context           string `json:"context"`
	MaxConcurrency    int    `json:"maxConcurrency"`
}

func (LLMSummarizePlugin) ProcessItems(ctx context.Context, items []types.FeedItem, entry config.PluginEntry, runCtx plugins.Context) ([]types.FeedItem, error) {
	adapter, err := requireProvider(runCtx, "powerful")
	if err != nil {
		return nil, err
	}
	var opts llmSummarizeOptions
	if len(entry.Options) > 0 {
		if err := json.Unmarshal(entry.Options, &opts); err != nil {
			return nil, err
		}
	}
	if opts.PreferredLanguage == "" {
		opts.PreferredLanguage = "zh-CN"
	}

	summaryPath := filepath.Join("output", runCtx.SourceName+"-llm-summary.json")
	writtenSummaries := make([]llm.SummaryResult, 0, len(items))
	out := make([]types.FeedItem, 0, len(items))
	for _, item := range items {
		if item.Level == types.LevelRejected {
			out = append(out, item)
			continue
		}
		if skip, _ := item.Extra["skipSummarize"].(bool); skip {
			out = append(out, item)
			continue
		}

		result, err := adapter.Summarize(ctx, llm.SummaryRequest{
			PreferredLanguage: opts.PreferredLanguage,
			SourceContext:     runCtx.SourceContext,
			Context:           opts.Context,
			GUID:              item.GUID,
			Title:             item.Title,
			Description:       item.Description,
			Extra:             item.Extra,
			WriteSummary: func(ctx context.Context, result llm.SummaryResult) error {
				if runCtx.IsDryRun {
					return nil
				}
				writtenSummaries = append(writtenSummaries, result)
				return writeSummaryResultsFile(ctx, summaryPath, writtenSummaries)
			},
		})
		if err != nil {
			return nil, err
		}
		if result.GUID == "" {
			result.GUID = item.GUID
		}
		if result.GUID != item.GUID {
			return nil, fmt.Errorf("summary guid mismatch")
		}
		if result.Rejected {
			item.Level = types.LevelRejected
			out = append(out, item)
			continue
		}
		if result.Title != "" {
			item.Title = result.Title
		}
		if result.Description != "" {
			item.Description = result.Description
		}
		out = append(out, item)
	}
	return out, nil
}

func requireProvider(runCtx plugins.Context, tier string) (llm.Provider, error) {
	if runCtx.LLM == nil {
		return nil, fmt.Errorf("llm provider not configured")
	}
	return runCtx.LLM(tier)
}

func stringFromExtra(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func init() {
	plugins.Register("builtin/llm-summarize", LLMSummarizePlugin{})
}
