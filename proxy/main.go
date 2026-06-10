// proxy/main.go
// GCR Regional Proxy — deploy to Google Cloud Run when enabling regional mode.
// See docs/superpowers/specs/2026-06-10-redpanda-llm-tracker-design.md for deployment steps.
//
// Receives: POST /complete with JSON body {prompt, model, provider_type, endpoint, api_key}
// Returns:  JSON {response, region, latency_ms}
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	http.HandleFunc("/complete", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "proxy not yet implemented — see design spec", http.StatusNotImplemented)
	})
	log.Printf("proxy listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
