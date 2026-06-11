// internal/git/git.go
package git

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/chandler767/redpanda-rec-room/internal/runner"
)

// indexMu serializes read-modify-write access to index.json within the same process.
var indexMu sync.Mutex

// AddCommitPush stages path, commits with message, and pushes to remote/branch.
// dryRun=true commits but does not push.
func AddCommitPush(repoDir, path, message, remote, branch string, dryRun bool) error {
	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %v: %w\n%s", args, err, out)
		}
		return nil
	}

	if err := run("add", path); err != nil {
		return err
	}
	if err := run("commit", "-m", fmt.Sprintf("report: %s", message)); err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	return run("push", remote, branch)
}

// SaveReport writes report JSON to reports/{id}.json and prepends to reports/index.json.
// indexMu serializes in-process callers; cross-process collisions remain a known limitation.
func SaveReport(repoDir string, report *runner.Report) error {
	indexMu.Lock()
	defer indexMu.Unlock()
	reportsDir := filepath.Join(repoDir, "reports")
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(reportsDir, report.ID+".json"), data, 0644); err != nil {
		return err
	}

	indexPath := filepath.Join(reportsDir, "index.json")
	var index []runner.IndexEntry
	if existing, err := os.ReadFile(indexPath); err == nil {
		if err := json.Unmarshal(existing, &index); err != nil {
			return fmt.Errorf("parse index.json: %w", err)
		}
	}

	entry := runner.IndexEntry{
		ID:               report.ID,
		Timestamp:        report.Timestamp,
		Mode:             report.Mode,
		Origin:           report.Origin.DetectedLocation,
		RedpandaMentions: report.Summary.RedpandaMentions,
		TotalCalls:       report.Summary.TotalCalls,
		MentionRate:      report.Summary.MentionRate,
	}
	index = append([]runner.IndexEntry{entry}, index...) // newest first

	indexData, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(indexPath, indexData, 0644)
}
