// cmd/run/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/chandler767/redpanda-rec-room/internal/config"
	"github.com/chandler767/redpanda-rec-room/internal/geo"
	"github.com/chandler767/redpanda-rec-room/internal/git"
	"github.com/chandler767/redpanda-rec-room/internal/llm"
	"github.com/chandler767/redpanda-rec-room/internal/runner"
	"github.com/chandler767/redpanda-rec-room/internal/schedule"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Run without saving or pushing")
	installSched := flag.Bool("install-schedule", false, "Install launchd job (3x/day)")
	uninstallSched := flag.Bool("uninstall-schedule", false, "Remove launchd job")
	flag.Parse()

	repoDir, err := findRepoRoot()
	if err != nil {
		log.Fatalf("cannot find repo root: %v", err)
	}

	if *installSched {
		binary, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot resolve binary path: %v", err)
		}
		if err := schedule.Install(binary); err != nil {
			log.Fatalf("install schedule: %v", err)
		}
		fmt.Println("Scheduled: 08:00, 14:00, 20:00 daily")
		return
	}

	if *uninstallSched {
		if err := schedule.Uninstall(); err != nil {
			log.Fatalf("uninstall schedule: %v", err)
		}
		fmt.Println("Schedule removed")
		return
	}

	cfg, err := config.Load(
		filepath.Join(repoDir, "config.yaml"),
		filepath.Join(repoDir, "config.local.yaml"),
	)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	clients := map[string]llm.Client{}
	for _, p := range cfg.Providers {
		if !p.Enabled {
			continue
		}
		key := cfg.APIKeys[p.ID]
		if key == "" {
			log.Printf("SKIP %s: no API key", p.Name)
			continue
		}
		c, err := llm.New(p, key)
		if err != nil {
			log.Fatalf("build client %s: %v", p.ID, err)
		}
		clients[p.ID] = c
	}

	geoInfo, _ := geo.Locate()
	fmt.Printf("Origin: %s (%s)\n", geoInfo.Location, geoInfo.IP)

	report, err := runner.Run(context.Background(), cfg, clients, geoInfo, func(msg string) {
		fmt.Println(msg)
	})
	if err != nil {
		log.Fatalf("run: %v", err)
	}

	fmt.Printf("\n%d/%d calls mentioned Redpanda (%.0f%%)\n",
		report.Summary.RedpandaMentions,
		report.Summary.Completed,
		report.Summary.MentionRate*100,
	)

	if *dryRun {
		fmt.Println("Dry run — not saving")
		return
	}

	if err := git.SaveReport(repoDir, report); err != nil {
		log.Fatalf("save report: %v", err)
	}
	fmt.Printf("Saved: reports/%s.json\n", report.ID)

	if cfg.Git.AutoPush {
		if err := git.AddCommitPush(repoDir, "reports/", report.ID, false); err != nil {
			log.Fatalf("git push: %v", err)
		}
		fmt.Println("Pushed to GitHub")
	}
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
			return "", fmt.Errorf("config.yaml not found in any parent directory")
		}
		dir = parent
	}
}
