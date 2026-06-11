// internal/server/server.go
package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/chandler767/redpanda-rec-room/internal/config"
)

// Server is the local HTTP API server for the dashboard.
type Server struct {
	cfg      *config.Config
	repoDir  string
	mux      *http.ServeMux
	broker   *SSEBroker
	brokerMu sync.RWMutex
	runMu    sync.Mutex
}

// New creates a Server. cfg may be nil (status endpoint still works).
func New(cfg *config.Config, repoDir string) *Server {
	s := &Server{
		cfg:     cfg,
		repoDir: repoDir,
		mux:     http.NewServeMux(),
		broker:  NewSSEBroker(),
	}
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/config", s.handleConfig)
	s.mux.HandleFunc("/api/run", s.handleRun)
	s.mux.HandleFunc("/api/run/stream", s.handleRunStream)
	s.mux.HandleFunc("/api/schedule", s.handleSchedule)
	if repoDir != "" {
		// Block sensitive files before the catch-all file server.
		block := func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		}
		s.mux.HandleFunc("/config.local.yaml", block)
		s.mux.HandleFunc("/.git/", block)
		s.mux.HandleFunc("/internal/", block)
		s.mux.HandleFunc("/cmd/", block)
		s.mux.Handle("/", http.FileServer(http.Dir(repoDir)))
	}
	return s
}

// ServeHTTP adds CORS headers and delegates to the mux.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}

// Start listens on addr (e.g. ":8765").
func (s *Server) Start(addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0, // 0 = no write timeout; required for SSE long-poll streams
		IdleTimeout:       120 * time.Second,
	}
	return srv.ListenAndServe()
}
