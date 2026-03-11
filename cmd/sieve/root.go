package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/llm"
	_ "github.com/liuerfire/sieve/internal/plugins/all"
	"github.com/liuerfire/sieve/internal/workflow"
)

type rootRunner func(cmd *cobra.Command, args []string, configPath string, dryRun bool) error

var runRoot rootRunner = defaultRunRoot

func swapRunRoot(fn rootRunner) func() {
	prev := runRoot
	runRoot = fn
	return func() {
		runRoot = prev
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sieve <source-name>",
		Short: "Run the RSS pipeline for one source",
		Long:  `Sieve runs an AI-assisted RSS pipeline for the named source.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			dryRun, err := cmd.Flags().GetBool("dry-run")
			if err != nil {
				return err
			}
			return runRoot(cmd, args, configPath, dryRun)
		},
	}

	cmd.Flags().String("config", "config.json", "path to config file")
	cmd.Flags().Bool("dry-run", false, "run without persisting normal output effects")
	return cmd
}

var rootCmd = newRootCmd()

func defaultRunRoot(cmd *cobra.Command, args []string, configPath string, dryRun bool) error {
	cfg, err := config.LoadWorkflowConfig(configPath)
	if err != nil {
		return err
	}

	sourceName := args[0]
	var source *config.WorkflowSourceConfig
	for i := range cfg.Sources {
		if cfg.Sources[i].Name == sourceName {
			source = &cfg.Sources[i]
			break
		}
	}
	if source == nil {
		return fmt.Errorf("source %q not found in config", sourceName)
	}

	logger := slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), nil))
	logger.Info("starting workflow", "source", sourceName, "config", configPath, "dryRun", dryRun)
	return workflow.Run(context.Background(), workflow.Params{
		SourceName:          sourceName,
		SourceConfig:        *source,
		LLMConfig:           cfg.LLM,
		GlobalPluginOptions: cfg.Plugins,
		IsDryRun:            dryRun,
		Logger:              logger,
		LLMFactory: func(tier string) any {
			model := cfg.LLM.Models.Balanced
			switch tier {
			case "fast":
				model = cfg.LLM.Models.Fast
			case "powerful":
				model = cfg.LLM.Models.Powerful
			}
			provider, err := llm.CreateProvider(llm.Config{
				Provider: cfg.LLM.Provider,
				Model:    model,
				BaseURL:  cfg.LLM.BaseURL,
			})
			if err != nil {
				return llm.StaticProvider{GradeErr: err, SummaryErr: err}
			}
			return provider
		},
	})
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
