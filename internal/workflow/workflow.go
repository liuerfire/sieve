package workflow

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/plugin"
	"github.com/liuerfire/sieve/internal/types"
)

var pipelinePrefix = []string{
	"builtin/deduplicate",
	"builtin/clean-text",
}

type Params struct {
	SourceName          string
	SourceConfig        config.WorkflowSourceConfig
	LLMConfig           config.WorkflowLLMConfig
	GlobalPluginOptions map[string]json.RawMessage
	IsDryRun            bool
	Logger              *slog.Logger
	LLMFactory          func(tier string) any
}

func Run(ctx context.Context, params Params) error {
	logInfo(params.Logger, "starting workflow", "source", params.SourceName, "dryRun", params.IsDryRun)

	runCtx := plugin.WorkflowContext{
		SourceName:    params.SourceName,
		SourceContext: params.SourceConfig.Context,
		IsDryRun:      params.IsDryRun,
		Logger:        params.Logger,
		LLM:           params.LLMFactory,
	}

	sourceEntries := make([]config.WorkflowPluginEntry, 0, len(params.SourceConfig.Plugins))
	for _, entry := range params.SourceConfig.Plugins {
		sourceEntries = append(sourceEntries, config.WorkflowPluginEntry{
			Name:    entry.Name,
			Options: mergeOptions(params.GlobalPluginOptions[entry.Name], entry.Options),
		})
	}

	sourcePlugins, err := plugin.LoadWorkflowPlugins(sourceEntries)
	if err != nil {
		return err
	}

	prefixEntries := make([]config.WorkflowPluginEntry, 0, len(pipelinePrefix))
	for _, name := range pipelinePrefix {
		prefixEntries = append(prefixEntries, config.WorkflowPluginEntry{
			Name:    name,
			Options: params.GlobalPluginOptions[name],
		})
	}

	prefixPlugins, err := plugin.LoadWorkflowPlugins(prefixEntries)
	if err != nil {
		return err
	}

	var collectedTitle string
	var items []types.FeedItem
	for _, loaded := range sourcePlugins {
		logInfo(params.Logger, "running collect plugin", "source", params.SourceName, "plugin", loaded.Name)
		result, err := loaded.Plugin.Collect(ctx, loaded.Entry, runCtx)
		if err != nil {
			return err
		}
		if result.Title != "" {
			collectedTitle = result.Title
		}
		items = append(items, result.Items...)
		logInfo(params.Logger, "collect completed", "source", params.SourceName, "plugin", loaded.Name, "items", len(result.Items), "title", result.Title)
	}

	processed := items
	for _, loaded := range prefixPlugins {
		logInfo(params.Logger, "running process plugin", "source", params.SourceName, "plugin", loaded.Name, "items", len(processed))
		processed = plugin.ApplyProcessItems(ctx, processed, loaded, runCtx)
	}
	for _, loaded := range sourcePlugins {
		logInfo(params.Logger, "running process plugin", "source", params.SourceName, "plugin", loaded.Name, "items", len(processed))
		processed = plugin.ApplyProcessItems(ctx, processed, loaded, runCtx)
	}

	visibleCount := 0
	rejectedCount := 0
	for _, item := range processed {
		if item.Level == types.LevelRejected {
			rejectedCount++
			continue
		}
		visibleCount++
	}
	logInfo(params.Logger, "processing completed", "source", params.SourceName, "items", len(processed), "visible", visibleCount, "rejected", rejectedCount)

	reportTitle := params.SourceConfig.Title
	if reportTitle == "" {
		reportTitle = collectedTitle
	}

	for _, loaded := range sourcePlugins {
		logInfo(params.Logger, "running report plugin", "source", params.SourceName, "plugin", loaded.Name, "items", len(processed), "title", reportTitle)
		reportEntry := loaded.Entry
		reportEntry.Options = mergeOptions(reportEntry.Options, mustMarshal(map[string]string{
			"sourceName": params.SourceName,
			"title":      reportTitle,
		}))
		if err := loaded.Plugin.Report(ctx, processed, reportEntry, runCtx); err != nil {
			return err
		}
	}

	logInfo(params.Logger, "workflow completed", "source", params.SourceName, "items", len(processed), "visible", visibleCount, "rejected", rejectedCount, "title", reportTitle)

	return nil
}

func logInfo(logger *slog.Logger, msg string, args ...any) {
	if logger == nil {
		return
	}
	logger.Info(msg, args...)
}

func mergeOptions(global json.RawMessage, local json.RawMessage) json.RawMessage {
	if len(global) == 0 {
		return local
	}
	if len(local) == 0 {
		return global
	}

	merged := map[string]any{}
	_ = json.Unmarshal(global, &merged)
	localValues := map[string]any{}
	_ = json.Unmarshal(local, &localValues)
	for key, value := range localValues {
		merged[key] = value
	}
	return mustMarshal(merged)
}

func mustMarshal(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
