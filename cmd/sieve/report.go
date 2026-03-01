package main

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/liuerfire/sieve/internal/ai"
	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/engine"
	"github.com/liuerfire/sieve/internal/storage"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate reports from database",
	Long:  `Generate XML and HTML reports using existing items in the database without calling AI or fetching RSS.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		configFile, _ := cmd.Flags().GetString("config")
		dbFile, _ := cmd.Flags().GetString("db")
		jsonOutput, _ := cmd.Flags().GetString("json")
		htmlOutput, _ := cmd.Flags().GetString("html")
		skipJSON, _ := cmd.Flags().GetBool("skip-json")
		skipHTML, _ := cmd.Flags().GetBool("skip-html")

		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		s, err := storage.InitDB(ctx, dbFile)
		if err != nil {
			return fmt.Errorf("init storage: %w", err)
		}
		defer s.Close()

		// Dummy AI client since report generation doesn't need it
		a := ai.NewClient()
		eng := engine.NewEngine(cfg, s, a)

		if !skipJSON {
			slog.Info("Generating JSON report...", "output", jsonOutput)
			if err := eng.GenerateJSON(ctx, jsonOutput); err != nil {
				return fmt.Errorf("json generation: %w", err)
			}
		}

		if !skipHTML {
			slog.Info("Generating HTML report...", "output", htmlOutput)
			if err := eng.GenerateHTMLWithArchives(ctx, htmlOutput); err != nil {
				return fmt.Errorf("html generation: %w", err)
			}
		}

		return nil
	},
}

func init() {
	reportCmd.Flags().String("json", "index.json", "output JSON file")
	reportCmd.Flags().String("html", "index.html", "output HTML file")
	reportCmd.Flags().Bool("skip-json", false, "skip JSON generation")
	reportCmd.Flags().Bool("skip-html", false, "skip HTML generation")
	rootCmd.AddCommand(reportCmd)
}
