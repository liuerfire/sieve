package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/liuerfire/sieve/internal/storage"
)

func TestHandleGetItems(t *testing.T) {
	ctx := t.Context()
	s, err := storage.InitDB(ctx, ":memory:")
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}
	defer s.Close()

	srv := NewServer(s)

	req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	w := httptest.NewRecorder()

	srv.handleGetItems(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var items []any
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func TestHandleGetSources_EmptyArrayNotNull(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	srv := NewServer(s)
	req := httptest.NewRequest(http.MethodGet, "/api/items/sources", nil)
	w := httptest.NewRecorder()
	srv.handleGetSources(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	body := strings.TrimSpace(w.Body.String())
	if body != "[]" {
		t.Fatalf("expected empty array body, got %q", body)
	}
}

func TestHandleSourceStats_EmptyArrayNotNull(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	srv := NewServer(s)
	req := httptest.NewRequest(http.MethodGet, "/api/items/source-stats", nil)
	w := httptest.NewRecorder()
	srv.handleSourceStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	body := strings.TrimSpace(w.Body.String())
	if body != "[]" {
		t.Fatalf("expected empty array body, got %q", body)
	}
}

func TestHandleSourceSuggestions_EmptyArrayNotNull(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	srv := NewServer(s)
	req := httptest.NewRequest(http.MethodGet, "/api/items/source-suggestions", nil)
	w := httptest.NewRecorder()
	srv.handleSourceSuggestions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	body := strings.TrimSpace(w.Body.String())
	if body != "[]" {
		t.Fatalf("expected empty array body, got %q", body)
	}
}

func TestHandleUpdateItem_MethodNotAllowed(t *testing.T) {
	srv := NewServer(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/items/123", nil)
	w := httptest.NewRecorder()

	srv.handleUpdateItem(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandleUpdateItem_Patch_Level(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	srv := NewServer(s)

	// Save a test item first
	item := &storage.Item{
		ID:            "test-123",
		Title:         "Test",
		InterestLevel: "interest",
		PublishedAt:   time.Now(),
	}
	if err := s.SaveItem(ctx, item); err != nil {
		t.Fatalf("failed to save test item: %v", err)
	}

	// Test PATCH to update level
	body := `{"level": "high_interest"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/items/test-123", strings.NewReader(body))
	w := httptest.NewRecorder()

	srv.handleUpdateItem(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}

func TestHandleUpdateItem_Delete(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	srv := NewServer(s)

	// Save a test item first
	item := &storage.Item{
		ID:          "test-456",
		Title:       "Test",
		PublishedAt: time.Now(),
	}
	if err := s.SaveItem(ctx, item); err != nil {
		t.Fatalf("failed to save test item: %v", err)
	}

	// Test DELETE
	req := httptest.NewRequest(http.MethodDelete, "/api/items/test-456", nil)
	w := httptest.NewRecorder()

	srv.handleUpdateItem(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}

func TestHandleUpdateItem_Patch_Saved(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	srv := NewServer(s)

	item := &storage.Item{
		ID:          "saved-123",
		Title:       "Save target",
		PublishedAt: time.Now(),
	}
	if err := s.SaveItem(ctx, item); err != nil {
		t.Fatalf("failed to save test item: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/items/saved-123", strings.NewReader(`{"saved": true}`))
	w := httptest.NewRecorder()
	srv.handleUpdateItem(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", w.Code)
	}

	items, err := s.GetItems(ctx, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !items[0].Saved {
		t.Fatalf("expected item saved=true, got %#v", items)
	}
}

func TestHandleUpdateItem_Patch_UserInterestOverride(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	srv := NewServer(s)

	item := &storage.Item{
		ID:            "override-123",
		Title:         "Override target",
		InterestLevel: "uninterested",
		PublishedAt:   time.Now(),
	}
	if err := s.SaveItem(ctx, item); err != nil {
		t.Fatalf("failed to save test item: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/items/override-123", strings.NewReader(`{"user_interest_override": "high_interest"}`))
	w := httptest.NewRecorder()
	srv.handleUpdateItem(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", w.Code)
	}

	items, err := s.GetItems(ctx, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if items[0].UserInterestOverride == nil || *items[0].UserInterestOverride != "high_interest" {
		t.Fatalf("expected override high_interest, got %#v", items[0].UserInterestOverride)
	}
}

func TestHandleSearchItems_FilterBySavedAndLevel(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	srv := NewServer(s)

	for _, it := range []*storage.Item{
		{
			ID:            "search-1",
			Title:         "AI launch",
			Description:   "AI innovation",
			InterestLevel: "high_interest",
			Saved:         true,
			PublishedAt:   time.Now(),
		},
		{
			ID:            "search-2",
			Title:         "Other topic",
			Description:   "random",
			InterestLevel: "interest",
			PublishedAt:   time.Now(),
		},
	} {
		if err := s.SaveItem(ctx, it); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/items/search?q=AI&saved=true&level=high_interest", nil)
	w := httptest.NewRecorder()
	srv.handleSearchItems(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var items []storage.Item
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "search-1" {
		t.Fatalf("expected search-1 only, got %#v", items)
	}
}

func TestHandleDigest_ReturnsSavedAndHighInterest(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	srv := NewServer(s)

	for _, it := range []*storage.Item{
		{
			ID:            "d-saved",
			Title:         "Saved item",
			InterestLevel: "uninterested",
			Saved:         true,
			PublishedAt:   time.Now().AddDate(0, 0, -20),
		},
		{
			ID:            "d-hi",
			Title:         "High item",
			InterestLevel: "high_interest",
			PublishedAt:   time.Now().AddDate(0, 0, -1),
		},
		{
			ID:            "d-low",
			Title:         "Low item",
			InterestLevel: "interest",
			PublishedAt:   time.Now(),
		},
	} {
		if err := s.SaveItem(ctx, it); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/digest", nil)
	w := httptest.NewRecorder()
	srv.handleDigest(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var items []storage.Item
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 digest items, got %d", len(items))
	}
}

func TestHandleDigest_InvalidDays(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	srv := NewServer(s)
	req := httptest.NewRequest(http.MethodGet, "/api/digest?days=0", nil)
	w := httptest.NewRecorder()

	srv.handleDigest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestHandleGetSources(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	for _, it := range []*storage.Item{
		{ID: "src-1", Source: "alpha", PublishedAt: time.Now()},
		{ID: "src-2", Source: "beta", PublishedAt: time.Now()},
		{ID: "src-3", Source: "alpha", PublishedAt: time.Now()},
	} {
		if err := s.SaveItem(ctx, it); err != nil {
			t.Fatal(err)
		}
	}

	srv := NewServer(s)
	req := httptest.NewRequest(http.MethodGet, "/api/items/sources", nil)
	w := httptest.NewRecorder()

	srv.handleGetSources(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var got []string
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("unexpected sources: %#v", got)
	}
}

func TestHandleGetStats(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	for _, it := range []*storage.Item{
		{ID: "stat-1", InterestLevel: "high_interest", PublishedAt: time.Now()},
		{ID: "stat-2", InterestLevel: "interest", Saved: true, PublishedAt: time.Now()},
	} {
		if err := s.SaveItem(ctx, it); err != nil {
			t.Fatal(err)
		}
	}

	srv := NewServer(s)
	req := httptest.NewRequest(http.MethodGet, "/api/items/stats", nil)
	w := httptest.NewRecorder()

	srv.handleGetStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var got storage.ItemStats
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.TotalVisible != 2 || got.Saved != 1 || got.HighInterest != 1 {
		t.Fatalf("unexpected stats: %#v", got)
	}
}

func TestHandleSourceStats(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	for _, it := range []*storage.Item{
		{ID: "src-stat-1", Source: "alpha", InterestLevel: "high_interest", PublishedAt: time.Now()},
		{ID: "src-stat-2", Source: "beta", InterestLevel: "interest", PublishedAt: time.Now()},
	} {
		if err := s.SaveItem(ctx, it); err != nil {
			t.Fatal(err)
		}
	}

	srv := NewServer(s)
	req := httptest.NewRequest(http.MethodGet, "/api/items/source-stats?limit=5", nil)
	w := httptest.NewRecorder()

	srv.handleSourceStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var got []storage.SourceStats
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 source stats rows, got %d", len(got))
	}
}

func TestHandleSourceSuggestions(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	for i := 0; i < 3; i++ {
		if err := s.SaveItem(ctx, &storage.Item{
			ID:            fmt.Sprintf("sug-low-%d", i),
			Source:        "low-source",
			InterestLevel: "uninterested",
			PublishedAt:   time.Now(),
		}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 3; i++ {
		if err := s.SaveItem(ctx, &storage.Item{
			ID:            fmt.Sprintf("sug-good-%d", i),
			Source:        "good-source",
			InterestLevel: "high_interest",
			PublishedAt:   time.Now(),
		}); err != nil {
			t.Fatal(err)
		}
	}

	srv := NewServer(s)
	req := httptest.NewRequest(http.MethodGet, "/api/items/source-suggestions?min_visible=2&limit=5", nil)
	w := httptest.NewRecorder()
	srv.handleSourceSuggestions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var got []storage.SourceSuggestion
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Source != "low-source" {
		t.Fatalf("unexpected suggestions: %#v", got)
	}
}

func TestHandleSearchItems_FilterUnread(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	for _, it := range []*storage.Item{
		{
			ID:            "u-search",
			Title:         "Unread AI",
			Description:   "ai",
			InterestLevel: "interest",
			IsRead:        false,
			PublishedAt:   time.Now(),
		},
		{
			ID:            "r-search",
			Title:         "Read AI",
			Description:   "ai",
			InterestLevel: "interest",
			IsRead:        true,
			PublishedAt:   time.Now(),
		},
	} {
		if err := s.SaveItem(ctx, it); err != nil {
			t.Fatal(err)
		}
	}

	srv := NewServer(s)
	req := httptest.NewRequest(http.MethodGet, "/api/items/search?q=AI&unread=true", nil)
	w := httptest.NewRecorder()
	srv.handleSearchItems(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var got []storage.Item
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "u-search" {
		t.Fatalf("unexpected unread results: %#v", got)
	}
}

func TestHandleBulkRead(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	for _, it := range []*storage.Item{
		{ID: "bulk-srv-1", InterestLevel: "interest", IsRead: false, PublishedAt: time.Now()},
		{ID: "bulk-srv-2", InterestLevel: "interest", IsRead: false, PublishedAt: time.Now()},
	} {
		if err := s.SaveItem(ctx, it); err != nil {
			t.Fatal(err)
		}
	}

	srv := NewServer(s)
	req := httptest.NewRequest(http.MethodPost, "/api/items/bulk-read", strings.NewReader(`{"ids":["bulk-srv-1","bulk-srv-2"],"read":true}`))
	w := httptest.NewRecorder()
	srv.handleBulkRead(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", w.Code)
	}

	items, err := s.GetItems(ctx, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if !it.IsRead {
			t.Fatalf("expected all items read, found unread: %s", it.ID)
		}
	}
}

func TestHandleFeedsCRUD(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()
	srv := NewServer(s)

	createReq := httptest.NewRequest(http.MethodPost, "/api/feeds", strings.NewReader(`{"id":"feed-1","name":"Feed 1","url":"https://example.com/rss","enabled":true}`))
	createW := httptest.NewRecorder()
	srv.handleFeeds(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("expected 201 on create, got %d", createW.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/feeds", nil)
	listW := httptest.NewRecorder()
	srv.handleFeeds(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 on list, got %d", listW.Code)
	}
	var feeds []storage.Feed
	if err := json.NewDecoder(listW.Body).Decode(&feeds); err != nil {
		t.Fatal(err)
	}
	if len(feeds) != 1 || feeds[0].ID != "feed-1" {
		t.Fatalf("unexpected feed list: %#v", feeds)
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/feeds/feed-1", strings.NewReader(`{"name":"Updated Feed","url":"https://example.com/rss","enabled":false}`))
	patchW := httptest.NewRecorder()
	srv.handleFeedByID(patchW, patchReq)
	if patchW.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on patch, got %d", patchW.Code)
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/api/feeds/feed-1", nil)
	delW := httptest.NewRecorder()
	srv.handleFeedByID(delW, delReq)
	if delW.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on delete, got %d", delW.Code)
	}
}

func TestHandleSettings_GetPatch(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()
	srv := NewServer(s)

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/settings", strings.NewReader(`{"preferred_language":"en","ai_provider":"gemini"}`))
	patchW := httptest.NewRecorder()
	srv.handleSettings(patchW, patchReq)
	if patchW.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on settings patch, got %d", patchW.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	getW := httptest.NewRecorder()
	srv.handleSettings(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 on settings get, got %d", getW.Code)
	}
	var settings map[string]string
	if err := json.NewDecoder(getW.Body).Decode(&settings); err != nil {
		t.Fatal(err)
	}
	if settings["preferred_language"] != "en" || settings["ai_provider"] != "gemini" {
		t.Fatalf("unexpected settings map: %#v", settings)
	}
}

func TestHandleSearchItems_FilterByFeedID(t *testing.T) {
	ctx := t.Context()
	s, _ := storage.InitDB(ctx, ":memory:")
	defer s.Close()

	for _, it := range []*storage.Item{
		{
			ID:            "feed-item-1",
			FeedID:        "feed-a",
			Title:         "Feed A item",
			InterestLevel: "interest",
			PublishedAt:   time.Now(),
		},
		{
			ID:            "feed-item-2",
			FeedID:        "feed-b",
			Title:         "Feed B item",
			InterestLevel: "interest",
			PublishedAt:   time.Now(),
		},
	} {
		if err := s.SaveItem(ctx, it); err != nil {
			t.Fatal(err)
		}
	}

	srv := NewServer(s)
	req := httptest.NewRequest(http.MethodGet, "/api/items/search?feed_id=feed-a", nil)
	w := httptest.NewRecorder()
	srv.handleSearchItems(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var got []storage.Item
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "feed-item-1" {
		t.Fatalf("unexpected feed_id results: %#v", got)
	}
}
