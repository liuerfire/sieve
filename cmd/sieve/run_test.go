package main

import (
	"strings"
	"testing"
)

func TestRunCmd_Help(t *testing.T) {
	output, err := executeCommand(rootCmd, "run", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Fetch RSS feeds") {
		t.Error("expected run help to contain description")
	}
}

func TestRunCmd_Flags(t *testing.T) {
	runCmd.SetArgs([]string{"--help"})
	output, err := executeCommand(rootCmd, "run", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "--config") {
		t.Error("expected run command to have --config flag")
	}
	if !strings.Contains(output, "--db") {
		t.Error("expected run command to have --db flag")
	}
	if !strings.Contains(output, "--ui") {
		t.Error("expected run command to have --ui flag")
	}
}
