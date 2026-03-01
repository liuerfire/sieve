package main

import (
	"strings"
	"testing"
)

func TestReportCmd_Help(t *testing.T) {
	output, err := executeCommand(rootCmd, "report", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Generate reports from existing items") {
		t.Error("expected report help to contain description")
	}
}

func TestReportCmd_Flags(t *testing.T) {
	output, err := executeCommand(rootCmd, "report", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "--format") {
		t.Error("expected report command to have --format flag")
	}
	if !strings.Contains(output, "--output") {
		t.Error("expected report command to have --output flag")
	}
}
