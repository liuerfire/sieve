package main

import (
	"os"
	"path/filepath"
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
	root := newRootCmd()
	output, err := executeCommand(root, "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Usage:") {
		t.Error("expected help output to contain 'Usage:'")
	}
}

func TestRootCmd_HelpOmitsReport(t *testing.T) {
	root := newRootCmd()
	output, err := executeCommand(root, "--help")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, "report") {
		t.Fatal("expected root help to omit report command")
	}
}

func TestRootCmd_HelpOmitsRun(t *testing.T) {
	root := newRootCmd()
	output, err := executeCommand(root, "--help")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, "\n  run ") {
		t.Fatal("expected root help to omit run command")
	}
}

func TestRootCommand_RequiresSourceName(t *testing.T) {
	root := newRootCmd()

	output, err := executeCommand(root)
	if err == nil {
		t.Fatal("expected missing source name to return an error")
	}
	if !strings.Contains(output, "Usage:") {
		t.Fatalf("expected usage output, got %q", output)
	}
	if !strings.Contains(output, "accepts 1 arg(s), received 0") {
		t.Fatalf("expected missing arg error, got %q", output)
	}
}

func TestRootCommand_ParsesConfigAndDryRun(t *testing.T) {
	root := newRootCmd()

	called := false
	restore := swapRunRoot(func(_ *cobra.Command, args []string, configPath string, dryRun bool) error {
		called = true
		if len(args) != 1 || args[0] != "hacker-news" {
			t.Fatalf("unexpected args: %#v", args)
		}
		if configPath != "custom.json" {
			t.Fatalf("unexpected config path: %q", configPath)
		}
		if !dryRun {
			t.Fatal("expected dry-run to be true")
		}
		return nil
	})
	defer restore()

	output, err := executeCommand(root, "hacker-news", "--config", "custom.json", "--dry-run")
	if err != nil {
		t.Fatalf("expected no error, got %v with output %q", err, output)
	}
	if !called {
		t.Fatal("expected runRoot to be called")
	}
}

func TestCLI_RunWorkflowForNamedSource(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	outputPath := filepath.Join(dir, "feed.xml")

	err := os.WriteFile(configPath, []byte(`{
  "llm": {
    "provider": "openai",
    "models": {
      "fast": "gpt-fast",
      "balanced": "gpt-balanced",
      "powerful": "gpt-powerful"
    }
  },
  "plugins": {
    "builtin/reporter-rss": {
      "outputPath": "`+outputPath+`"
    }
  },
  "sources": [
    {
      "name": "test-source",
      "plugins": [
        {
          "name": "builtin/reporter-rss",
          "options": {
            "title": "Test Feed"
          }
        }
      ]
    }
  ]
}`), 0o644)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	root := newRootCmd()
	output, err := executeCommand(root, "test-source", "--config", configPath, "--dry-run")
	if err != nil {
		t.Fatalf("expected no error, got %v with output %q", err, output)
	}
	if !strings.Contains(output, "starting workflow") {
		t.Fatalf("expected startup log, got %q", output)
	}
	if !strings.Contains(output, "config="+configPath) {
		t.Fatalf("expected config path in logs, got %q", output)
	}
	if !strings.Contains(output, "workflow completed") {
		t.Fatalf("expected completion log, got %q", output)
	}
}
