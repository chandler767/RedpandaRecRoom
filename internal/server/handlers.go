// internal/server/handlers.go
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/chandler767/redpanda-rec-room/internal/geo"
	"github.com/chandler767/redpanda-rec-room/internal/git"
	"github.com/chandler767/redpanda-rec-room/internal/llm"
	"github.com/chandler767/redpanda-rec-room/internal/runner"
	"github.com/chandler767/redpanda-rec-room/internal/schedule"
)

var runMu sync.Mutex // one run at a time

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
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
	if !runMu.TryLock() {
		http.Error(w, "run already in progress", http.StatusConflict)
		return
	}

	// Create a fresh broker for this run and replace the old one.
	broker := NewSSEBroker()
	s.broker = broker

	go func() {
		defer runMu.Unlock()
		defer broker.Close()

		clients := map[string]llm.Client{}
		for _, p := range s.cfg.Providers {
			if !p.Enabled {
				continue
			}
			key := s.cfg.APIKeys[p.ID]
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
		report, err := runner.Run(context.Background(), s.cfg, clients, geoInfo, func(msg string) {
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
		if s.cfg.Git.AutoPush {
			if err := git.AddCommitPush(s.repoDir, "reports/", report.ID, false); err != nil {
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
	s.broker.ServeHTTP(w, r)
}

func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Action string `json:"action"` // "install" | "uninstall"
	}
	json.NewDecoder(r.Body).Decode(&body)

	w.Header().Set("Content-Type", "application/json")
	switch body.Action {
	case "install":
		if err := schedule.Install(s.repoDir + "/run"); err != nil {
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
