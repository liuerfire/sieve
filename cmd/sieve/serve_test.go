package main

import (
	"strings"
	"testing"
)

func TestServeCmd_Flags(t *testing.T) {
	output, err := executeCommand(rootCmd, "serve", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "--refresh-now") {
		t.Error("expected serve command to have --refresh-now flag")
	}
	if strings.Contains(output, "--schedule ") {
		t.Error("expected serve command to omit --schedule flag")
	}
	if !strings.Contains(output, "--schedule-interval") {
		t.Error("expected serve command to have --schedule-interval flag")
	}
}
