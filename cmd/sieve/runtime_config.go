package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/storage"
)

func loadRuntimeConfig(ctx context.Context, s *storage.Storage) (*config.Config, error) {
	feeds, err := s.ListFeeds(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("list feeds: %w", err)
	}
	if len(feeds) == 0 {
		return nil, fmt.Errorf("no enabled feeds found in database")
	}

	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}

	cfg := &config.Config{
		Global: config.GlobalConfig{
			HighInterest:          strings.TrimSpace(settings["high_interest"]),
			Interest:              strings.TrimSpace(settings["interest"]),
			Uninterested:          strings.TrimSpace(settings["uninterested"]),
			Exclude:               strings.TrimSpace(settings["exclude"]),
			PreferredLanguage:     strings.TrimSpace(settings["preferred_language"]),
			Timeout:               parseIntSetting(settings, "timeout"),
			AITimeBetweenRequests: parseIntSetting(settings, "ai_time_between_ms"),
			AIBurstLimit:          parseIntSetting(settings, "ai_burst_limit"),
			AIMaxConcurrency:      parseIntSetting(settings, "ai_max_concurrency"),
			HTMLMaxAgeDays:        parseIntSetting(settings, "html_max_age_days"),
			EnableArchives:        parseBoolSetting(settings, "enable_archives"),
		},
		Sources: make([]config.SourceConfig, 0, len(feeds)),
	}

	if cfg.Global.PreferredLanguage == "" {
		cfg.Global.PreferredLanguage = "en"
	}

	globalProvider := strings.TrimSpace(settings["ai_provider"])
	globalModel := strings.TrimSpace(settings["ai_model"])
	if globalProvider != "" || globalModel != "" {
		cfg.Global.AI = &config.AIConfig{
			Provider: globalProvider,
			Model:    globalModel,
		}
	}

	for _, feed := range feeds {
		src := config.SourceConfig{
			ID:           feed.ID,
			Name:         feed.Name,
			Title:        feed.Name,
			URL:          feed.URL,
			HighInterest: feed.HighInterest,
			Interest:     feed.Interest,
			Uninterested: feed.Uninterested,
			Exclude:      feed.Exclude,
			Plugins:      feed.Plugins,
			Summarize:    feed.Summarize,
			Timeout:      feed.Timeout,
		}
		if feed.AIProvider != "" || feed.AIModel != "" {
			src.AI = &config.AIConfig{
				Provider: feed.AIProvider,
				Model:    feed.AIModel,
			}
		}
		cfg.Sources = append(cfg.Sources, src)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid runtime config from database: %w", err)
	}
	return cfg, nil
}

func parseIntSetting(values map[string]string, key string) int {
	raw := strings.TrimSpace(values[key])
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return n
}

func parseBoolSetting(values map[string]string, key string) bool {
	raw := strings.TrimSpace(values[key])
	if raw == "" {
		return false
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}
	return v
}
