// internal/git/git.go
package git

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/chandler767/redpanda-rec-room/internal/runner"
)

// AddCommitPush stages path, commits with message, and pushes.
// dryRun=true commits but does not push.
func AddCommitPush(repoDir, path, message string, dryRun bool) error {
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
	return run("push")
}

// SaveReport writes report JSON to reports/{id}.json and prepends to reports/index.json.
func SaveReport(repoDir string, report *runner.Report) error {
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
		json.Unmarshal(existing, &index)
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
