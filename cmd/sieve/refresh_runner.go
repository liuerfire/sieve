package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/liuerfire/sieve/internal/ai"
	"github.com/liuerfire/sieve/internal/engine"
	"github.com/liuerfire/sieve/internal/refresh"
	"github.com/liuerfire/sieve/internal/storage"
)

func newRefreshCoordinator(s *storage.Storage) *refresh.Coordinator {
	return refresh.NewCoordinator(func(ctx context.Context, report func(engine.ProgressEvent)) (*engine.EngineResult, error) {
		cfg, err := loadRuntimeConfig(ctx, s)
		if err != nil {
			return nil, fmt.Errorf("load runtime config from db: %w", err)
		}

		client, err := newAIClientFromEnv()
		if err != nil {
			return nil, err
		}

		eng := engine.NewEngine(cfg, s, client)
		eng.OnProgress = report
		return eng.Run(ctx)
	})
}

func newAIClientFromEnv() (*ai.Client, error) {
	client := ai.NewClient()
	hasProvider := false

	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		client.AddProvider(ai.Gemini, key)
		hasProvider = true
	}
	if key := os.Getenv("QWEN_API_KEY"); key != "" {
		client.AddProvider(ai.Qwen, key)
		hasProvider = true
	}

	if !hasProvider {
		return nil, fmt.Errorf("GEMINI_API_KEY or QWEN_API_KEY must be set")
	}

	return client, nil
}

func runScheduledRefresh(ctx context.Context, coordinator *refresh.Coordinator, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := coordinator.Trigger(ctx, "schedule"); err != nil {
				if err == refresh.ErrAlreadyRunning {
					slog.Info("Skipping scheduled refresh because another refresh is already running")
					continue
				}
				slog.Warn("Scheduled refresh failed", "error", err)
			}
		}
	}
}
