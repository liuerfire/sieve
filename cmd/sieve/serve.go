package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/liuerfire/sieve/internal/ai"
	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/server"
	"github.com/liuerfire/sieve/internal/storage"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Web UI server",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		configFile, _ := cmd.Flags().GetString("config")
		dbFile, _ := cmd.Flags().GetString("db")
		port, _ := cmd.Flags().GetInt("port")

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

		srv := server.NewServer(cfg, s, a)
		return srv.ListenAndServe(fmt.Sprintf(":%d", port))
	},
}

func init() {
	serveCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}
