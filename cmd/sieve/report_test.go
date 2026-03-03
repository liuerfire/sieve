package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseFormats(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{"empty", "", []string{"json", "html"}},
		{"all", "all", []string{"json", "html"}},
		{"json only", "json", []string{"json"}},
		{"html only", "html", []string{"html"}},
		{"both", "json,html", []string{"json", "html"}},
		{"case insensitive", "JSON,HTML", []string{"json", "html"}},
		{"with spaces", "json, html", []string{"json", "html"}},
		{"invalid", "pdf", []string{"json", "html"}},
		{"mixed case with spaces", " Json , HTML ", []string{"json", "html"}},
		{"partial invalid", "json,pdf", []string{"json"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFormats(tt.input)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("parseFormats(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

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
