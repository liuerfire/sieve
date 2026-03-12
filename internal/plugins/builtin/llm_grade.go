package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/llm"
	"github.com/liuerfire/sieve/internal/plugins"
	"github.com/liuerfire/sieve/internal/types"
)

type LLMGradePlugin struct {
	plugins.BasePlugin
}

type llmGradeOptions struct {
	GlobalHighInterest string `json:"globalHighInterest"`
	GlobalInterest     string `json:"globalInterest"`
	GlobalUninterested string `json:"globalUninterested"`
	GlobalAvoid        string `json:"globalAvoid"`
	HighInterest       string `json:"highInterest"`
	Interest           string `json:"interest"`
	Uninterested       string `json:"uninterested"`
	Avoid              string `json:"avoid"`
	Context            string `json:"context"`
}

func (LLMGradePlugin) ProcessItems(ctx context.Context, items []types.FeedItem, entry config.PluginEntry, runCtx plugins.Context) ([]types.FeedItem, error) {
	adapter, err := requireProvider(runCtx, "balanced")
	if err != nil {
		return nil, err
	}
	var opts llmGradeOptions
	if len(entry.Options) > 0 {
		if err := json.Unmarshal(entry.Options, &opts); err != nil {
			return nil, err
		}
	}

	toGrade := make([]types.FeedItem, 0)
	reqItems := make([]llm.GradeItem, 0)
	for _, item := range items {
		if item.Level == types.LevelRejected {
			continue
		}
		toGrade = append(toGrade, item)
		reqItems = append(reqItems, llm.GradeItem{
			GUID:  item.GUID,
			Title: item.Title,
			Meta:  stringFromExtra(item.Extra["meta"]),
		})
	}
	if len(reqItems) == 0 {
		return items, nil
	}

	results, err := adapter.Grade(ctx, llm.GradeRequest{
		SourceContext:      runCtx.SourceContext,
		Context:            opts.Context,
		GlobalHigh:         opts.GlobalHighInterest,
		GlobalInterest:     opts.GlobalInterest,
		GlobalUninterested: opts.GlobalUninterested,
		GlobalAvoid:        opts.GlobalAvoid,
		High:               opts.HighInterest,
		Interest:           opts.Interest,
		Uninterested:       opts.Uninterested,
		Avoid:              opts.Avoid,
		Items:              reqItems,
	})
	if err != nil {
		return nil, err
	}
	if len(results) != len(reqItems) {
		return nil, fmt.Errorf("grade result count mismatch")
	}

	gradeMap := make(map[string]llm.GradeResult, len(results))
	for _, result := range results {
		if result.GUID == "" || result.Reason == "" {
			return nil, fmt.Errorf("invalid grade result")
		}
		switch types.FeedLevel(result.Level) {
		case types.LevelCritical, types.LevelRecommended, types.LevelOptional, types.LevelRejected:
		default:
			return nil, fmt.Errorf("invalid grade level %q", result.Level)
		}
		gradeMap[result.GUID] = result
	}

	out := make([]types.FeedItem, 0, len(items))
	for _, item := range items {
		if result, ok := gradeMap[item.GUID]; ok {
			item.Level = types.FeedLevel(result.Level)
			item.Reason = result.Reason
		}
		out = append(out, item)
	}
	return out, nil
}

func init() {
	plugins.Register("builtin/llm-grade", LLMGradePlugin{})
}
