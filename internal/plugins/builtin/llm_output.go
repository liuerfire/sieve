package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/liuerfire/sieve/internal/llm"
)

type gradeResultsFile struct {
	Items []llm.GradeResult `json:"items"`
}

type summaryResultsFile struct {
	Items []llm.SummaryResult `json:"items"`
}

func writeGradeResultsFile(ctx context.Context, path string, results []llm.GradeResult) error {
	return writeJSONFile(ctx, path, gradeResultsFile{Items: results})
}

func writeSummaryResultsFile(ctx context.Context, path string, results []llm.SummaryResult) error {
	return writeJSONFile(ctx, path, summaryResultsFile{Items: results})
}

func writeJSONFile(_ context.Context, path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
