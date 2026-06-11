// internal/geo/geo.go
package geo

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const defaultURL = "https://ipinfo.io/json"

// Info holds the detected public IP and location string.
type Info struct {
	IP       string
	Location string
}

type ipinfoResponse struct {
	IP      string `json:"ip"`
	City    string `json:"city"`
	Region  string `json:"region"`
	Country string `json:"country"`
}

// Locate detects the current machine's public IP and location.
// Falls back to Info{IP:"unknown", Location:"unknown"} on any error.
func Locate() (Info, error) {
	return LocateWithURL(defaultURL)
}

// LocateWithURL is Locate with a configurable endpoint (for tests).
func LocateWithURL(url string) (Info, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return Info{IP: "unknown", Location: "unknown"}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Info{IP: "unknown", Location: "unknown"}, nil
	}

	var r ipinfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return Info{IP: "unknown", Location: "unknown"}, nil
	}

	parts := []string{}
	if r.City != "" {
		parts = append(parts, r.City)
	}
	if r.Region != "" {
		parts = append(parts, r.Region)
	}
	if r.Country != "" {
		parts = append(parts, r.Country)
	}
	location := strings.Join(parts, ", ")
	if location == "" {
		location = "unknown"
	}
	if r.IP == "" {
		r.IP = "unknown"
	}
	return Info{IP: r.IP, Location: location}, nil
}
