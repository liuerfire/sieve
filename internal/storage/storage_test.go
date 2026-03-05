package storage

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestInitDB(t *testing.T) {
	dbPath := "test_sieve.db"
	defer os.Remove(dbPath)
	db, err := InitDB(t.Context(), dbPath)
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
	s, err := InitDB(t.Context(), dbPath)
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

	ctx := t.Context()
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

	var items []*Item
	for it, err := range s.AllItems(ctx) {
		if err != nil {
			t.Fatalf("failed to get item: %v", err)
		}
		items = append(items, it)
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

func TestGetItems_PrioritizesInterestAndSkipsExclude(t *testing.T) {
	dbPath := "test_triage_order.db"
	defer os.Remove(dbPath)
	s, err := InitDB(t.Context(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().Truncate(time.Second)
	for _, it := range []*Item{
		{
			ID:            "u1",
			Title:         "Uninterested",
			InterestLevel: "uninterested",
			PublishedAt:   now,
		},
		{
			ID:            "i1",
			Title:         "Interest",
			InterestLevel: "interest",
			PublishedAt:   now.Add(-time.Minute),
		},
		{
			ID:            "h1",
			Title:         "High",
			InterestLevel: "high_interest",
			PublishedAt:   now.Add(-2 * time.Minute),
		},
		{
			ID:            "x1",
			Title:         "Excluded",
			InterestLevel: "exclude",
			PublishedAt:   now.Add(-3 * time.Minute),
		},
	} {
		if err := s.SaveItem(t.Context(), it); err != nil {
			t.Fatal(err)
		}
	}

	items, err := s.GetItems(t.Context(), 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 non-excluded items, got %d", len(items))
	}
	if items[0].ID != "h1" || items[1].ID != "i1" || items[2].ID != "u1" {
		t.Fatalf("unexpected order: %s, %s, %s", items[0].ID, items[1].ID, items[2].ID)
	}
}

func TestExists(t *testing.T) {
	dbPath := "test_exists.db"
	defer os.Remove(dbPath)
	s, err := InitDB(t.Context(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := t.Context()
	id := "test-exists-id"

	// Should not exist initially
	exists, err := s.Exists(ctx, id)
	if err != nil {
		t.Fatalf("failed to check exists: %v", err)
	}
	if exists {
		t.Fatal("expected false for non-existent item")
	}

	// Save and check again
	item := &Item{ID: id, PublishedAt: time.Now()}
	if err := s.SaveItem(ctx, item); err != nil {
		t.Fatalf("failed to save item: %v", err)
	}

	exists, err = s.Exists(ctx, id)
	if err != nil {
		t.Fatalf("failed to check exists after save: %v", err)
	}
	if !exists {
		t.Fatal("expected true for existing item")
	}
}

func TestSaveItem_PersistsSavedAndOverride(t *testing.T) {
	dbPath := "test_saved_override.db"
	defer os.Remove(dbPath)
	s, err := InitDB(t.Context(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().Truncate(time.Second)
	override := "interest"
	savedAt := now
	item := &Item{
		ID:                   "saved-1",
		Source:               "test",
		Title:                "Saved item",
		InterestLevel:        "uninterested",
		UserInterestOverride: &override,
		Saved:                true,
		SavedAt:              &savedAt,
		PublishedAt:          now,
	}
	if err := s.SaveItem(t.Context(), item); err != nil {
		t.Fatal(err)
	}

	items, err := s.GetItems(t.Context(), 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if !items[0].Saved {
		t.Fatal("expected saved=true")
	}
	if items[0].UserInterestOverride == nil || *items[0].UserInterestOverride != override {
		t.Fatalf("expected override=%q, got %#v", override, items[0].UserInterestOverride)
	}
}

func TestSaveItem_DuplicateOfRoundTrip(t *testing.T) {
	dbPath := "test_duplicate_of.db"
	defer os.Remove(dbPath)
	s, err := InitDB(t.Context(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().Truncate(time.Second)
	dup := "base-item"
	item := &Item{
		ID:          "dup-item",
		Source:      "test",
		Title:       "Duplicate item",
		DuplicateOf: &dup,
		PublishedAt: now,
	}
	if err := s.SaveItem(t.Context(), item); err != nil {
		t.Fatal(err)
	}

	items, err := s.GetItems(t.Context(), 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].DuplicateOf == nil || *items[0].DuplicateOf != dup {
		t.Fatalf("expected duplicate_of=%q, got %#v", dup, items[0].DuplicateOf)
	}
}

func TestSearchItems_FTS5(t *testing.T) {
	dbPath := "test_search.db"
	defer os.Remove(dbPath)
	s, err := InitDB(t.Context(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().Truncate(time.Second)
	items := []*Item{
		{
			ID:            "item-ai",
			Source:        "hn",
			Title:         "AI chips trend",
			Description:   "about AI accelerators",
			InterestLevel: "high_interest",
			PublishedAt:   now,
		},
		{
			ID:            "item-db",
			Source:        "db",
			Title:         "SQLite release",
			Description:   "database update",
			InterestLevel: "interest",
			PublishedAt:   now.Add(-time.Minute),
		},
	}
	for _, it := range items {
		if err := s.SaveItem(t.Context(), it); err != nil {
			t.Fatal(err)
		}
	}

	results, err := s.SearchItems(t.Context(), "AI", 10, SearchFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "item-ai" {
		t.Fatalf("expected item-ai, got %s", results[0].ID)
	}
}

func TestSearchItems_FilterUnread(t *testing.T) {
	dbPath := "test_search_unread.db"
	defer os.Remove(dbPath)
	s, err := InitDB(t.Context(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().Truncate(time.Second)
	for _, it := range []*Item{
		{
			ID:            "unread-item",
			Title:         "Unread AI item",
			Description:   "ai",
			InterestLevel: "interest",
			IsRead:        false,
			PublishedAt:   now,
		},
		{
			ID:            "read-item",
			Title:         "Read AI item",
			Description:   "ai",
			InterestLevel: "interest",
			IsRead:        true,
			PublishedAt:   now,
		},
	} {
		if err := s.SaveItem(t.Context(), it); err != nil {
			t.Fatal(err)
		}
	}

	unread := true
	got, err := s.SearchItems(t.Context(), "AI", 10, SearchFilters{Unread: &unread})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "unread-item" {
		t.Fatalf("expected unread-item only, got %#v", got)
	}
}

func TestDigestItems_ReturnsSavedAndHighInterest(t *testing.T) {
	dbPath := "test_digest.db"
	defer os.Remove(dbPath)
	s, err := InitDB(t.Context(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().Truncate(time.Second)
	savedAt := now
	for _, it := range []*Item{
		{
			ID:            "saved-old",
			Title:         "Saved old",
			InterestLevel: "uninterested",
			Saved:         true,
			SavedAt:       &savedAt,
			PublishedAt:   now.AddDate(0, 0, -30),
		},
		{
			ID:            "hi-recent",
			Title:         "High recent",
			InterestLevel: "high_interest",
			PublishedAt:   now.AddDate(0, 0, -1),
		},
		{
			ID:            "normal",
			Title:         "Normal",
			InterestLevel: "interest",
			PublishedAt:   now,
		},
	} {
		if err := s.SaveItem(t.Context(), it); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.DigestItems(t.Context(), now.AddDate(0, 0, -7), 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 digest items, got %d", len(got))
	}
}

func TestAllItems_UsesUserOverrideForExclude(t *testing.T) {
	dbPath := "test_all_items_override.db"
	defer os.Remove(dbPath)
	s, err := InitDB(t.Context(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().Truncate(time.Second)
	override := "exclude"
	if err := s.SaveItem(t.Context(), &Item{
		ID:                   "ovr-exclude",
		Title:                "Should hide",
		InterestLevel:        "high_interest",
		UserInterestOverride: &override,
		PublishedAt:          now,
	}); err != nil {
		t.Fatal(err)
	}

	count := 0
	for it, err := range s.AllItems(t.Context()) {
		if err != nil {
			t.Fatal(err)
		}
		if it != nil {
			count++
		}
	}
	if count != 0 {
		t.Fatalf("expected 0 visible items, got %d", count)
	}
}

func TestDigestItems_UsesUserOverrideForHighInterest(t *testing.T) {
	dbPath := "test_digest_override.db"
	defer os.Remove(dbPath)
	s, err := InitDB(t.Context(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().Truncate(time.Second)
	override := "high_interest"
	if err := s.SaveItem(t.Context(), &Item{
		ID:                   "digest-override",
		Title:                "Should appear via override",
		InterestLevel:        "uninterested",
		UserInterestOverride: &override,
		PublishedAt:          now,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := s.DigestItems(t.Context(), now.Add(-24*time.Hour), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 digest item, got %d", len(got))
	}
	if got[0].InterestLevel != "high_interest" {
		t.Fatalf("expected effective level high_interest, got %s", got[0].InterestLevel)
	}
}

func TestListSources(t *testing.T) {
	dbPath := "test_sources.db"
	defer os.Remove(dbPath)
	s, err := InitDB(t.Context(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().Truncate(time.Second)
	for _, it := range []*Item{
		{ID: "s1", Source: "alpha", PublishedAt: now},
		{ID: "s2", Source: "beta", PublishedAt: now},
		{ID: "s3", Source: "alpha", PublishedAt: now},
	} {
		if err := s.SaveItem(t.Context(), it); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.ListSources(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 unique sources, got %d", len(got))
	}
	if got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("unexpected sources: %#v", got)
	}
}

func TestItemStats(t *testing.T) {
	dbPath := "test_stats.db"
	defer os.Remove(dbPath)
	s, err := InitDB(t.Context(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().Truncate(time.Second)
	ex := "exclude"
	for _, it := range []*Item{
		{ID: "st-1", InterestLevel: "high_interest", IsRead: false, Saved: true, PublishedAt: now},
		{ID: "st-2", InterestLevel: "interest", IsRead: true, PublishedAt: now},
		{ID: "st-3", InterestLevel: "uninterested", IsRead: false, PublishedAt: now},
		{ID: "st-4", InterestLevel: "high_interest", UserInterestOverride: &ex, IsRead: false, PublishedAt: now},
	} {
		if err := s.SaveItem(t.Context(), it); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.ItemStats(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalVisible != 3 {
		t.Fatalf("expected total_visible=3, got %d", got.TotalVisible)
	}
	if got.Saved != 1 {
		t.Fatalf("expected saved=1, got %d", got.Saved)
	}
	if got.HighInterest != 1 {
		t.Fatalf("expected high_interest=1, got %d", got.HighInterest)
	}
	if got.UnreadVisible != 2 {
		t.Fatalf("expected unread_visible=2, got %d", got.UnreadVisible)
	}
}

func TestSourceStats(t *testing.T) {
	dbPath := "test_source_stats.db"
	defer os.Remove(dbPath)
	s, err := InitDB(t.Context(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().Truncate(time.Second)
	for _, it := range []*Item{
		{ID: "ss-1", Source: "alpha", InterestLevel: "high_interest", Saved: true, PublishedAt: now},
		{ID: "ss-2", Source: "alpha", InterestLevel: "interest", PublishedAt: now},
		{ID: "ss-3", Source: "beta", InterestLevel: "high_interest", PublishedAt: now},
		{ID: "ss-4", Source: "beta", InterestLevel: "exclude", PublishedAt: now},
	} {
		if err := s.SaveItem(t.Context(), it); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.SourceStats(t.Context(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(got))
	}
	if got[0].Source != "alpha" || got[0].Visible != 2 || got[0].Saved != 1 || got[0].HighInterest != 1 {
		t.Fatalf("unexpected alpha stats: %#v", got[0])
	}
}

func TestLowValueSourceSuggestions(t *testing.T) {
	dbPath := "test_source_suggestions.db"
	defer os.Remove(dbPath)
	s, err := InitDB(t.Context(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().Truncate(time.Second)
	for i := 0; i < 5; i++ {
		if err := s.SaveItem(t.Context(), &Item{
			ID:            fmt.Sprintf("lv-%d", i),
			Source:        "low-value",
			InterestLevel: "uninterested",
			PublishedAt:   now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 5; i++ {
		if err := s.SaveItem(t.Context(), &Item{
			ID:            fmt.Sprintf("hi-%d", i),
			Source:        "good-source",
			InterestLevel: "high_interest",
			PublishedAt:   now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.LowValueSourceSuggestions(t.Context(), 3, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(got))
	}
	if got[0].Source != "low-value" {
		t.Fatalf("expected low-value source, got %#v", got[0])
	}
}
