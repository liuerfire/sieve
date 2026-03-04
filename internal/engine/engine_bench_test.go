package engine

import (
	"fmt"
	"testing"
	"time"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/storage"
)

func BenchmarkGenerateJSON(b *testing.B) {
	ctx := b.Context()
	db, err := storage.InitDB(ctx, ":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	// Add test data - setup outside timed section
	now := time.Now()
	for i := range 100 {
		err := db.SaveItem(ctx, &storage.Item{
			ID:            fmt.Sprintf("item-%d", i),
			Title:         fmt.Sprintf("Test Item %d", i),
			Link:          fmt.Sprintf("http://example.com/%d", i),
			InterestLevel: "interest",
			PublishedAt:   now,
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	eng := NewEngine(&config.Config{}, db, nil)

	b.ResetTimer()
	for b.Loop() {
		eng.GenerateJSON(ctx, "/dev/null")
	}
}
