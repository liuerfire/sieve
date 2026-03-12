package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGuidTracker_FiltersOldGuidsAndPersistsSortedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	oldTime := time.Now().AddDate(0, 0, -10).UTC().Format(time.RFC3339)
	newTime := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	content := `{
  "guids": {
    "z-guid": "` + newTime + `",
    "a-guid": "` + oldTime + `"
  },
  "updated_at": "` + newTime + `"
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	tracker, err := NewGUIDTracker(path)
	if err != nil {
		t.Fatalf("NewGUIDTracker: %v", err)
	}

	tracker.MarkProcessed([]string{"b-guid"})
	tracker.Cleanup()
	if err := tracker.Persist(); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "a-guid") {
		t.Fatalf("expected old guid to be removed, got %s", text)
	}
	if !strings.Contains(text, "\"b-guid\"") || !strings.Contains(text, "\"z-guid\"") {
		t.Fatalf("expected persisted guids, got %s", text)
	}
	if strings.Index(text, "\"b-guid\"") > strings.Index(text, "\"z-guid\"") {
		t.Fatalf("expected guids to be sorted, got %s", text)
	}
}

func TestGuidTracker_PersistCreatesParentDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output", "history.json")

	tracker, err := NewGUIDTracker(path)
	if err != nil {
		t.Fatalf("NewGUIDTracker: %v", err)
	}

	tracker.MarkProcessed([]string{"guid-1"})
	if err := tracker.Persist(); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Stat: %v", err)
	}
}
