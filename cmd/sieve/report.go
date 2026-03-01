package main

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/liuerfire/sieve/internal/ai"
	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/engine"
	"github.com/liuerfire/sieve/internal/storage"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate reports from database",
	Long:  `Generate reports from existing items without fetching RSS or calling AI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		configFile, _ := cmd.Flags().GetString("config")
		dbFile, _ := cmd.Flags().GetString("db")
		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")

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
		eng := engine.NewEngine(cfg, s, a)

		formats := parseFormats(format)

		for _, f := range formats {
			switch f {
			case "json":
				outputPath := output
				if outputPath == "" {
					outputPath = "index.json"
				}
				slog.Info("Generating JSON report...", "output", outputPath)
				if err := eng.GenerateJSON(ctx, outputPath); err != nil {
					return fmt.Errorf("json generation: %w", err)
				}
			case "html":
				outputPath := output
				if outputPath == "" {
					outputPath = "index.html"
				}
				slog.Info("Generating HTML report...", "output", outputPath)
				if err := eng.GenerateHTMLWithArchives(ctx, outputPath); err != nil {
					return fmt.Errorf("html generation: %w", err)
				}
			}
		}

		return nil
	},
}

func parseFormats(format string) []string {
	if format == "" || format == "all" {
		return []string{"json", "html"}
	}

	parts := strings.Split(format, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if p == "json" || p == "html" {
			result = append(result, p)
		}
	}

	if len(result) == 0 {
		return []string{"json", "html"}
	}
	return result
}

func init() {
	reportCmd.Flags().StringP("format", "f", "all", "Output format: json, html, or comma-separated (e.g., 'json,html')")
	reportCmd.Flags().StringP("output", "o", "", "Output file path (defaults: index.json, index.html)")
	rootCmd.AddCommand(reportCmd)
}
