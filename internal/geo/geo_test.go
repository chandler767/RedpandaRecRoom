// internal/geo/geo_test.go
package geo_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chandler767/redpanda-rec-room/internal/geo"
)

func TestLocateWithURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"ip":      "1.2.3.4",
			"city":    "San Francisco",
			"region":  "California",
			"country": "US",
		})
	}))
	defer srv.Close()

	info, err := geo.LocateWithURL(srv.URL)
	if err != nil {
		t.Fatalf("LocateWithURL() error: %v", err)
	}
	if info.IP != "1.2.3.4" {
		t.Errorf("IP = %q, want 1.2.3.4", info.IP)
	}
	if info.Location != "San Francisco, California, US" {
		t.Errorf("Location = %q, want San Francisco, California, US", info.Location)
	}
}

func TestLocateWithURL_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	info, err := geo.LocateWithURL(srv.URL)
	if err != nil {
		t.Fatalf("expected graceful fallback, not error: %v", err)
	}
	if info.Location != "unknown" {
		t.Errorf("Location = %q, want unknown", info.Location)
	}
}
