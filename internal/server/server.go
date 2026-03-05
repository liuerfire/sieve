package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/storage"
)

// Default HTTP server timeouts
const (
	defaultReadTimeout  = 10 * time.Second
	defaultWriteTimeout = 30 * time.Second
	defaultIdleTimeout  = 120 * time.Second
)

type Server struct {
	cfg     *config.Config
	storage *storage.Storage
}

func NewServer(cfg *config.Config, s *storage.Storage) *Server {
	return &Server{
		cfg:     cfg,
		storage: s,
	}
}

func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/", StaticHandler())
	mux.HandleFunc("/api/items", s.handleGetItems)
	mux.HandleFunc("/api/items/stats", s.handleGetStats)
	mux.HandleFunc("/api/items/source-stats", s.handleSourceStats)
	mux.HandleFunc("/api/items/sources", s.handleGetSources)
	mux.HandleFunc("/api/items/search", s.handleSearchItems)
	mux.HandleFunc("/api/items/", s.handleUpdateItem)
	mux.HandleFunc("/api/digest", s.handleDigest)
	mux.HandleFunc("/api/config", s.handleConfig)

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

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.cfg)

	case http.MethodPut:
		var newCfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := newCfg.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Save to file (assuming config.json for now, but should ideally know the path)
		// For now we'll overwrite config.json in the current dir
		data, _ := json.MarshalIndent(newCfg, "", "  ")
		if err := os.WriteFile("config.json", data, 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.cfg = &newCfg
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
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

	items, err := s.storage.SearchItems(r.Context(), q, 100, storage.SearchFilters{
		Source: source,
		Level:  level,
		Saved:  saved,
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
