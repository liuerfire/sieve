package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/liuerfire/sieve/internal/ai"
	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/engine"
	"github.com/liuerfire/sieve/internal/storage"
	"github.com/liuerfire/sieve/internal/ui"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the aggregator",
	Long:  `Fetch RSS feeds, classify with AI, and generate reports.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		configFile, _ := cmd.Flags().GetString("config")
		dbFile, _ := cmd.Flags().GetString("db")
		useUI, _ := cmd.Flags().GetBool("ui")

		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		s, err := storage.InitDB(ctx, dbFile)
		if err != nil {
			return fmt.Errorf("init storage: %w", err)
		}
		defer s.Close()

		a := ai.NewClient()
		hasProvider := false

		if key := os.Getenv("GEMINI_API_KEY"); key != "" {
			a.AddProvider(ai.Gemini, key)
			hasProvider = true
		}
		if key := os.Getenv("QWEN_API_KEY"); key != "" {
			a.AddProvider(ai.Qwen, key)
			hasProvider = true
		}

		if !hasProvider {
			return fmt.Errorf("GEMINI_API_KEY or QWEN_API_KEY must be set")
		}

		eng := engine.NewEngine(cfg, s, a)

		if useUI {
			slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

			sourceNames := make([]string, len(cfg.Sources))
			for i, s := range cfg.Sources {
				sourceNames[i] = s.Name
			}

			return ui.RunDashboard(ctx, sourceNames, func(report func(engine.ProgressEvent)) error {
				eng.OnProgress = report
				_, err := eng.Run(ctx)
				return err
			})
		}

		slog.Info("Starting Sieve aggregator...")
		result, err := eng.Run(ctx)
		if err != nil {
			return fmt.Errorf("aggregator run: %w", err)
		}

		if result != nil {
			slog.Info("Sieve run completed",
				"sources", result.SourcesProcessed,
				"failed", len(result.SourcesFailed),
				"items", result.ItemsProcessed,
				"high_interest", result.ItemsHighInterest)
		}
		return nil
	},
}

func init() {
	runCmd.Flags().Bool("ui", false, "Show TUI dashboard")
	rootCmd.AddCommand(runCmd)
}
