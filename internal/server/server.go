package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/liuerfire/sieve/internal/storage"
)

// Default HTTP server timeouts
const (
	defaultReadTimeout  = 10 * time.Second
	defaultWriteTimeout = 30 * time.Second
	defaultIdleTimeout  = 120 * time.Second
)

type Server struct {
	storage *storage.Storage
}

func NewServer(s *storage.Storage) *Server {
	return &Server{
		storage: s,
	}
}

func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/", StaticHandler())
	mux.HandleFunc("/api/items", s.handleGetItems)
	mux.HandleFunc("/api/items/bulk-read", s.handleBulkRead)
	mux.HandleFunc("/api/items/stats", s.handleGetStats)
	mux.HandleFunc("/api/items/source-stats", s.handleSourceStats)
	mux.HandleFunc("/api/items/source-suggestions", s.handleSourceSuggestions)
	mux.HandleFunc("/api/items/sources", s.handleGetSources)
	mux.HandleFunc("/api/items/search", s.handleSearchItems)
	mux.HandleFunc("/api/items/", s.handleUpdateItem)
	mux.HandleFunc("/api/digest", s.handleDigest)
	mux.HandleFunc("/api/feeds", s.handleFeeds)
	mux.HandleFunc("/api/feeds/", s.handleFeedByID)
	mux.HandleFunc("/api/settings", s.handleSettings)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}

	fmt.Printf("Sieve Web UI listening on %s\n", addr)
	return server.ListenAndServe()
}

func (s *Server) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/items/")
	if id == "" {
		http.Error(w, "Item ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPatch:
		var req struct {
			Level                *string `json:"level"`
			Read                 *bool   `json:"read"`
			Saved                *bool   `json:"saved"`
			UserInterestOverride *string `json:"user_interest_override"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Level != nil {
			if err := s.storage.UpdateLevel(r.Context(), id, *req.Level); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if req.Read != nil {
			if err := s.storage.UpdateReadStatus(r.Context(), id, *req.Read); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if req.Saved != nil {
			if err := s.storage.UpdateSavedStatus(r.Context(), id, *req.Saved); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if req.UserInterestOverride != nil {
			level := strings.TrimSpace(*req.UserInterestOverride)
			if level == "" {
				if err := s.storage.UpdateUserInterestOverride(r.Context(), id, nil); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				if err := s.storage.UpdateUserInterestOverride(r.Context(), id, &level); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		if err := s.storage.DeleteItem(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleBulkRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		IDs  []string `json:"ids"`
		Read bool     `json:"read"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.IDs) == 0 {
		http.Error(w, "ids required", http.StatusBadRequest)
		return
	}
	if err := s.storage.UpdateReadStatusBulk(r.Context(), req.IDs, req.Read); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetItems(w http.ResponseWriter, r *http.Request) {
	items, err := s.storage.GetItems(r.Context(), 50, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleSearchItems(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	feedID := strings.TrimSpace(r.URL.Query().Get("feed_id"))
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	level := strings.TrimSpace(r.URL.Query().Get("level"))

	var saved *bool
	if raw := strings.TrimSpace(r.URL.Query().Get("saved")); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			http.Error(w, "invalid saved flag", http.StatusBadRequest)
			return
		}
		saved = &v
	}
	var unread *bool
	if raw := strings.TrimSpace(r.URL.Query().Get("unread")); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			http.Error(w, "invalid unread flag", http.StatusBadRequest)
			return
		}
		unread = &v
	}

	items, err := s.storage.SearchItems(r.Context(), q, 100, storage.SearchFilters{
		FeedID: feedID,
		Source: source,
		Level:  level,
		Saved:  saved,
		Unread: unread,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.storage.ItemStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleSourceStats(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 || v > 100 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = v
	}

	stats, err := s.storage.SourceStats(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleSourceSuggestions(w http.ResponseWriter, r *http.Request) {
	minVisible := 10
	if raw := strings.TrimSpace(r.URL.Query().Get("min_visible")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 || v > 1000 {
			http.Error(w, "invalid min_visible", http.StatusBadRequest)
			return
		}
		minVisible = v
	}

	limit := 10
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 || v > 100 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = v
	}

	suggestions, err := s.storage.LowValueSourceSuggestions(r.Context(), minVisible, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(suggestions); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleGetSources(w http.ResponseWriter, r *http.Request) {
	sources, err := s.storage.ListSources(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sources); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleDigest(w http.ResponseWriter, r *http.Request) {
	days := 7
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 || v > 365 {
			http.Error(w, "invalid days", http.StatusBadRequest)
			return
		}
		days = v
	}
	since := time.Now().AddDate(0, 0, -days)
	items, err := s.storage.DigestItems(r.Context(), since, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleFeeds(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		enabledOnly := strings.TrimSpace(r.URL.Query().Get("enabled")) == "true"
		feeds, err := s.storage.ListFeeds(r.Context(), enabledOnly)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(feeds); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodPost:
		var feed storage.Feed
		if err := json.NewDecoder(r.Body).Decode(&feed); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(feed.ID) == "" || strings.TrimSpace(feed.Name) == "" || strings.TrimSpace(feed.URL) == "" {
			http.Error(w, "id, name and url are required", http.StatusBadRequest)
			return
		}
		if err := s.storage.CreateFeed(r.Context(), &feed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleFeedByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/feeds/")
	if id == "" {
		http.Error(w, "Feed ID required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPatch:
		var feed storage.Feed
		if err := json.NewDecoder(r.Body).Decode(&feed); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		feed.ID = id
		if err := s.storage.UpdateFeed(r.Context(), &feed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if err := s.storage.DeleteFeed(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings, err := s.storage.GetSettings(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(settings); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodPatch:
		var values map[string]string
		if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.storage.UpdateSettings(r.Context(), values); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
