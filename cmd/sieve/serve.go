package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/liuerfire/sieve/internal/server"
	"github.com/liuerfire/sieve/internal/storage"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Web UI server",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		dbFile, _ := cmd.Flags().GetString("db")
		port, _ := cmd.Flags().GetInt("port")

		s, err := storage.InitDB(ctx, dbFile)
		if err != nil {
			return fmt.Errorf("init storage: %w", err)
		}
		defer s.Close()

		srv := server.NewServer(s)
		return srv.ListenAndServe(fmt.Sprintf(":%d", port))
	},
}

func init() {
	serveCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}
