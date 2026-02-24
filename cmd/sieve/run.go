package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/liuerfire/sieve/internal/ai"
	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/engine"
	"github.com/liuerfire/sieve/internal/storage"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the aggregator",
	Long:  `Run the RSS aggregator to fetch, filter, and summarize news.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create a context that is canceled when the user sends a termination signal
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		configFile, _ := cmd.Flags().GetString("config")
		dbFile, _ := cmd.Flags().GetString("db")

		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		s, err := storage.InitDB(ctx, dbFile)
		if err != nil {
			return fmt.Errorf("init storage: %w", err)
		}
		defer s.Close()

		apiKey := os.Getenv("GEMINI_API_KEY")
		provider := ai.Gemini
		if apiKey == "" {
			apiKey = os.Getenv("QWEN_API_KEY")
			provider = ai.Qwen
		}

		if apiKey == "" {
			return fmt.Errorf("GEMINI_API_KEY or QWEN_API_KEY must be set")
		}

		a := ai.NewClient(provider, apiKey)
		eng := engine.NewEngine(cfg, s, a)

		slog.Info("Starting Sieve aggregator...")
		if err := eng.Run(ctx); err != nil {
			return fmt.Errorf("aggregator run: %w", err)
		}

		slog.Info("Sieve run completed successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
