package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"
)

const historyRetentionDays = 4

type guidHistory struct {
	Guids     map[string]string `json:"guids"`
	UpdatedAt string            `json:"updated_at"`
}

type GUIDTracker struct {
	historyPath string
	processed   map[string]string
}

func NewGUIDTracker(historyPath string) (*GUIDTracker, error) {
	processed, err := loadGUIDHistory(historyPath)
	if err != nil {
		return nil, err
	}
	return &GUIDTracker{
		historyPath: historyPath,
		processed:   processed,
	}, nil
}

func loadGUIDHistory(historyPath string) (map[string]string, error) {
	data, err := os.ReadFile(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	var history guidHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("invalid guid history %q: %w", historyPath, err)
	}
	if history.Guids == nil {
		history.Guids = map[string]string{}
	}
	return history.Guids, nil
}

func (t *GUIDTracker) IsProcessed(guid string) bool {
	_, ok := t.processed[guid]
	return ok
}

func (t *GUIDTracker) MarkProcessed(guids []string) {
	now := time.Now().UTC().Format(time.RFC3339)
	for _, guid := range guids {
		if _, ok := t.processed[guid]; !ok {
			t.processed[guid] = now
		}
	}
}

func (t *GUIDTracker) Cleanup() {
	cutoff := time.Now().AddDate(0, 0, -historyRetentionDays)
	filtered := make(map[string]string, len(t.processed))
	for guid, processedTime := range t.processed {
		parsed, err := time.Parse(time.RFC3339, processedTime)
		if err != nil || !parsed.Before(cutoff) {
			filtered[guid] = processedTime
		}
	}
	t.processed = filtered
}

func (t *GUIDTracker) Persist() error {
	keys := make([]string, 0, len(t.processed))
	for guid := range t.processed {
		keys = append(keys, guid)
	}
	slices.Sort(keys)

	ordered := make(map[string]string, len(keys))
	for _, guid := range keys {
		ordered[guid] = t.processed[guid]
	}

	data, err := json.MarshalIndent(guidHistory{
		Guids:     ordered,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(t.historyPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(t.historyPath, data, 0o644)
}
