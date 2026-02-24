package storage

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestInitDB(t *testing.T) {
	dbPath := "test_sieve.db"
	defer os.Remove(dbPath)
	db, err := InitDB(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	if db == nil {
		t.Fatal("expected db instance")
	}
}

func TestSaveItemAndGetItems(t *testing.T) {
	dbPath := "test_items.db"
	defer os.Remove(dbPath)
	s, err := InitDB(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().Truncate(time.Second)
	item := &Item{
		ID:            "item-1",
		Source:        "test-source",
		Title:         "Test Item",
		Link:          "http://example.com/1",
		Description:   "Description",
		Content:       "Content",
		Summary:       "Summary",
		InterestLevel: "high_interest",
		PublishedAt:   now,
	}

	ctx := context.Background()
	if err := s.SaveItem(ctx, item); err != nil {
		t.Fatalf("failed to save item: %v", err)
	}

	// Test Upsert
	item.Title = "Updated Title"
	if err := s.SaveItem(ctx, item); err != nil {
		t.Fatalf("failed to upsert item: %v", err)
	}

	// Test Exclude
	excludedItem := &Item{
		ID:            "item-exclude",
		Source:        "test-source",
		Title:         "Excluded Item",
		InterestLevel: "exclude",
		PublishedAt:   now,
	}
	if err := s.SaveItem(ctx, excludedItem); err != nil {
		t.Fatal(err)
	}

	items, err := s.GetItems(ctx)
	if err != nil {
		t.Fatalf("failed to get items: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 non-excluded item, got %d", len(items))
	}

	if items[0].Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got '%s'", items[0].Title)
	}

	if !items[0].PublishedAt.Equal(now) {
		t.Errorf("expected published_at %v, got %v", now, items[0].PublishedAt)
	}
}
