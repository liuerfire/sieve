package storage

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func BenchmarkSaveItem(b *testing.B) {
	ctx := context.Background()
	s, err := InitDB(ctx, ":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()

	item := &Item{
		ID:            "test-id",
		Title:         "Test Item",
		Link:          "http://example.com",
		InterestLevel: "interest",
		PublishedAt:   time.Now(),
	}

	b.ResetTimer()
	for range b.N {
		s.SaveItem(ctx, item)
	}
}

func BenchmarkAllItems(b *testing.B) {
	ctx := context.Background()
	s, err := InitDB(ctx, ":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()

	// Add test data - setup outside timed section
	for i := range 1000 {
		err := s.SaveItem(ctx, &Item{
			ID:    fmt.Sprintf("item-%d", i),
			Title: "Test",
			Link:  "http://example.com",
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for range b.N {
		for _, err := range s.AllItems(ctx) {
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkExists(b *testing.B) {
	ctx := context.Background()
	s, err := InitDB(ctx, ":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()

	// Add test item - setup outside timed section
	err = s.SaveItem(ctx, &Item{
		ID:    "test-id",
		Title: "Test",
	})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for range b.N {
		s.Exists(ctx, "test-id")
	}
}
