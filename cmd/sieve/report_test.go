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
	if !strings.Contains(output, "Generate XML and HTML reports") {
		t.Error("expected report help to contain description")
	}
}

func TestReportCmd_Flags(t *testing.T) {
	output, err := executeCommand(rootCmd, "report", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "--json") {
		t.Error("expected report command to have --json flag")
	}
	if !strings.Contains(output, "--html") {
		t.Error("expected report command to have --html flag")
	}
	if !strings.Contains(output, "--skip-json") {
		t.Error("expected report command to have --skip-json flag")
	}
	if !strings.Contains(output, "--skip-html") {
		t.Error("expected report command to have --skip-html flag")
	}
}
