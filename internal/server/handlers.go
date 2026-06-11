// internal/server/handlers.go
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/chandler767/redpanda-rec-room/internal/geo"
	"github.com/chandler767/redpanda-rec-room/internal/git"
	"github.com/chandler767/redpanda-rec-room/internal/llm"
	"github.com/chandler767/redpanda-rec-room/internal/runner"
	"github.com/chandler767/redpanda-rec-room/internal/schedule"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	if s.cfg == nil {
		http.Error(w, "no config loaded", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.cfg)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if s.cfg == nil {
		http.Error(w, "no config loaded", http.StatusServiceUnavailable)
		return
	}

	// Parse optional per-run overrides from the request body.
	var overrides struct {
		SearchString string `json:"search_string"`
		CallsPerLLM  int    `json:"calls_per_llm"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	json.NewDecoder(r.Body).Decode(&overrides) // ignore error — overrides are optional

	// Build a local config copy applying any overrides.
	cfg := *s.cfg
	if overrides.SearchString != "" {
		cfg.SearchString = overrides.SearchString
	}
	if overrides.CallsPerLLM > 0 && overrides.CallsPerLLM <= 25 {
		cfg.CallsPerLLM = overrides.CallsPerLLM
	}

	if !s.runMu.TryLock() {
		http.Error(w, "run already in progress", http.StatusConflict)
		return
	}

	// Create a fresh broker for this run and replace the old one.
	broker := NewSSEBroker()
	s.brokerMu.Lock()
	s.broker = broker
	s.brokerMu.Unlock()

	go func() {
		defer s.runMu.Unlock()
		defer broker.Close()

		clients := map[string]llm.Client{}
		for _, p := range cfg.Providers {
			if !p.Enabled {
				continue
			}
			key := cfg.APIKeys[p.ID]
			if key == "" {
				continue
			}
			c, err := llm.New(p, key)
			if err != nil {
				continue
			}
			clients[p.ID] = c
		}

		geoInfo, _ := geo.Locate()
		report, err := runner.Run(context.Background(), &cfg, clients, geoInfo, func(msg string) {
			broker.Publish(msg)
		})
		if err != nil {
			broker.Publish("ERROR: " + err.Error())
			return
		}
		if err := git.SaveReport(s.repoDir, report); err != nil {
			broker.Publish("ERROR saving report: " + err.Error())
			return
		}
		if cfg.Git.AutoPush {
			if err := git.AddCommitPush(s.repoDir, "reports/", report.ID, cfg.Git.Remote, cfg.Git.Branch, false); err != nil {
				broker.Publish("ERROR pushing: " + err.Error())
				return
			}
		}
		broker.Publish("COMPLETE:" + report.ID)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

// handleRunStream serves the SSE stream from the current run broker.
func (s *Server) handleRunStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	s.brokerMu.RLock()
	b := s.broker
	s.brokerMu.RUnlock()
	b.ServeHTTP(w, r)
}

func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Action string `json:"action"` // "install" | "uninstall"
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	switch body.Action {
	case "install":
		exe, err := os.Executable()
		if err != nil {
			http.Error(w, "cannot determine executable path: "+err.Error(), http.StatusInternalServerError)
			return
		}
		runBin := filepath.Join(filepath.Dir(exe), "run")
		if err := schedule.Install(runBin); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "installed"})
	case "uninstall":
		if err := schedule.Uninstall(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "uninstalled"})
	default:
		http.Error(w, `action must be "install" or "uninstall"`, http.StatusBadRequest)
	}
}
