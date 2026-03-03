package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sieve",
	Short: "Sieve is an intelligent RSS news aggregator",
	Long:  `Sieve uses AI to automatically filter and summarize RSS feeds based on your interests.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", "config.json", "config file (default is config.json)")
	rootCmd.PersistentFlags().StringP("db", "d", "sieve.db", "database file (default is sieve.db)")
}
