package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/llm"
	"github.com/liuerfire/sieve/internal/plugin"
	"github.com/liuerfire/sieve/internal/types"
)

type LLMSummarizePlugin struct {
	plugin.BaseWorkflowPlugin
}

type llmSummarizeOptions struct {
	PreferredLanguage string `json:"preferredLanguage"`
	Context           string `json:"context"`
	MaxConcurrency    int    `json:"maxConcurrency"`
}

func (LLMSummarizePlugin) ProcessItems(ctx context.Context, items []types.FeedItem, entry config.WorkflowPluginEntry, runCtx plugin.WorkflowContext) ([]types.FeedItem, error) {
	adapter, err := requireProvider(runCtx, "balanced")
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

func requireProvider(runCtx plugin.WorkflowContext, tier string) (llm.Provider, error) {
	if runCtx.LLM == nil {
		return nil, fmt.Errorf("llm provider not configured")
	}
	provider, ok := runCtx.LLM(tier).(llm.Provider)
	if !ok {
		return nil, fmt.Errorf("llm provider has unexpected type")
	}
	return provider, nil
}

func stringFromExtra(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func init() {
	plugin.RegisterWorkflow("builtin/llm-summarize", LLMSummarizePlugin{})
}
