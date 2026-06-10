// cmd/serve/main.go
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/chandler767/redpanda-rec-room/internal/config"
	"github.com/chandler767/redpanda-rec-room/internal/server"
)

const addr = ":8765"

func main() {
	repoDir, err := findRepoRoot()
	if err != nil {
		log.Fatalf("cannot find repo root: %v", err)
	}

	cfg, err := config.Load(
		filepath.Join(repoDir, "config.yaml"),
		filepath.Join(repoDir, "config.local.yaml"),
	)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	s := server.New(cfg, repoDir)
	url := "http://localhost" + addr
	fmt.Printf("Dashboard:  %s\n", url)
	fmt.Printf("Local API:  %s/api/status\n", url)
	fmt.Println("Ctrl+C to stop")

	openBrowser(url)

	if err := s.Start(addr); err != nil {
		log.Fatal(err)
	}
}

func openBrowser(url string) {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	default:
		return
	}
	exec.Command(cmd, url).Start()
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "config.yaml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("config.yaml not found")
		}
		dir = parent
	}
}
