package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/liuerfire/sieve/internal/ai"
	"github.com/liuerfire/sieve/internal/config"
	"github.com/liuerfire/sieve/internal/engine"
	"github.com/liuerfire/sieve/internal/storage"
)

// Default HTTP server timeouts
const (
	defaultReadTimeout  = 10 * time.Second
	defaultWriteTimeout = 30 * time.Second
	defaultIdleTimeout  = 120 * time.Second
)

type Server struct {
	cfg       *config.Config
	storage   *storage.Storage
	ai        *ai.Client
	Events    chan engine.ProgressEvent
	runMu     sync.Mutex
	runCtx    context.Context
	runCancel context.CancelFunc
}

func NewServer(cfg *config.Config, s *storage.Storage, a *ai.Client) *Server {
	return &Server{
		cfg:     cfg,
		storage: s,
		ai:      a,
		Events:  make(chan engine.ProgressEvent, 100),
	}
}

// Shutdown cancels any running aggregation
func (s *Server) Shutdown() {
	s.runMu.Lock()
	cancel := s.runCancel
	s.runMu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/", StaticHandler())
	mux.HandleFunc("/api/items", s.handleGetItems)
	mux.HandleFunc("/api/items/", s.handleUpdateItem)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/run", s.handleRun)

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

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case ev := <-s.Events:
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.runMu.Lock()
	if s.runCancel != nil {
		s.runCancel()
	}
	runCtx, runCancel := context.WithCancel(context.Background())
	s.runCtx = runCtx
	s.runCancel = runCancel
	s.runMu.Unlock()

	go func(runCtx context.Context) {
		eng := engine.NewEngine(s.cfg, s.storage, s.ai)
		eng.OnProgress = func(ev engine.ProgressEvent) {
			select {
			case s.Events <- ev:
			case <-runCtx.Done():
			}
		}
		_, _ = eng.Run(runCtx)
	}(runCtx)

	w.WriteHeader(http.StatusAccepted)
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
			Level *string `json:"level"`
			Read  *bool   `json:"read"`
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
