package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/liuerfire/sieve/internal/server"
	"github.com/liuerfire/sieve/internal/storage"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Web UI server",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		dbFile, _ := cmd.Flags().GetString("db")
		port, _ := cmd.Flags().GetInt("port")
		refreshNow, _ := cmd.Flags().GetBool("refresh-now")
		scheduleInterval, _ := cmd.Flags().GetDuration("schedule-interval")

		s, err := storage.InitDB(ctx, dbFile)
		if err != nil {
			return fmt.Errorf("init storage: %w", err)
		}
		defer s.Close()

		coordinator := newRefreshCoordinator(s)

		if refreshNow {
			_, err := coordinator.Trigger(context.WithoutCancel(ctx), "cli")
			return err
		}

		if scheduleInterval > 0 {
			go runScheduledRefresh(ctx, coordinator, scheduleInterval)
		}

		srv := server.NewServer(s, coordinator)
		return srv.ListenAndServe(ctx, fmt.Sprintf(":%d", port))
	},
}

func init() {
	serveCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	serveCmd.Flags().Bool("refresh-now", false, "Run one refresh cycle and exit")
	serveCmd.Flags().Duration("schedule-interval", 0, "Enable in-process scheduled refreshes at the given interval")
	rootCmd.AddCommand(serveCmd)
}
