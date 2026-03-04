package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/storage"
)

func TestHandleGetItems(t *testing.T) {
	ctx := t.Context()
	s, err := storage.InitDB(ctx, ":memory:")
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}
	defer s.Close()

	cfg := &config.Config{}
	srv := NewServer(cfg, s)

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

func TestHandleConfig_Get(t *testing.T) {
	cfg := &config.Config{
		Global: config.GlobalConfig{
			PreferredLanguage: "en",
		},
	}
	srv := NewServer(cfg, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	srv.handleConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var returnedCfg config.Config
	if err := json.NewDecoder(w.Body).Decode(&returnedCfg); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if returnedCfg.Global.PreferredLanguage != "en" {
		t.Errorf("expected preferred language 'en', got '%s'", returnedCfg.Global.PreferredLanguage)
	}
}

func TestHandleUpdateItem_MethodNotAllowed(t *testing.T) {
	srv := NewServer(nil, nil)

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

	srv := NewServer(&config.Config{}, s)

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

	srv := NewServer(&config.Config{}, s)

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
