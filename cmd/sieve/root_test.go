package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	_, output, err = captureOutput(root, args...)
	return output, err
}

func captureOutput(root *cobra.Command, args ...string) (c *cobra.Command, output string, err error) {
	root.SetArgs(args)
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	c, err = root.ExecuteC()
	return c, buf.String(), err
}

func TestRootCmd_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"--help"})
	output, err := executeCommand(rootCmd)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Usage:") {
		t.Error("expected help output to contain 'Usage:'")
	}
}

func TestRootCmd_NoArgs(t *testing.T) {
	rootCmd.SetArgs([]string{})
	output, err := executeCommand(rootCmd)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Available Commands:") {
		t.Error("expected root command to show available commands")
	}
}
