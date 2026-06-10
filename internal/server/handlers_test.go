// internal/server/handlers_test.go
package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chandler767/redpanda-rec-room/internal/server"
)

func TestStatusHandler(t *testing.T) {
	s := server.New(nil, "")
	req := httptest.NewRequest("GET", "/api/status", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestCORSPreflight(t *testing.T) {
	s := server.New(nil, "")
	req := httptest.NewRequest("OPTIONS", "/api/status", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS header")
	}
}
